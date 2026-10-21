// Package session owns the in-memory session state: slots, device-token
// assignment, round phase, and input validation.
package session

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/PSZLBit/ProszkowskaDrift/server/internal/protocol"
	"github.com/PSZLBit/ProszkowskaDrift/server/internal/validation"
)

const (
	// MinPlayers and MaxPlayers bound the slots in one session.
	MinPlayers = 1
	MaxPlayers = 4

	defaultRateLimit  = 60
	defaultRateWindow = time.Second
)

// Command is a host command accepted by the server.
type Command string

const (
	CommandStart  Command = "start"
	CommandPause  Command = "pause"
	CommandResume Command = "resume"
	CommandReset  Command = "reset"
	CommandEnd    Command = "end"
	CommandKick   Command = "kick"
)

// ParseCommand converts a wire command into a Command.
func ParseCommand(value string) (Command, error) {
	switch Command(value) {
	case CommandStart, CommandPause, CommandResume, CommandReset, CommandEnd, CommandKick:
		return Command(value), nil
	default:
		return "", fmt.Errorf("%w: %q", ErrInvalidCommand, value)
	}
}

var (
	ErrSessionFull       = errors.New("session full")
	ErrInvalidToken      = errors.New("invalid device token")
	ErrDuplicateToken    = errors.New("device token already connected")
	ErrNotAssigned       = errors.New("slot not assigned")
	ErrWrongConnection   = errors.New("input not from the slot's connection")
	ErrNotPlaying        = errors.New("input only accepted while playing")
	ErrInvalidInput      = errors.New("invalid input")
	ErrStaleSequence     = errors.New("stale input sequence")
	ErrRateLimited       = errors.New("input rate exceeded")
	ErrInvalidCommand    = errors.New("invalid host command")
	ErrInvalidTransition = errors.New("host command not allowed in the current phase")
	ErrInvalidSlot       = errors.New("invalid slot")
	ErrInvalidPhase      = errors.New("invalid phase")
)

// Options configures a Session. Zero values use production defaults.
type Options struct {
	Clock      func() time.Time
	RateLimit  int
	RateWindow time.Duration
}

// SlotView is a snapshot of one slot for callers outside the session.
type SlotView struct {
	Number    int
	Label     string
	Color     string
	Assigned  bool
	Connected bool
}

// InputEvent is validated input ready to forward to the game.
type InputEvent struct {
	Slot     int
	Seq      int64
	Steer    float64
	Throttle float64
}

// CommandResult describes the effect of an accepted host command.
type CommandResult struct {
	Command       Command
	Slot          int
	Kicked        bool
	PreviousPhase protocol.Phase
	Phase         protocol.Phase
	PhaseChanged  bool
	RoundID       int64
}

type slotState struct {
	number       int
	label        string
	color        string
	token        string
	connected    bool
	connectionID uint64
	sequence     validation.Sequence
	limiter      *validation.RateLimiter
}

// Session is the concurrency-safe in-memory session state.
type Session struct {
	mu         sync.Mutex
	slots      [MaxPlayers]slotState
	phase      protocol.Phase
	roundID    int64
	clock      func() time.Time
	rateLimit  int
	rateWindow time.Duration
}

var slotColors = [MaxPlayers]string{"red", "blue", "green", "yellow"}

// New creates an empty session in the attract phase.
func New(opts Options) *Session {
	clock := opts.Clock
	if clock == nil {
		clock = time.Now
	}
	rateLimit := opts.RateLimit
	if rateLimit <= 0 {
		rateLimit = defaultRateLimit
	}
	rateWindow := opts.RateWindow
	if rateWindow <= 0 {
		rateWindow = defaultRateWindow
	}

	s := &Session{
		phase:      protocol.PhaseAttract,
		clock:      clock,
		rateLimit:  rateLimit,
		rateWindow: rateWindow,
	}
	for i := range s.slots {
		s.slots[i] = slotState{
			number:  i + 1,
			label:   fmt.Sprintf("P%d", i+1),
			color:   slotColors[i],
			limiter: validation.NewRateLimiter(rateLimit, rateWindow),
		}
	}
	return s
}

// Phase returns the current round phase.
func (s *Session) Phase() protocol.Phase {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.phase
}

// RoundID returns the current round identifier.
func (s *Session) RoundID() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.roundID
}

// Slots returns a snapshot of all four slots.
func (s *Session) Slots() []SlotView {
	s.mu.Lock()
	defer s.mu.Unlock()

	views := make([]SlotView, len(s.slots))
	for i := range s.slots {
		views[i] = s.slots[i].view()
	}
	return views
}

