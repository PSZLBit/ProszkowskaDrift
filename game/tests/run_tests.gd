extends SceneTree
## Headless tests for the game scaffold.
##
## Run with:
##   godot --headless --path game --import
##   godot --headless --path game --script res://tests/run_tests.gd

const GameProtocol := preload("res://scripts/game_protocol.gd")
const SlotInputStore := preload("res://scripts/slot_input_store.gd")
const CarScript := preload("res://scripts/car.gd")
const ServerInputScript := preload("res://scripts/server_input.gd")
const MfdMap := preload("res://scripts/mfd_map.gd")
const TaskAssignment := preload("res://scripts/task_assignment.gd")
const ParkingDetector := preload("res://scripts/parking_detector.gd")
const TaskManagerScript := preload("res://scripts/task_manager.gd")

var _checks := 0
var _failures := 0


class FakeCarInput:
	extends Node

	var last_slot := -1

	func get_input(slot: int) -> Vector2:
		last_slot = slot
		return Vector2(0.25, 1.0)


func _initialize() -> void:
	_test_protocol()
	_test_input_store()
	_test_car_movement()
	_test_quit_key()
	_test_mfd_map()
	_test_task_assignment()
	_test_parking_detector()
	_test_task_manager()
	_test_main_scene()
	_test_project_settings()

	if _failures == 0:
		print("All %d checks passed." % _checks)
		quit(0)
	else:
		push_error("%d of %d checks failed." % [_failures, _checks])
		quit(1)


func _test_protocol() -> void:
	var parsed := GameProtocol.parse(
		'{"v":1,"type":"input","data":{"slot":1,"seq":1,"steer":0.5,"throttle":1.0}}'
	)
	_check(bool(parsed["ok"]), "valid envelope parses")
	_check_eq(parsed["type"], "input", "envelope type is preserved")
	_check_eq(parsed["data"]["slot"], 1, "envelope data is preserved")

	var malformed := GameProtocol.parse("{not json")
	_check(not bool(malformed["ok"]), "malformed JSON is rejected")
	_check(not bool(malformed["fatal"]), "malformed JSON is not fatal")

	var bare := GameProtocol.parse('{"slot":1,"seq":1,"steer":0.5,"throttle":1.0}')
	_check(not bool(bare["ok"]), "bare input without an envelope is rejected")
	_check(bool(bare["fatal"]), "a missing protocol version is fatal")

	var wrong_version := GameProtocol.parse('{"v":2,"type":"config","data":{}}')
	_check(not bool(wrong_version["ok"]), "unsupported envelope version is rejected")
	_check(bool(wrong_version["fatal"]), "unsupported envelope version is fatal")

	var bad_data := GameProtocol.parse('{"v":1,"type":"config","data":[]}')
	_check(not bool(bad_data["ok"]), "non-object message data is rejected")

	for code in ["auth_failed", "kicked", "protocol_mismatch"]:
		_check(GameProtocol.is_fatal_error(code), "%s is a fatal error" % code)
	_check(
		not GameProtocol.is_fatal_error("game_unavailable"),
		"game_unavailable is not fatal",
	)
	_check(not GameProtocol.is_fatal_error(""), "an empty error code is not fatal")

	_check(
		bool(GameProtocol.validate_game_hello({"protocolVersion": 1})["ok"]),
		"matching game_hello protocol version is accepted",
	)
	var hello_mismatch := GameProtocol.validate_game_hello({"protocolVersion": 2})
	_check(not bool(hello_mismatch["ok"]), "mismatched game_hello version is rejected")
	_check(bool(hello_mismatch["fatal"]), "mismatched game_hello version is fatal")

	var config := GameProtocol.validate_config({
		"roundDurationMs": 180000,
		"tasksPerPlayer": 5,
		"maxPlayers": 4,
	})
	_check(bool(config["ok"]), "valid config is accepted")
	_check_eq(config["maxPlayers"], 4, "config maxPlayers is preserved")

	var zero_round := GameProtocol.validate_config({
		"roundDurationMs": 0,
		"tasksPerPlayer": 5,
		"maxPlayers": 4,
	})
	_check(not bool(zero_round["ok"]), "zero round duration is rejected")
	var too_many_players := GameProtocol.validate_config({
		"roundDurationMs": 180000,
		"tasksPerPlayer": 5,
		"maxPlayers": 5,
	})
	_check(not bool(too_many_players["ok"]), "more than four players is rejected")

	var input := GameProtocol.parse_input({
		"slot": 1,
		"seq": 0,
		"steer": -0.5,
		"throttle": 1.0,
	})
	_check(bool(input["ok"]), "valid input is accepted")
	_check_eq(input["slot"], 1, "input slot is preserved")
	_check_eq(input["seq"], 0, "input sequence is preserved")
	_check_eq(input["steer"], -0.5, "input steer is preserved")

	_check(
		not bool(GameProtocol.parse_input({"seq": 1, "steer": 0.0, "throttle": 0.0})["ok"]),
		"input without a slot is rejected",
	)
	_check(
		not bool(
			GameProtocol.parse_input({"slot": 5, "seq": 1, "steer": 0.0, "throttle": 0.0})["ok"]
		),
		"input outside slots 1-4 is rejected",
	)
	_check(
		not bool(
			GameProtocol.parse_input({"slot": 1, "steer": 0.0, "throttle": 0.0})["ok"]
		),
		"input without a sequence is rejected",
	)
	_check(
		not bool(
			GameProtocol.parse_input({
				"slot": 1,
				"seq": 1,
				"steer": "left",
				"throttle": 0.0,
			})["ok"]
		),
		"non-numeric input is rejected",
	)
	_check(
		not bool(GameProtocol.parse_input({
			"slot": 1,
			"seq": 1,
			"steer": 0.0,
			"throttle": INF,
		})["ok"]),
		"non-finite input is rejected",
	)

	var clamped := GameProtocol.parse_input({
		"slot": 1,
		"seq": 1,
		"steer": 2.0,
		"throttle": -2.0,
	})
	_check_eq(clamped["steer"], 1.0, "steer is clamped")
	_check_eq(clamped["throttle"], -1.0, "throttle is clamped")

	var neutral := GameProtocol.parse_input({"slot": 1, "seq": 1})
	_check_eq(neutral["steer"], 0.0, "missing steer is neutral")
	_check_eq(neutral["throttle"], 0.0, "missing throttle is neutral")


