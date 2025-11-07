package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// setupExtendedTestEnv sets up a more comprehensive test environment
// with multiple celestial bodies at different distances
func setupExtendedTestEnv() (func(), map[string]CelestialObject) {
	// Save original objects
	originalCelestialObjects := celestialObjects

	// Create multiple test celestial objects with varying distances/latencies
	testBodies := []CelestialObject{
		{
			Name:   "Sun",
			Type:   "star",
			Radius: 695700, // km
			Mass:   1.989e30, // kg
			// Sun is at the origin, no orbital elements needed
		},
		{
			Name:       "Earth",
			Type:       "planet",
			ParentName: "Sun",
			Radius:     6378.137,
			A:          1.00000261,
			E:          0.01671123,
			I:          -0.00001531,
			L:          100.46457166,
			Mass:       5.972e24,
		},
		{
			Name:       "Mars",
			Type:       "planet",
			ParentName: "Sun",
			Radius:     3396.19,
			A:          1.52366231,
			E:          0.09341233,
			I:          1.85061,
			L:          355.45332,
			Mass:       6.4171e23,
		},
		{
			Name:       "Jupiter",
			Type:       "planet",
			ParentName: "Sun",
			Radius:     71492,
			A:          5.20336301,
			E:          0.04839266,
			I:          1.30530,
			L:          34.40438,
			Mass:       1.8982e27,
		},
		{
			Name:       "Phobos",
			Type:       "moon",
			ParentName: "Mars",
			Radius:     11.2667,
			A:          9376,
			E:          0.0151,
			I:          1.093,
			Mass:       1.0659e16,
		},
		{
			Name:       "Europa",
			Type:       "moon",
			ParentName: "Jupiter",
			Radius:     1560.8,
			A:          671034,
			E:          0.009,
			I:          0.464,
			Mass:       4.8e22,
		},
		{
			Name:       "Voyager 1",
			Type:       "spacecraft",
			ParentName: "Sun", // Prevent warning
			Radius:     10,
			A:          150e9, // Very distant
			E:          0.1,
			I:          0.1,
			Mass:       1000,
		},
	}

	// Override global celestial objects
	celestialObjects = testBodies

	// Create a map for easy lookup in tests
	bodyMap := make(map[string]CelestialObject)
	for _, body := range testBodies {
		bodyMap[body.Name] = body
	}

	// Return cleanup function and the body map
	return func() {
		celestialObjects = originalCelestialObjects
	}, bodyMap
}

