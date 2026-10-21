package gameserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/PSZLBit/ProszkowskaDrift/server/internal/protocol"
)

const testGameToken = "test-game-token"

type testEnv struct {
	server *Server
	http   *httptest.Server
}

func newTestEnv(t *testing.T, mutate func(*Config)) *testEnv {
	t.Helper()

	cfg := DefaultConfig()
	cfg.GameToken = testGameToken
	cfg.ControllerTimeout = 2 * time.Second
	cfg.InvalidThreshold = 3
	cfg.RateThreshold = 3
	cfg.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	if mutate != nil {
		mutate(&cfg)
	}

	srv := New(cfg)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(func() {
		ts.CloseClientConnections()
		ts.Close()
	})

	return &testEnv{server: srv, http: ts}
}

type testClient struct {
	t    *testing.T
	conn *websocket.Conn
	mu   sync.Mutex
}

func (e *testEnv) dialController(t *testing.T) *testClient {
	t.Helper()

	wsURL := "ws" + strings.TrimPrefix(e.http.URL, "http") + "/ws/controller"
	return dialTestClient(t, wsURL)
}

func (e *testEnv) dialGame(t *testing.T, token string) *testClient {
	t.Helper()

	wsURL := "ws" + strings.TrimPrefix(e.http.URL, "http") + "/ws/game?token=" + url.QueryEscape(token)
	return dialTestClient(t, wsURL)
}

func dialTestClient(t *testing.T, wsURL string) *testClient {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("dial %s: %v", wsURL, err)
	}

	c := &testClient{t: t, conn: conn}
	t.Cleanup(c.close)
	return c
}

func (c *testClient) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.conn.Close(websocket.StatusNormalClosure, "test done")
}

func (c *testClient) send(typ string, seq *int64, data any) {
	c.t.Helper()
	raw, err := protocol.Encode(typ, seq, data)
	if err != nil {
		c.t.Fatalf("encode %s: %v", typ, err)
	}
	c.sendRaw(raw)
}

func (c *testClient) sendRaw(raw []byte) {
	c.t.Helper()
	c.mu.Lock()
	defer c.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := c.conn.Write(ctx, websocket.MessageText, raw); err != nil {
		c.t.Fatalf("write: %v", err)
	}
}

func (c *testClient) sendHello(token string) {
	c.t.Helper()
	c.send(protocol.TypeHello, nil, protocol.HelloData{DeviceToken: token, AppVersion: "test"})
}

func (c *testClient) read() protocol.Envelope {
	c.t.Helper()
	return c.readWithTimeout(5 * time.Second)
}

func (c *testClient) readWithTimeout(timeout time.Duration) protocol.Envelope {
	c.t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	typ, raw, err := c.conn.Read(ctx)
	if err != nil {
		c.t.Fatalf("read: %v", err)
	}
	if typ != websocket.MessageText {
		c.t.Fatalf("message type = %v, want text", typ)
	}
	env, err := protocol.ParseEnvelope(raw)
	if err != nil {
		c.t.Fatalf("parse %q: %v", raw, err)
	}
	return env
}

func (c *testClient) readType(want string) protocol.Envelope {
	c.t.Helper()
	env := c.read()
	if env.Type != want {
		c.t.Fatalf("message type = %q, want %q (data: %s)", env.Type, want, env.Data)
	}
	return env
}

func (c *testClient) expectWelcome(slot int) protocol.WelcomeData {
	c.t.Helper()
	env := c.readType(protocol.TypeWelcome)
	data, err := protocol.DecodeData[protocol.WelcomeData](env)
	if err != nil {
		c.t.Fatalf("decode welcome: %v", err)
	}
	if data.Slot != slot {
		c.t.Fatalf("welcome slot = %d, want %d", data.Slot, slot)
	}
	return data
}

