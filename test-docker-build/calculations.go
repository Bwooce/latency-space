package main

import (
	"fmt"
	"io"
	"log"
	"math"
	"sort"
	"strings"
	"time"
)

var celestialObjects []CelestialObject

func CalculateLatency(distanceKm float64) time.Duration {
	// Use test mode with fixed low latency if enabled
	if isTestMode {
		return testModeCalculateLatency(distanceKm)
	}

	// Normal latency calculation
	seconds := distanceKm / SPEED_OF_LIGHT
	return time.Duration(seconds * float64(time.Second))
}

// Convert degrees to radians
func degToRad(deg float64) float64 {
	return deg * math.Pi / 180.0
}

// Normalize angle to [0, 2π) radians
func normalizeRadians(angle float64) float64 {
	angle = math.Mod(angle, 2*math.Pi)
	if angle < 0 {
		angle += 2 * math.Pi
	}
	return angle
}

// Convert time.Time to Julian Date
func timeToJulianDate(t time.Time) float64 {
	// Convert to UTC
	t = t.UTC()

	// Extract date components
	Y := float64(t.Year())
	M := float64(t.Month())
	D := float64(t.Day())

	// Extract time components and convert to day fraction
	h := float64(t.Hour()) / 24.0
	m := float64(t.Minute()) / 1440.0
	s := float64(t.Second()) / 86400.0

	// Calculate day fraction
	dayFraction := h + m + s

	// Adjust months so that January and February are 13 and 14 of the previous year
	if M <= 2 {
		Y--
		M += 12
	}

	// Calculate Julian day number
	A := math.Floor(Y / 100.0)
	B := 2 - A + math.Floor(A/4.0)

	jd := math.Floor(365.25*(Y+4716)) + math.Floor(30.6001*(M+1)) + D + B - 1524.5

	// Add day fraction
	jd += dayFraction

	return jd
}

// Calculate the TDB (Barycentric Dynamical Time) - TT (Terrestrial Time) difference
func calculateTDBMinusTT(jd float64) float64 {
	// Simplified algorithm for TDB-TT
	// This is a polynomial approximation
	t := (jd - J2000_EPOCH) / DAYS_PER_CENTURY
	g := degToRad(357.53 + 35999.050*t) // Mean anomaly of the Sun

	// TDB - TT in seconds
	return 0.001658*math.Sin(g) + 0.000014*math.Sin(2*g)
}

// Convert TT to TDB Julian date
func TTtoTDB(ttJD float64) float64 {
	return ttJD + calculateTDBMinusTT(ttJD)/SECONDS_PER_DAY
}

// Calculate centuries since J2000 for TDB time
func centuriesSinceJ2000TDB(t time.Time) float64 {
	// Convert time to Julian date
	jdUTC := timeToJulianDate(t)

	// Add approximate TT-UTC correction (crude but sufficient for this purpose)
	// More accurate would be to use a table of Delta-T values
	ttJD := jdUTC + 70.0/SECONDS_PER_DAY // Approximate TT-UTC in 2025

	// Convert TT to TDB
	tdbJD := TTtoTDB(ttJD)

	// Calculate centuries
	return (tdbJD - J2000_EPOCH) / DAYS_PER_CENTURY
}

// Solve Kepler's equation using a high-precision algorithm
func solveKeplerEquation(M float64, e float64) float64 {
	// Initial estimate using Danby's starter formula
	var E float64
	if e < 0.8 {
		E = M + e*math.Sin(M)*(1.0+e*math.Cos(M))
	} else {
		// For high eccentricity, use a better approximation
		E = M + e*math.Sin(M)/(1.0-math.Sin(M+e)+math.Sin(M))
	}

	// Refine using Newton-Raphson iterations with higher precision
	for iter := 0; iter < 15; iter++ {
		error := E - e*math.Sin(E) - M
		if math.Abs(error) < 1e-14 {
			break
		}

		delta := error / (1.0 - e*math.Cos(E))
		E -= delta

		// Add damping for highly eccentric orbits
		if iter > 8 && e > 0.95 {
			E = E*0.5 + (M+e*math.Sin(E))*0.5
		}
	}

	return normalizeRadians(E)
}

