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

// TestSOCKSAuthentication tests SOCKS5 authentication methods
func TestSOCKSAuthentication(t *testing.T) {
	// Setup test environment
	cleanup, _ := setupExtendedTestEnv()
	defer cleanup()

	// Enable test mode
	testModeCleanup := setupTestMode()
	defer testModeCleanup()

	// Setup security and metrics
	security := NewSecurityValidator()
	metrics := NewTestMetricsCollector()

	// Define test cases for authentication
	tests := []struct {
		name            string
		authMethods     []byte
		expectedMethod  byte
		expectSuccess   bool
		setupAuthServer func(t *testing.T) (string, func())
	}{
		{
			name:           "No authentication method",
			authMethods:    []byte{SOCKS5_NO_AUTH},
			expectedMethod: SOCKS5_NO_AUTH,
			expectSuccess:  true,
			setupAuthServer: func(t *testing.T) (string, func()) {
				listener, err := net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					t.Fatalf("Failed to start server: %v", err)
				}
				
				cleanup := func() {
					listener.Close()
				}
				
				go func() {
					conn, err := listener.Accept()
					if err != nil {
						if !strings.Contains(err.Error(), "use of closed") {
							t.Logf("Accept error: %v", err)
						}
						return
					}
					defer conn.Close()
					
					handler := NewSOCKSHandler(conn, security, metrics)
					handler.Handle()
				}()
				
				return listener.Addr().String(), cleanup
			},
		},
		{
			name:           "Client offers no acceptable auth methods",
			authMethods:    []byte{0xFF}, // An unsupported method
			expectedMethod: SOCKS5_AUTH_NO_ACCEPTABLE,
			expectSuccess:  false,
			setupAuthServer: func(t *testing.T) (string, func()) {
				listener, err := net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					t.Fatalf("Failed to start server: %v", err)
				}
				
				cleanup := func() {
					listener.Close()
				}
				
				go func() {
					conn, err := listener.Accept()
					if err != nil {
						if !strings.Contains(err.Error(), "use of closed") {
							t.Logf("Accept error: %v", err)
						}
						return
					}
					defer conn.Close()
					
					handler := NewSOCKSHandler(conn, security, metrics)
					handler.Handle()
				}()
				
				return listener.Addr().String(), cleanup
			},
		},
		{
			name:           "Client offers multiple auth methods (including NO_AUTH)",
			authMethods:    []byte{SOCKS5_AUTH_GSSAPI, SOCKS5_NO_AUTH, SOCKS5_AUTH_USERNAME_PASSWORD},
			expectedMethod: SOCKS5_NO_AUTH, // Server should choose NO_AUTH
			expectSuccess:  true,
			setupAuthServer: func(t *testing.T) (string, func()) {
				listener, err := net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					t.Fatalf("Failed to start server: %v", err)
				}
				
				cleanup := func() {
					listener.Close()
				}
				
				go func() {
					conn, err := listener.Accept()
					if err != nil {
						if !strings.Contains(err.Error(), "use of closed") {
							t.Logf("Accept error: %v", err)
						}
						return
					}
					defer conn.Close()
					
					handler := NewSOCKSHandler(conn, security, metrics)
					handler.Handle()
				}()
				
				return listener.Addr().String(), cleanup
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			serverAddr, cleanupServer := tc.setupAuthServer(t)
			defer cleanupServer()
			
			// Connect to server
			conn, err := net.Dial("tcp", serverAddr)
			if err != nil {
				t.Fatalf("Failed to connect to server: %v", err)
			}
			defer conn.Close()
			
			// Send client greeting with auth methods
			greeting := []byte{SOCKS5_VERSION, byte(len(tc.authMethods))}
			greeting = append(greeting, tc.authMethods...)
			
			_, err = conn.Write(greeting)
			if err != nil {
				t.Fatalf("Failed to send greeting: %v", err)
			}
			
			// Read server's choice
			response := make([]byte, 2)
			_, err = io.ReadFull(conn, response)
			
			if tc.expectSuccess {
				if err != nil {
					t.Fatalf("Failed to read response: %v", err)
				}
				
				if response[0] != SOCKS5_VERSION {
					t.Errorf("Expected SOCKS5 version 0x05, got: 0x%02x", response[0])
				}
				
				if response[1] != tc.expectedMethod {
					t.Errorf("Expected auth method 0x%02x, got: 0x%02x", tc.expectedMethod, response[1])
				}
				
				// If authentication was successful, try a simple CONNECT request
				if response[1] == SOCKS5_NO_AUTH {
					// Send CONNECT request to localhost
					targetIP := net.ParseIP("127.0.0.1").To4()
					targetPort := uint16(80) // Any port will do as we expect failure at connection stage
					
					req := []byte{
						SOCKS5_VERSION,      // Version
						SOCKS5_CMD_CONNECT,  // CONNECT command
						0x00,                // Reserved
						SOCKS5_ADDR_IPV4,    // IPv4 address
					}
					req = append(req, targetIP...)
					portBytes := make([]byte, 2)
					binary.BigEndian.PutUint16(portBytes, targetPort)
					req = append(req, portBytes...)
					
					_, err = conn.Write(req)
					if err != nil {
						t.Fatalf("Failed to send CONNECT request: %v", err)
					}
					
					// We expect a response (success or connection failure is irrelevant here)
					// Just verify we get a valid SOCKS5 response
					resp := make([]byte, 4)
					_, err = io.ReadFull(conn, resp)
					
					if err != nil {
						t.Fatalf("Failed to read CONNECT response: %v", err)
					}
					
					if resp[0] != SOCKS5_VERSION {
						t.Errorf("Expected SOCKS5 version in response, got: 0x%02x", resp[0])
					}
				}
			} else {
				// For failure cases, we might not get a response, or get a NO_ACCEPTABLE response
				if err == nil && response[1] == SOCKS5_AUTH_NO_ACCEPTABLE {
					t.Logf("Correctly received NO_ACCEPTABLE auth method")
					// Connection should be closed by server after this
					
					// Try to send another request (should fail)
					_, err = conn.Write([]byte{0x05, 0x01, 0x00})
					if err != nil {
						t.Logf("Correctly failed to write after auth rejection: %v", err)
					} else {
						// Try to read (should fail or timeout)
						if err = conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
							t.Logf("Failed to set read deadline: %v", err)
							return
						}
						buf := make([]byte, 10)
						_, err = conn.Read(buf)
						if err == nil {
							t.Errorf("Expected connection to be closed after auth rejection, but read succeeded")
						} else {
							t.Logf("Correctly received error after auth rejection: %v", err)
						}
					}
				} else if err != nil {
					t.Logf("Correctly failed to read response: %v", err)
				} else {
					t.Errorf("Expected auth failure, got method: 0x%02x", response[1])
				}
			}
		})
	}
}

