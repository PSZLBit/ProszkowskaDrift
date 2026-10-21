package gameserver

import (
	"errors"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"

	"github.com/PSZLBit/ProszkowskaDrift/server/internal/protocol"
	"github.com/PSZLBit/ProszkowskaDrift/server/internal/session"
)

const maxControllerMessageSize = 16 * 1024

type controllerConn struct {
	*wsConn
	id       uint64
	log      *slog.Logger
	slot     int
	tokenRef string
	invalid  int
	rate     int
}

func (s *Server) handleController(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ws, err := websocket.Accept(w, r, nil)
	if err != nil {
		s.log.Debug("controller websocket accept failed", "error", err)
		return
	}
	ws.SetReadLimit(maxControllerMessageSize)
	defer ws.CloseNow()

	id := s.nextConnID.Add(1)
	conn := &controllerConn{
		wsConn: newWSConn(ws),
		id:     id,
		log:    s.log.With("component", "controller", "connection", id),
	}
	defer s.dropController(conn)

	conn.log.Info("controller connected", "remote", r.RemoteAddr)

	for {
		// coder/websocket closes the TCP connection when a read context
		// expires, which skips the close handshake. Use a watchdog that
		// performs a normal close instead so the controller can observe the
		// heartbeat_timeout reason.
		var timedOut atomic.Bool
		watchdog := time.AfterFunc(s.cfg.ControllerTimeout, func() {
			timedOut.Store(true)
			conn.close(websocket.StatusPolicyViolation, "heartbeat_timeout")
		})
		env, err := readEnvelope(r.Context(), ws, 0)
		watchdog.Stop()

		if timedOut.Load() {
			conn.log.Info("closing controller: heartbeat timeout", "timeout", s.cfg.ControllerTimeout)
			return
		}
		if err != nil {
			switch {
			case errors.Is(err, protocol.ErrUnsupportedVersion):
				conn.log.Warn("closing controller: protocol mismatch", "error", err)
				conn.sendError(protocol.CodeProtocolMismatch, "Unsupported protocol version.")
				conn.close(websocket.StatusPolicyViolation, "protocol_mismatch")
				return
			case errors.Is(err, protocol.ErrMalformed), errors.Is(err, errNonTextFrame):
				if s.rejectControllerInvalid(conn, protocol.CodeInvalidInput, "Malformed message.") {
					return
				}
				continue
			default:
				conn.log.Debug("controller read ended", "error", err)
				return
			}
		}

		switch env.Type {
		case protocol.TypePing:
			s.handleControllerPing(conn, env)
		case protocol.TypeHello:
			if s.handleControllerHello(conn, env) {
				return
			}
		case protocol.TypeInput:
			if s.handleControllerInput(conn, env) {
				return
			}
		case protocol.TypeLeave:
			s.handleControllerLeave(conn)
		default:
			conn.log.Debug("dropping unknown controller message type", "type", env.Type)
		}
	}
}

func (s *Server) handleControllerHello(conn *controllerConn, env protocol.Envelope) bool {
	data, err := protocol.DecodeData[protocol.HelloData](env)
	if err != nil {
		return s.rejectControllerInvalid(conn, protocol.CodeInvalidInput, "Malformed hello payload.")
	}
	if conn.slot != 0 {
		return s.rejectControllerInvalid(conn, protocol.CodeInvalidInput, "Controller is already assigned to a slot.")
	}

	view, reconnected, err := s.session.Hello(data.DeviceToken, conn.id)
	if err != nil {
		switch {
		case errors.Is(err, session.ErrInvalidToken):
			conn.log.Warn("controller hello rejected: invalid device token", "token", truncateToken(data.DeviceToken))
			conn.sendError(protocol.CodeAuthFailed, "Invalid device token.")
		case errors.Is(err, session.ErrDuplicateToken):
			conn.log.Warn("controller hello rejected: token already connected", "token", truncateToken(data.DeviceToken))
			conn.sendError(protocol.CodeAuthFailed, "This device token is already connected.")
		case errors.Is(err, session.ErrSessionFull):
			conn.log.Info("controller hello rejected: session full", "token", truncateToken(data.DeviceToken))
			conn.sendError(protocol.CodeSessionFull, "No free player slot.")
		default:
			conn.log.Error("controller hello failed", "error", err)
			conn.sendError(protocol.CodeInternalError, "Could not assign a slot.")
		}
		return false
	}

	conn.slot = view.Number
	conn.tokenRef = truncateToken(data.DeviceToken)
	s.registerController(conn)

	if err := conn.send(protocol.TypeWelcome, protocol.WelcomeData{
		Slot:  view.Number,
		Label: view.Label,
		Color: view.Color,
		Phase: s.session.Phase(),
	}); err != nil {
		conn.log.Debug("welcome write failed", "error", err)
		return true
	}

	s.sendPlayerJoined(view)
	conn.log.Info("controller joined",
		"slot", view.Number,
		"label", view.Label,
		"token", conn.tokenRef,
		"reconnected", reconnected,
	)
	return false
}

