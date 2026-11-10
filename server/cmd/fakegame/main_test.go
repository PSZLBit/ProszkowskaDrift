package main

import (
	"context"
	"io"
	"log"
	"os"
	"testing"
	"time"
)

func TestCommandLoopEOFDoesNotReturnUntilContextCancelled(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe writer: %v", err)
	}
	defer r.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := log.New(io.Discard, "", 0)
	done := make(chan struct{})
	go func() {
		commandLoop(ctx, r, logger, func(string, any) error { return nil })
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("commandLoop returned before context cancellation on stdin EOF")
	case <-time.After(100 * time.Millisecond):
	}

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("commandLoop did not return after context cancellation")
	}
}

func TestResolveServerURL(t *testing.T) {
	if got := resolveServerURL(defaultServerURL, "ws://127.0.0.1:8099/ws/game"); got != "ws://127.0.0.1:8099/ws/game" {
		t.Fatalf("env override not applied: got %q", got)
	}
	if got := resolveServerURL("ws://example/ws/game", "ws://127.0.0.1:8099/ws/game"); got != "ws://example/ws/game" {
		t.Fatalf("explicit flag overridden: got %q", got)
	}
	if got := resolveServerURL(defaultServerURL, ""); got != defaultServerURL {
		t.Fatalf("empty env changed value: got %q", got)
	}
}
