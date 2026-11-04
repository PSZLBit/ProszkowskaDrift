class_name ScoreModel
extends RefCounted
## Provisional scoring for the vertical slice (#47).
##
## The formula and constants are TBD pending #45. These values are placeholders
## for the scaffold and its tests; they are not the final scoring rules.

const COLLISION_PENALTY := 50
const COMPLETION_BONUS := 500
const PRECISION_SCALE := 100

var round_length_ms: int = 180000

var _players: Dictionary = {}


func register(slot: int) -> void:
	if slot < 1 or slot > 4:
		return
	_players[slot] = {
		"finish_time_ms": -1,
		"precision": 0.0,
		"collisions": 0,
		"finished": false,
	}


func has_player(slot: int) -> bool:
	return _players.has(slot)


func record_collision(slot: int) -> void:
	if not has_player(slot):
		return
	_players[slot]["collisions"] = int(_players[slot]["collisions"]) + 1


func set_precision(slot: int, value: float) -> void:
	if not has_player(slot):
		return
	_players[slot]["precision"] = clampf(value, 0.0, 1.0)


func finish(slot: int, finish_time_ms: int) -> void:
	if not has_player(slot):
		return
	_players[slot]["finished"] = true
	_players[slot]["finish_time_ms"] = maxi(0, finish_time_ms)


func total(slot: int) -> int:
	if not has_player(slot):
		return 0
	var p: Dictionary = _players[slot]
	if not bool(p["finished"]):
		return 0
	var time_score := round_length_ms - int(p["finish_time_ms"])
	if time_score < 0:
		time_score = 0
	var precision_score := int(round(float(p["precision"]) * PRECISION_SCALE))
	var collision_penalty := int(p["collisions"]) * COLLISION_PENALTY
	return time_score + precision_score - collision_penalty + COMPLETION_BONUS


func ranking() -> Array[int]:
	var slots: Array[int] = []
	for key in _players.keys():
		slots.append(int(key))
	slots.sort_custom(func(a: int, b: int) -> bool:
		var ta := total(a)
		var tb := total(b)
		if ta != tb:
			return ta > tb
		var ca := int(_players[a]["collisions"])
		var cb := int(_players[b]["collisions"])
		if ca != cb:
			return ca < cb
		var fa := int(_players[a]["finish_time_ms"])
		var fb := int(_players[b]["finish_time_ms"])
		if fa != fb:
			if fa < 0:
				return false
			if fb < 0:
				return true
			return fa < fb
		return a < b
	)
	return slots
