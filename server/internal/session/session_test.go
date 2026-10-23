package session

import (
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/PSZLBit/ProszkowskaDrift/server/internal/protocol"
)

type fakeClock struct {
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.now = c.now.Add(d)
}

func newTestSession(t *testing.T) (*Session, *fakeClock) {
	t.Helper()
	clock := &fakeClock{now: time.Unix(1000, 0)}
	return New(Options{Clock: clock.Now}), clock
}

func hello(t *testing.T, s *Session, token string, connectionID uint64) (SlotView, bool) {
	t.Helper()
	slot, reconnected, err := s.Hello(token, connectionID)
	if err != nil {
		t.Fatalf("Hello(%q, %d): %v", token, connectionID, err)
	}
	return slot, reconnected
}

func TestHelloAssignsSlotsInOrder(t *testing.T) {
	t.Parallel()

	s, _ := newTestSession(t)

	want := []struct {
		slot  int
		label string
		color string
	}{
		{slot: 1, label: "P1", color: "red"},
		{slot: 2, label: "P2", color: "blue"},
		{slot: 3, label: "P3", color: "green"},
		{slot: 4, label: "P4", color: "yellow"},
	}

	for i, tt := range want {
		got, reconnected := hello(t, s, "token-"+tt.label, uint64(i+1))
		if reconnected {
			t.Fatalf("slot %d: reconnected = true, want false", tt.slot)
		}
		if got.Number != tt.slot || got.Label != tt.label || got.Color != tt.color {
			t.Fatalf("slot = %+v, want number=%d label=%s color=%s", got, tt.slot, tt.label, tt.color)
		}
		if !got.Assigned || !got.Connected {
			t.Fatalf("slot = %+v, want assigned and connected", got)
		}
	}

	if _, _, err := s.Hello("token-extra", 5); !errors.Is(err, ErrSessionFull) {
		t.Fatalf("fifth hello error = %v, want %v", err, ErrSessionFull)
	}
}

func TestHelloRejectsInvalidToken(t *testing.T) {
	t.Parallel()

	tokens := []string{
		"",
		"short",
		strings.Repeat("a", 129),
		"has space",
		"has\ttab",
		"zażółć",
	}
	for _, token := range tokens {
		token := token
		t.Run(token, func(t *testing.T) {
			t.Parallel()

			s, _ := newTestSession(t)
			if _, _, err := s.Hello(token, 1); !errors.Is(err, ErrInvalidToken) {
				t.Fatalf("Hello(%q) error = %v, want %v", token, err, ErrInvalidToken)
			}
		})
	}
}

func TestHelloRejectsDuplicateActiveToken(t *testing.T) {
	t.Parallel()

	s, _ := newTestSession(t)
	first, _ := hello(t, s, "token-dup", 1)

	if _, _, err := s.Hello("token-dup", 2); !errors.Is(err, ErrDuplicateToken) {
		t.Fatalf("duplicate hello error = %v, want %v", err, ErrDuplicateToken)
	}

	slots := s.Slots()
	if slots[first.Number-1].Connected != true {
		t.Fatal("first connection should remain connected")
	}
}

func TestReconnectReclaimsSlot(t *testing.T) {
	t.Parallel()

	s, _ := newTestSession(t)
	first, _ := hello(t, s, "token-reconnect", 10)

	if slot, changed := s.Disconnect(first.Number, 10); !changed {
		t.Fatalf("Disconnect returned changed=false for slot %d", slot.Number)
	}
	if _, changed := s.Disconnect(first.Number, 99); changed {
		t.Fatal("Disconnect with the wrong connection id should not change the slot")
	}

	reclaimed, reconnected := hello(t, s, "token-reconnect", 11)
	if !reconnected {
		t.Fatal("reconnected = false, want true")
	}
	if reclaimed.Number != first.Number {
		t.Fatalf("slot = %d, want %d", reclaimed.Number, first.Number)
	}
	if !reclaimed.Connected {
		t.Fatal("reclaimed slot should be connected")
	}

	if _, changed := s.Disconnect(reclaimed.Number, 10); changed {
		t.Fatal("the old connection must not disconnect the reclaimed slot")
	}
	if _, changed := s.Disconnect(reclaimed.Number, 11); !changed {
		t.Fatal("the new connection should disconnect the slot")
	}
}

