# Design Feasibility Analysis - latency.space

**Date:** 2025-11-05
**Author:** Architecture Review
**Status:** ⚠️ CRITICAL DESIGN CONSTRAINTS IDENTIFIED

---

## Executive Summary

After comprehensive analysis of the subdomain-based routing design and real-world SSL/TLS constraints, **the current design has fundamental limitations that cannot be fully resolved**. The project must choose between:

1. **Partial HTTPS support** (only 1-level subdomains)
2. **HTTP-only operation** (security/browser concerns)
3. **Architectural redesign** (different routing mechanism)

This document evaluates each approach and provides a recommended path forward.

---

## Current Design Intent

The application uses DNS subdomains to route proxy traffic:

```
Pattern 1: mars.latency.space
           └── Access Mars info page

Pattern 2: phobos.mars.latency.space
           └── Access Mars moon Phobos

Pattern 3: example.com.mars.latency.space
           └── Proxy example.com through Mars latency

Pattern 4: example.com.phobos.mars.latency.space
           └── Proxy example.com through Phobos latency
```

**Design Goal:** Elegant, self-documenting URLs where the subdomain encodes routing information.

**Implementation:** `parseHostForCelestialBody()` extracts celestial body from Host header and applies latency.

---

## SSL/TLS Certificate Constraints (DEFINITIVE)

### ❌ What Does NOT Work

Based on industry-wide SSL/TLS standards and CA policies:

1. **Multi-level wildcards are NOT supported:**
   ```
   Certificate: *.*.latency.space
   Status: ❌ NO CA will issue this
   Reason: Not part of RFC 6125 specification
   Browser Support: 0%
   ```

2. **Single wildcard only covers ONE level:**
   ```
   Certificate: *.latency.space
   Covers:
     ✅ mars.latency.space
     ❌ phobos.mars.latency.space
     ❌ example.com.mars.latency.space
   ```

3. **Let's Encrypt limitations:**
   ```bash
   # This command from README.md:
   certbot certonly --standalone \
     -d latency.space \
     -d *.latency.space \
     -d *.*.latency.space  # ❌ WILL FAIL

   Error: "multiple wildcards are not allowed"
   ```

### ✅ What DOES Work

1. **Single-level wildcard:**
   ```
   Certificate: *.latency.space
   Covers:
     ✅ mars.latency.space
     ✅ jupiter.latency.space
     ✅ voyager1.latency.space
   ```

2. **Multiple single-level wildcards (SAN certificate):**
   ```
   Certificate with Subject Alternative Names:
     - latency.space
     - *.latency.space
     - *.mars.latency.space
     - *.jupiter.latency.space
     - *.earth.latency.space
     ... (enumerate all ~50 celestial bodies)

   Covers:
     ✅ latency.space
     ✅ mars.latency.space
     ✅ phobos.mars.latency.space
     ❌ example.com.mars.latency.space (still 3+ levels)
   ```

3. **Individual certificates per pattern:**
   ```
   Cert 1: *.latency.space
   Cert 2: *.mars.latency.space
   Cert 3: *.jupiter.latency.space
   ...

   Problems:
     - Need ~50+ certificates (planets, moons, asteroids, spacecraft)
     - Let's Encrypt rate limits: 50 certs/week per domain
     - Management overhead
     - Still can't cover example.com.mars.latency.space
   ```

### 🚫 The Fundamental Problem

**No SSL/TLS solution exists** for patterns like:
- `example.com.mars.latency.space` (arbitrary target + celestial body)
- `api.github.com.jupiter.latency.space`
- `www.google.com.phobos.mars.latency.space`

These require **3+ subdomain levels** which cannot be covered by wildcard certificates.

---

## Technical Deep Dive: Why This Fails

### SSL/TLS Matching Rules (RFC 6125)

When a browser connects to `example.com.mars.latency.space`:

