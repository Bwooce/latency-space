// proxy/src/socks.go
package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// SOCKSHandler handles SOCKS protocol connections
type SOCKSHandler struct {
	conn     net.Conn
	security *SecurityValidator
	metrics  *MetricsCollector
}

// NewSOCKSHandler creates a new SOCKS connection handler
func NewSOCKSHandler(conn net.Conn, security *SecurityValidator, metrics *MetricsCollector) *SOCKSHandler {
	return &SOCKSHandler{
		conn:     conn,
		security: security,
		metrics:  metrics,
	}
}

// Handle processes a SOCKS connection
func (s *SOCKSHandler) Handle() {
	defer s.conn.Close()

	// Process client greeting
	if !s.handleClientGreeting() {
		return
	}

	// Process client request
	err := s.handleClientRequest()
	if err != nil {
		log.Printf("SOCKS error: %v", err)
	}
}

// handleClientGreeting handles the initial SOCKS5 greeting
func (s *SOCKSHandler) handleClientGreeting() bool {
	// Read the SOCKS version and number of auth methods
	buf := make([]byte, 2)
	if _, err := io.ReadFull(s.conn, buf); err != nil {
		log.Printf("Failed to read SOCKS greeting: %v", err)
		return false
	}

	version, numMethods := buf[0], buf[1]
	if version != SOCKS5_VERSION {
		log.Printf("Unsupported SOCKS version: %d", version)
		return false
	}

	// Read authentication methods
	methods := make([]byte, numMethods)
	if _, err := io.ReadFull(s.conn, methods); err != nil {
		log.Printf("Failed to read SOCKS auth methods: %v", err)
		return false
	}

	// We only support no authentication (method 0)
	for _, method := range methods {
		if method == SOCKS5_NO_AUTH {
			// Send auth method choice (no auth)
			resp := []byte{SOCKS5_VERSION, SOCKS5_NO_AUTH}
			if _, err := s.conn.Write(resp); err != nil {
				log.Printf("Failed to send auth method choice: %v", err)
				return false
			}
			return true
		}
	}

	// No supported auth method
	resp := []byte{SOCKS5_VERSION, SOCKS5_AUTH_NO_ACCEPTABLE}
	if _, err := s.conn.Write(resp); err != nil {
		log.Printf("Failed to send auth rejection: %v", err)
	}
	log.Printf("No supported authentication method")
	return false
}

