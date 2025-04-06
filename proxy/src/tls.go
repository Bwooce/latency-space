// proxy/src/tls.go
package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/crypto/acme/autocert"
	"log"
	"os"
	"os/exec"
	"strings"
)

func isValidSubdomain(host string) bool {
	// Check if it's the base domain
	if host == "latency.space" {
		return true
	}

	// Split the host into parts
	parts := strings.Split(host, ".")
	if len(parts) < 2 {
		return false
	}

	// Check if it's a standard subdomain (e.g., mars.latency.space)
	if len(parts) == 3 && parts[1] == "latency" && parts[2] == "space" {
		// Verify it's a valid celestial body
		if _, exists := solarSystem[parts[0]]; exists {
			return true
		}
		if _, exists := spacecraft[parts[0]]; exists {
			return true
		}
	}

	// Check if it's a moon subdomain (e.g., enceladus.saturn.latency.space)
	if len(parts) == 4 && parts[2] == "latency" && parts[3] == "space" {
		planet, exists := solarSystem[parts[1]]
		if !exists {
			return false
		}
		// Verify the moon exists for this planet
		if _, moonExists := planet.Moons[parts[0]]; moonExists {
			return true
		}
	}

	// Check if it's our domain.body.latency.space format
	// Any domain followed by a valid celestial body and latency.space is valid
	if len(parts) >= 3 && parts[len(parts)-2] == "latency" && parts[len(parts)-1] == "space" {
		bodyName := parts[len(parts)-3]
		// Check if it's a valid celestial body
		if _, exists := solarSystem[bodyName]; exists {
			return true
		}
		if _, exists := spacecraft[bodyName]; exists {
			return true
		}
		
		// Check for moon format (domain.moon.planet.latency.space)
		if len(parts) >= 4 {
			moonName := parts[len(parts)-4]
			planetName := parts[len(parts)-3]
			if planet, exists := solarSystem[planetName]; exists {
				if _, moonExists := planet.Moons[moonName]; moonExists {
					return true
				}
			}
		}
	}

	return false
}

// Helper function to list all valid domains
func listValidDomains() []string {
	var domains []string
	domains = append(domains, "latency.space")

	// Add planets and spacecraft
	for name := range solarSystem {
		domains = append(domains, name+".latency.space")
		// Add moons for each planet
		for moon := range solarSystem[name].Moons {
			domains = append(domains, moon+"."+name+".latency.space")
		}
	}

	for name := range spacecraft {
		domains = append(domains, name+".latency.space")
	}

	// Add examples of the new format
	domains = append(domains, "www.google.com.earth.latency.space")
	domains = append(domains, "example.com.mars.latency.space")
	domains = append(domains, "api.github.com.jupiter.latency.space")

	return domains
}

func setupTLS() *tls.Config {
	// Create certificate cache directory
	err := os.MkdirAll("certs", 0700)
	if err != nil {
		log.Printf("Warning: Failed to create certs directory: %v", err)
	}

	// Create autocert manager
	manager := &autocert.Manager{
		Cache:  autocert.DirCache("certs"),
		Prompt: autocert.AcceptTOS,
		HostPolicy: func(ctx context.Context, host string) error {
			// Handle empty host (no SNI)
			if host == "" {
				log.Printf("Warning: Missing SNI, using default certificate")
				return nil // Will use default certificate
			}

			if isValidSubdomain(host) {
				log.Printf("Accepting certificate request for: %s", host)
				return nil
			}
			log.Printf("Rejecting certificate request for invalid host: %s", host)
			return fmt.Errorf("host %s not configured", host)
		},
	}

	// Get or create a default certificate
	defaultCert, err := getDefaultCertificate()
	if err != nil {
		log.Printf("Warning: Failed to load default certificate: %v", err)
	}

	// Configure TLS
	return &tls.Config{
		GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
			// Handle missing SNI
			if hello.ServerName == "" {
				log.Printf("Client from %s did not provide SNI", hello.Conn.RemoteAddr())
				if defaultCert != nil {
					return defaultCert, nil
				}
			}
			return manager.GetCertificate(hello)
		},
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.X25519, tls.CurveP256},
		PreferServerCipherSuites: true,
		NextProtos: []string{
			"h2", "http/1.1", "acme-tls/1", // Add ACME protocol support
		},
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA, tls.TLS_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_RSA_WITH_RC4_128_SHA,
		},
	}
}

// Helper function to get or create a default certificate
func getDefaultCertificate() (*tls.Certificate, error) {
	certPath := "certs/default.pem"
	keyPath := "certs/default.key"

	// Check if default certificate exists
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		// Generate self-signed certificate
		cmd := exec.Command("openssl", "req", "-x509", "-newkey", "rsa:4096",
			"-keyout", keyPath,
			"-out", certPath,
			"-days", "365",
			"-nodes",
			"-subj", "/CN=latency.space")

		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("failed to generate default certificate: %v", err)
		}
	}

	// Load the certificate
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load default certificate: %v", err)
	}

	return &cert, nil
}

// Add metrics for certificate operations
func init() {
	// Add prometheus metrics
	tlsHandshakeErrors := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tls_handshake_errors_total",
			Help: "Total number of TLS handshake errors",
		},
		[]string{"error_type"},
	)
	prometheus.MustRegister(tlsHandshakeErrors)
}