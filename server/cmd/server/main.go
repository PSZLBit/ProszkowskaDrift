package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/PSZLBit/ProszkowskaDrift/server/internal/gameserver"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "server:", err)
		os.Exit(1)
	}
}

func run() error {
	addr := flag.String("addr", ":8080", "HTTP/WebSocket listen address")
	gameToken := flag.String("game-token", os.Getenv("PD_GAME_TOKEN"), "game endpoint token (default: PD_GAME_TOKEN, or a random token)")
	controllerTimeout := flag.Duration("controller-timeout", 6*time.Second, "close a silent controller connection after this duration")
	allDisconnectedGrace := flag.Duration("all-disconnected-grace", 10*time.Second, "pause the round after every controller was disconnected for this duration")
	gameHeartbeatTimeout := flag.Duration("game-heartbeat-timeout", 3*time.Second, "close the game connection when no game message arrives for this duration")
	logLevel := flag.String("log-level", "info", "log level: debug, info, warn, error")
	flag.Parse()

	level, err := parseLogLevel(*logLevel)
	if err != nil {
		return err
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))

	token := *gameToken
	generated := false
	if token == "" {
		token, err = newGameToken()
		if err != nil {
			return fmt.Errorf("generate game token: %w", err)
		}
		generated = true
	}

	cfg := gameserver.DefaultConfig()
	cfg.Logger = logger
	cfg.GameToken = token
	cfg.ControllerTimeout = *controllerTimeout
	cfg.AllDisconnectedGrace = *allDisconnectedGrace
	cfg.GameHeartbeatTimeout = *gameHeartbeatTimeout
	srv := gameserver.New(cfg)

	listener, err := net.Listen("tcp", *addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", *addr, err)
	}

	if generated {
		logger.Info("generated per-launch game token", "game_token", token)
	}

	port := listener.Addr().(*net.TCPAddr).Port
	logger.Info("server listening",
		"addr", listener.Addr().String(),
		"controller_endpoint", fmt.Sprintf("ws://<host>:%d/ws/controller", port),
		"game_endpoint", fmt.Sprintf("ws://127.0.0.1:%d/ws/game", port),
		"controller_timeout", cfg.ControllerTimeout,
	)

	httpServer := &http.Server{
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- httpServer.Serve(listener)
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// The tick loop enforces the controller idle timeout, the
	// all-disconnected pause, and the game heartbeat timeout.
	go srv.StartTick(ctx)

	select {
	case <-ctx.Done():
		logger.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}
		return nil
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func parseLogLevel(value string) (slog.Level, error) {
	switch value {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("unknown log level %q", value)
	}
}

func newGameToken() (string, error) {
	raw := make([]byte, 18)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
