// Package supervisor starts and monitors the Godot game process.
//
// The server remains the owner of session state; this package only keeps the
// game process running and reports exits. A game crash therefore returns the
// session to attract through the existing game-disconnect path.
package supervisor

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"sync"
	"time"
)

// Config configures the game process supervisor.
type Config struct {
	// Binary is the Godot executable or exported game binary. If empty, the
	// supervisor is disabled.
	Binary string
	// Args are passed to Binary.
	Args []string
	// Env is appended to the process environment. PD_GAME_TOKEN is usually set
	// here so the game can authenticate with the server.
	Env []string
	// MaxRestarts is the number of unexpected exits tolerated before the
	// supervisor stops trying. Zero means no restarts.
	MaxRestarts int
	// InitialBackoff is the delay before the first restart.
	InitialBackoff time.Duration
	// MaxBackoff caps the exponential backoff between restarts.
	MaxBackoff time.Duration
	Logger     *slog.Logger
}

// Supervisor monitors one game process.
type Supervisor struct {
	cfg Config

	mu       sync.Mutex
	restarts int
	stopped  bool
	done     chan struct{}
	stopOnce sync.Once
}

// New creates a supervisor. It does not start the process.
func New(cfg Config) *Supervisor {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.InitialBackoff <= 0 {
		cfg.InitialBackoff = time.Second
	}
	if cfg.MaxBackoff <= 0 {
		cfg.MaxBackoff = 30 * time.Second
	}
	if cfg.MaxRestarts < 0 {
		cfg.MaxRestarts = 0
	}
	return &Supervisor{cfg: cfg, done: make(chan struct{})}
}

// Run starts the game and restarts it after unexpected exits until ctx is
// cancelled, Stop is called, or the restart budget is exhausted.

// appendEnv appends extra environment variables, replacing any existing value
// for the same key so overrides such as PD_GAME_TOKEN take effect.
func appendEnv(base, extra []string) []string {
	out := make([]string, 0, len(base)+len(extra))
	for _, item := range base {
		key, _, ok := splitEnv(item)
		if !ok {
			out = append(out, item)
			continue
		}
		if hasEnvKey(extra, key) {
			continue
		}
		out = append(out, item)
	}
	out = append(out, extra...)
	return out
}

func splitEnv(item string) (string, string, bool) {
	for i := 0; i < len(item); i++ {
		if item[i] == '=' {
			return item[:i], item[i+1:], true
		}
	}
	return item, "", false
}

func hasEnvKey(items []string, key string) bool {
	for _, item := range items {
		if k, _, ok := splitEnv(item); ok && k == key {
			return true
		}
	}
	return false
}

func (s *Supervisor) Run(ctx context.Context) error {
	if s.cfg.Binary == "" {
		return nil
	}
	resolved, err := exec.LookPath(s.cfg.Binary)
	if err != nil {
		return fmt.Errorf("game binary not found: %w", err)
	}
	s.cfg.Binary = resolved

	backoff := s.cfg.InitialBackoff
	for {
		s.mu.Lock()
		if s.stopped {
			s.mu.Unlock()
			return nil
		}
		s.mu.Unlock()

		select {
		case <-ctx.Done():
			return nil
		default:
		}

		s.cfg.Logger.Info("starting game process", "binary", s.cfg.Binary, "args", s.cfg.Args)
		cmd := exec.CommandContext(ctx, s.cfg.Binary, s.cfg.Args...)
		cmd.Env = appendEnv(os.Environ(), s.cfg.Env)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if ctx.Err() != nil {
			return nil
		}

		s.mu.Lock()
		if s.stopped {
			s.mu.Unlock()
			return nil
		}
		s.restarts++
		restarts := s.restarts
		s.mu.Unlock()

		s.cfg.Logger.Warn("game process exited", "error", err, "restart", restarts)

		if restarts > s.cfg.MaxRestarts {
			return fmt.Errorf("game crash loop: %d restarts exceeded", s.cfg.MaxRestarts)
		}

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(backoff):
		}

		backoff *= 2
		if backoff > s.cfg.MaxBackoff {
			backoff = s.cfg.MaxBackoff
		}
	}
}

// Stop prevents further restarts. It does not kill the current process.
func (s *Supervisor) Stop() {
	s.mu.Lock()
	s.stopped = true
	s.mu.Unlock()
	s.stopOnce.Do(func() {
		close(s.done)
	})
}

// Restarts returns the number of unexpected exits handled so far.
func (s *Supervisor) Restarts() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.restarts
}
