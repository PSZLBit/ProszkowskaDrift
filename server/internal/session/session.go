// Package session owns the in-memory server session: slots, device tokens,
// phases, input validation, reconnect support, and host-command authorisation.
package session

import (
	"math"
	"sync"
	"time"

	"github.com/PSZLBit/ProszkowskaDrift/server/internal/protocol"
)

// SlotID is a player slot number, 1-4.
type SlotID int

// Valid reports whether the slot is in the supported range.
func (s SlotID) Valid() bool { return s >= 1 && s <= 4 }

// SlotDef is the stable label and colour for a slot.
type SlotDef struct {
	ID    SlotID
	Label string
	Color string
}

// Timings contains validation and timeout thresholds.
type Timings struct {
	ControllerIdleTimeout time.Duration
	AllDisconnectedGrace  time.Duration
	InvalidErrorLimit     int
	RateErrorLimit        int
	ErrorWindow           time.Duration
	InputRatePerSecond    float64
	InputBurst            float64
}

// DefaultTimings returns production defaults.
func DefaultTimings() Timings {
	return Timings{
		ControllerIdleTimeout: 6 * time.Second,
		AllDisconnectedGrace:  10 * time.Second,
		InvalidErrorLimit:     10,
		RateErrorLimit:        3,
		ErrorWindow:           10 * time.Second,
		InputRatePerSecond:    60,
		InputBurst:            60,
	}
}

// Options configures a session.
type Options struct {
	Slots    []SlotDef
	Timings  Timings
	MaxSlots int
}

// SlotSnapshot is a copy of one slot's public state.
type SlotSnapshot struct {
	ID         SlotID
	Label      string
	Color      string
	Connected  bool
	Generation uint64
	Feedback   *protocol.PlayerFeedbackData
}

// State is a snapshot of the public session state.
type State struct {
	Phase           string
	RoundID         int64
	TimeLeftMs      int64
	LastRoundResult *protocol.RoundResultData
}

// ActionKind identifies a timer-driven session action.
type ActionKind string

const (
	ActionControllerTimeout    ActionKind = "controller_timeout"
	ActionPauseAllDisconnected ActionKind = "pause_all_disconnected"
)

// Action is returned by Tick.
type Action struct {
	Kind       ActionKind
	Slot       SlotID
	Generation uint64
}

// Session errors map to protocol error codes.
var (
	ErrInvalidToken    = protocol.NewError(protocol.CodeAuthFailed, "Invalid device token.")
	ErrDuplicateToken  = protocol.NewError(protocol.CodeAuthFailed, "Device token is already connected.")
	ErrSessionFull     = protocol.NewError(protocol.CodeSessionFull, "No free player slot.")
	ErrKickedToken     = protocol.NewError(protocol.CodeKicked, "This device was disabled by the host.")
	ErrSlotNotAssigned = protocol.NewError(protocol.CodeInvalidInput, "Slot is not assigned.")
	ErrWrongPhase      = protocol.NewError(protocol.CodeInvalidInput, "Input is not allowed in the current phase.")
	ErrStaleInput      = protocol.NewError(protocol.CodeInvalidInput, "Input sequence is stale.")
	ErrInvalidRange    = protocol.NewError(protocol.CodeInvalidInput, "Input value is out of range.")
	ErrRateLimited     = protocol.NewError(protocol.CodeRateLimited, "Input rate exceeded.")
	ErrInvalidPhase    = protocol.NewError(protocol.CodeInternalError, "Unknown phase.")
	ErrInvalidCommand  = protocol.NewError(protocol.CodeNotAuthorized, "Unknown host command.")
	ErrSessionEnded    = protocol.NewError(protocol.CodeAuthFailed, "Session has ended. Restart the server.")
)

type slotState struct {
	def        SlotDef
	token      string
	connected  bool
	generation uint64
	lastSeen   time.Time
	lastSeq    int64
	hasSeq     bool
	feedback   *protocol.PlayerFeedbackData
	limiter    rateLimiter
	errors     errorCounter
}

type rateLimiter struct {
	tokens float64
	last   time.Time
}

func (l *rateLimiter) reset() {
	l.tokens = 0
	l.last = time.Time{}
}

func (l *rateLimiter) allow(now time.Time, perSecond, burst float64) bool {
	if burst < 1 {
		burst = 1
	}
	if perSecond <= 0 {
		return true
	}
	if l.last.IsZero() {
		l.tokens = burst
		l.last = now
	}
	elapsed := now.Sub(l.last).Seconds()
	if elapsed > 0 {
		l.tokens = math.Min(burst, l.tokens+elapsed*perSecond)
		l.last = now
	}
	if l.tokens < 1 {
		return false
	}
	l.tokens--
	return true
}

