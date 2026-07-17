// proxy/src/main.go
package main

import (
	"context"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"encoding/json"
	"github.com/latency-space/shared/celestial"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// infoTemplate holds the parsed HTML template for the celestial body information page.
var infoTemplate *template.Template

// StatusEntry represents the data for a single celestial object returned by the status API.
type StatusEntry struct {
	Name       string  `json:"name"`
	Type       string  `json:"type"`
	ParentName string  `json:"parentName,omitempty"` // Omit if empty
	Distance   float64 `json:"distance_km"`
	Latency    float64 `json:"latency_seconds"` // Changed type to float64
	Occluded   bool    `json:"occluded"`
}

// ApiResponse defines the structure of the JSON response for the `/api/status-data` endpoint.
type ApiResponse struct {
	Timestamp time.Time                `json:"timestamp"`
	Objects   map[string][]StatusEntry `json:"objects"` // Keyed by object type (e.g., "planets", "moons")
}

// InfoPageData holds the data required to render the `info_page.html` template.
type InfoPageData struct {
	Name              string
	DistanceMkm       float64       // Distance from Earth in millions of kilometers
	LatencySec        float64       // One-way latency in seconds
	LatencyFriendly   string        // Human-readable latency (e.g., "5 minutes")
	RoundTripFriendly string        // Human-readable round-trip time
	OccludedClass     string        // CSS class for occlusion status ("status-visible" or "status-occluded")
	OccludedStatus    string        // Textual description of occlusion status
	MoonsHTML         template.HTML // Pre-rendered HTML for the moons list (if any)
	Domain            string        // The domain name for this body (e.g., "mars.latency.space")
}

// Server represents the main latency proxy application.
type Server struct {
	port               int  // Port for the HTTP server (HTTPS uses 443)
	https              bool // Flag indicating whether to enable HTTPS
	metrics            *MetricsCollector
	security           *SecurityValidator
	limiter            *RateLimiter // Per-IP rate/concurrency abuse controls
	dtn                *DTNStore    // Store-and-forward delivery for distant bodies
	httpServer         *http.Server
	httpsServer        *http.Server
	socksListener      net.Listener // Listener for the SOCKS5 server
	httpEnabled        bool         // Whether HTTP/HTTPS should run
	socksEnabled       bool         // Whether SOCKS5 should run
	fixedCelestialBody string       // Fixed celestial body for this instance (empty = dynamic)
}

// NewServer creates and returns a new Server instance.
func NewServer(port int, useHTTPS bool, httpEn bool, socksEn bool, fixedBody string) *Server {
	s := &Server{
		port:               port,
		https:              useHTTPS,
		metrics:            NewMetricsCollector(),
		security:           NewSecurityValidator(),
		limiter:            newRateLimiterFromEnv(),
		httpEnabled:        httpEn,
		socksEnabled:       socksEn,
		fixedCelestialBody: fixedBody,
	}
	// Store-and-forward jobs persist across restarts (DTN latencies span hours to
	// days). Path is overridable for tests/ops via DTN_STORE_PATH.
	storePath := os.Getenv("DTN_STORE_PATH")
	if storePath == "" {
		storePath = "/data/dtn-jobs.json"
	}
	s.dtn = NewDTNStore(storePath, s.security, s.metrics)
	return s
}

// clientIP extracts the bare IP (no port) from a net.Addr string.
func clientIP(remoteAddr string) string {
	if idx := strings.LastIndex(remoteAddr, ":"); idx > 0 {
		return remoteAddr[:idx]
	}
	return remoteAddr
}

// Start initializes and runs the HTTP, HTTPS (if enabled), and SOCKS5 servers.
// It listens for shutdown signals (SIGINT, SIGTERM) for graceful termination.
func (s *Server) Start() error {
	// Channel to listen for OS shutdown signals
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Background janitor to prune idle rate-limiter buckets
	stopCleanup := make(chan struct{})
	defer close(stopCleanup)
	go s.limiter.StartCleanup(stopCleanup)

	// Recover any in-flight store-and-forward jobs and start their retention sweep.
	if s.dtn != nil {
		s.dtn.Start(stopCleanup)
	}

	// Expose Prometheus metrics on a dedicated port (this is what Prometheus
	// scrapes; the /metrics HTTP handler only exists on the proxy's :80/:443 and
	// not on the SOCKS-only containers). Runs in every container. Configurable/
	// disableable via METRICS_ADDR; empty disables it.
	metricsAddr := os.Getenv("METRICS_ADDR")
	if metricsAddr == "" {
		metricsAddr = ":9090"
	}
	if metricsAddr != "-" {
		go s.metrics.ServeMetrics(metricsAddr)
	}

	// Publish current per-body latency as a gauge for the "Solar System Latency"
	// dashboard. Only the main proxy (HTTP enabled) emits it, so there is one
	// series per body instead of one per SOCKS container.
	if s.httpEnabled {
		go func() {
			publish := func() {
				for _, obj := range getCelestialObjects() {
					if d := getCurrentDistance(obj.Name); d > 0 {
						s.metrics.SetBodyLatency(obj.Name, CalculateLatency(d).Seconds())
					}
				}
			}
			publish()
			t := time.NewTicker(30 * time.Second)
			defer t.Stop()
			for {
				select {
				case <-stopCleanup:
					return
				case <-t.C:
					publish()
				}
			}
		}()
	}

	// Use a WaitGroup to wait for server goroutines to finish
	var wg sync.WaitGroup
	// Channel to receive errors from server goroutines
	errCh := make(chan error, 3) // Buffered channel for HTTP, HTTPS, SOCKS errors

	// Start HTTP server in a goroutine (only if HTTP enabled)
	if s.httpEnabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := s.startHTTPServer()
			if err != nil && err != http.ErrServerClosed {
				errCh <- fmt.Errorf("HTTP server error: %v", err)
			}
		}()
		log.Printf("HTTP server starting on port %d", s.port)
	} else {
		log.Printf("HTTP server disabled")
	}

	// Start HTTPS server in a goroutine (only if HTTP and HTTPS both enabled)
	if s.httpEnabled && s.https {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := s.startHTTPSServer()
			if err != nil && err != http.ErrServerClosed {
				errCh <- fmt.Errorf("HTTPS server error: %v", err)
			}
		}()
		log.Printf("HTTPS server starting on port 443")
	} else if s.httpEnabled {
		log.Printf("HTTPS server disabled")
	}

	// Start SOCKS5 server in a goroutine (only if SOCKS enabled)
	if s.socksEnabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := s.startSOCKSServer()
			if err != nil {
				errCh <- fmt.Errorf("SOCKS5 server error: %v", err)
			}
		}()
		log.Printf("SOCKS5 server starting on port 1080")
	} else {
		log.Printf("SOCKS5 server disabled")
	}

	// Wait for signals or errors
	select {
	case <-sigs:
		log.Println("Received shutdown signal")
	case err := <-errCh:
		log.Printf("Server error: %v", err)
	}

	// Graceful shutdown
	s.Stop()
	wg.Wait()
	return nil
}