// calculateVSOP87Position calculates planetary positions using VSOP87 algorithm
// This is a simplified version with only the main periodic terms
func calculateVSOP87Position(obj CelestialObject, T float64) Vector3 {
	// Calculate the object's orbital elements at time T (centuries from J2000)
	a := obj.A + T*obj.DA
	e := obj.E + T*obj.DE
	i := degToRad(obj.I + T*obj.DI)
	L := degToRad(obj.L + T*obj.DL)
	wbar := degToRad(obj.LP + T*obj.DLP)
	node := degToRad(obj.N + T*obj.DN)

	// Add some important planetary perturbations for higher accuracy
	// These are simplified forms of the major perturbation terms

	// For Earth-specific perturbations (simplified VSOP87 terms)
	if obj.Name == "Earth" && obj.F > 0 && obj.B > 0 {
		// Major perturbation from Jupiter
		jupiterTerm := 0.00013 * math.Sin(degToRad(3.0*obj.F-8.0*obj.B+3.0)) // Example term

		// Major perturbation from Venus
		venusTerm := 0.00022 * math.Sin(degToRad(5.0*obj.C-2.0*obj.F-0.9)) // Example term

		// Apply perturbations
		L += degToRad(jupiterTerm + venusTerm)
	}

	// For Mars-specific perturbations (simplified VSOP87 terms)
	if obj.Name == "Mars" && obj.F > 0 && obj.B > 0 {
		// Major perturbation terms from Jupiter
		perturbation := 0.00043 * math.Sin(degToRad(2.0*obj.B-5.0*obj.F+52.31))
		perturbation += 0.00027 * math.Sin(degToRad(3.0*obj.B-5.0*obj.F+4.25))

		// Apply perturbations
		L += degToRad(perturbation)
		e += 0.000045 * math.Cos(degToRad(2.0*obj.B-obj.F+106.3))
	}

	// Calculate the mean anomaly
	// M = L - wbar
	M := normalizeRadians(L - wbar)

	// Calculate the argument of perihelion
	w := normalizeRadians(wbar - node)

	// Solve Kepler's equation for the eccentric anomaly
	E := solveKeplerEquation(M, e)

	// Calculate the true anomaly
	v := 2.0 * math.Atan2(
		math.Sqrt(1.0+e)*math.Sin(E/2.0),
		math.Sqrt(1.0-e)*math.Cos(E/2.0),
	)

	// Calculate the heliocentric distance (in AU)
	r := a * (1.0 - e*math.Cos(E))

	// Calculate the heliocentric position in the orbital plane
	xOrbit := r * math.Cos(v)
	yOrbit := r * math.Sin(v)
	zOrbit := 0.0

	// Transform to the ecliptic plane

	// First, rotate around z by w (argument of perihelion)
	xEclOrbit := xOrbit*math.Cos(w) - yOrbit*math.Sin(w)
	yEclOrbit := xOrbit*math.Sin(w) + yOrbit*math.Cos(w)
	zEclOrbit := zOrbit

	// Next, rotate around x by i (inclination)
	xEcl := xEclOrbit
	yEcl := yEclOrbit*math.Cos(i) - zEclOrbit*math.Sin(i)
	zEcl := yEclOrbit*math.Sin(i) + zEclOrbit*math.Cos(i)

	// Finally, rotate around z by node (longitude of ascending node)
	x := xEcl*math.Cos(node) - yEcl*math.Sin(node)
	y := xEcl*math.Sin(node) + yEcl*math.Cos(node)
	z := zEcl

	return Vector3{X: x, Y: y, Z: z}
}

// Calculate local position relative to parent body
func calculateLocalPosition(obj CelestialObject, T float64) Vector3 {
	// Calculate the object's orbital elements at time T
	a := obj.A + T*obj.DA
	e := obj.E + T*obj.DE
	i := degToRad(obj.I + T*obj.DI)
	L := degToRad(obj.L + T*obj.DL)

	var w, node, M float64

	// For objects with defined argument of perigee (moons, spacecraft)
	if obj.W != 0 {
		w = degToRad(obj.W + T*obj.DW)
		node = degToRad(obj.N + T*obj.DN)
		// Calculate mean anomaly
		M = normalizeRadians(L - (node + w))
	} else {
		// For objects with longitude of perihelion
		lp := degToRad(obj.LP + T*obj.DLP)
		node = degToRad(obj.N + T*obj.DN)
		// Calculate argument of perihelion and mean anomaly
		w = normalizeRadians(lp - node)
		M = normalizeRadians(L - lp)
	}

	// Solve Kepler's equation for eccentric anomaly
	E := solveKeplerEquation(M, e)

	// Calculate true anomaly
	v := 2.0 * math.Atan2(
		math.Sqrt(1.0+e)*math.Sin(E/2.0),
		math.Sqrt(1.0-e)*math.Cos(E/2.0),
	)

	// Calculate distance from parent
	r := a * (1.0 - e*math.Cos(E))

	// Position in orbital plane
	xOrb := r * math.Cos(v)
	yOrb := r * math.Sin(v)
	zOrb := 0.0

	// Transform to reference plane (ecliptic for planets, equatorial for moons)
	// First, rotate around z by argument of perihelion
	xRef := xOrb*math.Cos(w) - yOrb*math.Sin(w)
	yRef := xOrb*math.Sin(w) + yOrb*math.Cos(w)
	zRef := zOrb

	// Next, rotate around x by inclination
	xInc := xRef
	yInc := yRef*math.Cos(i) - zRef*math.Sin(i)
	zInc := yRef*math.Sin(i) + zRef*math.Cos(i)

	// Finally, rotate around z by longitude of ascending node
	x := xInc*math.Cos(node) - yInc*math.Sin(node)
	y := xInc*math.Sin(node) + yInc*math.Cos(node)
	z := zInc

	return Vector3{X: x, Y: y, Z: z}
}

