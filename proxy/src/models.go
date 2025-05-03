package main

import (
	"math"
)

// Astronomical constants
const (
	SPEED_OF_LIGHT   = 299792.458  // km/s
	AU               = 149597870.7 // Astronomical unit in kilometers
	EARTH_RADIUS     = 6378.137    // Earth radius in kilometers
	SUN_RADIUS       = 695700.0    // Sun radius in kilometers
	SECONDS_PER_DAY  = 86400.0     // Seconds in a day
	DAYS_PER_CENTURY = 36525.0     // Days in a Julian century (365.25 * 100)
	J2000_EPOCH      = 2451545.0   // J2000 epoch in Julian days (January 1, 2000, 12:00 TT)
)

// CelestialObject defines the structure for storing data about any object in the solar system.
type CelestialObject struct {
	Name       string
	Type       string  // e.g., "planet", "dwarf_planet", "moon", "spacecraft", "asteroid", "star"
	ParentName string  // Name of parent body (empty for Sun, planet name for moons)
	Radius     float64 // Mean radius in kilometers

	// Orbital elements relative to the J2000 epoch.
	// - Planets/Dwarf Planets/Asteroids: Heliocentric elements (AU, degrees).
	// - Moons: Parent-centric elements (km, degrees).
	// - Spacecraft: Mission-specific or fixed elements (AU or km, degrees).
	A  float64 // Semi-major axis (AU for heliocentric, km otherwise)
	E  float64 // Eccentricity
	I  float64 // Inclination (degrees, relative to ecliptic or parent equator)
	L  float64 // Mean longitude (degrees)
	LP float64 // Longitude of perihelion (degrees) - used for heliocentric
	N  float64 // Longitude of ascending node (degrees)

	// Rates of change for orbital elements per Julian century.
	dA  float64 // Rate of change for semi-major axis (AU/century or km/century)
	dE  float64 // Rate of change for eccentricity (per century)
	dI  float64 // Rate of change for inclination (degrees/century)
	dL  float64 // Rate of change for mean longitude (degrees/century)
	dLP float64 // Rate of change for longitude of perihelion (degrees/century)
	dN  float64 // Rate of change for longitude of ascending node (degrees/century)

	// Additional parameters primarily for moons and spacecraft.
	W      float64 // Argument of perigee/periapsis (degrees) - used for parent-centric
	dW     float64 // Rate of change for argument of perigee (degrees/century)
	Period float64 // Orbital period (days) - can be calculated, but useful for reference

	// Parameters used in perturbation calculations (simplified VSOP87).
	b float64 // Coefficient (e.g., related to another body's period)
	c float64 // Coefficient (e.g., related to eccentricity)
	s float64 // Coefficient (e.g., sine term)
	f float64 // Coefficient (e.g., mean motion)

	// Physical properties.
	Mass float64 // Mass in kilograms

	// Spacecraft-specific parameters.
	TransmitterActive bool    // Is the spacecraft currently transmitting?
	LaunchDate        string  // Launch date (YYYY-MM-DD)
	FrequencyMHz      float64 // Primary downlink frequency in MHz
	MissionStatus     string  // e.g., "active", "extended", "completed", "failed"
}

// Vector3 represents a standard 3D vector with X, Y, Z components.
type Vector3 struct {
	X, Y, Z float64
}

// Add performs vector addition.
func (v Vector3) Add(other Vector3) Vector3 {
	return Vector3{
		X: v.X + other.X,
		Y: v.Y + other.Y,
		Z: v.Z + other.Z,
	}
}

// Subtract performs vector subtraction (v - other).
func (v Vector3) Subtract(other Vector3) Vector3 {
	return Vector3{
		X: v.X - other.X,
		Y: v.Y - other.Y,
		Z: v.Z - other.Z,
	}
}

// Scale multiplies the vector by a scalar factor.
func (v Vector3) Scale(factor float64) Vector3 {
	return Vector3{
		X: v.X * factor,
		Y: v.Y * factor,
		Z: v.Z * factor,
	}
}

// Magnitude calculates the Euclidean length (magnitude) of the vector.
func (v Vector3) Magnitude() float64 {
	return math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z)
}

// DotProduct calculates the dot product of two vectors.
func (v Vector3) DotProduct(other Vector3) float64 {
	return v.X*other.X + v.Y*other.Y + v.Z*other.Z
}

// CrossProduct calculates the cross product of two vectors (v x other).
func (v Vector3) CrossProduct(other Vector3) Vector3 {
	return Vector3{
		X: v.Y*other.Z - v.Z*other.Y,
		Y: v.Z*other.X - v.X*other.Z,
		Z: v.X*other.Y - v.Y*other.X,
	}
}

// Normalize returns a unit vector pointing in the same direction as the original vector.
// Returns a zero vector if the magnitude is close to zero.
func (v Vector3) Normalize() Vector3 {
	mag := v.Magnitude()
	// Avoid division by zero or very small numbers
	if mag < 1e-10 {
		return Vector3{0, 0, 0} // Return zero vector if magnitude is negligible
	}
	return Vector3{
		X: v.X / mag,
		Y: v.Y / mag,
		Z: v.Z / mag,
	}
}
