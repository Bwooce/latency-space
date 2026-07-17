package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/latency-space/shared/celestial"
)

// newDTNTestServer builds a Server wired with a DTN store at a temp path, without
// NewServer (which registers global Prometheus metrics). A nil limiter is a
// no-op, which is what we want here.
func newDTNTestServer(t *testing.T) *Server {
	t.Helper()
	sec := NewSecurityValidator()
	s := &Server{security: sec, metrics: NewTestMetricsCollector(), httpEnabled: true}
	s.dtn = NewDTNStore(t.TempDir()+"/dtn.json", sec, s.metrics)
	return s
}

func dtnSend(t *testing.T, s *Server, host, jsonBody string) (int, map[string]interface{}) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "http://x/dtn/send", strings.NewReader(jsonBody))
	req.Host = host
	rec := httptest.NewRecorder()
	s.handleDTN(rec, req)
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

func dtnStatus(t *testing.T, s *Server, id string) (int, map[string]interface{}) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "http://x/dtn/status/"+id, nil)
	rec := httptest.NewRecorder()
	s.handleDTN(rec, req)
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

// TestDTNLifecycle drives a job from submission through delivery, checking that
// the response is withheld until both light-travel legs have "elapsed".
func TestDTNLifecycle(t *testing.T) {
	// One-way latency of 40ms keeps the test fast but leaves a window where the
	// job is observably in transit before it is delivered (~80ms round trip).
	defer setupTestModeWithLatency(40 * time.Millisecond)()
	setCelestialObjects(celestial.InitSolarSystemObjects())

	// Destination echo server.
	dest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "hello from space")
	}))
	defer dest.Close()

	s := newDTNTestServer(t)

	code, out := dtnSend(t, s, "mars.latency.space",
		fmt.Sprintf(`{"url":%q,"method":"GET"}`, dest.URL))
	if code != http.StatusAccepted {
		t.Fatalf("send: expected 202, got %d (%v)", code, out)
	}
	id, _ := out["id"].(string)
	if id == "" {
		t.Fatalf("send: no job id in %v", out)
	}
	if out["body"] != "Mars" {
		t.Errorf("send: expected body Mars, got %v", out["body"])
	}

	// Immediately after submission the request is still travelling outbound.
	_, st := dtnStatus(t, s, id)
	if state := st["state"]; state != "in_transit" && state != "arriving" {
		t.Errorf("early state: expected in_transit/arriving, got %v", state)
	}
	if _, leaked := st["response"]; leaked {
		t.Error("response must not be revealed before delivery")
	}

	// Poll until delivered (bounded).
	var final map[string]interface{}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		_, final = dtnStatus(t, s, id)
		if final["state"] == "delivered" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if final["state"] != "delivered" {
		t.Fatalf("job never delivered; last state %v", final["state"])
	}
	resp, ok := final["response"].(map[string]interface{})
	if !ok {
		t.Fatalf("delivered job missing response: %v", final)
	}
	if resp["status"].(float64) != http.StatusOK {
		t.Errorf("expected status 200, got %v", resp["status"])
	}
	if body, _ := resp["body"].(string); body != "hello from space" {
		t.Errorf("expected echoed body, got %q", body)
	}
}

// TestDTNRejectsNonAllowlistedHost verifies the allowlist is enforced on submit.
func TestDTNRejectsNonAllowlistedHost(t *testing.T) {
	defer setupTestMode()()
	setCelestialObjects(celestial.InitSolarSystemObjects())
	s := newDTNTestServer(t)

	code, out := dtnSend(t, s, "mars.latency.space",
		`{"url":"https://evil.not-listed-anywhere.example/"}`)
	if code != http.StatusForbidden {
		t.Fatalf("expected 403 for non-allowlisted host, got %d (%v)", code, out)
	}
}

// TestDTNRequiresBody verifies a body must be identified (host or "via").
func TestDTNRequiresBody(t *testing.T) {
	defer setupTestMode()()
	setCelestialObjects(celestial.InitSolarSystemObjects())
	s := newDTNTestServer(t)

	// Apex host, no "via".
	code, _ := dtnSend(t, s, "latency.space", `{"url":"https://example.com/"}`)
	if code != http.StatusBadRequest {
		t.Fatalf("expected 400 when no body is identified, got %d", code)
	}

	// "via" fills it in.
	code, out := dtnSend(t, s, "latency.space", `{"url":"https://example.com/","via":"Jupiter"}`)
	if code != http.StatusAccepted {
		t.Fatalf("expected 202 with via=Jupiter, got %d (%v)", code, out)
	}
	if out["body"] != "Jupiter" {
		t.Errorf("expected body Jupiter, got %v", out["body"])
	}
}

