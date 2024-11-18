// proxy/src/main.go
package main

import (
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
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
	httpServer  *http.Server
	httpsServer *http.Server
	security    *SecurityValidator
	metrics     *MetricsCollector
	wg          sync.WaitGroup
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

	// Extract celestial body from subdomain
	hostParts := strings.Split(r.Host, ".")
	if len(hostParts) == 0 {
		http.Error(w, "Invalid host", http.StatusBadRequest)
		return
	}

	body, bodyName := getCelestialBody(hostParts[0])
	if body == nil {
		http.Error(w, "Unknown celestial body", http.StatusBadRequest)
		return
	}

	// Get destination from header or query param
	destination := r.Header.Get("X-Destination")
	if destination == "" {
		destination = r.URL.Query().Get("destination")
	}

	// Validate destination
	validDest, err := s.security.ValidateDestination(destination)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid destination: %v", err), http.StatusBadRequest)
		return
	}

	// Check for WebSocket upgrade
	if websocket.IsWebSocketUpgrade(r) {
		s.handleWebSocket(w, r, body, validDest)
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
	latency := calculateLatency(body.Distance * 1e6)
	time.Sleep(latency)

	// Apply bandwidth limiting
	s.metrics.TrackBandwidth(bodyName, r.ContentLength)

	// Forward the request
	proxy.ServeHTTP(w, r)
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

func (s *Server) Start() error {

	// Start metrics endpoint
	go s.metrics.ServeMetrics(":9090")

	// Start servers
	s.wg.Add(2)
	go func() {
		defer s.wg.Done()
		if err := s.startHTTPServer(); err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	go func() {
		defer s.wg.Done()
		if err := s.startHTTPSServer(); err != http.ErrServerClosed {
			log.Printf("HTTPS server error: %v", err)
		}
	}()

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
