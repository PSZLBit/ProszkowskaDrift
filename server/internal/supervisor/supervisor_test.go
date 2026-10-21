package supervisor

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"
)

type fakeProcess struct {
	pid      int
	wait     chan error
	killOnce sync.Once
}

func newFakeProcess(pid int) *fakeProcess {
	return &fakeProcess{pid: pid, wait: make(chan error, 1)}
}

func (p *fakeProcess) Wait() error { return <-p.wait }
func (p *fakeProcess) Pid() int    { return p.pid }
func (p *fakeProcess) Kill() error {
	p.killOnce.Do(func() { p.wait <- nil })
	return nil
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestWithEnvOverrides(t *testing.T) {
	got := withEnv([]string{"A=1", "B=2", "A=3"}, "B=4", "C=5")
	want := []string{"A=3", "B=4", "C=5"}
	if len(got) != len(want) {
		t.Fatalf("env = %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("env[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestWithEnvInheritsProcessEnvironment(t *testing.T) {
	t.Setenv("PD_TEST_INHERIT", "present")
	got := withEnv(nil, "PD_EXTRA=1")
	seenInherited := false
	seenExtra := false
	for _, item := range got {
		switch item {
		case "PD_TEST_INHERIT=present":
			seenInherited = true
		case "PD_EXTRA=1":
			seenExtra = true
		}
	}
	if !seenInherited || !seenExtra {
		t.Fatalf("env did not inherit and override: inherited=%v extra=%v", seenInherited, seenExtra)
	}
}

func TestNewToken(t *testing.T) {
	a, err := newToken()
	if err != nil {
		t.Fatal(err)
	}
	b, err := newToken()
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != 64 || a == b {
		t.Fatalf("tokens not unique 32-byte hex: %q %q", a, b)
	}
}

func TestRestartForIgnoresStaleToken(t *testing.T) {
	s := New(Spec{Path: "fake-game"}, discardLogger())
	s.setToken("current")

	s.RestartFor("old", "stale handler")
	if got := s.Token(); got != "current" {
		t.Fatalf("stale RestartFor cleared token to %q", got)
	}
	select {
	case reason := <-s.restartCh:
		t.Fatalf("stale RestartFor queued restart: %s", reason)
	default:
	}

	s.RestartFor("current", "socket lost")
	if got := s.Token(); got != "" {
		t.Fatalf("token = %q after RestartFor", got)
	}
	select {
	case reason := <-s.restartCh:
		if reason != "socket lost" {
			t.Fatalf("restart reason = %q", reason)
		}
	default:
		t.Fatal("RestartFor did not queue a restart")
	}
}

func TestNextBackoff(t *testing.T) {
	if got := nextBackoff(time.Second); got != 2*time.Second {
		t.Fatalf("backoff = %v", got)
	}
	if got := nextBackoff(8 * time.Second); got != 10*time.Second {
		t.Fatalf("backoff cap = %v", got)
	}
}

func TestStopEndsRunWithoutRestart(t *testing.T) {
	s := New(Spec{Path: "fake-game", GameURL: "ws://127.0.0.1:8080/ws/game"}, discardLogger())
	procs := make(chan *fakeProcess, 2)
	s.starter = func(spec Spec) (Process, error) {
		p := newFakeProcess(1)
		procs <- p
		return p, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		s.Run(ctx)
		close(done)
	}()
	select {
	case <-procs:
	case <-time.After(2 * time.Second):
		t.Fatal("process did not start")
	}
	s.Stop()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("supervisor did not stop")
	}
	select {
	case <-procs:
		t.Fatal("supervisor restarted after Stop")
	case <-time.After(100 * time.Millisecond):
	}
}

func TestRunRestartsOnRequest(t *testing.T) {
	s := New(Spec{Path: "fake-game", GameURL: "ws://127.0.0.1:8080/ws/game"}, discardLogger())
	procs := make(chan *fakeProcess, 4)
	pid := 0
	s.starter = func(spec Spec) (Process, error) {
		if spec.Env == nil {
			t.Fatal("starter received no environment")
		}
		pid++
		p := newFakeProcess(pid)
		procs <- p
		return p, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		s.Run(ctx)
		close(done)
	}()

	var first *fakeProcess
	select {
	case first = <-procs:
	case <-time.After(2 * time.Second):
		t.Fatal("first process did not start")
	}
	token1 := s.Token()
	if token1 == "" {
		t.Fatal("token was not set before launch")
	}

	s.Restart("test")
	select {
	case second := <-procs:
		if second.Pid() == first.Pid() {
			t.Fatal("process was not restarted")
		}
		if s.Token() == token1 {
			t.Fatal("game token was not rotated")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("second process did not start")
	}

	cancel()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("supervisor did not stop")
	}
}
