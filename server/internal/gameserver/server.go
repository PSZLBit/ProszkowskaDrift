// Package gameserver implements the WebSocket session boundary between
// controllers, the game, and the in-memory session state.
package gameserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"

	"github.com/PSZLBit/ProszkowskaDrift/server/internal/protocol"
	"github.com/PSZLBit/ProszkowskaDrift/server/internal/session"
)

const wsWriteTimeout = 2 * time.Second

var errNonTextFrame = errors.New("non-text websocket frame")

// Config configures the WebSocket server. Zero values are replaced by
// production defaults.
type Config struct {
	Logger            *slog.Logger
	GameToken         string
	ControllerTimeout time.Duration
	InvalidThreshold  int
	RateThreshold     int
	RateLimit         int
	RateWindow        time.Duration
	RoundDurationMs   int
	TasksPerPlayer    int
}

// DefaultConfig returns the Sprint 0 server defaults.
func DefaultConfig() Config {
	return Config{
		Logger:            slog.Default(),
		ControllerTimeout: 6 * time.Second,
		InvalidThreshold:  10,
		RateThreshold:     20,
		RateLimit:         60,
		RateWindow:        time.Second,
		RoundDurationMs:   180000,
		TasksPerPlayer:    5,
	}
}

func (c Config) withDefaults() Config {
	defaults := DefaultConfig()
	if c.Logger == nil {
		c.Logger = defaults.Logger
	}
	if c.ControllerTimeout <= 0 {
		c.ControllerTimeout = defaults.ControllerTimeout
	}
	if c.InvalidThreshold <= 0 {
		c.InvalidThreshold = defaults.InvalidThreshold
	}
	if c.RateThreshold <= 0 {
		c.RateThreshold = defaults.RateThreshold
	}
	if c.RateLimit <= 0 {
		c.RateLimit = defaults.RateLimit
	}
	if c.RateWindow <= 0 {
		c.RateWindow = defaults.RateWindow
	}
	if c.RoundDurationMs <= 0 {
		c.RoundDurationMs = defaults.RoundDurationMs
	}
	if c.TasksPerPlayer <= 0 {
		c.TasksPerPlayer = defaults.TasksPerPlayer
	}
	return c
}

// Server owns active connections and translates between the wire protocol and
// the session.
type Server struct {
	cfg     Config
	log     *slog.Logger
	session *session.Session

	mu          sync.Mutex
	controllers map[int]*controllerConn
	game        *gameConn

	nextConnID atomic.Uint64
}

// New creates a server with an empty session.
func New(cfg Config) *Server {
	cfg = cfg.withDefaults()
	s := &Server{
		cfg:         cfg,
		log:         cfg.Logger,
		session:     session.New(session.Options{Clock: time.Now, RateLimit: cfg.RateLimit, RateWindow: cfg.RateWindow}),
		controllers: make(map[int]*controllerConn),
	}
	return s
}

// Handler returns the HTTP handler with the controller and game endpoints.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws/controller", s.handleController)
	mux.HandleFunc("/ws/game", s.handleGame)
	return mux
}

// wsConn wraps a WebSocket with serialized writes.
type wsConn struct {
	ws *websocket.Conn
	mu sync.Mutex
}

func newWSConn(ws *websocket.Conn) *wsConn {
	return &wsConn{ws: ws}
}

func (c *wsConn) write(raw []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), wsWriteTimeout)
	defer cancel()
	return c.ws.Write(ctx, websocket.MessageText, raw)
}

func (c *wsConn) send(typ string, data any) error {
	raw, err := protocol.Encode(typ, nil, data)
	if err != nil {
		return err
	}
	return c.write(raw)
}

func (c *wsConn) sendError(code, message string) {
	_ = c.send(protocol.TypeError, protocol.ErrorData{Code: code, Message: message})
}

func (c *wsConn) close(code websocket.StatusCode, reason string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.ws.Close(code, reason)
}

func readEnvelope(ctx context.Context, ws *websocket.Conn, timeout time.Duration) (protocol.Envelope, error) {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	typ, raw, err := ws.Read(ctx)
	if err != nil {
		return protocol.Envelope{}, err
	}
	if typ != websocket.MessageText {
		return protocol.Envelope{}, errNonTextFrame
	}
	return protocol.ParseEnvelope(raw)
}

