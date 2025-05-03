// proxy/src/main.go
package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"encoding/json"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Global variable to hold the parsed info page template
var infoTemplate *template.Template

// Ensure tls package is used - needed for TLS configuration
var _ = tls.Config{}

// StatusEntry represents the data for a single celestial object in the status API
type StatusEntry struct {
	Name       string        `json:"name"`
	Type       string        `json:"type"`
	ParentName string        `json:"parentName,omitempty"` // Omit if empty
	Distance   float64       `json:"distance_km"`
	Latency    float64       `json:"latency_seconds"` // Changed type to float64
	Occluded   bool          `json:"occluded"`
}

// ApiResponse is the structure for the /api/status-data endpoint response
type ApiResponse struct {
	Timestamp time.Time              `json:"timestamp"`
	Objects   map[string][]StatusEntry `json:"objects"` // Keyed by object type (e.g., "planets", "moons")
}

// InfoPageData holds the data needed to render the celestial body info page template
type InfoPageData struct {
	Name              string
	DistanceMkm       float64 // Distance in millions of kilometers
	LatencySec        float64 // One-way latency in seconds
	LatencyFriendly   string  // Human-readable latency (e.g., "5 minutes")
	RoundTripFriendly string  // Human-readable round-trip time
	OccludedClass     string  // CSS class for occlusion status ("status-visible" or "status-occluded")
	OccludedStatus    string  // Textual description of occlusion status
	MoonsHTML         template.HTML // Pre-rendered HTML for the moons list (if any)
	Domain            string  // The domain name for this body (e.g., "mars.latency.space")
}

// Server is the main latency proxy server
type Server struct {
	port          int
	https         bool
	metrics       *MetricsCollector
	security      *SecurityValidator
	httpServer    *http.Server
	httpsServer   *http.Server
	socksListener net.Listener
}

// NewServer creates a new latency proxy server
func NewServer(port int, useHTTPS bool) *Server {
	return &Server{
		port:     port,
		https:    useHTTPS,
		metrics:  NewMetricsCollector(),
		security: NewSecurityValidator(),
	}
}

