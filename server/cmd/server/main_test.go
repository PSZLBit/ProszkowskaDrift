package main

import "testing"

func TestGameURLUsesLoopbackHost(t *testing.T) {
	cases := map[string]string{
		"0.0.0.0:8080":   "ws://127.0.0.1:8080/ws/game",
		":9090":          "ws://127.0.0.1:9090/ws/game",
		"192.168.1.5:80": "ws://127.0.0.1:80/ws/game",
		"[::]:8080":      "ws://127.0.0.1:8080/ws/game",
		"127.0.0.1":      "ws://127.0.0.1:8080/ws/game",
	}
	for listen, want := range cases {
		if got := gameURL(listen); got != want {
			t.Errorf("gameURL(%q) = %q, want %q", listen, got, want)
		}
	}
}