type errorCounter struct {
	windowStart time.Time
	invalid     int
	rate        int
}

func (c *errorCounter) reset() {
	c.windowStart = time.Time{}
	c.invalid = 0
	c.rate = 0
}

func (c *errorCounter) bump(now time.Time, rate bool, window time.Duration) (invalid, rateCount int) {
	if c.windowStart.IsZero() || now.Sub(c.windowStart) > window {
		c.windowStart = now
		c.invalid = 0
		c.rate = 0
	}
	if rate {
		c.rate++
	} else {
		c.invalid++
	}
	return c.invalid, c.rate
}

// Session is the server's in-memory session state. All methods are safe for
// concurrent use. Network I/O must happen outside the lock.
type Session struct {
	mu sync.Mutex

	slots         map[SlotID]*slotState
	order         []SlotID
	tokenToSlot   map[string]SlotID
	blockedTokens map[string]struct{}
	phase         string
	roundID       int64
	timeLeftMs    int64
	lastResult    *protocol.RoundResultData
	timings       Timings

	allDisconnectedSince time.Time
}

// New creates a session in Attract phase.
func New(opts Options) *Session {
	if len(opts.Slots) == 0 {
		opts.Slots = []SlotDef{
			{ID: 1, Label: "P1", Color: "blue"},
			{ID: 2, Label: "P2", Color: "red"},
			{ID: 3, Label: "P3", Color: "green"},
			{ID: 4, Label: "P4", Color: "yellow"},
		}
	}
	if opts.MaxSlots <= 0 || opts.MaxSlots > len(opts.Slots) {
		opts.MaxSlots = len(opts.Slots)
	}
	t := opts.Timings
	if t.ControllerIdleTimeout <= 0 {
		t = DefaultTimings()
	}
	s := &Session{
		slots:         make(map[SlotID]*slotState, opts.MaxSlots),
		tokenToSlot:   make(map[string]SlotID, opts.MaxSlots),
		blockedTokens: make(map[string]struct{}),
		phase:         protocol.PhaseAttract,
		timings:       t,
	}
	for i := 0; i < opts.MaxSlots; i++ {
		def := opts.Slots[i]
		def.ID = SlotID(i + 1)
		s.slots[def.ID] = &slotState{def: def}
		s.order = append(s.order, def.ID)
	}
	return s
}

