package session

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/PSZLBit/ProszkowskaDrift/server/internal/protocol"
)

var testNow = time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)

func testSession() *Session {
	t := DefaultTimings()
	t.ControllerIdleTimeout = 6 * time.Second
	t.AllDisconnectedGrace = 10 * time.Second
	return New(Options{Timings: t})
}

func generationOf(t *testing.T, s *Session, id SlotID) uint64 {
	t.Helper()
	snap, ok := s.SnapshotForSlot(id)
	if !ok {
		t.Fatalf("slot %d missing", id)
	}
	return snap.Generation
}

func TestAttachAssignsSlotsAndReclaims(t *testing.T) {
	s := testSession()
	tokens := []string{"t1", "t2", "t3", "t4"}
	for i, token := range tokens {
		snap, reclaimed, err := s.Attach(token, testNow)
		if err != nil {
			t.Fatalf("attach %s: %v", token, err)
		}
		if reclaimed {
			t.Fatalf("attach %s unexpectedly reclaimed", token)
		}
		if int(snap.ID) != i+1 {
			t.Fatalf("slot = %d, want %d", snap.ID, i+1)
		}
		if snap.Label != "P"+string(rune('1'+i)) {
			t.Fatalf("label = %q", snap.Label)
		}
		if snap.Color == "" {
			t.Fatal("color is empty")
		}
	}
	if _, _, err := s.Attach("t5", testNow); !errors.Is(err, ErrSessionFull) {
		t.Fatalf("fifth attach error = %v", err)
	}
	if _, _, err := s.Attach("t1", testNow); !errors.Is(err, ErrDuplicateToken) {
		t.Fatalf("duplicate attach error = %v", err)
	}
	if !s.Detach(1, generationOf(t, s, 1), testNow) {
		t.Fatal("detach failed")
	}
	snap, reclaimed, err := s.Attach("t1", testNow)
	if err != nil || !reclaimed || snap.ID != 1 {
		t.Fatalf("reclaim = %+v, %v, %v", snap, reclaimed, err)
	}
}

func TestKickBlocksTokenAndFreesSlot(t *testing.T) {
	s := testSession()
	_, _, _ = s.Attach("t1", testNow)
	token, ok := s.Kick(1, testNow)
	if !ok || token != "t1" {
		t.Fatalf("kick = %q, %v", token, ok)
	}
	if _, _, err := s.Attach("t1", testNow); !errors.Is(err, ErrKickedToken) {
		t.Fatalf("kicked token error = %v", err)
	}
	snap, _, err := s.Attach("replacement", testNow)
	if err != nil {
		t.Fatal(err)
	}
	if snap.ID != 1 {
		t.Fatalf("replacement slot = %d", snap.ID)
	}
}

func TestStaleGenerationCannotDetachNewConnection(t *testing.T) {
	s := testSession()
	first, _, _ := s.Attach("t1", testNow)
	_ = s.Detach(1, first.Generation, testNow)
	second, _, err := s.Attach("t1", testNow)
	if err != nil {
		t.Fatal(err)
	}
	if s.Detach(1, first.Generation, testNow) {
		t.Fatal("stale generation detached the new connection")
	}
	snap, ok := s.SnapshotForSlot(1)
	if !ok || !snap.Connected || snap.Generation != second.Generation {
		t.Fatalf("new connection state = %+v", snap)
	}
}

