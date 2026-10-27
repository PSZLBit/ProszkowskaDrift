extends RefCounted
## Latest validated input per player slot.
##
## One store is shared by the single server connection and all four cars. Each
## slot has its own timeout and sequence, so one player leaving or being kicked
## cannot stop the other cars.

const INPUT_TIMEOUT_MS := 500
const MAX_PLAYERS := 4

var _states: Dictionary = {}


## Stores a validated sample. Returns false for an invalid slot or a stale
## sequence. The caller resets a slot on player_joined, which handles the
## sequence reset that follows a controller reconnect.
func set_input(slot: int, steer: float, throttle: float, seq: int, now_ms: int) -> bool:
	if slot < 1 or slot > MAX_PLAYERS:
		return false

	if _states.has(slot):
		var current: Dictionary = _states[slot]
		if seq >= 0 and int(current.get("seq", -1)) >= seq:
			return false

	_states[slot] = {
		"steer": clampf(steer, -1.0, 1.0),
		"throttle": clampf(throttle, -1.0, 1.0),
		"seq": seq,
		"last_ms": now_ms,
	}
	return true


## Returns the latest (steer, throttle) for a slot, or neutral input when the
## slot has no sample or its sample is at least INPUT_TIMEOUT_MS old.
func get_input(slot: int, now_ms: int) -> Vector2:
	if not _states.has(slot):
		return Vector2.ZERO

	var state: Dictionary = _states[slot]
	if now_ms - int(state["last_ms"]) >= INPUT_TIMEOUT_MS:
		return Vector2.ZERO
	return Vector2(float(state["steer"]), float(state["throttle"]))


## True when the slot has a sample inside the timeout window, including a
## deliberately neutral sample.
func has_recent_input(slot: int, now_ms: int) -> bool:
	if not _states.has(slot):
		return false
	return now_ms - int(_states[slot]["last_ms"]) < INPUT_TIMEOUT_MS


func get_last_seq(slot: int) -> int:
	if not _states.has(slot):
		return -1
	return int(_states[slot]["seq"])


func clear_slot(slot: int) -> void:
	_states.erase(slot)


func clear_all() -> void:
	_states.clear()


func active_slots(now_ms: int) -> Array[int]:
	var slots: Array[int] = []
	for slot in _states.keys():
		if has_recent_input(int(slot), now_ms):
			slots.append(int(slot))
	slots.sort()
	return slots
