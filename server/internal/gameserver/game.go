package gameserver

import (
	"crypto/subtle"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/coder/websocket"

	"github.com/PSZLBit/ProszkowskaDrift/server/internal/protocol"
	"github.com/PSZLBit/ProszkowskaDrift/server/internal/session"
)

const maxGameMessageSize = 64 * 1024

type gameConn struct {
	*wsConn
	log     *slog.Logger
	invalid int
}

func (s *Server) handleGame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !isLoopbackRemoteAddr(r.RemoteAddr) {
		http.Error(w, "game endpoint is localhost only", http.StatusForbidden)
		return
	}

	ws, err := websocket.Accept(w, r, nil)
	if err != nil {
		s.log.Debug("game websocket accept failed", "error", err)
		return
	}
	ws.SetReadLimit(maxGameMessageSize)

	conn := &gameConn{
		wsConn: newWSConn(ws, s.cfg.GameQueueSize),
		log:    s.log.With("component", "game"),
	}
	defer conn.killNow()

	token := r.URL.Query().Get("token")
	if token == "" {
		token = r.Header.Get("X-Game-Token")
	}
	if token == "" || s.cfg.GameToken == "" ||
		subtle.ConstantTimeCompare([]byte(token), []byte(s.cfg.GameToken)) != 1 {
		conn.log.Warn("game connection rejected: auth failed")
		conn.sendError(protocol.CodeAuthFailed, "Invalid game token.")
		conn.enqueueClose(nil, websocket.StatusPolicyViolation, "auth_failed")
		return
	}

	// Send the handshake before registering the connection so a controller
	// cannot make player_joined the first frame the game sees.
	if !conn.send(protocol.TypeGameHello, protocol.GameHelloData{
		Token:           s.cfg.GameToken,
		ProtocolVersion: protocol.Version,
	}) {
		conn.log.Debug("game_hello enqueue failed")
		return
	}
	if !conn.send(protocol.TypeConfig, protocol.ConfigData{
		RoundDurationMs: s.cfg.RoundDurationMs,
		TasksPerPlayer:  s.cfg.TasksPerPlayer,
		MaxPlayers:      session.MaxPlayers,
	}) {
		conn.log.Debug("config enqueue failed")
		return
	}

	s.gameActionMu.Lock()
	s.gameNotifyMu.Lock()
	s.registerGame(conn)
	defer s.dropGame(conn)
	if err := s.sendExistingPlayers(conn); err != nil {
		s.gameNotifyMu.Unlock()
		s.gameActionMu.Unlock()
		conn.log.Debug("player_joined enqueue failed", "error", err)
		return
	}
	s.gameNotifyMu.Unlock()
	s.gameActionMu.Unlock()
	conn.log.Info("game connected")

	for {
		env, err := readEnvelope(r.Context(), ws, 0)
		if err != nil {
			switch {
			case errors.Is(err, protocol.ErrUnsupportedVersion):
				conn.log.Warn("closing game: protocol mismatch", "error", err)
				conn.sendError(protocol.CodeProtocolMismatch, "Unsupported protocol version.")
				conn.enqueueClose(nil, websocket.StatusPolicyViolation, "protocol_mismatch")
				return
			case errors.Is(err, protocol.ErrMalformed), errors.Is(err, errNonTextFrame):
				if s.rejectGameInvalid(conn, "Malformed message.") {
					return
				}
				continue
			default:
				conn.log.Info("game disconnected", "error", err)
				return
			}
		}

		// Any valid message from the current game refreshes its heartbeat
		// deadline. Messages from a replaced connection are dropped.
		if s.handleGameMessage(conn, env) {
			return
		}
	}
}

// handleGameMessage applies one game message while holding gameActionMu. It
// returns true when the connection should stop handling messages.
func (s *Server) handleGameMessage(conn *gameConn, env protocol.Envelope) bool {
	s.gameActionMu.Lock()
	defer s.gameActionMu.Unlock()

	if !s.gameIsCurrent(conn) {
		conn.log.Info("dropping message from replaced game", "type", env.Type)
		return true
	}
	s.setGameMessageTime(time.Now())

	switch env.Type {
	case protocol.TypeGameReady:
		s.handleGameReady(conn, env)
	case protocol.TypeHostRequest:
		return s.handleHostRequest(conn, env)
	case protocol.TypePhase:
		return s.handleGamePhase(conn, env)
	case protocol.TypeHeartbeat:
		s.handleGameHeartbeat(conn, env)
	default:
		conn.log.Debug("dropping unknown game message type", "type", env.Type)
	}
	return false
}

func (s *Server) handleGameReady(conn *gameConn, env protocol.Envelope) {
	data, err := protocol.DecodeData[protocol.GameReadyData](env)
	if err != nil {
		s.rejectGameInvalid(conn, "Malformed game_ready payload.")
		return
	}
	s.setGameReady(true)
	conn.log.Info("game ready", "game_version", data.GameVersion, "map_version", data.MapVersion)
}

