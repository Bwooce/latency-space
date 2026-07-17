// dtn_http.go - HTTP surface for the DTN store-and-forward API.
//
//	POST /dtn/send          submit a request; returns a job id
//	GET  /dtn/status/{id}   poll a job; returns the response once "delivered"
//
// The celestial body is taken from the request host (e.g. voyager-1.latency.space)
// or from the "via" field in the JSON body.
package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

// dtnSendRequest is the JSON accepted by POST /dtn/send.
type dtnSendRequest struct {
	URL     string            `json:"url"`
	Method  string            `json:"method,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Payload string            `json:"payload,omitempty"` // request body
	Via     string            `json:"via,omitempty"`     // celestial body, if host is the apex
}

func (s *Server) handleDTN(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/dtn/send":
		s.handleDTNSend(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/dtn/status/"):
		s.handleDTNStatus(w, r)
	default:
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "unknown DTN endpoint; use POST /dtn/send or GET /dtn/status/{id}",
		})
	}
}

func (s *Server) handleDTNSend(w http.ResponseWriter, r *http.Request) {
	// Abuse control, same as the other proxy paths.
	release, err := s.limiter.Acquire(clientIP(r.RemoteAddr))
	if err != nil {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": err.Error()})
		return
	}
	defer release()

	var req dtnSendRequest
	dec := json.NewDecoder(io.LimitReader(r.Body, dtnMaxBodyBytes))
	if err := dec.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body: " + err.Error()})
		return
	}

	// Resolve the celestial body: prefer the host subdomain, fall back to "via".
	bodyName := s.resolveCelestialHost(r.Host)
	if bodyName == "" && req.Via != "" {
		if obj, ok := findObjectByName(getCelestialObjects(), req.Via); ok {
			bodyName = obj.Name
		}
	}
	if bodyName == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "no celestial body: POST to a body host (e.g. voyager-1.latency.space) or set \"via\"",
		})
		return
	}

	oneWay := CalculateLatency(getCurrentDistance(bodyName))
	// Refuse bodies with negligible latency (Earth is 0). Without the light-travel
	// friction DTN would be a plain open proxy, which the SOCKS path also guards
	// against; keep Earth non-proxyable. Skipped in test mode, like the SOCKS guard.
	if !isTestMode.Load() && oneWay < time.Second {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": bodyName + " has insufficient latency to proxy (it would be an open proxy)",
		})
		return
	}

	job, err := s.dtn.Add(bodyName, req.Method, req.URL, req.Headers, req.Payload, oneWay)
	if err != nil {
		if errors.Is(err, errDTNStoreFull) {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"id":                   job.ID,
		"body":                 job.Body,
		"state":                job.state(time.Now()),
		"oneWayLatencySeconds": job.OneWay.Seconds(),
		"submittedAt":          job.SubmittedAt,
		"arrivesAt":            job.arrivalAt(),
		"estimatedDeliveryAt":  job.deliveryAt(),
		"statusUrl":            "/dtn/status/" + job.ID,
	})
}

func (s *Server) handleDTNStatus(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/dtn/status/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing job id"})
		return
	}
	job, ok := s.dtn.Get(id)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no such job (it may have expired)"})
		return
	}

	now := time.Now()
	state := job.state(now)
	out := map[string]interface{}{
		"id":                   job.ID,
		"body":                 job.Body,
		"state":                state,
		"oneWayLatencySeconds": job.OneWay.Seconds(),
		"submittedAt":          job.SubmittedAt,
		"arrivesAt":            job.arrivalAt(),
		"estimatedDeliveryAt":  job.deliveryAt(),
	}

	// The response is only revealed once it has finished travelling back to Earth.
	switch state {
	case "delivered":
		out["response"] = map[string]interface{}{
			"status":  job.RespStatus,
			"headers": job.RespHeaders,
			"body":    job.RespBody,
		}
	case "failed":
		out["error"] = job.FetchErr
	}

	writeJSON(w, http.StatusOK, out)
}

// writeJSON writes v as an indented JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}
