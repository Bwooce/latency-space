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
	httpServer  *http.Server
	httpsServer *http.Server
	udpProxy    *UDPProxy
	security    *SecurityValidator
	metrics     *MetricsCollector
	wg          sync.WaitGroup
}

func NewServer() *Server {
	return &Server{
		security: NewSecurityValidator(),
		udpProxy: NewUDPProxy(),
		metrics:  NewMetricsCollector(),
	}
}

func (s *Server) handleHTTP(w http.ResponseWriter, r *http.Request) {
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
	return s.httpsServer.ListenAndServeTLS("", "") // Cert handled by autocert
}

func (s *Server) startUDPServer() error {
	addr, err := net.ResolveUDPAddr("udp", ":53")
	if err != nil {
		return fmt.Errorf("failed to resolve UDP address: %v", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to start UDP server: %v", err)
	}
	defer conn.Close()

	log.Printf("Starting UDP server on :53")

	buffer := make([]byte, 65535)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("Error reading UDP: %v", err)
			continue
		}

		body, bodyName := getCelestialBody(strings.Split(remoteAddr.String(), ".")[0])
		if body == nil {
			continue
		}

		s.metrics.RecordUDPPacket(bodyName, int64(n))
		go s.udpProxy.handlePacket(conn, buffer[:n], remoteAddr, body)
	}
}

func (s *Server) Start() error {
	// Start DNS server
	dnsServer := NewDNSServer()
	go func() {
		if err := dnsServer.Start(); err != nil {
			log.Printf("DNS server error: %v", err)
		}
	}()

	// Start UDP server
	udpServer := NewUDPServer()
	go func() {
		if err := udpServer.Start(); err != nil {
			log.Printf("UDP server error: %v", err)
		}
	}()

	// Start metrics endpoint
	go s.metrics.ServeMetrics(":9090")

	// Start servers
	s.wg.Add(3)
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

	go func() {
		defer s.wg.Done()
		if err := s.startUDPServer(); err != nil {
			log.Printf("UDP server error: %v", err)
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
