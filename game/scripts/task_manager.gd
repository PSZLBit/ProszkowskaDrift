extends Node
## MFD task lifecycle for one parking task per player (D-38, issue #46).
##
## ServerInput starts a round with the active slots and the server roundId;
## this node assigns exclusive targets, tracks the 0.5 s parking hold with
## velocity.length() as the speed source, counts one collision per contact
## episode, and emits task_completed exactly once per slot. Scoring, timer,
## HUD, and results stay in #47; this signal is the hook #47 consumes.

signal task_completed(slot: int, result: Dictionary)
signal collisions_changed(slot: int, count: int)

const MfdMap := preload("res://scripts/mfd_map.gd")
const TaskAssignment := preload("res://scripts/task_assignment.gd")
const ParkingDetector := preload("res://scripts/parking_detector.gd")
const TargetSpot := preload("res://scripts/target_spot.gd")

var _tasks: Dictionary = {}
var _holds: Dictionary = {}
var _connected: Dictionary = {}
var _running := false
var _round_id := 0
var _beacons: Dictionary = {}
## Process timestamp (same origin as update_tasks now_ms) when the current
## pause started, or -1 when not paused. Paused intervals are shifted out of
## every assignedMs on resume so completionMs only counts active round time.
var _pause_started_ms := -1


func _ready() -> void:
	_wire_cars()


func set_running(running: bool, now_ms: int) -> void:
	if not running and _running:
		_pause_started_ms = now_ms
	elif running and not _running and _pause_started_ms >= 0:
		var paused_ms := now_ms - _pause_started_ms
		for slot in _tasks.keys():
			if str(_tasks[slot]["state"]) != "completed":
				_tasks[slot]["assignedMs"] = int(_tasks[slot]["assignedMs"]) + paused_ms
		_pause_started_ms = -1
	_running = running
	if not running:
		for slot in _holds.keys():
			var hold = _holds[slot]
			hold.reset()


func start_round(active_slots: Array, round_id: int, round_duration_ms: int, now_ms: int) -> void:
	clear_round()
	_round_id = round_id
	var assignments := TaskAssignment.assign(active_slots, round_id, round_duration_ms)
	for slot in assignments.keys():
		_tasks[slot] = {
			"taskId": assignments[slot]["taskId"],
			"spotIndex": assignments[slot]["spotIndex"],
			"spotLabel": assignments[slot]["spotLabel"],
			"hint": assignments[slot]["hint"],
			"deadlineMs": assignments[slot]["deadlineMs"],
			"spotRect": MfdMap.spot_rect(int(assignments[slot]["spotIndex"])),
			"spotFacing": MfdMap.spot_facing_deg(int(assignments[slot]["spotIndex"])),
			"state": "assigned",
			"collisions": 0,
			"assignedMs": now_ms,
		}
		_holds[slot] = ParkingDetector.Hold.new()
		_connected[slot] = true
	_spawn_beacons()


func clear_round() -> void:
	_tasks.clear()
	_holds.clear()
	_connected.clear()
	_round_id = 0
	_pause_started_ms = -1
	for slot in _beacons.keys():
		(_beacons[slot] as Node).queue_free()
	_beacons.clear()


func set_slot_connected(slot: int, connected: bool) -> void:
	if not _tasks.has(slot):
		return
	_connected[slot] = connected
	if not connected and _holds.has(slot):
		var hold = _holds[slot]
		hold.reset()


func release_slot(slot: int) -> void:
	_tasks.erase(slot)
	_holds.erase(slot)
	_connected.erase(slot)
	if _beacons.has(slot):
		(_beacons[slot] as Node).queue_free()
		_beacons.erase(slot)


func register_collision(slot: int) -> void:
	if not _tasks.has(slot):
		return
	if str(_tasks[slot]["state"]) == "completed":
		return
	_tasks[slot]["collisions"] = int(_tasks[slot]["collisions"]) + 1
	collisions_changed.emit(slot, int(_tasks[slot]["collisions"]))


