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

// isValidSubdomain checks if a hostname is valid for ACME certificate issuance
// based on defined patterns (base domain, body.domain, moon.planet.domain).
// It performs case-insensitive checks.
func isValidSubdomain(host string) bool {
	lowerHost := strings.ToLower(host)

	// Allow the base domain and www subdomain
	if lowerHost == "latency.space" || lowerHost == "www.latency.space" {
		return true
	}

	// Split hostname by dots.
	parts := strings.Split(lowerHost, ".")
	numParts := len(parts)

	// Check for the required suffix ".latency.space"
	if numParts < 3 || parts[numParts-1] != "space" || parts[numParts-2] != "latency" {
		return false // Doesn't end correctly
	}

	// Check for standard body subdomain (e.g., mars.latency.space).
	// Requires exactly 3 parts: body.latency.space
	if numParts == 3 {
		bodyName := parts[0]
		_, found := findObjectByName(celestialObjects, bodyName)
		return found // Valid if the body name exists
	}

	// Check for moon subdomain (e.g., phobos.mars.latency.space).
	// Requires exactly 4 parts: moon.planet.latency.space
	if numParts == 4 {
		moonName := parts[0]
		planetName := parts[1]
		moon, moonFound := findObjectByName(celestialObjects, moonName)
		// Check if moon exists, is a moon, and its parent matches the planet part
		return moonFound && moon.Type == "moon" && strings.EqualFold(moon.ParentName, planetName)
	}

	// Check for target routing format (e.g., example.com.mars.latency.space).
	// Requires >= 4 parts: target...body.latency.space
	if numParts >= 4 {
		bodyName := parts[numParts-3]
		// Check if it's a valid celestial body (non-moon)
		body, found := findObjectByName(celestialObjects, bodyName)
		if found && body.Type != "moon" {
			return true
		}

		// Check for target routing moon format (e.g., example.com.phobos.mars.latency.space).
		// Requires >= 5 parts: target...moon.planet.latency.space
		if numParts >= 5 {
			moonName := parts[numParts-4]
			planetName := parts[numParts-3]
			moon, moonFound := findObjectByName(celestialObjects, moonName)
			planet, planetFound := findObjectByName(celestialObjects, planetName)
			// Check if moon and planet exist, moon type is correct, and parent matches
			if moonFound && planetFound && moon.Type == "moon" && strings.EqualFold(moon.ParentName, planetName) {
				return true
			}
		}
	}

	return false // No valid pattern matched
}


