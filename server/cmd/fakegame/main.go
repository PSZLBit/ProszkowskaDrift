// Command fakegame exercises the game WebSocket endpoint without Godot. It is
// a Sprint 0 integration tool for S0-04 and S0-05, not production code.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/coder/websocket"

	"github.com/PSZLBit/ProszkowskaDrift/server/internal/protocol"
)

type options struct {
	server    string
	token     string
	duration  time.Duration
	autoStart bool
}

type inputCounter struct {
	mu     sync.Mutex
	counts [5]int
	logger *log.Logger
}

func main() {
	log.SetFlags(0)

	opts := options{}
	flag.StringVar(&opts.server, "server", "ws://127.0.0.1:8080/ws/game", "game WebSocket URL")
	flag.StringVar(&opts.token, "token", os.Getenv("PD_GAME_TOKEN"), "game token (or PD_GAME_TOKEN)")
	flag.DurationVar(&opts.duration, "duration", 0, "exit after this duration (0 runs until interrupted)")
	flag.BoolVar(&opts.autoStart, "auto-start", false, "send host_request start one second after connecting")
	flag.Parse()

	if opts.token == "" {
		fmt.Fprintln(os.Stderr, "fakegame: -token or PD_GAME_TOKEN is required")
		os.Exit(2)
	}
	if err := run(opts); err != nil {
		fmt.Fprintln(os.Stderr, "fakegame:", err)
		os.Exit(1)
	}
}

func run(opts options) error {
	wsURL, err := gameURL(opts.server, opts.token)
	if err != nil {
		return err
	}

	logger := log.New(os.Stdout, "[fakegame] ", 0)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer conn.CloseNow()

	var writeMu sync.Mutex
	send := func(typ string, data any) error {
		raw, err := protocol.Encode(typ, nil, data)
		if err != nil {
			return err
		}
		writeMu.Lock()
		defer writeMu.Unlock()
		writeCtx, cancelWrite := context.WithTimeout(ctx, 2*time.Second)
		defer cancelWrite()
		return conn.Write(writeCtx, websocket.MessageText, raw)
	}

	hello, err := readGameEnvelope(ctx, conn, 5*time.Second)
	if err != nil {
		return fmt.Errorf("read game_hello: %w", err)
	}
	if hello.Type != protocol.TypeGameHello {
		return fmt.Errorf("first message = %q, want game_hello", hello.Type)
	}
	helloData, err := protocol.DecodeData[protocol.GameHelloData](hello)
	if err != nil {
		return fmt.Errorf("decode game_hello: %w", err)
	}
	logger.Printf("game_hello protocolVersion=%d", helloData.ProtocolVersion)

	configEnv, err := readGameEnvelope(ctx, conn, 5*time.Second)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	if configEnv.Type != protocol.TypeConfig {
		return fmt.Errorf("second message = %q, want config", configEnv.Type)
	}
	config, err := protocol.DecodeData[protocol.ConfigData](configEnv)
	if err != nil {
		return fmt.Errorf("decode config: %w", err)
	}
	logger.Printf("config roundDurationMs=%d tasksPerPlayer=%d maxPlayers=%d",
		config.RoundDurationMs, config.TasksPerPlayer, config.MaxPlayers)

	if err := send(protocol.TypeGameReady, protocol.GameReadyData{
		GameVersion: "fakegame",
		MapVersion:  "none",
	}); err != nil {
		return fmt.Errorf("send game_ready: %w", err)
	}
	logger.Printf("game_ready sent")

	counter := &inputCounter{logger: logger}
	go readLoop(ctx, conn, logger, counter, cancel)
	go func() {
		// cancel on return so a quit or stdin EOF ends the heartbeat and
		// closes the connection.
		defer cancel()
		heartbeatLoop(ctx, send)
	}()

	if opts.autoStart {
		go func() {
			if !sleep(ctx, time.Second) {
				return
			}
			if err := send(protocol.TypeHostRequest, protocol.HostRequestData{Command: "start"}); err != nil {
				logger.Printf("auto-start failed: %v", err)
				return
			}
			logger.Printf("auto-start host_request sent")
		}()
	}
	go func() {
		defer cancel()
		commandLoop(ctx, os.Stdin, logger, send)
	}()

	if opts.duration > 0 {
		go func() {
			if sleep(ctx, opts.duration) {
				logger.Printf("duration reached")
				cancel()
			}
		}()
	}

	<-ctx.Done()
	_ = conn.Close(websocket.StatusNormalClosure, "done")
	logger.Printf("disconnected")
	return nil
}

