package supervisor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	switch os.Getenv("SUPERVISOR_TEST_HELPER") {
	case "crash":
		os.Exit(1)
	case "sleep":
		time.Sleep(time.Minute)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func helperBinary(t *testing.T) string {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	return exe
}

func TestMissingBinary(t *testing.T) {
	s := New(Config{
		Binary:         filepath.Join(os.TempDir(), "definitely-not-a-proszkowska-binary"),
		MaxRestarts:    1,
		InitialBackoff: time.Millisecond,
	})
	if err := s.Run(context.Background()); err == nil {
		t.Fatal("expected error for missing binary")
	}
}

func TestCrashLoop(t *testing.T) {
	s := New(Config{
		Binary:         helperBinary(t),
		Args:           nil,
		Env:            []string{"SUPERVISOR_TEST_HELPER=crash"},
		MaxRestarts:    1,
		InitialBackoff: time.Millisecond,
		MaxBackoff:     time.Millisecond,
	})
	err := s.Run(context.Background())
	if err == nil {
		t.Fatal("expected crash-loop error")
	}
	if !strings.Contains(err.Error(), "game crash loop") {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Restarts() < 2 {
		t.Fatalf("restarts = %d, want at least 2", s.Restarts())
	}
}

func TestMaxRestartsZero(t *testing.T) {
	s := New(Config{
		Binary:         helperBinary(t),
		Env:            []string{"SUPERVISOR_TEST_HELPER=crash"},
		MaxRestarts:    0,
		InitialBackoff: time.Millisecond,
		MaxBackoff:     time.Millisecond,
	})
	err := s.Run(context.Background())
	if err == nil {
		t.Fatal("expected crash-loop error")
	}
	if s.Restarts() != 1 {
		t.Fatalf("restarts = %d, want 1", s.Restarts())
	}
}

func TestStopsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	s := New(Config{
		Binary:         helperBinary(t),
		Env:            []string{"SUPERVISOR_TEST_HELPER=sleep"},
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

func TestStopResponsiveness(t *testing.T) {
	s := New(Config{
		Binary:         helperBinary(t),
		Env:            []string{"SUPERVISOR_TEST_HELPER=crash"},
		MaxRestarts:    5,
		InitialBackoff: time.Second,
		MaxBackoff:     time.Second,
	})
	done := make(chan error, 1)
	go func() { done <- s.Run(context.Background()) }()
	// Wait for the first crash to land in the backoff window.
	time.Sleep(100 * time.Millisecond)
	s.Stop()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run error = %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("supervisor did not stop after Stop()")
	}
}

func TestAppendEnvOverride(t *testing.T) {
	base := []string{"FOO=base", "BAR=base"}
	extra := []string{"FOO=override", "BAZ=extra", "foo=second"}
	got := appendEnv(base, extra)
	// FOO overridden (case-insensitive), BAR kept, BAZ added, duplicate FOO collapsed.
	joined := joinEnv(got)
	if !strings.Contains(joined, "foo=second") {
		t.Fatalf("expected last FOO override, got %v", got)
	}
	if strings.Contains(joined, "FOO=base") || strings.Contains(joined, "FOO=override") {
		t.Fatalf("stale FOO values remain: %v", got)
	}
	if !strings.Contains(joined, "BAR=base") || !strings.Contains(joined, "BAZ=extra") {
		t.Fatalf("missing expected entries: %v", got)
	}
	// duplicate keys in extra collapsed to one.
	count := 0
	for _, item := range got {
		if key, _, _ := splitEnv(item); strings.EqualFold(key, "foo") {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected one foo entry, got %d: %v", count, got)
	}
}

func joinEnv(items []string) string {
	out := ""
	for i, item := range items {
		if i > 0 {
			out += ";"
		}
		out += item
	}
	return out
}
