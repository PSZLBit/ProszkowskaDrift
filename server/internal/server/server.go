// Package server wires the protocol, session, WebSocket endpoints, and game
// link into the running Go process.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"

	"github.com/PSZLBit/ProszkowskaDrift/server/internal/config"
	"github.com/PSZLBit/ProszkowskaDrift/server/internal/protocol"
	"github.com/PSZLBit/ProszkowskaDrift/server/internal/session"
	"github.com/PSZLBit/ProszkowskaDrift/server/internal/supervisor"
)

// Options injects test doubles.
type Options struct {
	Now     func() time.Time
	Version string
}

// Server is the running game server.
type Server struct {
	cfg     config.Config
	log     *slog.Logger
	session *session.Session
	hub     *hub
	game    *gameLink
	sup     *supervisor.Supervisor
	opts    Options

	upgrader   websocket.Upgrader
	httpServer *http.Server
	gameMu     sync.Mutex
	stopping   atomic.Bool
	shutdown   sync.Once
}

// New builds a server. The supervisor may be nil in development mode.
func New(cfg config.Config, log *slog.Logger, sup *supervisor.Supervisor, opts Options) *Server {
	if log == nil {
		log = slog.Default()
	}
	if cfg.HealthPath == "" {
		cfg.HealthPath = "/healthz"
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.Version == "" {
		opts.Version = "dev"
	}
	s := &Server{
		cfg:     cfg,
		log:     log,
		session: session.New(sessionOptions(cfg)),
		hub:     newHub(),
		game:    newGameLink(),
		sup:     sup,
		opts:    opts,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(*http.Request) bool { return true },
		},
	}
	return s
}

// Handler exposes the HTTP routes for tests.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws/controller", s.handleController)
	mux.HandleFunc("/ws/game", s.handleGame)
	mux.HandleFunc(s.cfg.HealthPath, s.handleHealth)
	return mux
}

// Run starts the HTTP listener and blocks until ctx is cancelled or serving
// fails.
func (s *Server) Run(ctx context.Context) error {
	listener, err := net.Listen("tcp", s.cfg.Listen)
	if err != nil {
		return err
	}
	s.httpServer = &http.Server{
		Handler:           s.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.Shutdown(shutdownCtx)
	}()
	s.log.Info("server listening",
		"address", listener.Addr().String(),
		"buildVersion", s.opts.Version,
		"health", s.cfg.HealthPath,
	)
	go s.tickLoop(ctx)
	if s.sup != nil {
		go s.sup.Run(ctx)
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.Shutdown(shutdownCtx)
	}()
	err = s.httpServer.Serve(listener)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// Shutdown stops accepting connections and closes active WebSockets.
func (s *Server) Shutdown(ctx context.Context) error {
	var err error
	s.shutdown.Do(func() {
		s.stopping.Store(true)
		s.log.Info("server shutting down")
		s.hub.closeAll(websocket.CloseGoingAway, "server shutdown")
		s.game.close()
		if s.sup != nil {
			s.sup.Stop()
		}
		if s.httpServer != nil {
			err = s.httpServer.Shutdown(ctx)
		}
		if s.sup != nil {
			select {
			case <-s.sup.Done():
			case <-ctx.Done():
				s.log.Warn("timed out waiting for game supervisor to stop")
			}
		}
	})
	return err
}

func (s *Server) tickLoop(ctx context.Context) {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			s.tick(now)
		}
	}
}

func (s *Server) tick(now time.Time) {
	for _, action := range s.session.Tick(now) {
		switch action.Kind {
		case session.ActionControllerTimeout:
			s.log.Info("controller heartbeat timeout", "slot", int(action.Slot))
			s.hub.closeSlotIfGeneration(action.Slot, action.Generation, websocket.CloseGoingAway, "heartbeat timeout")
			snap, ok := s.session.SnapshotForSlot(action.Slot)
			if !ok || snap.Connected || snap.Generation != action.Generation {
				continue
			}
			s.game.playerLeft(action.Slot, protocol.LeftTimeout)
		case session.ActionPauseAllDisconnected:
			s.log.Info("all controllers disconnected; pausing round")
			s.game.hostCommand(protocol.HostPause, nil)
			s.broadcastPhase()
		}
	}
	if s.game.heartbeatExpired(now, s.cfg.GameHeartbeatTimeout()) {
		s.log.Warn("game heartbeat timeout")
		if conn := s.game.current(); conn != nil {
			conn.closeNow(websocket.CloseGoingAway, "heartbeat timeout")
		}
	}
}

func (s *Server) now() time.Time {
	return s.opts.Now()
}

func (s *Server) gameToken() string {
	if s.sup != nil {
		return s.sup.Token()
	}
	return s.cfg.GameToken
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	state := s.session.State()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":      "ok",
		"version":     s.opts.Version,
		"phase":       state.Phase,
		"roundId":     state.RoundID,
		"gameReady":   s.game.isReady(),
		"controllers": s.hub.count(),
	})
}

func sessionOptions(cfg config.Config) session.Options {
	defs := make([]session.SlotDef, 0, len(cfg.Slots))
	for _, slot := range cfg.Slots {
		defs = append(defs, session.SlotDef{
			ID:    session.SlotID(slot.ID),
			Label: slot.Label,
			Color: slot.Color,
		})
	}
	return session.Options{
		Slots:    defs,
		MaxSlots: cfg.MaxPlayers,
		Timings: session.Timings{
			ControllerIdleTimeout: cfg.ControllerIdleTimeout(),
			AllDisconnectedGrace:  cfg.AllDisconnectedGrace(),
			InvalidErrorLimit:     cfg.InvalidErrorLimit,
			RateErrorLimit:        cfg.RateErrorLimit,
			ErrorWindow:           cfg.ErrorWindow(),
			InputRatePerSecond:    cfg.InputRatePerSecond,
			InputBurst:            cfg.InputBurst,
		},
	}
}
