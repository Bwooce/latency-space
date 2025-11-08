# Implementation Notes for Port-Per-Body SOCKS5

## Changes Needed to proxy/src/main.go

### 1. Add global configuration variables

After the global variables section, add:

```go
var (
	// Configuration from environment variables
	fixedCelestialBody string // If set, use this body instead of detecting from hostname
	httpEnabled        bool   // Whether to start HTTP/HTTPS servers
	socksEnabled       bool   // Whether to start SOCKS5 server
)
```

### 2. Modify Server struct

Add fields to track configuration:

```go
type Server struct {
	port               int
	https              bool
	metrics            *MetricsCollector
	security           *SecurityValidator
	httpServer         *http.Server
	httpsServer        *http.Server
	socksListener      net.Listener
	httpEnabled        bool   // NEW: Whether HTTP/HTTPS should run
	socksEnabled       bool   // NEW: Whether SOCKS5 should run
	fixedCelestialBody string // NEW: Fixed celestial body for this instance
}
```

### 3. Modify NewServer constructor

```go
func NewServer(port int, useHTTPS bool, httpEn bool, socksEn bool, fixedBody string) *Server {
	return &Server{
		port:               port,
		https:              useHTTPS,
		metrics:            NewMetricsCollector(),
		security:           NewSecurityValidator(),
		httpEnabled:        httpEn,
		socksEnabled:       socksEn,
		fixedCelestialBody: fixedBody,
	}
}
```

### 4. Modify Start() method

Make HTTP/HTTPS and SOCKS5 servers conditional:

```go
func (s *Server) Start() error {
	// ... existing signal handling code ...

	// Start HTTP server only if enabled
	if s.httpEnabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := s.startHTTPServer()
			if err != nil && err != http.ErrServerClosed {
				errCh <- fmt.Errorf("HTTP server error: %v", err)
			}
		}()
	}

	// Start HTTPS server only if enabled and https flag is true
	if s.httpEnabled && s.https {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := s.startHTTPSServer()
			if err != nil && err != http.ErrServerClosed {
				errCh <- fmt.Errorf("HTTPS server error: %v", err)
			}
		}()
	}

	// Start SOCKS5 server only if enabled
	if s.socksEnabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := s.startSOCKSServer()
			if err != nil {
				errCh <- fmt.Errorf("SOCKS5 server error: %v", err)
			}
		}()
	}

	// ... rest of Start() method ...
}
```

### 5. Pass fixedCelestialBody to SOCKS handler

In startSOCKSServer(), modify where SOCKSHandler is created:

```go
// Handle the connection in a goroutine
go NewSOCKSHandler(conn, s.security, s.metrics, s.fixedCelestialBody).Handle()
```

### 6. Update main() function

Add environment variable reading:

```go
func main() {
	// Parse command-line arguments
	port := flag.Int("port", 80, "HTTP port to listen on")
	https := flag.Bool("https", true, "Enable HTTPS")
	flag.Parse()

	// Read environment variables
	fixedCelestialBody = os.Getenv("CELESTIAL_BODY")

	httpEnabledStr := os.Getenv("HTTP_ENABLED")
	httpEnabled = httpEnabledStr != "false" // Default to true unless explicitly false

	socksEnabledStr := os.Getenv("SOCKS_ENABLED")
	socksEnabled = socksEnabledStr != "false" // Default to true unless explicitly false

	// Log configuration
	log.Printf("Configuration:")
	log.Printf("  HTTP/HTTPS Enabled: %v", httpEnabled)
	log.Printf("  SOCKS5 Enabled: %v", socksEnabled)
	if fixedCelestialBody != "" {
		log.Printf("  Fixed Celestial Body: %s", fixedCelestialBody)
	} else {
		log.Printf("  Celestial Body: Dynamic (from hostname)")
	}

	// ... template loading code ...

	// Initialize celestial objects
	celestialObjects = celestial.InitSolarSystemObjects()

	// Validate fixed celestial body if set
	if fixedCelestialBody != "" {
		_, found := findObjectByName(celestialObjects, fixedCelestialBody)
		if !found {
			log.Fatalf("Invalid CELESTIAL_BODY: '%s' not found in solar system objects", fixedCelestialBody)
		}
	}

	// Create and start the server
	server := NewServer(*port, *https, httpEnabled, socksEnabled, fixedCelestialBody)
	err = server.Start()
	if err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
```

## Changes Needed to proxy/src/socks_helpers.go

### 1. Modify SOCKSHandler struct

Add field for fixed celestial body:

```go
type SOCKSHandler struct {
	conn               net.Conn
	security           *SecurityValidator
	metrics            *MetricsCollector
	fixedCelestialBody string // NEW: If set, use this instead of detecting from hostname
}
```

### 2. Modify NewSOCKSHandler

```go
func NewSOCKSHandler(conn net.Conn, security *SecurityValidator, metrics *MetricsCollector, fixedBody string) *SOCKSHandler {
	return &SOCKSHandler{
		conn:               conn,
		security:           security,
		metrics:            metrics,
		fixedCelestialBody: fixedBody,
	}
}
```

### 3. Modify getCelestialBodyFromConn

Update to use fixed body if available:

```go
func (s *SOCKSHandler) getCelestialBodyFromConn(addr net.Addr) (string, error) {
	// If a fixed celestial body is configured, use it
	if s.fixedCelestialBody != "" {
		log.Printf("Using fixed celestial body: %s", s.fixedCelestialBody)
		return s.fixedCelestialBody, nil
	}

	// Otherwise, try to determine from connection (existing logic)
	host := addr.String()

	// ... rest of existing getCelestialBodyFromConn logic ...
}
```

## Testing Plan

### 1. Test single SOCKS5 proxy for Mars:

```bash
docker-compose up socks-mars

# Test
curl --socks5-hostname latency.space:1080 https://example.com
```

### 2. Test multiple SOCKS5 proxies:

```bash
docker-compose up socks-mars socks-moon socks-jupiter

# Test Mars (port 1080)
curl --socks5-hostname latency.space:1080 https://example.com

# Test Moon (port 1081)
curl --socks5-hostname latency.space:1081 https://example.com

# Test Jupiter (port 1084)
curl --socks5-hostname latency.space:1084 https://example.com
```

### 3. Test main HTTP proxy still works:

```bash
docker-compose up proxy

# Should serve info pages
curl http://localhost:8080/
```

### 4. Verify metrics endpoints:

```bash
# Main proxy metrics
curl http://localhost:9090/metrics

# Mars SOCKS5 metrics
curl http://localhost:9100/metrics

# Moon SOCKS5 metrics
curl http://localhost:9101/metrics
```

## Deployment Steps

1. Build and tag images:
```bash
docker-compose build
```

2. Start all services:
```bash
docker-compose up -d
```

3. Verify all containers are running:
```bash
docker-compose ps
```

4. Check logs for each service:
```bash
docker-compose logs socks-mars
docker-compose logs socks-moon
```

5. Test connectivity to each port

6. Update firewall rules to allow all SOCKS5 ports (1080-1089, 2080-2089, 3080-3089)

## Notes

- Each SOCKS5 proxy container will log its configured celestial body on startup
- If CELESTIAL_BODY is invalid, container will fail immediately with clear error
- Main proxy container runs with HTTP_ENABLED=true (default), SOCKS_ENABLED=false
- SOCKS proxy containers run with HTTP_ENABLED=false, SOCKS_ENABLED=true (default)
- Metrics are exposed on different ports to avoid conflicts (9100+, 9200+, 9300+)