// Stop gracefully shuts down the server
func (s *Server) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if s.httpServer != nil {
		log.Println("Shutting down HTTP server...")
		if err := s.httpServer.Shutdown(ctx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}
	}

	if s.httpsServer != nil {
		log.Println("Shutting down HTTPS server...")
		if err := s.httpsServer.Shutdown(ctx); err != nil {
			log.Printf("HTTPS server shutdown error: %v", err)
		}
	}

	if s.socksListener != nil {
		log.Println("Shutting down SOCKS5 server...")
		s.socksListener.Close()
	}
}

// handleHTTP processes HTTP requests with celestial body latency
func (s *Server) handleHTTP(w http.ResponseWriter, r *http.Request) {
	//log.Printf("Host %s, Path being accessed: %s", r.Host, r.URL.Path)

	// Special case for metrics endpoint
	if r.URL.Path == "/metrics" {
		promhttp.Handler().ServeHTTP(w, r)
		return
	}

	// API endpoint for status data
	if r.URL.Path == "/api/status-data" {
		s.handleStatusData(w, r)
		return
	}

	// Store-and-forward (DTN) API for bodies too distant to proxy synchronously.
	if strings.HasPrefix(r.URL.Path, "/dtn/") {
		s.handleDTN(w, r)
		return
	}

	// Special case for debug endpoints
	if strings.HasPrefix(r.URL.Path, "/_debug/") {
		s.handleDebugEndpoint(w, r)
		return
	}

	// Handle CORS preflight for API and debug endpoints
	if r.Method == "OPTIONS" && (strings.HasPrefix(r.URL.Path, "/_debug/") || r.URL.Path == "/api/status-data") {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Resolve which celestial body (or moon) this hostname names.
	bodyName := s.resolveCelestialHost(r.Host)
	if bodyName == "" {
		http.Error(w, "Unknown celestial body", http.StatusBadRequest)
		return
	}

	// Over HTTP, latency.space subdomains are purely informational. Actual
	// proxying with light-travel latency is provided by the SOCKS interface
	// (one port per body). The old target-embedding form
	// (target.body.latency.space) was removed: a dotted target sitting under a
	// body can be covered by neither a DNS wildcard nor a TLS wildcard (both
	// match a single label), so those hostnames never resolved in practice.
	s.displayCelestialInfo(w, bodyName)
}

// displayCelestialInfo renders the information page for a celestial body using the template
func (s *Server) displayCelestialInfo(w http.ResponseWriter, name string) {
	// 5. Set Content-Type Header
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// 2. Calculate Data
	distance := getCurrentDistance(name) // km
	latency := CalculateLatency(distance)

	var occluded bool
	var occluderName string
	var occluder CelestialObject // Use struct type to match IsOccluded return type
	targetObject, targetFound := findObjectByName(getCelestialObjects(), name)
	earthObject, earthFound := findObjectByName(getCelestialObjects(), "Earth")

	if targetFound && earthFound {
		occluded, occluder = IsOccluded(earthObject, targetObject, getCelestialObjects(), time.Now())
		// Check if an actual occluding object was returned (Name will be non-empty)
		if occluded && occluder.Name != "" {
			occluderName = occluder.Name
		}
	} else {
		log.Printf("Warning: Could not perform occlusion check for %s (targetFound: %v, earthFound: %v)", name, targetFound, earthFound)
		// Proceed without occlusion data if objects aren't found
	}

	moons := celestial.GetMoons(name)

	// 3. Populate InfoPageData
	var moonsHTML template.HTML
	if len(moons) > 0 {
		var htmlBuilder strings.Builder
		for _, moon := range moons {
			// Construct the moon's domain (e.g., phobos.mars.latency.space)
			moonDomain := FormatMoonDomain(moon.Name, name)
			// Create the list item HTML, linking to the root of the moon's proxy domain
			htmlBuilder.WriteString(fmt.Sprintf(`<li><a href="http://%s/">%s</a></li>`, moonDomain, moon.Name))
		}
		moonsHTML = template.HTML(htmlBuilder.String()) // Convert final string to template.HTML
	}

	data := InfoPageData{
		Name:              name,                                      // Use the original case name for display
		DistanceMkm:       float64(int((distance/1e6)*100)) / 100,    // Convert km to million km with 2 decimal places
		LatencySec:        float64(int(latency.Seconds()*100)) / 100, // One-way latency in seconds with 2 decimal places
		LatencyFriendly:   latency.Round(time.Second).String(),       // Friendly one-way latency
		RoundTripFriendly: (2 * latency).Round(time.Second).String(), // Friendly round-trip latency
		Domain:            FormatFullDomain(name),                    // Formatted domain using utility function
		MoonsHTML:         moonsHTML,                                 // Assign generated HTML
	}

	// Set occlusion status and class based on calculated data
	if occluded {
		data.OccludedClass = "status-occluded"
		if occluderName != "" {
			data.OccludedStatus = fmt.Sprintf("Occluded by %s", occluderName)
		} else {
			data.OccludedStatus = "Occluded (Unknown Occluder)" // Fallback if occluder name is missing
			log.Printf("Warning: Occlusion detected for %s but occluder name is empty.", name)
		}
	} else {
		data.OccludedClass = "status-visible"
		data.OccludedStatus = "Visible"
	}

	// 4. Execute Template
	// Use the globally parsed infoTemplate
	err := infoTemplate.Execute(w, data)
	if err != nil {
		// Log the error
		log.Printf("Error executing info page template for %s: %v", name, err)
		// Attempt to send an error to the client, but only if headers haven't been written.
		// The template engine might have already started writing, so this might fail silently
		// or cause a "superfluous response.WriteHeader call" log, which is acceptable here.
		http.Error(w, "Failed to render information page", http.StatusInternalServerError)
		return // Stop further processing
	}
	// Note: No need to call w.WriteHeader(http.StatusOK) as Execute does this implicitly on success.
}

// resolveCelestialHost resolves a latency.space hostname to the name of the
// celestial body (or moon) it identifies, for that body's information page.
// Only two hostname shapes are recognised:
//
//	body.latency.space           - any non-moon body (planet, dwarf planet, spacecraft, ...)
//	moon.planet.latency.space    - a moon, validated against its parent planet
//
// The former target-embedding shapes (target.body.latency.space) are gone on
// purpose: a dotted target under a body can be covered by neither a DNS nor a
// TLS wildcard, so those hostnames never resolved. Actual proxying is done over
// SOCKS, not by embedding a target in the hostname. Returns "" if the host does
// not name a known body.
func (s *Server) resolveCelestialHost(host string) string {
	// Remove port from host if present
	if idx := strings.Index(host, ":"); idx > 0 {
		host = host[:idx]
	}

	// Ensure celestial objects are initialized
	if getCelestialObjects() == nil {
		setCelestialObjects(celestial.InitSolarSystemObjects())
	}

	// Must end with ".latency.space" (case-insensitive)
	suffix := ".latency.space"
	if len(host) < len(suffix) || !strings.EqualFold(host[len(host)-len(suffix):], suffix) {
		return ""
	}

	parts := strings.Split(host, ".")
	numParts := len(parts)
	if numParts < 3 || !strings.EqualFold(parts[numParts-1], "space") || !strings.EqualFold(parts[numParts-2], "latency") {
		return ""
	}

	switch numParts {
	case 3:
		// body.latency.space - any non-moon body.
		if body, found := findObjectByName(getCelestialObjects(), parts[0]); found && !strings.EqualFold(body.Type, "moon") {
			return body.Name
		}
	case 4:
		// moon.planet.latency.space - moon validated against its parent planet.
		moon, moonFound := findObjectByName(getCelestialObjects(), parts[0])
		planet, planetFound := findObjectByName(getCelestialObjects(), parts[1])
		if moonFound && planetFound &&
			moon.Type == "moon" &&
			(planet.Type == "planet" || planet.Type == "dwarf_planet") &&
			strings.EqualFold(moon.ParentName, planet.Name) {
			return moon.Name
		}
	}

	return ""
}

func (s *Server) startHTTPServer() error {
	addr := fmt.Sprintf(":%d", s.port)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      http.HandlerFunc(s.handleHTTP),
		ReadTimeout:  60 * time.Minute,  // Increased for distant celestial bodies
		WriteTimeout: 60 * time.Minute,  // Increased for distant celestial bodies
		IdleTimeout:  120 * time.Minute, // Allow long-lived connections
	}

	log.Printf("Starting HTTP server on %s", addr)
	err := s.httpServer.ListenAndServe()
	log.Printf("HTTP server stopped: %v", err) // This will tell you if the server stops
	return err
}

