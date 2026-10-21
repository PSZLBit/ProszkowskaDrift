package server

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/PSZLBit/ProszkowskaDrift/server/internal/config"
	"github.com/PSZLBit/ProszkowskaDrift/server/internal/protocol"
)

const testGameToken = "test-game-token"

func newTestServer(t *testing.T, edit func(*config.Config)) (*Server, *httptest.Server) {
	t.Helper()
	cfg := config.Default()
	cfg.DevNoGame = true
	cfg.GameToken = testGameToken
	cfg.Listen = "127.0.0.1:0"
	if edit != nil {
		edit(&cfg)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	s := New(cfg, log, nil, Options{Version: "test"})
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(func() {
		s.stopping.Store(true)
		s.hub.closeAll(websocket.CloseGoingAway, "test cleanup")
		s.game.close()
		ts.Close()
	})
	return s, ts
}

func wsURL(ts *httptest.Server, path string) string {
	return "ws" + strings.TrimPrefix(ts.URL, "http") + path
}

func writeFrame(t *testing.T, conn *websocket.Conn, messageType string, seq *int64, data any) {
	t.Helper()
	raw, err := protocol.EncodeMessage(protocol.Version, messageType, seq, data)
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, raw); err != nil {
		t.Fatal(err)
	}
}

func readFrame(t *testing.T, conn *websocket.Conn) protocol.Envelope {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, raw, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	env, err := protocol.ParseEnvelope(raw)
	if err != nil {
		t.Fatalf("parse: %v (%s)", err, raw)
	}
	return env
}

func readType(t *testing.T, conn *websocket.Conn, want string) protocol.Envelope {
	t.Helper()
	for i := 0; i < 20; i++ {
		env := readFrame(t, conn)
		if env.Type == want {
			return env
		}
		t.Logf("skipping %s while waiting for %s", env.Type, want)
	}
	t.Fatalf("did not receive %s", want)
	return protocol.Envelope{}
}

func dialController(t *testing.T, ts *httptest.Server, token string) (*websocket.Conn, protocol.WelcomeData) {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(wsURL(ts, "/ws/controller"), nil)
	if err != nil {
		t.Fatalf("dial controller: %v", err)
	}
	writeFrame(t, conn, protocol.TypeHello, nil, protocol.HelloData{DeviceToken: token, AppVersion: "0.1.0"})
	env := readType(t, conn, protocol.TypeWelcome)
	welcome, err := protocol.DecodeData[protocol.WelcomeData](env)
	if err != nil {
		t.Fatal(err)
	}
	_ = readType(t, conn, protocol.TypePhase)
	return conn, welcome
}

func dialGame(t *testing.T, ts *httptest.Server, token string) *websocket.Conn {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(wsURL(ts, "/ws/game?token="+token), nil)
	if err != nil {
		t.Fatalf("dial game: %v", err)
	}
	hello := readType(t, conn, protocol.TypeGameHello)
	if _, err := protocol.DecodeData[protocol.GameHelloData](hello); err != nil {
		t.Fatal(err)
	}
	writeFrame(t, conn, protocol.TypeGameReady, nil, protocol.GameReadyData{GameVersion: "0.1.0", MapVersion: "test"})
	readType(t, conn, protocol.TypeConfig)
	return conn
}