// GetObjectPosition calculates the position of an object at a given time
func GetObjectPosition(obj CelestialObject, objects []CelestialObject, t time.Time) Vector3 {
	// For the Sun, return the origin
	if obj.Name == "Sun" {
		return Vector3{X: 0, Y: 0, Z: 0}
	}

	// Calculate centuries since J2000 using TDB
	T := centuriesSinceJ2000TDB(t)

	// For planets and dwarf planets (heliocentric orbits)
	if obj.Type == "planet" || obj.Type == "dwarf_planet" || obj.Type == "asteroid" {
		return calculateVSOP87Position(obj, T)
	}

	// For moons and spacecraft (parent-relative orbits)
	if obj.Type == "moon" || obj.Type == "spacecraft" {
		// Get parent body's position
		var parent CelestialObject
		parentFound := false

		for _, p := range objects {
			if p.Name == obj.ParentName {
				parent = p
				parentFound = true
				break
			}
		}

		if !parentFound {
			fmt.Printf("Error: Parent body %s not found for %s\n", obj.ParentName, obj.Name)
			return Vector3{X: 0, Y: 0, Z: 0}
		}

		// Get parent position
		parentPos := GetObjectPosition(parent, objects, t)

		// Calculate object's position relative to parent
		localPos := calculateLocalPosition(obj, T)

		// Convert to AU if the position is in km
		if obj.Type == "moon" || obj.Type == "spacecraft" {
			localPos.X /= AU
			localPos.Y /= AU
			localPos.Z /= AU
		}

		// Add parent position to get heliocentric position
		return Vector3{
			X: parentPos.X + localPos.X,
			Y: parentPos.Y + localPos.Y,
			Z: parentPos.Z + localPos.Z,
		}
	}

	// Default case
	return Vector3{X: 0, Y: 0, Z: 0}
}

// CalculateDistance calculates the distance between two objects in kilometers
func CalculateDistance(obj1, obj2 CelestialObject, objects []CelestialObject, t time.Time) float64 {
	// Get positions
	pos1 := GetObjectPosition(obj1, objects, t)
	pos2 := GetObjectPosition(obj2, objects, t)

	// Calculate distance vector
	distanceVector := pos2.Subtract(pos1)

	// Calculate distance in AU and convert to kilometers
	distanceAU := distanceVector.Magnitude()
	distanceKm := distanceAU * AU

	return distanceKm
}

// IsOccluded determines if target is occluded from the viewpoint of observer by any other object
func IsOccluded(observer, target CelestialObject, objects []CelestialObject, t time.Time) (bool, CelestialObject) {
	// Get positions
	observerPos := GetObjectPosition(observer, objects, t)
	targetPos := GetObjectPosition(target, objects, t)

	// Calculate the direction vector from observer to target
	dirVector := targetPos.Subtract(observerPos)
	distToTarget := dirVector.Magnitude() * AU // Distance in km

	// Normalize the direction vector
	dirNorm := dirVector.Normalize()

	// Check each object to see if it occludes the target
	for _, obj := range objects {
		// Skip the observer and target
		if obj.Name == observer.Name || obj.Name == target.Name {
			continue
		}

		// Get the position of the potential occluding body
		objPos := GetObjectPosition(obj, objects, t)

		// Vector from observer to the object
		objVector := objPos.Subtract(observerPos)
		distToObj := objVector.Magnitude() * AU // Distance in km

		// If the object is further away than the target, it can't occlude
		if distToObj >= distToTarget {
			continue
		}

		// Project the object vector onto the direction vector
		projection := objVector.DotProduct(dirNorm)

		// If the projection is negative, the object is behind the observer
		if projection <= 0 {
			continue
		}

		// Calculate the perpendicular distance from the object to the line of sight
		projectionVector := dirNorm.Scale(projection)
		perpendicularVector := objVector.Subtract(projectionVector)
		perpendicularDist := perpendicularVector.Magnitude() * AU // in km

		// Check if the perpendicular distance is less than the radius of the object
		// Add margins for specific object types
		occlusionRadius := obj.Radius
		if obj.Name == "Sun" {
			// For the Sun, add a larger margin for the corona
			occlusionRadius *= 1.05
		} else if obj.Type == "planet" || obj.Type == "dwarf_planet" {
			// For planets, add a small margin for atmosphere
			occlusionRadius *= 1.02
		}

		if perpendicularDist < occlusionRadius {
			return true, obj
		}
	}

	// No occlusion found
	return false, CelestialObject{}
}

