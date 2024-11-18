// proxy/src/tls.go
package main

import (
    "context"
    "crypto/tls"
    "fmt"
    "log"
    "os"
    "strings"
    "golang.org/x/crypto/acme/autocert"
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

    return false
}

func setupTLS() *tls.Config {
    // Create certificate cache directory
    err := os.MkdirAll("certs", 0700)
    if err != nil {
        log.Printf("Warning: Failed to create certs directory: %v", err)
    }

    // Create autocert manager
    manager := &autocert.Manager{
        Cache:      autocert.DirCache("certs"),
        Prompt:     autocert.AcceptTOS,
        HostPolicy: func(ctx context.Context, host string) error {
            if isValidSubdomain(host) {
                log.Printf("Accepting certificate request for: %s", host)
                return nil
            }
            log.Printf("Rejecting certificate request for invalid host: %s", host)
            return fmt.Errorf("host %s not configured", host)
        },
    }

    // Configure TLS
    return &tls.Config{
        GetCertificate:           manager.GetCertificate,
        MinVersion:              tls.VersionTLS12,
        CurvePreferences:        []tls.CurveID{tls.X25519, tls.CurveP256},
        PreferServerCipherSuites: true,
        CipherSuites: []uint16{
            tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
            tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
            tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
            tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
            tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
            tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
        },
    }
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

    return domains
}

