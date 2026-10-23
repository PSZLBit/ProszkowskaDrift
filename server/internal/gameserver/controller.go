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
	slot     atomic.Int32
	tokenRef string
	invalid  int
	rate     int
}

func (c *controllerConn) slotNumber() int {
	return int(c.slot.Load())
}

func (c *controllerConn) setSlot(slot int) {
	c.slot.Store(int32(slot))
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

	id := s.nextConnID.Add(1)
	conn := &controllerConn{
		wsConn: newWSConn(ws, s.cfg.ControllerQueueSize),
		id:     id,
		log:    s.log.With("component", "controller", "connection", id),
	}
	defer conn.killNow()
	defer s.removeControllerConn(conn)
	defer s.dropController(conn)
	s.addControllerConn(conn)

	conn.log.Info("controller connected", "remote", r.RemoteAddr)

	for {
		// The controller idle timeout is enforced centrally by StartTick: a
		// timed-out connection is closed from the tick loop with
		// heartbeat_timeout, which unblocks this read.
		env, err := readEnvelope(r.Context(), ws, 0)

		if err != nil {
			switch {
			case errors.Is(err, protocol.ErrUnsupportedVersion):
				conn.log.Warn("closing controller: protocol mismatch", "error", err)
				conn.sendError(protocol.CodeProtocolMismatch, "Unsupported protocol version.")
				conn.enqueueClose(nil, websocket.StatusPolicyViolation, "protocol_mismatch")
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

		if slot := conn.slotNumber(); slot != 0 {
			s.session.Touch(slot, conn.id, time.Now())
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
	if conn.slotNumber() != 0 {
		return s.rejectControllerInvalid(conn, protocol.CodeInvalidInput, "Controller is already assigned to a slot.")
	}

	s.gameNotifyMu.Lock()
	view, reconnected, err := s.session.Hello(data.DeviceToken, conn.id)
	if err != nil {
		s.gameNotifyMu.Unlock()
		switch {
		case errors.Is(err, session.ErrInvalidToken):
			conn.log.Warn("controller hello rejected: invalid device token", "token", truncateToken(data.DeviceToken))
			conn.sendError(protocol.CodeAuthFailed, "Invalid device token.")
		case errors.Is(err, session.ErrKickedToken):
			conn.log.Warn("controller hello rejected: kicked device token", "token", conn.tokenRef)
			conn.sendError(protocol.CodeKicked, "This device was disabled by the host.")
		case errors.Is(err, session.ErrSessionEnded):
			conn.log.Info("controller hello rejected: session ended", "token", conn.tokenRef)
			conn.sendError(protocol.CodeAuthFailed, "Session has ended. Restart the server.")
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

	conn.setSlot(view.Number)
	conn.tokenRef = truncateToken(data.DeviceToken)
	if s.afterControllerHello != nil {
		s.afterControllerHello()
	}
	if !s.registerControllerAndWelcome(conn, view) {
		s.gameNotifyMu.Unlock()
		conn.log.Debug("welcome enqueue failed")
		return true
	}
	s.sendPlayerJoined(view)
	s.gameNotifyMu.Unlock()

	conn.log.Info("controller joined",
		"slot", view.Number,
		"label", view.Label,
		"token", conn.tokenRef,
		"reconnected", reconnected,
	)
	return false
}

func (s *Server) handleControllerInput(conn *controllerConn, env protocol.Envelope) bool {
	slot := conn.slotNumber()
	if slot == 0 {
		return s.rejectControllerInvalid(conn, protocol.CodeInvalidInput, "hello is required before input.")
	}
	if env.Seq == nil {
		return s.rejectControllerInvalid(conn, protocol.CodeInvalidInput, "Input is missing seq.")
	}

	data, err := protocol.DecodeData[protocol.InputData](env)
	if err != nil || data.Steer == nil || data.Throttle == nil {
		return s.rejectControllerInvalid(conn, protocol.CodeInvalidInput, "Malformed input payload.")
	}

	event, err := s.session.ValidateInput(slot, conn.id, *env.Seq, *data.Steer, *data.Throttle)
	if err != nil {
		switch {
		case errors.Is(err, session.ErrRateLimited):
			return s.rejectControllerRate(conn)
		case errors.Is(err, session.ErrNotPlaying):
			// Input outside the playing phase is normal traffic during
			// pauses and phase changes. It is dropped silently and does not
			// count toward the abuse thresholds.
			conn.log.Debug("dropping input outside the playing phase", "slot", slot, "phase", s.session.Phase())
			return false
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
	if !conn.send(protocol.TypePong, protocol.PongData{
		ClientTime: data.ClientTime,
		ServerTime: time.Now().UnixMilli(),
	}) {
		conn.log.Debug("pong enqueue failed")
	}
}

func (s *Server) handleControllerLeave(conn *controllerConn) {
	slot := conn.slotNumber()
	if slot == 0 {
		s.rejectControllerInvalid(conn, protocol.CodeInvalidInput, "leave received before hello.")
		return
	}

	s.gameNotifyMu.Lock()
	view, changed := s.session.Leave(slot, conn.id)
	s.removeController(slot, conn)
	conn.setSlot(0)
	conn.tokenRef = ""
	if changed {
		s.sendPlayerLeft(view.Number, "leave")
	}
	s.gameNotifyMu.Unlock()

	if changed {
		conn.log.Info("controller left; slot reserved", "slot", slot)
	}
}

func (s *Server) rejectControllerInvalid(conn *controllerConn, code, message string) bool {
	conn.invalid++
	conn.log.Warn("invalid controller message",
		"slot", conn.slotNumber(),
		"token", conn.tokenRef,
		"reason", message,
		"count", conn.invalid,
	)
	conn.sendError(code, message)

	if conn.invalid >= s.cfg.InvalidThreshold {
		conn.log.Warn("closing controller after repeated invalid messages",
			"slot", conn.slotNumber(),
			"token", conn.tokenRef,
			"count", conn.invalid,
		)
		conn.enqueueClose(nil, websocket.StatusPolicyViolation, "invalid_input")
		return true
	}
	return false
}

func (s *Server) rejectControllerRate(conn *controllerConn) bool {
	conn.rate++
	conn.log.Warn("controller input rate limited",
		"slot", conn.slotNumber(),
		"token", conn.tokenRef,
		"count", conn.rate,
	)
	conn.sendError(protocol.CodeRateLimited, "Input rate exceeded 60 messages per second.")

	if conn.rate >= s.cfg.RateThreshold {
		conn.log.Warn("closing controller after repeated rate violations",
			"slot", conn.slotNumber(),
			"token", conn.tokenRef,
			"count", conn.rate,
		)
		conn.enqueueClose(nil, websocket.StatusPolicyViolation, "rate_limited")
		return true
	}
	return false
}
