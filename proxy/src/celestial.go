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
	Offset        float64 // initial offset in radians
}{
	"mercury": {57.9, 0.2056, 88.0, 0.0},
	"venus":   {108.2, 0.0068, 224.7, 1.2},
	"earth":   {149.6, 0.0167, 365.2, 2.1},
	"mars":    {227.9, 0.0934, 687.0, 0.5},
	"jupiter": {778.6, 0.0484, 4331.0, 3.1},
	"saturn":  {1434.0, 0.0542, 10747.0, 4.2},
	"uranus":  {2871.0, 0.0472, 30589.0, 5.3},
	"neptune": {4495.0, 0.0086, 59800.0, 0.8},
	"pluto":   {5906.0, 0.2488, 90560.0, 1.7},
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

	// Calculate planet positions
	for name, orbit := range planetaryOrbits {
		// Calculate mean anomaly
		meanAnomaly := (2.0 * math.Pi * daysSinceEpoch / orbit.OrbitalPeriod) + orbit.Offset
		meanAnomaly = math.Mod(meanAnomaly, 2.0*math.Pi)
		
		// Solve Kepler's equation (approximation)
		eccentricAnomaly := meanAnomaly
		for i := 0; i < 5; i++ { // Usually converges in a few iterations
			eccentricAnomaly = meanAnomaly + orbit.Eccentricity*math.Sin(eccentricAnomaly)
		}
		
		// Calculate distance
		distance := orbit.SemiMajorAxis * (1.0 - orbit.Eccentricity*math.Cos(eccentricAnomaly))
		
		// For Earth, distance is 0 (reference point)
		if name == "earth" {
			distance = 0
		} else {
			// Calculate Earth's position
			earthMA := (2.0 * math.Pi * daysSinceEpoch / 365.2) + planetaryOrbits["earth"].Offset
			earthMA = math.Mod(earthMA, 2.0*math.Pi)
			
			earthEA := earthMA
			for i := 0; i < 5; i++ {
				earthEA = earthMA + planetaryOrbits["earth"].Eccentricity*math.Sin(earthEA)
			}
			
			earthDist := planetaryOrbits["earth"].SemiMajorAxis * (1.0 - planetaryOrbits["earth"].Eccentricity*math.Cos(earthEA))
			
			// Simplified distance calculation - ignores orbital inclination
			// and just uses the law of cosines for a rough approximation
			angleOffset := meanAnomaly - earthMA
			distance = math.Sqrt(earthDist*earthDist + distance*distance - 2*earthDist*distance*math.Cos(angleOffset))
		}
		
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