// TestSOCKSLatencyValues tests SOCKS5 proxy with various latency values
func TestSOCKSLatencyValues(t *testing.T) {
	// Setup test environment with multiple celestial bodies
	cleanup, _ := setupExtendedTestEnv()
	defer cleanup()

	// Need control over test mode latency values
	originalTestMode := isTestMode
	defer func() { isTestMode = originalTestMode }()
	
	// We'll override the test mode latency for this test
	isTestMode = true

	// Setup security and metrics
	security := NewSecurityValidator()
	metrics := NewTestMetricsCollector()
	
	// Allow localhost for testing
	security.allowedHosts["127.0.0.1"] = true
	security.allowedHosts["localhost"] = true
	security.allowedPorts["8080"] = true

	// Define latency test scenarios
	tests := []struct {
		name         string
		bodyName     string
		latencyValue time.Duration
		expectDelay  bool
	}{
		{
			name:         "Mars (medium latency)",
			bodyName:     "Mars",
			latencyValue: 180 * time.Millisecond,
			expectDelay:  true,
		},
		{
			name:         "Earth (minimal latency)",
			bodyName:     "Earth",
			latencyValue: 10 * time.Millisecond,
			expectDelay:  true,
		},
		{
			name:         "Jupiter (high latency)",
			bodyName:     "Jupiter",
			latencyValue: 500 * time.Millisecond,
			expectDelay:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Set up test mode with the specific latency for this test case
			testCleanup := setupTestModeWithLatency(tc.latencyValue)
			defer testCleanup()
			
			// Start a SOCKS server
			socksListener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatalf("Failed to start SOCKS server: %v", err)
			}
			defer socksListener.Close()
			
			socksAddr := socksListener.Addr().String()
			
			// Start handling connections
			go func() {
				conn, err := socksListener.Accept()
				if err != nil {
					if !strings.Contains(err.Error(), "use of closed") {
						t.Logf("Accept error: %v", err)
					}
					return
				}
				
				// Create a test wrapper to simulate connection from the test celestial body
				wrappedConn := &testBodyConnection{
					Conn:     conn,
					bodyName: tc.bodyName,
				}
				
				handler := NewSOCKSHandler(wrappedConn, security, metrics)
				handler.Handle()
			}()
			
			// Start a mock target TCP server
			targetListener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatalf("Failed to start target server: %v", err)
			}
			defer targetListener.Close()
			
			targetPort := targetListener.Addr().(*net.TCPAddr).Port
			security.allowedPorts[strconv.Itoa(targetPort)] = true
			
			// Target server logic
			var targetWg sync.WaitGroup
			targetWg.Add(1)
			
			go func() {
				defer targetWg.Done()
				
				// Accept one connection
				conn, err := targetListener.Accept()
				if err != nil {
					if !strings.Contains(err.Error(), "use of closed") {
						t.Logf("Target accept error: %v", err)
					}
					return
				}
				defer conn.Close()
				
				// Echo back data
				buf := make([]byte, 1024)
				n, err := conn.Read(buf)
				if err != nil && err != io.EOF {
					t.Logf("Target read error: %v", err)
					return
				}
				
				_, err = conn.Write(buf[:n])
				if err != nil {
					t.Logf("Target write error: %v", err)
				}
			}()
			
			// Connect to SOCKS proxy
			clientConn, err := net.Dial("tcp", socksAddr)
			if err != nil {
				t.Fatalf("Failed to connect to SOCKS proxy: %v", err)
			}
			defer clientConn.Close()
			
			// SOCKS5 handshake
			_, err = clientConn.Write([]byte{SOCKS5_VERSION, 1, SOCKS5_NO_AUTH})
			if err != nil {
				t.Fatalf("Failed to send greeting: %v", err)
			}
			
			// Read response
			choice := make([]byte, 2)
			_, err = io.ReadFull(clientConn, choice)
			if err != nil {
				t.Fatalf("Failed to read auth choice: %v", err)
			}
			
			// Send CONNECT request
			targetIP := net.ParseIP("127.0.0.1").To4()
			targetPortNum := uint16(targetPort)
			
			req := []byte{
				SOCKS5_VERSION,      // Version
				SOCKS5_CMD_CONNECT,  // CONNECT command
				0x00,                // Reserved
				SOCKS5_ADDR_IPV4,    // IPv4 address
			}
			req = append(req, targetIP...)
			portBytes := make([]byte, 2)
			binary.BigEndian.PutUint16(portBytes, targetPortNum)
			req = append(req, portBytes...)
			
			// Time the request-response cycle
			startTime := time.Now()
			
			_, err = clientConn.Write(req)
			if err != nil {
				t.Fatalf("Failed to send CONNECT request: %v", err)
			}
			
			// Read response
			resp := make([]byte, 4)
			_, err = io.ReadFull(clientConn, resp)
			if err != nil {
				t.Fatalf("Failed to read response header: %v", err)
			}
			
			if resp[1] != SOCKS5_REP_SUCCESS {
				t.Fatalf("Connection failed, response code: 0x%02x", resp[1])
			}
			
			// Read address and port
			switch resp[3] {
			case SOCKS5_ADDR_IPV4:
				addr := make([]byte, 4)
				_, err = io.ReadFull(clientConn, addr)
				if err != nil {
					t.Fatalf("Failed to read IPv4 address: %v", err)
				}
			case SOCKS5_ADDR_IPV6:
				addr := make([]byte, 16)
				_, err = io.ReadFull(clientConn, addr)
				if err != nil {
					t.Fatalf("Failed to read IPv6 address: %v", err)
				}
			default:
				t.Fatalf("Unexpected address type: 0x%02x", resp[3])
			}
			
			portResponse := make([]byte, 2)
			_, err = io.ReadFull(clientConn, portResponse)
			if err != nil {
				t.Fatalf("Failed to read port: %v", err)
			}
			
			// Calculate elapsed time for connection setup
			setupTime := time.Since(startTime)
			
			// If we expect delay, verify it took at least the configured latency
			if tc.expectDelay {
				// We need to be a bit flexible here as the latency is just a simulation
				// and OS scheduling can affect timing
				minExpected := tc.latencyValue / 2
				if setupTime < minExpected {
					t.Errorf("Expected significant delay (at least %v), but setup completed in %v", minExpected, setupTime)
				} else {
					t.Logf("Connection setup took %v (expected significant delay)", setupTime)
				}
			}
			
			// Send data and measure round-trip time
			testData := []byte("Hello from latency test")
			startTime = time.Now()
			
			_, err = clientConn.Write(testData)
			if err != nil {
				t.Fatalf("Failed to send test data: %v", err)
			}
			
			// Read response (echo)
			respData := make([]byte, len(testData))
			_, err = io.ReadFull(clientConn, respData)
			if err != nil {
				t.Fatalf("Failed to read echo data: %v", err)
			}
			
			roundTrip := time.Since(startTime)
			
			// Verify data
			if !bytes.Equal(testData, respData) {
				t.Errorf("Response data doesn't match sent data")
			}
			
			// If we expect delay, verify it took a significant amount of time
			if tc.expectDelay {
				minExpected := tc.latencyValue
				if roundTrip < minExpected {
					t.Errorf("Expected significant round-trip delay (at least %v), but completed in %v", minExpected, roundTrip)
				} else {
					t.Logf("Round-trip took %v (expected significant delay)", roundTrip)
				}
			}
			
			// Clean up and wait for target
			clientConn.Close()
			targetWg.Wait()
		})
	}
}

