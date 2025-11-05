# Implementation Plan - Hybrid Approach for latency.space

**Date:** 2025-11-05
**Approach:** Option C - Hybrid Approach
**Goal:** Ship working proxy functionality while maintaining elegant info pages

---

## Overview

This plan implements the recommended hybrid approach:
- ✅ **SOCKS5** as primary proxy interface (full functionality, no SSL limitations)
- ✅ **HTTPS** for info pages and status dashboard (beautiful UX)
- ✅ **HTTP API** for programmatic access (enables web demos)
- ✅ **Web demo interface** (interactive testing without SSL issues)

---

## Success Criteria

### Phase 1 Complete When:
- [ ] SOCKS5 proxy accessible and functional
- [ ] Users can proxy traffic through Mars with measurable latency
- [ ] Info pages work with valid HTTPS certificates
- [ ] Documentation clearly explains usage

### Phase 2 Complete When:
- [ ] Web demo allows testing without SOCKS5 configuration
- [ ] HTTP API handles proxy requests
- [ ] Setup guides make SOCKS5 configuration easy
- [ ] Browser extension (optional) simplifies setup

### Phase 3 Complete When:
- [ ] Advanced features documented and working
- [ ] Monitoring shows healthy usage
- [ ] User feedback incorporated

---

## Phase 1: Core Functionality (Week 1)

### 🎯 Goal: Get SOCKS5 working end-to-end

---

### Task 1.1: Fix Cloudflare Configuration

**Priority:** P0 - CRITICAL
**Time Estimate:** 15 minutes
**Dependencies:** None

**IMPORTANT:** Current configuration has individual A records (mars, jupiter, etc.), NOT a wildcard.

**Subtasks:**

- [ ] **1.1.1** Log into Cloudflare dashboard
- [ ] **1.1.2** Navigate to DNS settings for `latency.space`
- [ ] **1.1.3** **Add wildcard record:**
  - Click "Add record"
  - Type: `A`
  - Name: `*` (just asterisk)
  - Content: YOUR_SERVER_IP (same as latency.space)
  - **Proxy status: DNS only** (gray cloud) ⚠️
  - TTL: Auto
  - Save
- [ ] **1.1.4** **Change ALL celestial records to DNS-only:**
  - For each record (mars, jupiter, moon, phobos, etc.):
  - Click the orange cloud ☁️
  - Change to gray cloud (DNS only)
  - Save
  - Repeat for all ~50 records
- [ ] **1.1.5** **Keep main site proxied:**
  - `latency.space` → Leave orange ☁️ (Proxied)
  - `www.latency.space` → Leave orange ☁️ (Proxied)
- [ ] **1.1.6** Wait 2-5 minutes for DNS propagation
- [ ] **1.1.7** Test wildcard works: `nslookup test.latency.space`
- [ ] **1.1.8** Test direct resolution: `nslookup mars.latency.space` (should show YOUR_IP)
- [ ] **1.1.9** Test SOCKS5 accessible: `nc -zv mars.latency.space 1080`
- [ ] **1.1.10** Document in `docs/CLOUDFLARE_DNS_CONFIGURATION.md` (created)

**Verification:**
```bash
# Wildcard resolves to your server
nslookup example.com.mars.latency.space
# Should return YOUR_SERVER_IP

# Direct resolution works (not through Cloudflare)
nslookup mars.latency.space
# Should return YOUR_SERVER_IP (not Cloudflare IP range)

# SOCKS5 port accessible
nc -zv mars.latency.space 1080
# Should connect successfully
```

**Files Modified:**
- Cloudflare dashboard (add wildcard, change ~50 records)
- New file: `docs/CLOUDFLARE_DNS_CONFIGURATION.md` (detailed guide created)

---

### Task 1.2: Deploy Simplified SSL Certificate

**Priority:** P0 - CRITICAL
**Time Estimate:** 45 minutes
**Dependencies:** None

**Subtasks:**

- [ ] **1.2.1** SSH into production server
- [ ] **1.2.2** Install certbot if not present: `apt-get install certbot python3-certbot-dns-cloudflare`
- [ ] **1.2.3** Configure Cloudflare API credentials for DNS challenge
- [ ] **1.2.4** Run certificate request:
  ```bash
  certbot certonly --dns-cloudflare \
    --dns-cloudflare-credentials ~/.secrets/cloudflare.ini \
    -d latency.space \
    -d *.latency.space \
    --agree-tos \
    --non-interactive
  ```
- [ ] **1.2.5** Verify certificate files created in `/etc/letsencrypt/live/latency.space/`
- [ ] **1.2.6** Check certificate coverage:
  ```bash
  openssl x509 -in /etc/letsencrypt/live/latency.space/fullchain.pem -text -noout | grep DNS
  ```
- [ ] **1.2.7** Restart proxy service to load new certificate
- [ ] **1.2.8** Test HTTPS works: `curl -I https://mars.latency.space/`
- [ ] **1.2.9** Set up auto-renewal cron job
- [ ] **1.2.10** Document certificate setup process

**Verification:**
```bash
# Should show valid certificate for *.latency.space
curl -vI https://mars.latency.space/ 2>&1 | grep -i "subject\|issuer"

# Should not have SSL errors
curl https://mars.latency.space/ | grep -i "mars"
```

**Files Modified:**
- `/etc/letsencrypt/live/latency.space/*` (new certificates)
- New file: `docs/ssl-setup.md`

---

### Task 1.3: Test SOCKS5 Proxy Functionality

**Priority:** P0 - CRITICAL
**Time Estimate:** 30 minutes
**Dependencies:** Task 1.1

**Subtasks:**

- [ ] **1.3.1** Verify proxy service is running: `systemctl status latency-space-proxy`
- [ ] **1.3.2** Check SOCKS5 port is listening: `netstat -tuln | grep 1080`
- [ ] **1.3.3** Test basic SOCKS5 connection:
  ```bash
  curl --socks5 mars.latency.space:1080 https://example.com
  ```
