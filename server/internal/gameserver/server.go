// Package gameserver implements the WebSocket session boundary between
// controllers, the game, and the in-memory session state.
package gameserver

import (
	"context"
	"encoding/json"
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

// defaultTickInterval drives server-side timers such as the controller idle
// timeout, the all-disconnected pause, and the game heartbeat.
const defaultTickInterval = 250 * time.Millisecond

var errNonTextFrame = errors.New("non-text websocket frame")

// Config configures the WebSocket server. Zero values are replaced by
// production defaults.
type Config struct {
	Logger               *slog.Logger
	GameToken            string
	ControllerTimeout    time.Duration
	AllDisconnectedGrace time.Duration
	GameHeartbeatTimeout time.Duration
	TickInterval         time.Duration
	InvalidThreshold     int
	RateThreshold        int
	RateLimit            int
	RateWindow           time.Duration
	RoundDurationMs      int
	TasksPerPlayer       int
	ControllerQueueSize  int
	GameQueueSize        int
}

// DefaultConfig returns the Sprint 0 server defaults.
func DefaultConfig() Config {
	return Config{
		Logger:               slog.Default(),
		ControllerTimeout:    6 * time.Second,
		AllDisconnectedGrace: 10 * time.Second,
		GameHeartbeatTimeout: 3 * time.Second,
		TickInterval:         defaultTickInterval,
		InvalidThreshold:     10,
		RateThreshold:        20,
		RateLimit:            60,
		RateWindow:           time.Second,
		RoundDurationMs:      180000,
		TasksPerPlayer:       5,
		ControllerQueueSize:  64,
		GameQueueSize:        256,
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
	if c.AllDisconnectedGrace <= 0 {
		c.AllDisconnectedGrace = defaults.AllDisconnectedGrace
	}
	if c.GameHeartbeatTimeout <= 0 {
		c.GameHeartbeatTimeout = defaults.GameHeartbeatTimeout
	}
	if c.TickInterval <= 0 {
		c.TickInterval = defaults.TickInterval
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
	if c.ControllerQueueSize <= 0 {
		c.ControllerQueueSize = defaults.ControllerQueueSize
	}
	if c.GameQueueSize <= 0 {
		c.GameQueueSize = defaults.GameQueueSize
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

	// gameReady and lastGameMessage track the game handshake and liveness so
	// the tick loop can close a silent game connection without racing the
	// handshake.
	gameReady       bool
	lastGameMessage time.Time

	nextConnID atomic.Uint64
}

// New creates a server with an empty session.
func New(cfg Config) *Server {
	cfg = cfg.withDefaults()
	s := &Server{
		cfg: cfg,
		log: cfg.Logger,
		session: session.New(session.Options{
			Clock:                time.Now,
			RateLimit:            cfg.RateLimit,
			RateWindow:           cfg.RateWindow,
			IdleTimeout:          cfg.ControllerTimeout,
			AllDisconnectedGrace: cfg.AllDisconnectedGrace,
		}),
		controllers: make(map[int]*controllerConn),
	}
	return s
}

// Handler returns the HTTP handler with the controller, game, and health
// endpoints.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws/controller", s.handleController)
	mux.HandleFunc("/ws/game", s.handleGame)
	mux.HandleFunc("/healthz", s.handleHealth)
	return mux
}

// StartTick runs the timer loop until ctx is cancelled. The loop enforces the
// controller idle timeout, the all-disconnected pause, and the game heartbeat
// timeout.
func (s *Server) StartTick(ctx context.Context) {
	ticker := time.NewTicker(s.cfg.TickInterval)
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
			s.closeIdleController(action.Slot, action.ConnID)
		case session.ActionPauseAllDisconnected:
			s.log.Info("all controllers disconnected; pausing round")
			s.sendToGame(protocol.TypeHostCommand, protocol.HostCommandData{Command: string(session.CommandPause)})
			s.broadcastPhase(protocol.PhaseData{
				Phase:   s.session.Phase(),
				RoundID: s.session.RoundID(),
			})
		}
	}

	if s.gameHeartbeatExpired(now) {
		s.log.Warn("game connection closed: heartbeat timeout", "timeout", s.cfg.GameHeartbeatTimeout)
		s.mu.Lock()
		game := s.game
		s.mu.Unlock()
		if game != nil {
			game.enqueueClose(nil, websocket.StatusGoingAway, "game_heartbeat_timeout")
		}
	}
}

func (s *Server) gameHeartbeatExpired(now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.gameReady && s.game != nil &&
		now.Sub(s.lastGameMessage) > s.cfg.GameHeartbeatTimeout
}

func (s *Server) setGameMessageTime(now time.Time) {
	s.mu.Lock()
	s.lastGameMessage = now
	s.mu.Unlock()
}

func (s *Server) setGameReady(ready bool) {
	s.mu.Lock()
	s.gameReady = ready
	if ready {
		s.lastGameMessage = time.Now()
	}
	s.mu.Unlock()
}

func (s *Server) closeIdleController(slot int, connID uint64) {
	s.mu.Lock()
	conn := s.controllers[slot]
	s.mu.Unlock()
	if conn == nil || conn.id != connID {
		return
	}
	conn.log.Info("closing controller: heartbeat timeout", "slot", slot)
	s.sendToGame(protocol.TypePlayerLeft, protocol.PlayerLeftData{Slot: slot, Reason: "timeout"})
	conn.enqueueClose(nil, websocket.StatusPolicyViolation, "heartbeat_timeout")
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":      "ok",
		"phase":       string(s.session.Phase()),
		"roundId":     s.session.RoundID(),
		"controllers": len(s.controllerConnections()),
		"gameReady":   s.gameIsReady(),
	})
}

