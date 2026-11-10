class_name ScoreModel
extends RefCounted
## MFD scoring for one parking task per player (D-38, issue #47).
##
## Single implementation source is the MFD Baseline in
## documentation/02-game-domain/game-design.md. One task per player, no finish
## bonus. A player who completes the task keeps the task score; a player who
## does not complete scores 0 for the round.

## Half of the shorter bay side, in pixels (96 / 2).
const CENTER_HALF_PX := 48.0
## Collision penalty step and cap.
const COLLISION_STEP := 10
const MAX_COLLISION_PENALTY := 30
## Minimum time score floor.
const MIN_TIME_SCORE := 10

var _players: Dictionary = {}


## Registers one active slot for the round. Only registered slots appear in
## results. Registering twice keeps the existing entry.
func register(slot: int) -> bool:
	if slot < 1 or slot > 4:
		return false
	if not _players.has(slot):
		_players[slot] = _blank_entry()
	return true


func unregister(slot: int) -> void:
	_players.erase(slot)


func reset_round() -> void:
	_players.clear()


func has_player(slot: int) -> bool:
	return _players.has(slot)


## Records one task completion. Returns false for an unknown slot, a repeated
## completion, or non-finite/negative measurements.
func record_completion(
	slot: int,
	task_id: int,
	completion_ms: int,
	center_offset_px: float,
	angle_deg: float,
	collisions: int,
) -> bool:
	if not has_player(slot):
		return false
	var entry: Dictionary = _players[slot]
	if bool(entry["completed"]):
		return false
	if completion_ms < 0 or collisions < 0:
		return false
	if not is_finite(center_offset_px) or not is_finite(angle_deg):
		return false
	if center_offset_px < 0.0:
		return false

	var angle := clampf(absf(angle_deg), 0.0, 180.0)
	var time := compute_time_score(completion_ms)
	var center := compute_center_score(center_offset_px)
	var angle_score := compute_angle_score(angle)
	var precision := center + angle_score
	var penalty := compute_penalty(collisions)
	entry["completed"] = true
	entry["task_id"] = task_id
	entry["completion_ms"] = completion_ms
	entry["center_offset_px"] = center_offset_px
	entry["angle_deg"] = angle
	entry["collisions"] = collisions
	entry["time_score"] = time
	entry["precision_score"] = precision
	entry["center_score"] = center
	entry["angle_score"] = angle_score
	entry["collision_penalty"] = penalty
	entry["task_score"] = maxi(0, time + precision - penalty)
	return true


## Round score: the task score when completed, else 0.
func total(slot: int) -> int:
	if not has_player(slot):
		return 0
	var entry: Dictionary = _players[slot]
	if not bool(entry["completed"]):
		return 0
	return int(entry["task_score"])


## Completed tasks for the slot: 0 or 1 in the MFD.
func tasks_completed(slot: int) -> int:
	if not has_player(slot):
		return 0
	return 1 if bool(_players[slot]["completed"]) else 0


func collisions(slot: int) -> int:
	if not has_player(slot):
		return 0
	return int(_players[slot]["collisions"])


## Completion time in ms from round start, or -1 when not completed.
func completion_time_ms(slot: int) -> int:
	if not has_player(slot):
		return -1
	var entry: Dictionary = _players[slot]
	if not bool(entry["completed"]):
		return -1
	return int(entry["completion_ms"])


## First player in MFD ranking order, or 0 when nobody completed a task.
func winner_slot() -> int:
	var ordered := ranking()
	for slot in ordered:
		if tasks_completed(slot) > 0:
			return slot
	return 0


## MFD ranking: more completed tasks, higher score, fewer collisions, shorter
## completion time (a non-completing player is slower than any completing
## one), lower slot number.
func ranking() -> Array[int]:
	var slots: Array[int] = []
	for key in _players.keys():
		slots.append(int(key))
	slots.sort_custom(func(a: int, b: int) -> bool:
		var ca := tasks_completed(a)
		var cb := tasks_completed(b)
		if ca != cb:
			return ca > cb
		var ta := total(a)
		var tb := total(b)
		if ta != tb:
			return ta > tb
		var ka := collisions(a)
		var kb := collisions(b)
		if ka != kb:
			return ka < kb
		var fa := completion_time_ms(a)
		var fb := completion_time_ms(b)
		if fa != fb:
			return _completion_sort_key(fa) < _completion_sort_key(fb)
		return a < b
	)
	return slots


## Server -> controller task_result payload for one completed slot.
func task_result_payload(slot: int) -> Dictionary:
	if not has_player(slot):
		return {}
	var entry: Dictionary = _players[slot]
	if not bool(entry["completed"]):
		return {}
	var total_score := int(entry["task_score"])
	return {
		"taskId": int(entry["task_id"]),
		"points": total_score,
		"timeScore": int(entry["time_score"]),
		"precisionScore": int(entry["precision_score"]),
		"collisionPenalty": int(entry["collision_penalty"]),
		"rating": rating_for(int(entry["precision_score"])),
		"totalScore": total_score,
	}


## Server -> controller round_result payload for every registered slot.
func round_result_payload(round_id: int) -> Dictionary:
	var scores: Array = []
	for slot in ranking():
		scores.append({
			"slot": slot,
			"score": total(slot),
			"tasks": tasks_completed(slot),
			"collisions": collisions(slot),
		})
	return {
		"roundId": round_id,
		"winnerSlot": winner_slot(),
		"scores": scores,
	}


static func compute_time_score(elapsed_ms: int) -> int:
	return maxi(MIN_TIME_SCORE, 100 - (3 * elapsed_ms) / 1000)


static func compute_center_score(offset_px: float) -> int:
	return maxi(0, 20 - int(floor(20.0 * offset_px / CENTER_HALF_PX)))


static func compute_angle_score(angle_deg: float) -> int:
	if angle_deg <= 10.0:
		return 20
	if angle_deg >= 35.0:
		return 0
	return int(floor(20.0 * (35.0 - angle_deg) / 25.0))


static func compute_penalty(collisions: int) -> int:
	return mini(MAX_COLLISION_PENALTY, COLLISION_STEP * maxi(0, collisions))


## Precision rating for the task_result payload (displayed from the MVP).
static func rating_for(precision_score: int) -> String:
	if precision_score >= 35:
		return "perfect"
	if precision_score >= 25:
		return "great"
	if precision_score >= 15:
		return "good"
	return "parked"


static func _completion_sort_key(completion_ms: int) -> int:
	# A non-completing player (-1) sorts after every completing one.
	if completion_ms < 0:
		return 2147483647
	return completion_ms


static func _blank_entry() -> Dictionary:
	return {
		"completed": false,
		"task_id": 0,
		"completion_ms": -1,
		"center_offset_px": 0.0,
		"angle_deg": 0.0,
		"collisions": 0,
		"time_score": 0,
		"precision_score": 0,
		"center_score": 0,
		"angle_score": 0,
		"collision_penalty": 0,
		"task_score": 0,
	}