func _test_input_store() -> void:
	var store := SlotInputStore.new()
	_check(store.set_input(1, 0.5, 1.0, 1, 1000), "slot 1 accepts input")
	_check(store.set_input(2, -0.5, -1.0, 1, 1000), "slot 2 accepts input independently")
	_check_eq(store.get_input(1, 1000), Vector2(0.5, 1.0), "slot 1 returns its own input")
	_check_eq(store.get_input(2, 1000), Vector2(-0.5, -1.0), "slot 2 returns its own input")
	_check_eq(store.get_input(3, 1000), Vector2.ZERO, "an unseen slot is neutral")

	_check(
		not store.set_input(1, 0.0, 0.0, 1, 1001),
		"a duplicate sequence is ignored",
	)
	_check(
		not store.set_input(1, 0.0, 0.0, 0, 1001),
		"an older sequence is ignored",
	)
	_check(store.set_input(1, 0.0, 0.0, 2, 1001), "a newer sequence is accepted")
	_check_eq(store.get_last_seq(1), 2, "the last sequence is tracked")

	_check(store.has_recent_input(1, 1500), "input is recent before the timeout")
	_check_eq(store.get_input(1, 1500), Vector2.ZERO, "a neutral sample is still neutral")
	_check(not store.has_recent_input(1, 1501), "input expires at the timeout boundary")
	_check_eq(store.get_input(1, 1501), Vector2.ZERO, "expired input is neutral")

	store.clear_slot(2)
	_check_eq(store.get_input(2, 1000), Vector2.ZERO, "clearing a slot stops only that car")
	_check_eq(store.get_input(1, 1001), Vector2.ZERO, "other slots are unaffected by clear")
	_check(store.set_input(2, 0.25, 0.25, 0, 1002), "a cleared slot accepts a reset sequence")
	_check_eq(store.active_slots(1002), [1, 2], "active slots are reported")

	store.clear_all()
	_check_eq(store.get_input(1, 1002), Vector2.ZERO, "clear_all clears every slot")
	_check_eq(store.get_input(2, 1002), Vector2.ZERO, "clear_all clears every slot")
	_check(not store.set_input(0, 0.0, 0.0, 1, 0), "slot 0 is rejected")
	_check(not store.set_input(5, 0.0, 0.0, 1, 0), "slot 5 is rejected")


