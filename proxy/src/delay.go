// proxy/src/delay.go
//
// Latency simulation primitives. Light-speed delay shifts every byte in time
// by a constant; it does not reduce throughput. The old SOCKS relay slept one
// full one-way latency per 32KB chunk, so a Mars link (~12 min one-way)
// carried roughly 45 bytes/s and a TLS handshake could take over an hour.
// delayCopy instead timestamps chunks as they arrive and releases each one
// exactly `latency` later: throughput is preserved while every byte still
// arrives late by the light-travel time.
package main

import (
	"context"
	"io"
	"time"
)

const (
	delayChunkSize = 32 * 1024
	// delayQueueLen bounds buffered in-flight data per direction
	// (delayQueueLen * delayChunkSize = 512KB). When the queue is full the
	// reader stalls, which acts as crude bandwidth backpressure.
	delayQueueLen = 16
)

type timedChunk struct {
	data      []byte
	deliverAt time.Time
}

// delayCopy copies src to dst, delaying each chunk by latency. onBytes, if
// non-nil, is called with the size of each chunk written (for metrics).
// Returns the first error from either side; io.EOF is reported as nil.
func delayCopy(ctx context.Context, dst io.Writer, src io.Reader, latency time.Duration, onBytes func(int)) error {
	queue := make(chan timedChunk, delayQueueLen)
	readErr := make(chan error, 1)

	go func() {
		defer close(queue)
		for {
			buf := make([]byte, delayChunkSize)
			n, err := src.Read(buf)
			if n > 0 {
				select {
				case queue <- timedChunk{data: buf[:n], deliverAt: time.Now().Add(latency)}:
				case <-ctx.Done():
					readErr <- ctx.Err()
					return
				}
			}
			if err != nil {
				if err == io.EOF {
					readErr <- nil
				} else {
					readErr <- err
				}
				return
			}
		}
	}()

	for chunk := range queue {
		if err := sleepCtx(ctx, time.Until(chunk.deliverAt)); err != nil {
			return err
		}
		if _, err := dst.Write(chunk.data); err != nil {
			return err
		}
		if onBytes != nil {
			onBytes(len(chunk.data))
		}
	}
	return <-readErr
}

// sleepCtx sleeps for d but aborts early if ctx is cancelled.
func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
