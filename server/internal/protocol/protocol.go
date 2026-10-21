// Package protocol defines the JSON envelope and message payloads exchanged
// between controllers, the server, and the game.
package protocol

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

// Version is the only protocol version accepted by this server.
const Version = 1

// Message types.
const (
	TypeHello          = "hello"
	TypeInput          = "input"
	TypePing           = "ping"
	TypePong           = "pong"
	TypeLeave          = "leave"
	TypeWelcome        = "welcome"
	TypePhase          = "phase"
	TypeError          = "error"
	TypeGameHello      = "game_hello"
	TypeGameReady      = "game_ready"
	TypeConfig         = "config"
	TypePlayerJoined   = "player_joined"
	TypePlayerLeft     = "player_left"
	TypeHostRequest    = "host_request"
	TypeHostCommand    = "host_command"
	TypeHeartbeat      = "heartbeat"
	TypeTask           = "task"
	TypeTaskResult     = "task_result"
	TypeRoundResult    = "round_result"
	TypePlayerFeedback = "player_feedback"
)

// Error codes sent in an error message.
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

var (
	// ErrMalformed reports a frame that is not a valid protocol envelope.
	ErrMalformed = errors.New("malformed envelope")
	// ErrUnsupportedVersion reports an envelope with a version other than Version.
	ErrUnsupportedVersion = errors.New("unsupported protocol version")
)

// Phase is a round phase as defined in the protocol.
type Phase string

const (
	PhaseAttract   Phase = "attract"
	PhaseReady     Phase = "ready"
	PhaseCountdown Phase = "countdown"
	PhasePlaying   Phase = "playing"
	PhasePaused    Phase = "paused"
	PhaseResults   Phase = "results"
	PhaseEnded     Phase = "ended"
)

// Valid reports whether p is a phase defined by the protocol.
func (p Phase) Valid() bool {
	switch p {
	case PhaseAttract, PhaseReady, PhaseCountdown, PhasePlaying, PhasePaused, PhaseResults, PhaseEnded:
		return true
	default:
		return false
	}
}

// Envelope is the common frame used by every message.
type Envelope struct {
	V    int             `json:"v"`
	Type string          `json:"type"`
	Seq  *int64          `json:"seq,omitempty"`
	Data json.RawMessage `json:"data,omitempty"`
}

// ParseEnvelope validates a raw frame and returns its envelope. The data
// payload is not decoded here; use DecodeData for that.
func ParseEnvelope(raw []byte) (Envelope, error) {
	var env Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return Envelope{}, fmt.Errorf("%w: %v", ErrMalformed, err)
	}
	if env.V != Version {
		return Envelope{}, fmt.Errorf("%w: got %d, want %d", ErrUnsupportedVersion, env.V, Version)
	}
	if env.Type == "" {
		return Envelope{}, fmt.Errorf("%w: missing type", ErrMalformed)
	}
	if rawData := bytes.TrimSpace(env.Data); len(rawData) > 0 && rawData[0] != '{' {
		return Envelope{}, fmt.Errorf("%w: data must be a JSON object", ErrMalformed)
	}
	return env, nil
}

// DecodeData decodes the envelope's data payload into T. Unknown fields are
// ignored by encoding/json.
func DecodeData[T any](env Envelope) (T, error) {
	var out T
	if len(env.Data) == 0 {
		return out, fmt.Errorf("%w: missing data", ErrMalformed)
	}
	if err := json.Unmarshal(env.Data, &out); err != nil {
		return out, fmt.Errorf("%w: %v", ErrMalformed, err)
	}
	return out, nil
}

// Encode builds a protocol frame. seq is optional and may be nil.
func Encode(typ string, seq *int64, data any) ([]byte, error) {
	rawData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal %s data: %w", typ, err)
	}
	raw, err := json.Marshal(Envelope{V: Version, Type: typ, Seq: seq, Data: rawData})
	if err != nil {
		return nil, fmt.Errorf("marshal %s envelope: %w", typ, err)
	}
	return raw, nil
}

// HelloData is sent by a controller immediately after connecting.
type HelloData struct {
	DeviceToken string `json:"deviceToken"`
	AppVersion  string `json:"appVersion,omitempty"`
}

// InputData is a controller input sample.
type InputData struct {
	Steer    *float64 `json:"steer"`
	Throttle *float64 `json:"throttle"`
	TS       int64    `json:"ts,omitempty"`
}

// PingData is a controller heartbeat.
type PingData struct {
	ClientTime int64 `json:"clientTime"`
}

// WelcomeData assigns a slot to a controller.
type WelcomeData struct {
	Slot  int    `json:"slot"`
	Label string `json:"label"`
	Color string `json:"color"`
	Phase Phase  `json:"phase"`
}

// PhaseData reports the current round phase.
type PhaseData struct {
	Phase      Phase `json:"phase"`
	TimeLeftMs int64 `json:"timeLeftMs,omitempty"`
	RoundID    int64 `json:"roundId,omitempty"`
}

// ErrorData reports a rejected message or a failed action.
type ErrorData struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// PongData answers a controller ping.
type PongData struct {
	ClientTime int64 `json:"clientTime"`
	ServerTime int64 `json:"serverTime"`
}

// GameHelloData is the first message the server sends to the game.
type GameHelloData struct {
	Token           string `json:"token"`
	ProtocolVersion int    `json:"protocolVersion"`
}

// ConfigData carries the round configuration to the game.
type ConfigData struct {
	RoundDurationMs int `json:"roundDurationMs"`
	TasksPerPlayer  int `json:"tasksPerPlayer"`
	MaxPlayers      int `json:"maxPlayers"`
}

// PlayerJoinedData tells the game that a slot is connected.
type PlayerJoinedData struct {
	Slot  int    `json:"slot"`
	Label string `json:"label"`
	Color string `json:"color"`
}

// PlayerLeftData tells the game that a slot is no longer connected.
type PlayerLeftData struct {
	Slot   int    `json:"slot"`
	Reason string `json:"reason"`
}

// ForwardedInputData is validated controller input forwarded to the game.
type ForwardedInputData struct {
	Slot     int     `json:"slot"`
	Seq      int64   `json:"seq"`
	Steer    float64 `json:"steer"`
	Throttle float64 `json:"throttle"`
}

// HostCommandData is a validated host command sent to the game.
type HostCommandData struct {
	Command string `json:"command"`
	Slot    int    `json:"slot,omitempty"`
}

// GameReadyData is sent by the game once it is ready to receive commands.
type GameReadyData struct {
	GameVersion string `json:"gameVersion"`
	MapVersion  string `json:"mapVersion"`
}

// HostRequestData asks the server to execute a host command.
type HostRequestData struct {
	Command string `json:"command"`
	Slot    int    `json:"slot,omitempty"`
}

// HeartbeatData is the game's periodic liveness message.
type HeartbeatData struct {
	GameTimeMs int64 `json:"gameTimeMs"`
}
