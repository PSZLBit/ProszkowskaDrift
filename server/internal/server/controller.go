package server

import (
	"errors"
	"net/http"
	"time"

	"github.com/gorilla/websocket"

	"github.com/PSZLBit/ProszkowskaDrift/server/internal/logging"
	"github.com/PSZLBit/ProszkowskaDrift/server/internal/protocol"
	"github.com/PSZLBit/ProszkowskaDrift/server/internal/session"
)

func (s *Server) handleController(w http.ResponseWriter, r *http.Request) {
	if s.stopping.Load() {
		http.Error(w, "server is shutting down", http.StatusServiceUnavailable)
		return
	}
	ws, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.log.Warn("controller upgrade failed", "error", err)
		return
	}

	env, err := readEnvelope(ws, s.cfg.MaxMessageBytes, s.cfg.HandshakeTimeout())
	if err != nil {
		var pe *protocol.Error
		if !errors.As(err, &pe) {
			pe = protocol.NewError(protocol.CodeAuthFailed, "Handshake timeout.")
		}
		s.closeHandshake(ws, pe.Code, pe.Message)
		return
	}
	if env.Type != protocol.TypeHello {
		s.closeHandshake(ws, protocol.CodeAuthFailed, "The first message must be hello.")
		return
	}
	hello, err := protocol.DecodeData[protocol.HelloData](env)
	if err != nil {
		s.closeHandshake(ws, protocol.CodeInvalidInput, "Invalid hello payload.")
		return
	}
	if len(hello.DeviceToken) == 0 || len(hello.DeviceToken) > 128 {
		s.closeHandshake(ws, protocol.CodeAuthFailed, "Invalid device token.")
		return
	}

	snap, reclaimed, err := s.session.Attach(hello.DeviceToken, s.now())
	if err != nil {
		pe := protocolError(err)
		s.closeHandshake(ws, pe.Code, pe.Message)
		return
	}
	slot := snap.ID
	cc := newControllerConn(slot, snap.Generation, ws, s.cfg.WriteTimeout())
	s.hub.add(cc)
	s.log.Info("controller connected",
		"slot", int(slot),
		"reclaimed", reclaimed,
		"remote", r.RemoteAddr,
		"token", logging.RedactToken(hello.DeviceToken),
	)

	leaveReason := protocol.LeftDisconnect
	defer func() {
		s.hub.remove(slot, cc)
		if s.session.Detach(slot, cc.generation, s.now()) {
			s.game.playerLeft(slot, leaveReason)
		}
		if !cc.isClosing() {
			cc.closeNow(websocket.CloseNormalClosure, "")
		}
		s.log.Info("controller disconnected", "slot", int(slot))
	}()

	if !s.sendWelcome(cc, slot) {
		return
	}
	s.game.playerJoined(snap)

	for {
		env, err := readEnvelope(ws, s.cfg.MaxMessageBytes, s.cfg.ControllerIdleTimeout())
		if err != nil {
			var pe *protocol.Error
			if errors.As(err, &pe) {
				if pe.Code == protocol.CodeProtocolMismatch {
					if !cc.enqueueClose(errorFrame(pe.Code, pe.Message), websocket.ClosePolicyViolation, "protocol mismatch") {
						cc.closeNow(websocket.ClosePolicyViolation, "protocol mismatch")
					}
					return
				}
				if s.controllerViolation(cc, s.now(), false, pe.Code, pe.Message) {
					return
				}
				continue
			}
			return
		}
		now := s.now()
		s.session.Touch(slot, cc.generation, now)

		switch env.Type {
		case protocol.TypeInput:
			if env.Seq == nil {
				if s.controllerViolation(cc, now, false, protocol.CodeInvalidInput, "Input requires a sequence number.") {
					return
				}
				continue
			}
			data, err := protocol.DecodeData[protocol.InputData](env)
			if err != nil || data.Steer == nil || data.Throttle == nil {
				if s.controllerViolation(cc, now, false, protocol.CodeInvalidInput, "Invalid input payload.") {
					return
				}
				continue
			}
			forward, err := s.session.AcceptInput(slot, cc.generation, *env.Seq, *data.Steer, *data.Throttle, now)
			if err != nil {
				if errors.Is(err, session.ErrStaleInput) {
					continue
				}
				pe := protocolError(err)
				if pe.Code == protocol.CodeRateLimited {
					if s.controllerViolation(cc, now, true, pe.Code, pe.Message) {
						return
					}
					continue
				}
				if s.controllerViolation(cc, now, false, pe.Code, pe.Message) {
					return
				}
				continue
			}
			s.game.sendInput(forward)
		case protocol.TypePing:
			data, err := protocol.DecodeData[protocol.PingData](env)
			if err != nil {
				if s.controllerViolation(cc, now, false, protocol.CodeInvalidInput, "Invalid ping payload.") {
					return
				}
				continue
			}
			payload, _ := protocol.EncodeMessage(protocol.Version, protocol.TypePong, nil, protocol.PongData{
				ClientTime: data.ClientTime,
				ServerTime: now.UnixMilli(),
			})
			if !cc.enqueue(payload) {
				return
			}
		case protocol.TypeLeave:
			leaveReason = protocol.LeftLeave
			return
		case protocol.TypeHello:
			if s.controllerViolation(cc, now, false, protocol.CodeInvalidInput, "Duplicate hello.") {
				return
			}
		default:
			s.log.Warn("unknown message type from controller", "type", env.Type, "slot", int(slot))
		}
	}
}