func (s *Server) startHTTPSServer() error {
	nullLogger := log.New(io.Discard, "", 0)
	s.httpsServer = &http.Server{
		Addr:         ":443",
		Handler:      http.HandlerFunc(s.handleHTTP),
		TLSConfig:    setupTLS(),
		ErrorLog:     nullLogger,        // don't really need these errors right now
		ReadTimeout:  60 * time.Minute,  // Increased for distant celestial bodies
		WriteTimeout: 60 * time.Minute,  // Increased for distant celestial bodies
		IdleTimeout:  120 * time.Minute, // Allow long-lived connections
	}

	log.Printf("Starting HTTPS server on :443")
	return s.httpsServer.ListenAndServeTLS("", "") // Certificates handled by autocert
}

func (s *Server) startSOCKSServer() error {
	// Start SOCKS5 server on port 1080
	// Create a custom TCP listener with extended keepalive settings
	tcpAddr, err := net.ResolveTCPAddr("tcp", ":1080")
	if err != nil {
		return fmt.Errorf("failed to resolve TCP address: %v", err)
	}

	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on SOCKS port: %v", err)
	}

	log.Printf("SOCKS server using extended timeouts for interplanetary latency")
	s.socksListener = listener

	log.Printf("Starting SOCKS5 server on :1080")

	for {
		conn, err := listener.Accept()
		if err != nil {
			// Check if the listener was closed
			if strings.Contains(err.Error(), "use of closed network connection") {
				return nil
			}
			log.Printf("Failed to accept SOCKS connection: %v", err)
			continue
		}

		// Configure extended timeouts for TCP connections
		if tcpConn, ok := conn.(*net.TCPConn); ok {
			// Set keep-alive with a long period suitable for celestial distances
			if err := tcpConn.SetKeepAlive(true); err != nil {
				log.Printf("Warning: Failed to set TCP keepalive: %v", err)
			}
			if err := tcpConn.SetKeepAlivePeriod(10 * time.Minute); err != nil {
				log.Printf("Warning: Failed to set TCP keepalive period: %v", err)
			}

			// Disable Nagle's algorithm for low-latency transmission
			if err := tcpConn.SetNoDelay(true); err != nil {
				log.Printf("Warning: Failed to disable Nagle's algorithm: %v", err)
			}

			log.Printf("SOCKS: Configured extended timeouts for connection from %s", conn.RemoteAddr().String())
		}

		// Get client IP for rate limiting
		ip := clientIP(conn.RemoteAddr().String())

		if !s.security.IsAllowedIP(ip) {
			conn.Close()
			continue
		}

		// Abuse control: per-IP rate and concurrency limits. SOCKS bypasses
		// the front-end nginx, so this is the only such control on this path.
		release, err := s.limiter.Acquire(ip)
		if err != nil {
			log.Printf("SOCKS connection from %s rejected: %v", ip, err)
			conn.Close()
			continue
		}

		// Handle the connection in a goroutine
		// Pass the fixed celestial body if configured
		go func() {
			defer release()
			NewSOCKSHandler(conn, s.security, s.metrics, s.fixedCelestialBody).Handle()
		}()
	}
}

