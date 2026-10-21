package server

import (
	"crypto/subtle"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/PSZLBit/ProszkowskaDrift/server/internal/protocol"
	"github.com/PSZLBit/ProszkowskaDrift/server/internal/session"
)

type gameConn struct {
	ws           *websocket.Conn
	writeTimeout time.Duration

	send      chan []byte
	done      chan struct{}
	closeOnce sync.Once
}

func newGameConn(ws *websocket.Conn, writeTimeout time.Duration) *gameConn {
	c := &gameConn{
		ws:           ws,
		writeTimeout: writeTimeout,
		send:         make(chan []byte, 256),
		done:         make(chan struct{}),
	}
	go c.writer()
	return c
}

func (c *gameConn) enqueue(data []byte) bool {
	select {
	case c.send <- data:
		return true
	case <-c.done:
		return false
	default:
		return false
	}
}

func (c *gameConn) enqueueCritical(data []byte) bool {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case c.send <- data:
		return true
	case <-c.done:
		return false
	case <-timer.C:
		c.closeNow(websocket.ClosePolicyViolation, "game is not reading")
		return false
	}
}

func (c *gameConn) writer() {
	for {
		select {
		case payload := <-c.send:
			_ = c.ws.SetWriteDeadline(time.Now().Add(c.writeTimeout))
			if err := c.ws.WriteMessage(websocket.TextMessage, payload); err != nil {
				c.closeNow(websocket.CloseInternalServerErr, "write failed")
				return
			}
		case <-c.done:
			return
		}
	}
}

func (c *gameConn) closeNow(code int, reason string) {
	c.closeOnce.Do(func() {
		deadline := time.Now().Add(c.writeTimeout)
		_ = c.ws.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(code, reason),
			deadline,
		)
		_ = c.ws.Close()
		close(c.done)
	})
}

type gameLink struct {
	mu            sync.Mutex
	conn          *gameConn
	ready         bool
	lastHeartbeat time.Time
}

func newGameLink() *gameLink {
	return &gameLink{}
}

func (g *gameLink) trySet(conn *gameConn) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.conn != nil {
		return false
	}
	g.conn = conn
	g.ready = false
	g.lastHeartbeat = time.Time{}
	return true
}

func (g *gameLink) clear(conn *gameConn) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.conn != conn {
		return false
	}
	g.conn = nil
	g.ready = false
	g.lastHeartbeat = time.Time{}
	return true
}

func (g *gameLink) current() *gameConn {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.conn
}

func (g *gameLink) markReady(now time.Time) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.ready = true
	g.lastHeartbeat = now
}

func (g *gameLink) touch(now time.Time) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.lastHeartbeat = now
}

func (g *gameLink) isReady() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.ready && g.conn != nil
}

func (g *gameLink) heartbeatExpired(now time.Time, timeout time.Duration) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.ready && g.conn != nil && now.Sub(g.lastHeartbeat) > timeout
}

func (g *gameLink) send(messageType string, data any) bool {
	conn, ready := g.snapshot()
	if !ready || conn == nil {
		return false
	}
	payload, err := protocol.EncodeMessage(protocol.Version, messageType, nil, data)
	if err != nil {
		return false
	}
	return conn.enqueueCritical(payload)
}

func (g *gameLink) sendInput(data protocol.GameInputData) bool {
	conn, ready := g.snapshot()
	if !ready || conn == nil {
		return false
	}
	payload, err := protocol.EncodeMessage(protocol.Version, protocol.TypeInput, nil, data)
	if err != nil {
		return false
	}
	return conn.enqueue(payload)
}

func (g *gameLink) snapshot() (*gameConn, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.conn, g.ready
}

func (g *gameLink) close() {
	if conn := g.current(); conn != nil {
		conn.closeNow(websocket.CloseGoingAway, "server shutdown")
	}
}

func (g *gameLink) config(data protocol.ConfigData) bool {
	return g.send(protocol.TypeConfig, data)
}

func (g *gameLink) playerJoined(slot session.SlotSnapshot) bool {
	return g.send(protocol.TypePlayerJoined, protocol.PlayerJoinedData{
		Slot:  int(slot.ID),
		Label: slot.Label,
		Color: slot.Color,
	})
}

func (g *gameLink) playerLeft(slot session.SlotID, reason string) bool {
	return g.send(protocol.TypePlayerLeft, protocol.PlayerLeftData{Slot: int(slot), Reason: reason})
}

func (g *gameLink) hostCommand(command string, slot *session.SlotID) bool {
	data := protocol.HostCommandData{Command: command}
	if slot != nil {
		value := int(*slot)
		data.Slot = &value
	}
	return g.send(protocol.TypeHostCommand, data)
}

func (g *gameLink) hostDenied(command, code, message string) bool {
	return g.send(protocol.TypeHostDenied, protocol.HostDeniedData{
		Command: command,
		Code:    code,
		Message: message,
	})
}

