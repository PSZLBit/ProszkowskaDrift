extends RefCounted
## Exclusive MFD task assignment (D-38).
##
## At round start every connected slot receives one task. Active slots are
## processed in ascending order and the starting spot rotates per round:
## start = (roundId - 1) mod 4, spot = (start + i) mod 4. No two active
## players share a spot. Disconnected, empty, and kicked slots are never
## assigned. There is no mid-round assignment and no auto-skip in the MFD.

const MfdMap := preload("res://scripts/mfd_map.gd")


## Assigns one task per active slot. Returns {slot: assignment} where each
## assignment holds taskId, spotIndex, spotLabel, hint, and deadlineMs.
static func assign(active_slots: Array, round_id: int, round_duration_ms: int) -> Dictionary:
	var valid: Array[int] = []
	for slot_raw in active_slots:
		var slot := int(slot_raw)
		if slot >= 1 and slot <= 4 and not valid.has(slot):
			valid.append(slot)
	valid.sort()

	var count := MfdMap.spot_count()
	var start := posmod(round_id - 1, count)
	var tasks := {}
	var i := 0
	for slot in valid:
		var spot_index := posmod(start + i, count)
		tasks[slot] = make_task(slot, spot_index, round_id, round_duration_ms)
		i += 1
	return tasks


static func make_task(_slot: int, spot_index: int, round_id: int, round_duration_ms: int) -> Dictionary:
	return {
		"taskId": task_id(round_id, spot_index),
		"spotIndex": spot_index,
		"spotLabel": MfdMap.spot_label(spot_index),
		"hint": "north",
		"deadlineMs": round_duration_ms,
	}


static func task_id(round_id: int, spot_index: int) -> int:
	return round_id * 10 + (spot_index + 1)