func TestUnknownTokenClaimsFreeSlot(t *testing.T) {
	t.Parallel()

	s, _ := newTestSession(t)
	first, _ := hello(t, s, "token-first", 1)
	second, _ := hello(t, s, "token-second", 2)

	if first.Number == second.Number {
		t.Fatal("two tokens received the same slot")
	}
	if second.Number != 2 {
		t.Fatalf("second slot = %d, want 2", second.Number)
	}
}

func TestLeaveKeepsSlotReserved(t *testing.T) {
	t.Parallel()

	s, _ := newTestSession(t)
	first, _ := hello(t, s, "token-leave", 1)

	slot, changed := s.Leave(first.Number, 1)
	if !changed {
		t.Fatal("Leave returned changed=false")
	}
	if slot.Connected {
		t.Fatal("slot should be disconnected after Leave")
	}
	if !slot.Assigned {
		t.Fatal("slot should stay assigned after Leave")
	}

	reclaimed, reconnected := hello(t, s, "token-leave", 2)
	if !reconnected || reclaimed.Number != first.Number {
		t.Fatalf("reclaimed = %+v, reconnected=%v", reclaimed, reconnected)
	}
}

func TestKickClearsSlot(t *testing.T) {
	t.Parallel()

	s, _ := newTestSession(t)
	first, _ := hello(t, s, "token-kick", 1)
	_, _ = hello(t, s, "token-other", 2)

	result, err := s.HostCommand(CommandKick, 2)
	if err != nil {
		t.Fatalf("HostCommand(kick): %v", err)
	}
	if !result.Kicked || result.Slot != 2 {
		t.Fatalf("result = %+v, want kicked slot 2", result)
	}

	slots := s.Slots()
	if slots[1].Assigned || slots[1].Connected {
		t.Fatalf("kicked slot = %+v, want free", slots[1])
	}

	if _, _, err := s.Hello("token-other", 3); !errors.Is(err, ErrKickedToken) {
		t.Fatalf("kicked token hello error = %v, want %v", err, ErrKickedToken)
	}

	replacement, _ := hello(t, s, "token-replacement", 4)
	if replacement.Number != 2 {
		t.Fatalf("replacement slot = %d, want 2", replacement.Number)
	}

	if _, changed := s.Disconnect(first.Number, 1); !changed {
		t.Fatal("kicking another slot should not affect the first slot")
	}
}