func (s *Server) handleGame(w http.ResponseWriter, r *http.Request) {
	if s.stopping.Load() {
		http.Error(w, "server is shutting down", http.StatusServiceUnavailable)
		return
	}
	if !isLoopbackRequest(r) {
		http.Error(w, "game endpoint is loopback-only", http.StatusForbidden)
		return
	}
	expected := s.gameToken()
	provided := r.URL.Query().Get("token")
	if expected == "" || !constantTimeEqual(provided, expected) {
		s.log.Warn("rejected game connection", "remote", r.RemoteAddr, "reason", "auth")
		http.Error(w, "invalid game token", http.StatusUnauthorized)
		return
	}
	if s.game.current() != nil {
		http.Error(w, "game already connected", http.StatusConflict)
		return
	}

	ws, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.log.Warn("game upgrade failed", "error", err)
		return
	}
	gc := newGameConn(ws, s.cfg.WriteTimeout())
	s.gameMu.Lock()
	if !constantTimeEqual(provided, s.gameToken()) {
		s.gameMu.Unlock()
		gc.closeNow(websocket.ClosePolicyViolation, "invalid game token")
		return
	}
	installed := s.game.trySet(gc)
	s.gameMu.Unlock()
	if !installed {
		gc.closeNow(websocket.ClosePolicyViolation, "game already connected")
		return
	}
	defer func() {
		s.gameMu.Lock()
		wasCurrent := s.game.clear(gc)
		if wasCurrent {
			s.onGameDisconnected(provided)
		}
		s.gameMu.Unlock()
		gc.closeNow(websocket.CloseGoingAway, "game disconnected")
	}()

	hello, err := protocol.EncodeMessage(protocol.Version, protocol.TypeGameHello, nil, protocol.GameHelloData{
		ProtocolVersion: protocol.Version,
	})
	if err != nil || !gc.enqueueCritical(hello) {
		return
	}

	env, err := readEnvelope(ws, s.cfg.MaxMessageBytes, s.cfg.HandshakeTimeout())
	if err != nil {
		s.log.Warn("game handshake failed", "error", err)
		return
	}
	if env.Type != protocol.TypeGameReady {
		s.log.Warn("game handshake used wrong message", "type", env.Type)
		return
	}
	ready, err := protocol.DecodeData[protocol.GameReadyData](env)
	if err != nil {
		s.log.Warn("game_ready payload invalid", "error", err)
		return
	}
	now := s.now()
	s.game.markReady(now)
	s.session.ResetForGame(now)
	s.log.Info("game ready", "gameVersion", ready.GameVersion, "mapVersion", ready.MapVersion)
	s.game.config(protocol.ConfigData{
		RoundDurationMs: s.cfg.RoundDurationMs,
		TasksPerPlayer:  s.cfg.TasksPerPlayer,
		MaxPlayers:      s.cfg.MaxPlayers,
	})
	for _, slot := range s.session.ConnectedSlots() {
		s.game.playerJoined(slot)
	}
	s.broadcastPhase()

	for {
		env, err := readEnvelope(ws, s.cfg.MaxMessageBytes, s.cfg.GameHeartbeatTimeout()+time.Second)
		if err != nil {
			s.log.Info("game connection closed", "error", err)
			return
		}
		s.game.touch(s.now())
		s.handleGameMessage(gc, env)
	}
}

func (s *Server) handleGameMessage(gc *gameConn, env protocol.Envelope) {
	switch env.Type {
	case protocol.TypePhase:
		data, err := protocol.DecodeData[protocol.PhaseData](env)
		if err != nil || !protocol.ValidPhase(data.Phase) {
			s.log.Warn("invalid phase from game", "type", env.Type, "error", err, "phase", data.Phase)
			return
		}
		if err := s.session.SetPhase(data.Phase, data.RoundID, data.TimeLeftMs, s.now()); err != nil {
			s.log.Warn("rejected phase", "phase", data.Phase, "error", err)
			return
		}
		s.broadcastPhase()
	case protocol.TypePlayerFeedbk:
		data, err := protocol.DecodeData[protocol.PlayerFeedbackData](env)
		if err != nil || !session.SlotID(data.Slot).Valid() || !protocol.ValidFeedbackState(data.State) {
			s.log.Warn("invalid player_feedback", "error", err, "slot", data.Slot, "state", data.State)
			return
		}
		s.session.SetFeedback(session.SlotID(data.Slot), data)
		s.sendToController(session.SlotID(data.Slot), protocol.TypePlayerFeedbk, data)
	case protocol.TypeTaskResult:
		data, err := protocol.DecodeData[protocol.TaskResultWithSlot](env)
		if err != nil || !session.SlotID(data.Slot).Valid() {
			s.log.Warn("invalid task_result", "error", err, "slot", data.Slot)
			return
		}
		s.session.UpdateScore(session.SlotID(data.Slot), data.TotalScore)
		s.sendToController(session.SlotID(data.Slot), protocol.TypeTaskResult, data.TaskResultData)
	case protocol.TypeRoundResult:
		data, err := protocol.DecodeData[protocol.RoundResultData](env)
		if err != nil {
			s.log.Warn("invalid round_result", "error", err)
			return
		}
		s.session.SetRoundResult(data)
		s.broadcast(protocol.TypeRoundResult, data)
	case protocol.TypeHostRequest:
		data, err := protocol.DecodeData[protocol.HostRequestData](env)
		if err != nil {
			s.log.Warn("invalid host_request", "error", err)
			return
		}
		s.handleHostRequest(data)
	case protocol.TypeHeartbeat:
		// Touch happens before dispatch; decode only to catch malformed frames.
		if _, err := protocol.DecodeData[protocol.HeartbeatData](env); err != nil {
			s.log.Warn("invalid heartbeat", "error", err)
		}
	case protocol.TypeGameReady:
		s.log.Debug("duplicate game_ready ignored")
	default:
		s.log.Warn("unknown message type from game", "type", env.Type)
	}
}

