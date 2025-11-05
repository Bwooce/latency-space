# SOCKS5 Connection Error Analysis

**Date:** 2025-11-05
**Test Results:** SOCKS5 port accessible but connection fails

---

## Test Results Summary

### ✅ Test 1: DNS Resolution - **PASS**
```bash
$ nslookup mars.latency.space
Name:    mars.latency.space
Address: 168.119.226.143
```
**Status:** ✅ DNS working correctly!

---

### ✅ Test 2: Port Accessibility - **PASS**
```bash
$ nc -zv mars.latency.space 1080
Connection to mars.latency.space port 1080 [tcp/socks] succeeded!
```
**Status:** ✅ Port 1080 is OPEN and accepting connections!

---

### ❌ Test 3: SOCKS5 Proxy - **FAIL**
```bash
$ time curl --socks5 mars.latency.space:1080 https://example.com
curl: (97) Can't complete SOCKS5 connection to example.com. (4)
curl --socks5 mars.latency.space:1080 https://example.com  0.01s user 0.02s system 2% cpu 1.109 total
```

**Error Code:** 97
**SOCKS5 Reply Code:** 4
**Time Taken:** 1.109 seconds (too fast - no latency applied)

**Status:** ❌ SOCKS5 handshake failing!

---

## Error Analysis

### SOCKS5 Error Code 4 Meaning

According to RFC 1928 (SOCKS5 Protocol):
- Error code **4** = **Host unreachable**

This means the SOCKS5 server:
1. Successfully accepted the connection ✅
2. Completed SOCKS5 handshake ✅
3. Attempted to connect to destination (example.com) ❌
4. Could not reach the destination
5. Returned error code 4 to the client

### What This Tells Us

**Good news:**
- ✅ SOCKS5 server is running
- ✅ SOCKS5 protocol implementation working
- ✅ Port accessible
- ✅ Handshake completes

**Bad news:**
- ❌ Proxy server cannot reach destination (example.com)
- ❌ No latency applied (returned error in 1.1 seconds instead of ~40 minutes)
- ❌ Connection rejected before celestial latency applied

---

## Possible Causes

### 1. Security Validator Rejection (Most Likely)

**Issue:** The allowed hosts whitelist may not be working correctly

**Check:** `proxy/src/security.go:23-59`

The code has `example.com` in the allowed list:
```go
allowedHostsList := []string{
    ...
    "example.com", "www.example.com", // Standard example domain
    ...
}
```

**But:** There might be an issue with how the validation is called or how the hostname is extracted.

---

### 2. DNS Resolution on Server

**Issue:** The proxy server might not be able to resolve `example.com`

**Check:** SSH into server and test:
```bash
# From server:
dig example.com +short
# Should return IP addresses

# OR
curl https://example.com
# Should work
```

---

### 3. Outbound Firewall Rules

**Issue:** Server firewall blocking outbound HTTPS connections

**Check:** SSH into server and test:
```bash
# Try connecting from server to example.com
curl -I https://example.com
# Should return 200 OK

# Check firewall rules
sudo iptables -L OUTPUT -n -v
# Look for DROP rules
```

---

### 4. Celestial Body Extraction Failure

**Issue:** SOCKS5 handler not extracting "mars" from hostname

**Check:** Server logs should show:
```
Expected: "SOCKS: Configured extended timeouts for connection from..."
          "Accessing for ..., via body |mars|"
```

---

### 5. Network Routing Issue

**Issue:** Docker container can't route to internet

**Check:** SSH into server:
```bash
# Test from within container
docker exec latency-space-proxy-1 curl -I https://example.com

# Check DNS from container
docker exec latency-space-proxy-1 nslookup example.com
```

---

## Debugging Steps

### Step 1: Check Server Logs (CRITICAL)

```bash
# SSH into your server, then:

# View recent proxy logs
docker logs latency-space-proxy-1 --tail 100

# Watch logs in real-time
docker logs -f latency-space-proxy-1

# Then from your Mac, try the SOCKS5 connection again:
curl --socks5 mars.latency.space:1080 https://example.com

# Look for error messages in the logs
```

**What to look for:**
- SOCKS5 connection attempts
- Validation errors
- DNS resolution failures
- Connection errors
- Any error messages about example.com

---

### Step 2: Test Server Connectivity

```bash
# SSH into server

# Can the server resolve DNS?
dig example.com +short

# Can the server reach example.com?
curl -I https://example.com

# Can the Docker container reach example.com?
docker exec latency-space-proxy-1 curl -I https://example.com
```