func TestHostCommandTransitions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		connect   bool
		setup     []Command
		command   Command
		wantPhase protocol.Phase
		wantRound int64
		wantErr   error
	}{
		{
			name:      "start with no controllers is rejected",
			command:   CommandStart,
			wantPhase: protocol.PhaseAttract,
			wantRound: 0,
			wantErr:   ErrNoPlayers,
		},
		{
			name:      "start from attract",
			connect:   true,
			command:   CommandStart,
			wantPhase: protocol.PhasePlaying,
			wantRound: 1,
		},
		{
			name:      "start twice is rejected",
			connect:   true,
			setup:     []Command{CommandStart},
			command:   CommandStart,
			wantPhase: protocol.PhasePlaying,
			wantRound: 1,
			wantErr:   ErrInvalidTransition,
		},
		{
			name:      "pause from playing",
			connect:   true,
			setup:     []Command{CommandStart},
			command:   CommandPause,
			wantPhase: protocol.PhasePaused,
			wantRound: 1,
		},
		{
			name:      "pause from attract is rejected",
			connect:   true,
			command:   CommandPause,
			wantPhase: protocol.PhaseAttract,
			wantErr:   ErrInvalidTransition,
		},
		{
			name:      "resume from paused",
			connect:   true,
			setup:     []Command{CommandStart, CommandPause},
			command:   CommandResume,
			wantPhase: protocol.PhasePlaying,
			wantRound: 1,
		},
		{
			name:      "resume from playing is rejected",
			connect:   true,
			setup:     []Command{CommandStart},
			command:   CommandResume,
			wantPhase: protocol.PhasePlaying,
			wantRound: 1,
			wantErr:   ErrInvalidTransition,
		},
		{
			name:      "reset returns to ready",
			connect:   true,
			setup:     []Command{CommandStart},
			command:   CommandReset,
			wantPhase: protocol.PhaseReady,
			wantRound: 0,
		},
		{
			name:      "reset from attract is rejected",
			command:   CommandReset,
			wantPhase: protocol.PhaseAttract,
			wantErr:   ErrInvalidTransition,
		},
		{
			name:      "end from ready",
			connect:   true,
			setup:     []Command{CommandStart, CommandReset},
			command:   CommandEnd,
			wantPhase: protocol.PhaseEnded,
			wantRound: 0,
		},
		{
			name:      "start after reset increments round",
			connect:   true,
			setup:     []Command{CommandStart, CommandReset},
			command:   CommandStart,
			wantPhase: protocol.PhasePlaying,
			wantRound: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s, _ := newTestSession(t)
			if tt.connect {
				hello(t, s, "token-host", 1)
			}
			for _, cmd := range tt.setup {
				if _, err := s.HostCommand(cmd, 0); err != nil {
					t.Fatalf("setup command %q: %v", cmd, err)
				}
			}

			_, err := s.HostCommand(tt.command, 0)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("error = %v, want %v", err, tt.wantErr)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got := s.Phase(); got != tt.wantPhase {
				t.Fatalf("phase = %q, want %q", got, tt.wantPhase)
			}
			if got := s.RoundID(); got != tt.wantRound {
				t.Fatalf("round = %d, want %d", got, tt.wantRound)
			}
		})
	}
}

func TestHostCommandRejectsUnknownAndInvalidKick(t *testing.T) {
	t.Parallel()

	s, _ := newTestSession(t)

	if _, err := s.HostCommand(Command("dance"), 0); !errors.Is(err, ErrInvalidCommand) {
		t.Fatalf("unknown command error = %v, want %v", err, ErrInvalidCommand)
	}
	for _, slot := range []int{0, 5, -1} {
		if _, err := s.HostCommand(CommandKick, slot); !errors.Is(err, ErrInvalidSlot) {
			t.Fatalf("kick slot %d error = %v, want %v", slot, err, ErrInvalidSlot)
		}
	}
}

func TestSetPhase(t *testing.T) {
	t.Parallel()

	s, _ := newTestSession(t)

	if err := s.SetPhase(protocol.Phase("flying"), 0); !errors.Is(err, ErrInvalidPhase) {
		t.Fatalf("invalid phase error = %v, want %v", err, ErrInvalidPhase)
	}
	if err := s.SetPhase(protocol.PhaseCountdown, 7); err != nil {
		t.Fatalf("SetPhase: %v", err)
	}
	if got := s.Phase(); got != protocol.PhaseCountdown {
		t.Fatalf("phase = %q, want countdown", got)
	}
	if got := s.RoundID(); got != 7 {
		t.Fatalf("round = %d, want 7", got)
	}
}

