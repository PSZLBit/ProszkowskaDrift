// Command fakecontroller exercises the controller WebSocket endpoint without
// the Android app. It is a Sprint 0 manual-test tool, not production code.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/coder/websocket"

	"github.com/PSZLBit/ProszkowskaDrift/server/internal/protocol"
)

type options struct {
	server       string
	clients      int
	duration     time.Duration
	rate         time.Duration
	tokenPrefix  string
	invalidAfter time.Duration
	dropAfter    time.Duration
	silentAfter  time.Duration
}

type action int

const (
	actionDone action = iota
	actionReconnect
)

func main() {
	log.SetFlags(0)

	opts := options{}
	flag.StringVar(&opts.server, "server", "ws://127.0.0.1:8080/ws/controller", "controller WebSocket URL")
	flag.IntVar(&opts.clients, "clients", 4, "number of fake controllers")
	flag.DurationVar(&opts.duration, "duration", 12*time.Second, "how long each fake controller runs")
	flag.DurationVar(&opts.rate, "rate", 33*time.Millisecond, "input send interval (about 30 Hz)")
	flag.StringVar(&opts.tokenPrefix, "token-prefix", "fake", "prefix for generated device tokens")
	flag.DurationVar(&opts.invalidAfter, "invalid-after", 0, "send one out-of-range input after this duration (0 disables)")
	flag.DurationVar(&opts.dropAfter, "drop-after", 0, "drop and reconnect player 1 after this duration (0 disables)")
	flag.DurationVar(&opts.silentAfter, "silent-after", 0, "player 1 stops ping/input after this duration to hit the heartbeat timeout (0 disables)")
	flag.Parse()

	if err := validate(opts); err != nil {
		fmt.Fprintln(os.Stderr, "fakecontroller:", err)
		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup
	errs := make(chan error, opts.clients)
	for i := 1; i <= opts.clients; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			if err := runClient(ctx, opts, index); err != nil {
				errs <- fmt.Errorf("player %d: %w", index, err)
			}
		}(i)
	}
	wg.Wait()
	close(errs)

	failed := false
	for err := range errs {
		log.Printf("error: %v", err)
		failed = true
	}
	if failed {
		os.Exit(1)
	}
}

func validate(opts options) error {
	if opts.clients < 1 {
		return fmt.Errorf("clients must be at least 1")
	}
	if opts.duration <= 0 {
		return fmt.Errorf("duration must be positive")
	}
	if opts.rate <= 0 {
		return fmt.Errorf("rate must be positive")
	}
	if opts.rate < 16*time.Millisecond {
		return fmt.Errorf("rate must be at most about 60 Hz (use at least 16ms)")
	}
	return nil
}

func runClient(ctx context.Context, opts options, index int) error {
	token := fmt.Sprintf("%s-%02d-%s", opts.tokenPrefix, index, randomHex(8))
	logger := log.New(os.Stdout, fmt.Sprintf("[P%d] ", index), 0)
	deadline := time.Now().Add(opts.duration)
	attempt := 0
	dropDone := opts.dropAfter <= 0 || index != 1
	invalidSent := opts.invalidAfter <= 0

	for {
		if ctx.Err() != nil || time.Now().After(deadline) {
			return nil
		}
		attempt++

		conn, _, err := websocket.Dial(ctx, opts.server, nil)
		if err != nil {
			logger.Printf("connect attempt %d failed: %v", attempt, err)
			if !sleep(ctx, time.Second) {
				return nil
			}
			continue
		}

		welcome, err := handshake(ctx, conn, token)
		if err != nil {
			logger.Printf("handshake failed: %v", err)
			_ = conn.CloseNow()
			if !sleep(ctx, time.Second) {
				return nil
			}
			continue
		}
		logger.Printf("connected as slot %d (%s, %s), phase=%s, token=%s",
			welcome.Slot, welcome.Label, welcome.Color, welcome.Phase, truncate(token))

		readCtx, cancelRead := context.WithCancel(ctx)
		readDone := make(chan struct{})
		go func() {
			defer close(readDone)
			readLoop(readCtx, conn, logger)
		}()

		next := sendLoop(ctx, deadline, conn, logger, index, opts, &dropDone, &invalidSent, readDone)
		if next == actionDone {
			_ = writeEnvelope(ctx, conn, protocol.TypeLeave, nil, struct{}{})
			_ = conn.Close(websocket.StatusNormalClosure, "done")
			cancelRead()
			waitClosed(readDone)
			logger.Printf("session finished; slot stays reserved on the server")
			return nil
		}

		// Simulate an abrupt network drop, then reconnect with the same token.
		_ = conn.CloseNow()
		cancelRead()
		waitClosed(readDone)
		logger.Printf("dropped the connection; reconnecting with the same token")
		if !sleep(ctx, time.Second) || time.Now().After(deadline) {
			return nil
		}
	}
}

