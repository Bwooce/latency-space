// test_helpers.go - Contains helper functions for testing

package main

import (
	"sync/atomic"
	"time"
)

// Variable to enable test mode - shared between test files.
// atomic.Bool to match the proxy build, whose calculations.go (symlinked into
// this tool) reads it via isTestMode.Load().
var isTestMode atomic.Bool

// setupTestMode enables test mode for latency calculations and returns a cleanup function
// nolint:unused
func setupTestMode() func() {
	isTestMode.Store(true)
	return func() {
		isTestMode.Store(false)
	}
}

// testModeCalculateLatency returns a fixed low latency for testing
func testModeCalculateLatency(distance float64) time.Duration {
	return 3 * time.Millisecond
}
