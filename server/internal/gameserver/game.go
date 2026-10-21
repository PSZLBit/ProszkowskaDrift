package gameserver

import (
	"crypto/subtle"
	"errors"
	"log/slog"
	"net/http"

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
	defer ws.CloseNow()

	conn := &gameConn{
		wsConn: newWSConn(ws),
		log:    s.log.With("component", "game"),
	}

	token := r.URL.Query().Get("token")
	if token == "" {
		token = r.Header.Get("X-Game-Token")
	}
	if token == "" || s.cfg.GameToken == "" ||
		subtle.ConstantTimeCompare([]byte(token), []byte(s.cfg.GameToken)) != 1 {
		conn.log.Warn("game connection rejected: auth failed")
		conn.sendError(protocol.CodeAuthFailed, "Invalid game token.")
		conn.close(websocket.StatusPolicyViolation, "auth_failed")
		return
	}

	// Send the handshake before registering the connection so a controller
	// cannot make player_joined the first frame the game sees.
	if err := conn.send(protocol.TypeGameHello, protocol.GameHelloData{
		Token:           s.cfg.GameToken,
		ProtocolVersion: protocol.Version,
	}); err != nil {
		conn.log.Debug("game_hello write failed", "error", err)
		return
	}
	if err := conn.send(protocol.TypeConfig, protocol.ConfigData{
		RoundDurationMs: s.cfg.RoundDurationMs,
		TasksPerPlayer:  s.cfg.TasksPerPlayer,
		MaxPlayers:      session.MaxPlayers,
	}); err != nil {
		conn.log.Debug("config write failed", "error", err)
		return
	}

	s.registerGame(conn)
	defer s.dropGame(conn)
	conn.log.Info("game connected")

	if err := s.sendExistingPlayers(conn); err != nil {
		conn.log.Debug("player_joined write failed", "error", err)
		return
	}

	for {
		env, err := readEnvelope(r.Context(), ws, 0)
		if err != nil {
			switch {
			case errors.Is(err, protocol.ErrUnsupportedVersion):
				conn.log.Warn("closing game: protocol mismatch", "error", err)
				conn.sendError(protocol.CodeProtocolMismatch, "Unsupported protocol version.")
				conn.close(websocket.StatusPolicyViolation, "protocol_mismatch")
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

		switch env.Type {
		case protocol.TypeGameReady:
			s.handleGameReady(conn, env)
		case protocol.TypeHostRequest:
			if s.handleHostRequest(conn, env) {
				return
			}
		case protocol.TypePhase:
			if s.handleGamePhase(conn, env) {
				return
			}
		case protocol.TypeHeartbeat:
			s.handleGameHeartbeat(conn, env)
		default:
			conn.log.Debug("dropping unknown game message type", "type", env.Type)
		}
	}
}

func (s *Server) handleGameReady(conn *gameConn, env protocol.Envelope) {
	data, err := protocol.DecodeData[protocol.GameReadyData](env)
	if err != nil {
		s.rejectGameInvalid(conn, "Malformed game_ready payload.")
		return
	}
	conn.log.Info("game ready", "game_version", data.GameVersion, "map_version", data.MapVersion)
}

func (s *Server) handleHostRequest(conn *gameConn, env protocol.Envelope) bool {
	data, err := protocol.DecodeData[protocol.HostRequestData](env)
	if err != nil {
		return s.rejectGameInvalid(conn, "Malformed host_request payload.")
	}

	command, err := session.ParseCommand(data.Command)
	if err != nil {
		return s.rejectGameInvalid(conn, "Unknown host command.")
	}

	result, err := s.session.HostCommand(command, data.Slot)
	if err != nil {
		return s.rejectGameInvalid(conn, "Host command rejected: "+err.Error())
	}

	reply := protocol.HostCommandData{Command: string(command)}
	if command == session.CommandKick {
		reply.Slot = result.Slot
	}
	if err := conn.send(protocol.TypeHostCommand, reply); err != nil {
		conn.log.Debug("host_command write failed", "error", err)
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
		s.kickController(result.Slot)
	}
	return false
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
		conn.close(websocket.StatusPolicyViolation, "invalid_input")
		return true
	}
	return false
}
