// proxy/src/main.go
package main

import (
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
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
	wg            sync.WaitGroup
}

func NewServer() *Server {
	return &Server{
		security: NewSecurityValidator(),
		metrics:  NewMetricsCollector(),
	}
}

func (s *Server) handleHTTP(w http.ResponseWriter, r *http.Request) {

	if r.URL.Path == "/_debug/domains" {
		s.handleDebug(w, r)
		return
	}

	// Extract target domain from the hostname
	targetDomain, celestialBody, bodyName := s.extractDomainAndBody(r.Host)
	if celestialBody == nil {
		http.Error(w, "Unknown celestial body", http.StatusBadRequest)
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
	latency := calculateLatency(celestialBody.Distance * 1e6)
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

// Add this to your HTTP handler code
func (s *Server) handleDebug(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/_debug/domains" {
		http.NotFound(w, r)
		return
	}

	domains := listValidDomains()
	w.Header().Set("Content-Type", "text/plain")
	for _, domain := range domains {
		fmt.Fprintf(w, "%s - Valid: %v\n", domain, isValidSubdomain(domain))
	}
}