func (s *Server) handleHostRequest(data protocol.HostRequestData) {
	command := data.Command
	if !protocol.ValidHostCommand(command) {
		s.game.hostDenied(command, protocol.CodeNotAuthorized, "Unknown host command.")
		return
	}
	var slot *session.SlotID
	if data.Slot != nil {
		id := session.SlotID(*data.Slot)
		if !id.Valid() {
			s.game.hostDenied(command, protocol.CodeNotAuthorized, "Invalid slot.")
			return
		}
		slot = &id
	}
	if err := s.session.AuthorizeHost(command, slot); err != nil {
		pe := protocolError(err)
		s.game.hostDenied(command, pe.Code, pe.Message)
		s.log.Info("host command denied", "command", command, "code", pe.Code)
		return
	}
	now := s.now()
	switch command {
	case protocol.HostKick:
		if slot == nil {
			s.game.hostDenied(command, protocol.CodeNotAuthorized, "A slot is required.")
			return
		}
		if _, ok := s.session.Kick(*slot, now); !ok {
			s.game.hostDenied(command, protocol.CodeNotAuthorized, "That slot is no longer assigned.")
			return
		}
		s.hub.closeSlotWithError(*slot, protocol.CodeKicked, "Slot disabled by host.", websocket.ClosePolicyViolation, "kicked")
		s.game.hostCommand(protocol.HostKick, slot)
		s.log.Info("host kicked slot", "slot", *slot)
	case protocol.HostReset:
		s.game.hostCommand(command, nil)
		s.session.ResetRound(now)
		s.game.config(protocol.ConfigData{
			RoundDurationMs: s.cfg.RoundDurationMs,
			TasksPerPlayer:  s.cfg.TasksPerPlayer,
			MaxPlayers:      s.cfg.MaxPlayers,
		})
		s.broadcastPhase()
	case protocol.HostEnd:
		s.game.hostCommand(command, nil)
		s.session.End()
		s.broadcastPhase()
		s.hub.closeAllGraceful(websocket.CloseNormalClosure, "session ended")
	default:
		s.game.hostCommand(command, nil)
	}
	s.log.Info("host command accepted", "command", command)
}

func (s *Server) onGameDisconnected(gameToken string) {
	if s.stopping.Load() {
		return
	}
	if s.session.State().Phase == protocol.PhaseEnded {
		s.log.Info("game disconnected after session end")
		return
	}
	s.log.Warn("game disconnected; returning to attract")
	now := s.now()
	s.session.ResetForGame(now)
	s.broadcast(protocol.TypeError, protocol.ErrorData{
		Code:    protocol.CodeGameUnavailable,
		Message: "Game is unavailable. Waiting for restart.",
	})
	s.broadcastPhase()
	if s.sup != nil {
		s.sup.RestartFor(gameToken, "game connection lost")
	}
}

func (s *Server) sendToController(slot session.SlotID, messageType string, data any) bool {
	payload, err := protocol.EncodeMessage(protocol.Version, messageType, nil, data)
	if err != nil {
		return false
	}
	return s.hub.sendToSlot(slot, payload)
}

func (s *Server) broadcast(messageType string, data any) {
	payload, err := protocol.EncodeMessage(protocol.Version, messageType, nil, data)
	if err != nil {
		return
	}
	s.hub.broadcast(payload)
}

func (s *Server) broadcastPhase() {
	state := s.session.State()
	payload, err := protocol.EncodeMessage(protocol.Version, protocol.TypePhase, nil, protocol.PhaseData{
		Phase:      state.Phase,
		TimeLeftMs: state.TimeLeftMs,
		RoundID:    state.RoundID,
	})
	if err != nil {
		return
	}
	s.hub.broadcast(payload)
}

func isLoopbackRequest(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func constantTimeEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
