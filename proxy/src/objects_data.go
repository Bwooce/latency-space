package main

// InitSolarSystemObjects initializes all objects in the solar system
func InitSolarSystemObjects() []CelestialObject {
	objects := []CelestialObject{
		// Sun
		{
			Name:     "Sun",
			Type:     "star",
			Radius:   695700.0,
			Mass:     1.989e30,  // kg
		},
		
		// PLANETS
		// Mercury
		{
			Name:     "Mercury",
			Type:     "planet",
			ParentName: "Sun",
			Radius:   2439.7,
			A:        0.38709843,
			E:        0.20563661,
			I:        7.00559432,
			L:        252.25166724,
			LP:       77.45771895,
			N:        48.33961819,
			dA:       0.00000000,
			dE:       0.00002123,
			dI:       -0.00590158,
			dL:       149472.67486623,
			dLP:      0.15940013,
			dN:       -0.12214182,
			b:        87.969,    // Orbital period (days)
			c:        0.2056,    // Eccentricity for perturbation terms
			s:        0.1257,    // Sin term coefficient
			f:        4.0923,    // Mean motion (degrees/day)
			Mass:     3.301e23,  // kg
		},
		
		// Venus
		{
			Name:     "Venus",
			Type:     "planet",
			ParentName: "Sun",
			Radius:   6051.8,
			A:        0.72333566,
			E:        0.00677672,
			I:        3.39467605,
			L:        181.97970850,
			LP:       131.76755713,
			N:        76.67984255,
			dA:       0.00000390,
			dE:       -0.00004107,
			dI:       -0.00078890,
			dL:       58517.81538729,
			dLP:      0.05679648,
			dN:       -0.27769418,
			b:        224.701,   // Orbital period (days)
			c:        0.0067,    // Eccentricity for perturbation terms
			s:        0.0531,    // Sin term coefficient
			f:        1.6021,    // Mean motion (degrees/day)
			Mass:     4.867e24,  // kg
		},
		
		// Earth
		{
			Name:     "Earth",
			Type:     "planet",
			ParentName: "Sun",
			Radius:   6378.137,
			A:        1.00000261,
			E:        0.01671123,
			I:        -0.00001531,
			L:        100.46457166,
			LP:       102.93768193,
			N:        0.0,
			dA:       0.00000562,
			dE:       -0.00004392,
			dI:       -0.01294668,
			dL:       35999.37306329,
			dLP:      0.32327364,
			dN:       0.0,
			b:        365.256,   // Orbital period (days)
			c:        0.0167,    // Eccentricity for perturbation terms
			s:        0.0148,    // Sin term coefficient
			f:        0.9856,    // Mean motion (degrees/day)
			Mass:     5.972e24,  // kg
		},
		
		// Mars
		{
			Name:     "Mars",
			Type:     "planet",
			ParentName: "Sun",
			Radius:   3396.2,
			A:        1.52371034,
			E:        0.09339410,
			I:        1.84969142,
			L:        -4.55343205,
			LP:       -23.94362959,
			N:        49.55953891,
			dA:       0.00001847,
			dE:       0.00007882,
			dI:       -0.00813131,
			dL:       19140.30268499,
			dLP:      0.44441088,
			dN:       -0.29257343,
			b:        686.98,    // Orbital period (days)
			c:        0.0934,    // Eccentricity for perturbation terms
			s:        0.0518,    // Sin term coefficient
			f:        0.5240,    // Mean motion (degrees/day)
			Mass:     6.417e23,  // kg
		},
		
		// Jupiter
		{
			Name:     "Jupiter",
			Type:     "planet",
			ParentName: "Sun",
			Radius:   71492.0,
			A:        5.20288700,
			E:        0.04838624,
			I:        1.30439695,
			L:        34.39644051,
			LP:       14.72847983,
			N:        100.47390909,
			dA:       -0.00011607,
			dE:       -0.00013253,
			dI:       -0.00183714,
			dL:       3034.74612775,
			dLP:      0.21252668,
			dN:       0.20469106,
			b:        4332.59,   // Orbital period (days)
			c:        0.0484,    // Eccentricity for perturbation terms
			s:        0.0227,    // Sin term coefficient
			f:        0.0831,    // Mean motion (degrees/day)
			Mass:     1.898e27,  // kg
		},
		
		// Saturn
		{
			Name:     "Saturn",
			Type:     "planet",
			ParentName: "Sun",
			Radius:   60268.0,
			A:        9.53667594,
			E:        0.05386179,
			I:        2.48599187,
			L:        49.95424423,
			LP:       92.59887831,
			N:        113.66242448,
			dA:       -0.00125060,
			dE:       -0.00050991,
			dI:       0.00193609,
			dL:       1222.49362201,
			dLP:      -0.41897216,
			dN:       -0.28867794,
			b:        10759.22,  // Orbital period (days)
			c:        0.0539,    // Eccentricity for perturbation terms
			s:        0.0434,    // Sin term coefficient
			f:        0.0334,    // Mean motion (degrees/day)
			Mass:     5.683e26,  // kg
		},
		
		// Uranus
		{
			Name:     "Uranus",
			Type:     "planet",
			ParentName: "Sun",
			Radius:   25559.0,
			A:        19.18916464,
			E:        0.04725744,
			I:        0.77263783,
			L:        313.23810451,
			LP:       170.95427630,
			N:        74.01692503,
			dA:       -0.00196176,
			dE:       -0.00004397,
			dI:       -0.00242939,
			dL:       428.48202785,
			dLP:      0.40805281,
			dN:       0.04240589,
			b:        30685.4,   // Orbital period (days)
			c:        0.0473,    // Eccentricity for perturbation terms
			s:        0.0134,    // Sin term coefficient
			f:        0.0117,    // Mean motion (degrees/day)
			Mass:     8.681e25,  // kg
		},
		
		// Neptune
		{
			Name:     "Neptune",
			Type:     "planet",
			ParentName: "Sun",
			Radius:   24764.0,
			A:        30.06992276,
			E:        0.00859048,
			I:        1.77004347,
			L:        -55.12002969,
			LP:       44.96476227,
			N:        131.78422574,
			dA:       0.00026291,
			dE:       0.00005105,
			dI:       0.00035372,
			dL:       218.45945325,
			dLP:      -0.32241464,
			dN:       -0.00508664,
			b:        60189.0,   // Orbital period (days)
			c:        0.0086,    // Eccentricity for perturbation terms
			s:        0.0309,    // Sin term coefficient
			f:        0.0060,    // Mean motion (degrees/day)
			Mass:     1.024e26,  // kg
		},
		
		// DWARF PLANETS
		// Pluto
		{
			Name:     "Pluto",
			Type:     "dwarf_planet",
			ParentName: "Sun",
			Radius:   1188.3,
			A:        39.48211675,
			E:        0.24882730,
			I:        17.14001206,
			L:        238.92881780,
			LP:       224.06891629,
			N:        110.30393684,
			dA:       -0.00031596,
			dE:       0.00005170,
			dI:       0.00004818,
			dL:       145.20780515,
			dLP:      -0.04062942,
			dN:       -0.01183482,
			Period:   90560.0,    // Orbital period (days)
			Mass:     1.303e22,   // kg
		},
		
		// Ceres
		{
			Name:     "Ceres",
			Type:     "dwarf_planet",
			ParentName: "Sun",
			Radius:   469.7,
			A:        2.7653,
			E:        0.0758,
			I:        10.586,
			L:        95.989,
			LP:       73.597,
			N:        80.393,
			dL:       1680.5,     // Approx value for mean motion
			Period:   1681.0,     // Orbital period (days)
			Mass:     9.393e20,   // kg
		},
		
		// Eris
		{
			Name:     "Eris",
			Type:     "dwarf_planet",
			ParentName: "Sun",
			Radius:   1163.0,
			A:        67.864,
			E:        0.44177,
			I:        44.040,
			L:        204.16,
			LP:       151.639,
			N:        35.951,
			dL:       68.74,      // Approx value for mean motion
			Period:   203830.0,   // Orbital period (days)
			Mass:     1.66e22,    // kg
		},
		
		// Haumea
		{
			Name:     "Haumea",
			Type:     "dwarf_planet",
			ParentName: "Sun",
			Radius:   816.0,     // Equivalent spherical radius
			A:        43.335,
			E:        0.19126,
			I:        28.21,
			L:        240.582,
			LP:       239.512,
			N:        121.900,
			dL:       108.21,    // Approx value for mean motion
			Period:   104025.0,  // Orbital period (days)
			Mass:     4.006e21,  // kg
		},
		
		// Makemake
		{
			Name:     "Makemake",
			Type:     "dwarf_planet",
			ParentName: "Sun",
			Radius:   715.0,
			A:        45.791,
			E:        0.16254,
			I:        29.011,
			L:        268.05,
			LP:       296.534,
			N:        79.382,
			dL:       102.13,    // Approx value for mean motion
			Period:   111845.0,  // Orbital period (days)
			Mass:     3.1e21,    // kg
		},
		
		// MOONS
		// Earth's Moon
		{
			Name:       "Moon",
			Type:       "moon",
			ParentName: "Earth",
			Radius:     1737.4,
			A:          384399.0,   // Semi-major axis in km
			E:          0.0549,
			I:          5.145,
			L:          375.7,      // Mean longitude at epoch
			N:          125.08,     // Longitude of ascending node
			W:          318.15,     // Argument of perigee
			dL:         13.176358 * DAYS_PER_CENTURY, // Degrees per century
			dN:         -0.05295 * DAYS_PER_CENTURY,
			dW:         0.11140 * DAYS_PER_CENTURY,
			Period:     27.321661,
			Mass:       7.342e22,   // kg
		},
		
		// Mars' moons
		{
			Name:       "Phobos",
			Type:       "moon",
			ParentName: "Mars",
			Radius:     11.1,
			A:          9376.0,     // km
			E:          0.0151,
			I:          1.093,
			N:          208.2,
			W:          157.1,
			L:          165.8,
			dL:         1128.8 * 360.0 / 365.25 * DAYS_PER_CENTURY, // Approx value
			Period:     0.31891,
			Mass:       1.08e16,    // kg
		},
		
		{
			Name:       "Deimos",
			Type:       "moon",
			ParentName: "Mars",
			Radius:     6.2,
			A:          23458.0,    // km
			E:          0.00033,
			I:          1.791,
			N:          24.5,
			W:          260.7,
			L:          286.5,
			dL:         285.16 * 360.0 / 365.25 * DAYS_PER_CENTURY, // Approx value
			Period:     1.26244,
			Mass:       1.8e15,     // kg
		},
		
		// Jupiter's Galilean moons
		{
			Name:       "Io",
			Type:       "moon",
			ParentName: "Jupiter",
			Radius:     1821.5,
			A:          421800.0,   // km
			E:          0.0041,
			I:          0.05,
			N:          43.977,
			W:          84.129,
			L:          342.02,
			dL:         203.4889538 * 360.0 / 365.25 * DAYS_PER_CENTURY, // Approx value
			Period:     1.769138,
			Mass:       8.932e22,   // kg
		},
		
		{
			Name:       "Europa",
			Type:       "moon",
			ParentName: "Jupiter",
			Radius:     1560.8,
			A:          671100.0,   // km
			E:          0.0094,
			I:          0.47,
			N:          219.106,
			W:          88.970,
			L:          171.02,
			dL:         101.3747235 * 360.0 / 365.25 * DAYS_PER_CENTURY, // Approx value
			Period:     3.551181,
			Mass:       4.8e22,     // kg
		},
		
		{
			Name:       "Ganymede",
			Type:       "moon",
			ParentName: "Jupiter",
			Radius:     2631.2,
			A:          1070400.0,  // km
			E:          0.0013,
			I:          0.21,
			N:          63.552,
			W:          192.417,
			L:          317.54,
			dL:         50.3176081 * 360.0 / 365.25 * DAYS_PER_CENTURY, // Approx value
			Period:     7.154553,
			Mass:       1.4819e23,  // kg
		},
		
		{
			Name:       "Callisto",
			Type:       "moon",
			ParentName: "Jupiter",
			Radius:     2410.3,
			A:          1882700.0,  // km
			E:          0.0074,
			I:          0.51,
			N:          298.848,
			W:          52.643,
			L:          181.41,
			dL:         21.5710715 * 360.0 / 365.25 * DAYS_PER_CENTURY, // Approx value
			Period:     16.689018,
			Mass:       1.076e23,   // kg
		},
		
		// Saturn's major moons
		{
			Name:       "Titan",
			Type:       "moon",
			ParentName: "Saturn",
			Radius:     2574.7,
			A:          1221870.0,  // km
			E:          0.0288,
			I:          0.34854,
			N:          28.0212,
			W:          186.5442,
			L:          127.64,
			dL:         22.577 * 360.0 / 365.25 * DAYS_PER_CENTURY, // Approx value
			Period:     15.945,
			Mass:       1.3455e23,  // kg
		},
		
		{
			Name:       "Enceladus",
			Type:       "moon",
			ParentName: "Saturn",
			Radius:     252.1,
			A:          238042.0,   // km
			E:          0.0047,
			I:          0.019,
			N:          337.1,
			W:          337.8,
			L:          26.7,
			dL:         262.7318996 * 360.0 / 365.25 * DAYS_PER_CENTURY, // Approx value
			Period:     1.370218,
			Mass:       1.08e20,    // kg
		},
		
		{
			Name:       "Mimas",
			Type:       "moon",
			ParentName: "Saturn",
			Radius:     198.2,
			A:          185539.0,   // km
			E:          0.0196,
			I:          1.574,
			N:          333.2,
			W:          210.8,
			L:          218.0,
			dL:         381.9944943 * 360.0 / 365.25 * DAYS_PER_CENTURY, // Approx value
			Period:     0.942422,
			Mass:       3.75e19,    // kg
		},
		
		// Additional notable moons
		{
			Name:       "Rhea",
			Type:       "moon",
			ParentName: "Saturn",
			Radius:     763.8,
			A:          527108.0,   // km
			E:          0.0012,
			I:          0.345,
			N:          345.487,
			W:          162.1,
			L:          171.4,
			dL:         79.6900478 * 360.0 / 365.25 * DAYS_PER_CENTURY, // Approx value
			Period:     4.518212,
			Mass:       2.306e21,   // kg
		},
		
		{
			Name:       "Titania",
			Type:       "moon",
			ParentName: "Uranus",
			Radius:     788.9,
			A:          435910.0,   // km
			E:          0.0011,
			I:          0.340,
			N:          262.772,
			W:          284.400,
			L:          24.614,
			dL:         41.351431 * 360.0 / 365.25 * DAYS_PER_CENTURY, // Approx value
			Period:     8.705872,
			Mass:       3.4e21,    // kg
		},
		
		{
			Name:       "Triton",
			Type:       "moon",
			ParentName: "Neptune",
			Radius:     1353.4,
			A:          354759.0,   // km
			E:          0.000016,
			I:          156.885,    // Retrograde orbit
			N:          177.612,
			W:          237.234,
			L:          267.457,
			dL:         -61.2572637 * 360.0 / 365.25 * DAYS_PER_CENTURY, // Negative for retrograde
			Period:     5.876854,
			Mass:       2.14e22,    // kg
		},
		
		{
			Name:       "Charon",
			Type:       "moon",
			ParentName: "Pluto",
			Radius:     606.0,
			A:          19591.0,    // km
			E:          0.0002,
			I:          0.001,      // relative to Pluto's equator
			N:          223.0,
			W:          188.0,
			L:          56.0,
			dL:         56.3625225 * 360.0 / 365.25 * DAYS_PER_CENTURY, // Approx value
			Period:     6.3872304,
			Mass:       1.586e21,   // kg
		},
		
		// SPACECRAFT
		// Active deep space missions with transmitters
		{
			Name:           "Voyager 1",
			Type:           "spacecraft",
			ParentName:     "Sun",
			Radius:         0.01,     // Approximate spacecraft size
			A:              140.81,    // AU, as of 2025 (approximate)
			E:              0.988,     // High eccentricity for escape trajectory
			I:              35.13,     // Degrees
			LaunchDate:     "1977-09-05",
			TransmitterActive: true,
			FrequencyMHz:   8415.0,    // X-band downlink frequency
			MissionStatus:  "active",
		},
		
		{
			Name:           "Voyager 2",
			Type:           "spacecraft",
			ParentName:     "Sun",
			Radius:         0.01,     // Approximate spacecraft size
			A:              116.43,    // AU, as of 2025 (approximate)
			E:              0.981,     // High eccentricity for escape trajectory
			I:              46.2,      // Degrees
			LaunchDate:     "1977-08-20",
			TransmitterActive: true,
			FrequencyMHz:   8415.0,    // X-band downlink frequency
			MissionStatus:  "active",
		},
		
		{
			Name:           "New Horizons",
			Type:           "spacecraft",
			ParentName:     "Sun",
			Radius:         0.005,    // Approximate spacecraft size
			A:              45.21,     // AU, as of 2025 (approximate)
			E:              0.852,     // High eccentricity for escape trajectory
			I:              2.45,      // Degrees
			LaunchDate:     "2006-01-19",
			TransmitterActive: true,
			FrequencyMHz:   8438.0,    // X-band downlink frequency
			MissionStatus:  "active",
		},
		
		{
			Name:           "Parker Solar Probe",
			Type:           "spacecraft",
			ParentName:     "Sun",
			Radius:         0.005,    // Approximate spacecraft size
			A:              0.294,     // AU, highly elliptical orbit
			E:              0.860,     // Very high eccentricity
			I:              3.4,       // Degrees
			LaunchDate:     "2018-08-12",
			TransmitterActive: true,
			FrequencyMHz:   8421.0,    // X-band downlink frequency
			MissionStatus:  "active",
		},
		
		{
			Name:           "JWST",
			Type:           "spacecraft",
			ParentName:     "Sun",  // Actually orbits L2 point
			Radius:         0.01,    // Approximate spacecraft size
			A:              1.01,    // AU, L2 point distance
			E:              0.002,   // Nearly circular halo orbit
			I:              0.1,     // Small inclination
			LaunchDate:     "2021-12-25",
			TransmitterActive: true,
			FrequencyMHz:   25900.0, // Ka-band downlink frequency
			MissionStatus:  "active",
		},
		
		{
			Name:           "Mars Perseverance",
			Type:           "spacecraft",
			ParentName:     "Mars",
			Radius:         0.003,    // Approximate rover size
			A:              3396.2 + 0.01, // Mars radius + surface elevation
			E:              0.0,       // On surface
			I:              0.0,       // On surface
			LaunchDate:     "2020-07-30",
			TransmitterActive: true,
			FrequencyMHz:   8426.0,    // X-band downlink frequency
			MissionStatus:  "active",
		},
		
		// ASTEROIDS
		// Some notable main belt asteroids
		{
			Name:       "Vesta",
			Type:       "asteroid",
			ParentName: "Sun",
			Radius:     262.7,
			A:          2.361534,  // AU
			E:          0.089179,
			I:          7.14043,
			L:          103.85,
			LP:         149.85,
			N:          103.85,
			Period:     1325.75,   // days
			Mass:       2.59e20,   // kg
		},
		
		{
			Name:       "Pallas",
			Type:       "asteroid",
			ParentName: "Sun",
			Radius:     256.0,
			A:          2.772176,  // AU
			E:          0.231417,
			I:          34.83923,
			L:          309.93,
			LP:         310.95,
			N:          173.08,
			Period:     1685.98,   // days
			Mass:       2.04e20,   // kg
		},
		
		{
			Name:       "Hygiea",
			Type:       "asteroid",
			ParentName: "Sun",
			Radius:     217.5,
			A:          3.1370,    // AU
			E:          0.1143,
			I:          3.8383,
			L:          312.95,
			LP:         312.66,
			N:          283.41,
			Period:     2029.8,    // days
			Mass:       8.67e19,   // kg
		},
		
		// Near-Earth asteroids
		{
			Name:       "Bennu",
			Type:       "asteroid",
			ParentName: "Sun",
			Radius:     0.2625,
			A:          1.126391,  // AU
			E:          0.203731,
			I:          6.0349,
			L:          101.7039,
			LP:         2.7348,
			N:          66.2231,
			Period:     436.65,    // days
			Mass:       7.329e10,  // kg
		},
		
		{
			Name:       "Apophis",
			Type:       "asteroid",
			ParentName: "Sun",
			Radius:     0.185,
			A:          0.9224,    // AU
			E:          0.1911,
			I:          3.3366,
			L:          126.3992,
			LP:         126.3991,
			N:          204.0,
			Period:     323.6,     // days
			Mass:       6.1e10,    // kg
		},
	}
	
	// Normalize angles
	for i := range objects {
		if objects[i].Type != "star" && objects[i].Type != "spacecraft" {
			objects[i].L = normalizeDegrees(objects[i].L)
		}
	}
	
	return objects
}