// handleDebugEndpoint handles debug and info endpoints
func (s *Server) handleDebugEndpoint(w http.ResponseWriter, r *http.Request) {
	// Enable CORS for debug endpoints
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")

	// Extract the debug command
	path := strings.TrimPrefix(r.URL.Path, "/_debug/")

	switch path {
	case "metrics":
		promhttp.Handler().ServeHTTP(w, r)
	case "distances":
		s.printCelestialDistances(w)
	case "allowed-hosts":
		s.printAllowedHosts(w)
	case "help":
		s.printHelp(w)
	default:
		http.Error(w, "Unknown debug command: "+path, http.StatusNotFound)
	}
}

// printAllowedHosts lists the destination allowlist (hosts and ports) as JSON.
// The proxy only relays to these hosts; operators can extend the list via the
// ALLOWED_HOSTS environment variable.
func (s *Server) printAllowedHosts(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	payload := map[string]interface{}{
		"note":  "The proxy only relays to these hosts (and their subdomains) on these ports. Extend via the ALLOWED_HOSTS env var or a PR to security.go.",
		"hosts": s.security.AllowedHosts(),
		"ports": s.security.AllowedPorts(),
	}
	jsonData, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(jsonData)
}

// printCelestialDistances shows the current distances of all celestial bodies
func (s *Server) printCelestialDistances(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain")

	fmt.Fprintln(w, "Latency Space - Current Celestial Distances")
	fmt.Fprintln(w, "============================================")
	fmt.Fprintf(w, "Current Time: %s\n\n", time.Now().Format(time.RFC3339))

	// Call printObjectsByType without distanceEntries argument, as it now uses the global cache
	printObjectsByType(w, "planet")
	printObjectsByType(w, "moon")
	printObjectsByType(w, "asteroid")
	printObjectsByType(w, "dwarf_planet")
	printObjectsByType(w, "spacecraft")

}

