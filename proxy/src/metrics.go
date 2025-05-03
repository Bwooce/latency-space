// proxy/src/metrics.go
package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"time"
)

type MetricsCollector struct {
	requestDuration *prometheus.HistogramVec
	requestsTotal   *prometheus.CounterVec
	bandwidthUsage  *prometheus.CounterVec
	udpPackets      *prometheus.CounterVec // Counter for UDP packets handled by SOCKS UDP associate
}

// NewMetricsCollector creates and registers Prometheus metrics collectors.
func NewMetricsCollector() *MetricsCollector {
	m := &MetricsCollector{
		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "request_duration_seconds",
				Help: "Time spent processing request",
			},
			[]string{"body", "type"},
		),
		requestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "requests_total",
				Help: "Total number of requests",
			},
			[]string{"body", "type"},
		),
		bandwidthUsage: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "bandwidth_bytes_total",
				Help: "Total bandwidth usage in bytes",
			},
			[]string{"body", "direction"},
		),
		udpPackets: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "udp_packets_total",
				Help: "Total UDP packets processed",
			},
			[]string{"body"}, // Label by celestial body
		),
	}

	// Register Prometheus metrics.
	prometheus.MustRegister(m.requestDuration)
	prometheus.MustRegister(m.requestsTotal)
	prometheus.MustRegister(m.bandwidthUsage)
	prometheus.MustRegister(m.udpPackets)

	return m
}

// RecordRequest observes request duration and increments the total request count.
// Labels: body (celestial body name), type (http/socks).
func (m *MetricsCollector) RecordRequest(body, reqType string, duration time.Duration) {
	m.requestDuration.WithLabelValues(body, reqType).Observe(duration.Seconds())
	m.requestsTotal.WithLabelValues(body, reqType).Inc()
}

// TrackBandwidth tracks outgoing bandwidth usage (client -> target).
// Labels: body (celestial body name), direction ("out").
func (m *MetricsCollector) TrackBandwidth(body string, bytes int64) {
	if bytes > 0 {
		// Assuming this tracks bytes sent *from* the proxy *to* the target
		m.bandwidthUsage.WithLabelValues(body, "out").Add(float64(bytes))
	}
}

// RecordUDPPacket increments the UDP packet count and tracks incoming UDP bandwidth.
// Labels: body (celestial body name), direction ("in").
func (m *MetricsCollector) RecordUDPPacket(body string, bytes int64) {
	m.udpPackets.WithLabelValues(body).Inc()
	// Assuming this tracks bytes received *by* the proxy *from* the UDP client
	m.bandwidthUsage.WithLabelValues(body, "in").Add(float64(bytes))
}

// ServeMetrics starts an HTTP server to expose Prometheus metrics on the given address.
func (m *MetricsCollector) ServeMetrics(addr string) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	log.Printf("Starting Prometheus metrics server on %s", addr)
	err := http.ListenAndServe(addr, mux)
	if err != nil {
		log.Fatalf("Failed to start metrics server on %s: %v", addr, err)
	}
}
