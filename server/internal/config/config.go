// Package config loads and validates the server configuration.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

// Slot describes one fixed player slot.
type Slot struct {
	ID    int    `json:"id"`
	Label string `json:"label"`
	Color string `json:"color"`
}

// Config is the complete server configuration. Defaults are applied before
// a config file is decoded.
type Config struct {
	Listen                  string   `json:"listen"`
	GamePath                string   `json:"gamePath"`
	GameArgs                []string `json:"gameArgs"`
	GameWorkingDir          string   `json:"gameWorkingDir"`
	GameToken               string   `json:"gameToken,omitempty"`
	LogPath                 string   `json:"logPath"`
	LogLevel                string   `json:"logLevel"`
	RoundDurationMs         int64    `json:"roundDurationMs"`
	TasksPerPlayer          int      `json:"tasksPerPlayer"`
	MaxPlayers              int      `json:"maxPlayers"`
	Slots                   []Slot   `json:"slots"`
	ControllerIdleTimeoutMs int64    `json:"controllerIdleTimeoutMs"`
	AllDisconnectedGraceMs  int64    `json:"allDisconnectedGraceMs"`
	GameHeartbeatTimeoutMs  int64    `json:"gameHeartbeatTimeoutMs"`
	HandshakeTimeoutMs      int64    `json:"handshakeTimeoutMs"`
	WriteTimeoutMs          int64    `json:"writeTimeoutMs"`
	MaxMessageBytes         int64    `json:"maxMessageBytes"`
	InvalidErrorLimit       int      `json:"invalidErrorLimit"`
	RateErrorLimit          int      `json:"rateErrorLimit"`
	ErrorWindowMs           int64    `json:"errorWindowMs"`
	InputRatePerSecond      float64  `json:"inputRatePerSecond"`
	InputBurst              float64  `json:"inputBurst"`
	DevNoGame               bool     `json:"devNoGame"`
	HealthPath              string   `json:"healthPath"`
}

// Default returns the server defaults. Slots use stable P1-P4 labels and
// distinct colours; the colour names are also sent to the game and controller.
func Default() Config {
	return Config{
		Listen:                  "0.0.0.0:8080",
		LogPath:                 "",
		LogLevel:                "info",
		RoundDurationMs:         180000,
		TasksPerPlayer:          5,
		MaxPlayers:              4,
		ControllerIdleTimeoutMs: 6000,
		AllDisconnectedGraceMs:  10000,
		GameHeartbeatTimeoutMs:  3000,
		HandshakeTimeoutMs:      5000,
		WriteTimeoutMs:          2000,
		MaxMessageBytes:         4096,
		InvalidErrorLimit:       10,
		RateErrorLimit:          3,
		ErrorWindowMs:           10000,
		InputRatePerSecond:      60,
		InputBurst:              60,
		HealthPath:              "/healthz",
		Slots: []Slot{
			{ID: 1, Label: "P1", Color: "blue"},
			{ID: 2, Label: "P2", Color: "red"},
			{ID: 3, Label: "P3", Color: "green"},
			{ID: 4, Label: "P4", Color: "yellow"},
		},
	}
}