func TestControllerSlotsAndSessionFull(t *testing.T) {
	s, ts := newTestServer(t, nil)
	var conns []*websocket.Conn
	for i := 1; i <= 4; i++ {
		conn, welcome := dialController(t, ts, "token-"+string(rune('0'+i)))
		conns = append(conns, conn)
		if welcome.Slot != i {
			t.Fatalf("welcome slot = %d, want %d", welcome.Slot, i)
		}
		if welcome.Phase != protocol.PhaseAttract {
			t.Fatalf("welcome phase = %s", welcome.Phase)
		}
		if welcome.Label != "P"+string(rune('0'+i)) || welcome.Color == "" {
			t.Fatalf("welcome = %+v", welcome)
		}
	}
	if s.session.ConnectedCount() != 4 {
		t.Fatalf("connected = %d", s.session.ConnectedCount())
	}

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(ts, "/ws/controller"), nil)
	if err != nil {
		t.Fatal(err)
	}
	writeFrame(t, conn, protocol.TypeHello, nil, protocol.HelloData{DeviceToken: "token-5"})
	errEnv := readType(t, conn, protocol.TypeError)
	errData, _ := protocol.DecodeData[protocol.ErrorData](errEnv)
	if errData.Code != protocol.CodeSessionFull {
		t.Fatalf("error = %+v", errData)
	}
	_ = conn.Close()

	duplicate, _, err := websocket.DefaultDialer.Dial(wsURL(ts, "/ws/controller"), nil)
	if err != nil {
		t.Fatal(err)
	}
	writeFrame(t, duplicate, protocol.TypeHello, nil, protocol.HelloData{DeviceToken: "token-1"})
	errEnv = readType(t, duplicate, protocol.TypeError)
	errData, _ = protocol.DecodeData[protocol.ErrorData](errEnv)
	if errData.Code != protocol.CodeAuthFailed {
		t.Fatalf("duplicate error = %+v", errData)
	}
	_ = duplicate.Close()

	for _, conn := range conns {
		_ = conn.Close()
	}
}

func TestControllerIdleTimeoutClosesSocket(t *testing.T) {
	s, ts := newTestServer(t, func(cfg *config.Config) {
		cfg.ControllerIdleTimeoutMs = 50
	})
	game := dialGame(t, ts, testGameToken)
	c, _ := dialController(t, ts, "idle")

	s.tick(time.Now().Add(2 * time.Second))

	left := readType(t, game, protocol.TypePlayerLeft)
	leftData, _ := protocol.DecodeData[protocol.PlayerLeftData](left)
	if leftData.Slot != 1 || leftData.Reason != protocol.LeftTimeout {
		t.Fatalf("player_left = %+v", leftData)
	}

	_ = c.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	if _, _, err := c.ReadMessage(); err == nil {
		t.Fatal("timed-out controller stayed connected")
	}
	_ = c.Close()
	_ = game.Close()
}

func TestGameAuthenticationAndHealth(t *testing.T) {
	_, ts := newTestServer(t, nil)
	if _, resp, err := websocket.DefaultDialer.Dial(wsURL(ts, "/ws/game?token=wrong"), nil); err == nil {
		t.Fatal("bad game token accepted")
	} else if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("bad token status = %+v", resp)
	}

	game := dialGame(t, ts, testGameToken)
	deadline := time.Now().Add(time.Second)
	for {
		resp, err := http.Get(ts.URL + "/healthz")
		if err != nil {
			t.Fatal(err)
		}
		var health map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&health)
		_ = resp.Body.Close()
		if health["gameReady"] == true {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("game never became ready: %+v", health)
		}
		time.Sleep(10 * time.Millisecond)
	}

	if _, resp, err := websocket.DefaultDialer.Dial(wsURL(ts, "/ws/game?token="+testGameToken), nil); err == nil {
		t.Fatal("second game accepted")
	} else if resp == nil || resp.StatusCode != http.StatusConflict {
		t.Fatalf("second game status = %+v", resp)
	}
	_ = game.Close()
}