func (s *Server) gameIsReady() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.gameReady
}

// wsConn wraps a WebSocket with a serialized writer goroutine. Writes are
// queued, so broadcasts never block on a slow client; a client that stops
// draining its queue is closed.
type wsConn struct {
	ws    *websocket.Conn
	queue chan outbound
	done  chan struct{}

	closeOnce sync.Once
	closing   atomic.Bool
}

type outbound struct {
	data       []byte
	closeAfter bool
	code       websocket.StatusCode
	reason     string
}

func newWSConn(ws *websocket.Conn, queueSize int) *wsConn {
	c := &wsConn{
		ws:    ws,
		queue: make(chan outbound, queueSize),
		done:  make(chan struct{}),
	}
	go c.writeLoop()
	return c
}

// enqueue queues a text frame. It reports false when the connection is
// closing or its queue is full.
func (c *wsConn) enqueue(raw []byte) bool {
	if c.closing.Load() {
		return false
	}
	select {
	case c.queue <- outbound{data: raw}:
		return true
	case <-c.done:
		return false
	default:
		return false
	}
}

// enqueueClose queues a last optional frame, flushes earlier frames, and then
// performs the close handshake before returning. Call it with raw == nil to
// close after draining. When the queue is full the connection is closed
// immediately instead, because waiting would never drain it.
func (c *wsConn) enqueueClose(raw []byte, code websocket.StatusCode, reason string) bool {
	c.closing.Store(true)
	select {
	case c.queue <- outbound{data: raw, closeAfter: true, code: code, reason: reason}:
	case <-c.done:
		return false
	default:
		c.closeNow(code, reason)
		return false
	}
	<-c.done
	return true
}

func (c *wsConn) writeLoop() {
	for {
		select {
		case out := <-c.queue:
			if len(out.data) > 0 {
				if err := c.write(out.data); err != nil {
					c.closeNow(websocket.StatusInternalError, "write_failed")
					return
				}
			}
			if out.closeAfter {
				c.closeNow(out.code, out.reason)
				return
			}
		case <-c.done:
			return
		}
	}
}

// closeNow performs a synchronous close handshake. It is safe to call more
// than once.
func (c *wsConn) closeNow(code websocket.StatusCode, reason string) {
	c.closing.Store(true)
	c.closeOnce.Do(func() {
		_ = c.ws.Close(code, reason)
		close(c.done)
	})
}

func (c *wsConn) write(raw []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), wsWriteTimeout)
	defer cancel()
	return c.ws.Write(ctx, websocket.MessageText, raw)
}

// send serializes and queues a message. It reports false when the message
// could not be queued.
func (c *wsConn) send(typ string, data any) bool {
	raw, err := protocol.Encode(typ, nil, data)
	if err != nil {
		return false
	}
	return c.enqueue(raw)
}

// sendError queues an error message and reports whether it was queued.
func (c *wsConn) sendError(code, message string) bool {
	return c.send(protocol.TypeError, protocol.ErrorData{Code: code, Message: message})
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
	s.broadcastRaw(raw)
}

func (s *Server) broadcastRaw(raw []byte) {
	for _, conn := range s.controllerConnections() {
		if !conn.enqueue(raw) {
			conn.log.Debug("slow controller closed", "slot", conn.slot)
			conn.closeNow(websocket.StatusPolicyViolation, "slow_client")
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
	return game.send(typ, data)
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

// closeAllControllers performs graceful close handshakes, e.g. after a host
// end command, so controllers can distinguish a session end from a crash.
func (s *Server) closeAllControllers(reason string) {
	s.log.Info("closing controllers gracefully", "reason", reason)
	for _, conn := range s.controllerConnections() {
		conn.enqueueClose(nil, websocket.StatusNormalClosure, reason)
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
	conn.enqueueClose(nil, websocket.StatusPolicyViolation, "kicked")
}

func (s *Server) registerGame(conn *gameConn) {
	s.mu.Lock()
	old := s.game
	s.game = conn
	s.mu.Unlock()

	if old != nil {
		old.log.Info("game connection replaced")
		old.enqueueClose(nil, websocket.StatusGoingAway, "replaced")
	}
}

func (s *Server) dropGame(conn *gameConn) {
	s.mu.Lock()
	wasCurrent := s.game == conn
	if wasCurrent {
		s.game = nil
		s.gameReady = false
	}
	s.mu.Unlock()
	if wasCurrent {
		s.log.Info("game connection lost")
		s.onGameDisconnected()
	}
}

// onGameDisconnected returns the session to attract, tells connected
// controllers that the game is unavailable, and waits for the game (or a
// supervisor in a later sprint) to reconnect. Slots stay reserved.
func (s *Server) onGameDisconnected() {
	if s.session.Phase() == protocol.PhaseEnded {
		s.log.Info("game disconnected after session end")
		return
	}

	s.session.ResetForGame()
	s.broadcast(protocol.TypeError, protocol.ErrorData{
		Code:    protocol.CodeGameUnavailable,
		Message: "Game is unavailable. Waiting for restart.",
	})
	s.broadcastPhase(protocol.PhaseData{
		Phase:   s.session.Phase(),
		RoundID: s.session.RoundID(),
	})
}

func (s *Server) sendExistingPlayers(conn *gameConn) error {
	for _, slot := range s.session.Slots() {
		if !slot.Connected {
			continue
		}
		if !conn.send(protocol.TypePlayerJoined, protocol.PlayerJoinedData{
			Slot:  slot.Number,
			Label: slot.Label,
			Color: slot.Color,
		}) {
			return errors.New("player_joined queue failed")
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
