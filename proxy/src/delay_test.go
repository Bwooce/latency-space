// proxy/src/delay_test.go
package main

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"
)

// TestDelayCopyShiftsButDoesNotThrottle is the core throughput regression:
// the delay line must shift the whole stream by one latency, NOT sleep once
// per chunk (which the old relay did, throttling throughput to chunk/latency).
func TestDelayCopyShiftsButDoesNotThrottle(t *testing.T) {
	latency := 100 * time.Millisecond
	payload := bytes.Repeat([]byte("x"), 256*1024) // 8 chunks of 32KB

	var out bytes.Buffer
	start := time.Now()
	err := delayCopy(context.Background(), &out, bytes.NewReader(payload), latency, nil)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("delayCopy: %v", err)
	}
	if out.Len() != len(payload) {
		t.Fatalf("copied %d bytes, want %d", out.Len(), len(payload))
	}
	if elapsed < latency {
		t.Errorf("finished in %v, expected at least the %v delay", elapsed, latency)
	}
	// Old behavior: 8 chunks * 100ms = 800ms minimum. Delay-line behavior:
	// one 100ms shift for the whole stream.
	if elapsed > latency+200*time.Millisecond {
		t.Errorf("took %v: per-chunk sleeping detected, expected about %v total", elapsed, latency)
	}
}

// TestDelayCopyCancellation ensures a cancelled context tears the copy down
// promptly instead of holding the full (possibly interplanetary) delay.
func TestDelayCopyCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	pr, pw := io.Pipe()
	defer pw.Close()

	done := make(chan error, 1)
	go func() {
		var out bytes.Buffer
		done <- delayCopy(ctx, &out, pr, time.Hour, nil)
	}()

	if _, err := pw.Write([]byte("stranded in transit")); err != nil {
		t.Fatalf("write: %v", err)
	}
	cancel()

	select {
	case err := <-done:
		if err == nil {
			t.Error("expected context error, got nil")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("delayCopy did not return after cancellation")
	}
}

// TestDelayCopyMetricsCallback checks onBytes is invoked with the full byte
// count (the relay uses it for bandwidth metrics).
func TestDelayCopyMetricsCallback(t *testing.T) {
	payload := bytes.Repeat([]byte("y"), 100*1024)
	var counted int
	var out bytes.Buffer
	err := delayCopy(context.Background(), &out, bytes.NewReader(payload), time.Millisecond, func(n int) {
		counted += n
	})
	if err != nil {
		t.Fatalf("delayCopy: %v", err)
	}
	if counted != len(payload) {
		t.Errorf("onBytes counted %d, want %d", counted, len(payload))
	}
}