func TestInputForwardingAndValidation(t *testing.T) {
	_, ts := newTestServer(t, nil)
	game := dialGame(t, ts, testGameToken)
	controller, welcome := dialController(t, ts, "controller-1")
	if welcome.Slot != 1 {
		t.Fatalf("slot = %d", welcome.Slot)
	}
	_ = readType(t, game, protocol.TypePlayerJoined)

	writeFrame(t, game, protocol.TypePhase, nil, protocol.PhaseData{Phase: protocol.PhasePlaying, RoundID: 1, TimeLeftMs: 180000})
	_ = readType(t, controller, protocol.TypePhase)

	seq := int64(1)
	writeFrame(t, controller, protocol.TypeInput, &seq, protocol.InputData{Steer: ptr(0.5), Throttle: ptr(0.8)})
	inputEnv := readType(t, game, protocol.TypeInput)
	input, _ := protocol.DecodeData[protocol.GameInputData](inputEnv)
	if input.Slot != 1 || input.Seq != 1 || input.Steer != 0.5 || input.Throttle != 0.8 {
		t.Fatalf("forwarded input = %+v", input)
	}

	badSeq := int64(2)
	writeFrame(t, controller, protocol.TypeInput, &badSeq, protocol.InputData{Steer: ptr(2.0), Throttle: ptr(0.0)})
	errEnv := readType(t, controller, protocol.TypeError)
	errData, _ := protocol.DecodeData[protocol.ErrorData](errEnv)
	if errData.Code != protocol.CodeInvalidInput {
		t.Fatalf("range error = %+v", errData)
	}

	// The next valid sequence proves the invalid value was not forwarded.
	goodSeq := int64(3)
	writeFrame(t, controller, protocol.TypeInput, &goodSeq, protocol.InputData{Steer: ptr(-1.0), Throttle: ptr(1.0)})
	inputEnv = readType(t, game, protocol.TypeInput)
	input, _ = protocol.DecodeData[protocol.GameInputData](inputEnv)
	if input.Seq != 3 {
		t.Fatalf("forwarded seq = %d", input.Seq)
	}

	writeFrame(t, game, protocol.TypePhase, nil, protocol.PhaseData{Phase: protocol.PhasePaused, RoundID: 1, TimeLeftMs: 100000})
	_ = readType(t, controller, protocol.TypePhase)
	pausedSeq := int64(4)
	writeFrame(t, controller, protocol.TypeInput, &pausedSeq, protocol.InputData{Steer: ptr(0.0), Throttle: ptr(0.0)})
	errEnv = readType(t, controller, protocol.TypeError)
	errData, _ = protocol.DecodeData[protocol.ErrorData](errEnv)
	if errData.Code != protocol.CodeInvalidInput {
		t.Fatalf("paused error = %+v", errData)
	}

	_ = controller.Close()
	_ = game.Close()
}

func TestReconnectReclaimsSlotAndCachesFeedback(t *testing.T) {
	_, ts := newTestServer(t, nil)
	game := dialGame(t, ts, testGameToken)
	controller, welcome := dialController(t, ts, "controller-1")
	if welcome.Slot != 1 {
		t.Fatalf("slot = %d", welcome.Slot)
	}
	_ = readType(t, game, protocol.TypePlayerJoined)

	writeFrame(t, game, protocol.TypePhase, nil, protocol.PhaseData{Phase: protocol.PhasePlaying, RoundID: 1, TimeLeftMs: 180000})
	_ = readType(t, controller, protocol.TypePhase)
	task := &protocol.TaskData{TaskID: 12, SpotLabel: "B-07", SpotColor: "red", Hint: "north", DeadlineMs: 30000}
	writeFrame(t, game, protocol.TypePlayerFeedbk, nil, protocol.PlayerFeedbackData{Slot: 1, Task: task, Score: 240, State: protocol.StatePlaying})
	feedbackEnv := readType(t, controller, protocol.TypePlayerFeedbk)
	feedback, _ := protocol.DecodeData[protocol.PlayerFeedbackData](feedbackEnv)
	if feedback.Task == nil || feedback.Task.TaskID != 12 || feedback.Score != 240 {
		t.Fatalf("feedback = %+v", feedback)
	}

	_ = controller.Close()
	left := readType(t, game, protocol.TypePlayerLeft)
	leftData, _ := protocol.DecodeData[protocol.PlayerLeftData](left)
	if leftData.Slot != 1 || leftData.Reason != protocol.LeftDisconnect {
		t.Fatalf("left = %+v", leftData)
	}

	reconnected, welcome := dialController(t, ts, "controller-1")
	if welcome.Slot != 1 {
		t.Fatalf("reconnect slot = %d", welcome.Slot)
	}
	feedbackEnv = readType(t, reconnected, protocol.TypePlayerFeedbk)
	feedback, _ = protocol.DecodeData[protocol.PlayerFeedbackData](feedbackEnv)
	if feedback.Task == nil || feedback.Task.TaskID != 12 {
		t.Fatalf("reconnect feedback = %+v", feedback)
	}
	_ = readType(t, game, protocol.TypePlayerJoined)
	_ = reconnected.Close()
	_ = game.Close()
}

