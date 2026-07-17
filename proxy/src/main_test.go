package main

import (
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings" // Import strings for case-insensitive comparison later
	"testing"
	"time"

	"github.com/latency-space/shared/celestial"
)

// testCelestialObjects provides a simplified list of objects for testing parseHostForCelestialBody.
var testCelestialObjects = []CelestialObject{
	{Name: "Earth", Type: "planet"},
	{Name: "Mars", Type: "planet"},
	{Name: "Jupiter", Type: "planet"},
	{Name: "Moon", Type: "moon", ParentName: "Earth"},
	{Name: "Phobos", Type: "moon", ParentName: "Mars"},
	{Name: "Deimos", Type: "moon", ParentName: "Mars"},
	{Name: "Europa", Type: "moon", ParentName: "Jupiter"},
	// Multi-word name: advertised subdomain is the hyphenated slug "voyager-1".
	{Name: "Voyager 1", Type: "spacecraft"},
}

func TestResolveCelestialHost(t *testing.T) {
	// Override global celestialObjects with test data for this specific test.
	originalCelestialObjects := getCelestialObjects()
	setCelestialObjects(testCelestialObjects)
	defer func() { setCelestialObjects(originalCelestialObjects) }() // Restore original celestialObjects data after the test completes.

	testCases := []struct {
		name             string
		host             string
		expectedBodyName string // "" means the host names no known body
	}{
		// Valid info-page hosts.
		{"Planet", "mars.latency.space", "Mars"},
		{"Moon with parent", "phobos.mars.latency.space", "Phobos"},
		{"Multi-word body via hyphenated slug", "voyager-1.latency.space", "Voyager 1"},
		{"Case insensitivity - planet", "MARS.latency.space", "Mars"},
		{"Case insensitivity - moon", "PHOBOS.MARS.LATENCY.SPACE", "Phobos"},
		{"Host with port", "mars.latency.space:8080", "Mars"},

		// Target-embedding forms are no longer resolved (they never resolved in
		// public DNS; proxying is done over SOCKS).
		{"Embedded target on planet rejected", "www.example.com.mars.latency.space", ""},
		{"Embedded target on moon rejected", "www.example.com.phobos.mars.latency.space", ""},
		{"Embedded target slug rejected", "www.example.com.voyager-1.latency.space", ""},

		// Invalid / malformed hosts.
		{"Invalid moon parent", "phobos.jupiter.latency.space", ""}, // Phobos orbits Mars
		{"Moon under non-parent planet", "moon.mars.latency.space", ""},
		{"Non-existent body", "unknown.latency.space", ""},
		{"Bare apex", "latency.space", ""},
		{"Wrong TLD", "mars.latency.com", ""},
		{"Unrelated domain", "example.com", ""},
	}

	// Instantiate a Server to call the method under test.
	s := &Server{}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := s.resolveCelestialHost(tc.host)
			if !strings.EqualFold(got, tc.expectedBodyName) {
				t.Errorf("host '%s': expected body name '%s', got '%s'", tc.host, tc.expectedBodyName, got)
			}
		})
	}
}

// TestFindObjectByName tests the helper function directly.
func TestFindObjectByName(t *testing.T) {
	// Use the same test object data.
	originalCelestialObjects := getCelestialObjects()
	setCelestialObjects(testCelestialObjects)
	defer func() { setCelestialObjects(originalCelestialObjects) }()

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
				objectsToSearch = getCelestialObjects()
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

// TestDisplayCelestialInfoTemplate tests the rendering of the info page HTML template.
func TestDisplayCelestialInfoTemplate(t *testing.T) {
	// Initialize real celestial objects data.
	setCelestialObjects(celestial.InitSolarSystemObjects())
	// Ensure celestialObjects were loaded.
	if len(getCelestialObjects()) == 0 {
		t.Fatal("Failed to initialize celestialObjects (slice is nil or empty)")
	}

	// Populate the distance cache.
	calculateDistancesFromEarth(getCelestialObjects(), time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))

	// Parse the HTML template.
	var err error
	infoTemplate, err = template.ParseFiles("templates/info_page.html")
	if err != nil {
		t.Fatalf("Failed to parse info_page.html template: %v", err)
	}

	// Create a mock HTTP response recorder.
	recorder := httptest.NewRecorder()

	// Create a Server instance.
	s := NewServer(80, false, true, true, "") // Port/HTTPS don't matter for this test

	// Call the function being tested.
	testBodyName := "Mars"
	s.displayCelestialInfo(recorder, testBodyName)

	// Assert the HTTP status code is OK.
	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, recorder.Code)
	}

	// Assert the rendered HTML content.
	body := recorder.Body.String()

	// Check for key elements and text in the HTML output.
	expectedTitle := fmt.Sprintf("<title>%s - Latency Space Proxy</title>", testBodyName)
	if !strings.Contains(body, expectedTitle) {
		t.Errorf("Response body does not contain expected title: %s", expectedTitle)
	}
	expectedH1 := fmt.Sprintf("<h1>%s Proxy</h1>", testBodyName)
	if !strings.Contains(body, expectedH1) {
		t.Errorf("Response body does not contain expected H1: %s", expectedH1)
	}
	if !strings.Contains(body, "Distance from Earth:") {
		t.Errorf("Response body does not contain 'Distance from Earth:'")
	}
	if !strings.Contains(body, "Status:") {
		t.Errorf("Response body does not contain 'Status:'")
	}

	// Check if the correct domain is shown in the usage examples.

	// Check if the correct domain is shown in the usage examples.
	expectedDomain := fmt.Sprintf("<code>%s</code>", FormatFullDomain(testBodyName))
	if !strings.Contains(body, expectedDomain) {
		t.Errorf("Response body does not contain expected domain code block: %s", expectedDomain)
	}

	// Check for moon links if applicable (Mars has moons).
	if testBodyName == "Mars" {
		phobosLink := fmt.Sprintf(`<li><a href="http://%s/">Phobos</a></li>`, FormatMoonDomain("Phobos", "Mars"))
		if !strings.Contains(body, phobosLink) {
			t.Errorf("Response body for Mars does not contain Phobos link")
		}
		deimosLink := fmt.Sprintf(`<li><a href="http://%s/">Deimos</a></li>`, FormatMoonDomain("Deimos", "Mars"))
		if !strings.Contains(body, deimosLink) {
			t.Errorf("Response body for Mars does not contain Deimos link")
		}
	}

	// Assert the Content-Type header.
	expectedContentType := "text/html; charset=utf-8"
	actualContentType := recorder.Header().Get("Content-Type")
	if actualContentType != expectedContentType {
		t.Errorf("Expected Content-Type '%s', got '%s'", expectedContentType, actualContentType)
	}
}
