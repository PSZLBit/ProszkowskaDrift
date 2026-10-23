package validation

import (
	"math"
	"testing"
	"time"
)

func TestValidInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		steer    float64
		throttle float64
		wantErr  bool
	}{
		{name: "zero", steer: 0, throttle: 0},
		{name: "lower bounds", steer: -1, throttle: -1},
		{name: "upper bounds", steer: 1, throttle: 1},
		{name: "partial", steer: -0.35, throttle: 0.8},
		{name: "steer below range", steer: -1.01, throttle: 0, wantErr: true},
		{name: "steer above range", steer: 1.01, throttle: 0, wantErr: true},
		{name: "throttle below range", steer: 0, throttle: -1.01, wantErr: true},
		{name: "throttle above range", steer: 0, throttle: 1.01, wantErr: true},
		{name: "nan steer", steer: math.NaN(), throttle: 0, wantErr: true},
		{name: "nan throttle", steer: 0, throttle: math.NaN(), wantErr: true},
		{name: "positive infinity", steer: math.Inf(1), throttle: 0, wantErr: true},
		{name: "negative infinity", steer: 0, throttle: math.Inf(-1), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidInput(tt.steer, tt.throttle)
			if tt.wantErr && err == nil {
				t.Fatal("expected an error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestSequence(t *testing.T) {
	t.Parallel()

	var seq Sequence

	if !seq.Accept(0) {
		t.Fatal("first sequence number should be accepted")
	}
	if !seq.Accept(1) {
		t.Fatal("increasing sequence number should be accepted")
	}
	if seq.Accept(1) {
		t.Fatal("equal sequence number should be rejected")
	}
	if seq.Accept(0) {
		t.Fatal("older sequence number should be rejected")
	}
	if !seq.Accept(2) {
		t.Fatal("newer sequence number should be accepted after a rejection")
	}

	seq.Reset()
	if !seq.Accept(0) {
		t.Fatal("sequence should restart from any value after Reset")
	}
}

func TestSequenceFirstValueCanBeNegative(t *testing.T) {
	t.Parallel()

	var seq Sequence
	if !seq.Accept(-10) {
		t.Fatal("first sequence number -10 should be accepted")
	}
	if !seq.Accept(-9) {
		t.Fatal("sequence number -9 should be accepted after -10")
	}
	if seq.Accept(-11) {
		t.Fatal("sequence number -11 should be rejected")
	}
}

func TestRateLimiter(t *testing.T) {
	t.Parallel()

	now := time.Unix(1000, 0)
	limiter := NewRateLimiter(60, time.Second)

	for i := 0; i < 60; i++ {
		if !limiter.Allow(now) {
			t.Fatalf("message %d should be allowed", i+1)
		}
	}
	if limiter.Allow(now) {
		t.Fatal("message 61 in the same second should be rejected")
	}

	now = now.Add(500 * time.Millisecond)
	if limiter.Allow(now) {
		t.Fatal("message should still be rejected within the window")
	}

	now = now.Add(500 * time.Millisecond)
	if !limiter.Allow(now) {
		t.Fatal("message should be allowed once the first message leaves the window")
	}
}

func TestRateLimiterIsPerInstance(t *testing.T) {
	t.Parallel()

	now := time.Unix(1000, 0)
	first := NewRateLimiter(1, time.Second)
	second := NewRateLimiter(1, time.Second)

	if !first.Allow(now) {
		t.Fatal("first limiter should allow the first message")
	}
	if first.Allow(now) {
		t.Fatal("first limiter should reject the second message")
	}
	if !second.Allow(now) {
		t.Fatal("second limiter should be independent")
	}
}
