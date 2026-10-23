package protocol

import (
	"errors"
	"testing"
)

func TestParseEnvelope(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		wantErr error
		check   func(t *testing.T, env Envelope)
	}{
		{
			name: "valid input envelope",
			raw:  `{"v":1,"type":"input","seq":42,"data":{"steer":-0.5,"throttle":1}}`,
			check: func(t *testing.T, env Envelope) {
				t.Helper()
				if env.V != Version {
					t.Fatalf("V = %d, want %d", env.V, Version)
				}
				if env.Type != TypeInput {
					t.Fatalf("Type = %q, want %q", env.Type, TypeInput)
				}
				if env.Seq == nil || *env.Seq != 42 {
					t.Fatalf("Seq = %v, want 42", env.Seq)
				}
			},
		},
		{
			name: "seq is optional",
			raw:  `{"v":1,"type":"ping","data":{}}`,
			check: func(t *testing.T, env Envelope) {
				t.Helper()
				if env.Seq != nil {
					t.Fatalf("Seq = %v, want nil", *env.Seq)
				}
			},
		},
		{
			name: "unknown fields are ignored",
			raw:  `{"v":1,"type":"hello","seq":1,"data":{},"future":true}`,
			check: func(t *testing.T, env Envelope) {
				t.Helper()
				if env.Type != TypeHello {
					t.Fatalf("Type = %q, want %q", env.Type, TypeHello)
				}
			},
		},
		{
			name:    "malformed json",
			raw:     `{"v":1,`,
			wantErr: ErrMalformed,
		},
		{
			name:    "missing version",
			raw:     `{"type":"hello","data":{}}`,
			wantErr: ErrUnsupportedVersion,
		},
		{
			name:    "unsupported version",
			raw:     `{"v":2,"type":"hello","data":{}}`,
			wantErr: ErrUnsupportedVersion,
		},
		{
			name:    "missing type",
			raw:     `{"v":1,"data":{}}`,
			wantErr: ErrMalformed,
		},
		{
			name:    "data must be an object when present",
			raw:     `{"v":1,"type":"input","data":[]}`,
			wantErr: ErrMalformed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			env, err := ParseEnvelope([]byte(tt.raw))
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, env)
			}
		})
	}
}

func TestDecodeData(t *testing.T) {
	t.Parallel()

	t.Run("decodes known fields and ignores unknown fields", func(t *testing.T) {
		t.Parallel()

		env, err := ParseEnvelope([]byte(`{"v":1,"type":"hello","data":{"deviceToken":"abc-123","appVersion":"0.1.0","extra":"ignored"}}`))
		if err != nil {
			t.Fatalf("ParseEnvelope: %v", err)
		}

		got, err := DecodeData[HelloData](env)
		if err != nil {
			t.Fatalf("DecodeData: %v", err)
		}
		if got.DeviceToken != "abc-123" {
			t.Fatalf("DeviceToken = %q, want %q", got.DeviceToken, "abc-123")
		}
	})

	t.Run("rejects missing data", func(t *testing.T) {
		t.Parallel()

		env := Envelope{V: Version, Type: TypeInput}
		if _, err := DecodeData[InputData](env); !errors.Is(err, ErrMalformed) {
			t.Fatalf("error = %v, want %v", err, ErrMalformed)
		}
	})

	t.Run("rejects wrong payload type", func(t *testing.T) {
		t.Parallel()

		env := Envelope{V: Version, Type: TypeInput, Data: []byte(`{"steer":[]}`)}
		if _, err := DecodeData[InputData](env); !errors.Is(err, ErrMalformed) {
			t.Fatalf("error = %v, want %v", err, ErrMalformed)
		}
	})
}

func TestEncode(t *testing.T) {
	t.Parallel()

	seq := int64(7)
	raw, err := Encode(TypeWelcome, &seq, WelcomeData{
		Slot:  2,
		Label: "P2",
		Color: "blue",
		Phase: "ready",
	})
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	env, err := ParseEnvelope(raw)
	if err != nil {
		t.Fatalf("ParseEnvelope: %v", err)
	}
	if env.V != Version || env.Type != TypeWelcome {
		t.Fatalf("envelope = %+v", env)
	}
	if env.Seq == nil || *env.Seq != seq {
		t.Fatalf("Seq = %v, want %d", env.Seq, seq)
	}

	got, err := DecodeData[WelcomeData](env)
	if err != nil {
		t.Fatalf("DecodeData: %v", err)
	}
	if got != (WelcomeData{Slot: 2, Label: "P2", Color: "blue", Phase: "ready"}) {
		t.Fatalf("data = %+v", got)
	}
}

func TestValidPhase(t *testing.T) {
	t.Parallel()

	for _, phase := range []Phase{PhaseAttract, PhaseReady, PhaseCountdown, PhasePlaying, PhasePaused, PhaseResults, PhaseEnded} {
		if !phase.Valid() {
			t.Errorf("%q should be valid", phase)
		}
	}
	if Phase("unknown").Valid() {
		t.Fatal("unknown phase should not be valid")
	}
}
