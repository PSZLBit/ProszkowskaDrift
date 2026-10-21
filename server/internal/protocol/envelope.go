// Package protocol defines the versioned JSON envelope and message payloads
// shared by controllers, the server, and the game.
package protocol

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// Version is the MVP protocol version.
const Version = 1

// Message types.
const (
	TypeHello        = "hello"
	TypeInput        = "input"
	TypePing         = "ping"
	TypeLeave        = "leave"
	TypeWelcome      = "welcome"
	TypePhase        = "phase"
	TypeTaskResult   = "task_result"
	TypeRoundResult  = "round_result"
	TypeError        = "error"
	TypePong         = "pong"
	TypeGameHello    = "game_hello"
	TypeConfig       = "config"
	TypePlayerJoined = "player_joined"
	TypePlayerLeft   = "player_left"
	TypeHostCommand  = "host_command"
	TypeHostRequest  = "host_request"
	TypeHostDenied   = "host_denied"
	TypeGameReady    = "game_ready"
	TypePlayerFeedbk = "player_feedback"
	TypeHeartbeat    = "heartbeat"
)

// Error codes from protocol.md.
const (
	CodeAuthFailed       = "auth_failed"
	CodeSessionFull      = "session_full"
	CodeInvalidInput     = "invalid_input"
	CodeRateLimited      = "rate_limited"
	CodeProtocolMismatch = "protocol_mismatch"
	CodeGameUnavailable  = "game_unavailable"
	CodeKicked           = "kicked"
	CodeInternalError    = "internal_error"
)

// Host denial codes used by the server-to-game host_denied message.
const (
	CodeNoPlayers     = "no_players"
	CodeInvalidPhase  = "invalid_phase"
	CodeNotAuthorized = "not_authorized"
)

// Phases from protocol.md.
const (
	PhaseAttract   = "attract"
	PhaseReady     = "ready"
	PhaseCountdown = "countdown"
	PhasePlaying   = "playing"
	PhasePaused    = "paused"
	PhaseResults   = "results"
	PhaseEnded     = "ended"
)

// Host commands from protocol.md.
const (
	HostStart  = "start"
	HostPause  = "pause"
	HostResume = "resume"
	HostReset  = "reset"
	HostEnd    = "end"
	HostKick   = "kick"
)

// Player feedback states.
const (
	StateReady        = "ready"
	StatePlaying      = "playing"
	StateFinished     = "finished"
	StateReconnecting = "reconnecting"
)

// Envelope is the common message envelope.
type Envelope struct {
	V    int             `json:"v"`
	Type string          `json:"type"`
	Seq  *int64          `json:"seq,omitempty"`
	Data json.RawMessage `json:"data"`
}

// Error is a protocol-level error with a stable code.
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// NewError creates a protocol error.
func NewError(code, message string) *Error {
	return &Error{Code: code, Message: message}
}

// ParseEnvelope validates a raw JSON frame and returns its envelope.
func ParseEnvelope(raw []byte) (Envelope, error) {
	var env Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return Envelope{}, NewError(CodeInvalidInput, "Malformed JSON.")
	}
	if env.V != Version {
		return Envelope{}, NewError(CodeProtocolMismatch, "Unsupported protocol version.")
	}
	if strings.TrimSpace(env.Type) == "" {
		return Envelope{}, NewError(CodeInvalidInput, "Missing message type.")
	}
	if len(env.Data) == 0 {
		env.Data = json.RawMessage(`{}`)
	}
	trimmed := bytes.TrimSpace(env.Data)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return Envelope{}, NewError(CodeInvalidInput, "Message data must be an object.")
	}
	return env, nil
}

// DecodeData decodes the envelope data into T. Unknown fields are ignored as
// required by the protocol.
func DecodeData[T any](env Envelope) (T, error) {
	var out T
	if err := json.Unmarshal(env.Data, &out); err != nil {
		return out, NewError(CodeInvalidInput, "Invalid message payload.")
	}
	return out, nil
}

// EncodeMessage builds a protocol frame. A nil data value is encoded as {}.
func EncodeMessage(v int, messageType string, seq *int64, data any) ([]byte, error) {
	if v == 0 {
		v = Version
	}
	payload := json.RawMessage(`{}`)
	if data != nil {
		encoded, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		payload = encoded
	}
	return json.Marshal(Envelope{
		V:    v,
		Type: messageType,
		Seq:  seq,
		Data: payload,
	})
}

// ValidPhase reports whether phase is a protocol phase.
func ValidPhase(phase string) bool {
	switch phase {
	case PhaseAttract, PhaseReady, PhaseCountdown, PhasePlaying, PhasePaused, PhaseResults, PhaseEnded:
		return true
	default:
		return false
	}
}

// ValidHostCommand reports whether command is a protocol host command.
func ValidHostCommand(command string) bool {
	switch command {
	case HostStart, HostPause, HostResume, HostReset, HostEnd, HostKick:
		return true
	default:
		return false
	}
}

// ValidFeedbackState reports whether state is a documented player state.
func ValidFeedbackState(state string) bool {
	switch state {
	case StateReady, StatePlaying, StateFinished, StateReconnecting:
		return true
	default:
		return false
	}
}