// Attach assigns or reclaims a slot for token. The returned bool is true when
// an existing reserved slot was reclaimed.
func (s *Session) Attach(token string, now time.Time) (SlotSnapshot, bool, error) {
	if token == "" {
		return SlotSnapshot{}, false, ErrInvalidToken
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.phase == protocol.PhaseEnded {
		return SlotSnapshot{}, false, ErrSessionEnded
	}
	if _, blocked := s.blockedTokens[token]; blocked {
		return SlotSnapshot{}, false, ErrKickedToken
	}
	if id, ok := s.tokenToSlot[token]; ok {
		slot := s.slots[id]
		if slot.connected {
			return SlotSnapshot{}, false, ErrDuplicateToken
		}
		slot.connected = true
		slot.generation++
		slot.lastSeen = now
		slot.lastSeq = 0
		slot.hasSeq = false
		slot.limiter.reset()
		slot.errors.reset()
		s.updateAllDisconnectedLocked(now)
		return snapshotOf(slot), true, nil
	}

	var free *slotState
	for _, id := range s.order {
		slot := s.slots[id]
		if slot.token == "" {
			free = slot
			break
		}
	}
	if free == nil {
		return SlotSnapshot{}, false, ErrSessionFull
	}
	free.token = token
	free.connected = true
	free.generation++
	free.lastSeen = now
	free.lastSeq = 0
	free.hasSeq = false
	free.feedback = nil
	free.limiter.reset()
	free.errors.reset()
	s.tokenToSlot[token] = free.def.ID
	s.updateAllDisconnectedLocked(now)
	return snapshotOf(free), false, nil
}

// Detach marks a slot disconnected but keeps its token reserved. It returns
// false when the slot was already disconnected, unassigned, or the caller's
// attach generation is stale (a newer connection owns the slot).
func (s *Session) Detach(id SlotID, generation uint64, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	slot, ok := s.slots[id]
	if !ok || !slot.connected || slot.generation != generation {
		return false
	}
	slot.connected = false
	slot.lastSeq = 0
	slot.hasSeq = false
	slot.limiter.reset()
	slot.errors.reset()
	s.updateAllDisconnectedLocked(now)
	return true
}

// Touch records controller liveness for the current attach generation.
func (s *Session) Touch(id SlotID, generation uint64, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if slot, ok := s.slots[id]; ok && slot.connected && slot.generation == generation {
		slot.lastSeen = now
	}
}

// AcceptInput validates an input message and returns the payload to forward to
// the game. Stale input is reported with ErrStaleInput and must be dropped
// silently by the transport.
func (s *Session) AcceptInput(id SlotID, generation uint64, seq int64, steer, throttle float64, now time.Time) (protocol.GameInputData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	slot, ok := s.slots[id]
	if !ok || !slot.connected || slot.token == "" || slot.generation != generation {
		return protocol.GameInputData{}, ErrSlotNotAssigned
	}
	if s.phase != protocol.PhasePlaying {
		return protocol.GameInputData{}, ErrWrongPhase
	}
	if seq < 0 {
		return protocol.GameInputData{}, ErrInvalidRange
	}
	if slot.hasSeq && seq <= slot.lastSeq {
		return protocol.GameInputData{}, ErrStaleInput
	}
	if math.IsNaN(steer) || math.IsInf(steer, 0) || steer < -1 || steer > 1 ||
		math.IsNaN(throttle) || math.IsInf(throttle, 0) || throttle < -1 || throttle > 1 {
		return protocol.GameInputData{}, ErrInvalidRange
	}
	if !slot.limiter.allow(now, s.timings.InputRatePerSecond, s.timings.InputBurst) {
		return protocol.GameInputData{}, ErrRateLimited
	}
	slot.lastSeq = seq
	slot.hasSeq = true
	slot.lastSeen = now
	return protocol.GameInputData{Slot: int(id), Seq: seq, Steer: steer, Throttle: throttle}, nil
}

// RecordInvalid increments the per-connection invalid-message counter and
// reports whether the connection should be closed.
func (s *Session) RecordInvalid(id SlotID, generation uint64, now time.Time) bool {
	return s.recordError(id, generation, now, false)
}

// RecordRateLimit increments the per-connection rate-error counter and reports
// whether the connection should be closed.
func (s *Session) RecordRateLimit(id SlotID, generation uint64, now time.Time) bool {
	return s.recordError(id, generation, now, true)
}

func (s *Session) recordError(id SlotID, generation uint64, now time.Time, rate bool) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	slot, ok := s.slots[id]
	if !ok || !slot.connected || slot.generation != generation {
		return false
	}
	invalid, rateCount := slot.errors.bump(now, rate, s.timings.ErrorWindow)
	if rateCount >= s.timings.RateErrorLimit || invalid >= s.timings.InvalidErrorLimit {
		return true
	}
	return false
}