```
1. Browser receives certificate from server
2. Certificate contains: *.mars.latency.space
3. Browser checks: Does "example.com.mars.latency.space" match "*.mars.latency.space"?
4. Matching rule: * replaces exactly ONE label (between dots)

   *.mars.latency.space expands to:
     ✅ phobos.mars.latency.space (one label: "phobos")
     ❌ example.com.mars.latency.space (two labels: "example" + "com")

5. Result: Certificate validation FAILS
6. Browser shows: "NET::ERR_CERT_COMMON_NAME_INVALID"
```

### Cloudflare Complication

When using Cloudflare proxy mode:

```
                    ┌─────────────┐
                    │   Browser   │
                    └──────┬──────┘
                           │ HTTPS (Cloudflare's cert)
                           ▼
                    ┌─────────────┐
                    │ Cloudflare  │ ← Terminates SSL here
                    │   Proxy     │
                    └──────┬──────┘
                           │ HTTP/HTTPS (to origin)
                           ▼
                    ┌─────────────┐
                    │ Your Proxy  │
                    │   Server    │
                    └─────────────┘

Problems:
  1. Cloudflare terminates SSL, so your cert config doesn't matter
  2. Cloudflare's Universal SSL: *.latency.space (1 level only)
  3. Advanced certs (multi-level) require paid plan + manual config
  4. Even with paid plan, can't cover arbitrary subdomains
  5. Cloudflare doesn't understand proxy routing logic
  6. Port 1080 (SOCKS5) not proxied through Cloudflare
```

---

## Design Pattern Analysis

### Pattern Feasibility Matrix

| Pattern | Example | Levels | HTTPS (Wildcard) | HTTPS (SAN) | HTTP | SOCKS5 |
|---------|---------|--------|------------------|-------------|------|--------|
| **Info Page** | `mars.latency.space` | 1 | ✅ | ✅ | ✅ | N/A |
| **Moon Info** | `phobos.mars.latency.space` | 2 | ❌ | ✅ (with SAN) | ✅ | N/A |
| **Proxy-Through** | `example.com.mars.latency.space` | 3+ | ❌ | ❌ | ✅ | ✅ |
| **Moon Proxy** | `ex.com.phobos.mars.latency.space` | 4+ | ❌ | ❌ | ✅ | ✅ |

**Legend:**
- ✅ Technically possible
- ❌ Not possible with standard SSL/TLS
- N/A - Not applicable to this pattern

### What Actually Works Today

#### ✅ SOCKS5 Proxy (Recommended Primary Interface)

```bash
# Works perfectly - no SSL at proxy level
curl --socks5 mars.latency.space:1080 https://example.com

# Client handles SSL end-to-end
# Proxy just adds latency and forwards
# Works with ALL patterns
```

**Status:**
- ✅ Design is sound
- ✅ Code works correctly
- ❌ Currently blocked by Cloudflare (port 1080 not proxied)
- ✅ Works with DNS-only routing

**Requirements:**
1. DNS: `*.latency.space` → DNS-only (not Cloudflare proxied)
2. Firewall: Port 1080 open
3. No SSL certificate needed for SOCKS5

---

#### ⚠️ HTTP Proxy (1-Level Subdomains Only)

```bash
# Works with HTTPS
https://mars.latency.space/

# Can be configured to:
https://mars.latency.space/?url=https://example.com
https://mars.latency.space/proxy/https://example.com
```

**Status:**
- ✅ Info pages work
- ✅ Can implement URL parameter routing
- ⚠️ Less elegant than subdomain routing
- ✅ Full HTTPS support

---

#### ❌ HTTP Proxy (Multi-Level Subdomains)

```bash
# Cannot work with HTTPS
https://example.com.mars.latency.space/  ❌

# Could work with HTTP
http://example.com.mars.latency.space/   ⚠️
```

**Status:**
- ❌ HTTPS not possible (SSL constraint)
- ⚠️ HTTP possible but problematic in 2025
- ⚠️ Modern browsers increasingly block HTTP
- ⚠️ Mixed content warnings
- ❌ Not secure for real traffic

---

## Why You've Been Flip-Flopping

Based on your comment about flip-flopping between wildcards and specific names, here's what's likely happened:

