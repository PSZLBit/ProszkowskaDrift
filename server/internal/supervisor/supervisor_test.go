package supervisor

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMissingBinary(t *testing.T) {
	s := New(Config{Binary: "/definitely/not/a/binary", MaxRestarts: 1, InitialBackoff: time.Millisecond})
	if err := s.Run(context.Background()); err == nil {
		t.Fatal("expected error for missing binary")
	}
}

func TestCrashLoop(t *testing.T) {
	s := New(Config{
		Binary:         "/bin/false",
		MaxRestarts:    1,
		InitialBackoff: time.Millisecond,
		MaxBackoff:     time.Millisecond,
	})
	err := s.Run(context.Background())
	if err == nil {
		t.Fatal("expected crash-loop error")
	}
	if s.Restarts() < 2 {
		t.Fatalf("restarts = %d, want at least 2", s.Restarts())
	}
}

func TestStopsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	s := New(Config{
		Binary:         "/bin/sleep",
		Args:           []string{"10"},
		MaxRestarts:    1,
		InitialBackoff: time.Millisecond,
		MaxBackoff:     time.Millisecond,
	})
	done := make(chan error, 1)
	go func() { done <- s.Run(ctx) }()
	time.Sleep(50 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Fatalf("run error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("supervisor did not stop")
	}
}
