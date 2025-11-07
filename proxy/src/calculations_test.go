//go:build test
// +build test

package main

import (
	"math"
	"testing"
	"time"
)

// TestDistinctSpacecraftDistances verifies that calculateDistancesFromEarth
// computes different distances for spacecraft in different locations relative to Earth.
func TestDistinctSpacecraftDistances(t *testing.T) {
	// 1. Define Test Data (Simplified Orbital Parameters)
	// Using simplified heliocentric coordinates (AU) for a specific time.
	// These are NOT accurate orbital elements, just positions for testing.
	// Sun - Required as parent for other objects
	sun := CelestialObject{
		Name:   "Sun",
		Type:   "star",
		Radius: 695700, // km
		Mass:   1.989e30,
		// Sun is at the origin, no orbital elements needed
	}
	// Earth - Reference point for distance calculations
	earth := CelestialObject{
		Name:       "Earth",
		Type:       "planet",
		ParentName: "Sun",
		// Simplified position: 1 AU along X-axis
		A: 1.0, E: 0, I: 0, L: 0, LP: 0, N: 0, // Simplified elements for position calc
	}
	// Voyager 1 - Far out in the solar system
	voyager1 := CelestialObject{
		Name: "Voyager 1", Type: "spacecraft", ParentName: "Sun", // Heliocentric
		// Simplified position: ~150 AU along X-axis (very far)
		A: 150.0, E: 0, I: 0, L: 0, LP: 0, N: 0, // Simplified elements
		Radius: 1, // Placeholder
	}
	// JWST - Near Earth's L2 point (roughly 0.01 AU further from Sun than Earth)
	jwst := CelestialObject{
		Name: "JWST", Type: "spacecraft", ParentName: "Sun", // Heliocentric (simplified model for test)
		// Simplified position: ~1.01 AU along X-axis
		A: 1.01, E: 0, I: 0, L: 0, LP: 0, N: 0, // Simplified elements
		Radius: 1, // Placeholder
	}

	testObjects := []CelestialObject{sun, earth, voyager1, jwst}

	// 2. Define a fixed time
	testTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// 3. Call calculateDistancesFromEarth with test data
	// This function updates the global distanceEntries slice
	// Reset global state before test
	distanceEntries = []DistanceEntry{}
	lastDistanceUpdate = time.Time{}
	// Set the global celestialObjects for the test context IF NEEDED by dependencies
	// Since calculateDistancesFromEarth takes objects as arg, we don't strictly need this
	// But GetObjectPosition relies on the global slice if ParentName lookups occur
	originalCelestialObjects := celestialObjects // backup
	celestialObjects = testObjects               // set global for GetObjectPosition
	defer func() { celestialObjects = originalCelestialObjects }() // restore

	calculateDistancesFromEarth(testObjects, testTime)

	// 4. Read Results using RLock
	DistanceCacheMutex.RLock()
	var voyagerDist, jwstDist float64 = -1.0, -1.0 // Use -1 as sentinel for "not found"

	t.Logf("Reading distanceEntries (size: %d)", len(distanceEntries)) // Log cache size
	for _, entry := range distanceEntries {
		t.Logf("Found entry: %s, Dist: %f", entry.Object.Name, entry.Distance) // Log each entry
		if entry.Object.Name == "Voyager 1" {
			voyagerDist = entry.Distance
		} else if entry.Object.Name == "JWST" {
			jwstDist = entry.Distance
		}
	}
	DistanceCacheMutex.RUnlock()

	// 5. Assert distances were found
	if voyagerDist == -1.0 {
		t.Fatalf("Distance for Voyager 1 not found in cache")
	}
	if jwstDist == -1.0 {
		t.Fatalf("Distance for JWST not found in cache")
	}
	t.Logf("Voyager 1 Distance: %f km, JWST Distance: %f km", voyagerDist, jwstDist)

	// 6. Assert distances are distinct
	// Expect JWST to be much closer than Voyager 1
	// JWST should be roughly 0.01 AU from Earth (~1.5 million km)
	// Voyager 1 should be roughly 149 AU from Earth (~22 billion km)
	// Check they are significantly different (e.g., > 10 million km difference)
	if math.Abs(voyagerDist-jwstDist) < 10e6 { // 10 million km threshold
		t.Errorf("Expected distinct distances for Voyager 1 and JWST, but got Voyager: %f km, JWST: %f km (difference < 10M km)", voyagerDist, jwstDist)
	}

	// Optional: Add approximate checks for expected ranges
	expectedJwstDist := 0.01 * AU // ~1.5 million km
	if math.Abs(jwstDist-expectedJwstDist) > 0.5e6 { // Allow 500k km tolerance
		t.Errorf("JWST distance (%f km) is further than expected (%f km +/- 500k km) from Earth based on simplified model", jwstDist, expectedJwstDist)
	}

	expectedVoyagerDist := 149.0 * AU // ~22 billion km
	if math.Abs(voyagerDist-expectedVoyagerDist) > 1e9 { // Allow 1 billion km tolerance (large distance)
		t.Errorf("Voyager 1 distance (%f km) significantly different than expected (%f km +/- 1B km) from Earth based on simplified model", voyagerDist, expectedVoyagerDist)
	}
}
