package main

import (
	"math"
	"strings"
	"sync"
	"time"
)

// planetaryOrbits contains parameters needed to calculate current distances
var planetaryOrbits = map[string]struct {
	SemiMajorAxis float64 // in millions of km
	Eccentricity  float64
	OrbitalPeriod float64 // in Earth days
	Inclination   float64 // in degrees
	LongAscNode   float64 // longitude of ascending node in degrees
	LongPeri      float64 // longitude of perihelion in degrees
	MeanLong      float64 // mean longitude at epoch in degrees
}{
	"mercury": {57.9, 0.2056, 88.0, 7.0, 48.3, 77.5, 252.3},
	"venus":   {108.2, 0.0068, 224.7, 3.4, 76.7, 131.6, 181.2},
	"earth":   {149.6, 0.0167, 365.2, 0.0, 174.9, 102.9, 100.5},
	"mars":    {227.9, 0.0934, 687.0, 1.8, 49.6, 336.0, 355.5},
	"jupiter": {778.6, 0.0484, 4331.0, 1.3, 100.5, 14.8, 34.4},
	"saturn":  {1434.0, 0.0542, 10747.0, 2.5, 113.7, 92.4, 50.1},
	"uranus":  {2871.0, 0.0472, 30589.0, 0.8, 74.0, 170.7, 314.1},
	"neptune": {4495.0, 0.0086, 59800.0, 1.8, 131.8, 44.9, 304.3},
	"pluto":   {5906.0, 0.2488, 90560.0, 17.1, 110.3, 224.1, 238.9},
}

// spacecraftTrajectories contains parameters to calculate spacecraft positions
var spacecraftTrajectories = map[string]struct {
	BaseDistance float64 // in millions of km
	VelocityKmps float64 // velocity in km/s
	LaunchDate   time.Time
}{
	"voyager1":     {23000.0, 0.017, time.Date(1977, 9, 5, 0, 0, 0, 0, time.UTC)},
	"voyager2":     {19000.0, 0.016, time.Date(1977, 8, 20, 0, 0, 0, 0, time.UTC)},
	"newhorizons":  {7000.0, 0.014, time.Date(2006, 1, 19, 0, 0, 0, 0, time.UTC)},
	"jwst":         {1.5, 0.0, time.Date(2021, 12, 25, 0, 0, 0, 0, time.UTC)}, // Stationary at L2
	"iss":          {0.0004, 0.0, time.Time{}},                                // Orbit is negligible
	"perseverance": {225.0, 0.0, time.Time{}},                                 // On Mars
}

var (
	distanceCacheMu    sync.RWMutex
	distanceCache      = make(map[string]float64)
	lastDistanceUpdate time.Time
)

// Convert degrees to radians
func deg2rad(deg float64) float64 {
	return deg * math.Pi / 180.0
}

// getOrbitalPosition calculates the 3D position of a planet
func getOrbitalPosition(orbit struct {
	SemiMajorAxis float64
	Eccentricity  float64
	OrbitalPeriod float64
	Inclination   float64
	LongAscNode   float64
	LongPeri      float64
	MeanLong      float64
}, daysSinceEpoch float64) (x, y, z float64) {
	// Convert degrees to radians for calculations
	incl := deg2rad(orbit.Inclination)
	node := deg2rad(orbit.LongAscNode)
	peri := deg2rad(orbit.LongPeri)
	meanLong := deg2rad(orbit.MeanLong)
	
	// Calculate centuries since J2000.0
	T := daysSinceEpoch / 36525.0
	
	// Calculate mean anomaly
	// n = 2*pi/period (mean motion)
	n := 2.0 * math.Pi / orbit.OrbitalPeriod
	
	// Mean anomaly at epoch
	M0 := meanLong - peri
	// Current mean anomaly
	M := M0 + n*daysSinceEpoch
	M = math.Mod(M, 2.0*math.Pi)
	if M < 0 {
		M += 2.0 * math.Pi
	}
	
	// Solve Kepler's equation (better approximation)
	E := M
	dE := 1.0
	for i := 0; i < 10 && math.Abs(dE) > 1e-6; i++ {
		dE = (M + orbit.Eccentricity*math.Sin(E) - E) / (1.0 - orbit.Eccentricity*math.Cos(E))
		E += dE
	}
	
	// Calculate true anomaly
	nu := 2.0 * math.Atan2(math.Sqrt(1.0+orbit.Eccentricity)*math.Sin(E/2.0), 
						   math.Sqrt(1.0-orbit.Eccentricity)*math.Cos(E/2.0))
	
	// Calculate heliocentric distance
	r := orbit.SemiMajorAxis * (1.0 - orbit.Eccentricity*math.Cos(E))
	
	// Calculate position in orbital plane
	x = r * math.Cos(nu)
	y = r * math.Sin(nu)
	z = 0.0
	
	// Argument of perihelion
	w := peri - node
	
	// Rotation to the ecliptic plane using orbital elements
	// First rotate around z-axis by -w
	xw := x*math.Cos(-w) - y*math.Sin(-w)
	yw := x*math.Sin(-w) + y*math.Cos(-w)
	zw := z
	
	// Then rotate around x-axis by -i (inclination)
	xi := xw
	yi := yw*math.Cos(-incl) - zw*math.Sin(-incl)
	zi := yw*math.Sin(-incl) + zw*math.Cos(-incl)
	
	// Finally rotate around z-axis by -node
	xecl := xi*math.Cos(-node) - yi*math.Sin(-node)
	yecl := xi*math.Sin(-node) + yi*math.Cos(-node)
	zecl := zi
	
	return xecl, yecl, zecl
}