func TestValidateInput(t *testing.T) {
	t.Parallel()

	s, _ := newTestSession(t)
	slot, _ := hello(t, s, "token-input", 1)

	if _, err := s.ValidateInput(slot.Number, 1, 1, 0, 0); !errors.Is(err, ErrNotPlaying) {
		t.Fatalf("input before start error = %v, want %v", err, ErrNotPlaying)
	}

	if _, err := s.HostCommand(CommandStart, 0); err != nil {
		t.Fatalf("start: %v", err)
	}

	got, err := s.ValidateInput(slot.Number, 1, 10, -0.5, 1)
	if err != nil {
		t.Fatalf("valid input: %v", err)
	}
	if got.Slot != 1 || got.Seq != 10 || got.Steer != -0.5 || got.Throttle != 1 {
		t.Fatalf("input = %+v", got)
	}

	if _, err := s.ValidateInput(slot.Number, 1, 10, 0, 0); !errors.Is(err, ErrStaleSequence) {
		t.Fatalf("equal seq error = %v, want %v", err, ErrStaleSequence)
	}
	if _, err := s.ValidateInput(slot.Number, 1, 9, 0, 0); !errors.Is(err, ErrStaleSequence) {
		t.Fatalf("older seq error = %v, want %v", err, ErrStaleSequence)
	}
	if _, err := s.ValidateInput(slot.Number, 1, 11, 1.5, 0); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("out-of-range error = %v, want %v", err, ErrInvalidInput)
	}
	if _, err := s.ValidateInput(slot.Number, 1, 11, math.NaN(), 0); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("NaN error = %v, want %v", err, ErrInvalidInput)
	}
	if _, err := s.ValidateInput(slot.Number, 1, 11, 0, math.Inf(1)); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("Inf error = %v, want %v", err, ErrInvalidInput)
	}

	if _, err := s.ValidateInput(2, 1, 1, 0, 0); !errors.Is(err, ErrNotAssigned) {
		t.Fatalf("unassigned slot error = %v, want %v", err, ErrNotAssigned)
	}
	if _, err := s.ValidateInput(slot.Number, 99, 11, 0, 0); !errors.Is(err, ErrWrongConnection) {
		t.Fatalf("wrong connection error = %v, want %v", err, ErrWrongConnection)
	}

	if _, err := s.HostCommand(CommandPause, 0); err != nil {
		t.Fatalf("pause: %v", err)
	}
	if _, err := s.ValidateInput(slot.Number, 1, 12, 0, 0); !errors.Is(err, ErrNotPlaying) {
		t.Fatalf("input while paused error = %v, want %v", err, ErrNotPlaying)
	}
}

func TestRateLimitPerSlot(t *testing.T) {
	t.Parallel()

	clock := &fakeClock{now: time.Unix(1000, 0)}
	s := New(Options{Clock: clock.Now, RateLimit: 3, RateWindow: time.Second})
	slot, _ := hello(t, s, "token-rate", 1)
	if _, err := s.HostCommand(CommandStart, 0); err != nil {
		t.Fatalf("start: %v", err)
	}

	for seq := int64(1); seq <= 3; seq++ {
		if _, err := s.ValidateInput(slot.Number, 1, seq, 0, 0); err != nil {
			t.Fatalf("input %d: %v", seq, err)
		}
	}
	if _, err := s.ValidateInput(slot.Number, 1, 4, 0, 0); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("fourth input error = %v, want %v", err, ErrRateLimited)
	}

	clock.Advance(time.Second)
	if _, err := s.ValidateInput(slot.Number, 1, 5, 0, 0); err != nil {
		t.Fatalf("input after window: %v", err)
	}
}

func TestReconnectResetsSequenceButNotRate(t *testing.T) {
	t.Parallel()

	clock := &fakeClock{now: time.Unix(1000, 0)}
	s := New(Options{Clock: clock.Now, RateLimit: 2, RateWindow: time.Second})
	slot, _ := hello(t, s, "token-reset", 1)
	if _, err := s.HostCommand(CommandStart, 0); err != nil {
		t.Fatalf("start: %v", err)
	}

	for seq := int64(100); seq <= 101; seq++ {
		if _, err := s.ValidateInput(slot.Number, 1, seq, 0, 0); err != nil {
			t.Fatalf("input %d: %v", seq, err)
		}
	}
	s.Disconnect(slot.Number, 1)
	reclaimed, _ := hello(t, s, "token-reset", 2)
	if reclaimed.Number != slot.Number {
		t.Fatalf("slot = %d, want %d", reclaimed.Number, slot.Number)
	}

	if _, err := s.ValidateInput(slot.Number, 2, 1, 0, 0); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("input after reconnect error = %v, want rate limited", err)
	}

	clock.Advance(time.Second)
	if _, err := s.ValidateInput(slot.Number, 2, 1, 0, 0); err != nil {
		t.Fatalf("sequence should restart after reconnect: %v", err)
	}
}