func TestHostStartDenialKickAndReset(t *testing.T) {
	_, ts := newTestServer(t, nil)
	game := dialGame(t, ts, testGameToken)
	c1, _ := dialController(t, ts, "c1")
	c2, _ := dialController(t, ts, "c2")
	_ = readType(t, game, protocol.TypePlayerJoined)
	_ = readType(t, game, protocol.TypePlayerJoined)

	writeFrame(t, game, protocol.TypeHostRequest, nil, protocol.HostRequestData{Command: protocol.HostStart})
	command := readType(t, game, protocol.TypeHostCommand)
	commandData, _ := protocol.DecodeData[protocol.HostCommandData](command)
	if commandData.Command != protocol.HostStart {
		t.Fatalf("command = %+v", commandData)
	}
	writeFrame(t, game, protocol.TypePhase, nil, protocol.PhaseData{Phase: protocol.PhasePlaying, RoundID: 1, TimeLeftMs: 180000})
	_ = readType(t, c1, protocol.TypePhase)
	_ = readType(t, c2, protocol.TypePhase)

	writeFrame(t, game, protocol.TypeHostRequest, nil, protocol.HostRequestData{Command: protocol.HostStart})
	denied := readType(t, game, protocol.TypeHostDenied)
	deniedData, _ := protocol.DecodeData[protocol.HostDeniedData](denied)
	if deniedData.Code != protocol.CodeInvalidPhase {
		t.Fatalf("denied = %+v", deniedData)
	}

	slot := 2
	writeFrame(t, game, protocol.TypeHostRequest, nil, protocol.HostRequestData{Command: protocol.HostKick, Slot: &slot})
	kick := readType(t, game, protocol.TypeHostCommand)
	kickData, _ := protocol.DecodeData[protocol.HostCommandData](kick)
	if kickData.Command != protocol.HostKick || kickData.Slot == nil || *kickData.Slot != 2 {
		t.Fatalf("kick = %+v", kickData)
	}
	kicked := readType(t, c2, protocol.TypeError)
	kickedData, _ := protocol.DecodeData[protocol.ErrorData](kicked)
	if kickedData.Code != protocol.CodeKicked {
		t.Fatalf("kicked = %+v", kickedData)
	}
	_ = c2.Close()

	// The kicked token cannot immediately reclaim the slot.
	rejoin, _, err := websocket.DefaultDialer.Dial(wsURL(ts, "/ws/controller"), nil)
	if err != nil {
		t.Fatal(err)
	}
	writeFrame(t, rejoin, protocol.TypeHello, nil, protocol.HelloData{DeviceToken: "c2"})
	kicked = readType(t, rejoin, protocol.TypeError)
	kickedData, _ = protocol.DecodeData[protocol.ErrorData](kicked)
	if kickedData.Code != protocol.CodeKicked {
		t.Fatalf("rejoin kicked = %+v", kickedData)
	}
	_ = rejoin.Close()

	writeFrame(t, game, protocol.TypeHostRequest, nil, protocol.HostRequestData{Command: protocol.HostReset})
	reset := readType(t, game, protocol.TypeHostCommand)
	resetData, _ := protocol.DecodeData[protocol.HostCommandData](reset)
	if resetData.Command != protocol.HostReset {
		t.Fatalf("reset = %+v", resetData)
	}
	_ = readType(t, game, protocol.TypeConfig)
	_ = readType(t, c1, protocol.TypePhase)

	_ = c1.Close()
	_ = game.Close()
}

