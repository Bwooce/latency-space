// proxy/src/test_socks.go
//go:build ignore

package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"golang.org/x/net/proxy" // SOCKS5 client library
)

// Simple command-line client for testing the SOCKS5 proxy functionality.
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

	fmt.Printf("Attempting to connect to %s via SOCKS proxy %s\n", destination, proxyAddr)

	// Create a SOCKS5 dialer using the provided proxy address.
	dialer, err := proxy.SOCKS5("tcp", proxyAddr, nil, proxy.Direct) // Using proxy.Direct as fallback is fine here
	if err != nil {
		log.Fatalf("Error creating SOCKS5 dialer: %v", err)
	}

	// Create an HTTP transport that uses the SOCKS5 dialer.
	httpTransport := &http.Transport{
		DialContext: dialer.(proxy.ContextDialer).DialContext, // Use DialContext for better cancellation support
	}

	// Create an HTTP client configured to use the SOCKS transport.
	// Set a generous timeout considering potential high latency.
	client := &http.Client{
		Transport: httpTransport,
		Timeout:   10 * time.Minute, // Example: 10 minutes timeout, adjust as needed
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
	fmt.Println("Headers:")
	for key, values := range resp.Header {
		for _, value := range values {
			fmt.Printf("  %s: %s\n", key, value)
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

// testDNSFormat specifically tests connecting via the SOCKS proxy using the special DNS format.
// This function is not called by main but kept for potential separate testing.
func testDNSFormat() {
	// Test connection to www.example.com via mars.latency.space:1080.
	proxyAddr := "mars.latency.space:1080"
	targetAddr := "www.example.com:80" // Target service address

	fmt.Printf("Testing SOCKS DNS format: connecting to %s via proxy %s\n", targetAddr, proxyAddr)

	// Create the SOCKS5 dialer.
	dialer, err := proxy.SOCKS5("tcp", proxyAddr, nil, proxy.Direct)
	if err != nil {
		log.Fatalf("Error creating SOCKS5 dialer for DNS test: %v", err)
	}

	// Dial the target address (www.example.com:80) through the proxy.
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
	// Here you could potentially send/receive data over the raw TCP connection (conn)
	// For example, send an HTTP GET request manually.
}