func handshake(ctx context.Context, conn *websocket.Conn, token string) (protocol.WelcomeData, error) {
	if err := writeEnvelope(ctx, conn, protocol.TypeHello, nil, protocol.HelloData{
		DeviceToken: token,
		AppVersion:  "fakecontroller",
	}); err != nil {
		return protocol.WelcomeData{}, err
	}

	readCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	for {
		env, err := readEnvelope(readCtx, conn)
		if err != nil {
			return protocol.WelcomeData{}, err
		}
		switch env.Type {
		case protocol.TypeWelcome:
			return protocol.DecodeData[protocol.WelcomeData](env)
		case protocol.TypeError:
			data, _ := protocol.DecodeData[protocol.ErrorData](env)
			return protocol.WelcomeData{}, fmt.Errorf("%s: %s", data.Code, data.Message)
		default:
			// Ignore a phase broadcast that races with the welcome.
		}
	}
}

func readLoop(ctx context.Context, conn *websocket.Conn, logger *log.Logger) {
	for {
		env, err := readEnvelope(ctx, conn)
		if err != nil {
			if ctx.Err() == nil {
				logger.Printf("connection closed: %v", err)
			}
			return
		}

		switch env.Type {
		case protocol.TypeWelcome:
			data, _ := protocol.DecodeData[protocol.WelcomeData](env)
			logger.Printf("welcome slot=%d label=%s color=%s phase=%s", data.Slot, data.Label, data.Color, data.Phase)
		case protocol.TypePhase:
			data, err := protocol.DecodeData[protocol.PhaseData](env)
			if err != nil {
				logger.Printf("bad phase payload: %v", err)
				continue
			}
			logger.Printf("phase=%s round=%d timeLeftMs=%d", data.Phase, data.RoundID, data.TimeLeftMs)
		case protocol.TypeError:
			data, _ := protocol.DecodeData[protocol.ErrorData](env)
			logger.Printf("server error: code=%s message=%q", data.Code, data.Message)
		case protocol.TypePong:
			// Pongs are expected every two seconds; keep the output quiet.
		default:
			logger.Printf("message type=%s", env.Type)
		}
	}
}

func sendLoop(ctx context.Context, deadline time.Time, conn *websocket.Conn, logger *log.Logger, index int, opts options, dropDone, invalidSent *bool, readDone <-chan struct{}) action {
	start := time.Now()
	inputTicker := time.NewTicker(opts.rate)
	defer inputTicker.Stop()
	pingTicker := time.NewTicker(2 * time.Second)
	defer pingTicker.Stop()
	durationTimer := time.NewTimer(time.Until(deadline))
	defer durationTimer.Stop()

	silentDone := opts.silentAfter <= 0 || index != 1
	silent := false
	var seq int64

	for {
		select {
		case <-ctx.Done():
			return actionDone
		case <-readDone:
			return actionDone
		case <-durationTimer.C:
			logger.Printf("duration reached")
			return actionDone
		case <-inputTicker.C:
			if silent {
				continue
			}
			elapsed := time.Since(start)

			if !silentDone && elapsed >= opts.silentAfter {
				silent = true
				silentDone = true
				logger.Printf("stopped ping/input; waiting for the server heartbeat timeout")
				continue
			}
			if !*dropDone && elapsed >= opts.dropAfter {
				*dropDone = true
				logger.Printf("simulating disconnect after %s", elapsed.Round(time.Millisecond))
				return actionReconnect
			}

			seq++
			steer := 0.15 * float64(index)
			throttle := 0.6
			if !*invalidSent && opts.invalidAfter > 0 && elapsed >= opts.invalidAfter {
				*invalidSent = true
				steer = 2
				logger.Printf("sending one invalid input seq=%d steer=2", seq)
			}
			if err := writeEnvelope(ctx, conn, protocol.TypeInput, &seq, protocol.InputData{
				Steer:    &steer,
				Throttle: &throttle,
				TS:       elapsed.Milliseconds(),
			}); err != nil {
				logger.Printf("input write failed: %v", err)
				return actionDone
			}
		case <-pingTicker.C:
			if silent {
				continue
			}
			clientTime := time.Since(start).Milliseconds()
			if err := writeEnvelope(ctx, conn, protocol.TypePing, nil, protocol.PingData{ClientTime: clientTime}); err != nil {
				logger.Printf("ping write failed: %v", err)
				return actionDone
			}
		}
	}
}

func readEnvelope(ctx context.Context, conn *websocket.Conn) (protocol.Envelope, error) {
	typ, raw, err := conn.Read(ctx)
	if err != nil {
		return protocol.Envelope{}, err
	}
	if typ != websocket.MessageText {
		return protocol.Envelope{}, fmt.Errorf("unexpected websocket message type %v", typ)
	}
	return protocol.ParseEnvelope(raw)
}

func writeEnvelope(ctx context.Context, conn *websocket.Conn, typ string, seq *int64, data any) error {
	raw, err := protocol.Encode(typ, seq, data)
	if err != nil {
		return err
	}
	writeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return conn.Write(writeCtx, websocket.MessageText, raw)
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

func waitClosed(done <-chan struct{}) {
	select {
	case <-done:
	case <-time.After(2 * time.Second):
	}
}

func randomHex(bytes int) string {
	raw := make([]byte, bytes)
	if _, err := rand.Read(raw); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(raw)
}

func truncate(token string) string {
	if len(token) < 8 {
		return "***"
	}
	return token[:4] + "..." + token[len(token)-2:]
}
