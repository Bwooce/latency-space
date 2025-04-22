// proxy/src/security.go
package main

import (
	"fmt"
	"net/url"
	"strings"
)

type SecurityValidator struct {
	allowedPorts   map[string]bool
	maxRequestSize int64
	allowedSchemes map[string]bool
	allowedHosts   map[string]bool // List of allowed destination hosts
}

func NewSecurityValidator() *SecurityValidator {
	// Define allowed hosts - major websites that can handle traffic
	allowedHostsList := []string{
		"google.com", "www.google.com",
		"example.com", "www.example.com",
		"microsoft.com", "www.microsoft.com",
		"github.com", "www.github.com",
		"cloudflare.com", "www.cloudflare.com",
		"apple.com", "www.apple.com",
		"amazon.com", "www.amazon.com",
		"wikipedia.org", "www.wikipedia.org",
		"facebook.com", "www.facebook.com",
		"youtube.com", "www.youtube.com",
		"reddit.com", "www.reddit.com",
		"twitter.com", "www.twitter.com",
		"instagram.com", "www.instagram.com",
		"netflix.com", "www.netflix.com",
		"linkedin.com", "www.linkedin.com",
	}

	allowedHosts := make(map[string]bool)
	for _, host := range allowedHostsList {
		allowedHosts[host] = true
	}

	return &SecurityValidator{
		allowedPorts: map[string]bool{
			"80":   true,
			"443":  true,
			"8080": true,
			"53":   true,
			"":     true, // Default ports (80 for HTTP, 443 for HTTPS)
		},
		maxRequestSize: 100 * 1024 * 1024, // 100MB
		allowedSchemes: map[string]bool{
			"http":  true,
			"https": true,
			"ws":    true,
			"wss":   true,
		},
		allowedHosts: allowedHosts,
	}
}

func (s *SecurityValidator) ValidateDestination(dest string) (string, error) {
	if dest == "" {
		return "", fmt.Errorf("destination is required")
	}

	if !strings.Contains(dest, "://") {
		dest = "http://" + dest
	}

	u, err := url.Parse(dest)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %v", err)
	}

	if !s.allowedSchemes[u.Scheme] {
		return "", fmt.Errorf("scheme %s not allowed", u.Scheme)
	}

	if u.Port() != "" && !s.allowedPorts[u.Port()] {
		return "", fmt.Errorf("port %s not allowed", u.Port())
	}

	return dest, nil
}

// IsAllowedHost checks if the destination host is in the allowed list
func (s *SecurityValidator) IsAllowedHost(dest string) bool {
	// Parse the URL
	u, err := url.Parse(dest)
	if err != nil {
		return false
	}

	// Extract the host without port
	host := u.Hostname()

	// Check if it's in our allowed list
	if s.allowedHosts[host] {
		return true
	}

	// Check for subdomains of allowed hosts
	for allowedHost := range s.allowedHosts {
		if strings.HasSuffix(host, "."+allowedHost) {
			return true
		}
	}

	return false
}

// ValidateSocksDestination validates a SOCKS destination address and port
func (s *SecurityValidator) ValidateSocksDestination(host string, port uint16) error {
	portStr := fmt.Sprintf("%d", port)
	if port != 0 && !s.allowedPorts[portStr] {
		return fmt.Errorf("port %s not allowed", portStr)
	}

	return nil
}

// IsAllowedIP checks if an IP address is allowed to use the proxy
// Currently allows all IPs, but could be used for rate limiting or blocklists
func (s *SecurityValidator) IsAllowedIP(ip string) bool {
	// For now, allow all IPs to use the proxy
	// This can be enhanced in the future to implement rate limiting or IP blocklists
	return true
}