// Start runs the latency proxy server
func (s *Server) Start() error {
	// Set up signal channel for graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Start proxy servers
	var wg sync.WaitGroup
	errCh := make(chan error, 3) // Buffer for HTTP, HTTPS, and SOCKS errors

	// Start HTTP server
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := s.startHTTPServer()
		if err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("HTTP server error: %v", err)
		}
	}()

	// Start HTTPS server if enabled
	if s.https {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := s.startHTTPSServer()
			if err != nil && err != http.ErrServerClosed {
				errCh <- fmt.Errorf("HTTPS server error: %v", err)
			}
		}()
	}

	// Start SOCKS5 server
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := s.startSOCKSServer()
		if err != nil {
			errCh <- fmt.Errorf("SOCKS5 server error: %v", err)
		}
	}()

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

	// Process the host to determine if this is a celestial body request
	targetURL, _, bodyName := s.parseHostForCelestialBody(r.Host, r.URL)

	// Check if celestial body exists
	if bodyName == "" {
		http.Error(w, "Unknown celestial body", http.StatusBadRequest)
		return
	}

	log.Printf("Accessing for |%s|, via body |%s|", targetURL, bodyName)

	if celestialObjects == nil {
		log.Printf("Init celestial objects")
		celestialObjects = InitSolarSystemObjects()
	}

	// If there's no target URL, just display info about this celestial body
	if targetURL == "" || targetURL == "/" {
		s.displayCelestialInfo(w, bodyName)
		return
	}

	// Find the target and Earth objects
	targetObject, targetFound := findObjectByName(celestialObjects, bodyName)
	if !targetFound {
		log.Printf("Error: Target celestial body '%s' not found after host parsing.", bodyName)
		http.Error(w, "Internal server error: Target body not found", http.StatusInternalServerError)
		return
	}
	earthObject, earthFound := findObjectByName(celestialObjects, "Earth")
	if !earthFound {
		log.Printf("Error: Earth celestial object not found.")
		http.Error(w, "Internal server error: Earth object configuration missing", http.StatusInternalServerError)
		return
	}

	// Check for occlusion
	occluded, occluder := IsOccluded(earthObject, targetObject, celestialObjects, time.Now())
	if occluded {
		// If occluded is true, occluder is guaranteed to be non-nil by IsOccluded
		log.Printf("HTTP connection to %s rejected: occluded by %s", bodyName, occluder.Name)
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, "Connection refused: Target body '%s' is occluded by '%s'.", bodyName, occluder.Name)
		return
	}

	// Apply space latency
	distance := getCurrentDistance(bodyName) // Use the existing function for distance
	latency := CalculateLatency(distance)

	// Anti-DDoS: Only allow bodies with significant latency (>1s)
	// This prevents the proxy from being used for DDoS attacks
	if latency < 1*time.Second {
		log.Printf("Rejecting connection with insufficient latency: %s (%.2f ms)",
			bodyName, latency.Seconds()*1000)
		http.Error(w, "rejecting request with insufficient latency", http.StatusBadRequest)
		return
	}

	log.Printf("Proxy request for %s via %s (latency: %v)", targetURL, bodyName, latency)
	time.Sleep(latency)

	// Start metrics collection
	start := time.Now()
	defer func() {
		s.metrics.RecordRequest(bodyName, "http", time.Since(start))
	}()

	// Apply bandwidth limiting
	r.Header.Set("X-Celestial-Body", bodyName)

	// Forward the request to the target URL
	client := &http.Client{
		Timeout: latency * 2 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:    10,
			IdleConnTimeout: latency * 2 * time.Second,
		},
	}

	// Create a new request to the target URL
	proxyReq, err := http.NewRequest(r.Method, targetURL, r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error creating proxy request: %v", err), http.StatusInternalServerError)
		return
	}

	// Copy headers from original request
	for name, values := range r.Header {
		// Skip host header (case-insensitive check)
		if !strings.EqualFold(name, "host") {
			for _, value := range values {
				proxyReq.Header.Add(name, value)
			}
		}
	}

	// Make the request
	resp, err := client.Do(proxyReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error making proxy request: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Copy headers from response
	for name, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}

	// Apply interplanetary latency on the return path too
	time.Sleep(latency)

	// Copy status code
	w.WriteHeader(resp.StatusCode)

	// Copy body and check for errors
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.Printf("Error copying response body: %v", err)
	}
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
	targetObject, targetFound := findObjectByName(celestialObjects, name)
	earthObject, earthFound := findObjectByName(celestialObjects, "Earth")

	if targetFound && earthFound {
		occluded, occluder = IsOccluded(earthObject, targetObject, celestialObjects, time.Now())
		// Check if an actual occluding object was returned (Name will be non-empty)
		if occluded && occluder.Name != "" {
			occluderName = occluder.Name
		}
	} else {
		log.Printf("Warning: Could not perform occlusion check for %s (targetFound: %v, earthFound: %v)", name, targetFound, earthFound)
		// Proceed without occlusion data if objects aren't found
	}

	moons := GetMoons(name)

	// 3. Populate InfoPageData
	var moonsHTML template.HTML
	if len(moons) > 0 {
		var htmlBuilder strings.Builder
		for _, moon := range moons {
			// Construct the moon's domain (e.g., phobos.mars.latency.space)
			// Ensure both moon and planet names are lowercase for domain consistency
			moonDomain := fmt.Sprintf("%s.%s.latency.space", strings.ToLower(moon.Name), strings.ToLower(name))
			// Create the list item HTML, linking to the root of the moon's proxy domain
			htmlBuilder.WriteString(fmt.Sprintf(`<li><a href="http://%s/">%s</a></li>`, moonDomain, moon.Name))
		}
		moonsHTML = template.HTML(htmlBuilder.String()) // Convert final string to template.HTML
	}

	data := InfoPageData{
		Name:              name,                                        // Use the original case name for display
		DistanceMkm:       distance / 1e6,                              // Convert km to million km
		LatencySec:        latency.Seconds(),                           // One-way latency in seconds
		LatencyFriendly:   latency.Round(time.Second).String(),         // Friendly one-way latency
		RoundTripFriendly: (2 * latency).Round(time.Second).String(), // Friendly round-trip latency
		Domain:            fmt.Sprintf("%s.latency.space", strings.ToLower(name)), // Lowercase domain
		MoonsHTML:         moonsHTML,                                   // Assign generated HTML
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

// parseHostForCelestialBody extracts target URL and celestial body from request
func (s *Server) parseHostForCelestialBody(host string, reqURL *url.URL) (string, CelestialObject, string) {
	// Remove port from host if present
	if idx := strings.Index(host, ":"); idx > 0 {
		host = host[:idx]
	}

	// Ensure celestial objects are initialized
	if celestialObjects == nil {
		celestialObjects = InitSolarSystemObjects()
	}

	// Check if it's a latency.space domain (case-insensitive manual check)
	suffix := ".latency.space"
	if len(host) < len(suffix) || !strings.EqualFold(host[len(host)-len(suffix):], suffix) {
		// Host does NOT end with ".latency.space" case-insensitively
		return "", CelestialObject{}, ""
	}

	// Extract parts: [..., body, latency, space] or [..., moon, planet, latency, space]
	parts := strings.Split(host, ".")
	numParts := len(parts)

	// Basic validation (using strings.EqualFold for case-insensitive checks)
	if numParts < 3 || !strings.EqualFold(parts[numParts-1], "space") || !strings.EqualFold(parts[numParts-2], "latency") {
		return "", CelestialObject{}, "" // Invalid format: doesn't end in .latency.space
	}

	// Case 1: [target].[moon].[planet].latency.space (>= 5 parts)
	// Case 2: [moon].[planet].latency.space (4 parts, target is empty)
	if numParts >= 4 {
		potentialMoonName := parts[numParts-4]
		potentialPlanetName := parts[numParts-3]

		moon, moonFound := findObjectByName(celestialObjects, potentialMoonName)
		planet, planetFound := findObjectByName(celestialObjects, potentialPlanetName)

		// If both potential moon and planet are found, perform strict validation
		if moonFound && planetFound {
			// 1. Check if the identified 'moon' is actually a moon type
			if moon.Type != "moon" {
				// If not a moon, this format is invalid, return empty
				return "", CelestialObject{}, ""
			}
			// 2. Check if the identified 'planet' is a valid parent type
			if !(planet.Type == "planet" || planet.Type == "dwarf_planet") {
				// If the parent is not a planet/dwarf_planet, invalid format
                return "", CelestialObject{}, ""
			}
            // 3. Check if the moon's parent matches the identified planet (case-insensitive)
            if !strings.EqualFold(moon.ParentName, planet.Name) {
                // Invalid parent relationship, return empty
                return "", CelestialObject{}, ""
            }

            // If all checks pass, proceed to extract target and return moon
			targetDomain := ""
			if numParts >= 5 { // Only extract target if there are enough parts
				targetDomain = strings.Join(parts[:numParts-4], ".")
			}
			// Return the moon as the final body
			return targetDomain, moon, moon.Name
		}
	}

	// Case 3: [target].[planet].latency.space (>= 4 parts)
	// Case 4: [planet].latency.space (3 parts, target is empty)
	if numParts >= 3 {
		potentialBodyName := parts[numParts-3]
		body, bodyFound := findObjectByName(celestialObjects, potentialBodyName)

		// Check if body is found and is not a moon (case-insensitive check to avoid conflict with moon.planet format)
		if bodyFound && !strings.EqualFold(body.Type, "moon") {
			targetDomain := ""
			if numParts >= 4 { // Only extract target if there are enough parts
				targetDomain = strings.Join(parts[:numParts-3], ".")
			}
			// Return the planet/other body
			return targetDomain, body, body.Name
		}
	}

	// If none of the specific formats match, try the simple [body].latency.space format
	// This handles the case where someone just goes to mars.latency.space
	if numParts == 3 {
		potentialBodyName := parts[0]
		body, bodyFound := findObjectByName(celestialObjects, potentialBodyName)
		if bodyFound {
			return "", body, body.Name
		}
	}

	// If no valid format is found
	return "", CelestialObject{}, ""
}

func (s *Server) startHTTPServer() error {
	s.httpServer = &http.Server{
		Addr:         ":80",
		Handler:      http.HandlerFunc(s.handleHTTP),
		ReadTimeout:  60 * time.Minute,  // Increased for distant celestial bodies
		WriteTimeout: 60 * time.Minute,  // Increased for distant celestial bodies
		IdleTimeout:  120 * time.Minute, // Allow long-lived connections
	}

	log.Printf("Starting HTTP server on :80")
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
		ErrorLog: nullLogger, // don't really need these errors right now
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
		clientIP := conn.RemoteAddr().String()
		if idx := strings.Index(clientIP, ":"); idx > 0 {
			clientIP = clientIP[:idx]
		}

		// Apply rate limiting based on IP (simplified)
		// This is a basic form of rate limiting to prevent abuse
		if !s.security.IsAllowedIP(clientIP) {
			conn.Close()
			continue
		}

		// Handle the connection in a goroutine
		go NewSOCKSHandler(conn, s.security, s.metrics).Handle()
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
	case "help":
		s.printHelp(w)
	default:
		http.Error(w, "Unknown debug command: " + path, http.StatusNotFound)
	}
}

