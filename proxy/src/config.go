// proxy/src/config.go
package main

// Complete solar system configuration
var solarSystem = map[string]*CelestialBody{
	"mercury": {
		BandwidthKbps: DSN_HIGH,
		RateLimit:     600,
		Moons:         make(map[string]*CelestialBody), // Mercury has no moons
	},

	"venus": {
		BandwidthKbps: DSN_HIGH,
		RateLimit:     600,
		Moons:         make(map[string]*CelestialBody), // Venus has no moons
	},

	"earth": {
		BandwidthKbps: DSN_HIGH,
		RateLimit:     1200,
		Moons: map[string]*CelestialBody{
			"moon": {
				Distance:      0.384,
				BandwidthKbps: DSN_HIGH,
				RateLimit:     1000,
			},
		},
	},

	"mars": {
		Distance:      225.0,
		BandwidthKbps: DSN_MED,
		RateLimit:     300,
		Moons: map[string]*CelestialBody{
			"phobos": {
				Distance:      0.009377,
				BandwidthKbps: DSN_MED,
				RateLimit:     300,
			},
			"deimos": {
				Distance:      0.023460,
				BandwidthKbps: DSN_MED,
				RateLimit:     300,
			},
		},
	},

	"jupiter": {
		BandwidthKbps: DSN_LOW,
		RateLimit:     120,
		Moons: map[string]*CelestialBody{
			"io": {
				Distance:      0.421700,
				BandwidthKbps: DSN_LOW,
				RateLimit:     120,
			},
			"europa": {
				Distance:      0.671100,
				BandwidthKbps: DSN_LOW,
				RateLimit:     120,
			},
			"ganymede": {
				Distance:      1.070400,
				BandwidthKbps: DSN_LOW,
				RateLimit:     120,
			},
			"callisto": {
				Distance:      1.882700,
				BandwidthKbps: DSN_LOW,
				RateLimit:     120,
			},
		},
	},

	"saturn": {
		BandwidthKbps: DSN_OUTER,
		RateLimit:     60,
		Moons: map[string]*CelestialBody{
			"titan": {
				Distance:      1.221870,
				BandwidthKbps: DSN_OUTER,
				RateLimit:     60,
			},
			"enceladus": {
				Distance:      0.238040,
				BandwidthKbps: DSN_OUTER,
				RateLimit:     60,
			},
			"mimas": {
				Distance:      0.185520,
				BandwidthKbps: DSN_OUTER,
				RateLimit:     60,
			},
			"iapetus": {
				Distance:      3.561300,
				BandwidthKbps: DSN_OUTER,
				RateLimit:     60,
			},
			"rhea": {
				Distance:      0.527040,
				BandwidthKbps: DSN_OUTER,
				RateLimit:     60,
			},
		},
	},

	"uranus": {
		BandwidthKbps: DSN_OUTER,
		RateLimit:     30,
		Moons: map[string]*CelestialBody{
			"miranda": {
				Distance:      0.129900,
				BandwidthKbps: DSN_OUTER,
				RateLimit:     30,
			},
			"ariel": {
				Distance:      0.190900,
				BandwidthKbps: DSN_OUTER,
				RateLimit:     30,
			},
			"umbriel": {
				Distance:      0.266000,
				BandwidthKbps: DSN_OUTER,
				RateLimit:     30,
			},
			"titania": {
				Distance:      0.436300,
				BandwidthKbps: DSN_OUTER,
				RateLimit:     30,
			},
			"oberon": {
				Distance:      0.583500,
				BandwidthKbps: DSN_OUTER,
				RateLimit:     30,
			},
		},
	},

	"neptune": {
		BandwidthKbps: DSN_DISTANT,
		RateLimit:     15,
		Moons: map[string]*CelestialBody{
			"triton": {
				Distance:      0.354760,
				BandwidthKbps: DSN_DISTANT,
				RateLimit:     15,
			},
			"naiad": {
				Distance:      0.048227,
				BandwidthKbps: DSN_DISTANT,
				RateLimit:     15,
			},
			"nereid": {
				Distance:      5.513400,
				BandwidthKbps: DSN_DISTANT,
				RateLimit:     15,
			},
		},
	},

	"pluto": {
		BandwidthKbps: DSN_DISTANT,
		RateLimit:     10,
		Moons: map[string]*CelestialBody{
			"charon": {
				Distance:      0.019571,
				BandwidthKbps: DSN_DISTANT,
				RateLimit:     10,
			},
			"nix": {
				Distance:      0.048675,
				BandwidthKbps: DSN_DISTANT,
				RateLimit:     10,
			},
			"hydra": {
				Distance:      0.064738,
				BandwidthKbps: DSN_DISTANT,
				RateLimit:     10,
			},
		},
	},
}

// Spacecraft and space station configurations
var spacecraft = map[string]*CelestialBody{
	"voyager1": {
		BandwidthKbps: 32, // Very limited bandwidth
		RateLimit:     5,
	},

	"voyager2": {
		BandwidthKbps: 32,
		RateLimit:     5,
	},

	"newhorizons": {
		BandwidthKbps: 64,
		RateLimit:     10,
	},

	"jwst": { // James Webb Space Telescope
		BandwidthKbps: DSN_HIGH,
		RateLimit:     600,
	},

	"iss": { // International Space Station
		BandwidthKbps: DSN_HIGH,
		RateLimit:     1000,
	},

	"perseverance": { // Mars rover
		BandwidthKbps: DSN_MED,
		RateLimit:     300,
	},
}