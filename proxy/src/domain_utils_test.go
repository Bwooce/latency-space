package main

import (
	"testing"
)

func TestFormatDomainName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple name",
			input:    "mars",
			expected: "mars",
		},
		{
			name:     "Name with uppercase",
			input:    "Mars",
			expected: "mars",
		},
		{
			name:     "Name with spaces",
			input:    "Voyager 1",
			expected: "voyager-1",
		},
		{
			name:     "Name with multiple spaces",
			input:    "James Webb Space Telescope",
			expected: "james-webb-space-telescope",
		},
		{
			name:     "Already formatted",
			input:    "voyager-1",
			expected: "voyager-1",
		},
		{
			name:     "Mixed case with spaces",
			input:    "New Horizons",
			expected: "new-horizons",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDomainName(tt.input)
			if result != tt.expected {
				t.Errorf("FormatDomainName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatFullDomain(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple name",
			input:    "mars",
			expected: "mars.latency.space",
		},
		{
			name:     "Name with uppercase",
			input:    "Mars",
			expected: "mars.latency.space",
		},
		{
			name:     "Name with spaces",
			input:    "Voyager 1",
			expected: "voyager-1.latency.space",
		},
		{
			name:     "Name with multiple spaces",
			input:    "James Webb Space Telescope",
			expected: "james-webb-space-telescope.latency.space",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatFullDomain(tt.input)
			if result != tt.expected {
				t.Errorf("FormatFullDomain(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatMoonDomain(t *testing.T) {
	tests := []struct {
		name        string
		moonName    string
		planetName  string
		expected    string
	}{
		{
			name:        "Simple names",
			moonName:    "phobos",
			planetName:  "mars",
			expected:    "phobos.mars.latency.space",
		},
		{
			name:        "Names with uppercase",
			moonName:    "Phobos",
			planetName:  "Mars",
			expected:    "phobos.mars.latency.space",
		},
		{
			name:        "Names with spaces",
			moonName:    "Io",
			planetName:  "Jupiter",
			expected:    "io.jupiter.latency.space",
		},
		{
			name:        "Complex moon name",
			moonName:    "Europa North",
			planetName:  "Jupiter",
			expected:    "europa-north.jupiter.latency.space",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatMoonDomain(tt.moonName, tt.planetName)
			if result != tt.expected {
				t.Errorf("FormatMoonDomain(%q, %q) = %q, want %q", tt.moonName, tt.planetName, result, tt.expected)
			}
		})
	}
}

func TestFormatTargetDomain(t *testing.T) {
	tests := []struct {
		name          string
		targetDomain  string
		celestialName string
		expected      string
	}{
		{
			name:          "Simple target and celestial",
			targetDomain:  "example.com",
			celestialName: "mars",
			expected:      "example.com.mars.latency.space",
		},
		{
			name:          "Target with uppercase and celestial with space",
			targetDomain:  "Example.Com",
			celestialName: "Voyager 1",
			expected:      "Example.Com.voyager-1.latency.space",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatTargetDomain(tt.targetDomain, tt.celestialName)
			if result != tt.expected {
				t.Errorf("FormatTargetDomain(%q, %q) = %q, want %q", tt.targetDomain, tt.celestialName, result, tt.expected)
			}
		})
	}
}

func TestFormatMoonTargetDomain(t *testing.T) {
	tests := []struct {
		name          string
		targetDomain  string
		moonName      string
		planetName    string
		expected      string
	}{
		{
			name:          "Simple target and celestials",
			targetDomain:  "example.com",
			moonName:      "phobos",
			planetName:    "mars",
			expected:      "example.com.phobos.mars.latency.space",
		},
		{
			name:          "Complex names",
			targetDomain:  "Example.Com",
			moonName:      "Europa North",
			planetName:    "Jupiter",
			expected:      "Example.Com.europa-north.jupiter.latency.space",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatMoonTargetDomain(tt.targetDomain, tt.moonName, tt.planetName)
			if result != tt.expected {
				t.Errorf("FormatMoonTargetDomain(%q, %q, %q) = %q, want %q", tt.targetDomain, tt.moonName, tt.planetName, result, tt.expected)
			}
		})
	}
}