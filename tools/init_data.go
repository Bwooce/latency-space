// init_data.go
package main

// This file initializes the celestial object data when imported

import (
	"fmt"
	"math"
	"os"
	"strings"
)

// Initialize celestial objects data at startup
func init() {
	// Add the Sun at the origin
	celestialObjects = append(celestialObjects, CelestialObject{
		Name:           "Sun",
		Type:           "Star",
		SemiMajorAxis:  0,
		Eccentricity:   0,
		Inclination:    0,
		AscendingNode:  0,
		Perihelion:     0,
		MeanAnomaly:    0,
		TimestampEpoch: 0,
		Position:       Vector3{0, 0, 0},
		ParentBody:     "",
	})

	// Add planets
	celestialObjects = append(celestialObjects, CelestialObject{
		Name:           "Mercury",
		Type:           "Planet",
		SemiMajorAxis:  57.91e6,
		Eccentricity:   0.2056,
		Inclination:    7.005,
		AscendingNode:  48.331,
		Perihelion:     77.456,
		MeanAnomaly:    174.795,
		TimestampEpoch: 0,
		Position:       Vector3{0, 0, 0}, // Will be calculated
		ParentBody:     "Sun",
	})

	celestialObjects = append(celestialObjects, CelestialObject{
		Name:           "Venus",
		Type:           "Planet",
		SemiMajorAxis:  108.21e6,
		Eccentricity:   0.0068,
		Inclination:    3.39458,
		AscendingNode:  76.68,
		Perihelion:     131.533,
		MeanAnomaly:    50.115,
		TimestampEpoch: 0,
		Position:       Vector3{0, 0, 0}, // Will be calculated
		ParentBody:     "Sun",
	})

	celestialObjects = append(celestialObjects, CelestialObject{
		Name:           "Earth",
		Type:           "Planet",
		SemiMajorAxis:  149.6e6,
		Eccentricity:   0.0167,
		Inclination:    0.00005,
		AscendingNode:  -11.26064,
		Perihelion:     102.94719,
		MeanAnomaly:    357.51716,
		TimestampEpoch: 0,
		Position:       Vector3{0, 0, 0}, // Will be calculated
		ParentBody:     "Sun",
	})

	celestialObjects = append(celestialObjects, CelestialObject{
		Name:           "Mars",
		Type:           "Planet",
		SemiMajorAxis:  227.92e6,
		Eccentricity:   0.0935,
		Inclination:    1.85,
		AscendingNode:  49.558,
		Perihelion:     336.04,
		MeanAnomaly:    19.412,
		TimestampEpoch: 0,
		Position:       Vector3{0, 0, 0}, // Will be calculated
		ParentBody:     "Sun",
	})

	celestialObjects = append(celestialObjects, CelestialObject{
		Name:           "Jupiter",
		Type:           "Planet",
		SemiMajorAxis:  778.57e6,
		Eccentricity:   0.0489,
		Inclination:    1.303,
		AscendingNode:  100.464,
		Perihelion:     14.75,
		MeanAnomaly:    340.87,
		TimestampEpoch: 0,
		Position:       Vector3{0, 0, 0}, // Will be calculated
		ParentBody:     "Sun",
	})

	celestialObjects = append(celestialObjects, CelestialObject{
		Name:           "Saturn",
		Type:           "Planet",
		SemiMajorAxis:  1433.53e6,
		Eccentricity:   0.0565,
		Inclination:    2.485,
		AscendingNode:  113.665,
		Perihelion:     92.43194,
		MeanAnomaly:    14.72,
		TimestampEpoch: 0,
		Position:       Vector3{0, 0, 0}, // Will be calculated
		ParentBody:     "Sun",
	})

	celestialObjects = append(celestialObjects, CelestialObject{
		Name:           "Uranus",
		Type:           "Planet",
		SemiMajorAxis:  2872.46e6,
		Eccentricity:   0.0457,
		Inclination:    0.773,
		AscendingNode:  74.006,
		Perihelion:     170.964,
		MeanAnomaly:    244.197,
		TimestampEpoch: 0,
		Position:       Vector3{0, 0, 0}, // Will be calculated
		ParentBody:     "Sun",
	})

	celestialObjects = append(celestialObjects, CelestialObject{
		Name:           "Neptune",
		Type:           "Planet",
		SemiMajorAxis:  4495.06e6,
		Eccentricity:   0.0113,
		Inclination:    1.767975,
		AscendingNode:  131.784,
		Perihelion:     44.971,
		MeanAnomaly:    84.457,
		TimestampEpoch: 0,
		Position:       Vector3{0, 0, 0}, // Will be calculated
		ParentBody:     "Sun",
	})

	// Add Dwarf Planets
	celestialObjects = append(celestialObjects, CelestialObject{
		Name:           "Pluto",
		Type:           "DwarfPlanet",
		SemiMajorAxis:  5906.38e6,
		Eccentricity:   0.2488,
		Inclination:    17.14175,
		AscendingNode:  110.299,
		Perihelion:     224.06,
		MeanAnomaly:    14.86,
		TimestampEpoch: 0,
		Position:       Vector3{0, 0, 0}, // Will be calculated
		ParentBody:     "Sun",
	})

	// Add Moon to Earth
	celestialObjects = append(celestialObjects, CelestialObject{
		Name:           "Moon",
		Type:           "Moon",
		SemiMajorAxis:  384.4e3,
		Eccentricity:   0.0549,
		Inclination:    5.145,
		AscendingNode:  125.08,
		Perihelion:     318.15,
		MeanAnomaly:    115.3654,
		TimestampEpoch: 0,
		Position:       Vector3{0, 0, 0}, // Will be calculated
		ParentBody:     "Earth",
	})

	// Add Spacecraft
	celestialObjects = append(celestialObjects, CelestialObject{
		Name:           "Voyager 1",
		Type:           "Spacecraft",
		SemiMajorAxis:  10000e6, // Approximate
		Eccentricity:   0.1,
		Inclination:    35.1,
		AscendingNode:  15.0,
		Perihelion:     0.0,
		MeanAnomaly:    0.0,
		TimestampEpoch: 0,
		Position:       Vector3{0, 0, 0}, // Will be calculated
		ParentBody:     "Sun",
	})

	celestialObjects = append(celestialObjects, CelestialObject{
		Name:           "Voyager 2",
		Type:           "Spacecraft",
		SemiMajorAxis:  8000e6, // Approximate
		Eccentricity:   0.1,
		Inclination:    35.1,
		AscendingNode:  15.0,
		Perihelion:     0.0,
		MeanAnomaly:    0.0,
		TimestampEpoch: 0,
		Position:       Vector3{0, 0, 0}, // Will be calculated
		ParentBody:     "Sun",
	})

	// Print initialization confirmation
	fmt.Printf("Initialized %d celestial objects\n", len(celestialObjects))

	// Validate that all objects are set up correctly
	for _, obj := range celestialObjects {
		if obj.Type == "" {
			fmt.Printf("Warning: Object %s has no type\n", obj.Name)
		}
		if obj.Name == "" {
			fmt.Println("Warning: Found object with no name")
		}
	}
}

// Helper function to find and check if a flag is present in os.Args
func isCommandLineFlagPresent(flagName string) bool {
	flagWithDash := "-" + flagName
	flagWithDoubleDash := "--" + flagName
	
	for i, arg := range os.Args {
		// Check for both -ssl and --ssl formats
		if arg == flagWithDash || arg == flagWithDoubleDash {
			return true
		}
		
		// Check for -ssl=true format
		if strings.HasPrefix(arg, flagWithDash+"=") || strings.HasPrefix(arg, flagWithDoubleDash+"=") {
			parts := strings.Split(arg, "=")
			if len(parts) == 2 && strings.ToLower(parts[1]) == "true" {
				return true
			}
		}
		
		// Check for the next arg after -ssl 
		if (arg == flagWithDash || arg == flagWithDoubleDash) && i < len(os.Args)-1 {
			nextArg := os.Args[i+1]
			if strings.ToLower(nextArg) == "true" {
				return true
			}
		}
	}
	
	return false
}