func (s *Server) handleControllerInput(conn *controllerConn, env protocol.Envelope) bool {
	if conn.slot == 0 {
		return s.rejectControllerInvalid(conn, protocol.CodeInvalidInput, "hello is required before input.")
	}
	if env.Seq == nil {
		return s.rejectControllerInvalid(conn, protocol.CodeInvalidInput, "Input is missing seq.")
	}

	data, err := protocol.DecodeData[protocol.InputData](env)
	if err != nil || data.Steer == nil || data.Throttle == nil {
		return s.rejectControllerInvalid(conn, protocol.CodeInvalidInput, "Malformed input payload.")
	}

	event, err := s.session.ValidateInput(conn.slot, conn.id, *env.Seq, *data.Steer, *data.Throttle)
	if err != nil {
		switch {
		case errors.Is(err, session.ErrRateLimited):
			return s.rejectControllerRate(conn)
		case errors.Is(err, session.ErrNotPlaying):
			return s.rejectControllerInvalid(conn, protocol.CodeInvalidInput, "Input is only accepted while playing.")
		case errors.Is(err, session.ErrStaleSequence):
			return s.rejectControllerInvalid(conn, protocol.CodeInvalidInput, "Input sequence is stale.")
		case errors.Is(err, session.ErrInvalidInput):
			return s.rejectControllerInvalid(conn, protocol.CodeInvalidInput, "Input value is not a finite number in [-1, 1].")
		case errors.Is(err, session.ErrNotAssigned), errors.Is(err, session.ErrWrongConnection):
			return s.rejectControllerInvalid(conn, protocol.CodeInvalidInput, "Input does not belong to an assigned slot.")
		default:
			conn.log.Error("input validation failed", "error", err)
			conn.sendError(protocol.CodeInternalError, "Could not validate input.")
			return false
		}
	}

	conn.invalid = 0
	conn.rate = 0
	s.sendToGame(protocol.TypeInput, protocol.ForwardedInputData{
		Slot:     event.Slot,
		Seq:      event.Seq,
		Steer:    event.Steer,
		Throttle: event.Throttle,
	})
	return false
}

func (s *Server) handleControllerPing(conn *controllerConn, env protocol.Envelope) {
	data, err := protocol.DecodeData[protocol.PingData](env)
	if err != nil {
		s.rejectControllerInvalid(conn, protocol.CodeInvalidInput, "Malformed ping payload.")
		return
	}
	if err := conn.send(protocol.TypePong, protocol.PongData{
		ClientTime: data.ClientTime,
		ServerTime: time.Now().UnixMilli(),
	}); err != nil {
		conn.log.Debug("pong write failed", "error", err)
	}
}

func (s *Server) handleControllerLeave(conn *controllerConn) {
	if conn.slot == 0 {
		s.rejectControllerInvalid(conn, protocol.CodeInvalidInput, "leave received before hello.")
		return
	}

	slot := conn.slot
	view, changed := s.session.Leave(slot, conn.id)
	s.removeController(slot, conn)
	conn.slot = 0
	conn.tokenRef = ""
	if changed {
		s.sendPlayerLeft(view.Number, "leave")
		conn.log.Info("controller left; slot reserved", "slot", slot)
	}
}

func (s *Server) rejectControllerInvalid(conn *controllerConn, code, message string) bool {
	conn.invalid++
	conn.log.Warn("invalid controller message",
		"slot", conn.slot,
		"token", conn.tokenRef,
		"reason", message,
		"count", conn.invalid,
	)
	conn.sendError(code, message)

	if conn.invalid >= s.cfg.InvalidThreshold {
		conn.log.Warn("closing controller after repeated invalid messages",
			"slot", conn.slot,
			"token", conn.tokenRef,
			"count", conn.invalid,
		)
		conn.close(websocket.StatusPolicyViolation, "invalid_input")
		return true
	}
	return false
}

func (s *Server) rejectControllerRate(conn *controllerConn) bool {
	conn.rate++
	conn.log.Warn("controller input rate limited",
		"slot", conn.slot,
		"token", conn.tokenRef,
		"count", conn.rate,
	)
	conn.sendError(protocol.CodeRateLimited, "Input rate exceeded 60 messages per second.")

	if conn.rate >= s.cfg.RateThreshold {
		conn.log.Warn("closing controller after repeated rate violations",
			"slot", conn.slot,
			"token", conn.tokenRef,
			"count", conn.rate,
		)
		conn.close(websocket.StatusPolicyViolation, "rate_limited")
		return true
	}
	return false
}
