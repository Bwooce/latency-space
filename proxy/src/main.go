// proxy/src/main.go
package main

import (
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Server struct {
	httpServer    *http.Server
	httpsServer   *http.Server
	socksListener net.Listener
	security      *SecurityValidator
	metrics       *MetricsCollector
	rateLimits    map[string]time.Time // IP address -> last request time
	rateLimitMu   sync.Mutex           // Mutex for rate limiting map
	wg            sync.WaitGroup
}

func NewServer() *Server {
	return &Server{
		security:   NewSecurityValidator(),
		metrics:    NewMetricsCollector(),
		rateLimits: make(map[string]time.Time),
	}
}

// isRateLimited implements rate limiting for each IP address
// Returns true if the request should be limited
func (s *Server) isRateLimited(clientIP string) bool {
	s.rateLimitMu.Lock()
	defer s.rateLimitMu.Unlock()
	
	// Clean up old entries every 100 requests (approximately)
	if len(s.rateLimits) > 100 && rand.Intn(100) == 0 {
		now := time.Now()
		for ip, lastTime := range s.rateLimits {
			if now.Sub(lastTime) > 5*time.Minute {
				delete(s.rateLimits, ip)
			}
		}
	}
	
	// Check if this IP has requested recently
	lastTime, exists := s.rateLimits[clientIP]
	now := time.Now()
	
	// Allow 1 request per 2 seconds per IP
	if exists && now.Sub(lastTime) < 2*time.Second {
		return true // Rate limited
	}
	
	// Update last request time
	s.rateLimits[clientIP] = now
	return false // Not rate limited
}

func (s *Server) handleHTTP(w http.ResponseWriter, r *http.Request) {
	// Handle CORS preflight for debug endpoints
	if r.Method == "OPTIONS" && strings.HasPrefix(r.URL.Path, "/_debug/") {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusOK)
		return
	}
	
	// Add CORS headers for all debug endpoints
	if strings.HasPrefix(r.URL.Path, "/_debug/") {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	}
	
	// Add anti-DDoS protection
	clientIP := r.Header.Get("X-Forwarded-For")
	if clientIP == "" {
		clientIP = r.RemoteAddr
	}
	
	// Rate limiting per source IP
	if s.isRateLimited(clientIP) {
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	if strings.HasPrefix(r.URL.Path, "/_debug/") {
		s.handleDebug(w, r)
		return
	}

	// Extract target domain from the hostname
	targetDomain, celestialBody, bodyName := s.extractDomainAndBody(r.Host)
	if celestialBody == nil {
		http.Error(w, "Unknown celestial body", http.StatusBadRequest)
		return
	}

	// Anti-DDoS: Only allow bodies with significant latency (>1s)
	latency := calculateLatency(celestialBody.Distance * 1e6)
	if latency < 1*time.Second {
		http.Error(w, "This proxy is only for simulating deep space latency. Please use a body with >1s latency.", http.StatusForbidden)
		return
	}

	// If target domain is present (DNS proxy format), use it as destination
	var destination string
	if targetDomain != "" {
		// Construct destination URL
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		destination = fmt.Sprintf("%s://%s", scheme, targetDomain)
	} else {
		// Get destination from header or query param
		destination = r.Header.Get("X-Destination")
		if destination == "" {
			destination = r.URL.Query().Get("destination")
		}
	}

	// Validate destination
	validDest, err := s.security.ValidateDestination(destination)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid destination: %v", err), http.StatusBadRequest)
		return
	}

	// Anti-DDoS: Restrict destinations to common well-known sites
	// This protects against using the proxy to attack less-protected sites
	if !s.security.IsAllowedHost(validDest) {
		http.Error(w, "Destination not in allowed list", http.StatusForbidden)
		return
	}

	// Check for WebSocket upgrade
	if websocket.IsWebSocketUpgrade(r) {
		s.handleWebSocket(w, r, celestialBody, validDest)
		return
	}

	// Start metrics collection
	start := time.Now()
	defer func() {
		s.metrics.RecordRequest(bodyName, "http", time.Since(start))
	}()

	// Create reverse proxy
	target, _ := url.Parse(validDest)
	proxy := httputil.NewSingleHostReverseProxy(target)

	// Modify the request
	r.URL.Host = target.Host
	r.URL.Scheme = target.Scheme
	r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
	r.Host = target.Host

	// Apply space latency
	time.Sleep(latency)

	// Apply bandwidth limiting
	s.metrics.TrackBandwidth(bodyName, r.ContentLength)

	// Forward the request
	proxy.ServeHTTP(w, r)
}