// TestSocksTCPConnect tests the basic TCP CONNECT command with various targets
func TestSocksTCPConnect(t *testing.T) {
	cleanup, _ := setupExtendedTestEnv()
	defer cleanup()

	// Enable test mode for controllable latency
	testModeCleanup := setupTestMode()
	defer testModeCleanup()

	// Setup security validator and metrics
	security := NewSecurityValidator()
	metrics := NewTestMetricsCollector()
	
	// Allow localhost and common ports for testing
	security.allowedHosts["127.0.0.1"] = true
	security.allowedHosts["localhost"] = true
	security.allowedPorts["8080"] = true
	security.allowedPorts["1234"] = true

	// Start a mock target TCP server
	targetListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start target TCP server: %v", err)
	}
	defer targetListener.Close()

	targetAddr := targetListener.Addr().(*net.TCPAddr)
	targetPort := strconv.Itoa(targetAddr.Port)
	security.allowedPorts[targetPort] = true

	// Start handling target connections
	targetEchoMsgs := make(chan string, 10) // Collect messages received by target
	go func() {
		for {
			conn, err := targetListener.Accept()
			if err != nil {
				// Check if listener was closed
				if strings.Contains(err.Error(), "use of closed") {
					return
				}
				t.Logf("Target accept error: %v", err)
				continue
			}

			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 1024)
				n, err := c.Read(buf)
				if err != nil && err != io.EOF {
					t.Logf("Target read error: %v", err)
					return
				}
				
				msg := string(buf[:n])
				targetEchoMsgs <- msg
				
				// Echo the message back
				_, err = c.Write(buf[:n])
				if err != nil {
					t.Logf("Target write error: %v", err)
				}
			}(conn)
		}
	}()

	// Start a SOCKS proxy server
	socksListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start SOCKS server: %v", err)
	}
	defer socksListener.Close()

	socksAddr := socksListener.Addr().String()
	
	var serverWg sync.WaitGroup
	serverWg.Add(1)
	
	// Handle one connection
	go func() {
		defer serverWg.Done()
		
		conn, err := socksListener.Accept()
		if err != nil {
			if !strings.Contains(err.Error(), "use of closed") {
				t.Logf("SOCKS accept error: %v", err)
			}
			return
		}
		
		handler := NewSOCKSHandler(conn, security, metrics)
		handler.Handle()
	}()

	// Connect to the SOCKS proxy
	clientConn, err := net.Dial("tcp", socksAddr)
	if err != nil {
		t.Fatalf("Failed to connect to SOCKS proxy: %v", err)
	}
	defer clientConn.Close()

	// SOCKS5 handshake
	// 1. Client greeting
	_, err = clientConn.Write([]byte{SOCKS5_VERSION, 1, SOCKS5_NO_AUTH})
	if err != nil {
		t.Fatalf("Failed to send greeting: %v", err)
	}

	// 2. Read server choice
	choice := make([]byte, 2)
	_, err = io.ReadFull(clientConn, choice)
	if err != nil {
		t.Fatalf("Failed to read server choice: %v", err)
	}

	if choice[0] != SOCKS5_VERSION || choice[1] != SOCKS5_NO_AUTH {
		t.Fatalf("Unexpected server choice: %v", choice)
	}

	// 3. Send CONNECT request to target
	targetIP := net.ParseIP("127.0.0.1").To4()
	targetPortNum := uint16(targetAddr.Port)

	req := []byte{
		SOCKS5_VERSION,      // version
		SOCKS5_CMD_CONNECT,  // command (CONNECT)
		0x00,                // reserved
		SOCKS5_ADDR_IPV4,    // address type (IPv4)
	}
	req = append(req, targetIP...)  // append 4 bytes of IPv4
	portBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(portBytes, targetPortNum)
	req = append(req, portBytes...)  // append 2 bytes of port

	_, err = clientConn.Write(req)
	if err != nil {
		t.Fatalf("Failed to send CONNECT request: %v", err)
	}

	// 4. Read response
	resp := make([]byte, 4)
	_, err = io.ReadFull(clientConn, resp)
	if err != nil {
		t.Fatalf("Failed to read response header: %v", err)
	}

	if resp[0] != SOCKS5_VERSION {
		t.Fatalf("Unexpected response version: 0x%02x", resp[0])
	}

	if resp[1] != SOCKS5_REP_SUCCESS {
		t.Fatalf("Connection failed, response code: 0x%02x", resp[1])
	}

	// Read the bound address and port
	var boundIP net.IP
	switch resp[3] {
	case SOCKS5_ADDR_IPV4:
		boundIPBytes := make([]byte, 4)
		_, err = io.ReadFull(clientConn, boundIPBytes)
		if err != nil {
			t.Fatalf("Failed to read bound IP: %v", err)
		}
		boundIP = net.IP(boundIPBytes)
	case SOCKS5_ADDR_IPV6:
		boundIPBytes := make([]byte, 16)
		_, err = io.ReadFull(clientConn, boundIPBytes)
		if err != nil {
			t.Fatalf("Failed to read bound IP: %v", err)
		}
		boundIP = net.IP(boundIPBytes)
	default:
		t.Fatalf("Unexpected address type in response: 0x%02x", resp[3])
	}

	boundPortBytes := make([]byte, 2)
	_, err = io.ReadFull(clientConn, boundPortBytes)
	if err != nil {
		t.Fatalf("Failed to read bound port: %v", err)
	}
	boundPort := binary.BigEndian.Uint16(boundPortBytes)

	t.Logf("Connection established via SOCKS5, bound to %s:%d", boundIP, boundPort)

	// 5. Send test data through the established connection
	testMessage := "Hello, SOCKS5 TCP world!"
	_, err = clientConn.Write([]byte(testMessage))
	if err != nil {
		t.Fatalf("Failed to send test message: %v", err)
	}

	// Wait for target to receive the message
	var receivedMsg string
	select {
	case receivedMsg = <-targetEchoMsgs:
		if receivedMsg != testMessage {
			t.Fatalf("Target received wrong message: got %q, want %q", receivedMsg, testMessage)
		}
		t.Logf("Target successfully received: %q", receivedMsg)
	case <-time.After(2 * time.Second):
		t.Fatalf("Timeout waiting for target to receive message")
	}

	// 6. Read response data from the established connection
	respBuf := make([]byte, 1024)
	n, err := clientConn.Read(respBuf)
	if err != nil {
		t.Fatalf("Failed to read response from target: %v", err)
	}

	respMsg := string(respBuf[:n])
	if respMsg != testMessage {
		t.Fatalf("Received wrong response: got %q, want %q", respMsg, testMessage)
	}
	
	t.Logf("Client received echo response: %q", respMsg)
}