func _test_car_movement() -> void:
	var car = CarScript.new()

	car.throttle = 1.0
	car._apply_throttle(1.0)
	_check(is_equal_approx(car._speed, car.max_speed), "full throttle reaches max speed")

	car.throttle = 0.0
	car._apply_throttle(1.0)
	_check(
		is_equal_approx(car._speed, car.max_speed - car.coast_friction),
		"coasting reduces speed",
	)

	car._speed = car.max_speed
	car.throttle = -1.0
	car._apply_throttle(1.0)
	_check(
		is_equal_approx(car._speed, -car.max_reverse_speed),
		"holding brake reaches reverse speed",
	)

	car._speed = 0.0
	car.rotation = 0.0
	car.steer = 1.0
	car._apply_steering(1.0)
	_check(is_zero_approx(car.rotation), "the car does not turn at zero speed")

	car._speed = car.max_speed
	car.rotation = 0.0
	car.steer = 1.0
	car._apply_steering(1.0)
	_check(
		is_equal_approx(car.rotation, deg_to_rad(car.max_turn_speed_deg)),
		"full speed steering uses the maximum turn rate",
	)

	car._speed = -car.max_reverse_speed
	car.rotation = 0.0
	car.steer = 1.0
	car._apply_steering(1.0)
	_check(
		is_equal_approx(car.rotation, -deg_to_rad(car.max_turn_speed_deg * 0.5)),
		"reverse steering inverts direction",
	)

	var source := FakeCarInput.new()
	car.slot = 3
	car.input_source = source
	_check_eq(car._gather_input(), Vector2(0.25, 1.0), "the car reads its source")
	_check_eq(source.last_slot, 3, "the car passes its own slot to the source")

	car.input_source = null
	source.free()
	car.free()


func _test_quit_key() -> void:
	var press := InputEventKey.new()
	press.pressed = true
	press.keycode = KEY_Q
	_check(ServerInputScript.is_quit_event(press), "Q requests quit")

	press.echo = true
	_check(not ServerInputScript.is_quit_event(press), "key repeat does not request quit")

	press.echo = false
	press.pressed = false
	_check(not ServerInputScript.is_quit_event(press), "released Q does not request quit")

	press.pressed = true
	press.keycode = KEY_P
	_check(not ServerInputScript.is_quit_event(press), "P does not request quit")

	_check(
		not ServerInputScript.is_quit_event(InputEventMouseButton.new()),
		"a mouse event does not request quit",
	)


func _test_mfd_map() -> void:
	_check_eq(MfdMap.MAP_VERSION, "proszkowska-mfd-1", "MFD map version is set")
	_check_eq(MfdMap.spot_count(), 4, "MFD map has four spots")
	_check_eq(
		[MfdMap.spot_label(0), MfdMap.spot_label(1), MfdMap.spot_label(2), MfdMap.spot_label(3)],
		["A-01", "A-02", "A-03", "A-04"],
		"spot labels are A-01 to A-04",
	)
	_check_eq(MfdMap.spot_rect(0).get_center(), Vector2(384, 540), "A-01 center matches the baseline")
	_check_eq(MfdMap.spot_rect(3).get_center(), Vector2(1536, 540), "A-04 center matches the baseline")
	_check_eq(MfdMap.spot_rect(1).size, Vector2(96, 144), "parking bays are 96x144")
	_check_eq(MfdMap.spawn_pos(1), Vector2(672, 1000), "S1 spawn matches the baseline")
	_check_eq(MfdMap.spawn_pos(4), Vector2(1248, 1000), "S4 spawn matches the baseline")
	_check_eq(MfdMap.planter_rects().size(), 2, "MFD map has two planters")
	var colors := [
		MfdMap.slot_color(1), MfdMap.slot_color(2), MfdMap.slot_color(3), MfdMap.slot_color(4)
	]
	_check(colors[0] != colors[1] and colors[2] != colors[3], "slot colors are distinct")


func _test_task_assignment() -> void:
	var round1 := TaskAssignment.assign([1, 2, 3, 4], 1, 180000)
	_check_eq(round1.size(), 4, "four active slots receive four tasks")
	var labels: Array = []
	for slot in [1, 2, 3, 4]:
		labels.append(round1[slot]["spotLabel"])
	labels.sort()
	_check_eq(labels, ["A-01", "A-02", "A-03", "A-04"], "targets are exclusive among active players")
	_check_eq(round1[1]["spotIndex"], 0, "round 1 starts at spot 0 for the first slot")
	_check_eq(round1[1]["taskId"], 11, "taskId follows roundId * 10 + (spotIndex + 1)")
	_check_eq(round1[4]["taskId"], 14, "taskId follows the spot order")

	var round2 := TaskAssignment.assign([1, 2, 3, 4], 2, 180000)
	_check_eq(round2[1]["spotIndex"], 1, "round 2 rotates the starting spot")
	_check_eq(round2[4]["spotIndex"], 0, "rotation wraps around the four spots")

	var partial := TaskAssignment.assign([1, 3], 1, 180000)
	_check_eq(partial.size(), 2, "two connected slots receive two tasks")
	_check(partial[1]["spotIndex"] != partial[3]["spotIndex"], "partial slots stay exclusive")

	_check(TaskAssignment.assign([], 1, 180000).is_empty(), "an empty slot list assigns nothing")
	_check_eq(TaskAssignment.assign([0, 5], 1, 180000).size(), 0, "out-of-range slots are not assigned")
	_check_eq(
		TaskAssignment.assign([2, 2, 1], 1, 180000).size(), 2, "duplicate slots are assigned once"
	)
	_check_eq(round1[1]["hint"], "north", "hint matches the baseline")
	_check_eq(round1[1]["deadlineMs"], 180000, "deadline is the remaining round time")


