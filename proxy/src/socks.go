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

// SOCKS constants
const (
	SOCKS5_VERSION = 0x05

	// Authentication methods
	SOCKS5_NO_AUTH                = 0x00
	SOCKS5_AUTH_GSSAPI            = 0x01
	SOCKS5_AUTH_USERNAME_PASSWORD = 0x02
	SOCKS5_AUTH_NO_ACCEPTABLE     = 0xFF

	// Command types
	SOCKS5_CMD_CONNECT       = 0x01
	SOCKS5_CMD_BIND          = 0x02
	SOCKS5_CMD_UDP_ASSOCIATE = 0x03

	// Address types
	SOCKS5_ADDR_IPV4   = 0x01
	SOCKS5_ADDR_DOMAIN = 0x03
	SOCKS5_ADDR_IPV6   = 0x04

	// Reply codes
	SOCKS5_REP_SUCCESS             = 0x00
	SOCKS5_REP_GENERAL_FAILURE     = 0x01
	SOCKS5_REP_CONN_NOT_ALLOWED    = 0x02
	SOCKS5_REP_NETWORK_UNREACHABLE = 0x03
	SOCKS5_REP_HOST_UNREACHABLE    = 0x04
	SOCKS5_REP_CONN_REFUSED        = 0x05
	SOCKS5_REP_TTL_EXPIRED         = 0x06
	SOCKS5_REP_CMD_NOT_SUPPORTED   = 0x07
	SOCKS5_REP_ADDR_NOT_SUPPORTED  = 0x08
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

	// --- Handle different commands ---
	switch cmd {
	case SOCKS5_CMD_CONNECT:
		return s.handleConnect(addrType)
	case SOCKS5_CMD_UDP_ASSOCIATE:
		return s.handleUDPAssociate(addrType)
	// case SOCKS5_CMD_BIND: // BIND is not implemented
	// 	s.sendReply(SOCKS5_REP_CMD_NOT_SUPPORTED, net.IPv4zero, 0)
	// 	return fmt.Errorf("unsupported command: BIND")
	default:
		s.sendReply(SOCKS5_REP_CMD_NOT_SUPPORTED, net.IPv4zero, 0)
		return fmt.Errorf("unsupported command: %d", cmd)
	}
}

