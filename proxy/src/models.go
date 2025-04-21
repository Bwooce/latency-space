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
	Distance      float64 // used for moons only
	BandwidthKbps int     // bandwidth limit in Kbps
	RateLimit     int     // requests per minute
	Moons         map[string]*CelestialBody
}

func calculateLatency(distanceKm float64) time.Duration {
	seconds := distanceKm / speedOfLight
	return time.Duration(seconds * float64(time.Second))
}

// SOCKS constants
const (
	SOCKS5_VERSION = 0x05

	// Authentication methods
	SOCKS5_NO_AUTH                = 0x00
	SOCKS5_AUTH_GSSAPI            = 0x01
	SOCKS5_AUTH_USERNAME_PASSWORD = 0x02
	SOCKS5_AUTH_NO_ACCEPTABLE     = 0xFF

	// Command types
	SOCKS5_CMD_CONNECT      = 0x01
	SOCKS5_CMD_BIND         = 0x02
	SOCKS5_CMD_UDP_ASSOCIATE = 0x03

	// Address types
	SOCKS5_ADDR_IPV4   = 0x01
	SOCKS5_ADDR_DOMAIN = 0x03
	SOCKS5_ADDR_IPV6   = 0x04

	// Reply codes
	SOCKS5_REP_SUCCESS            = 0x00
	SOCKS5_REP_GENERAL_FAILURE    = 0x01
	SOCKS5_REP_CONN_NOT_ALLOWED   = 0x02
	SOCKS5_REP_NETWORK_UNREACHABLE = 0x03
	SOCKS5_REP_HOST_UNREACHABLE    = 0x04
	SOCKS5_REP_CONN_REFUSED       = 0x05
	SOCKS5_REP_TTL_EXPIRED        = 0x06
	SOCKS5_REP_CMD_NOT_SUPPORTED  = 0x07
	SOCKS5_REP_ADDR_NOT_SUPPORTED = 0x08
)