// SetPhase mirrors the game's authoritative phase and updates cached state.
func (s *Session) SetPhase(phase string, roundID, timeLeftMs int64, now time.Time) error {
	if !protocol.ValidPhase(phase) {
		return ErrInvalidPhase
	}
	if timeLeftMs < 0 {
		timeLeftMs = 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.phase == protocol.PhaseEnded {
		return ErrSessionEnded
	}
	if roundID != s.roundID {
		s.clearFeedbackLocked()
		s.lastResult = nil
	}
	if phase == protocol.PhaseEnded {
		s.clearFeedbackLocked()
	}
	s.phase = phase
	s.roundID = roundID
	s.timeLeftMs = timeLeftMs
	s.updateAllDisconnectedLocked(now)
	return nil
}

// SetFeedback caches the latest per-player feedback. It is kept while the slot
// is reserved so a reconnect can receive the current task and score.
func (s *Session) SetFeedback(id SlotID, feedback protocol.PlayerFeedbackData) {
	s.mu.Lock()
	defer s.mu.Unlock()
	slot, ok := s.slots[id]
	if !ok || slot.token == "" {
		return
	}
	copy := feedback
	copy.Slot = int(id)
	if feedback.Task != nil {
		task := *feedback.Task
		copy.Task = &task
	}
	slot.feedback = &copy
}

// UpdateScore updates the cached score after a task_result.
func (s *Session) UpdateScore(id SlotID, total int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	slot, ok := s.slots[id]
	if !ok || slot.token == "" {
		return
	}
	if slot.feedback == nil {
		slot.feedback = &protocol.PlayerFeedbackData{Slot: int(id), State: protocol.StatePlaying}
	}
	slot.feedback.Score = total
}

// SetRoundResult caches the public result for late reconnects.
func (s *Session) SetRoundResult(result protocol.RoundResultData) {
	s.mu.Lock()
	defer s.mu.Unlock()
	copy := result
	copy.Scores = append([]protocol.RoundScore(nil), result.Scores...)
	s.lastResult = &copy
}

// ResetRound abandons the current round and returns to Ready.
func (s *Session) ResetRound(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.phase == protocol.PhaseEnded {
		return
	}
	s.phase = protocol.PhaseReady
	s.roundID = 0
	s.timeLeftMs = 0
	s.lastResult = nil
	s.clearFeedbackLocked()
	s.allDisconnectedSince = time.Time{}
	_ = now
}

// ResetForGame prepares the session after the game process starts or restarts.
func (s *Session) ResetForGame(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.phase == protocol.PhaseEnded {
		return
	}
	s.phase = protocol.PhaseAttract
	s.roundID = 0
	s.timeLeftMs = 0
	s.lastResult = nil
	s.clearFeedbackLocked()
	s.allDisconnectedSince = time.Time{}
	_ = now
}

// End ends the session and clears slots, tokens, and round state.
func (s *Session) End() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.phase = protocol.PhaseEnded
	s.roundID = 0
	s.timeLeftMs = 0
	s.lastResult = nil
	s.tokenToSlot = make(map[string]SlotID)
	s.blockedTokens = make(map[string]struct{})
	s.clearSlotsLocked()
	s.allDisconnectedSince = time.Time{}
}

// AuthorizeHost validates a host request against the current phase and slots.
func (s *Session) AuthorizeHost(command string, slot *SlotID) error {
	if !protocol.ValidHostCommand(command) {
		return ErrInvalidCommand
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	switch command {
	case protocol.HostStart:
		if s.connectedCountLocked() == 0 {
			return protocol.NewError(protocol.CodeNoPlayers, "At least one controller must be connected.")
		}
		switch s.phase {
		case protocol.PhaseAttract, protocol.PhaseReady, protocol.PhaseResults:
			return nil
		default:
			return protocol.NewError(protocol.CodeInvalidPhase, "The round cannot start in the current phase.")
		}
	case protocol.HostPause:
		if s.phase != protocol.PhasePlaying {
			return protocol.NewError(protocol.CodeInvalidPhase, "Only a playing round can be paused.")
		}
		return nil
	case protocol.HostResume:
		if s.phase != protocol.PhasePaused {
			return protocol.NewError(protocol.CodeInvalidPhase, "Only a paused round can be resumed.")
		}
		return nil
	case protocol.HostReset:
		switch s.phase {
		case protocol.PhaseCountdown, protocol.PhasePlaying, protocol.PhasePaused, protocol.PhaseResults:
			return nil
		default:
			return protocol.NewError(protocol.CodeInvalidPhase, "There is no round to reset.")
		}
	case protocol.HostEnd:
		if s.phase == protocol.PhaseEnded {
			return protocol.NewError(protocol.CodeInvalidPhase, "The session has already ended.")
		}
		return nil
	case protocol.HostKick:
		if slot == nil || !slot.Valid() {
			return protocol.NewError(protocol.CodeNotAuthorized, "A valid slot is required.")
		}
		if st, ok := s.slots[*slot]; !ok || st.token == "" {
			return protocol.NewError(protocol.CodeNotAuthorized, "That slot is not assigned.")
		}
		return nil
	default:
		return ErrInvalidCommand
	}
}

// Kick clears a slot and blocks its device token for the rest of the session.
func (s *Session) Kick(id SlotID, now time.Time) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	slot, ok := s.slots[id]
	if !ok || slot.token == "" {
		return "", false
	}
	token := slot.token
	delete(s.tokenToSlot, token)
	s.blockedTokens[token] = struct{}{}
	slot.token = ""
	slot.connected = false
	slot.generation++
	slot.feedback = nil
	slot.lastSeq = 0
	slot.hasSeq = false
	slot.limiter.reset()
	slot.errors.reset()
	s.updateAllDisconnectedLocked(now)
	return token, true
}

