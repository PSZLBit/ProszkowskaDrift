// Package supervisor launches and restarts the Godot game process.
package supervisor

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Spec describes one game process launch.
type Spec struct {
	Path    string
	Args    []string
	Dir     string
	Env     []string
	GameURL string
}

// Process is the subset of os.Process the supervisor needs.
type Process interface {
	Wait() error
	Kill() error
	Pid() int
}

// Starter starts one process.
type Starter func(Spec) (Process, error)

// Supervisor owns the game process lifecycle and the current one-time game
// token.
type Supervisor struct {
	spec    Spec
	log     *slog.Logger
	starter Starter

	tokenMu sync.RWMutex
	token   string

	restartCh chan string
	stopCh    chan struct{}
	stopOnce  sync.Once
	done      chan struct{}
	stopping  atomic.Bool
}

// New creates a supervisor using the real OS process starter.
func New(spec Spec, log *slog.Logger) *Supervisor {
	if log == nil {
		log = slog.Default()
	}
	s := &Supervisor{
		spec:      spec,
		log:       log,
		starter:   execStarter(log),
		restartCh: make(chan string, 1),
		stopCh:    make(chan struct{}),
		done:      make(chan struct{}),
	}
	return s
}

// Done is closed when Run exits.
func (s *Supervisor) Done() <-chan struct{} { return s.done }

// Stop prevents further restarts and asks Run to stop the current process.
// It is safe to call more than once.
func (s *Supervisor) Stop() {
	s.stopping.Store(true)
	s.stopOnce.Do(func() { close(s.stopCh) })
}

// Token returns the token for the current game launch. It is empty before the
// first launch.
func (s *Supervisor) Token() string {
	s.tokenMu.RLock()
	defer s.tokenMu.RUnlock()
	return s.token
}

func (s *Supervisor) setToken(token string) {
	s.tokenMu.Lock()
	s.token = token
	s.tokenMu.Unlock()
}

// RestartFor handles a game-socket loss reported for token. A stale handler
// must not invalidate or restart a newer launch; in that case it is ignored.
// The current token is cleared before the restart request so a stale game
// client cannot reconnect in the meantime.
func (s *Supervisor) RestartFor(token, reason string) {
	s.tokenMu.Lock()
	if s.token != "" && s.token != token {
		s.tokenMu.Unlock()
		return
	}
	s.token = ""
	s.tokenMu.Unlock()
	s.restart(reason)
}

// Restart asks Run to kill and restart the game. It never blocks.
func (s *Supervisor) Restart(reason string) {
	s.restart(reason)
}

func (s *Supervisor) restart(reason string) {
	if s.stopping.Load() {
		return
	}
	select {
	case s.restartCh <- reason:
	default:
	}
}

// Run owns the process loop until ctx is cancelled.
func (s *Supervisor) Run(ctx context.Context) {
	defer close(s.done)
	defer s.stopping.Store(true)
	backoff := time.Second
	for {
		if ctx.Err() != nil || s.stopping.Load() {
			return
		}
		// A restart requested around a process exit must not cause a second
		// launch immediately after the next process starts.
		select {
		case <-s.restartCh:
		default:
		}
		token, err := newToken()
		if err != nil {
			s.log.Error("generate game token", "error", err)
			if !sleep(ctx, s.stopCh, backoff) {
				return
			}
			backoff = nextBackoff(backoff)
			continue
		}
		s.setToken(token)
		spec := s.spec
		spec.Env = withEnv(s.spec.Env,
			"PD_GAME_TOKEN="+token,
			"PD_SERVER_URL="+s.spec.GameURL,
			"PD_PROTOCOL_VERSION=1",
		)
		startedAt := time.Now()
		proc, err := s.starter(spec)
		if err != nil {
			s.log.Error("start game", "error", err, "path", s.spec.Path)
			if !sleep(ctx, s.stopCh, backoff) {
				return
			}
			backoff = nextBackoff(backoff)
			continue
		}
		s.log.Info("game started", "pid", proc.Pid(), "path", s.spec.Path)

		waitCh := make(chan error, 1)
		go func() { waitCh <- proc.Wait() }()

		select {
		case err := <-waitCh:
			if err != nil {
				s.log.Warn("game exited", "error", err)
			} else {
				s.log.Info("game exited")
			}
		case reason := <-s.restartCh:
			s.log.Warn("restarting game", "reason", reason)
			_ = proc.Kill()
			<-waitCh
		case <-s.stopCh:
			s.log.Info("stopping game")
			_ = proc.Kill()
			<-waitCh
			return
		case <-ctx.Done():
			s.log.Info("stopping game")
			_ = proc.Kill()
			<-waitCh
			return
		}

		if time.Since(startedAt) > 60*time.Second {
			backoff = time.Second
		} else {
			backoff = nextBackoff(backoff)
		}
		if !sleep(ctx, s.stopCh, backoff) {
			return
		}
	}
}

func nextBackoff(d time.Duration) time.Duration {
	d *= 2
	if d > 10*time.Second {
		return 10 * time.Second
	}
	return d
}

func sleep(ctx context.Context, stop <-chan struct{}, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-stop:
		return false
	case <-ctx.Done():
		return false
	}
}

func newToken() (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}

func withEnv(base []string, extra ...string) []string {
	if base == nil {
		base = os.Environ()
	}
	values := make(map[string]string, len(base)+len(extra))
	order := make([]string, 0, len(base)+len(extra))
	for _, item := range base {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		if _, seen := values[key]; !seen {
			order = append(order, key)
		}
		values[key] = value
	}
	for _, item := range extra {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		if _, seen := values[key]; !seen {
			order = append(order, key)
		}
		values[key] = value
	}
	out := make([]string, 0, len(order))
	for _, key := range order {
		out = append(out, key+"="+values[key])
	}
	return out
}

type execProcess struct {
	cmd *exec.Cmd
}

func (p execProcess) Wait() error { return p.cmd.Wait() }
func (p execProcess) Kill() error {
	if p.cmd.Process == nil {
		return nil
	}
	return p.cmd.Process.Kill()
}
func (p execProcess) Pid() int {
	if p.cmd.Process == nil {
		return 0
	}
	return p.cmd.Process.Pid
}

func execStarter(log *slog.Logger) Starter {
	return func(spec Spec) (Process, error) {
		if strings.TrimSpace(spec.Path) == "" {
			return nil, fmt.Errorf("game path is empty")
		}
		cmd := exec.Command(spec.Path, spec.Args...)
		cmd.Dir = spec.Dir
		cmd.Env = spec.Env
		cmd.Stdout = logWriter{log: log, stream: "stdout"}
		cmd.Stderr = logWriter{log: log, stream: "stderr"}
		if err := cmd.Start(); err != nil {
			return nil, err
		}
		return execProcess{cmd: cmd}, nil
	}
}

type logWriter struct {
	log    *slog.Logger
	stream string
}

func (w logWriter) Write(p []byte) (int, error) {
	line := strings.TrimSpace(string(p))
	if line != "" {
		w.log.Info("game output", "stream", w.stream, "line", line)
	}
	return len(p), nil
}