func TestInputRequiresAssignedConnection(t *testing.T) {
	t.Parallel()

	s, _ := newTestSession(t)
	slot, _ := hello(t, s, "token-disconnect", 1)
	if _, err := s.HostCommand(CommandStart, 0); err != nil {
		t.Fatalf("start: %v", err)
	}
	s.Disconnect(slot.Number, 1)

	if _, err := s.ValidateInput(slot.Number, 1, 1, 0, 0); !errors.Is(err, ErrNotAssigned) {
		t.Fatalf("input after disconnect error = %v, want %v", err, ErrNotAssigned)
	}
}

func TestTickTimesOutIdleControllers(t *testing.T) {
	t.Parallel()

	clock := &fakeClock{now: time.Unix(1000, 0)}
	s := New(Options{Clock: clock.Now, IdleTimeout: 6 * time.Second})
	slot, _ := hello(t, s, "token-idle", 7)

	if actions := s.Tick(clock.Now()); len(actions) != 0 {
		t.Fatalf("actions before timeout = %+v, want none", actions)
	}

	clock.Advance(5 * time.Second)
	s.Touch(slot.Number, 7, clock.Now())
	clock.Advance(10 * time.Second)
	actions := s.Tick(clock.Now())
	if len(actions) != 1 || actions[0].Kind != ActionControllerTimeout ||
		actions[0].Slot != slot.Number || actions[0].ConnID != 7 {
		t.Fatalf("actions = %+v, want timeout for slot %d conn 7", actions, slot.Number)
	}

	slots := s.Slots()
	if slots[0].Connected {
		t.Fatal("timed out slot must be disconnected")
	}
	if !slots[0].Assigned {
		t.Fatal("timed out slot must stay assigned")
	}

	// Touch by the stale connection must not resurrect liveness tracking.
	clock.Advance(time.Second)
	s.Touch(slot.Number, 7, clock.Now())
	if actions := s.Tick(clock.Now()); len(actions) != 0 {
		t.Fatalf("stale touch must not repeat the timeout: %+v", actions)
	}

	// A fresh connection prevents the timeout.
	reclaimed, reconnected := hello(t, s, "token-idle", 8)
	if !reconnected || reclaimed.Number != slot.Number {
		t.Fatalf("reclaimed = %+v reconnected = %v", reclaimed, reconnected)
	}
	clock.Advance(3 * time.Second)
	if actions := s.Tick(clock.Now()); len(actions) != 0 {
		t.Fatalf("connected controller must not time out early: %+v", actions)
	}
	clock.Advance(4 * time.Second)
	if actions := s.Tick(clock.Now()); len(actions) != 1 || actions[0].ConnID != 8 {
		t.Fatalf("actions = %+v, want timeout for conn 8 after idle", actions)
	}
}