- [ ] **1.3.4** Time the request to verify latency is applied:
  ```bash
  time curl --socks5 mars.latency.space:1080 https://example.com
  ```
- [ ] **1.3.5** Expected: ~40 minutes for Mars round-trip (check API for current distance)
- [ ] **1.3.6** Test with different celestial bodies:
  ```bash
  curl --socks5 jupiter.latency.space:1080 https://example.com
  curl --socks5 moon.earth.latency.space:1080 https://example.com
  ```
- [ ] **1.3.7** Test UDP ASSOCIATE (DNS queries):
  ```bash
  dig @8.8.8.8 example.com +tcp +socks=mars.latency.space:1080
  ```
- [ ] **1.3.8** Verify metrics are being collected: `curl http://localhost:9090/metrics | grep socks`
- [ ] **1.3.9** Check logs for any errors: `journalctl -u latency-space-proxy -n 100`
- [ ] **1.3.10** Document test results in `docs/testing-socks5.md`

**Verification:**
```bash
# Should return example.com content after Mars latency delay
time curl --socks5 mars.latency.space:1080 https://example.com

# Check current Mars distance
curl https://latency.space/api/status-data | jq '.objects.planets[] | select(.name=="Mars") | .latency_seconds'
```

**Files Modified:**
- New file: `docs/testing-socks5.md`

---

### Task 1.4: Update README.md - Emphasize SOCKS5

**Priority:** P0 - CRITICAL
**Time Estimate:** 1 hour
**Dependencies:** None

**Subtasks:**

- [ ] **1.4.1** Read current `README.md` to understand structure
- [ ] **1.4.2** Add prominent "Quick Start - SOCKS5 Proxy" section at top
- [ ] **1.4.3** Include copy-paste ready examples:
  ```markdown
  ## Quick Start - SOCKS5 Proxy (Recommended)

  Experience Mars latency in 30 seconds:

  ```bash
  # Single request through Mars
  curl --socks5 mars.latency.space:1080 https://example.com

  # Configure your browser (Firefox example):
  # Settings → Network Settings → Manual proxy configuration
  # SOCKS Host: mars.latency.space
  # Port: 1080
  # SOCKS v5: ✓
  ```
  ```
- [ ] **1.4.4** Update "Available Endpoints" section to prioritize SOCKS5
- [ ] **1.4.5** Add "Why SOCKS5?" explanation section
- [ ] **1.4.6** Document SSL limitations clearly:
  ```markdown
  ## HTTPS Limitations

  Due to SSL/TLS certificate limitations (RFC 6125), the following patterns
  do NOT work with HTTPS:

  - ❌ `https://example.com.mars.latency.space/` (3+ subdomain levels)
  - ❌ `https://api.github.com.jupiter.latency.space/`

  **Solution:** Use SOCKS5 proxy instead - it works perfectly with all patterns!
  ```
- [ ] **1.4.7** Remove or correct misleading certificate instructions (remove `*.*.latency.space`)
- [ ] **1.4.8** Add troubleshooting section for common SOCKS5 issues
- [ ] **1.4.9** Update examples to show SOCKS5 first, HTTP second
- [ ] **1.4.10** Add link to detailed SOCKS5 setup guide

**Verification:**
- [ ] README clearly states SOCKS5 is primary interface
- [ ] No false promises about multi-level HTTPS
- [ ] Examples work when copy-pasted

**Files Modified:**
- `README.md` (major updates)

---

### Task 1.5: Create SOCKS5 Setup Guide

**Priority:** P1 - HIGH
**Time Estimate:** 2 hours
**Dependencies:** None

**Subtasks:**

- [ ] **1.5.1** Create new file: `docs/SOCKS5_SETUP_GUIDE.md`
- [ ] **1.5.2** Write introduction explaining SOCKS5 benefits
- [ ] **1.5.3** Add browser-specific instructions:
  - [ ] **1.5.3.1** Firefox configuration (with screenshots if possible)
  - [ ] **1.5.3.2** Chrome/Chromium configuration
  - [ ] **1.5.3.3** Safari configuration
  - [ ] **1.5.3.4** Edge configuration
- [ ] **1.5.4** Add command-line tool examples:
  - [ ] **1.5.4.1** curl with SOCKS5
  - [ ] **1.5.4.2** wget with SOCKS5
  - [ ] **1.5.4.3** ssh with ProxyCommand
  - [ ] **1.5.4.4** git with SOCKS5 proxy
- [ ] **1.5.5** Add programming language examples:
  - [ ] **1.5.5.1** Python (requests library)
  - [ ] **1.5.5.2** Node.js (socks-proxy-agent)
  - [ ] **1.5.5.3** Go (proxy.SOCKS5)
  - [ ] **1.5.5.4** Java (Proxy class)
- [ ] **1.5.6** Add Docker usage examples
- [ ] **1.5.7** Add system-wide proxy configuration (Linux, macOS, Windows)
- [ ] **1.5.8** Add troubleshooting section:
  - Connection refused
  - DNS resolution issues
  - Authentication errors
  - Timeout problems
- [ ] **1.5.9** Add FAQ section
- [ ] **1.5.10** Link from README.md

**Verification:**
- [ ] Each example tested and works
- [ ] Guide covers all major use cases
- [ ] Troubleshooting addresses real issues

**Files Modified:**
- New file: `docs/SOCKS5_SETUP_GUIDE.md`
- `README.md` (add link)

---

### Task 1.6: Clean Up Codebase

**Priority:** P1 - HIGH
**Time Estimate:** 1 hour
**Dependencies:** None

**Subtasks:**

- [ ] **1.6.1** Remove all `.bak` files:
  ```bash
  find . -name "*.bak" -type f -delete
  ```
- [ ] **1.6.2** Update `.gitignore` to prevent future `.bak` files:
  ```
  *.bak
  *.tmp
  *~
  ```
- [ ] **1.6.3** Remove DEBUG log statements from production code:
  - [ ] `proxy/src/main.go:637` - DEBUG: distanceEntries
  - [ ] `proxy/src/main.go:695` - DEBUG: API Response
  - [ ] `proxy/src/socks.go:511` - DEBUG: UDP relay
  - [ ] `proxy/src/socks.go:610,636` - DEBUG: UDP relay messages
- [ ] **1.6.4** Remove duplicate `test-docker-build/` directory:
  ```bash
  rm -rf test-docker-build/
  ```
- [ ] **1.6.5** Clean up commented code in `docker-compose.yml`
- [ ] **1.6.6** Fix `go.mod` dependencies:
  ```bash
  cd proxy/src && go mod tidy
  ```
- [ ] **1.6.7** Run tests to ensure nothing broke:
  ```bash
  cd proxy/src && go test -v ./...
  ```
- [ ] **1.6.8** Commit cleanup changes
- [ ] **1.6.9** Update `CLAUDE.md` with new structure
- [ ] **1.6.10** Create `.dockerignore` file

**Verification:**
```bash
# No .bak files
find . -name "*.bak"

