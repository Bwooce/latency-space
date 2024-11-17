package main

// Global configuration variables
var (
	solarSystem = map[string]*CelestialBody{
		"mercury": {
			Distance:      77.3,
			UDPForward:    "1.1.1.1:53",
			TCPForward:    "example.com:80",
			BandwidthKbps: DSN_HIGH,
			RateLimit:     600,
			Moons:         make(map[string]*CelestialBody),
		},
		// ... rest of your solar system configuration ...
	}

	spacecraft = map[string]*CelestialBody{
		"voyager1": {
			Distance:      23000.0,
			UDPForward:    "1.1.1.1:53",
			TCPForward:    "example.com:80",
			BandwidthKbps: 32,
			RateLimit:     5,
		},
		// ... rest of your spacecraft configuration ...
	}
)
