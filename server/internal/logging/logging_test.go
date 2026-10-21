package logging

import "testing"

func TestRedactToken(t *testing.T) {
	tests := map[string]string{
		"":           "",
		"short":      "***",
		"12345678":   "***",
		"abcdefghij": "abcd...ij",
	}
	for input, want := range tests {
		if got := RedactToken(input); got != want {
			t.Fatalf("RedactToken(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestParseLevel(t *testing.T) {
	for _, level := range []string{"debug", "info", "warn", "error", "INFO", ""} {
		_ = parseLevel(level)
	}
}