func (s *Server) sendWelcome(cc *controllerConn, slot session.SlotID) bool {
	snap, ok := s.session.SnapshotForSlot(slot)
	if !ok || !snap.Connected {
		return false
	}
	state := s.session.State()
	payload, err := protocol.EncodeMessage(protocol.Version, protocol.TypeWelcome, nil, protocol.WelcomeData{
		Slot:  int(snap.ID),
		Label: snap.Label,
		Color: snap.Color,
		Phase: state.Phase,
	})
	if err != nil || !cc.enqueue(payload) {
		return false
	}
	phase, err := protocol.EncodeMessage(protocol.Version, protocol.TypePhase, nil, protocol.PhaseData{
		Phase:      state.Phase,
		TimeLeftMs: state.TimeLeftMs,
		RoundID:    state.RoundID,
	})
	if err != nil || !cc.enqueue(phase) {
		return false
	}
	if snap.Feedback != nil {
		feedback, err := protocol.EncodeMessage(protocol.Version, protocol.TypePlayerFeedbk, nil, snap.Feedback)
		if err != nil || !cc.enqueue(feedback) {
			return false
		}
	}
	if state.Phase == protocol.PhaseResults && state.LastRoundResult != nil {
		result, err := protocol.EncodeMessage(protocol.Version, protocol.TypeRoundResult, nil, state.LastRoundResult)
		if err != nil || !cc.enqueue(result) {
			return false
		}
	}
	return true
}

func (s *Server) controllerViolation(cc *controllerConn, now time.Time, rate bool, code, message string) bool {
	var closeConnection bool
	if rate {
		closeConnection = s.session.RecordRateLimit(cc.slot, cc.generation, now)
	} else {
		closeConnection = s.session.RecordInvalid(cc.slot, cc.generation, now)
	}
	if closeConnection {
		if !cc.enqueueClose(errorFrame(code, message), websocket.ClosePolicyViolation, "abuse") {
			cc.closeNow(websocket.ClosePolicyViolation, "abuse")
		}
		return true
	}
	if !cc.enqueue(errorFrame(code, message)) {
		return true
	}
	return false
}

func (s *Server) closeHandshake(ws *websocket.Conn, code, message string) {
	_ = writeMessage(ws, protocol.TypeError, protocol.ErrorData{Code: code, Message: message}, s.cfg.WriteTimeout())
	deadline := time.Now().Add(s.cfg.WriteTimeout())
	_ = ws.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.ClosePolicyViolation, code),
		deadline,
	)
	_ = ws.Close()
}