**Expected:** All should succeed

---

### Step 3: Test with Different Destinations

Try different allowed hosts to see if it's specific to example.com:

```bash
# From your Mac:

# Try Google (should be in allowed list)
curl --socks5 mars.latency.space:1080 https://www.google.com

# Try GitHub (should be in allowed list)
curl --socks5 mars.latency.space:1080 https://github.com

# Try Wikipedia (should be in allowed list)
curl --socks5 mars.latency.space:1080 https://en.wikipedia.org
```

**If ANY of these work:** The issue is specific to example.com
**If ALL fail:** The issue is with outbound connectivity from server

---

### Step 4: Check Security Validator Logic

There might be a bug in how the SOCKS5 handler validates destinations.

**Check in code:** `proxy/src/socks.go`

Look for the validation call - it should be checking if `example.com` is allowed.

Possible issues:
- Hostname extracted incorrectly (might include port?)
- Case sensitivity issue
- Validation happening before hostname extraction

---

### Step 5: Test Without Security Validation (Temporary)

**TEMPORARILY disable host validation to test if that's the issue:**

Edit `proxy/src/security.go`:
```go
func (s *SecurityValidator) IsAllowedHost(host string) bool {
    // TEMPORARY DEBUG - REMOVE AFTER TESTING
    return true

    // Original code:
    // if host == "" {
    //     return false
    // }
    // ... rest of validation
}
```

**Rebuild and test:**
```bash
docker compose down
docker compose build proxy
docker compose up -d

# Then test from Mac:
curl --socks5 mars.latency.space:1080 https://example.com
```

**If this works:** The issue is the security validator
**If still fails:** The issue is something else (DNS, routing, etc.)

**IMPORTANT:** Don't forget to revert this change after testing!

---

## Quick Diagnostic Script

Create this script on your Mac to test multiple scenarios:

```bash
#!/bin/bash
# test-socks5.sh

echo "Testing SOCKS5 Proxy..."
echo ""

echo "Test 1: example.com"
curl --socks5 mars.latency.space:1080 --max-time 10 -I https://example.com 2>&1 | head -5
echo ""

echo "Test 2: www.google.com"
curl --socks5 mars.latency.space:1080 --max-time 10 -I https://www.google.com 2>&1 | head -5
echo ""

echo "Test 3: github.com"
curl --socks5 mars.latency.space:1080 --max-time 10 -I https://github.com 2>&1 | head -5
echo ""

echo "Test 4: wikipedia.org"
curl --socks5 mars.latency.space:1080 --max-time 10 -I https://en.wikipedia.org 2>&1 | head -5
```

Run with: `bash test-socks5.sh`

---

## Most Likely Root Cause

Based on the error pattern and code analysis, I've identified **the most likely issue**:

### **Theory 1: Mars is Currently Occluded** (MOST LIKELY)

The proxy has **occlusion detection** that blocks connections when a celestial body is behind another object (like the Sun).

**Evidence:**
- Error happens in 1.1 seconds (no latency applied) ✅
- Error code 4 returned (host unreachable) ✅
- Occurs BEFORE the latency sleep in the code ✅

**Code location:** `proxy/src/socks.go:251-258`
```go
occluded, occluder := IsOccluded(earthObject, targetObject, celestialObjects, time.Now())
if occluded {
    log.Printf("SOCKS connection to %s rejected: occluded by %s", bodyName, occluder.Name)
    s.sendReply(SOCKS5_REP_HOST_UNREACHABLE, net.IPv4zero, 0)
    return fmt.Errorf("SOCKS connection rejected: %s occluded by %s", bodyName, occluder.Name)
}
```

**This would explain everything!**

---

### Theory 2: Server Firewall Blocking Outbound

The Docker container might not be able to make outbound HTTPS connections.

**Test:**
```bash
# SSH to server
docker exec latency-space-proxy-1 curl -I https://example.com
```

If this fails, you need to fix Docker networking or firewall rules.

---

### Theory 3: Security Validator Bug

The SOCKS5 handler might be calling `IsAllowedHost()` incorrectly or the hostname extraction has a bug.

**Check logs for:** Messages about rejected hosts or validation failures

**Relevant code:**
- `proxy/src/socks.go:219-222` - Security validation for SOCKS5
- `proxy/src/security.go:124-148` - IsAllowedHost implementation

---

