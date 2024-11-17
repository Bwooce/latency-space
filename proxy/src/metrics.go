// proxy/src/metrics.go
package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"time"
)

type MetricsCollector struct {
	requestDuration *prometheus.HistogramVec
	requestsTotal   *prometheus.CounterVec
	bandwidthUsage  *prometheus.CounterVec
	udpPackets      *prometheus.CounterVec
}

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
			[]string{"body"},
		),
	}

	// Register metrics
	prometheus.MustRegister(m.requestDuration)
	prometheus.MustRegister(m.requestsTotal)
	prometheus.MustRegister(m.bandwidthUsage)
	prometheus.MustRegister(m.udpPackets)

	return m
}

func (m *MetricsCollector) RecordRequest(body, reqType string, duration time.Duration) {
	m.requestDuration.WithLabelValues(body, reqType).Observe(duration.Seconds())
	m.requestsTotal.WithLabelValues(body, reqType).Inc()
}

func (m *MetricsCollector) TrackBandwidth(body string, bytes int64) {
	if bytes > 0 {
		m.bandwidthUsage.WithLabelValues(body, "out").Add(float64(bytes))
	}
}

func (m *MetricsCollector) RecordUDPPacket(body string, bytes int64) {
	m.udpPackets.WithLabelValues(body).Inc()
	m.bandwidthUsage.WithLabelValues(body, "in").Add(float64(bytes))
}

func (m *MetricsCollector) ServeMetrics(addr string) {
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(addr, nil)
}