func TestHostEndClearsSessionAndClosesControllers(t *testing.T) {
	s, ts := newTestServer(t, nil)
	game := dialGame(t, ts, testGameToken)
	c1, _ := dialController(t, ts, "c1")
	_ = readType(t, game, protocol.TypePlayerJoined)

	writeFrame(t, game, protocol.TypeHostRequest, nil, protocol.HostRequestData{Command: protocol.HostEnd})
	command := readType(t, game, protocol.TypeHostCommand)
	commandData, _ := protocol.DecodeData[protocol.HostCommandData](command)
	if commandData.Command != protocol.HostEnd {
		t.Fatalf("command = %+v", commandData)
	}
	phase := readType(t, c1, protocol.TypePhase)
	phaseData, _ := protocol.DecodeData[protocol.PhaseData](phase)
	if phaseData.Phase != protocol.PhaseEnded {
		t.Fatalf("phase = %+v", phaseData)
	}
	if s.session.ConnectedCount() != 0 || len(s.session.AssignedSlots()) != 0 {
		t.Fatalf("session not cleared: connected=%d assigned=%d", s.session.ConnectedCount(), len(s.session.AssignedSlots()))
	}
	_ = c1.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	if _, _, err := c1.ReadMessage(); err == nil {
		t.Fatal("controller stayed connected after session end")
	}

	late, _, err := websocket.DefaultDialer.Dial(wsURL(ts, "/ws/controller"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer late.Close()
	writeFrame(t, late, protocol.TypeHello, nil, protocol.HelloData{DeviceToken: "late"})
	lateEnv := readType(t, late, protocol.TypeError)
	lateData, _ := protocol.DecodeData[protocol.ErrorData](lateEnv)
	if lateData.Code != protocol.CodeAuthFailed {
		t.Fatalf("late controller error = %+v", lateData)
	}
	_ = game.Close()
}

func TestFeedbackPrivacyAndRoundResultBroadcast(t *testing.T) {
	_, ts := newTestServer(t, nil)
	game := dialGame(t, ts, testGameToken)
	c1, _ := dialController(t, ts, "c1")
	c2, _ := dialController(t, ts, "c2")
	_ = readType(t, game, protocol.TypePlayerJoined)
	_ = readType(t, game, protocol.TypePlayerJoined)

	result := protocol.RoundResultData{
		RoundID:    7,
		WinnerSlot: 1,
		Scores:     []protocol.RoundScore{{Slot: 1, Score: 620, Tasks: 5}, {Slot: 2, Score: 710, Tasks: 5}},
	}
	writeFrame(t, game, protocol.TypeRoundResult, nil, result)
	_ = readType(t, c1, protocol.TypeRoundResult)
	_ = readType(t, c2, protocol.TypeRoundResult)

	task := &protocol.TaskData{TaskID: 12, SpotLabel: "B-07", SpotColor: "blue", Hint: "north", DeadlineMs: 30000}
	writeFrame(t, game, protocol.TypePlayerFeedbk, nil, protocol.PlayerFeedbackData{Slot: 1, Task: task, Score: 100, State: protocol.StatePlaying})
	feedback := readType(t, c1, protocol.TypePlayerFeedbk)
	data, _ := protocol.DecodeData[protocol.PlayerFeedbackData](feedback)
	if data.Slot != 1 || data.Task == nil || data.Task.TaskID != 12 {
		t.Fatalf("feedback = %+v", data)
	}

	_ = c2.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	if _, _, err := c2.ReadMessage(); err == nil {
		t.Fatal("controller 2 received controller 1's private feedback")
	}
	_ = c1.Close()
	_ = c2.Close()
	_ = game.Close()
}

func ptr[T any](v T) *T { return &v }