### Iteration 1: "Let's use wildcards!"
```bash
certbot certonly -d *.latency.space
```
**Result:**
- ✅ `mars.latency.space` works
- ❌ `phobos.mars.latency.space` SSL errors
- **Decision:** Need more wildcards...

### Iteration 2: "Let's use multi-level wildcards!"
```bash
certbot certonly -d *.latency.space -d *.*.latency.space
```
**Result:**
- ❌ `Error: multiple wildcards not allowed`
- **Decision:** Let's try specific names...

### Iteration 3: "Let's enumerate all subdomains!"
```bash
certbot certonly \
  -d mars.latency.space \
  -d phobos.mars.latency.space \
  -d deimos.mars.latency.space \
  -d jupiter.latency.space \
  -d io.jupiter.latency.space \
  ... (need 50+ domains)
```
**Result:**
- ✅ Works for enumerated domains
- ❌ Can't cover `example.com.mars.latency.space`
- ❌ Maintenance nightmare
- ❌ New celestial bodies require new cert
- **Decision:** Back to wildcards...

### Iteration 4: "Let's use Cloudflare!"
```bash
# Use Cloudflare for SSL termination
```
**Result:**
- ✅ SSL works everywhere
- ❌ Proxy functionality breaks
- ❌ SOCKS5 not accessible
- **Decision:** Turn off Cloudflare proxy...

### Iteration 5: "DNS-only mode?"
```bash
# Cloudflare DNS but not proxied
```
**Result:**
- ✅ Proxy logic works
- ❌ Back to SSL wildcard problems
- **Decision:** Flip-flop continues...

### The Real Issue

**You're hitting a fundamental SSL/TLS limitation, not a configuration problem.**

No amount of cert shuffling will make `example.com.mars.latency.space` work with HTTPS using standard certificates.

---

## Viable Design Options

### Option A: SOCKS5-Primary Design ⭐ RECOMMENDED

**Approach:** Focus on SOCKS5 as primary interface, HTTP for info pages only.

**Architecture:**
```
1. Info Pages (HTTPS):
   - https://latency.space/ (main site)
   - https://mars.latency.space/ (celestial info)
   - https://phobos.mars.latency.space/ (moon info) [with SAN cert]

2. Proxy Traffic (SOCKS5):
   - socks5://mars.latency.space:1080
   - socks5://jupiter.latency.space:1080
   - No SSL at proxy layer (end-to-end encryption to destination)

3. Documentation:
   - Emphasize SOCKS5 usage
   - Provide browser/app configuration guides
   - Docker image with pre-configured proxy settings
```

**SSL Certificate:**
```bash
# Single cert with SANs for info pages
certbot certonly --dns-cloudflare \
  -d latency.space \
  -d *.latency.space \
  -d *.mars.latency.space \
  -d *.jupiter.latency.space \
  -d *.earth.latency.space \
  -d *.saturn.latency.space \
  ... (enumerate major bodies)
```

**Pros:**
- ✅ SOCKS5 works perfectly with design intent
- ✅ No SSL limitations for proxy traffic
- ✅ Info pages have proper HTTPS
- ✅ Most flexible for users (browser, CLI, apps)
- ✅ Standard protocol (SOCKS5 RFC 1928)

**Cons:**
- ⚠️ Requires client configuration (not click-and-go)
- ⚠️ Less discovery than HTTP subdomain routing
- ⚠️ Users must understand SOCKS5

**Deployment:**
- DNS: `*.latency.space` → DNS-only (not Cloudflare proxied)
- Firewall: Port 1080 open
- Certificate: Let's Encrypt with DNS challenge (for SAN)

**User Experience:**
```bash
# Firefox/Chrome proxy settings:
SOCKS Host: mars.latency.space
Port: 1080
SOCKS v5: ✓

# Command line:
curl --socks5 mars.latency.space:1080 https://example.com
ssh -o "ProxyCommand=nc -X 5 -x jupiter.latency.space:1080 %h %p" server.com

# Docker:
docker run --network host -e http_proxy=socks5://mars.latency.space:1080 myapp
```

---

### Option B: Path-Based Routing