// Helper function to find an object by name
func findObjectByName(objects []CelestialObject, name string) (CelestialObject, bool) {
	for _, obj := range objects {
		if obj.Name == name || strings.EqualFold(obj.Name, name) {
			return obj, true
		}
	}
	return CelestialObject{}, false
}

// ParseDate parses a date string in format YYYY-MM-DD
func ParseDate(dateStr string) (time.Time, error) {
	return time.Parse("2006-01-02", dateStr)
}

// FormatDistance formats a distance in kilometers with appropriate units
func FormatDistance(dist float64) string {
	if dist < 1000 {
		return fmt.Sprintf("%.1f km", dist)
	} else if dist < 1000000 {
		return fmt.Sprintf("%.1f thousand km", dist/1000)
	} else {
		return fmt.Sprintf("%.2f million km", dist/1000000)
	}
}

// Create a slice to store results
type DistanceEntry struct {
	Object     CelestialObject
	Distance   float64
	Occluded   bool
	OccludedBy CelestialObject
}

var lastDistanceUpdate time.Time
var distanceEntries []DistanceEntry // store the current distances

// Calculate distances from Earth to all objects
func calculateDistancesFromEarth(objects []CelestialObject, t time.Time) {

	if len(distanceEntries) > 0 && time.Since(lastDistanceUpdate) < time.Hour {
		log.Printf("No distances update required")
		return
	}

	// Find Earth
	earth, found := findObjectByName(objects, "Earth")
	if !found {
		fmt.Println("Error: Earth data not found")
		return
	}

	fmt.Printf("\nDistances from Earth on %s:\n\n", t.Format("2006-01-02"))
	distanceEntries = make([]DistanceEntry, 0, 20)
	// Calculate distances to all objects except Earth
	for _, obj := range objects {
		if obj.Name != "Earth" && obj.Name != "" {
			// Calculate distance
			distance := CalculateDistance(earth, obj, objects, t)

			// Check for occlusion
			occluded, occluderObj := IsOccluded(earth, obj, objects, t)

			distanceEntries = append(distanceEntries, DistanceEntry{
				Object:     obj,
				Distance:   distance,
				Occluded:   occluded,
				OccludedBy: occluderObj,
			})
		}
	}

	lastDistanceUpdate = time.Now()
}

func getCurrentDistance(bodyName string) float64 {
	log.Printf("Size of celestialObjects: %d", len(celestialObjects))
	calculateDistancesFromEarth(celestialObjects, time.Now()) // update if required
	log.Printf("Size of distanceObjects: %d", len(distanceEntries))
	for _, body := range distanceEntries {
		//log.Printf("body: %s, type: %s", body.Object.Name, body.Object.Type)
		if strings.EqualFold(body.Object.Name, bodyName) {
			return body.Distance
		}
	}
	log.Printf("getCurrentDistance: invalid body %s", bodyName)
	return 0
}

// Display objects of a specific type
func printObjectsByType(w io.Writer, entries []DistanceEntry, objectType string) {
	// Filter entries by type
	filteredEntries := make([]DistanceEntry, 0, 10)
	for _, entry := range entries {
		if entry.Object.Type == objectType {
			filteredEntries = append(filteredEntries, entry)
		}
	}

	if len(filteredEntries) == 0 {
		return
	}

	// Sort by distance
	sort.Slice(filteredEntries, func(i, j int) bool {
		return filteredEntries[i].Distance < filteredEntries[j].Distance
	})

	// Print header for this type
	typeName := objectType
	switch objectType {
	case "dwarf_planet":
		typeName = "Dwarf Planets"
	case "planet":
		typeName = "Planets"
	case "moon":
		typeName = "Moons"
	case "asteroid":
		typeName = "Asteroids"
	case "spacecraft":
		typeName = "Spacecraft"
	}

	fmt.Fprintf(w, "\n--- %s ---\n", typeName)
	fmt.Fprintf(w, "%-15s | %-10s | %-18s | %-15s | %-15s | %s\n",
		"Name", "Type", "Distance (km)", "Distance", "RTT", "Visibility")
	fmt.Fprintln(w, "--------------------------------------------------------------------------------------")

	for _, entry := range filteredEntries {
		visibility := "Visible"
		if entry.Occluded {
			visibility = fmt.Sprintf("Occluded by %s", entry.OccludedBy.Name)
		}

		fmt.Fprintf(w, "%-15s | %-10s | %18.0f | %-15s | %-15s | %s\n",
			entry.Object.Name,
			entry.Object.Type,
			entry.Distance,
			FormatDistance(entry.Distance),
			CalculateLatency(entry.Distance*2).Round(time.Second).String(),
			visibility)
	}
}