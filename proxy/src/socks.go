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
	s.handleClientRequest()
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
	s.conn.Write(resp)
	log.Printf("No supported authentication method")
	return false
}

// handleClientRequest processes the SOCKS5 connection request
func (s *SOCKSHandler) handleClientRequest() {
	// Read request header
	buf := make([]byte, 4)
	if _, err := io.ReadFull(s.conn, buf); err != nil {
		log.Printf("Failed to read SOCKS request: %v", err)
		return
	}

	version, cmd, _, addrType := buf[0], buf[1], buf[2], buf[3]
	if version != SOCKS5_VERSION {
		log.Printf("Unsupported SOCKS version: %d", version)
		return
	}

	// We only support CONNECT command
	if cmd != SOCKS5_CMD_CONNECT {
		s.sendReply(SOCKS5_REP_CMD_NOT_SUPPORTED, net.IPv4zero, 0)
		log.Printf("Unsupported command: %d", cmd)
		return
	}

	// Read destination address based on address type
	var dstAddr string
	var err error

	switch addrType {
	case SOCKS5_ADDR_IPV4:
		// IPv4 address (4 bytes)
		ipv4 := make([]byte, 4)
		if _, err := io.ReadFull(s.conn, ipv4); err != nil {
			log.Printf("Failed to read IPv4 address: %v", err)
			return
		}
		dstAddr = net.IP(ipv4).String()

	case SOCKS5_ADDR_DOMAIN:
		// Domain name (first byte is length)
		lenBuf := make([]byte, 1)
		if _, err := io.ReadFull(s.conn, lenBuf); err != nil {
			log.Printf("Failed to read domain length: %v", err)
			return
		}
		domainLen := int(lenBuf[0])
		
		domain := make([]byte, domainLen)
		if _, err := io.ReadFull(s.conn, domain); err != nil {
			log.Printf("Failed to read domain: %v", err)
			return
		}
		dstAddr = string(domain)
		
		// Process special domain format: address.celestialbody.latency.space
		dstAddr, err = s.processDomainName(dstAddr)
		if err != nil {
			log.Printf("Failed to process domain name: %v", err)
			s.sendReply(SOCKS5_REP_HOST_UNREACHABLE, net.IPv4zero, 0)
			return
		}

	case SOCKS5_ADDR_IPV6:
		// IPv6 address (16 bytes)
		ipv6 := make([]byte, 16)
		if _, err := io.ReadFull(s.conn, ipv6); err != nil {
			log.Printf("Failed to read IPv6 address: %v", err)
			return
		}
		dstAddr = net.IP(ipv6).String()

	default:
		s.sendReply(SOCKS5_REP_ADDR_NOT_SUPPORTED, net.IPv4zero, 0)
		log.Printf("Unsupported address type: %d", addrType)
		return
	}

	// Read destination port (2 bytes)
	portBuf := make([]byte, 2)
	if _, err := io.ReadFull(s.conn, portBuf); err != nil {
		log.Printf("Failed to read port: %v", err)
		return
	}
	dstPort := binary.BigEndian.Uint16(portBuf)

	// Destination address in host:port format
	dstAddrPort := net.JoinHostPort(dstAddr, strconv.Itoa(int(dstPort)))

	// Extract celestial body and apply latency
	celestialBody, bodyName := getCelestialBodyFromConn(s.conn.RemoteAddr())
	if celestialBody == nil {
		celestialBody, bodyName = solarSystem["earth"], "earth"
	}

	// Apply space latency for the connection
	latency := calculateLatency(celestialBody.Distance * 1e6)
	time.Sleep(latency)

	// Start metrics collection
	start := time.Now()
	defer func() {
		s.metrics.RecordRequest(bodyName, "socks", time.Since(start))
	}()

	// Connect to destination
	log.Printf("SOCKS connect to %s from %s", dstAddrPort, s.conn.RemoteAddr().String())
	target, err := net.DialTimeout("tcp", dstAddrPort, 10*time.Second)
	if err != nil {
		log.Printf("Failed to connect to %s: %v", dstAddrPort, err)
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
		return
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
		if len(parts) >= 3 {
			// Extract target domain (everything before the celestial body part)
			targetDomain := strings.Join(parts[:len(parts)-2], ".")
			return targetDomain, nil
		}
		return "", fmt.Errorf("invalid latency.space domain format")
	}
	return domain, nil
}

// getCelestialBodyFromConn extracts the celestial body from the connection
func getCelestialBodyFromConn(addr net.Addr) (*CelestialBody, string) {
	// In a real implementation, you might associate client IPs with celestial bodies
	// For now, we'll default to Earth
	return solarSystem["earth"], "earth"
}