**Approach:** Use URL paths instead of subdomains for routing.

**Architecture:**
```
HTTPS Endpoints:
  - https://latency.space/
  - https://latency.space/mars/
  - https://latency.space/mars/proxy?url=https://example.com
  - https://latency.space/phobos@mars/proxy?url=https://example.com

OR with cleaner syntax:
  - https://latency.space/via/mars/https://example.com
  - https://latency.space/via/phobos.mars/https://example.com
```

**SSL Certificate:**
```bash
# Single cert for main domain
certbot certonly -d latency.space -d www.latency.space
```

**Pros:**
- ✅ Full HTTPS support
- ✅ Single certificate needed
- ✅ No subdomain limitations
- ✅ Easy to deploy with Cloudflare
- ✅ RESTful API design

**Cons:**
- ❌ Less elegant than subdomain routing
- ❌ Breaks existing URLs/bookmarks
- ❌ Requires rewriting `parseHostForCelestialBody()` logic
- ⚠️ More complex URL parsing

**Code Impact:**
- Major: Rewrite routing logic
- Major: Update all documentation
- Medium: Update tests
- Major: Breaking change for existing users

---

### Option C: Hybrid Approach ⭐ BEST UX

**Approach:** Combine multiple access methods for different use cases.

**Architecture:**
```
1. Info Pages (HTTPS with subdomains):
   - https://mars.latency.space/ (info page)
   - Uses wildcard cert: *.latency.space

2. SOCKS5 Proxy (Primary for actual proxying):
   - socks5://mars.latency.space:1080
   - Supports all patterns without SSL issues

3. HTTP API for programmatic access:
   - https://latency.space/api/proxy
   - POST with body: {"via": "mars", "url": "https://example.com"}
   - Returns proxied content with latency

4. Web-based demo interface:
   - https://latency.space/demo
   - Interactive UI to test latency from different bodies
   - Fetches via API, no SSL routing issues
```

**SSL Certificate:**
```bash
# Main domain + 1-level wildcard
certbot certonly --dns-cloudflare \
  -d latency.space \
  -d *.latency.space
```

**Pros:**
- ✅ Info pages remain elegant
- ✅ SOCKS5 handles actual proxying
- ✅ HTTP API for web interfaces
- ✅ Multiple entry points for different use cases
- ✅ Minimal breaking changes
- ✅ Can use Cloudflare for main site

**Cons:**
- ⚠️ More complex architecture
- ⚠️ Need to document multiple access methods
- ⚠️ Slightly more code to maintain

**Deployment Strategy:**
```
Cloudflare DNS:
  - latency.space → Proxied ☁️ (main site, info pages, API)
  - *.latency.space → DNS Only 🌐 (SOCKS5 access)

OR separate subdomain:
  - latency.space → Proxied ☁️
  - www.latency.space → Proxied ☁️
  - proxy.latency.space → DNS Only 🌐
  - *.proxy.latency.space → DNS Only 🌐

Usage:
  - Info: https://mars.latency.space/
  - Proxy: socks5://mars.proxy.latency.space:1080
```

---

### Option D: HTTP-Only Proxy (Not Recommended)

**Approach:** Accept HTTP-only for proxy-through patterns.

**Architecture:**
```
HTTPS:
  - https://latency.space/ (main site)
  - https://mars.latency.space/ (info pages)

HTTP (proxy-through):
  - http://example.com.mars.latency.space/
  - http://api.github.com.jupiter.latency.space/
```

**Pros:**
- ✅ Works with current design as-is
- ✅ Elegant subdomain routing preserved

**Cons:**
- ❌ Mixed HTTP/HTTPS confusing for users
- ❌ Modern browsers block/warn on HTTP
- ❌ Insecure for real traffic (MITM attacks)
- ❌ Many sites force HTTPS upgrade (HSTS)
- ❌ Corporate firewalls often block HTTP
- ❌ Poor user experience in 2025

**Verdict:** Not recommended for production.

---

## Recommended Path Forward

### Phase 1: Immediate Fix (Week 1)

