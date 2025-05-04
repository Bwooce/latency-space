// test_helpers.go - Contains helper functions for testing

package main

import "time"

// Variable to enable test mode - shared between test files
var isTestMode = false

// Variables to override test behavior for specific tests
var testModeLatencyOverride time.Duration
// UDP relay close delay for testing connection termination
// nolint:unused
var testModeUDPRelayCloseDelay time.Duration = 500 * time.Millisecond

// setupTestMode enables test mode for latency calculations and returns a cleanup function
func setupTestMode() func() {
	// Set test mode
	isTestMode = true
	
	// Reset latency override
	testModeLatencyOverride = 0
	
	// Return a function to reset it
	return func() {
		isTestMode = false
		testModeLatencyOverride = 0
	}
}

// setupTestModeWithLatency enables test mode with a specific latency and returns a cleanup function
func setupTestModeWithLatency(latency time.Duration) func() {
	// Set test mode
	isTestMode = true
	
	// Set latency override
	testModeLatencyOverride = latency
	
	// Return a function to reset it
	return func() {
		isTestMode = false
		testModeLatencyOverride = 0
	}
}

// testModeCalculateLatency returns configurable latency for testing
func testModeCalculateLatency(distance float64) time.Duration {
	// If we have an override set, use that
	if testModeLatencyOverride > 0 {
		return testModeLatencyOverride
	}
	
	// Default test latency
	return 3 * time.Millisecond
}