// Hello assigns a slot to token. A token that was assigned before and is not
// currently connected reclaims its slot. The returned bool reports a reconnect.
func (s *Session) Hello(token string, connectionID uint64) (SlotView, bool, error) {
	if !validToken(token) {
		return SlotView{}, false, ErrInvalidToken
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	reserved := -1
	for i := range s.slots {
		slot := &s.slots[i]
		if slot.token != token {
			continue
		}
		if slot.connected {
			return slot.view(), false, ErrDuplicateToken
		}
		slot.connected = true
		slot.connectionID = connectionID
		slot.sequence.Reset()
		return slot.view(), true, nil
	}

	for i := range s.slots {
		if s.slots[i].token == "" {
			reserved = i
			break
		}
	}
	if reserved < 0 {
		return SlotView{}, false, ErrSessionFull
	}

	slot := &s.slots[reserved]
	slot.token = token
	slot.connected = true
	slot.connectionID = connectionID
	slot.sequence.Reset()
	slot.limiter.Reset()
	return slot.view(), false, nil
}

// Disconnect marks the slot disconnected if connectionID still owns it. It
// reports whether the slot changed. The slot and its token stay reserved.
func (s *Session) Disconnect(slotNumber int, connectionID uint64) (SlotView, bool) {
	return s.release(slotNumber, connectionID)
}

// Leave is a clean controller leave. Like Disconnect, it keeps the slot
// reserved for the session.
func (s *Session) Leave(slotNumber int, connectionID uint64) (SlotView, bool) {
	return s.release(slotNumber, connectionID)
}

func (s *Session) release(slotNumber int, connectionID uint64) (SlotView, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if slotNumber < 1 || slotNumber > MaxPlayers {
		return SlotView{}, false
	}
	slot := &s.slots[slotNumber-1]
	view := slot.view()
	if !slot.connected || slot.connectionID != connectionID {
		return view, false
	}
	slot.connected = false
	slot.connectionID = 0
	view.Connected = false
	return view, true
}

// ValidateInput checks slot ownership, phase, rate, sequence, and ranges, in
// that order. It records the rate event and sequence number only when the
// whole message is accepted.
func (s *Session) ValidateInput(slotNumber int, connectionID uint64, seq int64, steer, throttle float64) (InputEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	slot, err := s.assignedSlot(slotNumber, connectionID)
	if err != nil {
		return InputEvent{}, err
	}
	if s.phase != protocol.PhasePlaying {
		return InputEvent{}, ErrNotPlaying
	}
	if !slot.limiter.Allow(s.clock()) {
		return InputEvent{}, ErrRateLimited
	}
	if err := validation.ValidInput(steer, throttle); err != nil {
		return InputEvent{}, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}
	if !slot.sequence.Accept(seq) {
		return InputEvent{}, ErrStaleSequence
	}

	return InputEvent{
		Slot:     slot.number,
		Seq:      seq,
		Steer:    steer,
		Throttle: throttle,
	}, nil
}

// SetPhase applies a phase reported by the trusted game. A positive roundID
// updates the round identifier.
func (s *Session) SetPhase(phase protocol.Phase, roundID int64) error {
	if !phase.Valid() {
		return fmt.Errorf("%w: %q", ErrInvalidPhase, phase)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.phase = phase
	if roundID > 0 {
		s.roundID = roundID
	}
	return nil
}

// HostCommand applies a validated host command. Kick uses the slot argument.
func (s *Session) HostCommand(command Command, slotNumber int) (CommandResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := CommandResult{
		Command:       command,
		Slot:          slotNumber,
		PreviousPhase: s.phase,
		Phase:         s.phase,
		RoundID:       s.roundID,
	}

	switch command {
	case CommandKick:
		if slotNumber < 1 || slotNumber > MaxPlayers {
			return CommandResult{}, fmt.Errorf("%w: %d", ErrInvalidSlot, slotNumber)
		}
		slot := &s.slots[slotNumber-1]
		if slot.token == "" {
			return result, nil
		}
		slot.token = ""
		slot.connected = false
		slot.connectionID = 0
		slot.sequence.Reset()
		slot.limiter.Reset()
		result.Kicked = true
		result.PhaseChanged = false
		return result, nil

	case CommandStart:
		if !startable(s.phase) {
			return CommandResult{}, fmt.Errorf("%w: cannot start from %q", ErrInvalidTransition, s.phase)
		}
		s.phase = protocol.PhasePlaying
		s.roundID++
	case CommandPause:
		if s.phase != protocol.PhasePlaying {
			return CommandResult{}, fmt.Errorf("%w: cannot pause from %q", ErrInvalidTransition, s.phase)
		}
		s.phase = protocol.PhasePaused
	case CommandResume:
		if s.phase != protocol.PhasePaused {
			return CommandResult{}, fmt.Errorf("%w: cannot resume from %q", ErrInvalidTransition, s.phase)
		}
		s.phase = protocol.PhasePlaying
	case CommandReset:
		s.phase = protocol.PhaseReady
	case CommandEnd:
		s.phase = protocol.PhaseEnded
	default:
		return CommandResult{}, fmt.Errorf("%w: %q", ErrInvalidCommand, command)
	}

	result.Phase = s.phase
	result.PhaseChanged = result.Phase != result.PreviousPhase
	result.RoundID = s.roundID
	return result, nil
}

func (s *Session) assignedSlot(slotNumber int, connectionID uint64) (*slotState, error) {
	if slotNumber < 1 || slotNumber > MaxPlayers {
		return nil, ErrNotAssigned
	}
	slot := &s.slots[slotNumber-1]
	if slot.token == "" || !slot.connected {
		return nil, ErrNotAssigned
	}
	if slot.connectionID != connectionID {
		return nil, ErrWrongConnection
	}
	return slot, nil
}

func (s *slotState) view() SlotView {
	return SlotView{
		Number:    s.number,
		Label:     s.label,
		Color:     s.color,
		Assigned:  s.token != "",
		Connected: s.connected,
	}
}

func startable(phase protocol.Phase) bool {
	switch phase {
	case protocol.PhaseAttract, protocol.PhaseReady, protocol.PhaseResults, protocol.PhaseEnded:
		return true
	default:
		return false
	}
}

func validToken(token string) bool {
	if len(token) < 8 || len(token) > 128 {
		return false
	}
	for i := 0; i < len(token); i++ {
		if token[i] < 0x21 || token[i] > 0x7e {
			return false
		}
	}
	return true
}