**Goal:** Get core functionality working

1. **Fix Cloudflare Configuration**
   ```
   DNS Settings:
     latency.space → Proxied (for main site)
     *.latency.space → DNS-only (for proxy access)
   ```

2. **Deploy SOCKS5 as Primary Interface**
   ```bash
   # Ensure port 1080 is accessible
   # Document SOCKS5 usage prominently
   # Create browser configuration guide
   ```

3. **Simplify SSL Strategy**
   ```bash
   # Single wildcard for info pages
   certbot certonly --dns-cloudflare \
     -d latency.space \
     -d *.latency.space

   # Document that proxy-through requires SOCKS5
   ```

4. **Update Documentation**
   ```markdown
   # Primary Usage: SOCKS5 Proxy
   Works with ALL patterns, no SSL limitations

   # Secondary: Info Pages (HTTPS)
   View celestial body information

   # Not Supported: HTTPS proxy-through subdomains
   Technical limitation of SSL/TLS standard
   Use SOCKS5 instead
   ```

---

### Phase 2: Enhanced UX (Month 1)

1. **Add Web-Based Demo**
   - Interactive UI at `https://latency.space/demo`
   - Select celestial body from dropdown
   - Enter target URL
   - Fetch via backend API (avoids SSL issues)
   - Show latency visualization

2. **Create Configuration Tools**
   - Browser extension for easy SOCKS5 setup
   - Docker image with proxy pre-configured
   - CLI tool: `latency-space config browser mars`

3. **Enhanced Documentation**
   - Video tutorials
   - Step-by-step guides for major browsers
   - Troubleshooting common issues

---

### Phase 3: Advanced Features (Month 2-3)

1. **Multi-Hop Routing**
   ```bash
   # Route through multiple bodies
   curl --socks5 earth-mars-jupiter.latency.space:1080 https://example.com

   # Total latency: Earth→Mars + Mars→Jupiter
   ```

2. **Custom Latency Profiles**
   ```bash
   # API to create custom celestial bodies
   POST /api/custom-body
   {
     "name": "my-satellite",
     "orbit": {...},
     "parent": "earth"
   }
   ```

3. **Telemetry Dashboard**
   - Real-time latency graphs
   - Historical position data
   - Usage statistics

---

## Design Decision: Final Recommendation

### ⭐ Choose Option C: Hybrid Approach

**Rationale:**
1. **SOCKS5** solves the SSL problem elegantly - it's what proxies are designed for
2. **Info pages** remain beautiful and discoverable via HTTPS
3. **API** enables web interfaces without SSL routing constraints
4. **Multiple entry points** serve different user needs
5. **Minimal breaking changes** - SOCKS5 already implemented
6. **Future-proof** - can add more interfaces without redesign

### Implementation Priority

```
P0 (This Week):
  ✅ Fix Cloudflare DNS to DNS-only for *.latency.space
  ✅ Verify SOCKS5 works end-to-end
  ✅ Update docs to emphasize SOCKS5
  ✅ Deploy single wildcard cert for info pages

P1 (Next 2 Weeks):
  ✅ Create SOCKS5 setup guides
  ✅ Add web demo interface
  ✅ HTTP API for programmatic proxying
  ✅ Browser extension for easy configuration

P2 (Month 2):
  ⚠️ Enhanced documentation
  ⚠️ Video tutorials
  ⚠️ Advanced features

P3 (Future):
  🔮 Multi-hop routing
  🔮 Custom celestial bodies
  🔮 Historical data
```

---

## What To Stop Doing

### ❌ Stop Trying to Fix SSL for Multi-Level Subdomains

**Reality:** It's not possible with standard SSL/TLS. Period.

No amount of:
- ❌ Wildcard shuffling
- ❌ Certificate authority changes
- ❌ Cloudflare configuration
- ❌ OpenSSL tweaking

...will make `example.com.mars.latency.space` work with HTTPS.

**Accept:** This is a limitation of the SSL/TLS specification (RFC 6125), not your implementation.

---

### ❌ Stop Flip-Flopping on Certificate Strategy

