// dtn.go - Delay/Disruption-Tolerant Networking: store-and-forward delivery for
// bodies whose light-travel latency far exceeds any client's timeout.
//
// A transparent proxy cannot serve a body that is hours or days away: the client
// socket times out long before the simulated delay elapses. Instead, a caller
// POSTs a request to /dtn/send, gets a job id back immediately, and polls
// /dtn/status/{id}. The server models both legs of the light-travel delay: the
// request "arrives" at the destination at submit + one-way, is fetched then, and
// the response is "delivered" one-way later. This mirrors how deep-space networks
// actually move data.
package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	dtnMaxBodyBytes = 1 << 20            // cap stored request/response bodies at 1 MiB
	dtnRetention    = 7 * 24 * time.Hour // keep delivered jobs this long after delivery
	dtnFetchTimeout = 60 * time.Second   // real network timeout for the actual fetch
	dtnMaxJobs      = 512                // hard cap on live jobs, bounding memory + store size
)

// errDTNStoreFull is returned by Add when the store is at capacity.
var errDTNStoreFull = errors.New("DTN store is at capacity; try again later")

// DTNJob is a single store-and-forward request and its (eventual) response.
type DTNJob struct {
	ID          string            `json:"id"`
	Body        string            `json:"body"` // celestial body name
	OneWay      time.Duration     `json:"oneWayNs"`
	SubmittedAt time.Time         `json:"submittedAt"`
	Method      string            `json:"method"`
	URL         string            `json:"url"`
	ReqHeaders  map[string]string `json:"reqHeaders,omitempty"`
	ReqBody     string            `json:"reqBody,omitempty"`

	// Filled in once the outbound request "arrives" and the fetch runs.
	Fetched     bool              `json:"fetched"`
	FetchedAt   time.Time         `json:"fetchedAt,omitempty"`
	RespStatus  int               `json:"respStatus,omitempty"`
	RespHeaders map[string]string `json:"respHeaders,omitempty"`
	RespBody    string            `json:"respBody,omitempty"`
	FetchErr    string            `json:"fetchErr,omitempty"`
}

func (j *DTNJob) arrivalAt() time.Time  { return j.SubmittedAt.Add(j.OneWay) }
func (j *DTNJob) deliveryAt() time.Time { return j.arrivalAt().Add(j.OneWay) }

// state returns the human-facing lifecycle stage at time now.
func (j *DTNJob) state(now time.Time) string {
	if !j.Fetched {
		if now.Before(j.arrivalAt()) {
			return "in_transit" // request still travelling outbound
		}
		return "arriving" // request has reached the destination; fetch pending
	}
	if now.Before(j.FetchedAt.Add(j.OneWay)) {
		return "returning" // response travelling back to Earth
	}
	if j.FetchErr != "" {
		return "failed"
	}
	return "delivered"
}

// DTNStore holds jobs, persists them to disk, and schedules the fetch leg.
type DTNStore struct {
	path     string
	security *SecurityValidator
	metrics  *MetricsCollector

	mu     sync.Mutex
	jobs   map[string]*DTNJob
	timers map[string]*time.Timer
}

// NewDTNStore builds a store backed by the given file and loads any saved jobs.
func NewDTNStore(path string, security *SecurityValidator, metrics *MetricsCollector) *DTNStore {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0o700) // best effort; save() logs if writes still fail
	}
	s := &DTNStore{
		path:     path,
		security: security,
		metrics:  metrics,
		jobs:     make(map[string]*DTNJob),
		timers:   make(map[string]*time.Timer),
	}
	s.load()
	return s
}

// load reads persisted jobs (best effort - a missing or corrupt file is ignored).
func (s *DTNStore) load() {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var jobs []*DTNJob
	if err := json.Unmarshal(data, &jobs); err != nil {
		log.Printf("DTN: ignoring unreadable store %s: %v", s.path, err)
		return
	}
	for _, j := range jobs {
		s.jobs[j.ID] = j
	}
	log.Printf("DTN: loaded %d job(s) from %s", len(jobs), s.path)
}

// save writes all jobs atomically. Caller must hold s.mu.
func (s *DTNStore) save() {
	if s.path == "" {
		return
	}
	jobs := make([]*DTNJob, 0, len(s.jobs))
	for _, j := range s.jobs {
		jobs = append(jobs, j)
	}
	data, err := json.MarshalIndent(jobs, "", "  ")
	if err != nil {
		log.Printf("DTN: marshal failed: %v", err)
		return
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		log.Printf("DTN: write failed: %v", err)
		return
	}
	if err := os.Rename(tmp, s.path); err != nil {
		log.Printf("DTN: rename failed: %v", err)
	}
}

// Start reschedules the fetch leg for any pending jobs (surviving a restart) and
// launches the retention janitor. stop closes to shut the janitor down.
func (s *DTNStore) Start(stop <-chan struct{}) {
	s.mu.Lock()
	for _, j := range s.jobs {
		if !j.Fetched {
			s.scheduleFetchLocked(j)
		}
	}
	s.mu.Unlock()

	go func() {
		t := time.NewTicker(time.Hour)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				s.sweep()
			}
		}
	}()
}

