// proxy/src/ratelimit.go
//
// Abuse controls for the proxy paths. The SOCKS listener listens on its own
// port and is NOT behind the front-end nginx, so nginx's limit_req/limit_conn
// rules never see it. The allowlist (security.go) stops the proxy being used
// to reach arbitrary hosts, but does nothing against connection floods or an
// attacker holding many latency-delayed tunnels open (each tunnel keeps a
// goroutine plus a buffered delay queue alive for the full light-travel time).
// These limits bound that.
//
// Two things are bounded per client IP:
//   - new connections per minute (a token bucket), and
//   - concurrent in-flight proxied connections (also globally).
//
// A small hand-rolled token bucket is used deliberately so this needs no
// external dependency.
package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"
)

// limiterIdleTTL is how long an idle per-IP bucket is kept before the janitor
// prunes it. Long enough that a client cannot reset its rate by reconnecting.
const limiterIdleTTL = 15 * time.Minute

type ipBucket struct {
	tokens     float64
	lastRefill time.Time
	lastSeen   time.Time
}

// RateLimiter enforces per-IP connection rate and concurrency caps.
// A nil *RateLimiter is a valid no-op limiter (admits everything).
type RateLimiter struct {
	ratePerSec float64 // token refill rate; <=0 disables the rate check
	burst      float64
	maxPerIP   int // <=0 disables
	maxTotal   int // <=0 disables

	mu      sync.Mutex
	buckets map[string]*ipBucket
	perIP   map[string]int
	total   int
}

// NewRateLimiter builds a limiter. Zero/negative caps disable the matching
// check, which tests and trusted deployments can rely on.
func NewRateLimiter(connRatePerMin float64, burst, maxPerIP, maxTotal int) *RateLimiter {
	return &RateLimiter{
		ratePerSec: connRatePerMin / 60.0,
		burst:      float64(burst),
		maxPerIP:   maxPerIP,
		maxTotal:   maxTotal,
		buckets:    make(map[string]*ipBucket),
		perIP:      make(map[string]int),
	}
}

// newRateLimiterFromEnv reads the abuse-control settings from the environment,
// falling back to sensible defaults.
func newRateLimiterFromEnv() *RateLimiter {
	return NewRateLimiter(
		envFloat("CONN_RATE_PER_MIN", 60),
		envInt("CONN_BURST", 20),
		envInt("MAX_CONNS_PER_IP", 20),
		envInt("MAX_CONNS_TOTAL", 500),
	)
}

// Acquire admits a new proxied connection from ip. On success it returns a
// release function that MUST be called when the connection finishes. On
// rejection it returns an error naming the limit that was hit.
func (r *RateLimiter) Acquire(ip string) (func(), error) {
	if r == nil {
		return func() {}, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()

	// Connection-rate token bucket (skipped when rate <= 0). Retained across
	// connection close (pruned only by the janitor) so reconnecting cannot
	// reset the rate.
	if r.ratePerSec > 0 {
		b, ok := r.buckets[ip]
		if !ok {
			b = &ipBucket{tokens: r.burst, lastRefill: now}
			r.buckets[ip] = b
		}
		elapsed := now.Sub(b.lastRefill).Seconds()
		b.tokens += elapsed * r.ratePerSec
		if b.tokens > r.burst {
			b.tokens = r.burst
		}
		b.lastRefill = now
		b.lastSeen = now
		if b.tokens < 1 {
			return nil, fmt.Errorf("connection rate limit exceeded for %s", ip)
		}
		b.tokens--
	}

	if r.maxTotal > 0 && r.total >= r.maxTotal {
		return nil, fmt.Errorf("server connection limit reached (%d)", r.maxTotal)
	}
	if r.maxPerIP > 0 && r.perIP[ip] >= r.maxPerIP {
		return nil, fmt.Errorf("per-IP connection limit reached for %s (%d)", ip, r.maxPerIP)
	}

	r.perIP[ip]++
	r.total++

	released := false
	return func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		if released {
			return
		}
		released = true
		r.total--
		if r.perIP[ip]--; r.perIP[ip] <= 0 {
			delete(r.perIP, ip)
			// The rate bucket is intentionally NOT deleted here; the janitor
			// prunes it once genuinely idle.
		}
	}, nil
}

// StartCleanup runs a background janitor that prunes idle per-IP buckets until
// stop is closed. Call once from the server; tests may omit it.
func (r *RateLimiter) StartCleanup(stop <-chan struct{}) {
	if r == nil {
		return
	}
	ticker := time.NewTicker(limiterIdleTTL)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			r.prune()
		}
	}
}

func (r *RateLimiter) prune() {
	cutoff := time.Now().Add(-limiterIdleTTL)
	r.mu.Lock()
	defer r.mu.Unlock()
	for ip, b := range r.buckets {
		if r.perIP[ip] == 0 && b.lastSeen.Before(cutoff) {
			delete(r.buckets, ip)
		}
	}
}

func envInt(name string, def int) int {
	if v := os.Getenv(name); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			return n
		}
		log.Printf("Ignoring invalid %s=%q", name, os.Getenv(name))
	}
	return def
}

func envFloat(name string, def float64) float64 {
	if v := os.Getenv(name); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f >= 0 {
			return f
		}
		log.Printf("Ignoring invalid %s=%q", name, os.Getenv(name))
	}
	return def
}