**Choose One:**

**For Info Pages:**
```bash
# Single wildcard is sufficient
certbot certonly --dns-cloudflare -d latency.space -d *.latency.space
```

**For Proxy Traffic:**
```
Use SOCKS5 (no certificate needed)
```

**Done.** Stick with it.

---

### ❌ Stop Using Cloudflare Proxy Mode for Proxy Subdomains

**Why:** Cloudflare proxy fundamentally conflicts with your proxy logic.

**Solution:**
```
Cloudflare DNS Settings:
  latency.space        → Proxied ☁️ (CDN benefits)
  www.latency.space    → Proxied ☁️
  *.latency.space      → DNS Only 🌐 (direct to your server)
```

This way:
- Main site gets CDN/DDoS protection
- Proxy subdomains connect directly
- SOCKS5 port accessible
- Proxy logic works

---

## Testing the Recommended Design

### Test 1: SOCKS5 Proxy (Should Work)

```bash
# Set DNS to DNS-only for *.latency.space

# Test SOCKS5 connection
curl --socks5 mars.latency.space:1080 https://example.com

# Expected:
#   - Connection succeeds
#   - ~40 minute delay (Mars RTT)
#   - Returns example.com content
```

### Test 2: Info Page (Should Work)

```bash
# Test info page with HTTPS
curl https://mars.latency.space/

# Expected:
#   - SSL valid (wildcard cert)
#   - Returns Mars info page
#   - No latency delay for info page
```

### Test 3: What Won't Work (And That's OK)

```bash
# This will never work with HTTPS
curl https://example.com.mars.latency.space/

# Expected: SSL error
# Solution: Use SOCKS5 instead
```

---

## Conclusion

### Can The Design Work? **YES** ✅

**But with critical caveats:**

1. ✅ **SOCKS5 proxy:** Works perfectly as designed, no limitations
2. ✅ **Info pages:** Work with HTTPS for 1-level subdomains
3. ⚠️ **2-level subdomains:** Require SAN certificate with enumeration
4. ❌ **3+ level subdomains:** Cannot work with HTTPS (SSL limitation)

### The Fundamental Truth

**Subdomain-based routing conflicts with SSL/TLS wildcard limitations.**

You must choose:
- **SOCKS5 primary + HTTP info pages** → Full functionality, some UX friction
- **Path-based routing** → Full HTTPS, different design paradigm
- **HTTP-only proxy** → Works but insecure and problematic
- **Hybrid approach** → Best of both worlds, recommended

### Stop Flip-Flopping By

1. **Accepting SSL limitations** as unchangeable
2. **Embracing SOCKS5** as the primary proxy interface
3. **Using HTTPS** only for info pages and APIs
4. **Documenting clearly** what works and what doesn't

---

## Action Items

### Immediate
- [ ] Set `*.latency.space` to DNS-only in Cloudflare
- [ ] Deploy wildcard cert: `-d latency.space -d *.latency.space`
- [ ] Update README.md with SOCKS5 emphasis
- [ ] Test SOCKS5 end-to-end
- [ ] Remove false promises from docs (e.g., "multi-level wildcard cert")

### Short-term
- [ ] Create SOCKS5 setup guide
- [ ] Build web demo interface
- [ ] Develop HTTP API for proxying
- [ ] Browser extension for configuration

### Long-term
- [ ] Consider path-based routing as alternative
- [ ] Evaluate if HTTP-only proxy needed
- [ ] Multi-hop routing design
- [ ] Advanced features roadmap

---

**Final Verdict:** The design CAN work, but requires accepting SOCKS5 as the primary interface for proxy functionality. Stop trying to make HTTPS work for arbitrary subdomains - it's an SSL/TLS impossibility, not a configuration issue.

**Recommendation:** Implement Option C (Hybrid Approach) immediately. The code is already there, you just need to:
1. Fix Cloudflare configuration
2. Update documentation priorities
3. Stop certificate flip-flopping

---

**Remember:** "Perfect is the enemy of good." The SOCKS5 proxy works beautifully. Ship it. 🚀