// Tick enforces controller idle timeouts and the all-disconnected grace timer.
func (s *Session) Tick(now time.Time) []Action {
	s.mu.Lock()
	defer s.mu.Unlock()
	var actions []Action
	for _, id := range s.order {
		slot := s.slots[id]
		if slot.connected && now.Sub(slot.lastSeen) > s.timings.ControllerIdleTimeout {
			slot.connected = false
			slot.generation++
			slot.lastSeq = 0
			slot.hasSeq = false
			slot.limiter.reset()
			slot.errors.reset()
			actions = append(actions, Action{Kind: ActionControllerTimeout, Slot: id, Generation: slot.generation})
		}
	}
	if s.connectedCountLocked() == 0 &&
		(s.phase == protocol.PhaseCountdown || s.phase == protocol.PhasePlaying) {
		if s.allDisconnectedSince.IsZero() {
			s.allDisconnectedSince = now
		} else if now.Sub(s.allDisconnectedSince) >= s.timings.AllDisconnectedGrace {
			s.phase = protocol.PhasePaused
			s.allDisconnectedSince = time.Time{}
			actions = append(actions, Action{Kind: ActionPauseAllDisconnected})
		}
	} else {
		s.allDisconnectedSince = time.Time{}
	}
	return actions
}

// State returns a copy of the public session state.
func (s *Session) State() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return State{
		Phase:           s.phase,
		RoundID:         s.roundID,
		TimeLeftMs:      s.timeLeftMs,
		LastRoundResult: cloneRoundResult(s.lastResult),
	}
}

// SnapshotForSlot returns one slot's public state.
func (s *Session) SnapshotForSlot(id SlotID) (SlotSnapshot, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	slot, ok := s.slots[id]
	if !ok {
		return SlotSnapshot{}, false
	}
	return snapshotOf(slot), true
}

// ConnectedSlots returns connected slots in slot order.
func (s *Session) ConnectedSlots() []SlotSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]SlotSnapshot, 0, len(s.order))
	for _, id := range s.order {
		slot := s.slots[id]
		if slot.connected {
			out = append(out, snapshotOf(slot))
		}
	}
	return out
}

// AssignedSlots returns every slot with a reserved token in slot order.
func (s *Session) AssignedSlots() []SlotSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]SlotSnapshot, 0, len(s.order))
	for _, id := range s.order {
		slot := s.slots[id]
		if slot.token != "" {
			out = append(out, snapshotOf(slot))
		}
	}
	return out
}

// ConnectedCount returns the number of connected controllers.
func (s *Session) ConnectedCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.connectedCountLocked()
}

// SlotDefinitions returns the configured slot labels and colours.
func (s *Session) SlotDefinitions() []SlotDef {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]SlotDef, 0, len(s.order))
	for _, id := range s.order {
		out = append(out, s.slots[id].def)
	}
	return out
}

func (s *Session) connectedCountLocked() int {
	count := 0
	for _, id := range s.order {
		if s.slots[id].connected {
			count++
		}
	}
	return count
}

func (s *Session) updateAllDisconnectedLocked(now time.Time) {
	if s.connectedCountLocked() == 0 &&
		(s.phase == protocol.PhaseCountdown || s.phase == protocol.PhasePlaying) {
		if s.allDisconnectedSince.IsZero() {
			s.allDisconnectedSince = now
		}
		return
	}
	s.allDisconnectedSince = time.Time{}
}

func (s *Session) clearFeedbackLocked() {
	for _, slot := range s.slots {
		slot.feedback = nil
	}
}

func (s *Session) clearSlotsLocked() {
	for _, slot := range s.slots {
		slot.token = ""
		slot.connected = false
		slot.generation++
		slot.lastSeen = time.Time{}
		slot.lastSeq = 0
		slot.hasSeq = false
		slot.feedback = nil
		slot.limiter.reset()
		slot.errors.reset()
	}
}

func snapshotOf(slot *slotState) SlotSnapshot {
	snap := SlotSnapshot{
		ID:         slot.def.ID,
		Label:      slot.def.Label,
		Color:      slot.def.Color,
		Connected:  slot.connected,
		Generation: slot.generation,
	}
	if slot.feedback != nil {
		copy := *slot.feedback
		if slot.feedback.Task != nil {
			task := *slot.feedback.Task
			copy.Task = &task
		}
		snap.Feedback = &copy
	}
	return snap
}

func cloneRoundResult(result *protocol.RoundResultData) *protocol.RoundResultData {
	if result == nil {
		return nil
	}
	copy := *result
	copy.Scores = append([]protocol.RoundScore(nil), result.Scores...)
	return &copy
}
