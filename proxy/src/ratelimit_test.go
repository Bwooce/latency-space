// proxy/src/ratelimit_test.go
package main

import "testing"

func TestRateLimiterPerIPConcurrency(t *testing.T) {
	rl := NewRateLimiter(0 /* rate disabled */, 0, 2 /* maxPerIP */, 100)

	r1, err := rl.Acquire("1.2.3.4")
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	if _, err := rl.Acquire("1.2.3.4"); err != nil {
		t.Fatalf("second acquire: %v", err)
	}
	if _, err := rl.Acquire("1.2.3.4"); err == nil {
		t.Fatal("third concurrent connection from same IP should be rejected")
	}

	// A different IP is unaffected.
	if _, err := rl.Acquire("5.6.7.8"); err != nil {
		t.Fatalf("other IP should be admitted: %v", err)
	}

	// Releasing one frees a slot.
	r1()
	if _, err := rl.Acquire("1.2.3.4"); err != nil {
		t.Fatalf("acquire after release should succeed: %v", err)
	}
}

func TestRateLimiterGlobalCap(t *testing.T) {
	rl := NewRateLimiter(0, 0, 100, 2 /* maxTotal */)
	if _, err := rl.Acquire("a"); err != nil {
		t.Fatal(err)
	}
	if _, err := rl.Acquire("b"); err != nil {
		t.Fatal(err)
	}
	if _, err := rl.Acquire("c"); err == nil {
		t.Fatal("global cap should reject the third connection regardless of IP")
	}
}

func TestRateLimiterRate(t *testing.T) {
	// 1/min with burst 3, and concurrency uncapped so only the rate bucket
	// is under test. A very low rate keeps refill negligible during the test.
	rl := NewRateLimiter(1 /* per min */, 3 /* burst */, 0, 0)

	for i := 0; i < 3; i++ {
		rel, err := rl.Acquire("9.9.9.9")
		if err != nil {
			t.Fatalf("burst acquire %d rejected: %v", i, err)
		}
		rel()
	}
	if _, err := rl.Acquire("9.9.9.9"); err == nil {
		t.Fatal("fourth connection should exceed the burst/rate")
	}
}

func TestRateLimiterNilAndDisabled(t *testing.T) {
	var rl *RateLimiter // nil is a valid no-op limiter
	if rel, err := rl.Acquire("x"); err != nil || rel == nil {
		t.Fatal("nil limiter should admit everything")
	}

	// All-zero config disables every check.
	open := NewRateLimiter(0, 0, 0, 0)
	for i := 0; i < 50; i++ {
		if _, err := open.Acquire("y"); err != nil {
			t.Fatalf("all-zero config should disable limits, got %v", err)
		}
	}
}
