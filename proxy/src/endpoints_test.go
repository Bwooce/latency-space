package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/latency-space/shared/celestial"
)

// TestDebugStatusEndpoint covers /_debug/status, which nginx routed but the app
// previously 404'd.
func TestDebugStatusEndpoint(t *testing.T) {
	setCelestialObjects(celestial.InitSolarSystemObjects())
	s := &Server{security: NewSecurityValidator(), metrics: NewTestMetricsCollector(), httpEnabled: true}
	s.dtn = NewDTNStore(t.TempDir()+"/dtn.json", s.security, s.metrics)

	req := httptest.NewRequest(http.MethodGet, "http://latency.space/_debug/status", nil)
	rec := httptest.NewRecorder()
	s.handleHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("status response not JSON: %v", err)
	}
	for _, k := range []string{"httpEnabled", "socksEnabled", "celestialObjects", "allowedHosts", "dtnJobs"} {
		if _, ok := out[k]; !ok {
			t.Errorf("status JSON missing key %q (got %v)", k, out)
		}
	}
}

// TestRobotsTxt verifies body hosts serve a Disallow-all robots.txt.
func TestRobotsTxt(t *testing.T) {
	s := &Server{security: NewSecurityValidator(), metrics: NewTestMetricsCollector()}
	req := httptest.NewRequest(http.MethodGet, "http://mars.latency.space/robots.txt", nil)
	rec := httptest.NewRecorder()
	s.handleHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("expected text/plain, got %q", ct)
	}
	if !strings.Contains(rec.Body.String(), "Disallow: /") {
		t.Errorf("robots.txt should disallow all, got %q", rec.Body.String())
	}
}