// extractDomainAndBody parses domain.body.latency.space format
// returns the target domain and celestial body
func (s *Server) extractDomainAndBody(host string) (string, *CelestialBody, string) {
	// Remove port if present
	if idx := strings.Index(host, ":"); idx > 0 {
		host = host[:idx]
	}

	// Check for latency.space domain
	if !strings.HasSuffix(host, ".latency.space") {
		return "", nil, ""
	}

	// Split hostname parts
	parts := strings.Split(host, ".")
	if len(parts) < 3 {
		return "", nil, ""
	}

	// If format is domain.body.latency.space
	// Extract the celestial body and target domain
	if len(parts) >= 3 {
		// The celestial body is the second-to-last part before "latency.space"
		bodyIndex := len(parts) - 3
		
		// Everything before the celestial body is the target domain
		targetParts := parts[:bodyIndex]
		targetDomain := strings.Join(targetParts, ".")
		
		// Get the celestial body
		bodyName := parts[bodyIndex]
		celestialBody, bodyFullName := getCelestialBody(bodyName)
		
		if celestialBody != nil {
			return targetDomain, celestialBody, bodyFullName
		}
	}

	// Default behavior for standard celestial body subdomains
	hostParts := strings.Split(host, ".")
	if len(hostParts) > 0 {
		body, bodyName := getCelestialBody(hostParts[0])
		return "", body, bodyName
	}

	return "", nil, ""
}

func (s *Server) startHTTPServer() error {
	s.httpServer = &http.Server{
		Addr:         ":80",
		Handler:      http.HandlerFunc(s.handleHTTP),
		ReadTimeout:  5 * time.Minute,
		WriteTimeout: 5 * time.Minute,
	}

	log.Printf("Starting HTTP server on :80")
	return s.httpServer.ListenAndServe()
}

func (s *Server) startHTTPSServer() error {
	s.httpsServer = &http.Server{
		Addr:      ":443",
		Handler:   http.HandlerFunc(s.handleHTTP),
		TLSConfig: setupTLS(),
	}

	log.Printf("Starting HTTPS server on :443")
	return s.httpsServer.ListenAndServeTLS("", "") // Certificates handled by autocert
}

func (s *Server) startSOCKSServer() error {
	// Start SOCKS5 server on port 1080
	listener, err := net.Listen("tcp", ":1080")
	if err != nil {
		return fmt.Errorf("failed to listen on SOCKS port: %v", err)
	}
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

		// Get client IP for rate limiting
		clientIP := conn.RemoteAddr().String()
		if idx := strings.Index(clientIP, ":"); idx > 0 {
			clientIP = clientIP[:idx]
		}
		
		// Apply rate limiting
		if s.isRateLimited(clientIP) {
			log.Printf("Rate limiting SOCKS connection from %s", clientIP)
			conn.Close()
			continue
		}

		// Handle each connection in a separate goroutine
		go func(c net.Conn) {
			handler := NewSOCKSHandler(c, s.security, s.metrics)
			handler.Handle()
		}(conn)
	}
}