// printCelestialDistances shows the current distances of all celestial bodies
func (s *Server) printCelestialDistances(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain")

	fmt.Fprintln(w, "Latency Space - Current Celestial Distances")
	fmt.Fprintln(w, "============================================")
	fmt.Fprintf(w, "Current Time: %s\n\n", time.Now().Format(time.RFC3339))

	printObjectsByType(w, distanceEntries, "planet")
	printObjectsByType(w, distanceEntries, "moon")
	printObjectsByType(w, distanceEntries, "asteroid")
	printObjectsByType(w, distanceEntries, "dwarf_planet")
	printObjectsByType(w, distanceEntries, "spacecraft")

}

// handleStatusData provides celestial body status data as JSON
func (s *Server) handleStatusData(w http.ResponseWriter, r *http.Request) {
	// Set CORS and Content-Type headers
	w.Header().Set("Access-Control-Allow-Origin", "*") // Allow requests from any origin
	w.Header().Set("Content-Type", "application/json")

	// Ensure distance data is up-to-date
	now := time.Now()
	calculateDistancesFromEarth(celestialObjects, now) // Refresh cache
	log.Printf("DEBUG: distanceEntries after calculation: %+v\n", distanceEntries)

	// Prepare the response structure
	response := ApiResponse{
		Timestamp: now,
		Objects:   make(map[string][]StatusEntry),
	}

	// Populate the response data
	for _, obj := range celestialObjects {
		if obj.Type == "star" { // Skip the Sun for this endpoint
			continue
		}

		// Find the corresponding distance entry by iterating through the slice
		var distance float64
		var occluded bool
		var found bool // Flag to track if the entry was found

		for _, entry := range distanceEntries {
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
			Distance:   distance,
			Latency:    float64(latency / time.Second), // Explicitly convert to float64
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
	fmt.Fprintln(w, "HTTP Proxy Usage:")
	fmt.Fprintln(w, "----------------")
	fmt.Fprintln(w, "1. Direct URL: http://mars.latency.space/http://example.com")
	fmt.Fprintln(w, "2. Domain format: http://example.com.mars.latency.space/")
	fmt.Fprintln(w, "3. Query parameter: http://mars.latency.space/?url=http://example.com")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "SOCKS5 Proxy:")
	fmt.Fprintln(w, "------------")
	fmt.Fprintln(w, "Host: mars.latency.space (or any celestial body subdomain)")
	fmt.Fprintln(w, "Port: 1080")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Debug Endpoints:")
	fmt.Fprintln(w, "---------------")
	fmt.Fprintln(w, "/_debug/distances - Current distances and latencies")
	fmt.Fprintln(w, "/_debug/help - This help information")
}

func main() {
	// Parse command-line arguments
	port := flag.Int("port", 80, "HTTP port to listen on")
	https := flag.Bool("https", true, "Enable HTTPS")

	flag.Parse()

	// Parse the info page template at startup
	var err error
	infoTemplate, err = template.ParseFiles("proxy/src/templates/info_page.html")
	if err != nil {
		log.Fatalf("Failed to parse info page template: %v", err)
	}

	celestialObjects = InitSolarSystemObjects()

	// Create and start the server
	server := NewServer(*port, *https)
	err = server.Start() // Use = instead of := as err is already declared
	if err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
