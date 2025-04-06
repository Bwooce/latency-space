// proxy/src/test_socks.go
// +build ignore

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

	"golang.org/x/net/proxy"
)

// Simple test client for the SOCKS proxy
func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run test_socks.go <proxy-host:port> <destination-url>")
		fmt.Println("Example: go run test_socks.go mars.latency.space:1080 https://example.com")
		os.Exit(1)
	}

	proxyAddr := os.Args[1]
	destination := os.Args[2]

	fmt.Printf("Testing SOCKS proxy at %s to %s\n", proxyAddr, destination)

	// Create a SOCKS5 dialer
	dialer, err := proxy.SOCKS5("tcp", proxyAddr, nil, proxy.Direct)
	if err != nil {
		log.Fatalf("Failed to create SOCKS5 dialer: %v", err)
	}

	// Create HTTP transport with the SOCKS5 dialer
	httpTransport := &http.Transport{
		Dial: dialer.Dial,
	}

	// Create HTTP client
	client := &http.Client{
		Transport: httpTransport,
		Timeout:   30 * time.Second,
	}

	// Parse the destination URL
	destURL, err := url.Parse(destination)
	if err != nil {
		log.Fatalf("Invalid destination URL: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("GET", destURL.String(), nil)
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}

	// Time the request
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}
	elapsed := time.Since(start)

	// Print response details
	fmt.Printf("Response Status: %s\n", resp.Status)
	fmt.Printf("Time taken: %v\n", elapsed)
	fmt.Println("Headers:")
	for key, value := range resp.Header {
		fmt.Printf("  %s: %s\n", key, value)
	}

	// Print a snippet of the response body
	fmt.Println("\nResponse body (first 500 bytes):")
	body := make([]byte, 500)
	n, _ := io.ReadAtLeast(resp.Body, body, 1)
	fmt.Println(string(body[:n]))

	resp.Body.Close()
}

// Test DNS format
func testDNSFormat() {
	// Example domain format: www.example.com.mars.latency.space
	// Connect to the proxy as mars.latency.space:1080
	// Request www.example.com
	proxyAddr := "mars.latency.space:1080"
	
	// Create a SOCKS5 dialer
	dialer, err := proxy.SOCKS5("tcp", proxyAddr, nil, proxy.Direct)
	if err != nil {
		log.Fatalf("Failed to create SOCKS5 dialer: %v", err)
	}

	// Connect to destination via the SOCKS5 proxy
	start := time.Now()
	conn, err := dialer.Dial("tcp", "www.example.com:80")
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	elapsed := time.Since(start)
	
	fmt.Printf("Connected to www.example.com via %s in %v\n", proxyAddr, elapsed)
	conn.Close()
}