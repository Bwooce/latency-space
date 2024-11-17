package main

import (
	"strings"
)

func getCelestialBody(name string) (*CelestialBody, string) {
	// Check for spacecraft first
	if craft, ok := spacecraft[name]; ok {
		return craft, name
	}

	// Check for moon (format: moon.planet)
	parts := strings.Split(name, ".")
	if len(parts) >= 2 {
		if planet, ok := solarSystem[parts[1]]; ok {
			if moon, ok := planet.Moons[parts[0]]; ok {
				moonCopy := *moon
				moonCopy.Distance += planet.Distance
				return &moonCopy, parts[0] + "." + parts[1]
			}
		}
	}

	// Check for planet
	if planet, ok := solarSystem[name]; ok {
		return planet, name
	}

	return nil, ""
}
