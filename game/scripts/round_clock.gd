class_name RoundClock
extends Node
## Provisional round timer for the vertical slice (#47).
##
## Phase behaviour and the 180 s length are TBD pending #45; this scaffold
## exposes the interface the HUD and game loop depend on and is covered by
## headless tests. It is not the final round controller.

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
	_set_phase(Phase.PLAYING)


func pause() -> void:
	if phase == Phase.PLAYING:
		_set_phase(Phase.PAUSED)


func resume() -> void:
	if phase == Phase.PAUSED:
		_set_phase(Phase.PLAYING)


func reset() -> void:
	remaining_ms = round_length_ms
	_remainder_ms = 0.0
	_set_phase(Phase.READY)


func show_results() -> void:
	if phase == Phase.FINISHED:
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