func TestInputValidation(t *testing.T) {
	s := testSession()
	_, _, _ = s.Attach("t1", testNow)
	gen := generationOf(t, s, 1)
	if err := s.SetPhase(protocol.PhasePlaying, 1, 180000, testNow); err != nil {
		t.Fatal(err)
	}

	if _, err := s.AcceptInput(1, gen, 1, 0, 0.5, testNow); err != nil {
		t.Fatalf("valid input: %v", err)
	}
	if _, err := s.AcceptInput(1, gen, 1, 0, 0.5, testNow); !errors.Is(err, ErrStaleInput) {
		t.Fatalf("stale input error = %v", err)
	}
	if _, err := s.AcceptInput(1, gen, 2, 1.1, 0, testNow); !errors.Is(err, ErrInvalidRange) {
		t.Fatalf("range error = %v", err)
	}
	if _, err := s.AcceptInput(1, gen, 3, math.NaN(), 0, testNow); !errors.Is(err, ErrInvalidRange) {
		t.Fatalf("NaN error = %v", err)
	}
	if _, err := s.AcceptInput(1, gen, 4, math.Inf(1), 0, testNow); !errors.Is(err, ErrInvalidRange) {
		t.Fatalf("Inf error = %v", err)
	}
	if _, err := s.AcceptInput(1, gen, 5, 1, -1, testNow); err != nil {
		t.Fatalf("boundary input: %v", err)
	}
	if _, err := s.AcceptInput(2, 0, 1, 0, 0, testNow); !errors.Is(err, ErrSlotNotAssigned) {
		t.Fatalf("unassigned slot error = %v", err)
	}
	if _, err := s.AcceptInput(1, gen+1, 6, 0, 0, testNow); !errors.Is(err, ErrSlotNotAssigned) {
		t.Fatalf("stale generation error = %v", err)
	}
	if err := s.SetPhase(protocol.PhasePaused, 1, 1000, testNow); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AcceptInput(1, gen, 7, 0, 0, testNow); !errors.Is(err, ErrWrongPhase) {
		t.Fatalf("wrong phase error = %v", err)
	}
}

func TestReconnectResetsSequence(t *testing.T) {
	s := testSession()
	_, _, _ = s.Attach("t1", testNow)
	firstGen := generationOf(t, s, 1)
	_ = s.SetPhase(protocol.PhasePlaying, 1, 180000, testNow)
	if _, err := s.AcceptInput(1, firstGen, 100, 0, 0, testNow); err != nil {
		t.Fatal(err)
	}
	_ = s.Detach(1, firstGen, testNow)
	snap, _, err := s.Attach("t1", testNow)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.AcceptInput(1, firstGen, 101, 0, 0, testNow); !errors.Is(err, ErrSlotNotAssigned) {
		t.Fatalf("stale generation input error = %v", err)
	}
	if _, err := s.AcceptInput(1, snap.Generation, 1, 0, 0, testNow); err != nil {
		t.Fatalf("sequence did not reset: %v", err)
	}
}

func TestInputRateLimit(t *testing.T) {
	timings := DefaultTimings()
	timings.InputRatePerSecond = 2
	timings.InputBurst = 2
	s := New(Options{Timings: timings})
	_, _, _ = s.Attach("t1", testNow)
	gen := generationOf(t, s, 1)
	_ = s.SetPhase(protocol.PhasePlaying, 1, 180000, testNow)
	for seq := int64(1); seq <= 2; seq++ {
		if _, err := s.AcceptInput(1, gen, seq, 0, 0, testNow); err != nil {
			t.Fatalf("input %d: %v", seq, err)
		}
	}
	if _, err := s.AcceptInput(1, gen, 3, 0, 0, testNow); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("rate error = %v", err)
	}
}

func TestErrorCounters(t *testing.T) {
	timings := DefaultTimings()
	timings.InvalidErrorLimit = 2
	timings.RateErrorLimit = 2
	timings.ErrorWindow = time.Second
	s := New(Options{Timings: timings})
	_, _, _ = s.Attach("t1", testNow)
	gen := generationOf(t, s, 1)
	if s.RecordInvalid(1, gen, testNow) {
		t.Fatal("closed too early")
	}
	if !s.RecordInvalid(1, gen, testNow.Add(100*time.Millisecond)) {
		t.Fatal("did not close at limit")
	}
	if s.RecordRateLimit(1, gen+1, testNow.Add(2*time.Second)) {
		t.Fatal("stale generation closed the connection")
	}
	if s.RecordRateLimit(1, gen, testNow.Add(2*time.Second)) {
		t.Fatal("old window was not reset")
	}
	if !s.RecordRateLimit(1, gen, testNow.Add(2*time.Second)) {
		t.Fatal("rate limit did not close")
	}
}

