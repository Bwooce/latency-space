package main

import (
	"main"
)

// Re-export the shared celestial types
type CelestialObject = celestial.CelestialObject
type Vector3 = celestial.Vector3

// Re-export shared constants
const (
	SPEED_OF_LIGHT   = celestial.SPEED_OF_LIGHT
	AU               = celestial.AU
	EARTH_RADIUS     = celestial.EARTH_RADIUS
	SUN_RADIUS       = celestial.SUN_RADIUS
	SECONDS_PER_DAY  = celestial.SECONDS_PER_DAY
	DAYS_PER_CENTURY = celestial.DAYS_PER_CENTURY
	J2000_EPOCH      = celestial.J2000_EPOCH
)

