package main

import (
	"net/url"
	"testing"
	"strings" // Import strings for case-insensitive comparison later
)

// Mock celestial objects for testing parseHostForCelestialBody
var testCelestialObjects = []CelestialObject{
	{Name: "Earth", Type: "planet"},
	{Name: "Mars", Type: "planet"},
	{Name: "Jupiter", Type: "planet"},
	{Name: "Moon", Type: "moon", ParentName: "Earth"},
	{Name: "Phobos", Type: "moon", ParentName: "Mars"},
	{Name: "Deimos", Type: "moon", ParentName: "Mars"},
	{Name: "Europa", Type: "moon", ParentName: "Jupiter"},
}

func TestParseHostForCelestialBody(t *testing.T) {
	// Override the global celestialObjects with our test data for this test
	originalCelestialObjects := celestialObjects
	celestialObjects = testCelestialObjects
	defer func() { celestialObjects = originalCelestialObjects }() // Restore original data after test

	dummyURL, _ := url.Parse("http://example.com") // Dummy URL, path doesn't matter

	testCases := []struct {
		name         string
		host         string
		expectedURL  string
		expectedBody CelestialObject // Compare the actual object
		expectedBodyName string // Also compare the name for clarity in errors
	}{
		{
			name:         "Moon format with target",
			host:         "www.example.com.phobos.mars.latency.space",
			expectedURL:  "www.example.com",
			expectedBody: testCelestialObjects[4], // Phobos
			expectedBodyName: "Phobos",
		},
		{
			name:         "Moon format without target",
			host:         "phobos.mars.latency.space",
			expectedURL:  "",
			expectedBody: testCelestialObjects[4], // Phobos
			expectedBodyName: "Phobos",
		},
		{
			name:         "Planet format with target",
			host:         "www.example.com.mars.latency.space",
			expectedURL:  "www.example.com",
			expectedBody: testCelestialObjects[1], // Mars
			expectedBodyName: "Mars",
		},
		{
			name:         "Planet format without target",
			host:         "mars.latency.space",
			expectedURL:  "",
			expectedBody: testCelestialObjects[1], // Mars
			expectedBodyName: "Mars",
		},
		{
			name:         "Invalid moon parent", // Phobos orbits Mars, not Jupiter
			host:         "www.example.com.phobos.jupiter.latency.space",
			expectedURL:  "", // Should fail moon check, potentially fallback or fail entirely
			expectedBody: CelestialObject{}, // Expect empty object
			expectedBodyName: "", // Expect empty name
		},
		{
            name:         "Moon format with wrong planet type", // Earth is a planet, but Moon doesn't orbit Mars
            host:         "www.example.com.moon.mars.latency.space",
            expectedURL:  "",
            expectedBody: CelestialObject{},
            expectedBodyName: "",
        },
		{
			name:         "Non-existent body",
			host:         "www.example.com.unknown.latency.space",
			expectedURL:  "",
			expectedBody: CelestialObject{},
			expectedBodyName: "",
		},
		{
			name:         "Invalid format - just domain",
			host:         "latency.space",
			expectedURL:  "",
			expectedBody: CelestialObject{},
			expectedBodyName: "",
		},
		{
			name:         "Invalid format - wrong TLD",
			host:         "mars.latency.com",
			expectedURL:  "",
			expectedBody: CelestialObject{},
			expectedBodyName: "",
		},
		{
            name:         "Invalid format - unrelated domain",
            host:         "example.com",
            expectedURL:  "",
            expectedBody: CelestialObject{},
            expectedBodyName: "",
        },
		{
			name:         "Case insensitivity - Moon format with target",
			host:         "WWW.EXAMPLE.COM.PHOBOS.MARS.LATENCY.SPACE",
			// Note: Go's URL/host parsing tends to lowercase the host,
			// but our function uses the host string directly. Let's test if it handles it.
			// The target domain extraction *should* preserve case.
			// The body name lookup *should* be case-insensitive (handled by findObjectByName).
			expectedURL:  "WWW.EXAMPLE.COM",
			expectedBody: testCelestialObjects[4], // Phobos
			expectedBodyName: "Phobos",
		},
		{
            name:         "Case insensitivity - Planet format without target",
            host:         "MARS.latency.space",
            expectedURL:  "",
            expectedBody: testCelestialObjects[1], // Mars
            expectedBodyName: "Mars",
        },
		{
			name:         "Host with port",
			host:         "mars.latency.space:8080",
			expectedURL:  "",
			expectedBody: testCelestialObjects[1], // Mars
			expectedBodyName: "Mars",
		},
		{
            name:         "Moon format with target and port",
            host:         "www.example.com.phobos.mars.latency.space:443",
            expectedURL:  "www.example.com",
            expectedBody: testCelestialObjects[4], // Phobos
            expectedBodyName: "Phobos",
        },
	}

	// Instantiate the server struct to call the method
	// We don't need metrics or security for this test
	s := &Server{}

	for _, tc := range testCases {
		// Use t.Run to create sub-tests for each case
		t.Run(tc.name, func(t *testing.T) {
			actualURL, actualBody, actualBodyName := s.parseHostForCelestialBody(tc.host, dummyURL)

			// Check target URL
			if actualURL != tc.expectedURL {
				t.Errorf("host '%s': expected target URL '%s', got '%s'", tc.host, tc.expectedURL, actualURL)
			}

			// Check body name (case-insensitive comparison for robustness, although findObjectByName should handle it)
			if strings.ToLower(actualBodyName) != strings.ToLower(tc.expectedBodyName) {
				t.Errorf("host '%s': expected body name '%s', got '%s'", tc.host, tc.expectedBodyName, actualBodyName)
			}

			// Check the returned CelestialObject itself (by comparing names as a proxy, assuming names are unique in test data)
			if actualBody.Name != tc.expectedBody.Name {
                 t.Errorf("host '%s': expected body object name '%s', got '%s'", tc.host, tc.expectedBody.Name, actualBody.Name)
            }
		})
	}
}

// Add a separate test for findObjectByName for robustness
func TestFindObjectByName(t *testing.T) {
    // Use the same test data
    originalCelestialObjects := celestialObjects
	celestialObjects = testCelestialObjects
	defer func() { celestialObjects = originalCelestialObjects }()

    testCases := []struct {
        name          string
        searchName    string
        expectedFound bool
        expectedName  string // Expected name if found
    }{
        {"Find existing planet", "Mars", true, "Mars"},
        {"Find existing moon", "Phobos", true, "Phobos"},
        {"Find existing case-insensitive", "phobos", true, "Phobos"},
        {"Find existing case-insensitive upper", "MARS", true, "Mars"},
        {"Find non-existent", "Unknown", false, ""},
        {"Find empty string", "", false, ""},
        {"Find planet in nil slice", "Earth", false, ""}, // Test edge case
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            var objectsToSearch []CelestialObject
            if tc.name == "Find planet in nil slice" {
                objectsToSearch = nil
            } else {
                objectsToSearch = celestialObjects
            }

            foundBody, found := findObjectByName(objectsToSearch, tc.searchName)

            if found != tc.expectedFound {
                t.Errorf("searchName '%s': expected found status %v, got %v", tc.searchName, tc.expectedFound, found)
            }

            if found && foundBody.Name != tc.expectedName {
                t.Errorf("searchName '%s': expected object name '%s', got '%s'", tc.searchName, tc.expectedName, foundBody.Name)
            }

            if !found && foundBody != (CelestialObject{}) {
                 t.Errorf("searchName '%s': expected empty object when not found, got %+v", tc.searchName, foundBody)
            }
        })
    }
}
