// proxy/src/test_socks.go
//go:build ignore

package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/net/proxy" // SOCKS5 client library
)

// Simple command-line client for testing the SOCKS5 proxy functionality with HTTPS support.
// NOTE: This file has a build tag `//go:build ignore` at the top,
// so it won't be built with the main proxy application.
// Run it directly using `go run proxy/src/test_socks.go <args>`.
func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run proxy/src/test_socks.go <proxy-host:port> <destination-url>")
		fmt.Println("Example: go run test_socks.go mars.latency.space:1080 https://example.com")
		os.Exit(1)
	}

	proxyAddr := os.Args[1]   // e.g., "mars.latency.space:1080"
	destination := os.Args[2] // e.g., "https://example.com"

	// Check if protocol is specified, default to https:// if not
	if !strings.HasPrefix(destination, "http://") && !strings.HasPrefix(destination, "https://") {
		destination = "https://" + destination
		fmt.Printf("No protocol specified, using HTTPS: %s\n", destination)
	}

	fmt.Printf("Attempting to connect to %s via SOCKS proxy %s\n", destination, proxyAddr)

	// Create a SOCKS5 dialer using the provided proxy address.
	dialer, err := proxy.SOCKS5("tcp", proxyAddr, nil, proxy.Direct) // Using proxy.Direct as fallback is fine here
	if err != nil {
		log.Fatalf("Error creating SOCKS5 dialer: %v", err)
	}

	// Configure TLS with InsecureSkipVerify for testing purposes
	// In production, you should use proper certificate validation
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, // Skip certificate verification for testing
	}

	// Create an HTTP transport that uses the SOCKS5 dialer.
	httpTransport := &http.Transport{
		DialContext:     dialer.(proxy.ContextDialer).DialContext, // Use DialContext for better cancellation support
		TLSClientConfig: tlsConfig,                                // Add TLS config for HTTPS support
	}

	// Create an HTTP client configured to use the SOCKS transport.
	// Set a generous timeout considering potential high latency.
	client := &http.Client{
		Transport: httpTransport,
		Timeout:   10 * time.Minute, // Example: 10 minutes timeout, adjust as needed
		// Don't follow redirects automatically to better analyze the response
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Allow up to 10 redirects
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			fmt.Printf("Redirect to: %s\n", req.URL.String())
			return nil
		},
	}

	// Parse the target URL.
	destURL, err := url.Parse(destination)
	if err != nil {
		log.Fatalf("Invalid destination URL '%s': %v", destination, err)
	}

	// Create a GET request for the target URL.
	req, err := http.NewRequest("GET", destURL.String(), nil)
	if err != nil {
		log.Fatalf("Error creating HTTP request: %v", err)
	}

	// Execute the request and measure the time taken.
	fmt.Println("Sending request...")
	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start)
	if err != nil {
		log.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close() // Ensure response body is closed

	// Print the response status and headers.
	fmt.Printf("\n--- Response ---")
	fmt.Printf("\nStatus: %s\n", resp.Status)
	fmt.Printf("Time taken: %v\n", elapsed)
	fmt.Printf("Protocol: %s\n", resp.Proto)
	fmt.Println("Headers:")
	for key, values := range resp.Header {
		for _, value := range values {
			fmt.Printf("  %s: %s\n", key, value)
		}
	}

	// Print connection information if available
	if resp.TLS != nil {
		fmt.Println("\nTLS Information:")
		fmt.Printf("  Version: %s\n", tlsVersionString(resp.TLS.Version))
		fmt.Printf("  Cipher Suite: %s\n", tlsCipherSuiteString(resp.TLS.CipherSuite))
		fmt.Printf("  Server Name: %s\n", resp.TLS.ServerName)
		if len(resp.TLS.PeerCertificates) > 0 {
			cert := resp.TLS.PeerCertificates[0]
			fmt.Printf("  Certificate Subject: %s\n", cert.Subject)
			fmt.Printf("  Certificate Issuer: %s\n", cert.Issuer)
			fmt.Printf("  Certificate Valid Until: %s\n", cert.NotAfter)
		}
	}

	// Print the beginning of the response body.
	fmt.Println("\nBody (first 500 bytes):")
	body := make([]byte, 500)
	n, readErr := io.ReadFull(resp.Body, body) // Read up to 500 bytes
	if readErr != nil && readErr != io.ErrUnexpectedEOF && readErr != io.EOF {
		log.Printf("Warning: Error reading response body: %v", readErr)
	}
	if n > 0 {
		fmt.Println(string(body[:n]))
	} else {
		fmt.Println("[No body content read]")
	}
	fmt.Println("\n----------------")

	fmt.Println("\nTest finished successfully.")
}

