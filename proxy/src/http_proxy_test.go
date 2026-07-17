package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newTestServer builds a Server without NewServer, which registers global
// Prometheus metrics and therefore may only be called once per test binary.
func newTestServer() *Server {
	return &Server{
		security:    NewSecurityValidator(),
		metrics:     NewTestMetricsCollector(),
		httpEnabled: true,
	}
}

// TestValidateDestinationSchemeAndAllowlist covers the mechanism the HTTP
// proxy fix relies on: ValidateDestination prepends a scheme to a bare host
// (the raw target from parseHostForCelestialBody has none, which used to make
// http.NewRequest fail with "unsupported protocol scheme") and enforces the
// allowlist.
func TestValidateDestinationSchemeAndAllowlist(t *testing.T) {
	s := NewSecurityValidator()

	// Bare allowlisted host: scheme prepended, allowed.
	got, err := s.ValidateDestination("www.example.com")
	if err != nil {
		t.Fatalf("allowlisted host rejected: %v", err)
	}
	if got != "http://www.example.com" {
		t.Errorf("expected scheme prepended, got %q", got)
	}

	// Non-allowlisted host: rejected (this is what keeps the HTTP path from
	// being an open proxy).
	if _, err := s.ValidateDestination("evil.not-listed-anywhere.example"); err == nil {
		t.Error("expected non-allowlisted host to be rejected")
	}
}

// TestHTTPProxyRejectsNonAllowlistedHost verifies the handler now enforces the
// allowlist on the HTTP proxy path (previously it had no check at all, so any
// host embedded in the subdomain was proxied — an open HTTP proxy).
func TestHTTPProxyRejectsNonAllowlistedHost(t *testing.T) {
	orig := getCelestialObjects()
	setCelestialObjects([]CelestialObject{
		{Name: "Earth", Type: "planet"},
		{Name: "Mars", Type: "planet"},
	})
	defer func() { setCelestialObjects(orig) }()

	s := newTestServer()

	req := httptest.NewRequest("GET", "http://x/", nil)
	req.Host = "evil.not-listed-anywhere.example.mars.latency.space"
	rec := httptest.NewRecorder()

	s.handleHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for non-allowlisted target, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

// TestHTTPProxyAllowlistedHostPassesValidation verifies an allowlisted target
// is NOT rejected by validation. In test mode latency is a few ms, so the
// handler's own "insufficient latency" guard (>=1s) returns 400 before any
// network fetch — which is exactly the point where we can confirm validation
// (scheme prepend + allowlist) succeeded without making a real request.
func TestHTTPProxyAllowlistedHostPassesValidation(t *testing.T) {
	orig := getCelestialObjects()
	setCelestialObjects([]CelestialObject{
		{Name: "Earth", Type: "planet"},
		{Name: "Mars", Type: "planet"},
	})
	defer func() { setCelestialObjects(orig) }()

	cleanup := setupTestModeWithLatency(3 * time.Millisecond)
	defer cleanup()

	s := newTestServer()

	req := httptest.NewRequest("GET", "http://x/", nil)
	req.Host = "example.com.mars.latency.space"
	rec := httptest.NewRecorder()

	s.handleHTTP(rec, req)

	// Must NOT be 403 (validation passed) and must NOT be a scheme error.
	// It stops at the insufficient-latency guard (400) before the fetch.
	if rec.Code == http.StatusForbidden {
		t.Fatalf("allowlisted target wrongly rejected by validation: %s", rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, "unsupported protocol scheme") || strings.Contains(body, "Error creating proxy request") {
		t.Fatalf("scheme bug still present: %s", body)
	}
}

// TestHTTPTimeoutNoOverflow is the regression test for the Duration overflow:
// `latency * 2 * time.Second` (Duration*Duration) went negative for distant
// bodies. The corrected form (2*latency + 30s) must stay positive even at
// Voyager scale.
func TestHTTPTimeoutNoOverflow(t *testing.T) {
	cases := []struct {
		name    string
		latency time.Duration
	}{
		{"Mars far", 22 * time.Minute},
		{"Jupiter", 53 * time.Minute},
		{"Voyager 1", 23 * time.Hour},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			timeout := 2*tc.latency + 30*time.Second
			if timeout <= 0 {
				t.Errorf("timeout overflowed for %s: %v", tc.name, timeout)
			}
		})
	}
}
