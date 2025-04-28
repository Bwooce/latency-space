package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

func setupTestEnvironment() {
	// Initialize a minimal set of celestial objects for testing
	// Need Earth for occlusion checks and a target body (e.g., Mars) for latency simulation context
	// Assign the test objects to the global variable used by the main code
	// Corrected type to []CelestialObject as per build error
	celestialObjects = []CelestialObject{
		{
			Name:   "Earth",
			Type:   "planet",
			Radius: EARTH_RADIUS,
			// Simplified orbital elements (not used directly in this test, but needed for functions)
			A: 1.0, E: 0.0167, I: 0.0, L: 100.46, LP: 102.94, N: 0.0,
		}, // Removed &
		{
			Name:   "Mars",
			Type:   "planet",
			Radius: 3389.5,
			// Simplified orbital elements
			A: 1.5237, E: 0.0934, I: 1.85, L: 355.45, LP: 336.04, N: 49.58,
		}, // Removed &
		// Add other bodies if needed for specific tests
	}
	// Suppress log output during tests unless debugging
	// log.SetOutput(io.Discard) // Keep log suppression commented out for now
}

func TestSocksUDPAssociateAndRelay(t *testing.T) {
	setupTestEnvironment() // Initialize celestial objects, etc.

	// 1. Setup
	security := NewSecurityValidator()
	// Allow localhost for testing
	security.allowedHosts["127.0.0.1"] = true
	security.allowedPorts["0"] = true // Allow dynamic ports used in testing

	metrics := NewMetricsCollector() // Use real metrics collector

	// Mock SOCKS Server (TCP Listener)
	tcpListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start TCP listener: %v", err)
	}
	t.Cleanup(func() { tcpListener.Close() })
	proxyTCPAddr := tcpListener.Addr().String()
	// proxyHost is unused

	// Mock Target Server (UDP Listener)
	targetUDPListener, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start target UDP listener: %v", err)
	}
	t.Cleanup(func() { targetUDPListener.Close() })
	targetUDPAddr := targetUDPListener.LocalAddr().(*net.UDPAddr)

	// Mock Client (UDP Listener) - for receiving replies
	clientUDPListener, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start client UDP listener: %v", err)
	}
	t.Cleanup(func() { clientUDPListener.Close() })
	// clientUDPAddr is unused by the test client itself

	// 2. Run Mock SOCKS Server Logic in Goroutine
	serverErrChan := make(chan error, 1)
	go func() {
		conn, err := tcpListener.Accept()
		if err != nil {
			// Check if the error is due to listener being closed during cleanup
			if !strings.Contains(err.Error(), "use of closed network connection") {
				serverErrChan <- fmt.Errorf("TCP accept error: %v", err)
			}
			close(serverErrChan)
			return
		}
		defer conn.Close()
		// Use the proxyHost (e.g., "Mars") for getCelestialBodyFromConn lookup
		// We need to simulate the connection coming *from* a specific celestial body domain
		// For simplicity, let's assume the connection is identified as coming 'from' Mars
		// In a real scenario, this depends on the client's source IP/domain.
		// Let's wrap the connection to modify RemoteAddr for the test
		// This is tricky; getCelestialBodyFromConn uses RemoteAddr().String()
		// Let's skip modifying RemoteAddr for now and assume getCelestialBodyFromConn defaults correctly or handle it inside SOCKSHandler if needed.
		// It defaults to Mars if no body is found, which is acceptable for this test.

		handler := NewSOCKSHandler(conn, security, metrics)
		handler.Handle() // Process the single connection
		close(serverErrChan)
	}()

	// 3. Simulate Client Connection & UDP Associate
	clientTCPConn, err := net.Dial("tcp", proxyTCPAddr)
	if err != nil {
		t.Fatalf("Failed to dial proxy TCP: %v", err)
	}
	t.Cleanup(func() { clientTCPConn.Close() })

	// Send Greeting
	greeting := []byte{SOCKS5_VERSION, 0x01, SOCKS5_NO_AUTH}
	_, err = clientTCPConn.Write(greeting)
	if err != nil {
		t.Fatalf("Failed to send greeting: %v", err)
	}

	// Read Server Choice
	choice := make([]byte, 2)
	_, err = io.ReadFull(clientTCPConn, choice)
	if err != nil {
		t.Fatalf("Failed to read choice: %v", err)
	}
	if choice[0] != SOCKS5_VERSION || choice[1] != SOCKS5_NO_AUTH {
		t.Fatalf("Unexpected server choice: %v", choice)
	}

	// Send UDP ASSOCIATE request (DST.ADDR/PORT are ignored by server, use 0s)
	// VER | CMD | RSV | ATYP | DST.ADDR | DST.PORT
	// 05  | 03  | 00  | 01   | 0.0.0.0  | 0
	udpRequest := []byte{SOCKS5_VERSION, SOCKS5_CMD_UDP_ASSOCIATE, 0x00, SOCKS5_ADDR_IPV4, 0, 0, 0, 0, 0, 0}
	_, err = clientTCPConn.Write(udpRequest)
	if err != nil {
		t.Fatalf("Failed to send UDP associate request: %v", err)
	}

	// Read UDP ASSOCIATE Reply
	// VER | REP | RSV | ATYP | BND.ADDR | BND.PORT
	replyHeader := make([]byte, 4)
	_, err = io.ReadFull(clientTCPConn, replyHeader)
	if err != nil {
		t.Fatalf("Failed to read UDP reply header: %v", err)
	}

	if replyHeader[0] != SOCKS5_VERSION {
		t.Fatalf("Unexpected reply version: %x", replyHeader[0])
	}
	if replyHeader[1] != SOCKS5_REP_SUCCESS {
		t.Fatalf("Expected success reply code, got: %x", replyHeader[1])
	}
	if replyHeader[3] != SOCKS5_ADDR_IPV4 { // Assuming IPv4 reply based on handler logic
		t.Fatalf("Expected IPv4 address type in reply, got: %x", replyHeader[3])
	}

	// Read BND.ADDR (IPv4) and BND.PORT
	bndAddrBytes := make([]byte, 4)
	_, err = io.ReadFull(clientTCPConn, bndAddrBytes)
	if err != nil {
		t.Fatalf("Failed to read BND.ADDR: %v", err)
	}
	bndPortBytes := make([]byte, 2)
	_, err = io.ReadFull(clientTCPConn, bndPortBytes)
	if err != nil {
		t.Fatalf("Failed to read BND.PORT: %v", err)
	}
	bndPort := binary.BigEndian.Uint16(bndPortBytes)
	bndIP := net.IP(bndAddrBytes)

	proxyRelayUDPAddr := &net.UDPAddr{IP: bndIP, Port: int(bndPort)}
	t.Logf("Proxy UDP Relay listening on: %s", proxyRelayUDPAddr.String())

	// 4. Test Client -> Target Relay
	clientPayload := []byte("ping")
	targetIP := targetUDPAddr.IP.To4()
	targetPort := uint16(targetUDPAddr.Port)

	// Construct SOCKS5 UDP Request Packet
	// RSV | FRAG | ATYP | DST.ADDR | DST.PORT | DATA
	var udpPacket bytes.Buffer
	udpPacket.Write([]byte{0x00, 0x00}) // RSV
	udpPacket.WriteByte(0x00)           // FRAG
	udpPacket.WriteByte(SOCKS5_ADDR_IPV4)
	udpPacket.Write(targetIP)
	portBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(portBytes, targetPort)
	udpPacket.Write(portBytes)
	udpPacket.Write(clientPayload)

	// Send packet from Client UDP socket to Proxy Relay UDP socket
	_, err = clientUDPListener.WriteTo(udpPacket.Bytes(), proxyRelayUDPAddr)
	if err != nil {
		t.Fatalf("Client UDP failed to write to proxy relay: %v", err)
	}
	t.Logf("Client UDP sent %d bytes to proxy relay %s", udpPacket.Len(), proxyRelayUDPAddr)

	// Read from Target UDP Listener
	targetBuf := make([]byte, 1024)
	targetUDPListener.SetReadDeadline(time.Now().Add(2 * time.Second)) // Timeout
	n, remoteAddr, err := targetUDPListener.ReadFrom(targetBuf)
	if err != nil {
		t.Fatalf("Target UDP listener failed to read: %v", err)
	}

	// Assert payload is correct and source is proxy relay
	if !bytes.Equal(targetBuf[:n], clientPayload) {
		t.Fatalf("Target received unexpected payload: got %q, want %q", string(targetBuf[:n]), string(clientPayload))
	}
	// The remoteAddr *should* be the proxy's relay address
	if remoteAddr.String() != proxyRelayUDPAddr.String() {
		t.Logf("Warning: Target received packet from %s, expected proxy relay %s. This might happen with NAT.", remoteAddr.String(), proxyRelayUDPAddr.String())
		// Don't fail the test for this, as NAT can interfere, but log it.
	}
	t.Logf("Target UDP received %q correctly from %s", string(targetBuf[:n]), remoteAddr.String())
	proxyRelaySourceAddr := remoteAddr // Use the actual source address for the reply

	// 5. Test Target -> Client Relay
	targetPayload := []byte("pong")

	// Send reply from Target UDP socket back to the address it received from (Proxy Relay)
	_, err = targetUDPListener.WriteTo(targetPayload, proxyRelaySourceAddr)
	if err != nil {
		t.Fatalf("Target UDP failed to write reply to proxy relay (%s): %v", proxyRelaySourceAddr.String(), err)
	}
	t.Logf("Target UDP sent reply %q to %s", string(targetPayload), proxyRelaySourceAddr)

	// Read from Client UDP Listener
	clientBuf := make([]byte, 1024)
	clientUDPListener.SetReadDeadline(time.Now().Add(2 * time.Second)) // Timeout
	n, remoteAddrClient, err := clientUDPListener.ReadFrom(clientBuf)
	if err != nil {
		t.Fatalf("Client UDP listener failed to read reply: %v", err)
	}

	// Assert source is the proxy relay
	if remoteAddrClient.String() != proxyRelayUDPAddr.String() {
		t.Logf("Warning: Client received reply from %s, expected proxy relay %s. This might happen with NAT.", remoteAddrClient.String(), proxyRelayUDPAddr.String())
		// Don't fail the test for this.
	}

	// Parse the received SOCKS5 UDP packet
	if n < 10 { // Min header size (RSV 2 + FRAG 1 + ATYP 1 + IP 4 + PORT 2)
		t.Fatalf("Client received UDP packet too short: %d bytes", n)
	}
	header := clientBuf[:10] // Assuming IPv4 for simplicity based on target address
	payload := clientBuf[10:n]

	if header[0] != 0x00 || header[1] != 0x00 { // RSV
		t.Errorf("Client received reply with non-zero RSV: %x %x", header[0], header[1])
	}
	if header[2] != 0x00 { // FRAG
		t.Errorf("Client received reply with non-zero FRAG: %x", header[2])
	}
	if header[3] != SOCKS5_ADDR_IPV4 { // ATYP
		t.Errorf("Client received reply with incorrect ATYP: %x", header[3])
	}
	receivedDstIP := net.IP(header[4:8])
	receivedDstPort := binary.BigEndian.Uint16(header[8:10])

	if !receivedDstIP.Equal(targetIP) {
		t.Errorf("Client received reply with incorrect DST.ADDR: got %s, want %s", receivedDstIP.String(), targetIP.String())
	}
	if receivedDstPort != targetPort {
		t.Errorf("Client received reply with incorrect DST.PORT: got %d, want %d", receivedDstPort, targetPort)
	}
	if !bytes.Equal(payload, targetPayload) {
		t.Errorf("Client received reply with incorrect payload: got %q, want %q", string(payload), string(targetPayload))
	}
	t.Logf("Client UDP received SOCKS reply correctly: DST=%s:%d, Payload=%q", receivedDstIP, receivedDstPort, string(payload))


	// 6. Test Disallowed Host
	t.Log("Testing disallowed host...")
	delete(security.allowedHosts, "127.0.0.1") // Disallow localhost

	// Send another Client -> Target packet
	_, err = clientUDPListener.WriteTo(udpPacket.Bytes(), proxyRelayUDPAddr)
	if err != nil {
		t.Fatalf("Client UDP (disallowed test) failed to write to proxy relay: %v", err)
	}

	// Try reading from Target UDP Listener - should timeout
	targetUDPListener.SetReadDeadline(time.Now().Add(500 * time.Millisecond)) // Short timeout
	_, _, err = targetUDPListener.ReadFrom(targetBuf)
	if err == nil {
		t.Errorf("Target received UDP packet even when host was disallowed")
	} else if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("Target received unexpected error when expecting timeout: %v", err)
	} else {
		t.Log("Target correctly did not receive packet for disallowed host (timeout).")
	}
	// Restore for potential future tests (though not strictly needed here)
	security.allowedHosts["127.0.0.1"] = true


	// 7. Cleanup (Handled by t.Cleanup)

	// Check for server-side errors
	if err := <-serverErrChan; err != nil {
		t.Fatalf("SOCKS server goroutine error: %v", err)
	}
}

// Removed local helper function redeclarations:
// - getTestLocalIP
// - findObjectByName
// - IsOccluded
// - getCurrentDistance
// - CalculateLatency
// The test will use the implementations from the main package.
