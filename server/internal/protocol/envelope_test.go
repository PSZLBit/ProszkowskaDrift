package protocol

import (
	"errors"
	"testing"
)

func TestParseEnvelope(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		wantCode string
	}{
		{name: "valid", raw: `{"v":1,"type":"ping","data":{}}`},
		{name: "valid with seq", raw: `{"v":1,"type":"input","seq":7,"data":{"steer":0,"throttle":0}}`},
		{name: "missing data becomes empty object", raw: `{"v":1,"type":"leave"}`},
		{name: "wrong version", raw: `{"v":2,"type":"ping","data":{}}`, wantCode: CodeProtocolMismatch},
		{name: "missing version", raw: `{"type":"ping","data":{}}`, wantCode: CodeProtocolMismatch},
		{name: "missing type", raw: `{"v":1,"data":{}}`, wantCode: CodeInvalidInput},
		{name: "malformed json", raw: `{"v":1`, wantCode: CodeInvalidInput},
		{name: "data is array", raw: `{"v":1,"type":"ping","data":[]}`, wantCode: CodeInvalidInput},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env, err := ParseEnvelope([]byte(tt.raw))
			if tt.wantCode == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if env.V != Version {
					t.Fatalf("version = %d", env.V)
				}
				return
			}
			var pe *Error
			if !errors.As(err, &pe) {
				t.Fatalf("error = %v, want protocol error", err)
			}
			if pe.Code != tt.wantCode {
				t.Fatalf("code = %q, want %q", pe.Code, tt.wantCode)
			}
		})
	}
}

func TestDecodeDataIgnoresUnknownFields(t *testing.T) {
	env, err := ParseEnvelope([]byte(`{"v":1,"type":"hello","data":{"deviceToken":"abc","future":true}}`))
	if err != nil {
		t.Fatal(err)
	}
	hello, err := DecodeData[HelloData](env)
	if err != nil {
		t.Fatal(err)
	}
	if hello.DeviceToken != "abc" {
		t.Fatalf("token = %q", hello.DeviceToken)
	}
}

func TestEncodeMessage(t *testing.T) {
	raw, err := EncodeMessage(Version, TypePong, nil, PongData{ClientTime: 1, ServerTime: 2})
	if err != nil {
		t.Fatal(err)
	}
	env, err := ParseEnvelope(raw)
	if err != nil {
		t.Fatal(err)
	}
	pong, err := DecodeData[PongData](env)
	if err != nil {
		t.Fatal(err)
	}
	if pong.ClientTime != 1 || pong.ServerTime != 2 {
		t.Fatalf("pong = %+v", pong)
	}

	raw, err = EncodeMessage(Version, TypeLeave, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	env, err = ParseEnvelope(raw)
	if err != nil {
		t.Fatal(err)
	}
	if string(env.Data) != "{}" {
		t.Fatalf("empty data = %s", env.Data)
	}
}

func TestTaskResultWithSlotDecodes(t *testing.T) {
	env, err := ParseEnvelope([]byte(`{"v":1,"type":"task_result","data":{"slot":2,"taskId":12,"points":86,"timeScore":62,"precisionScore":34,"collisionPenalty":10,"rating":"great","totalScore":240}}`))
	if err != nil {
		t.Fatal(err)
	}
	result, err := DecodeData[TaskResultWithSlot](env)
	if err != nil {
		t.Fatal(err)
	}
	if result.Slot != 2 || result.TaskID != 12 || result.TotalScore != 240 {
		t.Fatalf("result = %+v", result)
	}
}

func TestValidPhaseAndHostCommand(t *testing.T) {
	for _, phase := range []string{PhaseAttract, PhaseReady, PhaseCountdown, PhasePlaying, PhasePaused, PhaseResults, PhaseEnded} {
		if !ValidPhase(phase) {
			t.Fatalf("phase %q should be valid", phase)
		}
	}
	if ValidPhase("unknown") {
		t.Fatal("unknown phase accepted")
	}
	for _, command := range []string{HostStart, HostPause, HostResume, HostReset, HostEnd, HostKick} {
		if !ValidHostCommand(command) {
			t.Fatalf("command %q should be valid", command)
		}
	}
	if ValidHostCommand("explode") {
		t.Fatal("unknown host command accepted")
	}
	for _, state := range []string{StateReady, StatePlaying, StateFinished, StateReconnecting} {
		if !ValidFeedbackState(state) {
			t.Fatalf("state %q should be valid", state)
		}
	}
	if ValidFeedbackState("unknown") {
		t.Fatal("unknown feedback state accepted")
	}
}