func (c *testClient) expectError(code string) protocol.ErrorData {
	c.t.Helper()
	env := c.readType(protocol.TypeError)
	data, err := protocol.DecodeData[protocol.ErrorData](env)
	if err != nil {
		c.t.Fatalf("decode error: %v", err)
	}
	if data.Code != code {
		c.t.Fatalf("error code = %q, want %q (%s)", data.Code, code, data.Message)
	}
	return data
}

func (c *testClient) expectClosed(code websocket.StatusCode, reason string) {
	c.t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for {
		timeout := time.Until(deadline)
		if timeout <= 0 {
			c.t.Fatalf("connection did not close with %q", reason)
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		_, _, err := c.conn.Read(ctx)
		cancel()
		if err == nil {
			continue
		}

		var closeErr websocket.CloseError
		if !errors.As(err, &closeErr) {
			c.t.Fatalf("close error = %v, want websocket close %q", err, reason)
		}
		if closeErr.Code != code || closeErr.Reason != reason {
			c.t.Fatalf("close = (%v, %q), want (%v, %q)", closeErr.Code, closeErr.Reason, code, reason)
		}
		return
	}
}

func TestFourControllersGetSlotsAndInputFlows(t *testing.T) {
	env := newTestEnv(t, nil)

	game := env.dialGame(t, testGameToken)
	helloEnv := game.readType(protocol.TypeGameHello)
	gameHello, err := protocol.DecodeData[protocol.GameHelloData](helloEnv)
	if err != nil {
		t.Fatalf("decode game_hello: %v", err)
	}
	if gameHello.ProtocolVersion != protocol.Version || gameHello.Token != testGameToken {
		t.Fatalf("game_hello = %+v", gameHello)
	}

	configEnv := game.readType(protocol.TypeConfig)
	config, err := protocol.DecodeData[protocol.ConfigData](configEnv)
	if err != nil {
		t.Fatalf("decode config: %v", err)
	}
	if config.RoundDurationMs != 180000 || config.TasksPerPlayer != 5 || config.MaxPlayers != 4 {
		t.Fatalf("config = %+v", config)
	}

	game.send(protocol.TypeGameReady, nil, protocol.GameReadyData{GameVersion: "test", MapVersion: "test"})

	colors := []string{"red", "blue", "green", "yellow"}
	controllers := make([]*testClient, 4)
	for i := range controllers {
		controllers[i] = env.dialController(t)
		controllers[i].sendHello(fmt.Sprintf("token-player-%d", i+1))
		welcome := controllers[i].expectWelcome(i + 1)
		if welcome.Label != fmt.Sprintf("P%d", i+1) || welcome.Color != colors[i] {
			t.Fatalf("welcome = %+v", welcome)
		}
		if welcome.Phase != protocol.PhaseAttract {
			t.Fatalf("phase = %q, want attract", welcome.Phase)
		}
	}

	joined := map[int]bool{}
	for range controllers {
		env := game.readType(protocol.TypePlayerJoined)
		data, err := protocol.DecodeData[protocol.PlayerJoinedData](env)
		if err != nil {
			t.Fatalf("decode player_joined: %v", err)
		}
		joined[data.Slot] = true
	}
	for slot := 1; slot <= 4; slot++ {
		if !joined[slot] {
			t.Fatalf("missing player_joined for slot %d", slot)
		}
	}

	fifth := env.dialController(t)
	fifth.sendHello("token-player-5")
	fifth.expectError(protocol.CodeSessionFull)

	game.send(protocol.TypeHostRequest, nil, protocol.HostRequestData{Command: "start"})
	hostCommandEnv := game.readType(protocol.TypeHostCommand)
	hostCommand, err := protocol.DecodeData[protocol.HostCommandData](hostCommandEnv)
	if err != nil {
		t.Fatalf("decode host_command: %v", err)
	}
	if hostCommand.Command != "start" {
		t.Fatalf("host_command = %+v", hostCommand)
	}

	for i := range controllers {
		phaseEnv := controllers[i].readType(protocol.TypePhase)
		phase, err := protocol.DecodeData[protocol.PhaseData](phaseEnv)
		if err != nil {
			t.Fatalf("decode phase: %v", err)
		}
		if phase.Phase != protocol.PhasePlaying || phase.RoundID != 1 {
			t.Fatalf("phase = %+v", phase)
		}
		controllers[i].send(protocol.TypeInput, int64Ptr(1), protocol.InputData{
			Steer:    float64Ptr(0.1 * float64(i+1)),
			Throttle: float64Ptr(0.9),
		})
	}

	for range controllers {
		inputEnv := game.readType(protocol.TypeInput)
		input, err := protocol.DecodeData[protocol.ForwardedInputData](inputEnv)
		if err != nil {
			t.Fatalf("decode forwarded input: %v", err)
		}
		if input.Slot < 1 || input.Slot > 4 || input.Seq != 1 {
			t.Fatalf("forwarded input = %+v", input)
		}
		if input.Steer != 0.1*float64(input.Slot) {
			t.Fatalf("forwarded steer = %v for slot %d", input.Steer, input.Slot)
		}
	}

	controllers[0].send(protocol.TypeInput, int64Ptr(2), protocol.InputData{
		Steer:    float64Ptr(2),
		Throttle: float64Ptr(0),
	})
	controllers[0].expectError(protocol.CodeInvalidInput)

	controllers[0].send(protocol.TypeInput, int64Ptr(2), protocol.InputData{
		Steer:    float64Ptr(0.25),
		Throttle: float64Ptr(0.5),
	})
	inputEnv := game.readType(protocol.TypeInput)
	input, err := protocol.DecodeData[protocol.ForwardedInputData](inputEnv)
	if err != nil {
		t.Fatalf("decode forwarded input: %v", err)
	}
	if input.Slot != 1 || input.Seq != 2 || input.Steer != 0.25 {
		t.Fatalf("forwarded input = %+v; invalid input must not be forwarded", input)
	}

	game.send(protocol.TypeHostRequest, nil, protocol.HostRequestData{Command: "pause"})
	if env := game.readType(protocol.TypeHostCommand); env.Type != protocol.TypeHostCommand {
		t.Fatalf("unexpected envelope %+v", env)
	}
	for i := range controllers {
		phaseEnv := controllers[i].readType(protocol.TypePhase)
		phase, err := protocol.DecodeData[protocol.PhaseData](phaseEnv)
		if err != nil {
			t.Fatalf("decode phase: %v", err)
		}
		if phase.Phase != protocol.PhasePaused {
			t.Fatalf("phase = %+v, want paused", phase)
		}
	}
	controllers[0].send(protocol.TypeInput, int64Ptr(3), protocol.InputData{
		Steer:    float64Ptr(0),
		Throttle: float64Ptr(0),
	})
	controllers[0].expectError(protocol.CodeInvalidInput)

	game.send(protocol.TypeHostRequest, nil, protocol.HostRequestData{Command: "resume"})
	game.readType(protocol.TypeHostCommand)
	for i := range controllers {
		phaseEnv := controllers[i].readType(protocol.TypePhase)
		phase, err := protocol.DecodeData[protocol.PhaseData](phaseEnv)
		if err != nil {
			t.Fatalf("decode phase: %v", err)
		}
		if phase.Phase != protocol.PhasePlaying {
			t.Fatalf("phase = %+v, want playing", phase)
		}
	}
	controllers[0].send(protocol.TypeInput, int64Ptr(3), protocol.InputData{
		Steer:    float64Ptr(0.5),
		Throttle: float64Ptr(1),
	})
	inputEnv = game.readType(protocol.TypeInput)
	input, err = protocol.DecodeData[protocol.ForwardedInputData](inputEnv)
	if err != nil {
		t.Fatalf("decode forwarded input: %v", err)
	}
	if input.Slot != 1 || input.Seq != 3 || input.Steer != 0.5 {
		t.Fatalf("forwarded input = %+v", input)
	}
}

func TestReconnectReclaimsSlot(t *testing.T) {
	env := newTestEnv(t, nil)

	first := env.dialController(t)
	first.sendHello("token-reconnect")
	first.expectWelcome(1)

	duplicate := env.dialController(t)
	duplicate.sendHello("token-reconnect")
	duplicate.expectError(protocol.CodeAuthFailed)

	first.close()
	waitFor(t, func() bool {
		slots := env.server.session.Slots()
		return !slots[0].Connected
	})

	second := env.dialController(t)
	second.sendHello("token-reconnect")
	second.expectWelcome(1)

	unknown := env.dialController(t)
	unknown.sendHello("token-new-player")
	unknown.expectWelcome(2)
}

func TestHeartbeatTimeoutReleasesSlot(t *testing.T) {
	env := newTestEnv(t, func(cfg *Config) {
		cfg.ControllerTimeout = 250 * time.Millisecond
	})

	first := env.dialController(t)
	first.sendHello("token-heartbeat")
	first.expectWelcome(1)

	first.expectClosed(websocket.StatusPolicyViolation, "heartbeat_timeout")
	waitFor(t, func() bool {
		return !env.server.session.Slots()[0].Connected
	})

	second := env.dialController(t)
	second.sendHello("token-heartbeat")
	second.expectWelcome(1)
}

func TestPingPongLeaveAndUnknownType(t *testing.T) {
	env := newTestEnv(t, nil)

	c := env.dialController(t)
	c.send(protocol.TypePing, nil, protocol.PingData{ClientTime: 123})
	pongEnv := c.readType(protocol.TypePong)
	pong, err := protocol.DecodeData[protocol.PongData](pongEnv)
	if err != nil {
		t.Fatalf("decode pong: %v", err)
	}
	if pong.ClientTime != 123 || pong.ServerTime == 0 {
		t.Fatalf("pong = %+v", pong)
	}

	c.send("future_message", nil, map[string]any{"value": 1})
	c.send(protocol.TypePing, nil, protocol.PingData{ClientTime: 124})
	pongEnv = c.readType(protocol.TypePong)
	pong, err = protocol.DecodeData[protocol.PongData](pongEnv)
	if err != nil {
		t.Fatalf("decode pong after unknown type: %v", err)
	}
	if pong.ClientTime != 124 {
		t.Fatalf("pong = %+v", pong)
	}

	c.sendHello("token-leave")
	c.expectWelcome(1)
	c.send(protocol.TypeLeave, nil, struct{}{})
	c.sendHello("token-leave")
	c.expectWelcome(1)
}

func TestInvalidInputClosesConnection(t *testing.T) {
	env := newTestEnv(t, func(cfg *Config) {
		cfg.InvalidThreshold = 3
	})

	c := env.dialController(t)
	c.sendHello("token-invalid")
	c.expectWelcome(1)
	if err := env.server.session.SetPhase(protocol.PhasePlaying, 1); err != nil {
		t.Fatalf("SetPhase: %v", err)
	}

	for seq := int64(1); seq <= 3; seq++ {
		c.send(protocol.TypeInput, &seq, protocol.InputData{
			Steer:    float64Ptr(1.5),
			Throttle: float64Ptr(0),
		})
		c.expectError(protocol.CodeInvalidInput)
	}
	c.expectClosed(websocket.StatusPolicyViolation, "invalid_input")

	// The server must stay up after closing an abusive connection.
	other := env.dialController(t)
	other.sendHello("token-after-invalid")
	other.expectWelcome(2)
}

func TestRateLimitClosesConnection(t *testing.T) {
	env := newTestEnv(t, func(cfg *Config) {
		cfg.RateLimit = 2
		cfg.RateWindow = 10 * time.Second
		cfg.RateThreshold = 3
	})

	c := env.dialController(t)
	c.sendHello("token-rate")
	c.expectWelcome(1)
	if err := env.server.session.SetPhase(protocol.PhasePlaying, 1); err != nil {
		t.Fatalf("SetPhase: %v", err)
	}

	for seq := int64(1); seq <= 2; seq++ {
		c.send(protocol.TypeInput, &seq, protocol.InputData{
			Steer:    float64Ptr(0),
			Throttle: float64Ptr(0),
		})
	}
	for range 3 {
		c.send(protocol.TypeInput, int64Ptr(99), protocol.InputData{
			Steer:    float64Ptr(0),
			Throttle: float64Ptr(0),
		})
		c.expectError(protocol.CodeRateLimited)
	}
	c.expectClosed(websocket.StatusPolicyViolation, "rate_limited")
}

func TestMalformedControllerMessageIsRejected(t *testing.T) {
	env := newTestEnv(t, nil)

	c := env.dialController(t)
	c.sendRaw([]byte(`{"v":1,"type":"hello"`))
	c.expectError(protocol.CodeInvalidInput)

	c.send(protocol.TypePing, nil, protocol.PingData{ClientTime: 5})
	c.readType(protocol.TypePong)
}

func TestControllerCannotIssueHostCommands(t *testing.T) {
	env := newTestEnv(t, nil)

	game := env.dialGame(t, testGameToken)
	game.readType(protocol.TypeGameHello)
	game.readType(protocol.TypeConfig)

	c := env.dialController(t)
	c.sendHello("token-not-host")
	c.expectWelcome(1)
	game.readType(protocol.TypePlayerJoined)

	c.send(protocol.TypeHostRequest, nil, protocol.HostRequestData{Command: "start"})
	c.send(protocol.TypePing, nil, protocol.PingData{ClientTime: 6})
	c.readType(protocol.TypePong)

	if phase := env.server.session.Phase(); phase != protocol.PhaseAttract {
		t.Fatalf("controller host_request changed phase to %q", phase)
	}

	game.send(protocol.TypeHostRequest, nil, protocol.HostRequestData{Command: "start"})
	hostCommandEnv := game.readType(protocol.TypeHostCommand)
	hostCommand, err := protocol.DecodeData[protocol.HostCommandData](hostCommandEnv)
	if err != nil {
		t.Fatalf("decode host_command: %v", err)
	}
	if hostCommand.Command != "start" {
		t.Fatalf("host_command = %+v", hostCommand)
	}
}

func TestProtocolMismatchClosesConnection(t *testing.T) {
	env := newTestEnv(t, nil)

	c := env.dialController(t)
	c.sendRaw([]byte(`{"v":2,"type":"hello","data":{"deviceToken":"token-version"}}`))
	c.expectError(protocol.CodeProtocolMismatch)
	c.expectClosed(websocket.StatusPolicyViolation, "protocol_mismatch")
}

func TestKickClosesControllerAndFreesSlot(t *testing.T) {
	env := newTestEnv(t, nil)

	game := env.dialGame(t, testGameToken)
	game.readType(protocol.TypeGameHello)
	game.readType(protocol.TypeConfig)

	first := env.dialController(t)
	first.sendHello("token-kick-one")
	first.expectWelcome(1)

	second := env.dialController(t)
	second.sendHello("token-kick-two")
	second.expectWelcome(2)

	game.readType(protocol.TypePlayerJoined)
	game.readType(protocol.TypePlayerJoined)

	game.send(protocol.TypeHostRequest, nil, protocol.HostRequestData{Command: "kick", Slot: 2})
	hostCommandEnv := game.readType(protocol.TypeHostCommand)
	hostCommand, err := protocol.DecodeData[protocol.HostCommandData](hostCommandEnv)
	if err != nil {
		t.Fatalf("decode host_command: %v", err)
	}
	if hostCommand.Command != "kick" || hostCommand.Slot != 2 {
		t.Fatalf("host_command = %+v", hostCommand)
	}

	leftEnv := game.readType(protocol.TypePlayerLeft)
	left, err := protocol.DecodeData[protocol.PlayerLeftData](leftEnv)
	if err != nil {
		t.Fatalf("decode player_left: %v", err)
	}
	if left.Slot != 2 || left.Reason != "kicked" {
		t.Fatalf("player_left = %+v", left)
	}

	second.expectError(protocol.CodeKicked)
	second.expectClosed(websocket.StatusPolicyViolation, "kicked")

	first.send(protocol.TypePing, nil, protocol.PingData{ClientTime: 1})
	first.readType(protocol.TypePong)

	replacement := env.dialController(t)
	replacement.sendHello("token-kick-replacement")
	replacement.expectWelcome(2)
}

func TestLeaveReservesSlotAndNotifiesGame(t *testing.T) {
	env := newTestEnv(t, nil)

	game := env.dialGame(t, testGameToken)
	game.readType(protocol.TypeGameHello)
	game.readType(protocol.TypeConfig)

	c := env.dialController(t)
	c.sendHello("token-leave")
	c.expectWelcome(1)
	game.readType(protocol.TypePlayerJoined)

	c.send(protocol.TypeLeave, nil, struct{}{})
	leftEnv := game.readType(protocol.TypePlayerLeft)
	left, err := protocol.DecodeData[protocol.PlayerLeftData](leftEnv)
	if err != nil {
		t.Fatalf("decode player_left: %v", err)
	}
	if left.Slot != 1 || left.Reason != "leave" {
		t.Fatalf("player_left = %+v", left)
	}

	slots := env.server.session.Slots()
	if !slots[0].Assigned || slots[0].Connected {
		t.Fatalf("slot after leave = %+v, want assigned and disconnected", slots[0])
	}

	c.sendHello("token-leave")
	c.expectWelcome(1)
	game.readType(protocol.TypePlayerJoined)
}

func TestGameAuthFails(t *testing.T) {
	env := newTestEnv(t, nil)

	for _, token := range []string{"wrong-token", ""} {
		t.Run("token="+token, func(t *testing.T) {
			game := env.dialGame(t, token)
			game.expectError(protocol.CodeAuthFailed)
			game.expectClosed(websocket.StatusPolicyViolation, "auth_failed")
		})
	}
}

func TestGamePhaseDrivesControllerPhaseAndInput(t *testing.T) {
	env := newTestEnv(t, nil)

	game := env.dialGame(t, testGameToken)
	game.readType(protocol.TypeGameHello)
	game.readType(protocol.TypeConfig)
	game.send(protocol.TypeGameReady, nil, protocol.GameReadyData{GameVersion: "test"})

	c := env.dialController(t)
	c.sendHello("token-phase")
	c.expectWelcome(1)
	game.readType(protocol.TypePlayerJoined)

	game.send(protocol.TypePhase, nil, protocol.PhaseData{Phase: protocol.PhaseCountdown, RoundID: 5})
	phaseEnv := c.readType(protocol.TypePhase)
	phase, err := protocol.DecodeData[protocol.PhaseData](phaseEnv)
	if err != nil {
		t.Fatalf("decode phase: %v", err)
	}
	if phase.Phase != protocol.PhaseCountdown || phase.RoundID != 5 {
		t.Fatalf("phase = %+v", phase)
	}

	game.send(protocol.TypePhase, nil, protocol.PhaseData{Phase: protocol.PhasePlaying, RoundID: 5, TimeLeftMs: 120000})
	phaseEnv = c.readType(protocol.TypePhase)
	phase, err = protocol.DecodeData[protocol.PhaseData](phaseEnv)
	if err != nil {
		t.Fatalf("decode phase: %v", err)
	}
	if phase.Phase != protocol.PhasePlaying || phase.RoundID != 5 || phase.TimeLeftMs != 120000 {
		t.Fatalf("phase = %+v", phase)
	}

	c.send(protocol.TypeInput, int64Ptr(7), protocol.InputData{
		Steer:    float64Ptr(0.3),
		Throttle: float64Ptr(-0.4),
	})
	inputEnv := game.readType(protocol.TypeInput)
	input, err := protocol.DecodeData[protocol.ForwardedInputData](inputEnv)
	if err != nil {
		t.Fatalf("decode input: %v", err)
	}
	if input.Slot != 1 || input.Seq != 7 || input.Steer != 0.3 || input.Throttle != -0.4 {
		t.Fatalf("forwarded input = %+v", input)
	}
}

func TestGameConnectingAfterControllerGetsPlayerJoined(t *testing.T) {
	env := newTestEnv(t, nil)

	c := env.dialController(t)
	c.sendHello("token-early")
	c.expectWelcome(1)

	game := env.dialGame(t, testGameToken)
	game.readType(protocol.TypeGameHello)
	game.readType(protocol.TypeConfig)

	joinedEnv := game.readType(protocol.TypePlayerJoined)
	joined, err := protocol.DecodeData[protocol.PlayerJoinedData](joinedEnv)
	if err != nil {
		t.Fatalf("decode player_joined: %v", err)
	}
	if joined.Slot != 1 || joined.Label != "P1" || joined.Color != "red" {
		t.Fatalf("player_joined = %+v", joined)
	}
}

func TestLoopbackRemoteAddr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		addr string
		want bool
	}{
		{addr: "127.0.0.1:1234", want: true},
		{addr: "[::1]:1234", want: true},
		{addr: "192.168.1.10:1234", want: false},
		{addr: "10.0.0.5:1234", want: false},
		{addr: "not-an-address", want: false},
		{addr: "", want: false},
	}
	for _, tt := range tests {
		if got := isLoopbackRemoteAddr(tt.addr); got != tt.want {
			t.Errorf("isLoopbackRemoteAddr(%q) = %v, want %v", tt.addr, got, tt.want)
		}
	}
}

