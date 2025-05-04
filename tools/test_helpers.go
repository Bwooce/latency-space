// test_helpers.go - Contains helper functions for testing

package main

import "time"

// Variable to enable test mode - shared between test files
var isTestMode = false

// setupTestMode enables test mode for latency calculations and returns a cleanup function
// nolint:unused
func setupTestMode() func() {
	// Set test mode
	isTestMode = true
	
	// Return a function to reset it
	return func() {
		isTestMode = false
	}
}

// testModeCalculateLatency returns a fixed low latency for testing
func testModeCalculateLatency(distance float64) time.Duration {
	return 3 * time.Millisecond
}