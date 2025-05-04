package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
)

// sendReply sends a SOCKS5 reply message
func (s *SOCKSHandler) sendReply(rep byte, ip net.IP, port uint16) {
	// Determine address type and address bytes
	var atyp byte
	var addr []byte

	if ip.To4() != nil {
		atyp = SOCKS5_ADDR_IPV4
		addr = ip.To4()
	} else if ip.To16() != nil {
		atyp = SOCKS5_ADDR_IPV6
		addr = ip.To16()
	} else {
		// Default to IPv4 null address
		atyp = SOCKS5_ADDR_IPV4
		addr = net.IPv4zero.To4()
	}

	// Create reply message
	reply := []byte{
		SOCKS5_VERSION, // SOCKS version
		rep,            // Reply code
		0x00,           // Reserved
		atyp,           // Address type
	}

	// Add address bytes
	reply = append(reply, addr...)

	// Add port (2 bytes, big endian)
	portBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(portBytes, port)
	reply = append(reply, portBytes...)

	// Send reply
	_, err := s.conn.Write(reply)
	if err != nil {
		log.Printf("Failed to send SOCKS reply: %v", err)
	}
}

// processDomainName checks if the domain has our latency.space suffix
// and extracts the actual destination host if needed
func (s *SOCKSHandler) processDomainName(domain string) (string, error) {
	// Check if this is our special format
	if strings.HasSuffix(domain, ".latency.space") {
		parts := strings.Split(domain, ".")
		if len(parts) < 3 {
			return "", fmt.Errorf("invalid latency.space domain format")
		}

		// If format is domain.body.latency.space
		// Extract the celestial body and target domain
		// The celestial body is the second-to-last part before "latency.space"
		bodyIndex := len(parts) - 3

		// Everything before the celestial body is the target domain
		targetParts := parts[:bodyIndex]
		if len(targetParts) == 0 {
			return "", fmt.Errorf("missing target domain in latency.space format")
		}

		targetDomain := strings.Join(targetParts, ".")

		// Get the celestial body name for logging
		bodyName := parts[bodyIndex]
		_, found := findObjectByName(celestialObjects, bodyName)

		if !found {
			return "", fmt.Errorf("unknown celestial body: %s", bodyName)
		}

		log.Printf("SOCKS: Extracted target domain %s from %s.latency.space format", targetDomain, bodyName)
		return targetDomain, nil
	}
	return domain, nil
}

// isAllowedDestination checks if a destination is in the allowed list
func (s *SOCKSHandler) isAllowedDestination(host string) bool {
	// Create a dummy URL to use the security validator
	url := "http://" + host
	allowed := s.security.IsAllowedHost(url)

	// Log the result for debugging
	if !allowed {
		log.Printf("SOCKS destination rejected: %s is not in the allowed list", host)
	} else {
		log.Printf("SOCKS destination allowed: %s", host)
	}

	// Enforce the allowed hosts list to prevent proxy abuse
	// This is an important security measure
	return allowed
}

// isNetClosingErr checks if an error indicates a closed network connection.
func isNetClosingErr(err error) bool {
	if err == nil {
		return false
	}
	// Check for specific sentinel errors
	if errors.Is(err, net.ErrClosed) {
		return true
	}
	// Check for common string patterns (less ideal but often necessary)
	// Use ToLower to make the check case-insensitive
	errString := strings.ToLower(err.Error())
	// Common errors across different OSes when a connection is closed
	if strings.Contains(errString, "use of closed network connection") || // Go's standard error
		strings.Contains(errString, "closed connection") || // Sometimes seen
		strings.Contains(errString, "broken pipe") || // Often on write after close (client->target)
		strings.Contains(errString, "connection reset by peer") || // Remote side closed forcefully
		strings.Contains(errString, "forcibly closed by the remote host") || // Windows specific
		strings.Contains(errString, "operation on closed file") { // Can happen with UDP sockets too
		return true
	}
	return false
}

// getCelestialBodyFromConn extracts the celestial body from the connection
func getCelestialBodyFromConn(addr net.Addr) (string, error) {
	host := addr.String()

	// Extract the host part without port
	if idx := strings.Index(host, ":"); idx > 0 {
		host = host[:idx]
	}

	// Log the connection host for debugging
	log.Printf("SOCKS connection from host: %s", host)

	// Check if this is a celestial body domain
	if strings.HasSuffix(host, ".latency.space") {
		parts := strings.Split(host, ".")
		if len(parts) >= 3 {
			// The celestial body is the second-to-last part before "latency.space"
			bodyIndex := len(parts) - 3
			bodyName := parts[bodyIndex]
			celestialBody, found := findObjectByName(celestialObjects, bodyName)
			if found {
				log.Printf("Using celestial body from domain: %s", celestialBody.Name)
				return celestialBody.Name, nil
			}
		}
	}

	// Check if the first part of the hostname is a celestial body
	hostParts := strings.Split(host, ".")
	if len(hostParts) > 0 {
		body, found := findObjectByName(celestialObjects, hostParts[0])
		if found {
			log.Printf("Using celestial body from hostname: %s", body.Name)
			return body.Name, nil
		}
	}

	// For clients connecting directly via IP, use Earth with minimal latency for testing
	log.Printf("No celestial body detected in hostname, using Mars for connection from |%s|", host)
	body, _ := findObjectByName(celestialObjects, "Mars")
	return body.Name, nil
}