func _test_parking_detector() -> void:
	var rect := Rect2(Vector2(336, 468), Vector2(96, 144))
	var centered := ParkingDetector.car_corners(Vector2(384, 540), deg_to_rad(-90.0), MfdMap.CAR_SIZE)
	_check(ParkingDetector.all_inside(centered, rect), "a centered car is inside the spot")

	var far := ParkingDetector.car_corners(Vector2(900, 540), deg_to_rad(-90.0), MfdMap.CAR_SIZE)
	_check(not ParkingDetector.all_inside(far, rect), "a car in another spot is outside")

	var half_out := ParkingDetector.car_corners(Vector2(384 + 60, 540), deg_to_rad(-90.0), MfdMap.CAR_SIZE)
	_check(not ParkingDetector.all_inside(half_out, rect), "a corner outside fails the check")

	var edge := [
		rect.position,
		rect.position + Vector2(rect.size.x, 0),
		rect.position + rect.size,
		rect.position + Vector2(0, rect.size.y),
	]
	_check(ParkingDetector.all_inside(edge, rect), "the boundary counts as inside")

	_check(ParkingDetector.speed_ok(0.0), "a stopped car passes the speed gate")
	_check(ParkingDetector.speed_ok(40.0), "40 px/s passes the speed gate")
	_check(not ParkingDetector.speed_ok(41.0), "above 40 px/s fails the speed gate")

	_check_eq(ParkingDetector.angle_diff_deg(-90.0, -90.0), 0.0, "aligned facing has zero difference")
	_check_eq(ParkingDetector.angle_diff_deg(90.0, -90.0), 180.0, "opposite facing differs by 180")

	var hold := ParkingDetector.Hold.new()
	_check(not hold.call("update", 0.25, true, true), "half the hold time does not complete")
	_check(hold.call("update", 0.25, true, true), "0.5 s continuous hold completes")
	var reset_hold := ParkingDetector.Hold.new()
	reset_hold.call("update", 0.4, true, true)
	reset_hold.call("update", 0.1, false, true)
	_check(
		not reset_hold.call("update", 0.4, true, true), "leaving the spot resets the hold timer"
	)


func _parked_cars(manager, speed: float) -> Dictionary:
	var cars := {}
	for slot in [1, 2, 3, 4]:
		var entry: Dictionary = manager.get_task(slot)
		if entry.is_empty():
			continue
		var rect: Rect2 = entry["spotRect"]
		cars[slot] = {
			"position": rect.get_center(),
			"rotation": deg_to_rad(float(entry["spotFacing"])),
			"speed": speed,
		}
	return cars


