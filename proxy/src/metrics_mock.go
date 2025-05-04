// proxy/src/metrics_mock.go
package main

import (
	"github.com/prometheus/client_golang/prometheus"
)

// NewTestMetricsCollector creates a metrics collector specifically for testing
// that won't conflict with existing metrics registrations.
func NewTestMetricsCollector() *MetricsCollector {
	// Create unregistered metrics using NewHistogramVec and NewCounterVec
	// without registering them to avoid prometheus registration conflicts
	requestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "test_request_duration_seconds",
			Help: "Time spent processing request (test)",
		},
		[]string{"body", "type"},
	)

	requestsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "test_requests_total",
			Help: "Total number of requests (test)",
		},
		[]string{"body", "type"},
	)

	bandwidthUsage := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "test_bandwidth_bytes_total",
			Help: "Total bandwidth usage in bytes (test)",
		},
		[]string{"body", "direction"},
	)

	udpPackets := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "test_udp_packets_total",
			Help: "Total UDP packets processed (test)",
		},
		[]string{"body"}, // Label by celestial body
	)

	// Create the metrics collector without registering the metrics
	return &MetricsCollector{
		requestDuration: requestDuration,
		requestsTotal:   requestsTotal,
		bandwidthUsage:  bandwidthUsage,
		udpPackets:      udpPackets,
	}
}