func TestTickTimeoutsAndAllDisconnectedPause(t *testing.T) {
	s := testSession()
	first, _, _ := s.Attach("t1", testNow)
	_ = s.SetPhase(protocol.PhasePlaying, 1, 180000, testNow)
	actions := s.Tick(testNow.Add(7 * time.Second))
	if len(actions) != 1 || actions[0].Kind != ActionControllerTimeout || actions[0].Slot != 1 {
		t.Fatalf("timeout actions = %+v", actions)
	}
	if actions[0].Generation == 0 || actions[0].Generation == first.Generation {
		t.Fatalf("timeout generation = %d, attach generation = %d", actions[0].Generation, first.Generation)
	}
	if s.ConnectedCount() != 0 {
		t.Fatal("slot still connected")
	}
	reconnected, _, _ := s.Attach("t1", testNow.Add(8*time.Second))
	if reconnected.Generation <= actions[0].Generation {
		t.Fatalf("reconnect generation = %d, timeout generation = %d", reconnected.Generation, actions[0].Generation)
	}

	// A fresh session: disconnect during play starts the grace timer.
	s = testSession()
	_, _, _ = s.Attach("t1", testNow)
	_ = s.SetPhase(protocol.PhasePlaying, 1, 180000, testNow)
	_ = s.Detach(1, generationOf(t, s, 1), testNow)
	if actions := s.Tick(testNow.Add(time.Second)); len(actions) != 0 {
		t.Fatalf("paused too early: %+v", actions)
	}
	actions = s.Tick(testNow.Add(11 * time.Second))
	if len(actions) != 1 || actions[0].Kind != ActionPauseAllDisconnected {
		t.Fatalf("pause actions = %+v", actions)
	}
	if s.State().Phase != protocol.PhasePaused {
		t.Fatalf("phase = %s", s.State().Phase)
	}

	// Reconnect before the grace period cancels the pause.
	s = testSession()
	_, _, _ = s.Attach("t1", testNow)
	_ = s.SetPhase(protocol.PhasePlaying, 1, 180000, testNow)
	_ = s.Detach(1, generationOf(t, s, 1), testNow)
	_, _, _ = s.Attach("t1", testNow.Add(5*time.Second))
	if actions := s.Tick(testNow.Add(11 * time.Second)); len(actions) != 0 {
		t.Fatalf("paused after reconnect: %+v", actions)
	}
}

func TestFeedbackSurvivesReconnect(t *testing.T) {
	s := testSession()
	_, _, _ = s.Attach("t1", testNow)
	task := &protocol.TaskData{TaskID: 12, SpotLabel: "B-07", SpotColor: "red", Hint: "north", DeadlineMs: 30000}
	s.SetFeedback(1, protocol.PlayerFeedbackData{Slot: 1, Task: task, Score: 240, State: protocol.StatePlaying})
	_ = s.Detach(1, generationOf(t, s, 1), testNow)
	snap, _, err := s.Attach("t1", testNow)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Feedback == nil || snap.Feedback.Task == nil || snap.Feedback.Task.TaskID != 12 {
		t.Fatalf("feedback = %+v", snap.Feedback)
	}
	task.TaskID = 99
	if snap.Feedback.Task.TaskID != 12 {
		t.Fatal("feedback was not deep-copied")
	}
	s.UpdateScore(1, 300)
	snap, _ = s.SnapshotForSlot(1)
	if snap.Feedback.Score != 300 {
		t.Fatalf("score = %d", snap.Feedback.Score)
	}
}

func TestHostAuthorisation(t *testing.T) {
	s := testSession()
	if err := s.AuthorizeHost(protocol.HostStart, nil); err == nil {
		t.Fatal("start with no players accepted")
	}
	_, _, _ = s.Attach("t1", testNow)
	if err := s.AuthorizeHost(protocol.HostStart, nil); err != nil {
		t.Fatalf("start in attract: %v", err)
	}
	if err := s.SetPhase(protocol.PhasePlaying, 1, 180000, testNow); err != nil {
		t.Fatal(err)
	}
	if err := s.AuthorizeHost(protocol.HostStart, nil); err == nil {
		t.Fatal("start while playing accepted")
	}
	if err := s.AuthorizeHost(protocol.HostPause, nil); err != nil {
		t.Fatalf("pause: %v", err)
	}
	if err := s.SetPhase(protocol.PhasePaused, 1, 1000, testNow); err != nil {
		t.Fatal(err)
	}
	if err := s.AuthorizeHost(protocol.HostResume, nil); err != nil {
		t.Fatalf("resume: %v", err)
	}
	slot := SlotID(1)
	if err := s.AuthorizeHost(protocol.HostKick, &slot); err != nil {
		t.Fatalf("kick: %v", err)
	}
	bad := SlotID(2)
	if err := s.AuthorizeHost(protocol.HostKick, &bad); err == nil {
		t.Fatal("kick of empty slot accepted")
	}
	if err := s.AuthorizeHost("explode", nil); err == nil {
		t.Fatal("unknown command accepted")
	}
}