func _test_task_manager() -> void:
	var manager = TaskManagerScript.new()
	manager.start_round([1, 2, 3, 4], 1, 180000, 0)
	manager.set_running(true)
	_check_eq(manager.task_count(), 4, "four connected slots start with four tasks")
	_check(manager.get_task(4)["spotLabel"] != manager.get_task(1)["spotLabel"], "tasks stay exclusive")

	var completed: Array = []
	manager.task_completed.connect(func(slot: int, result: Dictionary) -> void: completed.append([slot, result]))
	manager.register_collision(1)
	manager.register_collision(1)
	manager.update_tasks(0.6, _parked_cars(manager, 0.0), 12000)
	_check_eq(completed.size(), 4, "parked cars complete after the hold time")
	var first: Dictionary = completed[0][1]
	_check_eq(first["collisions"], 2 if int(first["slot"]) == 1 else 0, "collisions are recorded per slot")
	manager.update_tasks(0.6, _parked_cars(manager, 0.0), 13000)
	_check_eq(completed.size(), 4, "a completed task never completes twice")
	manager.register_collision(int(first["slot"]))
	_check_eq(
		manager.get_task(int(first["slot"]))["collisions"], int(first["collisions"]),
		"collisions after completion are ignored"
	)

	var early = TaskManagerScript.new()
	early.start_round([1], 3, 180000, 0)
	early.set_running(true)
	var early_done: Array = []
	early.task_completed.connect(func(slot: int, result: Dictionary) -> void: early_done.append(slot))
	early.update_tasks(0.3, _parked_cars(early, 0.0), 300)
	early.update_tasks(0.3, _parked_cars(early, 100.0), 600)
	_check(early_done.is_empty(), "leaving the spot or speeding up prevents completion")

	var slow = TaskManagerScript.new()
	slow.start_round([2], 1, 180000, 0)
	slow.set_running(true)
	var slow_done: Array = []
	slow.task_completed.connect(func(slot: int, result: Dictionary) -> void: slow_done.append(slot))
	slow.update_tasks(0.6, _parked_cars(slow, 100.0), 600)
	_check(slow_done.is_empty(), "a fast roll through the spot does not complete")

	var held = TaskManagerScript.new()
	held.start_round([1, 2], 1, 180000, 0)
	held.set_running(true)
	held.set_slot_connected(2, false)
	var held_done: Array = []
	held.task_completed.connect(func(slot: int, result: Dictionary) -> void: held_done.append(slot))
	held.update_tasks(0.6, _parked_cars(held, 0.0), 600)
	_check_eq(held_done, [1], "a disconnected car keeps its task but does not complete")
	_check(not held.get_task(2).is_empty(), "a disconnected slot keeps its reserved task")
	held.set_slot_connected(2, true)
	held.update_tasks(0.6, _parked_cars(held, 0.0), 1200)
	_check_eq(held_done, [1, 2], "reconnecting resumes the same task")

	var kicked = TaskManagerScript.new()
	kicked.start_round([1, 2], 1, 180000, 0)
	kicked.release_slot(2)
	_check(kicked.get_task(2).is_empty(), "a kicked slot releases its spot")
	_check_eq(kicked.task_count(), 1, "kicking removes only that slot")

	var paused = TaskManagerScript.new()
	paused.start_round([1], 1, 180000, 0)
	paused.set_running(false)
	var paused_done: Array = []
	paused.task_completed.connect(func(slot: int, result: Dictionary) -> void: paused_done.append(slot))
	paused.update_tasks(0.6, _parked_cars(paused, 0.0), 600)
	_check(paused_done.is_empty(), "no completion is detected while paused")

	for node in [manager, early, slow, held, kicked, paused]:
		(node as Node).free()


func _test_main_scene() -> void:
	var packed := load("res://scenes/main.tscn") as PackedScene
	_check(packed != null, "main.tscn loads")
	if packed == null:
		return

	var scene := packed.instantiate()
	var cars := scene.find_children("Car*", "CharacterBody2D", true, false)
	_check_eq(cars.size(), 4, "the main scene has four cars")

	var server_input := scene.find_child("ServerInput", true, false)
	_check(server_input != null, "the main scene has one server input node")

	var slots: Array[int] = []
	for car in cars:
		slots.append(int(car.slot))
		_check_eq(car.collision_layer, 2, "cars are on the car collision layer")
		_check_eq(car.collision_mask, 1, "cars collide with the environment only")
		_check_eq(
			car.motion_mode,
			CharacterBody2D.MOTION_MODE_FLOATING,
			"cars use floating motion",
		)
		_check(car.get_node_or_null("Sprite2D") != null, "each car has a sprite")
		_check(
			car.get_node_or_null("CollisionShape2D") != null,
			"each car has a collision shape",
		)
		_check_eq(
			car.input_source,
			server_input,
			"each car reads input from the shared server input node",
		)
	slots.sort()
	_check_eq(slots, [1, 2, 3, 4], "cars cover slots 1-4 exactly once")
	scene.free()


func _test_project_settings() -> void:
	_check_eq(
		str(ProjectSettings.get_setting("application/run/main_scene")),
		"res://scenes/main.tscn",
		"main scene is configured",
	)
	_check_eq(
		int(ProjectSettings.get_setting("display/window/size/viewport_width")),
		1920,
		"viewport width is 1920",
	)
	_check_eq(
		int(ProjectSettings.get_setting("display/window/size/viewport_height")),
		1080,
		"viewport height is 1080",
	)
	_check_eq(
		int(ProjectSettings.get_setting("display/window/size/mode")),
		3,
		"the game starts fullscreen",
	)


func _check(condition: bool, message: String) -> void:
	_checks += 1
	if not condition:
		_failures += 1
		push_error("FAIL: %s" % message)


func _check_eq(actual: Variant, expected: Variant, message: String) -> void:
	_check(
		actual == expected,
		"%s (expected %s, got %s)" % [message, str(expected), str(actual)],
	)