func get_task(slot: int) -> Dictionary:
	return _tasks.get(slot, {})


func task_count() -> int:
	return _tasks.size()


## Called every physics frame with {slot: {position, rotation, speed}} where
## speed is velocity.length(). No completion is recorded while paused or
## stopped; a disconnected slot keeps its task but its hold timer resets.
func update_tasks(delta_sec: float, cars: Dictionary, now_ms: int) -> void:
	if not _running:
		return
	for slot in _tasks.keys():
		if str(_tasks[slot]["state"]) == "completed":
			continue
		if not bool(_connected.get(slot, false)):
			continue
		if not cars.has(slot):
			continue
		var car: Dictionary = cars[slot]
		var corners: Array = ParkingDetector.car_corners(
			car["position"], float(car["rotation"]), MfdMap.CAR_SIZE
		)
		var inside: bool = ParkingDetector.all_inside(corners, _tasks[slot]["spotRect"])
		var slow: bool = ParkingDetector.speed_ok(float(car["speed"]))
		var hold = _holds[slot]
		if hold.call("update", delta_sec, inside, slow):
			_complete(slot, car, now_ms)


func _complete(slot: int, car: Dictionary, now_ms: int) -> void:
	if str(_tasks[slot]["state"]) == "completed":
		return
	_tasks[slot]["state"] = "completed"
	var rect: Rect2 = _tasks[slot]["spotRect"]
	var car_pos: Vector2 = car["position"]
	var center_offset: float = car_pos.distance_to(rect.get_center())
	var angle_diff: float = ParkingDetector.angle_diff_deg(
		rad_to_deg(float(car["rotation"])), float(_tasks[slot]["spotFacing"])
	)
	var result := {
		"slot": slot,
		"roundId": _round_id,
		"taskId": int(_tasks[slot]["taskId"]),
		"spotLabel": str(_tasks[slot]["spotLabel"]),
		"completionMs": now_ms - int(_tasks[slot]["assignedMs"]),
		"centerOffset": center_offset,
		"angleDiff": angle_diff,
		"collisions": int(_tasks[slot]["collisions"]),
	}
	task_completed.emit(slot, result)


func _physics_process(delta: float) -> void:
	if not _running or _tasks.is_empty():
		return
	var cars := {}
	for node in _car_nodes():
		var body := node as CharacterBody2D
		if body == null:
			continue
		var slot := int(body.get("slot"))
		if _tasks.has(slot):
			cars[slot] = {
				"position": body.position,
				"rotation": body.rotation,
				"speed": body.velocity.length(),
			}
	if not cars.is_empty():
		update_tasks(delta, cars, Time.get_ticks_msec())


func _wire_cars() -> void:
	var parent := get_parent()
	if parent == null:
		return
	for node in parent.find_children("Car*", "CharacterBody2D", true, false):
		if node.has_signal("collision_started"):
			var slot := int(node.get("slot"))
			if not node.is_connected("collision_started", register_collision):
				node.connect("collision_started", register_collision.bind(slot))


func _car_nodes() -> Array:
	var parent := get_parent()
	if parent == null:
		return []
	return parent.find_children("Car*", "CharacterBody2D", true, false)


func _spawn_beacons() -> void:
	var parent := get_parent()
	if parent == null:
		return
	for slot in _tasks.keys():
		var beacon = TargetSpot.new()
		var rect: Rect2 = _tasks[slot]["spotRect"]
		beacon.position = rect.get_center()
		parent.add_child(beacon)
		# Above the fullscreen Background but below the cars: both use the
		# default Z index, so tree order decides what is visible.
		parent.move_child(beacon, 1)
		beacon.setup(
			str(_tasks[slot]["spotLabel"]),
			MfdMap.slot_color(int(slot)),
			rect.size,
		)
		_beacons[slot] = beacon
