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
	"strings"
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
	// StableDuration is how long a game process must run before a later crash
	// is treated as a fresh failure rather than part of the same crash loop.
	// Zero defaults to 30s.
	StableDuration time.Duration
	Logger         *slog.Logger
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
	if cfg.StableDuration <= 0 {
		cfg.StableDuration = 30 * time.Second
	}
	return &Supervisor{cfg: cfg, done: make(chan struct{})}
}

// Run starts the game and restarts it after unexpected exits until ctx is
// cancelled, Stop is called, or the restart budget is exhausted.
func (s *Supervisor) Run(ctx context.Context) error {
	if s.cfg.Binary == "" {
		return nil
	}
	binary, err := exec.LookPath(s.cfg.Binary)
	if err != nil {
		return fmt.Errorf("game binary not found: %w", err)
	}

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
		case <-s.done:
			return nil
		default:
		}

		s.cfg.Logger.Info("starting game process", "binary", binary, "args", s.cfg.Args)
		started := time.Now()
		cmd := exec.CommandContext(ctx, binary, s.cfg.Args...)
		cmd.Env = appendEnv(os.Environ(), s.cfg.Env)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		runErr := cmd.Run()
		runtime := time.Since(started)

		if ctx.Err() != nil {
			return nil
		}

		s.mu.Lock()
		if s.stopped {
			s.mu.Unlock()
			return nil
		}
		if runtime >= s.cfg.StableDuration {
			s.restarts = 0
			backoff = s.cfg.InitialBackoff
		}
		s.restarts++
		restarts := s.restarts
		s.mu.Unlock()

		s.cfg.Logger.Warn("game process exited", "error", runErr, "restart", restarts, "runtime", runtime)

		if restarts > s.cfg.MaxRestarts {
			return fmt.Errorf("game crash loop: %d restarts exceeded", s.cfg.MaxRestarts)
		}

		select {
		case <-ctx.Done():
			return nil
		case <-s.done:
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

// appendEnv appends extra environment variables, replacing any existing value
// for the same key so overrides such as PD_GAME_TOKEN take effect. Duplicate
// keys in extra are collapsed, keeping the last value. Key comparison is
// case-insensitive.
func appendEnv(base, extra []string) []string {
	dedupedExtra := dedupEnv(extra)
	out := make([]string, 0, len(base)+len(dedupedExtra))
	for _, item := range base {
		key, _, ok := splitEnv(item)
		if !ok {
			out = append(out, item)
			continue
		}
		if hasEnvKey(dedupedExtra, key) {
			continue
		}
		out = append(out, item)
	}
	out = append(out, dedupedExtra...)
	return out
}

func dedupEnv(items []string) []string {
	values := make(map[string]string)
	order := make([]string, 0, len(items))
	for _, item := range items {
		key, _, ok := splitEnv(item)
		if !ok {
			outKey := item
			if _, exists := values[outKey]; !exists {
				order = append(order, outKey)
			}
			values[outKey] = item
			continue
		}
		upper := strings.ToUpper(key)
		if _, exists := values[upper]; !exists {
			order = append(order, upper)
		}
		values[upper] = item
	}
	out := make([]string, 0, len(order))
	for _, key := range order {
		out = append(out, values[key])
	}
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
	upper := strings.ToUpper(key)
	for _, item := range items {
		if k, _, ok := splitEnv(item); ok && strings.ToUpper(k) == upper {
			return true
		}
	}
	return false
}