func (s *Server) Start() error {

	// Start metrics endpoint
	go s.metrics.ServeMetrics(":9090")
	
	// Start celestial distance update routine
	go func() {
		for {
			updateCelestialDistances()
			// Update every hour
			time.Sleep(1 * time.Hour)
		}
	}()

	// Start servers
	s.wg.Add(3) // HTTP, HTTPS, and SOCKS

	go func() {
		defer s.wg.Done()
		if err := s.startHTTPServer(); err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	go func() {
		defer s.wg.Done()
		if err := s.startHTTPSServer(); err != http.ErrServerClosed {
			// Log the error but continue - HTTP server will still work
			// This is important for multi-level subdomains that might not have valid certs
			log.Printf("HTTPS server error: %v", err)
			log.Printf("Multi-level subdomains will still work via HTTP")
		}
	}()

	go func() {
		defer s.wg.Done()
		if err := s.startSOCKSServer(); err != nil {
			log.Printf("SOCKS server error: %v", err)
		}
	}()

	log.Printf("Service started - Note that multi-level subdomains (*.*.latency.space) will be served over HTTP only")
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	// Shutdown HTTP servers gracefully
	if err := s.httpServer.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}
	if err := s.httpsServer.Shutdown(ctx); err != nil {
		log.Printf("HTTPS server shutdown error: %v", err)
	}

	// Close SOCKS listener
	if s.socksListener != nil {
		if err := s.socksListener.Close(); err != nil {
			log.Printf("SOCKS server shutdown error: %v", err)
		}
	}

	// Wait for all servers to finish
	s.wg.Wait()
	return nil
}