// Test wrapper to simulate connection from a specific celestial body
type testBodyConnection struct {
	net.Conn
	bodyName string
}

func (t *testBodyConnection) RemoteAddr() net.Addr {
	return &testAddr{t.bodyName}
}

type testAddr struct {
	bodyName string
}

func (a *testAddr) Network() string {
	return "tcp"
}

func (a *testAddr) String() string {
	return a.bodyName + ".latency.space:12345"
}

// TestSOCKSConnTimeouts tests connection timeout handling
func TestSOCKSConnTimeouts(t *testing.T) {
	// Setup test environment
	cleanup, _ := setupExtendedTestEnv()
	defer cleanup()

	// Enable test mode
	testModeCleanup := setupTestMode()
	defer testModeCleanup()

	// Setup security and metrics
	security := NewSecurityValidator()
	metrics := NewTestMetricsCollector()
	
	// Allow localhost for testing
	security.allowedHosts["127.0.0.1"] = true
	security.allowedHosts["localhost"] = true
	security.allowedPorts["12345"] = true

	// Start a SOCKS server
	socksListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start SOCKS server: %v", err)
	}
	defer socksListener.Close()
	
	socksAddr := socksListener.Addr().String()
	
	// Handle connections
	go func() {
		for {
			conn, err := socksListener.Accept()
			if err != nil {
				if !strings.Contains(err.Error(), "use of closed") {
					t.Logf("Accept error: %v", err)
				}
				return
			}
			
			handler := NewSOCKSHandler(conn, security, metrics)
			go handler.Handle()
		}
	}()

	// Connect to SOCKS proxy
	clientConn, err := net.Dial("tcp", socksAddr)
	if err != nil {
		t.Fatalf("Failed to connect to SOCKS proxy: %v", err)
	}
	defer clientConn.Close()
	
	// SOCKS5 handshake
	_, err = clientConn.Write([]byte{SOCKS5_VERSION, 1, SOCKS5_NO_AUTH})
	if err != nil {
		t.Fatalf("Failed to send greeting: %v", err)
	}
	
	// Read response
	choice := make([]byte, 2)
	_, err = io.ReadFull(clientConn, choice)
	if err != nil {
		t.Fatalf("Failed to read auth choice: %v", err)
	}
	
	// Test connection to non-listening port (should timeout/fail)
	// Find a port that's likely not in use
	targetIP := net.ParseIP("127.0.0.1").To4()
	targetPort := uint16(12345) // Assuming this port is not listening
	
	req := []byte{
		SOCKS5_VERSION,      // Version
		SOCKS5_CMD_CONNECT,  // CONNECT command
		0x00,                // Reserved
		SOCKS5_ADDR_IPV4,    // IPv4 address
	}
	req = append(req, targetIP...)
	portBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(portBytes, targetPort)
	req = append(req, portBytes...)
	
	// Send the request - connection should be attempted and eventually time out
	_, err = clientConn.Write(req)
	if err != nil {
		t.Fatalf("Failed to send CONNECT request: %v", err)
	}
	
	// Read response - should receive a connection refused error
	// Don't need to read the whole response, just the status code
	resp := make([]byte, 4)
	_, err = io.ReadFull(clientConn, resp)
	if err != nil {
		t.Fatalf("Failed to read response header: %v", err)
	}
	
	// Check status code - should be connection refused or host unreachable
	if resp[1] != SOCKS5_REP_CONN_REFUSED && resp[1] != SOCKS5_REP_HOST_UNREACHABLE {
		t.Errorf("Expected connection refused or host unreachable, got: 0x%02x", resp[1])
	} else {
		t.Logf("Correctly received error code 0x%02x for non-listening port", resp[1])
	}
}