// Load reads a JSON config file over the defaults. An empty path returns the
// defaults. A non-empty path must exist.
func Load(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		return cfg, cfg.Validate()
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Validate checks ranges and cross-field constraints.
func (c *Config) Validate() error {
	if strings.TrimSpace(c.Listen) == "" {
		return errors.New("listen address is required")
	}
	if _, _, err := net.SplitHostPort(c.Listen); err != nil {
		return fmt.Errorf("listen address must be host:port: %w", err)
	}
	if c.RoundDurationMs <= 0 {
		return errors.New("roundDurationMs must be positive")
	}
	if c.TasksPerPlayer <= 0 {
		return errors.New("tasksPerPlayer must be positive")
	}
	if c.MaxPlayers < 1 || c.MaxPlayers > 4 {
		return errors.New("maxPlayers must be between 1 and 4")
	}
	if len(c.Slots) != c.MaxPlayers {
		return fmt.Errorf("slots must contain exactly %d entries", c.MaxPlayers)
	}
	seenID := map[int]bool{}
	seenLabel := map[string]bool{}
	seenColor := map[string]bool{}
	for i, slot := range c.Slots {
		if slot.ID != i+1 {
			return fmt.Errorf("slots must be ordered by id: expected id %d, got %d", i+1, slot.ID)
		}
		if slot.ID < 1 || slot.ID > c.MaxPlayers {
			return fmt.Errorf("slot id %d out of range", slot.ID)
		}
		if seenID[slot.ID] {
			return fmt.Errorf("duplicate slot id %d", slot.ID)
		}
		seenID[slot.ID] = true
		if strings.TrimSpace(slot.Label) == "" || strings.TrimSpace(slot.Color) == "" {
			return fmt.Errorf("slot %d needs a label and color", slot.ID)
		}
		if seenLabel[slot.Label] {
			return fmt.Errorf("duplicate slot label %q", slot.Label)
		}
		seenLabel[slot.Label] = true
		if seenColor[slot.Color] {
			return fmt.Errorf("duplicate slot color %q", slot.Color)
		}
		seenColor[slot.Color] = true
	}
	if c.ControllerIdleTimeoutMs <= 0 || c.AllDisconnectedGraceMs <= 0 ||
		c.GameHeartbeatTimeoutMs <= 0 || c.HandshakeTimeoutMs <= 0 ||
		c.WriteTimeoutMs <= 0 || c.ErrorWindowMs <= 0 {
		return errors.New("timeouts and error window must be positive")
	}
	if c.MaxMessageBytes < 256 {
		return errors.New("maxMessageBytes must be at least 256")
	}
	if c.InvalidErrorLimit <= 0 || c.RateErrorLimit <= 0 {
		return errors.New("error limits must be positive")
	}
	if c.InputRatePerSecond <= 0 || c.InputBurst < 1 {
		return errors.New("input rate and burst must be positive")
	}
	if c.LogLevel != "" {
		switch strings.ToLower(c.LogLevel) {
		case "debug", "info", "warn", "error":
		default:
			return fmt.Errorf("unknown log level %q", c.LogLevel)
		}
	}
	if !strings.HasPrefix(c.HealthPath, "/") {
		return errors.New("healthPath must start with /")
	}
	if c.HealthPath == "/ws/controller" || c.HealthPath == "/ws/game" {
		return errors.New("healthPath conflicts with a WebSocket endpoint")
	}
	return nil
}

// ControllerIdleTimeout is the controller read/heartbeat timeout.
func (c Config) ControllerIdleTimeout() time.Duration {
	return time.Duration(c.ControllerIdleTimeoutMs) * time.Millisecond
}

// AllDisconnectedGrace is the delay before an all-disconnected round pauses.
func (c Config) AllDisconnectedGrace() time.Duration {
	return time.Duration(c.AllDisconnectedGraceMs) * time.Millisecond
}

// GameHeartbeatTimeout is the missing-heartbeat restart threshold.
func (c Config) GameHeartbeatTimeout() time.Duration {
	return time.Duration(c.GameHeartbeatTimeoutMs) * time.Millisecond
}

// HandshakeTimeout bounds hello/game_ready exchanges.
func (c Config) HandshakeTimeout() time.Duration {
	return time.Duration(c.HandshakeTimeoutMs) * time.Millisecond
}

// WriteTimeout bounds a single WebSocket write.
func (c Config) WriteTimeout() time.Duration {
	return time.Duration(c.WriteTimeoutMs) * time.Millisecond
}

// ErrorWindow is the rolling window for abuse counters.
func (c Config) ErrorWindow() time.Duration {
	return time.Duration(c.ErrorWindowMs) * time.Millisecond
}

// HasGameProcess reports whether the server should supervise a game process.
func (c Config) HasGameProcess() bool {
	return !c.DevNoGame && strings.TrimSpace(c.GamePath) != ""
}
