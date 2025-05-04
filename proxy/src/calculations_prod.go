// +build !test

package main

import "time"

// This file contains production-only definitions that are not used in test mode

// Variables for test mode (not used in production)
var isTestMode = false

// Stub function for production builds
func testModeCalculateLatency(distance float64) time.Duration {
	return CalculateLatency(distance)
}