// TestSocksTCPDomainName tests connecting using domain name resolution instead of IP
func TestSocksTCPDomainName(t *testing.T) {
	cleanup, _ := setupExtendedTestEnv()
	defer cleanup()

	// Enable test mode for controllable latency
	testModeCleanup := setupTestMode()
	defer testModeCleanup()

	// Setup security validator and metrics
	security := NewSecurityValidator()
	metrics := NewTestMetricsCollector()
	
	// Allow test domains
	security.allowedHosts["localhost"] = true
	security.allowedHosts["example.com"] = true
	security.allowedPorts["80"] = true
	security.allowedPorts["443"] = true

	// Start a mock target TCP server
	targetListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start target TCP server: %v", err)
	}
	defer targetListener.Close()

	targetAddr := targetListener.Addr().(*net.TCPAddr)
	targetPort := strconv.Itoa(targetAddr.Port)
	security.allowedPorts[targetPort] = true

	// Start handling target connections
	var targetWg sync.WaitGroup
	targetWg.Add(1)
	go func() {
		defer targetWg.Done()
		conn, err := targetListener.Accept()
		if err != nil {
			if !strings.Contains(err.Error(), "use of closed") {
				t.Logf("Target accept error: %v", err)
			}
			return
		}

		// Just echo what was received
		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil && err != io.EOF {
			t.Logf("Target read error: %v", err)
			conn.Close()
			return
		}
		
		_, err = conn.Write(buf[:n])
		if err != nil {
			t.Logf("Target write error: %v", err)
		}
		conn.Close()
	}()

	// Start a SOCKS proxy server
	socksListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start SOCKS server: %v", err)
	}
	defer socksListener.Close()

	socksAddr := socksListener.Addr().String()
	
	var serverWg sync.WaitGroup
	serverWg.Add(1)
	
	// Handle one connection
	go func() {
		defer serverWg.Done()
		
		conn, err := socksListener.Accept()
		if err != nil {
			if !strings.Contains(err.Error(), "use of closed") {
				t.Logf("SOCKS accept error: %v", err)
			}
			return
		}
		
		handler := NewSOCKSHandler(conn, security, metrics)
		handler.Handle()
	}()

	// Connect to the SOCKS proxy
	clientConn, err := net.Dial("tcp", socksAddr)
	if err != nil {
		t.Fatalf("Failed to connect to SOCKS proxy: %v", err)
	}
	defer clientConn.Close()

	// SOCKS5 handshake
	// 1. Client greeting
	_, err = clientConn.Write([]byte{SOCKS5_VERSION, 1, SOCKS5_NO_AUTH})
	if err != nil {
		t.Fatalf("Failed to send greeting: %v", err)
	}

	// 2. Read server choice
	choice := make([]byte, 2)
	_, err = io.ReadFull(clientConn, choice)
	if err != nil {
		t.Fatalf("Failed to read server choice: %v", err)
	}

	if choice[0] != SOCKS5_VERSION || choice[1] != SOCKS5_NO_AUTH {
		t.Fatalf("Unexpected server choice: %v", choice)
	}

	// 3. Send CONNECT request with domain name
	// We'll use "localhost" as the domain which should resolve to 127.0.0.1
	domain := "localhost"
	targetPortNum := uint16(targetAddr.Port)

	req := []byte{
		SOCKS5_VERSION,      // version
		SOCKS5_CMD_CONNECT,  // command (CONNECT)
		0x00,                // reserved
		SOCKS5_ADDR_DOMAIN,  // address type (Domain)
		byte(len(domain)),   // domain name length
	}
	req = append(req, []byte(domain)...)  // append domain name
	portBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(portBytes, targetPortNum)
	req = append(req, portBytes...)  // append 2 bytes of port

	_, err = clientConn.Write(req)
	if err != nil {
		t.Fatalf("Failed to send CONNECT request: %v", err)
	}

	// 4. Read response
	resp := make([]byte, 4)
	_, err = io.ReadFull(clientConn, resp)
	if err != nil {
		t.Fatalf("Failed to read response header: %v", err)
	}

	if resp[0] != SOCKS5_VERSION {
		t.Fatalf("Unexpected response version: 0x%02x", resp[0])
	}

	if resp[1] != SOCKS5_REP_SUCCESS {
		t.Fatalf("Connection failed, response code: 0x%02x", resp[1])
	}

	// Read the bound address and port
	var boundAddr string
	switch resp[3] {
	case SOCKS5_ADDR_IPV4:
		boundIPBytes := make([]byte, 4)
		_, err = io.ReadFull(clientConn, boundIPBytes)
		if err != nil {
			t.Fatalf("Failed to read bound IP: %v", err)
		}
		boundAddr = net.IP(boundIPBytes).String()
	case SOCKS5_ADDR_IPV6:
		boundIPBytes := make([]byte, 16)
		_, err = io.ReadFull(clientConn, boundIPBytes)
		if err != nil {
			t.Fatalf("Failed to read bound IP: %v", err)
		}
		boundAddr = net.IP(boundIPBytes).String()
	case SOCKS5_ADDR_DOMAIN:
		lenByte := make([]byte, 1)
		_, err = io.ReadFull(clientConn, lenByte)
		if err != nil {
			t.Fatalf("Failed to read domain length: %v", err)
		}
		domainLen := int(lenByte[0])
		domainBytes := make([]byte, domainLen)
		_, err = io.ReadFull(clientConn, domainBytes)
		if err != nil {
			t.Fatalf("Failed to read domain: %v", err)
		}
		boundAddr = string(domainBytes)
	default:
		t.Fatalf("Unexpected address type in response: 0x%02x", resp[3])
	}

	boundPortBytes := make([]byte, 2)
	_, err = io.ReadFull(clientConn, boundPortBytes)
	if err != nil {
		t.Fatalf("Failed to read bound port: %v", err)
	}
	boundPort := binary.BigEndian.Uint16(boundPortBytes)

	t.Logf("Connection established via SOCKS5 using domain name, bound to %s:%d", boundAddr, boundPort)

	// 5. Send test data through the established connection
	testMessage := "Hello from domain resolution test!"
	_, err = clientConn.Write([]byte(testMessage))
	if err != nil {
		t.Fatalf("Failed to send test message: %v", err)
	}

	// 6. Read response data from the established connection
	respBuf := make([]byte, 1024)
	n, err := clientConn.Read(respBuf)
	if err != nil {
		t.Fatalf("Failed to read response from target: %v", err)
	}

	respMsg := string(respBuf[:n])
	if respMsg != testMessage {
		t.Fatalf("Received wrong response: got %q, want %q", respMsg, testMessage)
	}
	
	t.Logf("Client received correct echo response: %q", respMsg)

	// Close connections and wait for goroutines
	clientConn.Close()
	targetListener.Close()
	socksListener.Close()

	// Wait for the target and server goroutines to finish
	targetWg.Wait()
	serverWg.Wait()
}