func TestTruncateToken(t *testing.T) {
	t.Parallel()

	token := "12345678-1234-1234-1234-123456789abc"
	got := truncateToken(token)
	if got == token {
		t.Fatal("token must be truncated in logs")
	}
	if !strings.Contains(got, "...") {
		t.Fatalf("truncated token = %q", got)
	}
	if got := truncateToken("short"); strings.Contains(got, "short") {
		t.Fatalf("short token must not be logged verbatim: %q", got)
	}
}

func TestUnknownJSONFieldsAreIgnored(t *testing.T) {
	env := newTestEnv(t, nil)

	c := env.dialController(t)
	c.sendRaw([]byte(`{"v":1,"type":"hello","data":{"deviceToken":"token-unknown-fields","future":true},"future":1}`))
	c.expectWelcome(1)
}

func TestConfigJSONShape(t *testing.T) {
	env := newTestEnv(t, nil)
	game := env.dialGame(t, testGameToken)
	game.readType(protocol.TypeGameHello)
	configEnv := game.readType(protocol.TypeConfig)

	var payload struct {
		RoundDurationMs int `json:"roundDurationMs"`
		TasksPerPlayer  int `json:"tasksPerPlayer"`
		MaxPlayers      int `json:"maxPlayers"`
	}
	if err := json.Unmarshal(configEnv.Data, &payload); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	if payload.RoundDurationMs != 180000 || payload.TasksPerPlayer != 5 || payload.MaxPlayers != 4 {
		t.Fatalf("config payload = %+v", payload)
	}
}

func int64Ptr(v int64) *int64       { return &v }
func float64Ptr(v float64) *float64 { return &v }

func waitFor(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not met before timeout")
}