// denialCode maps a session host-command error to a host_denied code.
func denialCode(err error) (string, string) {
	switch {
	case errors.Is(err, session.ErrNoPlayers):
		return protocol.DenialNoPlayers, "No controller is connected."
	case errors.Is(err, session.ErrInvalidTransition):
		return protocol.DenialInvalidPhase, "Command not allowed in the current phase."
	case errors.Is(err, session.ErrInvalidCommand), errors.Is(err, session.ErrInvalidSlot):
		return protocol.DenialNotAuthorized, "Unknown command or slot."
	default:
		return protocol.DenialNotAuthorized, "Command rejected by the server."
	}
}

func (s *Server) handleHostRequest(conn *gameConn, env protocol.Envelope) bool {
	data, err := protocol.DecodeData[protocol.HostRequestData](env)
	if err != nil {
		return s.rejectGameInvalid(conn, "Malformed host_request payload.")
	}

	command, err := session.ParseCommand(data.Command)
	if err != nil {
		s.sendHostDenied(conn, data.Command, data.Slot, protocol.DenialNotAuthorized, "Unknown host command.")
		return false
	}
	if !s.gameIsReady() {
		s.sendHostDenied(conn, string(command), data.Slot, protocol.DenialInvalidPhase, "Game is not ready.")
		return false
	}

	// Serialize a kick with controller slot transitions so a reconnect cannot
	// claim the freed slot before the old controller is notified and closed.
	var kickConn *controllerConn
	if command == session.CommandKick {
		s.gameNotifyMu.Lock()
		defer s.gameNotifyMu.Unlock()
		kickConn = s.controllerForSlot(data.Slot)
	}

	result, err := s.session.HostCommand(command, data.Slot)
	if err != nil {
		code, message := denialCode(err)
		conn.log.Info("host command denied", "command", command, "slot", data.Slot, "code", code, "error", err)
		s.sendHostDenied(conn, string(command), data.Slot, code, message)
		return false
	}

	reply := protocol.HostCommandData{Command: string(command)}
	if command == session.CommandKick {
		reply.Slot = result.Slot
	}
	if !conn.send(protocol.TypeHostCommand, reply) {
		conn.log.Debug("host_command enqueue failed")
		return true
	}

	conn.log.Info("host command accepted",
		"command", command,
		"slot", result.Slot,
		"phase", result.Phase,
		"round", result.RoundID,
		"kicked", result.Kicked,
	)

	if result.PhaseChanged {
		s.broadcastPhase(protocol.PhaseData{
			Phase:   result.Phase,
			RoundID: result.RoundID,
		})
	}
	if result.Kicked {
		s.kickController(kickConn, result.Slot)
	}
	if command == session.CommandEnd {
		s.closeAllControllers("session_ended")
	}
	return false
}

func (s *Server) sendHostDenied(conn *gameConn, command string, slot int, code, message string) {
	if !conn.send(protocol.TypeHostDenied, protocol.HostDeniedData{
		Command: command,
		Slot:    slot,
		Code:    code,
		Message: message,
	}) {
		conn.log.Debug("host_denied enqueue failed")
	}
}

func (s *Server) handleGamePhase(conn *gameConn, env protocol.Envelope) bool {
	data, err := protocol.DecodeData[protocol.PhaseData](env)
	if err != nil {
		return s.rejectGameInvalid(conn, "Malformed phase payload.")
	}
	if !data.Phase.Valid() {
		return s.rejectGameInvalid(conn, "Unknown phase.")
	}
	if err := s.session.SetPhase(data.Phase, data.RoundID); err != nil {
		if errors.Is(err, session.ErrSessionEnded) {
			// After a session end the server stops forwarding phase
			// changes. This is not an abuse signal.
			conn.log.Info("dropping game phase after session end", "phase", data.Phase)
			return false
		}
		return s.rejectGameInvalid(conn, "Phase rejected: "+err.Error())
	}

	roundID := data.RoundID
	if roundID <= 0 {
		roundID = s.session.RoundID()
	}
	data.RoundID = roundID
	conn.log.Info("game phase", "phase", data.Phase, "round", data.RoundID, "time_left_ms", data.TimeLeftMs)
	s.broadcastPhase(data)
	return false
}

func (s *Server) handleGameHeartbeat(conn *gameConn, env protocol.Envelope) {
	data, err := protocol.DecodeData[protocol.HeartbeatData](env)
	if err != nil {
		s.rejectGameInvalid(conn, "Malformed heartbeat payload.")
		return
	}
	conn.log.Debug("game heartbeat", "game_time_ms", data.GameTimeMs)
}

func (s *Server) rejectGameInvalid(conn *gameConn, message string) bool {
	conn.invalid++
	conn.log.Warn("invalid game message", "reason", message, "count", conn.invalid)
	conn.sendError(protocol.CodeInvalidInput, message)

	if conn.invalid >= s.cfg.InvalidThreshold {
		conn.log.Warn("closing game after repeated invalid messages", "count", conn.invalid)
		conn.enqueueClose(nil, websocket.StatusPolicyViolation, "invalid_input")
		return true
	}
	return false
}