// TestSocksTCPErrorHandling tests various error conditions with TCP connections
func TestSocksTCPErrorHandling(t *testing.T) {
	cleanup, _ := setupExtendedTestEnv()
	defer cleanup()

	// Enable test mode for controllable latency
	testModeCleanup := setupTestMode()
	defer testModeCleanup()

	// Setup security validator and metrics
	security := NewSecurityValidator()
	metrics := NewTestMetricsCollector()
	
	// Allow localhost for testing
	security.allowedHosts["127.0.0.1"] = true
	security.allowedHosts["localhost"] = true

	// Test cases for error conditions
	testCases := []struct {
		name        string
		setupFunc   func(t *testing.T) net.Conn // Setup function that creates a SOCKS client connection
		validateFunc func(t *testing.T, conn net.Conn) // Validation function to verify error handling
	}{
		{
			name: "Connection to disallowed host",
			setupFunc: func(t *testing.T) net.Conn {
				// Start a SOCKS proxy server
				socksListener, err := net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					t.Fatalf("Failed to start SOCKS server: %v", err)
				}
				t.Cleanup(func() { socksListener.Close() })

				socksAddr := socksListener.Addr().String()
				
				// Handle one connection
				go func() {
					conn, err := socksListener.Accept()
					if err != nil {
						if !strings.Contains(err.Error(), "use of closed") {
							t.Logf("SOCKS accept error: %v", err)
						}
						return
					}
					
					handler := NewSOCKSHandler(conn, security, metrics)
					handler.Handle()
				}()

				// Connect to the SOCKS proxy
				clientConn, err := net.Dial("tcp", socksAddr)
				if err != nil {
					t.Fatalf("Failed to connect to SOCKS proxy: %v", err)
				}

				// SOCKS5 handshake
				// 1. Client greeting
				_, err = clientConn.Write([]byte{SOCKS5_VERSION, 1, SOCKS5_NO_AUTH})
				if err != nil {
					t.Fatalf("Failed to send greeting: %v", err)
				}

				// 2. Read server choice
				choice := make([]byte, 2)
				_, err = io.ReadFull(clientConn, choice)
				if err != nil {
					t.Fatalf("Failed to read server choice: %v", err)
				}

				return clientConn
			},
			validateFunc: func(t *testing.T, conn net.Conn) {
				defer conn.Close()
				
				// Send CONNECT request to disallowed host (8.8.8.8)
				disallowedIP := net.ParseIP("8.8.8.8").To4()
				targetPort := uint16(80)

				req := []byte{
					SOCKS5_VERSION,      // version
					SOCKS5_CMD_CONNECT,  // command (CONNECT)
					0x00,                // reserved
					SOCKS5_ADDR_IPV4,    // address type (IPv4)
				}
				req = append(req, disallowedIP...)  // append 4 bytes of IPv4
				portBytes := make([]byte, 2)
				binary.BigEndian.PutUint16(portBytes, targetPort)
				req = append(req, portBytes...)  // append 2 bytes of port

				_, err := conn.Write(req)
				if err != nil {
					t.Fatalf("Failed to send CONNECT request: %v", err)
				}

				// Read response - should be an error
				resp := make([]byte, 4)
				_, err = io.ReadFull(conn, resp)
				if err != nil {
					t.Fatalf("Failed to read response header: %v", err)
				}

				if resp[0] != SOCKS5_VERSION {
					t.Fatalf("Unexpected response version: 0x%02x", resp[0])
				}

				// Should get a "connection not allowed" error
				// Server might return either CONN_NOT_ALLOWED or HOST_UNREACHABLE
				if resp[1] != SOCKS5_REP_CONN_NOT_ALLOWED && resp[1] != SOCKS5_REP_HOST_UNREACHABLE {
					t.Fatalf("Expected CONN_NOT_ALLOWED (0x02) or HOST_UNREACHABLE (0x04), got: 0x%02x", resp[1])
				}

				t.Logf("Correctly received CONN_NOT_ALLOWED for disallowed host")
			},
		},
		{
			name: "Connection to non-existent port",
			setupFunc: func(t *testing.T) net.Conn {
				// Start a SOCKS proxy server
				socksListener, err := net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					t.Fatalf("Failed to start SOCKS server: %v", err)
				}
				t.Cleanup(func() { socksListener.Close() })

				socksAddr := socksListener.Addr().String()
				
				// Handle one connection
				go func() {
					conn, err := socksListener.Accept()
					if err != nil {
						if !strings.Contains(err.Error(), "use of closed") {
							t.Logf("SOCKS accept error: %v", err)
						}
						return
					}
					
					handler := NewSOCKSHandler(conn, security, metrics)
					handler.Handle()
				}()

				// Connect to the SOCKS proxy
				clientConn, err := net.Dial("tcp", socksAddr)
				if err != nil {
					t.Fatalf("Failed to connect to SOCKS proxy: %v", err)
				}

				// SOCKS5 handshake
				// 1. Client greeting
				_, err = clientConn.Write([]byte{SOCKS5_VERSION, 1, SOCKS5_NO_AUTH})
				if err != nil {
					t.Fatalf("Failed to send greeting: %v", err)
				}

				// 2. Read server choice
				choice := make([]byte, 2)
				_, err = io.ReadFull(clientConn, choice)
				if err != nil {
					t.Fatalf("Failed to read server choice: %v", err)
				}

				return clientConn
			},
			validateFunc: func(t *testing.T, conn net.Conn) {
				defer conn.Close()
				
				// Allow port 12345 (likely unused)
				security.allowedPorts["12345"] = true
				
				// Send CONNECT request to allowed host but non-existent port
				targetIP := net.ParseIP("127.0.0.1").To4()
				targetPort := uint16(12345) // Most likely nothing listening on this port

				req := []byte{
					SOCKS5_VERSION,      // version
					SOCKS5_CMD_CONNECT,  // command (CONNECT)
					0x00,                // reserved
					SOCKS5_ADDR_IPV4,    // address type (IPv4)
				}
				req = append(req, targetIP...)  // append 4 bytes of IPv4
				portBytes := make([]byte, 2)
				binary.BigEndian.PutUint16(portBytes, targetPort)
				req = append(req, portBytes...)  // append 2 bytes of port

				_, err := conn.Write(req)
				if err != nil {
					t.Fatalf("Failed to send CONNECT request: %v", err)
				}

				// Read response - should be a connection refused error
				resp := make([]byte, 4)
				_, err = io.ReadFull(conn, resp)
				if err != nil {
					t.Fatalf("Failed to read response header: %v", err)
				}

				if resp[0] != SOCKS5_VERSION {
					t.Fatalf("Unexpected response version: 0x%02x", resp[0])
				}

				// Might get CONN_REFUSED or HOST_UNREACHABLE depending on OS/network
				if resp[1] != SOCKS5_REP_CONN_REFUSED && resp[1] != SOCKS5_REP_HOST_UNREACHABLE {
					t.Fatalf("Expected connection error, got: 0x%02x", resp[1])
				}

				t.Logf("Correctly received error (0x%02x) for non-existent port", resp[1])
			},
		},
		{
			name: "Invalid command",
			setupFunc: func(t *testing.T) net.Conn {
				// Start a SOCKS proxy server
				socksListener, err := net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					t.Fatalf("Failed to start SOCKS server: %v", err)
				}
				t.Cleanup(func() { socksListener.Close() })

				socksAddr := socksListener.Addr().String()
				
				// Handle one connection
				go func() {
					conn, err := socksListener.Accept()
					if err != nil {
						if !strings.Contains(err.Error(), "use of closed") {
							t.Logf("SOCKS accept error: %v", err)
						}
						return
					}
					
					handler := NewSOCKSHandler(conn, security, metrics)
					handler.Handle()
				}()

				// Connect to the SOCKS proxy
				clientConn, err := net.Dial("tcp", socksAddr)
				if err != nil {
					t.Fatalf("Failed to connect to SOCKS proxy: %v", err)
				}

				// SOCKS5 handshake
				// 1. Client greeting
				_, err = clientConn.Write([]byte{SOCKS5_VERSION, 1, SOCKS5_NO_AUTH})
				if err != nil {
					t.Fatalf("Failed to send greeting: %v", err)
				}

				// 2. Read server choice
				choice := make([]byte, 2)
				_, err = io.ReadFull(clientConn, choice)
				if err != nil {
					t.Fatalf("Failed to read server choice: %v", err)
				}

				return clientConn
			},
			validateFunc: func(t *testing.T, conn net.Conn) {
				defer conn.Close()
				
				// Send an invalid command (0x04)
				targetIP := net.ParseIP("127.0.0.1").To4()
				targetPort := uint16(80)

				req := []byte{
					SOCKS5_VERSION, // version
					0x04,           // invalid command (not CONNECT, BIND, or UDP)
					0x00,           // reserved
					SOCKS5_ADDR_IPV4,    // address type (IPv4)
				}
				req = append(req, targetIP...)  // append 4 bytes of IPv4
				portBytes := make([]byte, 2)
				binary.BigEndian.PutUint16(portBytes, targetPort)
				req = append(req, portBytes...)  // append 2 bytes of port

				_, err := conn.Write(req)
				if err != nil {
					t.Fatalf("Failed to send request with invalid command: %v", err)
				}

				// Read response - should be a command not supported error
				resp := make([]byte, 4)
				_, err = io.ReadFull(conn, resp)
				if err != nil {
					t.Fatalf("Failed to read response header: %v", err)
				}

				if resp[0] != SOCKS5_VERSION {
					t.Fatalf("Unexpected response version: 0x%02x", resp[0])
				}

				if resp[1] != SOCKS5_REP_CMD_NOT_SUPPORTED {
					t.Fatalf("Expected CMD_NOT_SUPPORTED (0x07), got: 0x%02x", resp[1])
				}

				t.Logf("Correctly received CMD_NOT_SUPPORTED for invalid command")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			conn := tc.setupFunc(t)
			tc.validateFunc(t, conn)
		})
	}
}