// setupTLS configures and returns a *tls.Config suitable for the HTTPS server,
// including ACME autocert support for automatic certificate management.
func setupTLS() *tls.Config {
	// Ensure the certificate cache directory exists.
	err := os.MkdirAll("certs", 0700)
	if err != nil {
		// Log warning but proceed, autocert might handle it or use memory cache
		log.Printf("Warning: Failed to create certs directory: %v", err)
	}

	// Create the autocert manager.
	manager := &autocert.Manager{
		Cache:  autocert.DirCache("certs"), // Cache certificates in the "certs" directory
		Prompt: autocert.AcceptTOS,        // Automatically accept Let's Encrypt TOS
		// HostPolicy defines which hostnames are allowed for certificate requests.
		HostPolicy: func(ctx context.Context, host string) error {
			// Handle requests without Server Name Indication (SNI).
			// These will use the default certificate generated later.
			if host == "" {
				log.Println("TLS: Request missing SNI, will use default certificate.")
				// Returning nil here allows autocert to proceed, but GetCertificate below handles it.
				return nil
			}

			// Validate the requested hostname against allowed patterns.
			if isValidSubdomain(host) {
				log.Printf("TLS: Accepting certificate request for valid host: %s", host)
				return nil // Host is allowed
			}

			// Reject requests for invalid hostnames.
			log.Printf("TLS: Rejecting certificate request for invalid host: %s", host)
			return fmt.Errorf("hostname %s is not allowed by HostPolicy", host)
		},
		Email: os.Getenv("SSL_EMAIL"), // Get email from environment variable for ACME account
	}

	// Get or generate a default self-signed certificate for requests without SNI or for fallback.
	defaultCert, err := getDefaultCertificate()
	if err != nil {
		// Log warning but the server might still function for valid SNI requests
		log.Printf("Warning: Failed to load or generate default TLS certificate: %v", err)
	}

	// Configure the main TLS settings.
	return &tls.Config{
		// GetCertificate is called by the TLS handshake process.
		// It uses the autocert manager for valid hostnames and falls back to the default cert.
		GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
			// Use default certificate if SNI is missing and default cert is available.
			if hello.ServerName == "" {
				if defaultCert != nil {
					// log.Printf("TLS: Using default certificate for request without SNI from %s", hello.Conn.RemoteAddr())
					return defaultCert, nil
				}
				// If defaultCert is nil, let autocert handle it (which might fail if no cert exists yet)
				log.Println("TLS: Warning - No SNI provided and no default certificate available.")
			}
			// For requests with SNI, delegate to the autocert manager.
			return manager.GetCertificate(hello)
		},
		MinVersion:               tls.VersionTLS12, // Enforce minimum TLS 1.2
		CurvePreferences:         []tls.CurveID{tls.X25519, tls.CurveP256}, // Prefer modern curves
		PreferServerCipherSuites: true,
		// Define supported application layer protocols (HTTP/2, HTTP/1.1, ACME TLS challenge)
		NextProtos: []string{
			"h2", "http/1.1", "acme-tls/1",
		},
		// Define preferred cipher suites (modern and secure options first)
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305, // Go 1.8+
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,   // Go 1.8+
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			// Add fallback ciphers if needed, but prioritize modern ones
			// tls.TLS_RSA_WITH_AES_256_GCM_SHA384, // Requires Go 1.5+
			// tls.TLS_RSA_WITH_AES_128_GCM_SHA256, // Requires Go 1.5+
		},
	}
}

// getDefaultCertificate loads the default certificate or generates a self-signed one if it doesn't exist.
// This is used for clients that don't provide SNI.
func getDefaultCertificate() (*tls.Certificate, error) {
	certPath := "certs/default.pem"
	keyPath := "certs/default.key"

	// Check if the default certificate files exist.
	_, certErr := os.Stat(certPath)
	_, keyErr := os.Stat(keyPath)

	// If either file is missing, attempt to generate a new self-signed certificate.
	if os.IsNotExist(certErr) || os.IsNotExist(keyErr) {
		log.Println("Default certificate not found, generating a self-signed certificate...")
		// Generate a self-signed certificate using openssl if it doesn't exist.
		// Requires openssl to be installed and in the system PATH.
		cmd := exec.Command("openssl", "req", "-x509", "-newkey", "rsa:4096",
			"-keyout", keyPath,
			"-out", certPath,
			"-days", "365", // Valid for 1 year
			"-nodes",      // Do not encrypt the private key
			"-subj", "/CN=latency.space") // Common Name for the certificate

		// Run the command and capture output/error
		output, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("failed to generate default certificate using openssl: %v\nOutput: %s", err, string(output))
		}
		log.Println("Successfully generated self-signed default certificate.")
	}

	// Load the certificate and key pair from the files.
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load default certificate from %s and %s: %v", certPath, keyPath, err)
	}

	log.Println("Loaded default certificate.")
	return &cert, nil
}

// Register Prometheus metrics for TLS operations.
func init() {
	tlsHandshakeErrors := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tls_handshake_errors_total",
			Help: "Total number of TLS handshake errors encountered by the server.",
		},
		[]string{"reason"}, // Label by error reason if possible (might be hard to capture specific reasons)
	)
	prometheus.MustRegister(tlsHandshakeErrors)
	// TODO: Potentially add more TLS-related metrics (e.g., versions, cipher suites used)
}