## Expected Behavior When Working

When SOCKS5 is working correctly, you should see:

```bash
$ time curl --socks5 mars.latency.space:1080 https://example.com

[... waits for ~20 minutes (Mars one-way latency) ...]
[... request sent ...]
[... waits for ~20 minutes (Mars return latency) ...]

<!doctype html>
<html>
<head>
    <title>Example Domain</title>
...

real    40m23.456s  # ~40 minutes for round-trip
user    0m0.023s
sys     0m0.012s
```

---

## **URGENT: Root Cause Analysis**

### ✅ Mars is NOT Occluded

I checked the API and confirmed **Mars is currently NOT occluded**, so occlusion is not the issue.

### 🔍 Remaining Possible Causes (In Order of Likelihood)

Given that the error occurs in 1.1 seconds with no latency applied, the failure happens BEFORE line 282 (the latency sleep). This narrows it down to:

1. **Docker Container DNS Resolution Failure** (MOST LIKELY)
   - Container cannot resolve `example.com` to an IP address
   - Would cause "host unreachable" error at connection attempt

2. **Security Validator Incorrectly Rejecting**
   - Despite `example.com` being in allowed list, validation might be failing
   - Could be case sensitivity or unexpected format issue

3. **Docker Network Routing Issue**
   - Container can resolve DNS but cannot route to external internet
   - Outbound connections blocked by firewall or network config

---

## Immediate Action Items

### **Priority 1: Check Server Logs (CRITICAL)**

```bash
# SSH to your server and check SOCKS5 logs:
docker logs latency-space-proxy-1 --tail 100

# Look for these specific messages:
# - "SOCKS destination allowed: example.com" (security passed)
# - "SOCKS destination rejected: example.com is not in the allowed list" (security failed)
# - "SOCKS connect to example.com:443 from..." (attempting connection)
# - "failed to connect to example.com:443" (connection failure details)
```

**The logs will tell us EXACTLY where it's failing!**

---

### **Priority 2: Test DNS Resolution Inside Container**

```bash
# SSH to your server:
ssh your-server

# Test DNS resolution from inside the proxy container:
docker exec latency-space-proxy-1 nslookup example.com

# Expected output should show:
# Name: example.com
# Address: 93.184.216.34

# If DNS fails, that's the root cause!
```

---

### **Priority 3: Test Outbound Connectivity from Container**

```bash
# SSH to your server:
ssh your-server

# Test if container can reach example.com:
docker exec latency-space-proxy-1 curl -I -m 10 https://example.com

# Expected: HTTP/1.1 200 OK or similar
# If this fails, outbound connections are blocked
```

---

### Priority 2: Check Full Server Logs
```bash
ssh your-server
docker logs latency-space-proxy-1 --tail 100
```

Look for:
- **"occluded by"** messages (most important!)
- SOCKS5 connection attempts
- Validation errors
- DNS failures
- Network errors

### Priority 3: Test Server Connectivity (If Not Occluded)
```bash
# From server:
curl -I https://example.com
docker exec latency-space-proxy-1 curl -I https://example.com
```

### Priority 4: Try Different Destinations
```bash
# From your Mac:
curl --socks5 mars.latency.space:1080 https://www.google.com
```

---

## Next Steps

1. **Check server logs** - This will tell us exactly what's failing
2. **Test server connectivity** - Verify outbound connections work
3. **Try alternative destinations** - Determine if issue is specific to example.com
4. Report back findings and we'll fix the specific issue

---

## Code Locations to Review

If we need to debug the code:

1. **SOCKS5 Handler:** `proxy/src/socks.go:200-300`
2. **Security Validation:** `proxy/src/security.go:150-166`
3. **Host Validation:** `proxy/src/security.go:124-148`
4. **Allowed Hosts List:** `proxy/src/security.go:23-59`

---

## Progress So Far

| Step | Status | Notes |
|------|--------|-------|
| DNS configured | ✅ Working | mars.latency.space → 168.119.226.143 |
| Cloudflare direct routing | ✅ Working | No CF interception |
| Port 1080 open | ✅ Working | TCP connection succeeds |
| SOCKS5 service running | ✅ Working | Accepts connections |
| SOCKS5 handshake | ✅ Working | Protocol negotiation works |
| Destination connection | ❌ **FAILING** | Error code 4: Host unreachable |

**We're 80% there!** Just need to fix the destination connection issue.

---

Please run the log check commands and let me know what you see!