// TestSocksUDPReliability tests UDP reliability with packet loss and reordering
func TestSocksUDPReliability(t *testing.T) {
	cleanup, _ := setupExtendedTestEnv()
	defer cleanup()

	// Enable test mode for low latency
	testModeCleanup := setupTestMode()
	defer testModeCleanup()

	// Setup test-specific validator and metrics
	security := NewSecurityValidator()
	metrics := NewTestMetricsCollector()

	// Allow loopback host for testing
	security.allowedHosts["127.0.0.1"] = true
	security.allowedHosts["localhost"] = true

	// Mock SOCKS Server (TCP Listener)
	tcpListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start TCP listener: %v", err)
	}
	defer tcpListener.Close()
	proxyTCPAddr := tcpListener.Addr().String()

	// Mock Target Server (UDP Listener)
	targetUDPListener, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start target UDP listener: %v", err)
	}
	defer targetUDPListener.Close()
	targetUDPAddr := targetUDPListener.LocalAddr().(*net.UDPAddr)

	// Allow the specific dynamic target port
	targetPortStr := strconv.Itoa(targetUDPAddr.Port)
	security.allowedPorts[targetPortStr] = true

	// Run Mock SOCKS Server
	go func() {
		conn, err := tcpListener.Accept()
		if err != nil {
			if !strings.Contains(err.Error(), "use of closed") {
				t.Logf("TCP accept error: %v", err)
			}
			return
		}
		defer conn.Close()

		handler := NewSOCKSHandler(conn, security, metrics)
		handler.Handle()
	}()

	// Connect to SOCKS server and perform UDP ASSOCIATE
	clientTCPConn, err := net.Dial("tcp", proxyTCPAddr)
	if err != nil {
		t.Fatalf("Failed to dial proxy TCP: %v", err)
	}
	defer clientTCPConn.Close()

	// SOCKS5 Handshake - Auth Negotiation
	_, err = clientTCPConn.Write([]byte{SOCKS5_VERSION, 1, SOCKS5_NO_AUTH})
	if err != nil {
		t.Fatalf("Failed to send greeting: %v", err)
	}

	choice := make([]byte, 2)
	_, err = io.ReadFull(clientTCPConn, choice)
	if err != nil {
		t.Fatalf("Failed to read auth choice: %v", err)
	}

	// UDP ASSOCIATE request
	udpRequest := []byte{
		SOCKS5_VERSION, SOCKS5_CMD_UDP_ASSOCIATE, 0x00, 
		SOCKS5_ADDR_IPV4, 0, 0, 0, 0, 0, 0,
	}
	_, err = clientTCPConn.Write(udpRequest)
	if err != nil {
		t.Fatalf("Failed to send UDP associate: %v", err)
	}

	// Read UDP ASSOCIATE reply
	replyHeader := make([]byte, 4)
	_, err = io.ReadFull(clientTCPConn, replyHeader)
	if err != nil {
		t.Fatalf("Failed to read UDP reply header: %v", err)
	}

	if replyHeader[1] != SOCKS5_REP_SUCCESS {
		t.Fatalf("UDP ASSOCIATE failed with code: 0x%02x", replyHeader[1])
	}

	// Read bound address
	var boundAddr []byte
	switch replyHeader[3] {
	case SOCKS5_ADDR_IPV4:
		boundAddr = make([]byte, 4)
	case SOCKS5_ADDR_IPV6:
		boundAddr = make([]byte, 16)
	default:
		t.Fatalf("Unexpected address type: 0x%02x", replyHeader[3])
	}

	_, err = io.ReadFull(clientTCPConn, boundAddr)
	if err != nil {
		t.Fatalf("Failed to read bound address: %v", err)
	}

	// Read bound port
	boundPortBytes := make([]byte, 2)
	_, err = io.ReadFull(clientTCPConn, boundPortBytes)
	if err != nil {
		t.Fatalf("Failed to read bound port: %v", err)
	}
	boundPort := binary.BigEndian.Uint16(boundPortBytes)

	// Create a UDP connection for client
	clientUDPConn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create client UDP: %v", err)
	}
	defer clientUDPConn.Close()

	// Get the proxy UDP relay address
	proxyUDPAddr := &net.UDPAddr{
		IP:   net.IP(boundAddr),
		Port: int(boundPort),
	}
	t.Logf("Proxy UDP relay at %s", proxyUDPAddr.String())

	// Setup target to record received packets
	receivedPackets := make(chan []byte, 100)
	packetMemo := make(map[string]bool) // For deduplication

	go func() {
		buffer := make([]byte, 2048)
		for {
			n, addr, err := targetUDPListener.ReadFrom(buffer)
			if err != nil {
				if !strings.Contains(err.Error(), "use of closed") {
					t.Logf("Target read error: %v", err)
				}
				return
			}

			// Process the packet
			packet := make([]byte, n)
			copy(packet, buffer[:n])
			
			// Check for duplicate (using content as key)
			packetKey := string(packet)
			if !packetMemo[packetKey] {
				packetMemo[packetKey] = true
				receivedPackets <- packet
			}

			// Echo back the packet
			_, err = targetUDPListener.WriteTo(packet, addr)
			if err != nil {
				t.Logf("Target echo error: %v", err)
			}
		}
	}()

	// Test data - multiple packets of different sizes
	testData := []struct {
		id   int
		data []byte
	}{
		{1, []byte("Small packet #1")},
		{2, bytes.Repeat([]byte("Medium size packet #2 with some repetition "), 10)},
		{3, bytes.Repeat([]byte("A"), 1000)}, // ~1KB packet
		{4, []byte("Final small packet #4")},
	}

	// Send all packets from client to proxy
	for _, data := range testData {
		// Build the SOCKS UDP datagram
		var datagram bytes.Buffer
		datagram.Write([]byte{0, 0, 0}) // RSV + FRAG
		datagram.WriteByte(SOCKS5_ADDR_IPV4) // ATYP
		datagram.Write(targetUDPAddr.IP.To4()) // DST.ADDR
		
		portBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(portBytes, uint16(targetUDPAddr.Port))
		datagram.Write(portBytes) // DST.PORT
		
		datagram.Write(data.data) // DATA

		// Send to proxy relay
		_, err = clientUDPConn.WriteTo(datagram.Bytes(), proxyUDPAddr)
		if err != nil {
			t.Fatalf("Failed to send UDP packet #%d: %v", data.id, err)
		}
		t.Logf("Sent packet #%d, size: %d bytes", data.id, len(data.data))
	}

	// Verify all packets were received by the target
	receivedCount := 0
	timeout := time.After(5 * time.Second)

	// We need to receive all 4 packets