// handleStatusData provides celestial body status data as JSON
func (s *Server) handleStatusData(w http.ResponseWriter, r *http.Request) {
	// Set CORS and Content-Type headers
	w.Header().Set("Access-Control-Allow-Origin", "*") // Allow requests from any origin
	w.Header().Set("Content-Type", "application/json")

	// Ensure distance data is up-to-date
	now := time.Now()
	calculateDistancesFromEarth(getCelestialObjects(), now) // Refresh cache
	log.Printf("DEBUG: distanceEntries after calculation: %+v\n", distanceEntries)

	// Prepare the response structure
	response := ApiResponse{
		Timestamp: now,
		Objects:   make(map[string][]StatusEntry),
	}

	// Acquire read lock to safely access distanceEntries
	DistanceCacheMutex.RLock()
	defer DistanceCacheMutex.RUnlock() // Ensure lock is released

	// Populate the response data
	for _, obj := range getCelestialObjects() {
		if obj.Type == "star" { // Skip the Sun for this endpoint
			continue
		}

		// Find the corresponding distance entry by iterating through the slice (under read lock)
		var distance float64
		var occluded bool
		var found bool // Flag to track if the entry was found

		for _, entry := range distanceEntries { // Accessing shared data
			// Compare names case-insensitively
			if strings.EqualFold(entry.Object.Name, obj.Name) {
				distance = entry.Distance
				occluded = entry.Occluded
				found = true
				break // Found the matching entry, exit the inner loop
			}
		}

		// Check if the entry was found in the slice
		if !found {
			log.Printf("Warning: Failed to find distance entry for obj.Name='%s' in distanceEntries lookup. Skipping object.", obj.Name)
			continue // Skip this object if no distance data is found
		}

		// Calculate latency using the found distance
		latency := CalculateLatency(distance)

		// Create the status entry using the found data
		entry := StatusEntry{
			Name:       obj.Name,
			Type:       obj.Type,
			ParentName: obj.ParentName,
			Distance:   float64(int(distance*100)) / 100,              // Limit distance to 2 decimal places
			Latency:    float64(int((latency/time.Second)*100)) / 100, // Limit latency to 2 decimal places
			Occluded:   occluded,
		}

		// Group objects by type
		objectTypeKey := obj.Type + "s" // e.g., "planets", "moons"
		response.Objects[objectTypeKey] = append(response.Objects[objectTypeKey], entry)
	}

	// Add debug log before marshaling
	log.Printf("DEBUG: API Response data before marshaling: %+v\n", response)

	// Marshal the response to JSON
	jsonData, err := json.MarshalIndent(response, "", "  ") // Use Indent for readability
	if err != nil {
		log.Printf("Error marshaling status data to JSON: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Write the JSON response
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(jsonData)
	if err != nil {
		log.Printf("Error writing JSON response for status data: %v", err)
	}
}

// printHelp displays usage information
func (s *Server) printHelp(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain")

	fmt.Fprintln(w, "Latency Space - Interplanetary Internet Simulator")
	fmt.Fprintln(w, "===============================================")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "This service simulates the latency of Internet access from different")
	fmt.Fprintln(w, "celestial bodies in our solar system.")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Proxy Usage (SOCKS5):")
	fmt.Fprintln(w, "---------------------")
	fmt.Fprintln(w, "Proxying is done over SOCKS5, one port per body (Mars 1080, Moon 1081, ...).")
	fmt.Fprintln(w, "  curl --socks5-hostname mars.latency.space:1080 https://example.com")
	fmt.Fprintln(w, "TCP via CONNECT, UDP via UDP ASSOCIATE. Near bodies only - distant bodies")
	fmt.Fprintln(w, "have latencies that exceed normal client timeouts.")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Body Pages (HTTP, informational):")
	fmt.Fprintln(w, "---------------------------------")
	fmt.Fprintln(w, "https://mars.latency.space/           - a body's info page")
	fmt.Fprintln(w, "https://phobos.mars.latency.space/    - a moon (under its planet)")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Store-and-Forward (DTN) - for distant bodies:")
	fmt.Fprintln(w, "---------------------------------------------")
	fmt.Fprintln(w, "Distant bodies (hours/days away) exceed live-proxy timeouts, so requests")
	fmt.Fprintln(w, "are delivered asynchronously - submit now, poll for the response later:")
	fmt.Fprintln(w, "  POST https://voyager-1.latency.space/dtn/send   {\"url\":\"https://example.com/\"}")
	fmt.Fprintln(w, "  GET  https://voyager-1.latency.space/dtn/status/{id}")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Debug Endpoints:")
	fmt.Fprintln(w, "---------------")
	fmt.Fprintln(w, "/_debug/distances - Current distances and latencies")
	fmt.Fprintln(w, "/_debug/allowed-hosts - Destination allowlist (hosts and ports)")
	fmt.Fprintln(w, "/_debug/help - This help information")
}

func main() {
	// Parse command-line arguments
	port := flag.Int("port", 80, "HTTP port to listen on")
	https := flag.Bool("https", true, "Enable HTTPS")
	flag.Parse()

	// Read environment variables for configuration
	fixedCelestialBody := os.Getenv("CELESTIAL_BODY")

	httpEnabledStr := os.Getenv("HTTP_ENABLED")
	httpEnabled := httpEnabledStr != "false" // Default to true unless explicitly "false"

	socksEnabledStr := os.Getenv("SOCKS_ENABLED")
	socksEnabled := socksEnabledStr != "false" // Default to true unless explicitly "false"

	// Log configuration
	log.Printf("===== latency.space Proxy Configuration =====")
	log.Printf("  HTTP/HTTPS Enabled: %v", httpEnabled)
	log.Printf("  SOCKS5 Enabled: %v", socksEnabled)
	if fixedCelestialBody != "" {
		log.Printf("  Fixed Celestial Body: %s", fixedCelestialBody)
	} else {
		log.Printf("  Celestial Body: Dynamic (detected from hostname)")
	}
	log.Printf("==============================================")

	// Parse the info page template at startup (only if HTTP enabled)
	var err error
	if httpEnabled {
		// Try different paths for the template (container paths first, then local development paths)
		templatePaths := []string{
			"/app/templates/info_page.html",      // Docker container path (new)
			"templates/info_page.html",           // Relative path
			"src/templates/info_page.html",       // Another relative path
			"proxy/src/templates/info_page.html", // Original path
		}

		var templateErr error
		for _, path := range templatePaths {
			infoTemplate, templateErr = template.ParseFiles(path)
			if templateErr == nil {
				log.Printf("Successfully loaded template from: %s", path)
				break
			}
		}

		if infoTemplate == nil {
			log.Fatalf("Failed to parse info page template: %v", templateErr)
		}
	} else {
		log.Printf("HTTP disabled, skipping template loading")
	}

	// Initialize celestial objects for calculation
	setCelestialObjects(celestial.InitSolarSystemObjects())

	// Validate fixed celestial body if set
	if fixedCelestialBody != "" {
		_, found := findObjectByName(getCelestialObjects(), fixedCelestialBody)
		if !found {
			log.Fatalf("Invalid CELESTIAL_BODY: '%s' not found in solar system objects", fixedCelestialBody)
		}
		log.Printf("Validated fixed celestial body: %s", fixedCelestialBody)
	}

	// Create and start the server
	server := NewServer(*port, *https, httpEnabled, socksEnabled, fixedCelestialBody)
	err = server.Start() // Use = instead of := as err is already declared
	if err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
