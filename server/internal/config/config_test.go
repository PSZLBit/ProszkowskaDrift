package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultIsValid(t *testing.T) {
	cfg := Default()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("default invalid: %v", err)
	}
	if cfg.MaxPlayers != 4 || len(cfg.Slots) != 4 {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	if cfg.HasGameProcess() {
		t.Fatal("supervision should be off without a path")
	}
}

func TestExampleConfigLoads(t *testing.T) {
	cfg, err := Load(filepath.Join("..", "..", "config.example.json"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MaxPlayers != 4 || len(cfg.Slots) != 4 {
		t.Fatalf("example config = %+v", cfg)
	}
	if cfg.HealthPath != "/healthz" {
		t.Fatalf("health path = %q", cfg.HealthPath)
	}
}

func TestLoadOverridesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	body := `{
		"listen": "127.0.0.1:9090",
		"roundDurationMs": 120000,
		"gamePath": "godot.exe",
		"slots": [
			{"id":1,"label":"P1","color":"blue"},
			{"id":2,"label":"P2","color":"red"},
			{"id":3,"label":"P3","color":"green"},
			{"id":4,"label":"P4","color":"yellow"}
		]
	}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Listen != "127.0.0.1:9090" || cfg.RoundDurationMs != 120000 {
		t.Fatalf("cfg = %+v", cfg)
	}
	if cfg.TasksPerPlayer != 5 {
		t.Fatalf("defaults not retained: %+v", cfg)
	}
	if !cfg.HasGameProcess() {
		t.Fatal("game path did not enable supervision")
	}
}

func TestLoadRejectsUnknownFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"unknown":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("unknown field accepted")
	}
}

func TestValidateRejectsBadValues(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Config)
	}{
		{name: "empty listen", edit: func(c *Config) { c.Listen = "" }},
		{name: "bad listen", edit: func(c *Config) { c.Listen = "8080" }},
		{name: "health endpoint conflict", edit: func(c *Config) { c.HealthPath = "/ws/game" }},
		{name: "zero round", edit: func(c *Config) { c.RoundDurationMs = 0 }},
		{name: "too many players", edit: func(c *Config) { c.MaxPlayers = 5 }},
		{name: "slot count", edit: func(c *Config) { c.Slots = c.Slots[:3] }},
		{name: "slot order", edit: func(c *Config) { c.Slots[0], c.Slots[1] = c.Slots[1], c.Slots[0] }},
		{name: "duplicate color", edit: func(c *Config) { c.Slots[1].Color = c.Slots[0].Color }},
		{name: "health path", edit: func(c *Config) { c.HealthPath = "healthz" }},
		{name: "zero timeout", edit: func(c *Config) { c.ControllerIdleTimeoutMs = 0 }},
		{name: "bad log level", edit: func(c *Config) { c.LogLevel = "verbose" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			tt.edit(&cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatal("invalid config accepted")
			}
		})
	}
}