expectedPackets:
	for receivedCount < len(testData) {
		select {
		case packet := <-receivedPackets:
			receivedCount++
			t.Logf("Target received packet #%d, size: %d bytes", receivedCount, len(packet))
		case <-timeout:
			t.Fatalf("Timeout waiting for packets. Received %d/%d", receivedCount, len(testData))
			break expectedPackets
		}
	}

	// Verify client receives replies
	clientReceivedCount := 0
	_ = time.After(5 * time.Second) // Unused but kept for reference
	clientBuf := make([]byte, 2048)

	// Set a read deadline on the client UDP connection
	if err := clientUDPConn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("Failed to set read deadline: %v", err)
	}

	// We should receive replies for all 4 packets
	for clientReceivedCount < len(testData) {
		n, _, err := clientUDPConn.ReadFrom(clientBuf)
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				t.Fatalf("Timeout receiving replies. Got %d/%d", clientReceivedCount, len(testData))
			}
			t.Fatalf("Error reading from UDP: %v", err)
		}

		// Process the SOCKS UDP response format
		if n < 10 { // Min UDP SOCKS header size (4 + IPv4 + port)
			t.Fatalf("Received UDP response too small: %d bytes", n)
		}

		// Extract payload from UDP SOCKS header
		payloadStart := 0
		atyp := clientBuf[3]
		switch atyp {
		case SOCKS5_ADDR_IPV4:
			payloadStart = 10 // 4 + 4(IPv4) + 2(port)
		case SOCKS5_ADDR_IPV6:
			payloadStart = 22 // 4 + 16(IPv6) + 2(port)
		case SOCKS5_ADDR_DOMAIN:
			domainLen := int(clientBuf[4])
			payloadStart = 7 + domainLen // 4 + 1(len) + domainLen + 2(port)
		default:
			t.Fatalf("Invalid address type in response: 0x%02x", atyp)
		}

		if n <= payloadStart {
			t.Fatalf("No payload in response. Total size: %d, header size: %d", n, payloadStart)
		}

		payload := clientBuf[payloadStart:n]
		clientReceivedCount++
		t.Logf("Client received reply #%d, payload size: %d bytes", clientReceivedCount, len(payload))
	}

	t.Logf("Successfully received all %d packets and replies", len(testData))
}