func TestResetAndEndClearRoundState(t *testing.T) {
	s := testSession()
	_, _, _ = s.Attach("t1", testNow)
	_ = s.SetPhase(protocol.PhasePlaying, 7, 1000, testNow)
	s.SetFeedback(1, protocol.PlayerFeedbackData{Slot: 1, Score: 10, State: protocol.StatePlaying})
	s.SetRoundResult(protocol.RoundResultData{RoundID: 7, WinnerSlot: 1, Scores: []protocol.RoundScore{{Slot: 1, Score: 10}}})
	s.ResetRound(testNow)
	state := s.State()
	if state.Phase != protocol.PhaseReady || state.RoundID != 0 || state.LastRoundResult != nil {
		t.Fatalf("reset state = %+v", state)
	}
	snap, _ := s.SnapshotForSlot(1)
	if snap.Feedback != nil {
		t.Fatal("feedback was not cleared")
	}
	s.End()
	if s.State().Phase != protocol.PhaseEnded {
		t.Fatalf("phase = %s", s.State().Phase)
	}
	if s.ConnectedCount() != 0 || len(s.AssignedSlots()) != 0 {
		t.Fatalf("session end left slots assigned: connected=%d assigned=%d", s.ConnectedCount(), len(s.AssignedSlots()))
	}
	if _, _, err := s.Attach("late", testNow); !errors.Is(err, ErrSessionEnded) {
		t.Fatalf("attach after end = %v, want ErrSessionEnded", err)
	}
	s.ResetForGame(testNow)
	if err := s.SetPhase(protocol.PhaseAttract, 0, 0, testNow); !errors.Is(err, ErrSessionEnded) {
		t.Fatalf("phase after end = %v, want ErrSessionEnded", err)
	}
	if s.State().Phase != protocol.PhaseEnded {
		t.Fatalf("session reopened after end: phase = %s", s.State().Phase)
	}
}

func TestSetPhaseClearsFeedbackOnNewRoundOnly(t *testing.T) {
	s := testSession()
	_, _, _ = s.Attach("t1", testNow)
	_ = s.SetPhase(protocol.PhasePlaying, 1, 180000, testNow)
	s.SetFeedback(1, protocol.PlayerFeedbackData{Slot: 1, Score: 10, State: protocol.StatePlaying})
	_ = s.SetPhase(protocol.PhasePaused, 1, 1000, testNow)
	snap, _ := s.SnapshotForSlot(1)
	if snap.Feedback == nil {
		t.Fatal("same-round phase change cleared feedback")
	}
	_ = s.SetPhase(protocol.PhasePlaying, 2, 180000, testNow)
	snap, _ = s.SnapshotForSlot(1)
	if snap.Feedback != nil {
		t.Fatal("new round did not clear feedback")
	}
}

func TestFeedbackForUnassignedSlotIsIgnored(t *testing.T) {
	s := testSession()
	s.SetFeedback(2, protocol.PlayerFeedbackData{Slot: 2, Score: 10, State: protocol.StatePlaying})
	snap, _ := s.SnapshotForSlot(2)
	if snap.Feedback != nil {
		t.Fatal("unassigned slot accepted feedback")
	}
}

func TestSetPhaseRejectsUnknown(t *testing.T) {
	s := testSession()
	if err := s.SetPhase("dancing", 0, 0, testNow); !errors.Is(err, ErrInvalidPhase) {
		t.Fatalf("error = %v", err)
	}
}