# go.mod has dependencies
cat go.mod | grep require

# Tests pass
cd proxy/src && go test ./...
```

**Files Modified:**
- `.gitignore`
- `proxy/src/main.go`
- `proxy/src/socks.go`
- `docker-compose.yml`
- `go.mod`
- New file: `.dockerignore`
- `CLAUDE.md`

---

### Task 1.7: Verify End-to-End Functionality

**Priority:** P0 - CRITICAL
**Time Estimate:** 1 hour
**Dependencies:** Tasks 1.1, 1.2, 1.3

**Subtasks:**

- [ ] **1.7.1** Test API endpoint returns accurate data:
  ```bash
  curl https://latency.space/api/status-data | jq '.objects.planets[0]'
  ```
- [ ] **1.7.2** Test info pages load with valid SSL:
  ```bash
  curl -I https://mars.latency.space/
  curl -I https://jupiter.latency.space/
  ```
- [ ] **1.7.3** Test SOCKS5 proxy with latency measurement:
  ```bash
  time curl --socks5 mars.latency.space:1080 https://example.com
  ```
- [ ] **1.7.4** Verify latency matches expected value from API
- [ ] **1.7.5** Test multiple celestial bodies:
  - [ ] Moon (should be ~3 seconds RTT)
  - [ ] Mars (should be ~20-40 minutes RTT depending on position)
  - [ ] Jupiter (should be ~1-2 hours RTT)
- [ ] **1.7.6** Test occlusion handling (if any bodies currently occluded)
- [ ] **1.7.7** Test metrics collection:
  ```bash
  curl http://localhost:9090/metrics | grep latency_space
  ```
- [ ] **1.7.8** Test debug endpoints:
  ```bash
  curl https://latency.space/_debug/distances
  curl https://latency.space/_debug/help
  ```
- [ ] **1.7.9** Document test results
- [ ] **1.7.10** Take screenshots/recordings for documentation

**Verification:**
- [ ] All endpoints return expected responses
- [ ] Latency simulation works correctly
- [ ] No SSL errors
- [ ] Metrics show activity

**Files Modified:**
- New file: `docs/end-to-end-test-results.md`

---

## Phase 2: Enhanced User Experience (Week 2-3)

### 🎯 Goal: Make SOCKS5 easy to use and add web demo

---

### Task 2.1: Create Web Demo Interface

**Priority:** P1 - HIGH
**Time Estimate:** 8 hours
**Dependencies:** None

**Subtasks:**

- [ ] **2.1.1** Create new React component: `status/src/pages/Demo.jsx`
- [ ] **2.1.2** Add routing in `status/src/App.jsx`: `/demo` route
- [ ] **2.1.3** Design UI mockup:
  - Dropdown to select celestial body
  - Input field for target URL
  - "Fetch" button
  - Loading indicator with countdown timer
  - Result display area
  - Latency visualization graph
- [ ] **2.1.4** Implement celestial body dropdown:
  ```jsx
  // Fetch from API
  const [bodies, setBodies] = useState([]);
  useEffect(() => {
    fetch('/api/status-data')
      .then(r => r.json())
      .then(data => {
        const allBodies = [
          ...data.objects.planets,
          ...data.objects.moons,
          ...data.objects.spacecraft
        ];
        setBodies(allBodies);
      });
  }, []);
  ```
- [ ] **2.1.5** Add target URL input with validation
- [ ] **2.1.6** Implement fetch functionality (calls backend API)
- [ ] **2.1.7** Add latency countdown timer during fetch
- [ ] **2.1.8** Display fetched content in iframe or formatted div
- [ ] **2.1.9** Add latency visualization (progress bar showing time elapsed)
- [ ] **2.1.10** Show comparison: "This took X minutes. At Earth it would be instant."
- [ ] **2.1.11** Add example URLs for quick testing
- [ ] **2.1.12** Add error handling for failed fetches
- [ ] **2.1.13** Make responsive for mobile
- [ ] **2.1.14** Add link to demo from main landing page
- [ ] **2.1.15** Test with multiple celestial bodies

**Verification:**
- [ ] Can select Mars and fetch example.com
- [ ] Latency delay is accurately shown
- [ ] Content displays correctly
- [ ] Works on mobile

**Files Modified:**
- New file: `status/src/pages/Demo.jsx`
- `status/src/App.jsx` (add route)
- `status/src/pages/Landing.jsx` (add link)

---

### Task 2.2: Implement HTTP Proxy API

**Priority:** P1 - HIGH
**Time Estimate:** 4 hours
**Dependencies:** None

**Subtasks:**

- [ ] **2.2.1** Add new endpoint in `proxy/src/main.go`:
  ```go
  if r.URL.Path == "/api/proxy" {
      s.handleProxyAPI(w, r)
      return
  }
  ```
- [ ] **2.2.2** Implement `handleProxyAPI` function:
  ```go
  func (s *Server) handleProxyAPI(w http.ResponseWriter, r *http.Request) {
      // Parse JSON body
      // Extract: via (celestial body), url (target)
      // Validate inputs
      // Calculate latency
      // Apply delay
      // Fetch from target URL
      // Return content with headers
  }
  ```
- [ ] **2.2.3** Add CORS headers for API:
  ```go
  w.Header().Set("Access-Control-Allow-Origin", "https://latency.space")
  w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
  ```
- [ ] **2.2.4** Add request body structure:
  ```go
  type ProxyRequest struct {
      Via string `json:"via"`  // celestial body name
      URL string `json:"url"`  // target URL
  }
  ```
- [ ] **2.2.5** Add response structure:
  ```go
  type ProxyResponse struct {
      Content     string            `json:"content"`
      StatusCode  int               `json:"status_code"`
      Headers     map[string]string `json:"headers"`
      Latency     float64           `json:"latency_seconds"`
      Distance    float64           `json:"distance_km"`
  }
  ```
- [ ] **2.2.6** Add authentication/rate limiting (use existing security validator)
- [ ] **2.2.7** Add metrics collection for API usage
- [ ] **2.2.8** Add unit tests for proxy API
- [ ] **2.2.9** Update OpenAPI spec (if exists) or create one
- [ ] **2.2.10** Document API in `docs/API.md`
- [ ] **2.2.11** Test API endpoint:
  ```bash
  curl -X POST https://latency.space/api/proxy \
    -H "Content-Type: application/json" \
    -d '{"via": "mars", "url": "https://example.com"}'
  ```
- [ ] **2.2.12** Update web demo to use this API

**Verification:**
```bash
# Should return example.com content after Mars latency
time curl -X POST https://latency.space/api/proxy \
  -H "Content-Type: application/json" \
  -d '{"via": "mars", "url": "https://example.com"}'