func readLoop(ctx context.Context, conn *websocket.Conn, logger *log.Logger, counter *inputCounter, cancel context.CancelFunc) {
	for {
		env, err := readGameEnvelope(ctx, conn, 0)
		if err != nil {
			if ctx.Err() == nil {
				logger.Printf("connection closed: %v", err)
			}
			cancel()
			return
		}

		switch env.Type {
		case protocol.TypeInput:
			data, err := protocol.DecodeData[protocol.ForwardedInputData](env)
			if err != nil {
				logger.Printf("bad input payload: %v", err)
				continue
			}
			counter.record(data)
		case protocol.TypePlayerJoined:
			data, _ := protocol.DecodeData[protocol.PlayerJoinedData](env)
			logger.Printf("player_joined slot=%d label=%s color=%s", data.Slot, data.Label, data.Color)
		case protocol.TypePlayerLeft:
			data, _ := protocol.DecodeData[protocol.PlayerLeftData](env)
			logger.Printf("player_left slot=%d reason=%s", data.Slot, data.Reason)
		case protocol.TypeHostCommand:
			data, _ := protocol.DecodeData[protocol.HostCommandData](env)
			logger.Printf("host_command command=%s slot=%d", data.Command, data.Slot)
		case protocol.TypeError:
			data, _ := protocol.DecodeData[protocol.ErrorData](env)
			logger.Printf("server error: code=%s message=%q", data.Code, data.Message)
		case protocol.TypeGameHello:
			logger.Printf("game_hello received")
		default:
			logger.Printf("message type=%s", env.Type)
		}
	}
}

func heartbeatLoop(ctx context.Context, send func(string, any) error) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	start := time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := send(protocol.TypeHeartbeat, protocol.HeartbeatData{GameTimeMs: time.Since(start).Milliseconds()}); err != nil {
				return
			}
		}
	}
}

func commandLoop(ctx context.Context, in *os.File, logger *log.Logger, send func(string, any) error) {
	logger.Printf("commands: start, pause, resume, reset, end, kick <slot>, phase <name> [timeLeftMs] [roundId], quit")
	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if ctx.Err() != nil {
			return
		}
		fields := strings.Fields(line)
		switch fields[0] {
		case "quit", "exit":
			return
		case "start", "pause", "resume", "reset", "end":
			if err := send(protocol.TypeHostRequest, protocol.HostRequestData{Command: fields[0]}); err != nil {
				logger.Printf("host_request failed: %v", err)
			} else {
				logger.Printf("host_request command=%s sent", fields[0])
			}
		case "kick":
			if len(fields) != 2 {
				logger.Printf("usage: kick <slot>")
				continue
			}
			slot, err := strconv.Atoi(fields[1])
			if err != nil {
				logger.Printf("invalid slot %q", fields[1])
				continue
			}
			if err := send(protocol.TypeHostRequest, protocol.HostRequestData{Command: "kick", Slot: slot}); err != nil {
				logger.Printf("host_request failed: %v", err)
			} else {
				logger.Printf("host_request command=kick slot=%d sent", slot)
			}
		case "phase":
			if len(fields) < 2 {
				logger.Printf("usage: phase <name> [timeLeftMs] [roundId]")
				continue
			}
			phase := protocol.Phase(fields[1])
			if !phase.Valid() {
				logger.Printf("invalid phase %q", fields[1])
				continue
			}
			data := protocol.PhaseData{Phase: phase}
			if len(fields) >= 3 {
				value, err := strconv.ParseInt(fields[2], 10, 64)
				if err != nil {
					logger.Printf("invalid timeLeftMs %q", fields[2])
					continue
				}
				data.TimeLeftMs = value
			}
			if len(fields) >= 4 {
				value, err := strconv.ParseInt(fields[3], 10, 64)
				if err != nil {
					logger.Printf("invalid roundId %q", fields[3])
					continue
				}
				data.RoundID = value
			}
			if err := send(protocol.TypePhase, data); err != nil {
				logger.Printf("phase failed: %v", err)
			} else {
				logger.Printf("phase %s sent", phase)
			}
		default:
			logger.Printf("unknown command %q", fields[0])
		}
	}
}

func readGameEnvelope(ctx context.Context, conn *websocket.Conn, timeout time.Duration) (protocol.Envelope, error) {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	typ, raw, err := conn.Read(ctx)
	if err != nil {
		return protocol.Envelope{}, err
	}
	if typ != websocket.MessageText {
		return protocol.Envelope{}, fmt.Errorf("unexpected websocket message type %v", typ)
	}
	return protocol.ParseEnvelope(raw)
}

func gameURL(server, token string) (string, error) {
	parsed, err := url.Parse(server)
	if err != nil {
		return "", fmt.Errorf("parse server URL: %w", err)
	}
	query := parsed.Query()
	query.Set("token", token)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func (c *inputCounter) record(data protocol.ForwardedInputData) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if data.Slot < 1 || data.Slot > 4 {
		c.logger.Printf("input with invalid slot=%d", data.Slot)
		return
	}
	c.counts[data.Slot]++
	if c.counts[data.Slot]%60 == 0 {
		c.logger.Printf("slot %d input #%d seq=%d steer=%.2f throttle=%.2f",
			data.Slot, c.counts[data.Slot], data.Seq, data.Steer, data.Throttle)
	}
}

func sleep(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