func (s *Server) registerController(conn *controllerConn) {
	s.mu.Lock()
	s.controllers[conn.slot] = conn
	s.mu.Unlock()
}

func (s *Server) removeController(slot int, conn *controllerConn) {
	s.mu.Lock()
	if s.controllers[slot] == conn {
		delete(s.controllers, slot)
	}
	s.mu.Unlock()
}

func (s *Server) controllerConnections() []*controllerConn {
	s.mu.Lock()
	defer s.mu.Unlock()

	conns := make([]*controllerConn, 0, len(s.controllers))
	for _, conn := range s.controllers {
		conns = append(conns, conn)
	}
	return conns
}

func (s *Server) broadcast(typ string, data any) {
	raw, err := protocol.Encode(typ, nil, data)
	if err != nil {
		s.log.Error("encode broadcast", "type", typ, "error", err)
		return
	}
	for _, conn := range s.controllerConnections() {
		if err := conn.write(raw); err != nil {
			conn.log.Debug("broadcast write failed", "type", typ, "error", err)
		}
	}
}

func (s *Server) broadcastPhase(data protocol.PhaseData) {
	s.broadcast(protocol.TypePhase, data)
}

func (s *Server) sendToGame(typ string, data any) bool {
	s.mu.Lock()
	game := s.game
	s.mu.Unlock()
	if game == nil {
		return false
	}
	raw, err := protocol.Encode(typ, nil, data)
	if err != nil {
		s.log.Error("encode game message", "type", typ, "error", err)
		return false
	}
	if err := game.write(raw); err != nil {
		game.log.Debug("game write failed", "type", typ, "error", err)
		return false
	}
	return true
}

func (s *Server) sendPlayerJoined(slot session.SlotView) {
	if !s.sendToGame(protocol.TypePlayerJoined, protocol.PlayerJoinedData{
		Slot:  slot.Number,
		Label: slot.Label,
		Color: slot.Color,
	}) {
		s.log.Debug("player_joined not sent; game unavailable", "slot", slot.Number)
	}
}

func (s *Server) sendPlayerLeft(slot int, reason string) {
	if !s.sendToGame(protocol.TypePlayerLeft, protocol.PlayerLeftData{Slot: slot, Reason: reason}) {
		s.log.Debug("player_left not sent; game unavailable", "slot", slot, "reason", reason)
	}
}

func (s *Server) dropController(conn *controllerConn) {
	if conn.slot == 0 {
		return
	}

	view, changed := s.session.Disconnect(conn.slot, conn.id)
	s.removeController(conn.slot, conn)
	if changed {
		s.sendPlayerLeft(view.Number, "disconnect")
		conn.log.Info("controller disconnected", "slot", conn.slot, "token", conn.tokenRef)
	}
}

func (s *Server) kickController(slot int) {
	s.mu.Lock()
	conn := s.controllers[slot]
	delete(s.controllers, slot)
	s.mu.Unlock()

	if conn == nil {
		return
	}

	conn.sendError(protocol.CodeKicked, "Slot was kicked by the host.")
	conn.log.Info("controller kicked", "slot", slot)
	// Notify the game before the close handshake. Close waits for the
	// controller to acknowledge, and the game should not wait on that.
	s.sendPlayerLeft(slot, "kicked")
	conn.close(websocket.StatusPolicyViolation, "kicked")
}

func (s *Server) registerGame(conn *gameConn) {
	s.mu.Lock()
	old := s.game
	s.game = conn
	s.mu.Unlock()

	if old != nil {
		old.log.Info("game connection replaced")
		old.close(websocket.StatusGoingAway, "replaced")
	}
}

func (s *Server) dropGame(conn *gameConn) {
	s.mu.Lock()
	if s.game == conn {
		s.game = nil
	}
	s.mu.Unlock()
}

func (s *Server) sendExistingPlayers(conn *gameConn) error {
	for _, slot := range s.session.Slots() {
		if !slot.Connected {
			continue
		}
		if err := conn.send(protocol.TypePlayerJoined, protocol.PlayerJoinedData{
			Slot:  slot.Number,
			Label: slot.Label,
			Color: slot.Color,
		}); err != nil {
			return err
		}
	}
	return nil
}

func isLoopbackRemoteAddr(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func truncateToken(token string) string {
	if len(token) < 8 {
		return "***"
	}
	return fmt.Sprintf("%s...%s", token[:4], token[len(token)-2:])
}