// handleConnect handles the SOCKS5 CONNECT command
func (s *SOCKSHandler) handleConnect(addrType byte) error {
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
	bodyName, err := getCelestialBodyFromConn(s.conn.RemoteAddr())
	if err != nil {
		log.Printf("No valid body found in %v: %v", s.conn.RemoteAddr(), err)
		// If no body is found, getCelestialBodyFromConn defaults to Mars, so proceed
	}

	// --- Occlusion Check ---
	if celestialObjects == nil {
		log.Printf("Error: celestialObjects not initialized during SOCKS request.")
		s.sendReply(SOCKS5_REP_GENERAL_FAILURE, net.IPv4zero, 0)
		return fmt.Errorf("internal server error: celestial objects not initialized")
	}

	targetObject, targetFound := findObjectByName(celestialObjects, bodyName)
	if !targetFound {
		log.Printf("Error: SOCKS: Target celestial body '%s' not found.", bodyName)
		s.sendReply(SOCKS5_REP_GENERAL_FAILURE, net.IPv4zero, 0)
		return fmt.Errorf("internal server error: target body '%s' not found", bodyName)
	}
	earthObject, earthFound := findObjectByName(celestialObjects, "Earth")
	if !earthFound {
		log.Printf("Error: SOCKS: Earth celestial object not found.")
		s.sendReply(SOCKS5_REP_GENERAL_FAILURE, net.IPv4zero, 0)
		return fmt.Errorf("internal server error: earth object configuration missing")
	}

	occluded, occluder := IsOccluded(earthObject, targetObject, celestialObjects, time.Now())
	if occluded {
		// If occluded is true, occluder is guaranteed to be non-nil by IsOccluded
		log.Printf("SOCKS connection to %s rejected: occluded by %s", bodyName, occluder.Name)
		s.sendReply(SOCKS5_REP_HOST_UNREACHABLE, net.IPv4zero, 0) // Host unreachable due to occlusion
		// Return an error indicating the reason for rejection
		return fmt.Errorf("SOCKS connection rejected: %s occluded by %s", bodyName, occluder.Name)
	}
	// --- End Occlusion Check ---

	// Calculate latency based on celestial distance
	distance := getCurrentDistance(bodyName) // Get distance for latency calc
	latency := CalculateLatency(distance)

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

// handleUDPAssociate handles the SOCKS5 UDP ASSOCIATE command
func (s *SOCKSHandler) handleUDPAssociate(addrType byte) error {
	log.Printf("SOCKS UDP ASSOCIATE request from %s", s.conn.RemoteAddr())

	// Read and discard the client's requested address and port (they are ignored per RFC)
	switch addrType {
	case SOCKS5_ADDR_IPV4:
		discard := make([]byte, 4+2) // IPv4 (4) + port (2)
		if _, err := io.ReadFull(s.conn, discard); err != nil {
			s.sendReply(SOCKS5_REP_GENERAL_FAILURE, net.IPv4zero, 0)
			return fmt.Errorf("failed to read/discard UDP request address: %v", err)
		}
	case SOCKS5_ADDR_DOMAIN:
		lenBuf := make([]byte, 1)
		if _, err := io.ReadFull(s.conn, lenBuf); err != nil {
			s.sendReply(SOCKS5_REP_GENERAL_FAILURE, net.IPv4zero, 0)
			return fmt.Errorf("failed to read/discard UDP domain length: %v", err)
		}
		domainLen := int(lenBuf[0])
		discard := make([]byte, domainLen+2) // Domain + port (2)
		if _, err := io.ReadFull(s.conn, discard); err != nil {
			s.sendReply(SOCKS5_REP_GENERAL_FAILURE, net.IPv4zero, 0)
			return fmt.Errorf("failed to read/discard UDP domain address: %v", err)
		}
	case SOCKS5_ADDR_IPV6:
		discard := make([]byte, 16+2) // IPv6 (16) + port (2)
		if _, err := io.ReadFull(s.conn, discard); err != nil {
			s.sendReply(SOCKS5_REP_GENERAL_FAILURE, net.IPv4zero, 0)
			return fmt.Errorf("failed to read/discard UDP IPv6 address: %v", err)
		}
	default:
		s.sendReply(SOCKS5_REP_ADDR_NOT_SUPPORTED, net.IPv4zero, 0)
		return fmt.Errorf("unsupported address type in UDP ASSOCIATE: %d", addrType)
	}

	// Create UDP socket
	udpConn, err := net.ListenPacket("udp", ":0") // Listen on any available port
	if err != nil {
		s.sendReply(SOCKS5_REP_GENERAL_FAILURE, net.IPv4zero, 0)
		return fmt.Errorf("failed to create UDP socket: %v", err)
	}
	// Don't defer close here, handleUDPRelay will manage it

	// Get the local address and port the UDP socket is bound to
	udpAddr, ok := udpConn.LocalAddr().(*net.UDPAddr)
	if !ok {
		udpConn.Close() // Clean up the socket we just created
		s.sendReply(SOCKS5_REP_GENERAL_FAILURE, net.IPv4zero, 0)
		return fmt.Errorf("failed to get UDP local address")
	}

	log.Printf("SOCKS UDP relay listening on %s", udpAddr.String())

	// Send success reply with the UDP socket's IP and port
	// Use IPv4 address type for simplicity, as requested.
	// If the bound IP is IPv6, we might need a more robust way to get an IPv4 address
	// or send an IPv6 reply if the client supports it. For now, assume IPv4.
	replyIP := udpAddr.IP.To4()
	if replyIP == nil {
		// If not an IPv4 address, try to find an IPv4 interface address
		// This is a simplification; a robust server might need better handling
		addrs, _ := net.InterfaceAddrs()
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					replyIP = ipnet.IP.To4()
					log.Printf("Warning: UDP bound to IPv6, replying with local IPv4 %s", replyIP)
					break
				}
			}
		}
		if replyIP == nil {
			log.Printf("Warning: Could not find suitable IPv4 address for UDP reply, using 0.0.0.0")
			replyIP = net.IPv4zero // Fallback
		}
	}
	s.sendReply(SOCKS5_REP_SUCCESS, replyIP, uint16(udpAddr.Port))

	// Start the UDP relay handler in a new goroutine
	// Pass the UDP connection, the *original* client TCP address (for celestial body/latency calcs),
	// security validator, and metrics collector.
	clientTCPAddr := s.conn.RemoteAddr()
	go s.handleUDPRelay(udpConn, clientTCPAddr, s.security, s.metrics)

	// Keep the TCP connection alive until it's closed or an error occurs
	log.Printf("SOCKS UDP association established for %s. Keeping TCP connection alive.", clientTCPAddr)
	// Simple blocking read to detect connection closure
	buf := make([]byte, 1)
	for {
		_, err := s.conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading from client TCP connection (UDP associate): %v", err)
			} else {
				log.Printf("Client TCP connection closed (UDP associate): %s", clientTCPAddr)
			}
			// Close the UDP socket when the TCP connection closes
			udpConn.Close()
			break // Exit the loop and function
		}
		// We don't expect data here, just waiting for closure
	}

	return nil // TCP connection closed normally
}