```

**Files Modified:**
- `proxy/src/main.go` (add handler)
- New file: `proxy/src/api_handlers.go` (proxy API logic)
- New file: `proxy/src/api_handlers_test.go`
- New file: `docs/API.md`

---

### Task 2.3: Create Browser Extension (Optional)

**Priority:** P2 - MEDIUM
**Time Estimate:** 12 hours
**Dependencies:** None

**Subtasks:**

- [ ] **2.3.1** Create extension directory structure:
  ```
  extension/
    manifest.json
    popup.html
    popup.js
    background.js
    icons/
  ```
- [ ] **2.3.2** Write `manifest.json` for Chrome/Firefox compatibility
- [ ] **2.3.3** Create popup UI:
  - Toggle to enable/disable proxy
  - Dropdown to select celestial body
  - Current latency display
  - Quick tips
- [ ] **2.3.4** Implement proxy.settings API usage:
  ```js
  chrome.proxy.settings.set({
    value: {
      mode: "fixed_servers",
      rules: {
        singleProxy: {
          scheme: "socks5",
          host: "mars.latency.space",
          port: 1080
        }
      }
    },
    scope: "regular"
  });
  ```
- [ ] **2.3.5** Add auto-detection of current celestial position
- [ ] **2.3.6** Show notification when proxy is enabled
- [ ] **2.3.7** Add bypass list for local/internal sites
- [ ] **2.3.8** Test in Chrome
- [ ] **2.3.9** Test in Firefox
- [ ] **2.3.10** Package for Chrome Web Store
- [ ] **2.3.11** Package for Firefox Add-ons
- [ ] **2.3.12** Write extension documentation
- [ ] **2.3.13** Create demo video
- [ ] **2.3.14** Submit to stores (if desired)

**Verification:**
- [ ] Extension installs successfully
- [ ] Proxy configuration works
- [ ] UI is intuitive
- [ ] Works in both browsers

**Files Modified:**
- New directory: `extension/`
- New file: `docs/BROWSER_EXTENSION.md`

---

### Task 2.4: Enhanced Documentation

**Priority:** P1 - HIGH
**Time Estimate:** 4 hours
**Dependencies:** Tasks 2.1, 2.2

**Subtasks:**

- [ ] **2.4.1** Create `docs/ARCHITECTURE.md`:
  - System architecture diagram
  - Component descriptions
  - Data flow diagrams
  - Technology stack
- [ ] **2.4.2** Create `docs/DEPLOYMENT.md`:
  - Server requirements
  - Docker deployment
  - Cloudflare configuration
  - SSL certificate setup
  - Monitoring setup
- [ ] **2.4.3** Enhance `docs/SOCKS5_SETUP_GUIDE.md`:
  - Add video walkthrough links (if created)
  - Add common error solutions
  - Add advanced configurations
- [ ] **2.4.4** Create `docs/API.md`:
  - Document `/api/status-data` endpoint
  - Document `/api/proxy` endpoint
  - Document `/_debug/*` endpoints
  - Add request/response examples
  - Add error codes
- [ ] **2.4.5** Create `docs/CONTRIBUTING.md`:
  - How to add new celestial bodies
  - How to modify latency calculations
  - Code style guidelines
  - Testing requirements
- [ ] **2.4.6** Update main `README.md`:
  - Add "Featured on..." badges (if applicable)
  - Add demo GIF/video
  - Add architecture diagram link
  - Reorganize for better flow
- [ ] **2.4.7** Create `docs/FAQ.md`:
  - Why SOCKS5 instead of HTTP?
  - Why doesn't HTTPS work for all subdomains?
  - How accurate are the latencies?
  - Can I add custom celestial bodies?
  - Is this production-ready?
- [ ] **2.4.8** Create `docs/TROUBLESHOOTING.md`:
  - Connection refused errors
  - SSL certificate issues
  - SOCKS5 not working
  - Latency seems wrong
  - Cloudflare issues
- [ ] **2.4.9** Add link tree to main README
- [ ] **2.4.10** Review all docs for consistency

**Verification:**
- [ ] All docs are accurate
- [ ] Links work
- [ ] Examples are tested
- [ ] Typos fixed

**Files Modified:**
- New file: `docs/ARCHITECTURE.md`
- New file: `docs/DEPLOYMENT.md`
- New file: `docs/API.md`
- New file: `docs/CONTRIBUTING.md`
- New file: `docs/FAQ.md`
- New file: `docs/TROUBLESHOOTING.md`
- `README.md` (updates)
- `docs/SOCKS5_SETUP_GUIDE.md` (enhancements)

---

### Task 2.5: Add Health Check Endpoints

**Priority:** P1 - HIGH
**Time Estimate:** 2 hours
**Dependencies:** None

**Subtasks:**

- [ ] **2.5.1** Add `/health` endpoint in `proxy/src/main.go`:
  ```go
  if r.URL.Path == "/health" {
      s.handleHealth(w, r)
      return
  }
  ```
- [ ] **2.5.2** Implement `handleHealth`:
  ```go
  func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
      health := struct {
          Status     string    `json:"status"`
          Timestamp  time.Time `json:"timestamp"`
          Version    string    `json:"version"`
          Uptime     float64   `json:"uptime_seconds"`
          Goroutines int       `json:"goroutines"`
      }{
          Status:     "healthy",
          Timestamp:  time.Now(),
          Version:    os.Getenv("VERSION"),
          Uptime:     time.Since(startTime).Seconds(),
          Goroutines: runtime.NumGoroutine(),
      }
      w.Header().Set("Content-Type", "application/json")
      json.NewEncoder(w).Encode(health)
  }
  ```
- [ ] **2.5.3** Add `/ready` endpoint for readiness probe
- [ ] **2.5.4** Check dependencies in readiness check:
  - Celestial data loaded
  - Distance cache initialized
  - SOCKS5 listener active
- [ ] **2.5.5** Add startup time tracking (global variable)
- [ ] **2.5.6** Add version from environment variable
- [ ] **2.5.7** Add health check tests
- [ ] **2.5.8** Update Docker Compose with health checks:
  ```yaml
  healthcheck:
    test: ["CMD", "curl", "-f", "http://localhost/health"]
    interval: 30s
    timeout: 10s
    retries: 3
  ```
- [ ] **2.5.9** Update Kubernetes manifests (if applicable)
- [ ] **2.5.10** Document health endpoints

**Verification:**
```bash
# Should return healthy status
curl https://latency.space/health | jq .

# Should show ready
curl https://latency.space/ready | jq .
```

**Files Modified:**
- `proxy/src/main.go` (add endpoints)
- `docker-compose.yml` (add healthcheck)
- New file: `proxy/src/health_test.go`
- `docs/API.md` (document endpoints)

---

## Phase 3: Advanced Features (Week 4+)

### 🎯 Goal: Add unique features and polish

---

### Task 3.1: Enhanced Metrics and Monitoring

**Priority:** P2 - MEDIUM
**Time Estimate:** 4 hours
**Dependencies:** None

**Subtasks:**

- [ ] **3.1.1** Add cache hit/miss metrics:
  ```go
  cacheHits := prometheus.NewCounter(...)
  cacheMisses := prometheus.NewCounter(...)
  ```
- [ ] **3.1.2** Add occlusion event metrics:
  ```go
  occlusionRejections := prometheus.NewCounterVec(
      prometheus.CounterOpts{Name: "occlusion_rejections_total"},
      []string{"body", "occluder"},
  )
  ```
- [ ] **3.1.3** Add rate limiting rejection metrics
- [ ] **3.1.4** Add certificate expiration metric:
  ```go
  certExpiry := prometheus.NewGauge(...)
  // Set to days until expiration
  ```
- [ ] **3.1.5** Add goroutine count metric
- [ ] **3.1.6** Add memory usage metrics
- [ ] **3.1.7** Add HTTP status code distribution
- [ ] **3.1.8** Update Grafana dashboards with new metrics
- [ ] **3.1.9** Create alert rules for:
  - Certificate expiring soon (< 7 days)
  - High error rate (> 5%)
  - High goroutine count (> 1000)
  - Memory usage > 80%
- [ ] **3.1.10** Document metrics in `docs/MONITORING.md`

**Verification:**
```bash
# Check new metrics exist
curl http://localhost:9090/metrics | grep -E "cache|occlusion|cert_expiry"
```

**Files Modified:**
- `proxy/src/metrics.go` (add new metrics)
- `monitoring/grafana/dashboards/solar-system-latency.json` (update)
- `monitoring/prometheus/alerts.yml` (new rules)
- New file: `docs/MONITORING.md`

---

### Task 3.2: Multi-Hop Routing

**Priority:** P3 - LOW
**Time Estimate:** 8 hours
**Dependencies:** None

**Subtasks:**

- [ ] **3.2.1** Design multi-hop URL format:
  ```
  earth-mars-jupiter.latency.space:1080
  or
  mars+jupiter.latency.space:1080
  ```
- [ ] **3.2.2** Update `parseHostForCelestialBody` to handle multi-hop
- [ ] **3.2.3** Implement cumulative latency calculation:
  ```go
  totalLatency = latency(Earth→Mars) + latency(Mars→Jupiter) + latency(Jupiter→Destination)
  ```
- [ ] **3.2.4** Add multi-hop validation (max 5 hops?)
- [ ] **3.2.5** Add multi-hop to web demo UI
- [ ] **3.2.6** Add multi-hop metrics
- [ ] **3.2.7** Add multi-hop tests
- [ ] **3.2.8** Document multi-hop usage
- [ ] **3.2.9** Add visualization showing path

**Verification:**
```bash
# Should apply Earth→Mars + Mars→Jupiter latency
curl --socks5 earth-mars-jupiter.latency.space:1080 https://example.com
```

**Files Modified:**
- `proxy/src/main.go` (update parsing)
- `proxy/src/calculations.go` (multi-hop logic)
- `status/src/pages/Demo.jsx` (add multi-hop UI)
- New file: `proxy/src/multihop_test.go`
- `docs/SOCKS5_SETUP_GUIDE.md` (add multi-hop examples)

---

### Task 3.3: Custom Celestial Bodies API

**Priority:** P3 - LOW
**Time Estimate:** 12 hours
**Dependencies:** None

**Subtasks:**

- [ ] **3.3.1** Design custom body API:
  ```
  POST /api/custom-body
  {
    "name": "my-satellite",
    "type": "spacecraft",
    "orbital_elements": {...},
    "parent": "earth"
  }
  ```
- [ ] **3.3.2** Add validation for orbital parameters
- [ ] **3.3.3** Store custom bodies in database (SQLite?)
- [ ] **3.3.4** Load custom bodies on startup
- [ ] **3.3.5** Add authentication for creating custom bodies
- [ ] **3.3.6** Add UI for custom body creation
- [ ] **3.3.7** Add custom body listing endpoint
- [ ] **3.3.8** Add custom body deletion
- [ ] **3.3.9** Add custom body sharing (export/import JSON)
- [ ] **3.3.10** Add gallery of community custom bodies
- [ ] **3.3.11** Add tests for custom bodies
- [ ] **3.3.12** Document custom body API

**Verification:**
```bash
# Create custom body
curl -X POST https://latency.space/api/custom-body \
  -H "Content-Type: application/json" \
  -d '{"name": "ISS", "type": "spacecraft", ...}'

# Use custom body
curl --socks5 iss.latency.space:1080 https://example.com
```

**Files Modified:**
- New file: `proxy/src/custom_bodies.go`
- New file: `proxy/src/custom_bodies_test.go`
- `proxy/src/main.go` (add API routes)
- `status/src/pages/CustomBodies.jsx` (new page)
- New file: `docs/CUSTOM_BODIES.md`

---

### Task 3.4: Historical Data and Visualization

**Priority:** P3 - LOW
**Time Estimate:** 16 hours
**Dependencies:** None

**Subtasks:**

- [ ] **3.4.1** Set up TimescaleDB or InfluxDB for time-series data
- [ ] **3.4.2** Add periodic job to record celestial positions every hour
- [ ] **3.4.3** Store: timestamp, body name, distance, latency, position (x,y,z)
- [ ] **3.4.4** Add API endpoint for historical queries:
  ```
  GET /api/history?body=mars&from=2025-01-01&to=2025-12-31
  ```
- [ ] **3.4.5** Create visualization page showing:
  - Distance over time (line graph)
  - Latency over time (line graph)
  - Orbital position (3D or 2D plot)
- [ ] **3.4.6** Add "replay" feature to see past configurations
- [ ] **3.4.7** Add comparison view (multiple bodies on same graph)
- [ ] **3.4.8** Add export to CSV functionality
- [ ] **3.4.9** Add sharing of historical views (permalink)
- [ ] **3.4.10** Add caching for historical queries
- [ ] **3.4.11** Add data retention policy (keep 1 year?)
- [ ] **3.4.12** Document historical data API

**Verification:**
```bash
# Get Mars distance history for 2025
curl "https://latency.space/api/history?body=mars&from=2025-01-01&to=2025-12-31"
```

**Files Modified:**
- New file: `proxy/src/history.go`
- New file: `docker-compose.timescaledb.yml`
- New file: `status/src/pages/History.jsx`
- New file: `proxy/src/jobs/record_positions.go`
- New file: `docs/HISTORICAL_DATA.md`

---

### Task 3.5: Rate Limiting Implementation

**Priority:** P1 - HIGH (moved from P2)
**Time Estimate:** 4 hours
**Dependencies:** None

**Subtasks:**

- [ ] **3.5.1** Add rate limiting dependency:
  ```bash
  go get golang.org/x/time/rate
  ```
- [ ] **3.5.2** Implement rate limiter in `proxy/src/security.go`:
  ```go
  type RateLimiter struct {
      limiters sync.Map // map[string]*rate.Limiter
      rate     rate.Limit
      burst    int
  }
  ```
- [ ] **3.5.3** Add rate limiting to `IsAllowedIP`:
  ```go
  func (rl *RateLimiter) Allow(ip string) bool {
      limiter := rl.getLimiter(ip)
      return limiter.Allow()
  }
  ```
- [ ] **3.5.4** Make rate limits configurable via environment:
  ```
  RATE_LIMIT_REQUESTS_PER_SECOND=10
  RATE_LIMIT_BURST=20
  ```
- [ ] **3.5.5** Add rate limit metrics (rejections counter)
- [ ] **3.5.6** Add rate limit HTTP headers:
  ```
  X-RateLimit-Limit: 10
  X-RateLimit-Remaining: 7
  X-RateLimit-Reset: 1234567890
  ```
- [ ] **3.5.7** Return 429 status for rate limited requests
- [ ] **3.5.8** Add rate limiting tests
- [ ] **3.5.9** Add rate limiting to SOCKS5 connections
- [ ] **3.5.10** Document rate limits in API docs

**Verification:**
```bash
# Should get rate limited after burst
for i in {1..100}; do curl https://latency.space/api/status-data; done
# Some should return 429
```

**Files Modified:**
- `proxy/src/security.go` (implement rate limiting)
- `proxy/src/main.go` (apply rate limiting)
- `proxy/src/socks.go` (apply to SOCKS5)
- New file: `proxy/src/security_test.go` (rate limit tests)
- `go.mod` (add dependency)
- `docs/API.md` (document rate limits)

---

## Phase 4: Polish and Launch (Week 5)

### 🎯 Goal: Production-ready and launched

---

### Task 4.1: Security Audit

**Priority:** P1 - HIGH
**Time Estimate:** 4 hours
**Dependencies:** All previous tasks

**Subtasks:**

- [ ] **4.1.1** Run security scanner on codebase:
  ```bash
  gosec ./...
  ```
- [ ] **4.1.2** Fix any critical/high severity issues found
- [ ] **4.1.3** Review allowed hosts whitelist - ensure it's appropriate
- [ ] **4.1.4** Review rate limiting settings - ensure anti-abuse
- [ ] **4.1.5** Check for hardcoded secrets (use `git-secrets`)
- [ ] **4.1.6** Review CORS settings - ensure not too permissive
- [ ] **4.1.7** Add security headers:
  ```go
  w.Header().Set("X-Content-Type-Options", "nosniff")
  w.Header().Set("X-Frame-Options", "DENY")
  w.Header().Set("X-XSS-Protection", "1; mode=block")
  ```
- [ ] **4.1.8** Review input validation thoroughly
- [ ] **4.1.9** Check for SQL injection risks (if using DB)
- [ ] **4.1.10** Review Docker security (non-root user, minimal image)
- [ ] **4.1.11** Set up Dependabot for dependency updates
- [ ] **4.1.12** Document security considerations

**Verification:**
- [ ] gosec shows no critical issues
- [ ] Security headers present
- [ ] No secrets in code

**Files Modified:**
- `proxy/src/main.go` (add security headers)
- `.github/dependabot.yml` (new file)
- New file: `SECURITY.md`

---

### Task 4.2: Performance Testing

**Priority:** P1 - HIGH
**Time Estimate:** 6 hours
**Dependencies:** All Phase 1-3 tasks

**Subtasks:**

- [ ] **4.2.1** Set up load testing tool (k6, wrk, or similar)
- [ ] **4.2.2** Create load test scenarios:
  - Concurrent API requests
  - Concurrent SOCKS5 connections
  - Mixed traffic patterns
- [ ] **4.2.3** Run load tests and record metrics:
  ```bash
  k6 run load-test.js
  ```
- [ ] **4.2.4** Identify bottlenecks (CPU, memory, network?)
- [ ] **4.2.5** Optimize hot paths if needed
- [ ] **4.2.6** Test connection pool sizing
- [ ] **4.2.7** Test cache effectiveness
- [ ] **4.2.8** Measure API response times under load
- [ ] **4.2.9** Test SOCKS5 throughput
- [ ] **4.2.10** Document performance characteristics
- [ ] **4.2.11** Set up continuous performance monitoring
- [ ] **4.2.12** Create performance dashboard in Grafana

**Verification:**
- [ ] Can handle 100 concurrent SOCKS5 connections
- [ ] API responds in < 100ms under normal load
- [ ] Memory usage stays under 512MB
- [ ] CPU usage stays under 50% average

**Files Modified:**
- New file: `tests/load/k6-test.js`
- New file: `docs/PERFORMANCE.md`
- `monitoring/grafana/dashboards/performance.json` (new)

---

### Task 4.3: Create Demo Video

**Priority:** P2 - MEDIUM
**Time Estimate:** 4 hours
**Dependencies:** Task 2.1

**Subtasks:**

- [ ] **4.3.1** Write video script covering:
  - What is latency.space?
  - Why does it exist?
  - SOCKS5 setup demo (browser)
  - Web demo usage
  - Real-time latency visualization
  - Use cases (education, testing)
- [ ] **4.3.2** Record screen capture:
  - Configure Firefox with SOCKS5
  - Visit example.com with Mars latency
  - Show comparison with/without latency
  - Use web demo interface
  - Show API usage
- [ ] **4.3.3** Record voiceover
- [ ] **4.3.4** Edit video (add titles, transitions)
- [ ] **4.3.5** Add background music (ensure licensing)
- [ ] **4.3.6** Export in multiple formats (1080p, 720p)
- [ ] **4.3.7** Create GIF from highlight (< 10MB for README)
- [ ] **4.3.8** Upload to YouTube
- [ ] **4.3.9** Add to README.md and landing page
- [ ] **4.3.10** Create shorter clips for social media

**Verification:**
- [ ] Video is clear and understandable
- [ ] Audio quality is good
- [ ] Demonstrates key features
- [ ] Under 5 minutes

**Files Modified:**
- `README.md` (embed video)
- `status/src/pages/Landing.jsx` (embed video)
- New file: `media/demo.mp4`
- New file: `media/demo.gif`

---

### Task 4.4: Write Launch Blog Post

**Priority:** P2 - MEDIUM
**Time Estimate:** 3 hours
**Dependencies:** None

**Subtasks:**

- [ ] **4.4.1** Outline blog post structure:
  - Introduction - the problem
  - Solution - latency.space
  - How it works (brief technical overview)
  - Features
  - Use cases
  - Demo
  - Getting started
  - Call to action
- [ ] **4.4.2** Write draft
- [ ] **4.4.3** Add screenshots and diagrams
- [ ] **4.4.4** Add code examples
- [ ] **4.4.5** Edit for clarity and brevity
- [ ] **4.4.6** Get feedback from others
- [ ] **4.4.7** Finalize and proofread
- [ ] **4.4.8** Publish to blog platform (Medium, Dev.to, personal blog)
- [ ] **4.4.9** Submit to Hacker News
- [ ] **4.4.10** Share on Reddit (r/programming, r/networking, r/space)
- [ ] **4.4.11** Share on Twitter/X
- [ ] **4.4.12** Share on LinkedIn

**Verification:**
- [ ] Blog post is published
- [ ] Shared on social media
- [ ] Links work

**Files Modified:**
- New file: `blog/launch-post.md`

---

### Task 4.5: Final Documentation Review

**Priority:** P1 - HIGH
**Time Estimate:** 3 hours
**Dependencies:** All previous docs

**Subtasks:**

- [ ] **4.5.1** Review README.md for accuracy
- [ ] **4.5.2** Check all links work (use link checker)
- [ ] **4.5.3** Ensure all code examples are tested
- [ ] **4.5.4** Fix any typos or grammatical errors
- [ ] **4.5.5** Ensure consistent formatting across docs
- [ ] **4.5.6** Add table of contents to long docs
- [ ] **4.5.7** Add "Edit on GitHub" links
- [ ] **4.5.8** Ensure all images load correctly
- [ ] **4.5.9** Add LICENSE file if not present
- [ ] **4.5.10** Add CODE_OF_CONDUCT.md
- [ ] **4.5.11** Add CONTRIBUTING.md with detailed guidelines
- [ ] **4.5.12** Create documentation website (GitHub Pages, Docusaurus, etc.)

**Verification:**
- [ ] All docs are accurate
- [ ] No broken links
- [ ] Examples work
- [ ] Consistent style

**Files Modified:**
- All `docs/*.md` files (review and fix)
- `README.md` (final polish)
- New file: `LICENSE`
- New file: `CODE_OF_CONDUCT.md`
- Updated: `CONTRIBUTING.md`

---

### Task 4.6: Launch Checklist

**Priority:** P0 - CRITICAL
**Time Estimate:** 2 hours
**Dependencies:** All previous tasks

**Pre-Launch Checklist:**

- [ ] **4.6.1** All Phase 1 tasks completed ✅
- [ ] **4.6.2** All Phase 2 critical tasks completed ✅
- [ ] **4.6.3** SOCKS5 proxy working end-to-end ✅
- [ ] **4.6.4** SSL certificates valid and auto-renewing ✅
- [ ] **4.6.5** API returns accurate data ✅
- [ ] **4.6.6** Web demo functional ✅
- [ ] **4.6.7** Documentation complete and accurate ✅
- [ ] **4.6.8** Health checks passing ✅
- [ ] **4.6.9** Monitoring dashboards configured ✅
- [ ] **4.6.10** Alerts configured ✅
- [ ] **4.6.11** Rate limiting active ✅
- [ ] **4.6.12** Security audit passed ✅
- [ ] **4.6.13** Performance tests passed ✅
- [ ] **4.6.14** Backup procedures documented ✅
- [ ] **4.6.15** Rollback plan ready ✅
- [ ] **4.6.16** Demo video created ✅
- [ ] **4.6.17** Blog post ready ✅
- [ ] **4.6.18** Social media posts scheduled ✅

**Launch Day:**

- [ ] **4.6.19** Final smoke test of all features
- [ ] **4.6.20** Check monitoring dashboards
- [ ] **4.6.21** Publish blog post
- [ ] **4.6.22** Submit to Hacker News
- [ ] **4.6.23** Post on Reddit
- [ ] **4.6.24** Tweet announcement
- [ ] **4.6.25** Monitor for issues
- [ ] **4.6.26** Respond to comments/questions
- [ ] **4.6.27** Celebrate! 🎉

---

## Tracking Progress

### How to Use This Plan

1. **Copy to GitHub Issues:**
   - Create one issue per major task
   - Use checkboxes for subtasks
   - Label with priority (P0, P1, P2, P3)
   - Assign to team members

2. **Track in Project Board:**
   - Columns: Todo, In Progress, Review, Done
   - Move tasks as you work
   - Update estimates

3. **Daily Standup:**
   - What did I complete yesterday?
   - What am I working on today?
   - Any blockers?

4. **Weekly Review:**
   - Are we on track?
   - Need to reprioritize?
   - Celebrate completed tasks!

---

## Quick Reference: Priority Tasks

### Must Do This Week (P0):
- [ ] Task 1.1: Fix Cloudflare Configuration
- [ ] Task 1.2: Deploy Simplified SSL Certificate
- [ ] Task 1.3: Test SOCKS5 Proxy Functionality
- [ ] Task 1.4: Update README.md - Emphasize SOCKS5
- [ ] Task 1.7: Verify End-to-End Functionality
- [ ] Task 4.6: Launch Checklist

### Should Do This Week (P1):
- [ ] Task 1.5: Create SOCKS5 Setup Guide
- [ ] Task 1.6: Clean Up Codebase
- [ ] Task 2.1: Create Web Demo Interface
- [ ] Task 2.2: Implement HTTP Proxy API
- [ ] Task 2.4: Enhanced Documentation
- [ ] Task 2.5: Add Health Check Endpoints
- [ ] Task 3.5: Rate Limiting Implementation

### Nice to Have (P2):
- [ ] Task 2.3: Create Browser Extension
- [ ] Task 3.1: Enhanced Metrics and Monitoring
- [ ] Task 4.3: Create Demo Video
- [ ] Task 4.4: Write Launch Blog Post

### Future (P3):
- [ ] Task 3.2: Multi-Hop Routing
- [ ] Task 3.3: Custom Celestial Bodies API
- [ ] Task 3.4: Historical Data and Visualization

---

## Success Metrics

### Week 1 Goals:
- [ ] SOCKS5 accessible and working
- [ ] At least 10 test connections through Mars
- [ ] Documentation updated
- [ ] No SSL errors on info pages

### Week 2 Goals:
- [ ] Web demo live
- [ ] HTTP API functional
- [ ] 50+ page views on documentation
- [ ] Browser extension alpha (optional)

### Week 3 Goals:
- [ ] All documentation complete
- [ ] Performance testing done
- [ ] Security audit passed
- [ ] Ready for launch

### Launch Goals:
- [ ] 100+ visitors on launch day
- [ ] 10+ SOCKS5 users
- [ ] Hacker News front page (dream goal!)
- [ ] Positive feedback from users

---

## Contact & Support

**If you get stuck:**
1. Check the troubleshooting docs
2. Review this implementation plan
3. Check existing GitHub issues
4. Create new issue with details

**Questions about priorities:**
1. Focus on P0 tasks first
2. P1 tasks are important for UX
3. P2/P3 can wait until after launch

**Remember:**
- Progress > Perfection
- Ship early, iterate often
- Get feedback from real users
- Celebrate small wins!

---

## Appendix: Command Reference

### Quick Commands

```bash
# Fix Cloudflare DNS
# (Done via dashboard)

# Deploy SSL cert
certbot certonly --dns-cloudflare \
  --dns-cloudflare-credentials ~/.secrets/cloudflare.ini \
  -d latency.space -d *.latency.space

# Test SOCKS5
curl --socks5 mars.latency.space:1080 https://example.com

# Check health
curl https://latency.space/health | jq .

# View metrics
curl http://localhost:9090/metrics | grep latency_space

# Run tests
cd proxy/src && go test -v ./...

# Clean up .bak files
find . -name "*.bak" -delete

# Fix go.mod
cd proxy/src && go mod tidy

# Build Docker images
docker compose build

# Deploy
docker compose up -d

# View logs
docker compose logs -f proxy

# Restart service
docker compose restart proxy
```

---

**Let's ship this! 🚀**
