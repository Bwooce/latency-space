package main

import (
	"fmt"
	"strings"
)

// FormatDomainName formats the name of a celestial body or spacecraft into a valid domain name
// It converts the name to lowercase and replaces spaces with hyphens
func FormatDomainName(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", "-"))
}

// FormatFullDomain formats a full domain with the latency.space suffix
func FormatFullDomain(name string) string {
	return fmt.Sprintf("%s.latency.space", FormatDomainName(name))
}

// FormatMoonDomain formats a moon domain with its parent planet
func FormatMoonDomain(moonName, planetName string) string {
	return fmt.Sprintf("%s.%s.latency.space", FormatDomainName(moonName), FormatDomainName(planetName))
}

// FormatTargetDomain formats a target domain with a celestial body
// Example: example.com.mars.latency.space
func FormatTargetDomain(targetDomain, celestialName string) string {
	return fmt.Sprintf("%s.%s.latency.space", targetDomain, FormatDomainName(celestialName))
}

// FormatMoonTargetDomain formats a target domain with a moon and its parent planet
// Example: example.com.europa.jupiter.latency.space
func FormatMoonTargetDomain(targetDomain, moonName, planetName string) string {
	return fmt.Sprintf("%s.%s.%s.latency.space", targetDomain, FormatDomainName(moonName), FormatDomainName(planetName))
}
