// Package validation contains the pure input checks used by the session.
package validation

import (
	"errors"
	"math"
	"time"
)

// ErrInputRange reports a steer or throttle value that is not a finite number
// in [-1, 1].
var ErrInputRange = errors.New("input value out of range")

// ValidInput reports whether steer and throttle are finite numbers in [-1, 1].
func ValidInput(steer, throttle float64) error {
	if !validAxis(steer) || !validAxis(throttle) {
		return ErrInputRange
	}
	return nil
}

func validAxis(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= -1 && value <= 1
}

// Sequence tracks the last accepted input sequence number for one slot. A
// reconnect calls Reset so a controller may restart its counter.
type Sequence struct {
	last int64
	seen bool
}

// Accept reports whether seq is newer than the last accepted value and records
// it when accepted. The first value after Reset is always accepted.
func (s *Sequence) Accept(seq int64) bool {
	if s.seen && seq <= s.last {
		return false
	}
	s.last = seq
	s.seen = true
	return true
}

// Reset forgets the last accepted sequence number.
func (s *Sequence) Reset() {
	s.last = 0
	s.seen = false
}

// RateLimiter allows at most limit events in any sliding window.
type RateLimiter struct {
	limit  int
	window time.Duration
	times  []time.Time
}

// NewRateLimiter creates a limiter for limit events per window.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{limit: limit, window: window}
}

// Allow records now and reports whether the event is within the limit.
func (r *RateLimiter) Allow(now time.Time) bool {
	if r.limit <= 0 {
		return false
	}

	kept := r.times[:0]
	for _, t := range r.times {
		if now.Sub(t) < r.window {
			kept = append(kept, t)
		}
	}
	r.times = kept

	if len(r.times) >= r.limit {
		return false
	}
	r.times = append(r.times, now)
	return true
}

// Reset clears the rate window.
func (r *RateLimiter) Reset() {
	r.times = r.times[:0]
}