func TestTickPausesRoundAfterAllDisconnected(t *testing.T) {
	t.Parallel()

	clock := &fakeClock{now: time.Unix(1000, 0)}
	s := New(Options{Clock: clock.Now, IdleTimeout: time.Minute, AllDisconnectedGrace: 10 * time.Second})
	slot, _ := hello(t, s, "token-pause", 1)
	if _, err := s.HostCommand(CommandStart, 0); err != nil {
		t.Fatalf("start: %v", err)
	}

	if err := s.SetPhase(protocol.PhaseCountdown, 1); err != nil {
		t.Fatalf("countdown: %v", err)
	}
	s.Disconnect(slot.Number, 1)

	clock.Advance(9 * time.Second)
	if actions := s.Tick(clock.Now()); len(actions) != 0 {
		t.Fatalf("actions inside grace = %+v, want none", actions)
	}

	clock.Advance(2 * time.Second)
	actions := s.Tick(clock.Now())
	if len(actions) != 1 || actions[0].Kind != ActionPauseAllDisconnected {
		t.Fatalf("actions = %+v, want pause after grace", actions)
	}
	if got := s.Phase(); got != protocol.PhasePaused {
		t.Fatalf("phase = %q, want paused", got)
	}

	// A new game phase restarts the grace timer instead of pausing instantly.
	if err := s.SetPhase(protocol.PhasePlaying, 1); err != nil {
		t.Fatalf("playing: %v", err)
	}
	clock.Advance(4 * time.Second)
	if actions := s.Tick(clock.Now()); len(actions) != 0 {
		t.Fatalf("round must not pause again inside the new grace window: %+v", actions)
	}
	if got := s.Phase(); got != protocol.PhasePlaying {
		t.Fatalf("phase = %q, want playing", got)
	}

	// A reconnecting controller stops the all-disconnected counting.
	reclaimed, reconnected := hello(t, s, "token-pause", 2)
	if !reconnected || reclaimed.Number != slot.Number {
		t.Fatalf("reclaim = %+v reconnected = %v", reclaimed, reconnected)
	}
	clock.Advance(11 * time.Second)
	if got := s.Phase(); got != protocol.PhasePlaying {
		t.Fatalf("phase = %q, want playing while a controller is connected", got)
	}
}

func TestKickOfUnassignedSlotIsRejected(t *testing.T) {
	t.Parallel()

	s, _ := newTestSession(t)
	if _, err := s.HostCommand(CommandKick, 2); !errors.Is(err, ErrInvalidSlot) {
		t.Fatalf("kick unassigned slot error = %v, want %v", err, ErrInvalidSlot)
	}
}

func TestEndClearsSlotsAndRejectsNewHellos(t *testing.T) {
	t.Parallel()

	s, _ := newTestSession(t)
	hello(t, s, "token-end", 1)
	if _, err := s.HostCommand(CommandStart, 0); err != nil {
		t.Fatalf("start: %v", err)
	}

	result, err := s.HostCommand(CommandEnd, 0)
	if err != nil {
		t.Fatalf("end: %v", err)
	}
	if result.Phase != protocol.PhaseEnded {
		t.Fatalf("phase = %q, want ended", result.Phase)
	}

	slots := s.Slots()
	for i, slot := range slots {
		if slot.Assigned {
			t.Fatalf("slot %d = %+v, want cleared", i+1, slot)
		}
	}
	if got := s.RoundID(); got != 0 {
		t.Fatalf("round = %d, want 0", got)
	}

	if _, _, err := s.Hello("token-end", 2); !errors.Is(err, ErrSessionEnded) {
		t.Fatalf("hello after end error = %v, want %v", err, ErrSessionEnded)
	}
	if _, _, err := s.Hello("token-new", 3); !errors.Is(err, ErrSessionEnded) {
		t.Fatalf("new hello after end error = %v, want %v", err, ErrSessionEnded)
	}
	if err := s.SetPhase(protocol.PhasePlaying, 1); !errors.Is(err, ErrSessionEnded) {
		t.Fatalf("phase after end error = %v, want %v", err, ErrSessionEnded)
	}
	for _, cmd := range []Command{CommandStart, CommandPause, CommandResume, CommandReset, CommandKick} {
		if _, err := s.HostCommand(cmd, 0); !errors.Is(err, ErrInvalidTransition) {
			t.Fatalf("command %q after end error = %v, want %v", cmd, err, ErrInvalidTransition)
		}
	}
}