// handleUDPRelay manages packet forwarding for a UDP association
func (s *SOCKSHandler) handleUDPRelay(udpConn net.PacketConn, clientTCPAddr net.Addr, security *SecurityValidator, metrics *MetricsCollector) {
	defer udpConn.Close() // Ensure UDP socket is closed when this goroutine exits

	var clientUDPAddr net.Addr // Store the client's source UDP address once we receive the first packet
	buffer := make([]byte, 65535) // Max UDP packet size

	// Determine celestial body and latency based on the *initial* TCP connection
	bodyName, err := getCelestialBodyFromConn(clientTCPAddr)
	if err != nil {
		log.Printf("UDP Relay: Error getting celestial body for %v: %v. Using default.", clientTCPAddr, err)
		// getCelestialBodyFromConn defaults to Mars, proceed with that
	}
	distance := getCurrentDistance(bodyName)
	latency := CalculateLatency(distance)
	log.Printf("UDP Relay for %s (%s): Using body '%s', latency %v", clientTCPAddr, clientUDPAddr, bodyName, latency)


	// Get Earth object for occlusion check (assuming Earth is the proxy location)
	earthObject, earthFound := findObjectByName(celestialObjects, "Earth")
	if !earthFound {
		log.Printf("Error: UDP Relay: Earth celestial object not found. Occlusion checks disabled.")
		// Proceed without occlusion checks if Earth object is missing
	}
	targetObject, targetFound := findObjectByName(celestialObjects, bodyName)
	if !targetFound {
		log.Printf("Error: UDP Relay: Target celestial body '%s' not found. Occlusion checks disabled.", bodyName)
		// Proceed without occlusion checks if target object is missing
	}


	for {
		n, remoteAddr, err := udpConn.ReadFrom(buffer)
		if err != nil {
			log.Printf("UDP Relay: Error reading from UDP socket: %v", err)
			// If the error indicates the socket is closed (likely by the TCP handler), exit gracefully
			if strings.Contains(err.Error(), "use of closed network connection") {
				log.Printf("UDP Relay: Socket closed, terminating for %s.", clientTCPAddr)
				return
			}
			continue // Otherwise, log and try to continue reading
		}

		// First packet received? Identify the client's UDP source address.
		// Compare host parts only, as ports might differ.
		if clientUDPAddr == nil {
			// Simple IP comparison (might fail for complex cases like NAT)
			clientTCPHost, _, _ := net.SplitHostPort(clientTCPAddr.String())
			remoteHost, _, _ := net.SplitHostPort(remoteAddr.String())
			if clientTCPHost == remoteHost {
				clientUDPAddr = remoteAddr
				log.Printf("UDP Relay: Identified client UDP address for %s as %s", clientTCPAddr, clientUDPAddr)
			} else {
				// Packet from an unknown source before client sent anything? Drop it.
				log.Printf("UDP Relay: Dropping packet from unexpected source %s before client %s (%s) sent data.", remoteAddr, clientTCPAddr, clientTCPHost)
				continue
			}
		}


		// Decide if the packet is from the client or an external target
		if remoteAddr.String() == clientUDPAddr.String() {
			// --- Packet from Client -> Target ---
			log.Printf("UDP Relay: Received %d bytes from client %s", n, remoteAddr)

			if n < 6 { // Minimum SOCKS UDP header size (VER+RSV+FRAG+ATYP+DST.ADDR(1)+DST.PORT(2))
				log.Printf("UDP Relay: Packet from client too short (%d bytes), dropping.", n)
				continue
			}

			// Parse SOCKS5 UDP Request Header (RFC 1928 Section 6)
			// +----+------+------+----------+----------+----------+
			// |RSV | FRAG | ATYP | DST.ADDR | DST.PORT |   DATA   |
			// +----+------+------+----------+----------+----------+
			// | 2  |  1   |  1   | Variable |    2     | Variable |
			// +----+------+------+----------+----------+----------+
			rsv := binary.BigEndian.Uint16(buffer[0:2])
			frag := buffer[2]
			addrType := buffer[3]

			if rsv != 0 {
				log.Printf("UDP Relay: RSV field is non-zero (%d), dropping packet.", rsv)
				continue // Reserved field must be 0
			}
			if frag != 0 {
				log.Printf("UDP Relay: Fragmentation not supported (FRAG=%d), dropping packet.", frag)
				continue // We don't support fragmentation
			}

			var dstHost string
			var dstPort uint16
			var dataOffset int

			switch addrType {
			case SOCKS5_ADDR_IPV4:
				if n < 4+4+2 { // Header(4) + IPv4(4) + Port(2)
					log.Printf("UDP Relay: IPv4 packet from client too short (%d bytes), dropping.", n)
					continue
				}
				dstHost = net.IP(buffer[4:8]).String()
				dstPort = binary.BigEndian.Uint16(buffer[8:10])
				dataOffset = 10
			case SOCKS5_ADDR_DOMAIN:
				if n < 4+1 { // Header(4) + DomainLen(1)
					log.Printf("UDP Relay: Domain packet header from client too short (%d bytes), dropping.", n)
					continue
				}
				domainLen := int(buffer[4])
				if n < 4+1+domainLen+2 { // Header(4) + Len(1) + Domain(len) + Port(2)
					log.Printf("UDP Relay: Domain packet from client too short (%d bytes for domain len %d), dropping.", n, domainLen)
					continue
				}
				domain := string(buffer[5 : 5+domainLen])
				dstPort = binary.BigEndian.Uint16(buffer[5+domainLen : 5+domainLen+2])
				dataOffset = 5 + domainLen + 2

				// Process domain (e.g., extract target from .latency.space)
				var err error
				dstHost, err = s.processDomainName(domain)
				if err != nil {
					log.Printf("UDP Relay: Failed to process domain name '%s': %v. Dropping packet.", domain, err)
					continue
				}
				// Note: processDomainName might have returned the original domain if not special format
				// We might need to resolve this domain to an IP here if WriteTo needs an IP.
				// However, net.DialUDP which WriteTo uses often handles resolution. Let's try first.

			case SOCKS5_ADDR_IPV6:
				if n < 4+16+2 { // Header(4) + IPv6(16) + Port(2)
					log.Printf("UDP Relay: IPv6 packet from client too short (%d bytes), dropping.", n)
					continue
				}
				dstHost = net.IP(buffer[4:20]).String()
				dstPort = binary.BigEndian.Uint16(buffer[20:22])
				dataOffset = 22
			default:
				log.Printf("UDP Relay: Unsupported address type (%d) from client, dropping packet.", addrType)
				continue
			}

			if dataOffset > n {
				log.Printf("UDP Relay: Calculated data offset (%d) exceeds packet size (%d), dropping.", dataOffset, n)
				continue // Should not happen if previous length checks passed, but safety first
			}
			payload := buffer[dataOffset:n]
			dstAddrPort := net.JoinHostPort(dstHost, strconv.Itoa(int(dstPort)))

			// --- Security Checks ---
			// Use a dummy scheme for IsAllowedHost check
			if !security.IsAllowedHost("http://" + dstHost) {
				log.Printf("UDP Relay: Destination host %s not allowed, dropping packet.", dstHost)
				continue
			}
			// Check port validity (using the same SOCKS validator logic)
			if err := security.ValidateSocksDestination(dstHost, dstPort); err != nil {
				log.Printf("UDP Relay: Destination port %d not allowed for host %s: %v, dropping packet.", dstPort, dstHost, err)
				continue
			}

			// --- Occlusion Check ---
			if earthFound && targetFound { // Only check if we found both Earth and the target body
				occluded, occluder := IsOccluded(earthObject, targetObject, celestialObjects, time.Now())
				if occluded {
					log.Printf("UDP Relay: Path to %s occluded by %s, dropping packet.", bodyName, occluder.Name)
					continue
				}
			} else {
				// Log if occlusion check is skipped
                // log.Printf("UDP Relay: Occlusion check skipped (EarthFound: %v, TargetFound: %v)", earthFound, targetFound)
			}
			// --- End Occlusion Check ---


			log.Printf("UDP Relay: Relaying %d bytes from %s to %s (via %s, latency %v)",
				len(payload), clientUDPAddr, dstAddrPort, bodyName, latency)

			// Apply forward latency
			time.Sleep(latency)

			// Send payload to destination
			targetUDPAddr, err := net.ResolveUDPAddr("udp", dstAddrPort)
			if err != nil {
				log.Printf("UDP Relay: Failed to resolve destination UDP address %s: %v", dstAddrPort, err)
				continue
			}

			_, err = udpConn.WriteTo(payload, targetUDPAddr)
			if err != nil {
				log.Printf("UDP Relay: Error writing %d bytes to target %s: %v", len(payload), targetUDPAddr, err)
				// Don't necessarily continue; could be a temporary error
			}

			// Record metrics (outgoing bandwidth from client perspective)
			metrics.TrackBandwidth(bodyName, int64(len(payload)))

		} else {
			// --- Packet from External Target -> Client --- (Stateless approach)
			log.Printf("UDP Relay: Received %d bytes from external source %s (presumed target reply)", n, remoteAddr)
			targetUDPAddr, ok := remoteAddr.(*net.UDPAddr)
			if !ok {
				log.Printf("UDP Relay: Received packet from non-UDP source %s? Dropping.", remoteAddr)
				continue
			}

			if clientUDPAddr == nil {
				log.Printf("UDP Relay: Received packet from target %s before client %s sent data. Dropping.", remoteAddr, clientTCPAddr)
				continue // Don't know where to send it back
			}


			// Construct SOCKS5 UDP Header for the reply
			var replyHeader []byte
			var atyp byte
			var addrBytes []byte

			if targetUDPAddr.IP.To4() != nil {
				atyp = SOCKS5_ADDR_IPV4
				addrBytes = targetUDPAddr.IP.To4()
			} else if targetUDPAddr.IP.To16() != nil {
				atyp = SOCKS5_ADDR_IPV6
				addrBytes = targetUDPAddr.IP.To16()
			} else {
				log.Printf("UDP Relay: Cannot determine address type for target reply source %s. Dropping packet.", targetUDPAddr.IP)
				continue
			}

			replyHeader = []byte{
				0x00, 0x00, // RSV
				0x00, // FRAG
				atyp, // Address Type
			}
			replyHeader = append(replyHeader, addrBytes...) // Target Address
			portBytes := make([]byte, 2)
			binary.BigEndian.PutUint16(portBytes, uint16(targetUDPAddr.Port))
			replyHeader = append(replyHeader, portBytes...) // Target Port

			// Combine header and original payload
			fullReply := append(replyHeader, buffer[:n]...) // n is the size of the payload received from target


			log.Printf("UDP Relay: Relaying %d bytes from target %s back to client %s (via %s, latency %v)",
						n, remoteAddr, clientUDPAddr, bodyName, latency)

			// Apply return latency
			time.Sleep(latency)

			// Send the full SOCKS UDP packet back to the client
			_, err = udpConn.WriteTo(fullReply, clientUDPAddr)
			if err != nil {
				log.Printf("UDP Relay: Error writing %d bytes back to client %s: %v", len(fullReply), clientUDPAddr, err)
			}

			// Record metrics (incoming packet to client perspective)
			metrics.RecordUDPPacket(bodyName, int64(n))
		}
	}
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