func main() {
	// Initialize server
	server := NewServer()

	// Handle shutdown gracefully
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	// Wait for shutdown signal
	<-signals
	log.Println("Shutting down servers...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	log.Println("Server shutdown complete")
}

// getExampleDomains returns a list of example domains for the proxy
func getExampleDomains() []string {
	var domains []string
	domains = append(domains, "latency.space")
	
	// Add all celestial bodies
	for name := range solarSystem {
		domains = append(domains, name+".latency.space")
		
		// Add example formats with domain
		domains = append(domains, "www.example.com."+name+".latency.space")
		
		// Add moons
		for moonName := range solarSystem[name].Moons {
			domains = append(domains, moonName+"."+name+".latency.space")
			domains = append(domains, "www.example.com."+moonName+"."+name+".latency.space")
		}
	}
	
	// Add spacecraft
	for name := range spacecraft {
		domains = append(domains, name+".latency.space")
		domains = append(domains, "www.example.com."+name+".latency.space")
	}
	
	// Add examples of special formats
	domains = append(domains, "www.google.com.earth.latency.space")
	domains = append(domains, "example.com.mars.latency.space")
	domains = append(domains, "api.github.com.jupiter.latency.space")
	
	return domains
}

// isValidProxyDomain checks if a domain is a valid proxy domain format
func isValidProxyDomain(domain string) bool {
	if !strings.HasSuffix(domain, ".latency.space") {
		return false
	}
	
	parts := strings.Split(domain, ".")
	if len(parts) < 3 {
		return false
	}
	
	// Check if it's a celestial body subdomain
	// Format: body.latency.space
	if len(parts) == 3 {
		body, _ := getCelestialBody(parts[0])
		return body != nil
	}
	
	// Check if it's a domain with celestial body
	// Format: domain.body.latency.space
	if len(parts) > 3 {
		// The celestial body is the second-to-last part before "latency.space"
		bodyIndex := len(parts) - 3
		bodyName := parts[bodyIndex]
		body, _ := getCelestialBody(bodyName)
		return body != nil
	}
	
	return false
}

// handleDebug provides debug information
func (s *Server) handleDebug(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	
	// Extract the debug endpoint type from the path
	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		http.NotFound(w, r)
		return
	}
	
	endpointType := parts[2]
	
	switch endpointType {
	case "domains":
		// List valid domains
		domains := getExampleDomains()
		w.Header().Set("Content-Type", "text/plain")
		for _, domain := range domains {
			fmt.Fprintf(w, "%s - Valid: %v\n", domain, isValidProxyDomain(domain))
		}
		
	case "bodies":
		// List celestial bodies and their attributes
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "Celestial Bodies (with real-time distances):\n\n")
		
		// Force an update of celestial distances
		updateCelestialDistances()
		
		// Planets and spacecraft
		for name, body := range solarSystem {
			// Get current distance
			distance := getCurrentDistance(name)
			latency := calculateLatency(distance * 1e6)
			
			fmt.Fprintf(w, "%s:\n", name)
			fmt.Fprintf(w, "  Distance: %.2f million km\n", distance)
			fmt.Fprintf(w, "  Latency: %v\n", latency)
			fmt.Fprintf(w, "  Bandwidth: %d kbps\n", body.BandwidthKbps)
			
			// Moons
			if len(body.Moons) > 0 {
				fmt.Fprintf(w, "  Moons:\n")
				for moonName, moon := range body.Moons {
					moonDistance := getCurrentDistance(moonName + "." + name)
					moonLatency := calculateLatency(moonDistance * 1e6)
					fmt.Fprintf(w, "    %s:\n", moonName)
					fmt.Fprintf(w, "      Distance: %.6f million km\n", moonDistance)
					fmt.Fprintf(w, "      Latency: %v\n", moonLatency)
					fmt.Fprintf(w, "      Bandwidth: %d kbps\n", moon.BandwidthKbps)
				}
			}
			fmt.Fprintf(w, "\n")
		}
		
		// Spacecraft
		fmt.Fprintf(w, "Spacecraft:\n\n")
		for name, craft := range spacecraft {
			distance := getCurrentDistance(name)
			latency := calculateLatency(distance * 1e6)
			fmt.Fprintf(w, "%s:\n", name)
			fmt.Fprintf(w, "  Distance: %.2f million km\n", distance)
			fmt.Fprintf(w, "  Latency: %v\n", latency)
			fmt.Fprintf(w, "  Bandwidth: %d kbps\n", craft.BandwidthKbps)
			fmt.Fprintf(w, "\n")
		}
		
	case "help":
		// Show help info
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "Interplanetary Latency Simulator\n\n")
		fmt.Fprintf(w, "HTTP Proxy Usage:\n")
		fmt.Fprintf(w, "  To use with built-in latency:\n")
		fmt.Fprintf(w, "  - http://mars.latency.space/destination?url=https://example.com\n")
		fmt.Fprintf(w, "  - http://www.example.com.mars.latency.space/ (DNS style routing)\n\n")
		fmt.Fprintf(w, "SOCKS5 Proxy Usage:\n")
		fmt.Fprintf(w, "  - Connect to mars.latency.space:1080 as your SOCKS5 proxy\n")
		fmt.Fprintf(w, "  - Or use www.example.com.mars.latency.space:1080 format to route to example.com\n\n")
		fmt.Fprintf(w, "Debug Endpoints:\n")
		fmt.Fprintf(w, "  - /_debug/domains - List valid domain formats\n")
		fmt.Fprintf(w, "  - /_debug/bodies - List celestial bodies and their properties (with real-time distances)\n")
		fmt.Fprintf(w, "  - /_debug/distances - Show current distances from Earth to all celestial bodies\n")
		fmt.Fprintf(w, "  - /_debug/help - Show this help message\n")
		
	case "distances":
		// Show current celestial distances and update time
		w.Header().Set("Content-Type", "text/plain")
		
		// Get current time
		now := time.Now().UTC()
		
		fmt.Fprintf(w, "Current Celestial Body Distances\n")
		fmt.Fprintf(w, "Current Time: %s UTC\n", now.Format(time.RFC3339))
		fmt.Fprintf(w, "Last Distance Update: %s UTC\n\n", lastDistanceUpdate.Format(time.RFC3339))
		
		// Force an update if more than an hour has passed
		if time.Since(lastDistanceUpdate) > time.Hour {
			updateCelestialDistances()
			fmt.Fprintf(w, "Distances updated now.\n\n")
		}
		
		// Lock to safely read the cache
		distanceCacheMu.RLock()
		defer distanceCacheMu.RUnlock()
		
		// Sort cache keys for consistent display
		var keys []string
		for k := range distanceCache {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		
		// Display all distances
		for _, name := range keys {
			distance := distanceCache[name]
			latency := calculateLatency(distance * 1e6)
			fmt.Fprintf(w, "%-20s: %.3f million km (latency: %v)\n", name, distance, latency)
		}
	
	default:
		http.NotFound(w, r)
	}
}