// Helper function to convert TLS version to string
func tlsVersionString(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return fmt.Sprintf("Unknown (%d)", version)
	}
}

// Helper function to convert cipher suite to string
func tlsCipherSuiteString(cipherSuite uint16) string {
	switch cipherSuite {
	case tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256:
		return "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"
	case tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384:
		return "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"
	case tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256:
		return "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256"
	case tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384:
		return "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384"
	case tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256:
		return "TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256"
	case tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256:
		return "TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256"
	default:
		return fmt.Sprintf("Unknown (%d)", cipherSuite)
	}
}

// testHTTPSFormat tests connecting to an HTTPS endpoint via the SOCKS proxy.
// This function is not called by main but kept for potential separate testing.
func testHTTPSFormat() {
	// Test connection to https://www.example.com via mars.latency.space:1080.
	proxyAddr := "mars.latency.space:1080"
	targetURL := "https://www.example.com" // HTTPS target

	fmt.Printf("Testing HTTPS connection: connecting to %s via proxy %s\n", targetURL, proxyAddr)

	// Create the SOCKS5 dialer.
	dialer, err := proxy.SOCKS5("tcp", proxyAddr, nil, proxy.Direct)
	if err != nil {
		log.Fatalf("Error creating SOCKS5 dialer for HTTPS test: %v", err)
	}

	// Configure TLS with InsecureSkipVerify for testing
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	// Create an HTTP transport that uses the SOCKS5 dialer.
	httpTransport := &http.Transport{
		DialContext:     dialer.(proxy.ContextDialer).DialContext,
		TLSClientConfig: tlsConfig,
	}

	// Create an HTTP client
	client := &http.Client{
		Transport: httpTransport,
		Timeout:   1 * time.Minute,
	}

	// Send a request
	fmt.Println("Sending HTTPS request...")
	start := time.Now()
	resp, err := client.Get(targetURL)
	elapsed := time.Since(start)
	if err != nil {
		log.Fatalf("HTTPS request failed: %v", err)
	}
	defer resp.Body.Close()

	fmt.Printf("Successfully connected to %s via %s in %v\n", targetURL, proxyAddr, elapsed)
	fmt.Printf("Response status: %s\n", resp.Status)

	// Additional TLS information
	if resp.TLS != nil {
		fmt.Println("TLS Connection Established:")
		fmt.Printf("  Protocol Version: %s\n", tlsVersionString(resp.TLS.Version))
		fmt.Printf("  Cipher Suite: %s\n", tlsCipherSuiteString(resp.TLS.CipherSuite))
	}
}

// testDNSFormat specifically tests connecting via the SOCKS proxy using the special DNS format.
// This function is not called by main but kept for potential separate testing.
func testDNSFormat() {
	// Test connection to www.example.com via mars.latency.space:1080.
	proxyAddr := "mars.latency.space:1080"
	targetAddr := "www.example.com:443" // HTTPS port for SSL/TLS

	fmt.Printf("Testing SOCKS DNS format: connecting to %s via proxy %s\n", targetAddr, proxyAddr)

	// Create the SOCKS5 dialer.
	dialer, err := proxy.SOCKS5("tcp", proxyAddr, nil, proxy.Direct)
	if err != nil {
		log.Fatalf("Error creating SOCKS5 dialer for DNS test: %v", err)
	}

	// Dial the target address (www.example.com:443) through the proxy.
	fmt.Println("Dialing...")
	start := time.Now()
	// Use the dialer directly to establish a TCP connection
	conn, err := dialer.Dial("tcp", targetAddr)
	elapsed := time.Since(start)
	if err != nil {
		log.Fatalf("Failed to connect to %s via proxy %s: %v", targetAddr, proxyAddr, err)
	}
	defer conn.Close() // Ensure connection is closed

	fmt.Printf("Successfully connected to %s via %s in %v\n", targetAddr, proxyAddr, elapsed)

	// Since this is a raw TCP connection to port 443, we could upgrade to TLS here:
	fmt.Println("Upgrading connection to TLS...")
	tlsConn := tls.Client(conn, &tls.Config{
		InsecureSkipVerify: true,              // Skip certificate verification for testing
		ServerName:         "www.example.com", // SNI
	})

	// Handshake to establish the TLS connection
	if err := tlsConn.Handshake(); err != nil {
		log.Fatalf("TLS handshake failed: %v", err)
	}

	fmt.Println("TLS connection established successfully!")
	// Now tlsConn is a secure connection that can be used for HTTPS requests
}
