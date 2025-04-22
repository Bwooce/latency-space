package main

import (
	"math"
)

// Astronomical constants
const (
	SPEED_OF_LIGHT     = 299792.458 // km/s
	AU                 = 149597870.7 // Astronomical unit in kilometers
	EARTH_RADIUS       = 6378.137    // Earth radius in kilometers
	SUN_RADIUS         = 695700.0    // Sun radius in kilometers
	SECONDS_PER_DAY    = 86400.0     // Seconds in a day
	DAYS_PER_CENTURY   = 36525.0     // Days in a Julian century (365.25 * 100)
	J2000_EPOCH        = 2451545.0   // J2000 epoch in Julian days (January 1, 2000, 12:00 TT)
)

// CelestialObject represents any object in the solar system (planet, moon, spacecraft, etc.)
type CelestialObject struct {
	Name       string
	Type       string  // "planet", "dwarf_planet", "moon", "spacecraft", etc.
	ParentName string  // Name of parent body (empty for Sun, planet name for moons)
	Radius     float64 // Mean radius in km
	
	// Orbital elements for J2000 epoch
	// For planets, dwarf planets: heliocentric elements (in AU and degrees)
	// For moons: parent-centric elements (semi-major axis in km, angles in degrees)
	// For spacecraft: mission-specific elements or fixed position
	A    float64 // Semi-major axis (AU for heliocentric, km for moon/spacecraft orbits)
	E    float64 // Eccentricity
	I    float64 // Inclination (degrees)
	L    float64 // Mean longitude (degrees)
	LP   float64 // Longitude of perihelion (degrees)
	N    float64 // Longitude of ascending node (degrees)
	
	// Century rates for orbital elements
	dA    float64 // Rate of semi-major axis change per century
	dE    float64 // Rate of eccentricity change per century
	dI    float64 // Rate of inclination change per century
	dL    float64 // Rate of mean longitude change per century
	dLP   float64 // Rate of longitude of perihelion change per century
	dN    float64 // Rate of longitude of ascending node change per century
	
	// Additional parameters for moons and spacecraft
	W      float64 // Argument of perigee (degrees)
	dW     float64 // Rate of argument of perigee change per century
	Period float64 // Orbital period (days)
	
	// For perturbation calculations
	b float64 // Orbital period (days) or other coefficient
	c float64 // Eccentricity for perturbation terms
	s float64 // Sin term coefficient
	f float64 // Mean motion (degrees/day)
	
	// Physical properties
	Mass   float64 // Mass in kg
	
	// Spacecraft specific parameters
	TransmitterActive bool    // Whether the spacecraft is currently transmitting
	LaunchDate        string  // Date of launch
	FrequencyMHz      float64 // Transmission frequency in MHz
	MissionStatus     string  // "active", "completed", "failed", etc.
}

// Vector3 represents a 3D vector
type Vector3 struct {
	X, Y, Z float64
}

// Add returns the sum of two vectors
func (v Vector3) Add(other Vector3) Vector3 {
	return Vector3{
		X: v.X + other.X,
		Y: v.Y + other.Y,
		Z: v.Z + other.Z,
	}
}

// Subtract returns the difference of two vectors
func (v Vector3) Subtract(other Vector3) Vector3 {
	return Vector3{
		X: v.X - other.X,
		Y: v.Y - other.Y,
		Z: v.Z - other.Z,
	}
}

// Scale returns the vector multiplied by a scalar
func (v Vector3) Scale(factor float64) Vector3 {
	return Vector3{
		X: v.X * factor,
		Y: v.Y * factor,
		Z: v.Z * factor,
	}
}

// Magnitude returns the magnitude (length) of the vector
func (v Vector3) Magnitude() float64 {
	return math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z)
}

// DotProduct returns the dot product of two vectors
func (v Vector3) DotProduct(other Vector3) float64 {
	return v.X*other.X + v.Y*other.Y + v.Z*other.Z
}

// CrossProduct returns the cross product of two vectors
func (v Vector3) CrossProduct(other Vector3) Vector3 {
	return Vector3{
		X: v.Y*other.Z - v.Z*other.Y,
		Y: v.Z*other.X - v.X*other.Z,
		Z: v.X*other.Y - v.Y*other.X,
	}
}

// Normalize returns the normalized vector (unit length)
func (v Vector3) Normalize() Vector3 {
	mag := v.Magnitude()
	if mag < 1e-10 {
		return Vector3{0, 0, 0}
	}
	return Vector3{
		X: v.X / mag,
		Y: v.Y / mag,
		Z: v.Z / mag,
	}
}
