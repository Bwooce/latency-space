package main

import (
	"os"
	"testing"
)

// TestAllowedHostsEnvMerge verifies ALLOWED_HOSTS extends the default allowlist
// without removing built-ins.
func TestAllowedHostsEnvMerge(t *testing.T) {
	orig, had := os.LookupEnv("ALLOWED_HOSTS")
	defer func() {
		if had {
			os.Setenv("ALLOWED_HOSTS", orig)
		} else {
			os.Unsetenv("ALLOWED_HOSTS")
		}
	}()

	os.Setenv("ALLOWED_HOSTS", "extra-one.example, Extra-Two.example ,, duplicate.example")
	s := NewSecurityValidator()

	// New hosts admitted (case-insensitive, trimmed, blanks ignored).
	for _, h := range []string{"extra-one.example", "extra-two.example", "duplicate.example"} {
		if !s.IsAllowedHost(h) {
			t.Errorf("env-added host %q should be allowed", h)
		}
	}
	// Built-in defaults still present.
	if !s.IsAllowedHost("github.com") {
		t.Error("built-in allowlist host github.com should remain")
	}
	// Unlisted host still rejected.
	if s.IsAllowedHost("not-added.example") {
		t.Error("host not in defaults or env must be rejected")
	}
}

// TestAllowedHostsAccessorSorted verifies the accessor returns a sorted,
// non-empty list (used by the /_debug/allowed-hosts endpoint).
func TestAllowedHostsAccessorSorted(t *testing.T) {
	os.Unsetenv("ALLOWED_HOSTS")
	s := NewSecurityValidator()
	hosts := s.AllowedHosts()
	if len(hosts) == 0 {
		t.Fatal("expected a non-empty allowlist")
	}
	for i := 1; i < len(hosts); i++ {
		if hosts[i] < hosts[i-1] {
			t.Errorf("AllowedHosts not sorted at %d: %q > %q", i, hosts[i-1], hosts[i])
		}
	}
	ports := s.AllowedPorts()
	if len(ports) == 0 {
		t.Error("expected non-empty port list")
	}
}
