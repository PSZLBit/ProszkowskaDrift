package gameserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"runtime/pprof"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/PSZLBit/ProszkowskaDrift/server/internal/protocol"
	"github.com/PSZLBit/ProszkowskaDrift/server/internal/session"
)

func countWriteLoops() int {
	var buf bytes.Buffer
	_ = pprof.Lookup("goroutine").WriteTo(&buf, 1)
	return strings.Count(buf.String(), "(*wsConn).writeLoop")
}

func healthGameReady(t *testing.T, env *testEnv) bool {
	t.Helper()

	resp, err := http.Get(env.http.URL + "/healthz")
	if err != nil {
		t.Fatalf("get /healthz: %v", err)
	}
	defer resp.Body.Close()

	var body struct {
		GameReady bool `json:"gameReady"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode /healthz: %v", err)
	}
	return body.GameReady
}

func TestDisconnectStopsWriterGoroutines(t *testing.T) {
	env := newTestEnv(t, nil)
	baseline := countWriteLoops()

	for i := 0; i < 5; i++ {
		controller := env.dialController(t)
		controller.sendHello("token-writer-leak")
		controller.expectWelcome(1)
		controller.close()

		waitFor(t, func() bool {
			return !env.server.session.Slots()[0].Connected
		})
		waitFor(t, func() bool {
			return countWriteLoops() <= baseline
		})
	}

	for i := 0; i < 3; i++ {
		game := env.dialGame(t, testGameToken)
		game.readType(protocol.TypeGameHello)
		game.readType(protocol.TypeConfig)
		game.close()

		waitFor(t, func() bool {
			env.server.mu.Lock()
			defer env.server.mu.Unlock()
			return env.server.game == nil
		})
		waitFor(t, func() bool {
			return countWriteLoops() <= baseline
		})
	}
}

func TestWelcomeIsQueuedBeforeConcurrentPhase(t *testing.T) {
	s := New(DefaultConfig())
	conn := &controllerConn{
		wsConn: &wsConn{
			queue: make(chan outbound, 8),
			done:  make(chan struct{}),
		},
		id: 1,
	}

	var wg sync.WaitGroup
	wg.Add(1)
	start := make(chan struct{})
	go func() {
		defer wg.Done()
		<-start
		s.broadcastPhase(protocol.PhaseData{Phase: protocol.PhasePlaying, RoundID: 1})
	}()
	close(start)

	if !s.registerControllerAndWelcome(conn, session.SlotView{
		Number: 1,
		Label:  "P1",
		Color:  "red",
	}) {
		t.Fatal("welcome enqueue failed")
	}
	wg.Wait()

	first := <-conn.queue
	env, err := protocol.ParseEnvelope(first.data)
	if err != nil {
		t.Fatalf("parse first frame: %v", err)
	}
	if env.Type != protocol.TypeWelcome {
		t.Fatalf("first frame type = %q, want %q", env.Type, protocol.TypeWelcome)
	}
	if len(conn.queue) > 0 {
		second := <-conn.queue
		env, err = protocol.ParseEnvelope(second.data)
		if err != nil {
			t.Fatalf("parse second frame: %v", err)
		}
		if env.Type != protocol.TypePhase {
			t.Fatalf("second frame type = %q, want %q", env.Type, protocol.TypePhase)
		}
	}
}

func TestLeaveDuringGameRegistrationPreservesEventOrder(t *testing.T) {
	env := newTestEnv(t, func(cfg *Config) {
		cfg.TickInterval = time.Hour
	})

	controller := env.dialController(t)
	controller.sendHello("token-registration-race")
	controller.expectWelcome(1)

	for i := 0; i < 10; i++ {
		game := env.dialGame(t, testGameToken)
		controller.send(protocol.TypeLeave, nil, struct{}{})

		game.readType(protocol.TypeGameHello)
		game.readType(protocol.TypeConfig)
		waitFor(t, func() bool {
			return !env.server.session.Slots()[0].Connected
		})

		joined := false
		for _, event := range game.drain(50 * time.Millisecond) {
			switch event.Type {
			case protocol.TypePlayerJoined:
				if joined {
					t.Fatal("game received duplicate player_joined")
				}
				joined = true
			case protocol.TypePlayerLeft:
				if !joined {
					t.Fatal("game received player_left before player_joined")
				}
				joined = false
			}
		}
		if joined {
			t.Fatalf("game retained player_joined without a matching player_left (iteration %d)", i)
		}

		game.close()
		waitFor(t, func() bool {
			env.server.mu.Lock()
			defer env.server.mu.Unlock()
			return env.server.game == nil
		})

		controller.sendHello("token-registration-race")
		controller.expectWelcome(1)
	}
}

func TestKickTargetsCapturedConnectionAfterReplacementHello(t *testing.T) {
	s := New(DefaultConfig())
	old := &controllerConn{
		wsConn: &wsConn{
			queue: make(chan outbound, 8),
			done:  make(chan struct{}),
		},
		log: s.log,
		id:  1,
	}
	view, _, err := s.session.Hello("token-old", old.id)
	if err != nil {
		t.Fatalf("old hello: %v", err)
	}
	old.setSlot(view.Number)
	s.controllers[view.Number] = old
	captured := s.controllerForSlot(view.Number)

	// The host command frees the slot before the replacement hello can claim
	// it. The kick must still target the captured connection.
	result, err := s.session.HostCommand(session.CommandKick, view.Number)
	if err != nil || !result.Kicked {
		t.Fatalf("kick command = %+v, %v; want kicked", result, err)
	}

	replacement := &controllerConn{
		wsConn: &wsConn{
			queue: make(chan outbound, 8),
			done:  make(chan struct{}),
		},
		log: s.log,
		id:  2,
	}
	newView, _, err := s.session.Hello("token-new", replacement.id)
	if err != nil {
		t.Fatalf("replacement hello: %v", err)
	}
	replacement.setSlot(newView.Number)
	s.controllers[newView.Number] = replacement

	s.kickController(captured, view.Number)

	if s.controllers[newView.Number] != replacement {
		t.Fatal("kick removed the replacement controller")
	}
	if slots := s.session.Slots(); !slots[newView.Number-1].Connected {
		t.Fatal("replacement slot is no longer connected")
	}
	select {
	case raw := <-replacement.wsConn.queue:
		t.Fatalf("replacement received message type %q", raw.data)
	default:
	}

	select {
	case out := <-old.wsConn.queue:
		env, err := protocol.ParseEnvelope(out.data)
		if err != nil {
			t.Fatalf("parse kicked error: %v", err)
		}
		if env.Type != protocol.TypeError {
			t.Fatalf("captured connection got %q, want %q", env.Type, protocol.TypeError)
		}
	default:
		t.Fatal("captured connection did not receive the kick error")
	}

	// Let closeGraceful's goroutine exit without a real WebSocket writer.
	old.closeDoneOnce()
}

func TestSlowControllerLogReadsSlotAtomically(t *testing.T) {
	s := New(DefaultConfig())
	conn := &controllerConn{
		wsConn: &wsConn{
			queue: make(chan outbound, 1),
			done:  make(chan struct{}),
		},
		log: s.log,
		id:  1,
	}
	conn.setSlot(1)
	conn.closing.Store(true)
	s.controllers[1] = conn

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 2000; i++ {
			conn.setSlot(1 + i%4)
		}
	}()
	for i := 0; i < 2000; i++ {
		s.broadcastRaw([]byte(`{"v":1,"type":"ping"}`))
	}
	wg.Wait()
}

func TestTimedOutControllerClosedAfterReconnect(t *testing.T) {
	env := newTestEnv(t, func(cfg *Config) {
		cfg.ControllerTimeout = 30 * time.Millisecond
		cfg.TickInterval = time.Hour
	})

	first := env.dialController(t)
	first.sendHello("token-timeout-race")
	first.expectWelcome(1)

	env.server.mu.Lock()
	oldID := env.server.controllers[1].id
	env.server.mu.Unlock()

	time.Sleep(60 * time.Millisecond)
	actions := env.server.session.Tick(time.Now())
	found := false
	for _, action := range actions {
		if action.Kind == session.ActionControllerTimeout && action.ConnID == oldID {
			found = true
		}
	}
	if !found {
		t.Fatalf("session.Tick actions = %+v, want timeout for connection %d", actions, oldID)
	}

	second := env.dialController(t)
	second.sendHello("token-timeout-race")
	second.expectWelcome(1)

	env.server.closeIdleController(1, oldID)
	first.expectClosed(websocket.StatusPolicyViolation, "heartbeat_timeout")

	second.send(protocol.TypePing, nil, protocol.PingData{ClientTime: 1})
	second.readType(protocol.TypePong)
}

func TestTimedOutControllerClosedAfterLeave(t *testing.T) {
	env := newTestEnv(t, func(cfg *Config) {
		cfg.TickInterval = time.Hour
	})

	controller := env.dialController(t)
	controller.sendHello("token-timeout-leave")
	controller.expectWelcome(1)

	env.server.mu.Lock()
	oldID := env.server.controllers[1].id
	env.server.mu.Unlock()

	controller.send(protocol.TypeLeave, nil, struct{}{})
	waitFor(t, func() bool {
		return !env.server.session.Slots()[0].Connected
	})

	env.server.closeIdleController(1, oldID)
	controller.expectClosed(websocket.StatusPolicyViolation, "heartbeat_timeout")
}

func TestGameHeartbeatExpiryClosesCapturedConnection(t *testing.T) {
	env := newTestEnv(t, func(cfg *Config) {
		cfg.GameHeartbeatTimeout = 100 * time.Millisecond
		cfg.TickInterval = time.Hour
	})

	first := env.dialGame(t, testGameToken)
	first.readType(protocol.TypeGameHello)
	first.readType(protocol.TypeConfig)
	first.send(protocol.TypeGameReady, nil, protocol.GameReadyData{GameVersion: "test", MapVersion: "test"})
	waitFor(t, func() bool {
		return env.server.gameIsReady()
	})

	env.server.mu.Lock()
	old := env.server.game
	env.server.lastGameMessage = time.Now().Add(-time.Second)
	env.server.mu.Unlock()
	if old == nil {
		t.Fatal("game connection was not registered")
	}

	expired := env.server.expiredGame(time.Now())
	if expired != old {
		t.Fatalf("expiredGame = %p, want old connection %p", expired, old)
	}

	second := env.dialGame(t, testGameToken)
	second.readType(protocol.TypeGameHello)
	second.readType(protocol.TypeConfig)
	waitFor(t, func() bool {
		env.server.mu.Lock()
		defer env.server.mu.Unlock()
		return env.server.game != nil && env.server.game != old
	})

	// Simulate the tick closing the connection it captured before the
	// replacement registered. The replacement must stay connected.
	expired.closeGraceful(websocket.StatusGoingAway, "game_heartbeat_timeout")

	second.send(protocol.TypeGameReady, nil, protocol.GameReadyData{GameVersion: "test", MapVersion: "test"})
	waitFor(t, func() bool {
		return env.server.gameIsReady()
	})
	second.send(protocol.TypeHostRequest, nil, protocol.HostRequestData{Command: "start"})
	denied := second.readType(protocol.TypeHostDenied)
	data, err := protocol.DecodeData[protocol.HostDeniedData](denied)
	if err != nil {
		t.Fatalf("decode host_denied: %v", err)
	}
	if data.Code != protocol.DenialNoPlayers {
		t.Fatalf("host_denied code = %q, want %q", data.Code, protocol.DenialNoPlayers)
	}
}

func TestReplacedGameMessageIsIgnored(t *testing.T) {
	s := New(DefaultConfig())
	replaced := &gameConn{log: s.log}
	current := &gameConn{log: s.log}

	s.mu.Lock()
	s.game = current
	s.gameReady = false
	s.lastGameMessage = time.Time{}
	s.mu.Unlock()

	raw, err := protocol.Encode(protocol.TypeGameReady, nil, protocol.GameReadyData{
		GameVersion: "test",
		MapVersion:  "test",
	})
	if err != nil {
		t.Fatalf("encode game_ready: %v", err)
	}
	env, err := protocol.ParseEnvelope(raw)
	if err != nil {
		t.Fatalf("parse game_ready: %v", err)
	}

	if !s.handleGameMessage(replaced, env) {
		t.Fatal("replaced game message was not rejected")
	}
	if s.gameIsReady() {
		t.Fatal("replaced game message changed readiness")
	}
	s.mu.Lock()
	untouched := s.lastGameMessage.IsZero()
	s.mu.Unlock()
	if !untouched {
		t.Fatal("replaced game message refreshed the heartbeat deadline")
	}

	if s.handleGameMessage(current, env) {
		t.Fatal("current game message closed the connection")
	}
	if !s.gameIsReady() {
		t.Fatal("current game message did not set readiness")
	}
}

func TestGameReplacementRequiresReady(t *testing.T) {
	env := newTestEnv(t, func(cfg *Config) {
		cfg.TickInterval = time.Hour
	})

	first := env.dialGame(t, testGameToken)
	first.readType(protocol.TypeGameHello)
	first.readType(protocol.TypeConfig)
	first.send(protocol.TypeGameReady, nil, protocol.GameReadyData{GameVersion: "test", MapVersion: "test"})
	waitFor(t, func() bool {
		return env.server.gameIsReady()
	})

	env.server.mu.Lock()
	old := env.server.game
	env.server.mu.Unlock()

	second := env.dialGame(t, testGameToken)
	second.readType(protocol.TypeGameHello)
	second.readType(protocol.TypeConfig)
	waitFor(t, func() bool {
		env.server.mu.Lock()
		defer env.server.mu.Unlock()
		return env.server.game != nil && env.server.game != old
	})

	if env.server.gameIsReady() {
		t.Fatal("replacement game inherited readiness from the previous game")
	}
	if healthGameReady(t, env) {
		t.Fatal("/healthz reported the replacement game ready before game_ready")
	}

	second.send(protocol.TypeGameReady, nil, protocol.GameReadyData{GameVersion: "test", MapVersion: "test"})
	waitFor(t, func() bool {
		return env.server.gameIsReady()
	})
	if !healthGameReady(t, env) {
		t.Fatal("/healthz did not report the replacement game ready after game_ready")
	}
}

func TestHostEndSerializesWithControllerHello(t *testing.T) {
	env := newTestEnv(t, func(cfg *Config) {
		cfg.TickInterval = time.Hour
	})

	game := env.dialGame(t, testGameToken)
	game.readType(protocol.TypeGameHello)
	game.readType(protocol.TypeConfig)
	game.send(protocol.TypeGameReady, nil, protocol.GameReadyData{GameVersion: "test", MapVersion: "test"})
	waitFor(t, func() bool {
		return env.server.gameIsReady()
	})

	helloReached := make(chan struct{})
	releaseHello := make(chan struct{})
	env.server.afterControllerHello = func() {
		close(helloReached)
		<-releaseHello
	}

	controller := env.dialController(t)
	controller.sendHello("token-end-race")
	select {
	case <-helloReached:
	case <-time.After(2 * time.Second):
		t.Fatal("controller hello did not reach the registration barrier")
	}

	game.send(protocol.TypeHostRequest, nil, protocol.HostRequestData{Command: "end"})

	// An end path that is not serialized with controller registration would
	// clear the session during this pause. The controller would then register
	// into an ended session and stay connected because its slot no longer
	// exists for the idle timer.
	time.Sleep(100 * time.Millisecond)
	if phase := env.server.session.Phase(); phase == protocol.PhaseEnded {
		t.Fatal("host end cleared the session while controller hello was in progress")
	}
	close(releaseHello)

	controller.expectWelcome(1)
	controller.expectClosed(websocket.StatusNormalClosure, "session_ended")

	joined := false
	for {
		event := game.read()
		switch event.Type {
		case protocol.TypePlayerJoined:
			joined = true
		case protocol.TypeHostCommand:
			if !joined {
				t.Fatal("game received host end before the controller joined")
			}
			data, err := protocol.DecodeData[protocol.HostCommandData](event)
			if err != nil {
				t.Fatalf("decode host_command: %v", err)
			}
			if data.Command != string(session.CommandEnd) {
				t.Fatalf("host_command = %+v, want end", data)
			}
			return
		}
	}
}

func TestGamePhaseWaitsForTimerSerialization(t *testing.T) {
	s := New(DefaultConfig())
	if err := s.session.SetPhase(protocol.PhasePlaying, 1); err != nil {
		t.Fatalf("set phase: %v", err)
	}

	raw, err := protocol.Encode(protocol.TypePhase, nil, protocol.PhaseData{
		Phase:   protocol.PhasePaused,
		RoundID: 1,
	})
	if err != nil {
		t.Fatalf("encode phase: %v", err)
	}
	env, err := protocol.ParseEnvelope(raw)
	if err != nil {
		t.Fatalf("parse phase: %v", err)
	}

	s.gameNotifyMu.Lock()
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.handleGamePhase(&gameConn{log: s.log}, env)
	}()

	select {
	case <-done:
		t.Fatal("game phase was applied while the timer held the transition lock")
	case <-time.After(50 * time.Millisecond):
	}
	s.gameNotifyMu.Unlock()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("game phase did not finish after the lock was released")
	}
	if phase := s.session.Phase(); phase != protocol.PhasePaused {
		t.Fatalf("phase = %q, want %q", phase, protocol.PhasePaused)
	}
}

func TestHostCommandWaitsForControllerTransitionLock(t *testing.T) {
	s := New(DefaultConfig())
	s.mu.Lock()
	s.gameReady = true
	s.mu.Unlock()

	raw, err := protocol.Encode(protocol.TypeHostRequest, nil, protocol.HostRequestData{
		Command: string(session.CommandEnd),
	})
	if err != nil {
		t.Fatalf("encode host_request: %v", err)
	}
	env, err := protocol.ParseEnvelope(raw)
	if err != nil {
		t.Fatalf("parse host_request: %v", err)
	}

	conn := &gameConn{
		wsConn: &wsConn{
			queue: make(chan outbound, 8),
			done:  make(chan struct{}),
		},
		log: s.log,
	}

	s.gameNotifyMu.Lock()
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.handleHostRequest(conn, env)
	}()

	select {
	case <-done:
		t.Fatal("host command was applied while controller registration held the transition lock")
	case <-time.After(50 * time.Millisecond):
	}
	s.gameNotifyMu.Unlock()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("host command did not finish after the lock was released")
	}
	if phase := s.session.Phase(); phase != protocol.PhaseEnded {
		t.Fatalf("phase = %q, want %q", phase, protocol.PhaseEnded)
	}
}