// handleClientRequest processes the SOCKS5 connection request
func (s *SOCKSHandler) handleClientRequest() error {
	// Read request header
	buf := make([]byte, 4)
	if _, err := io.ReadFull(s.conn, buf); err != nil {
		s.sendReply(SOCKS5_REP_GENERAL_FAILURE, net.IPv4zero, 0)
		return fmt.Errorf("failed to read SOCKS request: %v", err)
	}

	version, cmd, _, addrType := buf[0], buf[1], buf[2], buf[3]
	if version != SOCKS5_VERSION {
		s.sendReply(SOCKS5_REP_GENERAL_FAILURE, net.IPv4zero, 0)
		return fmt.Errorf("unsupported SOCKS version: %d", version)
	}

	// We only support CONNECT command
	if cmd != SOCKS5_CMD_CONNECT {
		s.sendReply(SOCKS5_REP_CMD_NOT_SUPPORTED, net.IPv4zero, 0)
		return fmt.Errorf("unsupported command: %d", cmd)
	}

	// Read destination address based on address type
	var dstAddr string
	var err error

	switch addrType {
	case SOCKS5_ADDR_IPV4:
		// IPv4 address (4 bytes)
		ipv4 := make([]byte, 4)
		if _, err := io.ReadFull(s.conn, ipv4); err != nil {
			s.sendReply(SOCKS5_REP_GENERAL_FAILURE, net.IPv4zero, 0)
			return fmt.Errorf("failed to read IPv4 address: %v", err)
		}
		dstAddr = net.IP(ipv4).String()

	case SOCKS5_ADDR_DOMAIN:
		// Domain name (first byte is length)
		lenBuf := make([]byte, 1)
		if _, err := io.ReadFull(s.conn, lenBuf); err != nil {
			s.sendReply(SOCKS5_REP_GENERAL_FAILURE, net.IPv4zero, 0)
			return fmt.Errorf("failed to read domain length: %v", err)
		}
		domainLen := int(lenBuf[0])
		
		domain := make([]byte, domainLen)
		if _, err := io.ReadFull(s.conn, domain); err != nil {
			s.sendReply(SOCKS5_REP_GENERAL_FAILURE, net.IPv4zero, 0)
			return fmt.Errorf("failed to read domain: %v", err)
		}
		dstAddr = string(domain)
		
		// Process special domain format: address.celestialbody.latency.space
		dstAddr, err = s.processDomainName(dstAddr)
		if err != nil {
			s.sendReply(SOCKS5_REP_HOST_UNREACHABLE, net.IPv4zero, 0)
			return fmt.Errorf("failed to process domain name: %v", err)
		}

	case SOCKS5_ADDR_IPV6:
		// IPv6 address (16 bytes)
		ipv6 := make([]byte, 16)
		if _, err := io.ReadFull(s.conn, ipv6); err != nil {
			s.sendReply(SOCKS5_REP_GENERAL_FAILURE, net.IPv4zero, 0)
			return fmt.Errorf("failed to read IPv6 address: %v", err)
		}
		dstAddr = net.IP(ipv6).String()

	default:
		s.sendReply(SOCKS5_REP_ADDR_NOT_SUPPORTED, net.IPv4zero, 0)
		return fmt.Errorf("unsupported address type: %d", addrType)
	}

	// Read destination port (2 bytes)
	portBuf := make([]byte, 2)
	if _, err := io.ReadFull(s.conn, portBuf); err != nil {
		s.sendReply(SOCKS5_REP_GENERAL_FAILURE, net.IPv4zero, 0)
		return fmt.Errorf("failed to read port: %v", err)
	}
	dstPort := binary.BigEndian.Uint16(portBuf)

	// Destination address in host:port format
	dstAddrPort := net.JoinHostPort(dstAddr, strconv.Itoa(int(dstPort)))

	// Anti-DDoS: Check if destination is in allowed list
	if !s.isAllowedDestination(dstAddr) {
		s.sendReply(SOCKS5_REP_HOST_UNREACHABLE, net.IPv4zero, 0)
		return fmt.Errorf("destination not in allowed list: %s", dstAddr)
	}

	// Extract celestial body and apply latency
	celestialBody, bodyName := getCelestialBodyFromConn(s.conn.RemoteAddr())
	if celestialBody == nil {
		celestialBody, bodyName = solarSystem["earth"], "earth"
	}

	// Calculate latency based on celestial distance
	latency := calculateLatency(celestialBody.Distance * 1e6)
	
	// Anti-DDoS: Only allow bodies with significant latency (>1s)
	// This prevents the proxy from being used for DDoS attacks
	if latency < 1*time.Second {
		log.Printf("Rejecting connection with insufficient latency: %s (%.2f ms)", 
			bodyName, latency.Seconds()*1000)
		s.sendReply(SOCKS5_REP_GENERAL_FAILURE, net.IPv4zero, 0)
		return fmt.Errorf("rejecting request with insufficient latency: %s", bodyName)
	}

	// Apply space latency for the connection
	time.Sleep(latency)

	// Start metrics collection
	start := time.Now()
	defer func() {
		s.metrics.RecordRequest(bodyName, "socks", time.Since(start))
	}()

	// Connect to destination
	log.Printf("SOCKS connect to %s from %s via %s (latency: %v)", 
		dstAddrPort, s.conn.RemoteAddr().String(), bodyName, latency)
		
	// Calculate appropriate timeout based on celestial distance
	// Minimum 30 seconds, maximum 24 hours, plus 3x the one-way latency
	connectTimeout := 30 * time.Second
	if latency > 10*time.Second {
		// For distant bodies, use a timeout that's at least 3x the one-way latency
		// This gives enough time for connection establishment plus latency
		connectTimeout = 3 * latency
		
		// Cap at 24 hours for extremely distant objects like Voyager
		maxTimeout := 24 * time.Hour
		if connectTimeout > maxTimeout {
			connectTimeout = maxTimeout
		}
	}
	
	log.Printf("Using connection timeout of %v for %s", connectTimeout, bodyName)
	target, err := net.DialTimeout("tcp", dstAddrPort, connectTimeout)
	if err != nil {
		// Send appropriate error code based on the error
		switch {
		case strings.Contains(err.Error(), "connection refused"):
			s.sendReply(SOCKS5_REP_CONN_REFUSED, net.IPv4zero, 0)
		case strings.Contains(err.Error(), "no route to host"):
			s.sendReply(SOCKS5_REP_HOST_UNREACHABLE, net.IPv4zero, 0)
		case strings.Contains(err.Error(), "network is unreachable"):
			s.sendReply(SOCKS5_REP_NETWORK_UNREACHABLE, net.IPv4zero, 0)
		default:
			s.sendReply(SOCKS5_REP_GENERAL_FAILURE, net.IPv4zero, 0)
		}
		return fmt.Errorf("failed to connect to %s: %v", dstAddrPort, err)
	}
	defer target.Close()

	// Send success reply with the bound address and port
	// Use the original client's address for simplicity
	localAddr := target.LocalAddr().(*net.TCPAddr)
	s.sendReply(SOCKS5_REP_SUCCESS, localAddr.IP, uint16(localAddr.Port))

	// Start proxying data between client and target
	var wg sync.WaitGroup
	wg.Add(2)

	// Client -> Target with celestial body latency
	go func() {
		defer wg.Done()
		buf := make([]byte, 32*1024)
		for {
			n, err := s.conn.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Printf("Error reading from client: %v", err)
				}
				break
			}

			// Track bandwidth usage
			s.metrics.TrackBandwidth(bodyName, int64(n))

			// Apply latency for each packet
			time.Sleep(latency)

			_, err = target.Write(buf[:n])
			if err != nil {
				log.Printf("Error writing to target: %v", err)
				break
			}
		}
		target.Close()
	}()

	// Target -> Client with celestial body latency
	go func() {
		defer wg.Done()
		buf := make([]byte, 32*1024)
		for {
			n, err := target.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Printf("Error reading from target: %v", err)
				}
				break
			}

			// Track bandwidth usage
			s.metrics.TrackBandwidth(bodyName, int64(n))

			// Apply latency for each packet
			time.Sleep(latency)

			_, err = s.conn.Write(buf[:n])
			if err != nil {
				log.Printf("Error writing to client: %v", err)
				break
			}
		}
		s.conn.Close()
	}()

	// Wait for both goroutines to complete
	wg.Wait()
	
	return nil
}

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
		celestialBody, _ := getCelestialBody(bodyName)
		
		if celestialBody == nil {
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

// getCelestialBodyFromConn extracts the celestial body from the connection
func getCelestialBodyFromConn(addr net.Addr) (*CelestialBody, string) {
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
			celestialBody, fullName := getCelestialBody(bodyName)
			
			if celestialBody != nil {
				log.Printf("Using celestial body from domain: %s", fullName)
				return celestialBody, fullName
			}
		}
	}
	
	// Check if the first part of the hostname is a celestial body
	hostParts := strings.Split(host, ".")
	if len(hostParts) > 0 {
		body, bodyName := getCelestialBody(hostParts[0])
		if body != nil {
			log.Printf("Using celestial body from hostname: %s", bodyName)
			return body, bodyName
		}
	}
	
	// For clients connecting directly via IP, use Earth with minimal latency for testing
	log.Printf("No celestial body detected in hostname, using Earth for connection from %s", host)
	// Default to Earth
	return solarSystem["earth"], "earth"
}