// test_helpers.go - Contains helper functions for testing

package main

import (
	"sync/atomic"
	"time"
)

// Test-mode flags. Atomic because test goroutines (e.g. the SOCKS UDP relay)
// can still be running and reading these when a test's cleanup resets them —
// plain reads/writes there are a data race under -race. In production these
// are set once at startup and never mutated.
var isTestMode atomic.Bool

// testModeLatencyOverride holds a fixed latency (nanoseconds) for tests; 0 = unset.
var testModeLatencyOverride atomic.Int64

// UDP relay close delay for testing connection termination
// nolint:unused
var testModeUDPRelayCloseDelay time.Duration = 500 * time.Millisecond

// setupTestMode enables test mode for latency calculations and returns a cleanup function
func setupTestMode() func() {
	isTestMode.Store(true)
	testModeLatencyOverride.Store(0)
	return func() {
		isTestMode.Store(false)
		testModeLatencyOverride.Store(0)
	}
}

// setupTestModeWithLatency enables test mode with a specific latency and returns a cleanup function
func setupTestModeWithLatency(latency time.Duration) func() {
	isTestMode.Store(true)
	testModeLatencyOverride.Store(int64(latency))
	return func() {
		isTestMode.Store(false)
		testModeLatencyOverride.Store(0)
	}
}

// testModeCalculateLatency returns configurable latency for testing
func testModeCalculateLatency(distance float64) time.Duration {
	// If we have an override set, use that
	if v := testModeLatencyOverride.Load(); v > 0 {
		return time.Duration(v)
	}

	// Default test latency
	return 3 * time.Millisecond
}
