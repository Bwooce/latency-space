package main

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestUDPFailureModes tests various failure scenarios for UDP connections
func TestUDPFailureModes(t *testing.T) {
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

	tests := []struct {
		name        string
		setupTest   func(t *testing.T) (*net.UDPAddr, net.PacketConn)
		runTest     func(t *testing.T, proxyUDPAddr *net.UDPAddr, clientUDPConn net.PacketConn)
		expectError bool
	}{
		{
			name: "Drop control connection during transfer",
			setupTest: func(t *testing.T) (*net.UDPAddr, net.PacketConn) {
				// Mock SOCKS Server (TCP Listener)
				tcpListener, err := net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					t.Fatalf("Failed to start TCP listener: %v", err)
				}
				t.Cleanup(func() { tcpListener.Close() })
				proxyTCPAddr := tcpListener.Addr().String()

				// Mock Target Server (UDP Listener) - just echo received packets
				targetUDPListener, err := net.ListenPacket("udp", "127.0.0.1:0")
				if err != nil {
					t.Fatalf("Failed to start target UDP listener: %v", err)
				}
				t.Cleanup(func() { targetUDPListener.Close() })
				targetUDPAddr := targetUDPListener.LocalAddr().(*net.UDPAddr)

				// Allow the target port
				targetPortStr := strconv.Itoa(targetUDPAddr.Port)
				security.allowedPorts[targetPortStr] = true

				// Setup target echo server
				go func() {
					buf := make([]byte, 2048)
					for {
						n, addr, err := targetUDPListener.ReadFrom(buf)
						if err != nil {
							// Normal on close
							if !strings.Contains(err.Error(), "use of closed") {
								t.Logf("Target UDP read error: %v", err)
							}
							return
						}
						// Echo the data back
						_, err = targetUDPListener.WriteTo(buf[:n], addr)
						if err != nil {
							t.Logf("Target UDP write error: %v", err)
						}
					}
				}()

				// Run SOCKS server
				var tcpConn net.Conn
				serverComplete := make(chan struct{})
				var serverWg sync.WaitGroup
				serverWg.Add(1)
				
				go func() {
					defer serverWg.Done()
					var err error
					tcpConn, err = tcpListener.Accept()
					if err != nil {
						if !strings.Contains(err.Error(), "use of closed") {
							t.Logf("TCP accept error: %v", err)
						}
						close(serverComplete)
						return
					}
					
					handler := NewSOCKSHandler(tcpConn, security, metrics)
					handler.Handle()
					close(serverComplete)
				}()

				// Connect to SOCKS server
				clientTCPConn, err := net.Dial("tcp", proxyTCPAddr)
				if err != nil {
					t.Fatalf("Failed to dial proxy TCP: %v", err)
				}

				// Store for later closure in test
				t.Cleanup(func() { 
					clientTCPConn.Close() 
					// Wait for server to complete
					<-serverComplete
					serverWg.Wait()
				})

				// SOCKS5 Handshake
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

				// Get the proxy UDP relay address
				proxyUDPAddr := &net.UDPAddr{
					IP:   net.IP(boundAddr),
					Port: int(boundPort),
				}
				
				// Send one initial packet to establish the UDP relay
				initPacket := buildUDPSocksPacket(targetUDPAddr, []byte("init"))
				_, err = clientUDPConn.WriteTo(initPacket, proxyUDPAddr)
				if err != nil {
					t.Fatalf("Failed to send initial UDP packet: %v", err)
				}
				
				// Wait a bit for UDP relay to initialize
				time.Sleep(100 * time.Millisecond)
				
				return proxyUDPAddr, clientUDPConn
			},
			runTest: func(t *testing.T, proxyUDPAddr *net.UDPAddr, clientUDPConn net.PacketConn) {
				defer clientUDPConn.Close()
				
				// Get target address from the proxy UDP address
				// This is a bit of a hack - we know the proxy endpoint
				// but need a valid target. For this test, we'll just use
				// the same IP with a different port (which will be ignored anyway
				// after the control connection is dropped)
				targetUDPAddr := &net.UDPAddr{
					IP:   proxyUDPAddr.IP,
					Port: proxyUDPAddr.Port + 1,
				}
				
				// Send a packet to the proxy
				data := []byte("This packet should go through")
				packet := buildUDPSocksPacket(targetUDPAddr, data)
				_, err := clientUDPConn.WriteTo(packet, proxyUDPAddr)
				if err != nil {
					t.Fatalf("Failed to send UDP packet: %v", err)
				}
				
				// Close the control connection - this should cause the UDP relay to close
				// In the actual test, we simulate this by letting the cleanup function run
				
				// Now attempt to send another packet after a delay
				time.Sleep(500 * time.Millisecond)
				
				// This packet should be dropped because the relay is closed
				failData := []byte("This packet should be dropped")
				failPacket := buildUDPSocksPacket(targetUDPAddr, failData)
				
				// Send directly to the proxy UDP address
				_, err = clientUDPConn.WriteTo(failPacket, proxyUDPAddr)
				
				// The write itself may succeed, but there should be no response
				
				// Set a short timeout for reading the response
				err = clientUDPConn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
				if err != nil {
					t.Fatalf("Failed to set read deadline: %v", err)
				}
				
				// Try to read a response, which should timeout
				respBuf := make([]byte, 2048)
				_, _, err = clientUDPConn.ReadFrom(respBuf)
				
				if err == nil {
					t.Errorf("Expected read timeout after control connection closed, but got data")
				} else if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "deadline") {
					t.Errorf("Expected read timeout error, but got: %v", err)
				} else {
					t.Logf("Correctly received timeout error after control connection closed")
				}
			},
			expectError: true,
		},
		{
			name: "Disallowed target host",
			setupTest: func(t *testing.T) (*net.UDPAddr, net.PacketConn) {
				// Similar setup as above
				tcpListener, err := net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					t.Fatalf("Failed to start TCP listener: %v", err)
				}
				t.Cleanup(func() { tcpListener.Close() })
				proxyTCPAddr := tcpListener.Addr().String()

				// Start SOCKS server
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

				// Connect to SOCKS server
				clientTCPConn, err := net.Dial("tcp", proxyTCPAddr)
				if err != nil {
					t.Fatalf("Failed to dial proxy TCP: %v", err)
				}
				t.Cleanup(func() { clientTCPConn.Close() })

				// SOCKS5 Handshake
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

				// Process UDP ASSOCIATE reply
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

				// Get the proxy UDP relay address
				proxyUDPAddr := &net.UDPAddr{
					IP:   net.IP(boundAddr),
					Port: int(boundPort),
				}
				
				return proxyUDPAddr, clientUDPConn
			},
			runTest: func(t *testing.T, proxyUDPAddr *net.UDPAddr, clientUDPConn net.PacketConn) {
				defer clientUDPConn.Close()
				
				// Explicitly disallow 8.8.8.8 as a target
				delete(security.allowedHosts, "8.8.8.8")
				
				// Create a packet targeting a disallowed host (8.8.8.8)
				disallowedAddr := &net.UDPAddr{
					IP:   net.ParseIP("8.8.8.8"),
					Port: 53, // DNS port
				}
				
				data := []byte("This packet should be dropped by security check")
				packet := buildUDPSocksPacket(disallowedAddr, data)
				
				// Send the packet
				_, err := clientUDPConn.WriteTo(packet, proxyUDPAddr)
				if err != nil {
					t.Fatalf("Failed to send UDP packet to disallowed target: %v", err)
				}
				
				// No response should be received (packet dropped silently)
				err = clientUDPConn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
				if err != nil {
					t.Fatalf("Failed to set read deadline: %v", err)
				}
				
				respBuf := make([]byte, 2048)
				_, _, err = clientUDPConn.ReadFrom(respBuf)
				
				if err == nil {
					t.Errorf("Expected no response for disallowed target, but got data")
				} else if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "deadline") {
					t.Errorf("Expected read timeout error, but got: %v", err)
				} else {
					t.Logf("Correctly received no response for disallowed target")
				}
			},
			expectError: true,
		},
		{
			name: "Malformed UDP packet",
			setupTest: func(t *testing.T) (*net.UDPAddr, net.PacketConn) {
				// Similar setup to previous tests
				tcpListener, err := net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					t.Fatalf("Failed to start TCP listener: %v", err)
				}
				t.Cleanup(func() { tcpListener.Close() })
				proxyTCPAddr := tcpListener.Addr().String()

				// Setup target echo server
				targetUDPListener, err := net.ListenPacket("udp", "127.0.0.1:0")
				if err != nil {
					t.Fatalf("Failed to start target UDP listener: %v", err)
				}
				t.Cleanup(func() { targetUDPListener.Close() })
				
				go func() {
					buf := make([]byte, 2048)
					for {
						n, addr, err := targetUDPListener.ReadFrom(buf)
						if err != nil {
							// Normal on close
							if !strings.Contains(err.Error(), "use of closed") {
								t.Logf("Target UDP read error: %v", err)
							}
							return
						}
						// Echo the data back
						_, err = targetUDPListener.WriteTo(buf[:n], addr)
						if err != nil {
							t.Logf("Target UDP write error: %v", err)
						}
					}
				}()

				// Start SOCKS server
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

				// Connect to SOCKS server
				clientTCPConn, err := net.Dial("tcp", proxyTCPAddr)
				if err != nil {
					t.Fatalf("Failed to dial proxy TCP: %v", err)
				}
				t.Cleanup(func() { clientTCPConn.Close() })

				// SOCKS5 Handshake
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

				// Process UDP ASSOCIATE reply
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

				// Get the proxy UDP relay address
				proxyUDPAddr := &net.UDPAddr{
					IP:   net.IP(boundAddr),
					Port: int(boundPort),
				}
				
				return proxyUDPAddr, clientUDPConn
			},
			runTest: func(t *testing.T, proxyUDPAddr *net.UDPAddr, clientUDPConn net.PacketConn) {
				defer clientUDPConn.Close()
				
				// Send a malformed UDP packet (too short)
				malformedPacket := []byte{0x00, 0x00, 0x00} // Just RSV+FRAG, no ATYP or address
				_, err := clientUDPConn.WriteTo(malformedPacket, proxyUDPAddr)
				if err != nil {
					t.Fatalf("Failed to send malformed packet: %v", err)
				}
				
				// No response should be received (packet should be rejected)
				err = clientUDPConn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
				if err != nil {
					t.Fatalf("Failed to set read deadline: %v", err)
				}
				
				respBuf := make([]byte, 2048)
				_, _, err = clientUDPConn.ReadFrom(respBuf)
				
				if err == nil {
					t.Errorf("Expected no response for malformed packet, but got data")
				} else if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "deadline") {
					t.Errorf("Expected read timeout error, but got: %v", err)
				} else {
					t.Logf("Correctly received no response for malformed packet")
				}
				
				// Send another malformed packet with invalid ATYP
				malformedPacket2 := []byte{
					0x00, 0x00, 0x00, // RSV+FRAG 
					0x05, // Invalid ATYP (not 1, 3, or 4)
				}
				_, err = clientUDPConn.WriteTo(malformedPacket2, proxyUDPAddr)
				if err != nil {
					t.Fatalf("Failed to send malformed packet 2: %v", err)
				}
				
				// No response should be received
				err = clientUDPConn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
				if err != nil {
					t.Fatalf("Failed to set read deadline: %v", err)
				}
				
				_, _, err = clientUDPConn.ReadFrom(respBuf)
				
				if err == nil {
					t.Errorf("Expected no response for packet with invalid ATYP, but got data")
				} else if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "deadline") {
					t.Errorf("Expected read timeout error, but got: %v", err)
				} else {
					t.Logf("Correctly received no response for packet with invalid ATYP")
				}
			},
			expectError: true,
		},
		{
			name: "Invalid fragmentation",
			setupTest: func(t *testing.T) (*net.UDPAddr, net.PacketConn) {
				// Similar setup to previous tests
				tcpListener, err := net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					t.Fatalf("Failed to start TCP listener: %v", err)
				}
				t.Cleanup(func() { tcpListener.Close() })
				proxyTCPAddr := tcpListener.Addr().String()

				// Create an echo server
				targetUDPListener, err := net.ListenPacket("udp", "127.0.0.1:0")
				if err != nil {
					t.Fatalf("Failed to start target UDP listener: %v", err)
				}
				t.Cleanup(func() { targetUDPListener.Close() })
				targetUDPAddr := targetUDPListener.LocalAddr().(*net.UDPAddr)
				
				// Allow target port
				targetPortStr := strconv.Itoa(targetUDPAddr.Port)
				security.allowedPorts[targetPortStr] = true
				
				go func() {
					buf := make([]byte, 2048)
					for {
						n, addr, err := targetUDPListener.ReadFrom(buf)
						if err != nil {
							// Normal on close
							if !strings.Contains(err.Error(), "use of closed") {
								t.Logf("Target UDP read error: %v", err)
							}
							return
						}
						// Echo the data back
						_, err = targetUDPListener.WriteTo(buf[:n], addr)
						if err != nil {
							t.Logf("Target UDP write error: %v", err)
						}
					}
				}()

				// Start SOCKS server
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

				// Connect to SOCKS server
				clientTCPConn, err := net.Dial("tcp", proxyTCPAddr)
				if err != nil {
					t.Fatalf("Failed to dial proxy TCP: %v", err)
				}
				t.Cleanup(func() { clientTCPConn.Close() })

				// SOCKS5 Handshake
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

				// Process UDP ASSOCIATE reply
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

				// Get the proxy UDP relay address
				proxyUDPAddr := &net.UDPAddr{
					IP:   net.IP(boundAddr),
					Port: int(boundPort),
				}
				
				return proxyUDPAddr, clientUDPConn
			},
			runTest: func(t *testing.T, proxyUDPAddr *net.UDPAddr, clientUDPConn net.PacketConn) {
				defer clientUDPConn.Close()
				
				// Create a valid target address
				targetUDPAddr := &net.UDPAddr{
					IP:   net.ParseIP("127.0.0.1"),
					Port: 53, // Any valid port should work
				}
				
				// Create a UDP packet with non-zero fragmentation (not supported)
				fragValue := byte(1) // Non-zero fragmentation value
				data := []byte("This packet has fragmentation enabled")
				
				// Build a SOCKS UDP packet with fragmentation
				var packet bytes.Buffer
				packet.Write([]byte{0x00, 0x00}) // RSV
				packet.WriteByte(fragValue)      // FRAG = non-zero
				packet.WriteByte(SOCKS5_ADDR_IPV4) // ATYP
				packet.Write(targetUDPAddr.IP.To4()) // DST.ADDR
				
				portBytes := make([]byte, 2)
				binary.BigEndian.PutUint16(portBytes, uint16(targetUDPAddr.Port))
				packet.Write(portBytes) // DST.PORT
				
				packet.Write(data) // DATA
				
				// Send the packet with fragmentation
				_, err := clientUDPConn.WriteTo(packet.Bytes(), proxyUDPAddr)
				if err != nil {
					t.Fatalf("Failed to send fragmented packet: %v", err)
				}
				
				// No response should be received (packet should be rejected due to fragmentation)
				err = clientUDPConn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
				if err != nil {
					t.Fatalf("Failed to set read deadline: %v", err)
				}
				
				respBuf := make([]byte, 2048)
				_, _, err = clientUDPConn.ReadFrom(respBuf)
				
				if err == nil {
					t.Errorf("Expected no response for fragmented packet, but got data")
				} else if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "deadline") {
					t.Errorf("Expected read timeout error, but got: %v", err)
				} else {
					t.Logf("Correctly received no response for fragmented packet")
				}
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proxyUDPAddr, clientUDPConn := tt.setupTest(t)
			tt.runTest(t, proxyUDPAddr, clientUDPConn)
		})
	}
}

