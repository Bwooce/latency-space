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
}

func NewSecurityValidator() *SecurityValidator {
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

// ValidateSocksDestination validates a SOCKS destination address and port
func (s *SecurityValidator) ValidateSocksDestination(host string, port uint16) error {
	portStr := fmt.Sprintf("%d", port)
	if port != 0 && !s.allowedPorts[portStr] {
		return fmt.Errorf("port %s not allowed", portStr)
	}

	return nil
}