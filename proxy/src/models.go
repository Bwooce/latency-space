package main

import (
	"time"
)

const (
	speedOfLight = 299792.458 // km/s

	// Bandwidth tiers (kbps)
	DSN_HIGH    = 2048 // 2 Mbps
	DSN_MED     = 1024 // 1 Mbps
	DSN_LOW     = 512  // 512 Kbps
	DSN_OUTER   = 256  // 256 Kbps
	DSN_DISTANT = 128  // 128 Kbps
)

type CelestialBody struct {
	Distance      float64 // millions of km (average)
	UDPForward    string
	TCPForward    string
	BandwidthKbps int // bandwidth limit in Kbps
	RateLimit     int // requests per minute
	Moons         map[string]*CelestialBody
}

func calculateLatency(distanceKm float64) time.Duration {
	seconds := distanceKm / speedOfLight
	return time.Duration(seconds * float64(time.Second))
}
