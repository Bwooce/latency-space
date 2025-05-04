// proxy/src/security.go
package main

import (
	"fmt"
	"net/url"
	"strconv" // Required for port conversion in ValidateSocksDestination
	"strings"
)

// SecurityValidator provides methods for validating proxy requests.
type SecurityValidator struct {
	allowedPorts   map[string]bool   // Allowed destination ports (e.g., "80", "443")
	maxRequestSize int64             // Maximum allowed request size (currently unused)
	allowedSchemes map[string]bool   // Allowed URL schemes (e.g., "http", "https")
	allowedHosts   map[string]bool   // Map of explicitly allowed destination hosts/domains
}

// NewSecurityValidator creates a new SecurityValidator with default rules.
func NewSecurityValidator() *SecurityValidator {
	// Define allowed destination hosts - primarily major websites assumed to handle traffic.
	// This helps prevent using the proxy for malicious purposes against arbitrary hosts.
	allowedHostsList := []string{
		// Major Search/Services
		"google.com", "www.google.com", "google.co.uk", "google.de", "google.fr", "google.com.au",
		"bing.com", "www.bing.com",
		"duckduckgo.com",
		// Common Tech/Dev Sites
		"example.com", "www.example.com", // Standard example domain
		"github.com", "api.github.com", "raw.githubusercontent.com",
		"stackoverflow.com",
		"microsoft.com", "www.microsoft.com",
		"apple.com", "www.apple.com",
		// Cloud/Infra
		"cloudflare.com", "www.cloudflare.com", "1.1.1.1",
		"amazon.com", "www.amazon.com", // AWS services often use subdomains
		"aws.amazon.com",
		// Content/Info
		"wikipedia.org", "www.wikipedia.org", "en.wikipedia.org",
		// Social Media
		"facebook.com", "www.facebook.com",
		"youtube.com", "www.youtube.com",
		"reddit.com", "www.reddit.com", "old.reddit.com",
		"twitter.com", "x.com", // Common Twitter domains
		"instagram.com", "www.instagram.com",
		"linkedin.com", "www.linkedin.com",
		// Entertainment/News (Examples)
		"netflix.com", "www.netflix.com",
		"bbc.co.uk", "www.bbc.co.uk", "bbc.com", "www.bbc.com",
		"nytimes.com", "www.nytimes.com",
		// Add latency.space itself for internal access
		"latency.space", /* status now integrated with main site */
		"microsoft.com", "www.microsoft.com",
		"github.com", "www.github.com",
		"cloudflare.com", "www.cloudflare.com",
		"apple.com", "www.apple.com",
		"amazon.com", "www.amazon.com",
		"wikipedia.org", "www.wikipedia.org",
	}

	// Populate the map for efficient lookup
	allowedHostsMap := make(map[string]bool)
	for _, host := range allowedHostsList {
		allowedHostsMap[strings.ToLower(host)] = true // Store lowercase for case-insensitive checks
	}

	return &SecurityValidator{
		allowedPorts: map[string]bool{
			"80":   true, // HTTP
			"443":  true, // HTTPS
			"8080": true, // Common alt HTTP
			"53":   true, // DNS (for potential UDP associate tests)
			"":     true, // Allow default ports (implicit 80/443)
			// Add other ports if needed, e.g., for SSH (22), FTP (21), etc.
		},
		maxRequestSize: 100 * 1024 * 1024, // 100MB (Currently informational)
		allowedSchemes: map[string]bool{
			"http":  true,
			"https": true,
			// "ws":    true, // WebSocket (enable if needed)
			// "wss":   true, // Secure WebSocket (enable if needed)
		},
		allowedHosts: allowedHostsMap,
	}
}

// ValidateDestination checks if the HTTP destination URL format, scheme, and port are allowed.
// It attempts to add a default scheme if missing.
func (s *SecurityValidator) ValidateDestination(dest string) (string, error) {
	if dest == "" {
		return "", fmt.Errorf("destination URL cannot be empty")
	}

	// Attempt to add a default scheme if none is present
	if !strings.Contains(dest, "://") {
		dest = "http://" + dest // Default to HTTP
	}

	// Parse the URL
	u, err := url.Parse(dest)
	if err != nil {
		return "", fmt.Errorf("invalid destination URL format: %v", err)
	}

	// Validate scheme
	if !s.allowedSchemes[strings.ToLower(u.Scheme)] {
		return "", fmt.Errorf("URL scheme '%s' is not allowed", u.Scheme)
	}

	// Validate port (if specified)
	port := u.Port()
	if port != "" && !s.allowedPorts[port] {
		return "", fmt.Errorf("destination port '%s' is not allowed", port)
	}

	// Validate host
	if !s.IsAllowedHost(u.Hostname()) {
		return "", fmt.Errorf("destination host '%s' is not allowed", u.Hostname())
	}


	return u.String(), nil // Return the potentially modified URL (with added scheme)
}

// IsAllowedHost checks if the destination host (or its parent domain) is in the allowed list.
// Performs case-insensitive matching.
func (s *SecurityValidator) IsAllowedHost(host string) bool {
	if host == "" {
		return false // Cannot allow empty host
	}
	lowerHost := strings.ToLower(host)

	// Direct match in allowed list
	if s.allowedHosts[lowerHost] {
		return true
	}

	// Check if it's a subdomain of an allowed host
	for allowed := range s.allowedHosts {
		// Ensure allowed host isn't empty and check suffix
		if allowed != "" && strings.HasSuffix(lowerHost, "."+allowed) {
			return true
		}
	}

	// Log rejection if desired
	// log.Printf("Rejected host: %s (not in allowed list or subdomain)", host)
	return false
}

// ValidateSocksDestination checks if the SOCKS destination port is allowed.
// Host validation is done separately using IsAllowedHost.
func (s *SecurityValidator) ValidateSocksDestination(host string, port uint16) error {
	// Validate host first
	if !s.IsAllowedHost(host) {
		return fmt.Errorf("destination host '%s' is not allowed", host)
	}

	// Validate port
	portStr := strconv.FormatUint(uint64(port), 10)
	// Allow port 0 (often used in BIND requests)
	if port != 0 && !s.allowedPorts[portStr] {
		return fmt.Errorf("destination port %s is not allowed", portStr)
	}


	return nil
}

// IsAllowedIP checks if a client IP address is allowed to use the proxy.
// Currently allows all IPs; can be extended for rate limiting or blocklists.
func (s *SecurityValidator) IsAllowedIP(ip string) bool {
	// TODO: Implement rate limiting or IP blocklist checks here if needed.
	return true
}