// TestDTNRedirectSSRFBlocked verifies a redirect to a non-allowlisted host is
// refused (not followed), so the response is never delivered.
func TestDTNRedirectSSRFBlocked(t *testing.T) {
	defer setupTestModeWithLatency(40 * time.Millisecond)()
	setCelestialObjects(celestial.InitSolarSystemObjects())

	// Allowlisted-in-test-mode loopback origin that redirects to a host that is
	// NOT on the allowlist (simulating an open redirect -> internal/SSRF target).
	dest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://evil.not-listed-anywhere.example/secret", http.StatusFound)
	}))
	defer dest.Close()

	s := newDTNTestServer(t)
	code, out := dtnSend(t, s, "mars.latency.space", fmt.Sprintf(`{"url":%q}`, dest.URL))
	if code != http.StatusAccepted {
		t.Fatalf("send: expected 202, got %d (%v)", code, out)
	}
	id := out["id"].(string)

	var final map[string]interface{}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		_, final = dtnStatus(t, s, id)
		st := final["state"]
		if st == "failed" || st == "delivered" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if final["state"] != "failed" {
		t.Fatalf("expected redirect-to-disallowed to fail, got state %v (response leaked: %v)",
			final["state"], final["response"])
	}
	if _, leaked := final["response"]; leaked {
		t.Error("SSRF: a blocked redirect must not deliver a response body")
	}
}

// TestDTNRejectsEarth verifies zero/negligible-latency bodies are refused
// outside test mode, keeping DTN from being an open proxy.
func TestDTNRejectsEarth(t *testing.T) {
	orig := isTestMode.Load()
	isTestMode.Store(false) // exercise the production guard
	defer isTestMode.Store(orig)
	setCelestialObjects(celestial.InitSolarSystemObjects())

	s := newDTNTestServer(t)
	code, out := dtnSend(t, s, "earth.latency.space", `{"url":"https://example.com/"}`)
	if code != http.StatusBadRequest {
		t.Fatalf("expected 400 for Earth (zero latency), got %d (%v)", code, out)
	}
}

// TestDTNStoreCapacity verifies Add refuses new jobs once the store is full.
func TestDTNStoreCapacity(t *testing.T) {
	defer setupTestMode()()
	store := NewDTNStore(t.TempDir()+"/dtn.json", NewSecurityValidator(), NewTestMetricsCollector())
	// Pre-fill the map to the cap without scheduling real fetches.
	for i := 0; i < dtnMaxJobs; i++ {
		store.jobs[fmt.Sprintf("job-%d", i)] = &DTNJob{ID: fmt.Sprintf("job-%d", i)}
	}
	_, err := store.Add("Mars", "GET", "https://example.com/", nil, "", time.Second)
	if !errors.Is(err, errDTNStoreFull) {
		t.Fatalf("expected errDTNStoreFull at capacity, got %v", err)
	}
}

// TestDTNStatusUnknownJob returns 404 for an unknown id.
func TestDTNStatusUnknownJob(t *testing.T) {
	s := newDTNTestServer(t)
	code, _ := dtnStatus(t, s, "does-not-exist")
	if code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown job, got %d", code)
	}
}

// TestDTNPersistenceReload verifies jobs survive a store reload (process restart).
func TestDTNPersistenceReload(t *testing.T) {
	defer setupTestModeWithLatency(50 * time.Millisecond)()
	setCelestialObjects(celestial.InitSolarSystemObjects())

	dir := t.TempDir()
	path := dir + "/dtn.json"
	sec := NewSecurityValidator()

	store1 := NewDTNStore(path, sec, NewTestMetricsCollector())
	// Loopback (allowed in test mode) so the scheduled fetch stays local.
	job, err := store1.Add("Mars", "GET", "http://127.0.0.1:80/", nil, "", 50*time.Millisecond)
	if err != nil {
		t.Fatalf("add: %v", err)
	}

	// A fresh store loads the persisted job.
	store2 := NewDTNStore(path, sec, NewTestMetricsCollector())
	if got, ok := store2.Get(job.ID); !ok || got.Body != "Mars" {
		t.Fatalf("reloaded store missing job %s", job.ID)
	}
}
