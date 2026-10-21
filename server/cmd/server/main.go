// Command server runs the Proszkowska Drift session server.
package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/PSZLBit/ProszkowskaDrift/server/internal/config"
	"github.com/PSZLBit/ProszkowskaDrift/server/internal/logging"
	"github.com/PSZLBit/ProszkowskaDrift/server/internal/server"
	"github.com/PSZLBit/ProszkowskaDrift/server/internal/supervisor"
)

var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	var (
		configPath = flag.String("config", "", "path to JSON config file")
		listen     = flag.String("listen", "", "listen address, e.g. 0.0.0.0:8080")
		gamePath   = flag.String("game", "", "Godot executable path (enables supervision)")
		noGame     = flag.Bool("no-game", false, "do not launch the game process")
		logPath    = flag.String("log", "", "log file path")
		gameToken  = flag.String("game-token", "", "static game token for development mode")
		show       = flag.Bool("version", false, "print version and exit")
	)
	flag.Parse()
	if *show {
		fmt.Printf("proszkowska-server %s (%s)\n", version, commit)
		return
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(1)
	}
	if *listen != "" {
		cfg.Listen = *listen
	}
	if *gamePath != "" {
		cfg.GamePath = *gamePath
	}
	if *noGame {
		cfg.DevNoGame = true
	}
	if *logPath != "" {
		cfg.LogPath = *logPath
	}
	if *gameToken != "" {
		cfg.GameToken = *gameToken
	}
	if err := cfg.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(1)
	}

	logger, closer, err := logging.New(cfg.LogPath, cfg.LogLevel)
	if err != nil {
		fmt.Fprintln(os.Stderr, "logging:", err)
		os.Exit(1)
	}
	defer closer.Close()
	logger = logger.With("component", "server", "version", version, "commit", commit)

	var sup *supervisor.Supervisor
	if cfg.HasGameProcess() {
		sup = supervisor.New(supervisor.Spec{
			Path:    cfg.GamePath,
			Args:    cfg.GameArgs,
			Dir:     cfg.GameWorkingDir,
			GameURL: gameURL(cfg.Listen),
		}, logger)
		logger.Info("game supervision enabled", "path", cfg.GamePath)
	} else if cfg.GameToken == "" {
		logger.Warn("no game process and no game token; /ws/game will reject connections")
	}

	srv := server.New(cfg, logger, sup, server.Options{Version: version})
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := srv.Run(ctx); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
	logger.Info("server stopped")
}

func gameURL(listen string) string {
	_, port, err := net.SplitHostPort(listen)
	if err != nil || port == "" {
		port = "8080"
	}
	// The game endpoint is loopback-only, so the supervised game always
	// connects to localhost even when the controller listener binds a LAN
	// address.
	return "ws://127.0.0.1:" + port + "/ws/game"
}
