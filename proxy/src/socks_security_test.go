package main

import "testing"

// TestSOCKSRejectsLoopbackInProduction is the regression test for the SSRF
// finding: an unauthenticated SOCKS client must not be able to CONNECT to a
// loopback address in production. Loopback stays allowed only in test mode.
func TestSOCKSRejectsLoopbackInProduction(t *testing.T) {
	orig := isTestMode
	defer func() { isTestMode = orig }()

	h := &SOCKSHandler{security: NewSecurityValidator()}

	isTestMode = false
	for _, addr := range []string{"127.0.0.1", "::1", "127.0.0.53"} {
		if h.isAllowedDestination(addr) {
			t.Errorf("loopback %s must be rejected in production", addr)
		}
	}
	// Non-loopback IP literals are rejected in both modes.
	if h.isAllowedDestination("169.254.169.254") {
		t.Error("link-local metadata IP must be rejected")
	}

	isTestMode = true
	if !h.isAllowedDestination("127.0.0.1") {
		t.Error("loopback should be allowed in test mode (tests use echo servers)")
	}
}

// TestSOCKSPortAllowlist verifies the port allowlist that the CONNECT path now
// enforces via ValidateSocksDestination: allowlisted host on a bad port fails.
func TestSOCKSPortAllowlist(t *testing.T) {
	s := NewSecurityValidator()

	if err := s.ValidateSocksDestination("github.com", 443); err != nil {
		t.Errorf("github.com:443 should be allowed: %v", err)
	}
	if err := s.ValidateSocksDestination("github.com", 80); err != nil {
		t.Errorf("github.com:80 should be allowed: %v", err)
	}
	// SSH and other non-web ports to an allowlisted host must be rejected —
	// this is the port-bypass the CONNECT path previously allowed.
	for _, port := range []uint16{22, 25, 3389, 6379} {
		if err := s.ValidateSocksDestination("github.com", port); err == nil {
			t.Errorf("github.com:%d must be rejected (port not allowlisted)", port)
		}
	}
	// Non-allowlisted host rejected regardless of port.
	if err := s.ValidateSocksDestination("evil.not-listed.example", 443); err == nil {
		t.Error("non-allowlisted host must be rejected")
	}
}
