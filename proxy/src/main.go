// proxy/src/main.go
package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
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
	
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

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
		s.httpServer.Shutdown(ctx)
	}

	if s.httpsServer != nil {
		log.Println("Shutting down HTTPS server...")
		s.httpsServer.Shutdown(ctx)
	}

	if s.socksListener != nil {
		log.Println("Shutting down SOCKS5 server...")
		s.socksListener.Close()
	}
}

// handleHTTP processes HTTP requests with celestial body latency
func (s *Server) handleHTTP(w http.ResponseWriter, r *http.Request) {
	// Special case for metrics endpoint
	if r.URL.Path == "/metrics" {
		promhttp.Handler().ServeHTTP(w, r)
		return
	}
	
	// Special case for debug endpoints
	if strings.HasPrefix(r.URL.Path, "/_debug/") {
		s.handleDebugEndpoint(w, r)
		return
	}
	
	// Handle CORS preflight for debug endpoints
	if r.Method == "OPTIONS" && strings.HasPrefix(r.URL.Path, "/_debug/") {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Process the host to determine if this is a celestial body request
	targetURL, celestialBody, bodyName := s.parseHostForCelestialBody(r.Host, r.URL)
	
	// Check if celestial body exists
	if celestialBody == nil {
		http.Error(w, "Unknown celestial body", http.StatusBadRequest)
		return
	}
	
	// Apply space latency
	latency := calculateLatency(celestialBody.Distance * 1e6)
	log.Printf("Proxy request for %s via %s (latency: %v)", targetURL, bodyName, latency)
	time.Sleep(latency)
	
	// Start metrics collection
	start := time.Now()
	defer func() {
		s.metrics.RecordRequest(bodyName, "http", time.Since(start))
	}()
	
	// If there's no target URL, just display info about this celestial body
	if targetURL == "" {
		s.displayCelestialInfo(w, celestialBody, bodyName, latency)
		return
	}
	
	// Apply bandwidth limiting
	r.Header.Set("X-Celestial-Body", bodyName)
	
	// Forward the request to the target URL
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:    10,
			IdleConnTimeout: 30 * time.Second,
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
		// Skip host header
		if strings.ToLower(name) != "host" {
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

	// Copy body
	io.Copy(w, resp.Body)
}

// displayCelestialInfo shows information about the celestial body
func (s *Server) displayCelestialInfo(w http.ResponseWriter, body *CelestialBody, name string, latency time.Duration) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	
	fmt.Fprintf(w, "<html><head><title>%s - Latency Space</title></head><body>", name)
	fmt.Fprintf(w, "<h1>%s</h1>", name)
	fmt.Fprintf(w, "<p>You are accessing the Solar System through %s.</p>", name)
	fmt.Fprintf(w, "<p>Current distance from Earth: %.2f million km</p>", body.Distance)
	fmt.Fprintf(w, "<p>One-way latency: %v</p>", latency)
	fmt.Fprintf(w, "<p>Round-trip latency: %v</p>", 2*latency)
	fmt.Fprintf(w, "<p>Bandwidth limit: %d Kbps</p>", body.BandwidthKbps)
	fmt.Fprintf(w, "<p>Rate limit: %d requests per minute</p>", body.RateLimit)
	
	fmt.Fprintf(w, "<h2>Usage</h2>")
	fmt.Fprintf(w, "<p>To browse a website through %s, use one of these formats:</p>", name)
	fmt.Fprintf(w, "<ul>")
	fmt.Fprintf(w, "<li><code>http://%s.latency.space/http://example.com</code></li>", name)
	fmt.Fprintf(w, "<li><code>http://example.com.%s.latency.space/</code></li>", name)
	fmt.Fprintf(w, "<li><code>http://%s.latency.space/?url=http://example.com</code></li>", name)
	fmt.Fprintf(w, "</ul>")
	
	fmt.Fprintf(w, "<h2>SOCKS5 Proxy</h2>")
	fmt.Fprintf(w, "<p>For SOCKS5 proxy access through %s:</p>", name)
	fmt.Fprintf(w, "<pre>Host: %s.latency.space\nPort: 1080\nType: SOCKS5</pre>", name)
	
	if len(body.Moons) > 0 {
		fmt.Fprintf(w, "<h2>Moons</h2>")
		fmt.Fprintf(w, "<p>%s has the following moons available:</p>", name)
		fmt.Fprintf(w, "<ul>")
		for moon := range body.Moons {
			fmt.Fprintf(w, "<li><a href=\"http://%s.%s.latency.space/\">%s</a></li>", moon, name, moon)
		}
		fmt.Fprintf(w, "</ul>")
	}
	
	fmt.Fprintf(w, "</body></html>")
}

// parseHostForCelestialBody extracts target URL and celestial body from request
func (s *Server) parseHostForCelestialBody(host string, reqURL *url.URL) (string, *CelestialBody, string) {
	// Remove port from host if present
	if idx := strings.Index(host, ":"); idx > 0 {
		host = host[:idx]
	}
	
	// Check for debug endpoints which don't need celestial body processing
	if strings.HasPrefix(reqURL.Path, "/_debug/") {
		return "", solarSystem["earth"], "earth"
	}
	
	// Not a latency.space domain
	if !strings.HasSuffix(host, ".latency.space") {
		// Default to Earth
		return reqURL.String(), solarSystem["earth"], "earth"
	}
	
	// Extract parts: [subdomain, latency, space]
	parts := strings.Split(host, ".")
	if len(parts) < 3 || parts[len(parts)-1] != "space" || parts[len(parts)-2] != "latency" {
		// Not a proper latency.space domain
		return reqURL.String(), solarSystem["earth"], "earth"
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
		ReadTimeout:  60 * time.Minute,  // Increased for distant celestial bodies
		WriteTimeout: 60 * time.Minute,  // Increased for distant celestial bodies
		IdleTimeout:  120 * time.Minute, // Allow long-lived connections
	}

	log.Printf("Starting HTTP server on :80")
	return s.httpServer.ListenAndServe()
}

func (s *Server) startHTTPSServer() error {
	s.httpsServer = &http.Server{
		Addr:         ":443",
		Handler:      http.HandlerFunc(s.handleHTTP),
		TLSConfig:    setupTLS(),
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
	case "distances":
		s.printCelestialDistances(w)
	case "bodies":
		s.printCelestialBodies(w)
	case "help":
		s.printHelp(w)
	default:
		http.Error(w, "Unknown debug command", http.StatusNotFound)
	}
}

// printCelestialDistances shows the current distances of all celestial bodies
func (s *Server) printCelestialDistances(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain")
	
	fmt.Fprintln(w, "Latency Space - Current Celestial Distances")
	fmt.Fprintln(w, "============================================")
	fmt.Fprintf(w, "Current Time: %s\n\n", time.Now().Format(time.RFC3339))
	
	// Print planets
	fmt.Fprintln(w, "PLANETS:")
	for name, body := range solarSystem {
		latency := calculateLatency(body.Distance * 1e6)
		fmt.Fprintf(w, "%s: %.2f million km (one-way latency: %v)\n", 
			name, body.Distance, latency)
			
		// Print moons
		for moonName, moon := range body.Moons {
			moonLatency := calculateLatency(moon.Distance * 1e6)
			fmt.Fprintf(w, "  %s.%s: %.6f million km (one-way latency: %v)\n", 
				moonName, name, moon.Distance, moonLatency)
		}
	}
	
	// Print spacecraft
	fmt.Fprintln(w, "\nSPACECRAFT:")
	for name, body := range spacecraft {
		latency := calculateLatency(body.Distance * 1e6)
		fmt.Fprintf(w, "%s: %.2f million km (one-way latency: %v)\n", 
			name, body.Distance, latency)
	}
}

// printCelestialBodies displays all celestial body info
func (s *Server) printCelestialBodies(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain")
	
	fmt.Fprintln(w, "Latency Space - Celestial Body Configuration")
	fmt.Fprintln(w, "===========================================")
	
	// Print planets
	fmt.Fprintln(w, "PLANETS:")
	for name, body := range solarSystem {
		fmt.Fprintf(w, "%s:\n", name)
		fmt.Fprintf(w, "  Distance: %.2f million km\n", body.Distance)
		fmt.Fprintf(w, "  Bandwidth: %d Kbps\n", body.BandwidthKbps)
		fmt.Fprintf(w, "  Rate Limit: %d requests/minute\n", body.RateLimit)
		fmt.Fprintf(w, "  Moons: %d\n", len(body.Moons))
		
		// Print moons
		for moonName, moon := range body.Moons {
			fmt.Fprintf(w, "  - %s:\n", moonName)
			fmt.Fprintf(w, "    Distance: %.6f million km\n", moon.Distance)
			fmt.Fprintf(w, "    Bandwidth: %d Kbps\n", moon.BandwidthKbps)
			fmt.Fprintf(w, "    Rate Limit: %d requests/minute\n", moon.RateLimit)
		}
		fmt.Fprintln(w)
	}
	
	// Print spacecraft
	fmt.Fprintln(w, "SPACECRAFT:")
	for name, body := range spacecraft {
		fmt.Fprintf(w, "%s:\n", name)
		fmt.Fprintf(w, "  Distance: %.2f million km\n", body.Distance)
		fmt.Fprintf(w, "  Bandwidth: %d Kbps\n", body.BandwidthKbps)
		fmt.Fprintf(w, "  Rate Limit: %d requests/minute\n", body.RateLimit)
		fmt.Fprintln(w)
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
	fmt.Fprintln(w, "/_debug/bodies - Detailed celestial body configurations")
	fmt.Fprintln(w, "/_debug/help - This help information")
}

func main() {
	// Parse command-line arguments
	port := flag.Int("port", 80, "HTTP port to listen on")
	https := flag.Bool("https", true, "Enable HTTPS")
	
	flag.Parse()
	
	// Create and start the server
	server := NewServer(*port, *https)
	err := server.Start()
	if err != nil {
		log.Fatalf("Server error: %v", err)
	}
}