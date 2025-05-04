// +build test

package main

import "time"

// This file contains test-specific functions for latency calculations

// Variable to enable test mode
var isTestMode = false

// testModeCalculateLatency is a test-specific version that returns a fixed tiny latency
func testModeCalculateLatency(distance float64) time.Duration {
	return 3 * time.Millisecond
}

// Function to set test mode
func setupTestMode() func() {
	// Set test mode
	isTestMode = true
	
	// Return a function to reset it
	return func() {
		isTestMode = false
	}
}