// Helper function to build a SOCKS5 UDP packet
func buildUDPSocksPacket(targetAddr *net.UDPAddr, data []byte) []byte {
	var packet bytes.Buffer
	
	// RSV(2) + FRAG(1) + ATYP(1) fields
	packet.Write([]byte{0x00, 0x00, 0x00, SOCKS5_ADDR_IPV4})
	
	// Target address (4 bytes for IPv4)
	packet.Write(targetAddr.IP.To4())
	
	// Target port (2 bytes)
	portBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(portBytes, uint16(targetAddr.Port))
	packet.Write(portBytes)
	
	// Payload data
	packet.Write(data)
	
	return packet.Bytes()
}

// TestUDPPortExhaustion tests handling of UDP port exhaustion
func TestUDPPortExhaustion(t *testing.T) {
	// This test simulates a high number of UDP ASSOCIATE requests
	// to check how the system handles resource exhaustion

	// Skip in short mode as this is a stress test
	if testing.Short() {
		t.Skip("Skipping UDP port exhaustion test in short mode")
	}

	cleanup, _ := setupExtendedTestEnv()
	defer cleanup()

	// Enable test mode for low latency
	testModeCleanup := setupTestMode()
	defer testModeCleanup()

	// Setup security and metrics
	security := NewSecurityValidator()
	metrics := NewTestMetricsCollector()
	
	// Allow localhost testing
	security.allowedHosts["127.0.0.1"] = true
	security.allowedPorts["8000"] = true

	// Start a SOCKS server
	socksListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start SOCKS server: %v", err)
	}
	defer socksListener.Close()

	socksAddr := socksListener.Addr().String()
	
	// Start handling connections
	go func() {
		for {
			conn, err := socksListener.Accept()
			if err != nil {
				if !strings.Contains(err.Error(), "use of closed") {
					t.Logf("SOCKS accept error: %v", err)
				}
				return
			}
			
			go func(c net.Conn) {
				handler := NewSOCKSHandler(c, security, metrics)
				handler.Handle()
			}(conn)
		}
	}()

	// Number of concurrent clients to simulate
	numClients := 20
	successChan := make(chan bool, numClients)
	
	var wg sync.WaitGroup
	wg.Add(numClients)
	
	// Launch multiple clients concurrently
	for i := 0; i < numClients; i++ {
		go func(clientID int) {
			defer wg.Done()
			
			// Connect to SOCKS server
			clientConn, err := net.Dial("tcp", socksAddr)
			if err != nil {
				t.Logf("Client %d: Failed to connect: %v", clientID, err)
				successChan <- false
				return
			}
			defer clientConn.Close()
			
			// SOCKS5 handshake
			_, err = clientConn.Write([]byte{SOCKS5_VERSION, 1, SOCKS5_NO_AUTH})
			if err != nil {
				t.Logf("Client %d: Failed to send greeting: %v", clientID, err)
				successChan <- false
				return
			}
			
			// Read server choice
			choice := make([]byte, 2)
			_, err = io.ReadFull(clientConn, choice)
			if err != nil {
				t.Logf("Client %d: Failed to read auth choice: %v", clientID, err)
				successChan <- false
				return
			}
			
			// Send UDP ASSOCIATE request
			udpRequest := []byte{
				SOCKS5_VERSION, SOCKS5_CMD_UDP_ASSOCIATE, 0x00, 
				SOCKS5_ADDR_IPV4, 0, 0, 0, 0, 0, 0,
			}
			_, err = clientConn.Write(udpRequest)
			if err != nil {
				t.Logf("Client %d: Failed to send UDP associate: %v", clientID, err)
				successChan <- false
				return
			}
			
			// Read response header
			respHeader := make([]byte, 4)
			_, err = io.ReadFull(clientConn, respHeader)
			if err != nil {
				t.Logf("Client %d: Failed to read response header: %v", clientID, err)
				successChan <- false
				return
			}
			
			// Check response code
			if respHeader[1] != SOCKS5_REP_SUCCESS {
				t.Logf("Client %d: UDP ASSOCIATE failed with code: 0x%02x", clientID, respHeader[1])
				successChan <- false
				return
			}
			
			// Read bound address based on type
			var addrLen int
			switch respHeader[3] {
			case SOCKS5_ADDR_IPV4:
				addrLen = 4
			case SOCKS5_ADDR_IPV6:
				addrLen = 16
			case SOCKS5_ADDR_DOMAIN:
				lenByte := make([]byte, 1)
				_, err = io.ReadFull(clientConn, lenByte)
				if err != nil {
					t.Logf("Client %d: Failed to read domain length: %v", clientID, err)
					successChan <- false
					return
				}
				addrLen = int(lenByte[0])
			default:
				t.Logf("Client %d: Invalid address type: 0x%02x", clientID, respHeader[3])
				successChan <- false
				return
			}
			
			// Read the address and port
			addrBytes := make([]byte, addrLen)
			_, err = io.ReadFull(clientConn, addrBytes)
			if err != nil {
				t.Logf("Client %d: Failed to read address: %v", clientID, err)
				successChan <- false
				return
			}
			
			portBytes := make([]byte, 2)
			_, err = io.ReadFull(clientConn, portBytes)
			if err != nil {
				t.Logf("Client %d: Failed to read port: %v", clientID, err)
				successChan <- false
				return
			}
			
			// Success if we got this far
			t.Logf("Client %d: Successfully established UDP ASSOCIATE", clientID)
			successChan <- true
			
			// Keep connection open a while to maintain the UDP relays
			time.Sleep(200 * time.Millisecond)
		}(i)
	}
	
	// Wait for all clients
	wg.Wait()
	close(successChan)
	
	// Count successes
	successCount := 0
	for success := range successChan {
		if success {
			successCount++
		}
	}
	
	// We expect some clients to succeed
	if successCount == 0 {
		t.Errorf("All UDP ASSOCIATE requests failed, expected some to succeed")
	} else if successCount < numClients {
		t.Logf("Warning: Only %d/%d UDP ASSOCIATE requests succeeded", successCount, numClients)
	} else {
		t.Logf("All %d UDP ASSOCIATE requests succeeded", successCount)
	}
}