// scheduleFetchLocked arms a timer to fetch the destination when the outbound
// request "arrives". If arrival is already in the past (e.g. after a restart),
// the fetch runs immediately. Caller must hold s.mu.
func (s *DTNStore) scheduleFetchLocked(j *DTNJob) {
	delay := time.Until(j.arrivalAt())
	if delay < 0 {
		delay = 0
	}
	id := j.ID
	s.timers[id] = time.AfterFunc(delay, func() { s.runFetch(id) })
}

// runFetch performs the real HTTP request for a job and records the result.
func (s *DTNStore) runFetch(id string) {
	s.mu.Lock()
	j, ok := s.jobs[id]
	delete(s.timers, id)
	if !ok || j.Fetched {
		s.mu.Unlock()
		return
	}
	// Snapshot the immutable request fields; release the lock during network I/O.
	method, rawURL, reqHeaders, reqBody := j.Method, j.URL, j.ReqHeaders, j.ReqBody
	s.mu.Unlock()

	status, respHeaders, respBody, fetchErr := s.fetch(method, rawURL, reqHeaders, reqBody)

	s.mu.Lock()
	j, ok = s.jobs[id]
	var bodyName string
	if ok {
		j.Fetched = true
		j.FetchedAt = time.Now()
		j.RespStatus = status
		j.RespHeaders = respHeaders
		j.RespBody = respBody
		j.FetchErr = fetchErr
		bodyName = j.Body
		s.save()
	}
	s.mu.Unlock()

	if s.metrics != nil && ok {
		s.metrics.RecordRequest(bodyName, "dtn", 0)
	}
}

// fetch does the actual outbound request. Returns status, headers, body, error.
func (s *DTNStore) fetch(method, rawURL string, headers map[string]string, body string) (int, map[string]string, string, string) {
	var reqBody io.Reader
	if body != "" {
		reqBody = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, rawURL, reqBody)
	if err != nil {
		return 0, nil, "", fmt.Sprintf("build request: %v", err)
	}
	for k, v := range headers {
		if !strings.EqualFold(k, "host") {
			req.Header.Set(k, v)
		}
	}
	client := &http.Client{
		Timeout: dtnFetchTimeout,
		// Re-validate every redirect hop against the allowlist. Without this an
		// open redirect on an allowlisted host could bounce the fetch to
		// 169.254.169.254 / 127.0.0.1 / internal services and return the body
		// via /dtn/status - an SSRF that defeats the initial-URL allowlist.
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			if _, err := s.security.ValidateHTTPTarget(req.URL.String()); err != nil {
				return fmt.Errorf("redirect to disallowed target: %w", err)
			}
			return nil
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, "", fmt.Sprintf("fetch: %v", err)
	}
	defer resp.Body.Close()

	rb, _ := io.ReadAll(io.LimitReader(resp.Body, dtnMaxBodyBytes))
	rh := make(map[string]string, len(resp.Header))
	for k := range resp.Header {
		rh[k] = resp.Header.Get(k)
	}
	return resp.StatusCode, rh, string(rb), ""
}

// sweep drops jobs whose retention window has passed.
func (s *DTNStore) sweep() {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	changed := false
	for id, j := range s.jobs {
		if j.Fetched && now.After(j.FetchedAt.Add(j.OneWay).Add(dtnRetention)) {
			delete(s.jobs, id)
			changed = true
		}
	}
	if changed {
		s.save()
	}
}

// Add validates and stores a new job, then schedules its fetch.
func (s *DTNStore) Add(bodyName, method, rawURL string, headers map[string]string, body string, oneWay time.Duration) (*DTNJob, error) {
	validatedURL, err := s.security.ValidateHTTPTarget(rawURL)
	if err != nil {
		return nil, err
	}
	if method == "" {
		method = http.MethodGet
	}
	if len(body) > dtnMaxBodyBytes {
		return nil, fmt.Errorf("request body exceeds %d bytes", dtnMaxBodyBytes)
	}

	j := &DTNJob{
		ID:          newDTNID(),
		Body:        bodyName,
		OneWay:      oneWay,
		SubmittedAt: time.Now(),
		Method:      strings.ToUpper(method),
		URL:         validatedURL,
		ReqHeaders:  headers,
		ReqBody:     body,
	}

	s.mu.Lock()
	if len(s.jobs) >= dtnMaxJobs {
		s.mu.Unlock()
		return nil, errDTNStoreFull
	}
	s.jobs[j.ID] = j
	s.scheduleFetchLocked(j)
	s.save()
	snap := *j
	s.mu.Unlock()
	return &snap, nil
}

// Get returns a snapshot of a job by id, copied under the lock so callers can
// read it without racing the fetch goroutine that mutates the stored job.
// Response header/body maps are assigned once (never mutated after) so sharing
// them in the copy is safe.
func (s *DTNStore) Get(id string) (*DTNJob, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.jobs[id]
	if !ok {
		return nil, false
	}
	snap := *j
	return &snap, true
}

func newDTNID() string {
	var b [12]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
