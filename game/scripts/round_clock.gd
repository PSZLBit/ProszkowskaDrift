class_name RoundClock
extends Node
## MFD round timer (D-38, issue #47).
##
## A dumb countdown owned by the round orchestration in
## `game/scripts/server_input.gd`. The server sends `host_command start` with
## the round id, and the game config carries `roundDurationMs`; the clock never
## starts, pauses, or resets itself. Every state change emits `time_changed`
## so the HUD always shows the current value.

signal time_changed(remaining_ms: int)
signal phase_changed(phase: int)
signal finished

enum Phase { READY, PLAYING, PAUSED, FINISHED, RESULTS }

@export var round_length_ms: int = 180000

var phase: Phase = Phase.READY
var remaining_ms: int = 180000

var _remainder_ms: float = 0.0


func _init() -> void:
	remaining_ms = round_length_ms


func _ready() -> void:
	remaining_ms = round_length_ms


func start() -> void:
	if phase in [Phase.PLAYING, Phase.PAUSED]:
		return
	remaining_ms = round_length_ms
	_remainder_ms = 0.0
	time_changed.emit(remaining_ms)
	_set_phase(Phase.PLAYING)


## Applies the server config duration. While not playing, the remaining time
## follows the new length so the HUD shows the configured round.
func configure(duration_ms: int) -> void:
	round_length_ms = maxi(1, duration_ms)
	if phase != Phase.PLAYING and phase != Phase.PAUSED:
		remaining_ms = round_length_ms
		_remainder_ms = 0.0
		time_changed.emit(remaining_ms)


func pause() -> void:
	if phase == Phase.PLAYING:
		_set_phase(Phase.PAUSED)


func resume() -> void:
	if phase == Phase.PAUSED:
		_set_phase(Phase.PLAYING)


func reset() -> void:
	remaining_ms = round_length_ms
	_remainder_ms = 0.0
	time_changed.emit(remaining_ms)
	_set_phase(Phase.READY)


func show_results() -> void:
	# Both the timer expiry and the all-completed early finish land here; the
	# round controller stops the clock first either way.
	if phase in [Phase.FINISHED, Phase.PLAYING]:
		_set_phase(Phase.RESULTS)


func tick(delta_seconds: float) -> void:
	if phase != Phase.PLAYING:
		return
	if delta_seconds <= 0.0 or not is_finite(delta_seconds):
		return
	var ms := delta_seconds * 1000.0
	_remainder_ms += ms
	var whole := int(_remainder_ms)
	_remainder_ms -= float(whole)
	remaining_ms = maxi(0, remaining_ms - whole)
	time_changed.emit(remaining_ms)
	if remaining_ms <= 0:
		remaining_ms = 0
		_set_phase(Phase.FINISHED)
		finished.emit()


func _set_phase(next: Phase) -> void:
	if next == phase:
		return
	phase = next
	phase_changed.emit(int(phase))