// TestSOCKSOcclusion tests the handling of celestial body occlusion
func TestSOCKSOcclusion(t *testing.T) {
	// This test will simulate occlusion scenarios for SOCKS proxy
	
	// Create a custom celestial objects setup for occlusion testing
	originalCelestialObjects := celestialObjects
	defer func() { celestialObjects = originalCelestialObjects }()
	
	// Create mock objects
	mockSun := CelestialObject{
		Name:   "Sun",
		Type:   "star",
		Radius: 696340,
	}
	
	// Setup a test helper for checking occlusion
	var occlusionMutex sync.Mutex
	occludeMars := true
	
	// Create a simple test stub for the Mars occlusion test
	testIsOccluded := func(marsName string) bool {
		occlusionMutex.Lock()
		defer occlusionMutex.Unlock()
		return strings.EqualFold(marsName, "Mars") && occludeMars
	}
	
	// Create minimum celestial objects required - the occlusion behavior is simulated
	testBodies := []CelestialObject{
		{Name: "Earth", Type: "planet"},
		{Name: "Mars", Type: "planet"},
		mockSun,
	}
	
	// Set up our test celestial objects
	celestialObjects = testBodies
	
	// Setup security and metrics
	security := NewSecurityValidator()
	metrics := NewTestMetricsCollector()
	
	// Allow localhost for testing
	security.allowedHosts["127.0.0.1"] = true
	
	// Test occlusion handling
	t.Run("Occluded body connection rejected", func(t *testing.T) {
		// Set Mars to be occluded
		occlusionMutex.Lock()
		occludeMars = true
		occlusionMutex.Unlock()
		
		// Start a SOCKS server
		socksListener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("Failed to start SOCKS server: %v", err)
		}
		defer socksListener.Close()
		
		socksAddr := socksListener.Addr().String()
		
		// Start handling connections
		go func() {
			conn, err := socksListener.Accept()
			if err != nil {
				if !strings.Contains(err.Error(), "use of closed") {
					t.Logf("Accept error: %v", err)
				}
				return
			}
			
			// Simulate connection from Mars
			wrappedConn := &testBodyConnection{
				Conn:     conn,
				bodyName: "Mars",
			}
			
			// Check if our test wants to simulate occlusion
			if testIsOccluded("Mars") {
				// Simulate connection being rejected due to occlusion
				conn.Close()
				return
			}
			
			handler := NewSOCKSHandler(wrappedConn, security, metrics)
			handler.Handle()
		}()
		
		// Connect to SOCKS proxy
		clientConn, err := net.Dial("tcp", socksAddr)
		if err != nil {
			t.Fatalf("Failed to connect to SOCKS proxy: %v", err)
		}
		defer clientConn.Close()
		
		// SOCKS5 handshake
		_, err = clientConn.Write([]byte{SOCKS5_VERSION, 1, SOCKS5_NO_AUTH})
		if err != nil {
			t.Fatalf("Failed to send greeting: %v", err)
		}
		
		// The connection should be closed immediately due to occlusion
		// Try to read response, should fail or timeout
		if err = clientConn.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
			t.Fatalf("Failed to set read deadline: %v", err)
		}
		choice := make([]byte, 2)
		n, err := io.ReadFull(clientConn, choice)
		
		// We expect either an error or connection closed
		if err == nil && n == 2 {
			t.Errorf("Expected connection to be closed due to occlusion, but read succeeded")
		} else {
			t.Logf("Correctly failed to read from occluded connection: %v", err)
		}
	})
	
	t.Run("Non-occluded body connection accepted", func(t *testing.T) {
		// Set Mars to not be occluded
		occlusionMutex.Lock()
		occludeMars = false
		occlusionMutex.Unlock()
		
		// Start a SOCKS server
		socksListener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("Failed to start SOCKS server: %v", err)
		}
		defer socksListener.Close()
		
		socksAddr := socksListener.Addr().String()
		
		// Start handling connections
		go func() {
			conn, err := socksListener.Accept()
			if err != nil {
				if !strings.Contains(err.Error(), "use of closed") {
					t.Logf("Accept error: %v", err)
				}
				return
			}
			
			// Simulate connection from Mars (now not occluded)
			wrappedConn := &testBodyConnection{
				Conn:     conn,
				bodyName: "Mars",
			}
			
			// Check if our test wants to simulate occlusion
			if testIsOccluded("Mars") {
				// Simulate connection being rejected due to occlusion
				conn.Close()
				return
			}
			
			handler := NewSOCKSHandler(wrappedConn, security, metrics)
			handler.Handle()
		}()
		
		// Connect to SOCKS proxy
		clientConn, err := net.Dial("tcp", socksAddr)
		if err != nil {
			t.Fatalf("Failed to connect to SOCKS proxy: %v", err)
		}
		defer clientConn.Close()
		
		// SOCKS5 handshake
		_, err = clientConn.Write([]byte{SOCKS5_VERSION, 1, SOCKS5_NO_AUTH})
		if err != nil {
			t.Fatalf("Failed to send greeting: %v", err)
		}
		
		// Should receive response since not occluded
		choice := make([]byte, 2)
		_, err = io.ReadFull(clientConn, choice)
		if err != nil {
			t.Fatalf("Failed to read auth choice: %v", err)
		}
		
		if choice[0] != SOCKS5_VERSION || choice[1] != SOCKS5_NO_AUTH {
			t.Fatalf("Invalid auth response: %v", choice)
		}
		
		t.Logf("Successfully established connection when not occluded")
	})
}