// updateCelestialDistances calculates current distances from Earth to celestial bodies
func updateCelestialDistances() {
	distanceCacheMu.Lock()
	defer distanceCacheMu.Unlock()
	
	// Only update once per hour to avoid excessive calculations
	if time.Since(lastDistanceUpdate) < time.Hour {
		return
	}
	
	// Current time for calculations
	now := time.Now()
	lastDistanceUpdate = now
	
	// Reference time for orbital calculations (J2000 epoch)
	epoch := time.Date(2000, 1, 1, 12, 0, 0, 0, time.UTC)
	daysSinceEpoch := now.Sub(epoch).Hours() / 24.0
	
	// Get Earth's position first
	earthX, earthY, earthZ := getOrbitalPosition(planetaryOrbits["earth"], daysSinceEpoch)
	
	// For Earth, distance is 0 (reference point)
	distanceCache["earth"] = 0.0
	
	// Calculate planet positions and distances from Earth
	for name, orbit := range planetaryOrbits {
		if name == "earth" {
			continue // Already handled
		}
		
		// Get planet position
		x, y, z := getOrbitalPosition(orbit, daysSinceEpoch)
		
		// Calculate distance from Earth (3D Euclidean distance)
		dx := x - earthX
		dy := y - earthY
		dz := z - earthZ
		distance := math.Sqrt(dx*dx + dy*dy + dz*dz)
		
		// Update distance in cache
		distanceCache[name] = distance
		
		// Update moons (their distances relative to their planets remain constant)
		for moonName := range solarSystem[name].Moons {
			moonDistance := solarSystem[name].Moons[moonName].Distance
			distanceCache[moonName+"."+name] = distance + moonDistance
		}
	}
	
	// Update spacecraft positions
	for name, trajectory := range spacecraftTrajectories {
		if trajectory.LaunchDate.IsZero() {
			// For static spacecraft, use the base distance
			distanceCache[name] = trajectory.BaseDistance
			continue
		}
		
		// For moving spacecraft, calculate additional distance based on velocity
		daysSinceLaunch := now.Sub(trajectory.LaunchDate).Hours() / 24.0
		additionalDist := (trajectory.VelocityKmps * 86400 * daysSinceLaunch) / 1e6 // convert to millions km
		distanceCache[name] = trajectory.BaseDistance + additionalDist
	}
}

// getCurrentDistance gets the current distance for a celestial body
func getCurrentDistance(name string) float64 {
	distanceCacheMu.RLock()
	distance, ok := distanceCache[name]
	distanceCacheMu.RUnlock()
	
	if !ok {
		// If distance not in cache, trigger an update
		updateCelestialDistances()
		
		// Try again with updated cache
		distanceCacheMu.RLock()
		distance = distanceCache[name]
		distanceCacheMu.RUnlock()
	}
	
	return distance
}

func getCelestialBody(name string) (*CelestialBody, string) {
	// Ensure distances are up-to-date
	if time.Since(lastDistanceUpdate) > time.Hour {
		updateCelestialDistances()
	}
	
	// Check for spacecraft first
	if craft, ok := spacecraft[name]; ok {
		// Create a copy with updated distance
		craftCopy := *craft
		craftCopy.Distance = getCurrentDistance(name)
		return &craftCopy, name
	}

	// Check for moon (format: moon.planet)
	parts := strings.Split(name, ".")
	if len(parts) >= 2 {
		if planet, ok := solarSystem[parts[1]]; ok {
			if moon, ok := planet.Moons[parts[0]]; ok {
				moonCopy := *moon
				planetDistance := getCurrentDistance(parts[1])
				moonDistance := getCurrentDistance(parts[0] + "." + parts[1])
				moonCopy.Distance = moonDistance
				return &moonCopy, parts[0] + "." + parts[1]
			}
		}
	}

	// Check for planet
	if planet, ok := solarSystem[name]; ok {
		// Create a copy with updated distance
		planetCopy := *planet
		planetCopy.Distance = getCurrentDistance(name)
		return &planetCopy, name
	}

	return nil, ""
}

// Initialize the distance cache at startup
func init() {
	updateCelestialDistances()
}