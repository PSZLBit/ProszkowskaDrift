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
const ParkingLotArt := preload("res://scripts/parking_lot_art.gd")
const ParkingLotMapScript := preload("res://scripts/parking_lot_map.gd")
const RoundClock := preload("res://scripts/round_clock.gd")
const ScoreModel := preload("res://scripts/score_model.gd")
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
	_test_parking_lot_tileset()
	_test_parking_lot_map()
	_test_parking_lot_art_swap()
	_test_round_clock()
	_test_score_model()
	_test_hud_scene()
	_test_scriptless_hud_tolerance()
	_test_debug_drive()
	_test_main_round_lifecycle()
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
		"tasksPerPlayer": 1,
		"maxPlayers": 4,
	})
	_check(bool(config["ok"]), "valid config is accepted")
	_check_eq(config["maxPlayers"], 4, "config maxPlayers is preserved")

	var zero_round := GameProtocol.validate_config({
		"roundDurationMs": 0,
		"tasksPerPlayer": 1,
		"maxPlayers": 4,
	})
	_check(not bool(zero_round["ok"]), "zero round duration is rejected")
	var too_many_players := GameProtocol.validate_config({
		"roundDurationMs": 180000,
		"tasksPerPlayer": 1,
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
	manager.set_running(true, 0)
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
	_check_eq(first["completionMs"], 12000, "completion time uses the supplied timestamps")
	manager.update_tasks(0.6, _parked_cars(manager, 0.0), 13000)
	_check_eq(completed.size(), 4, "a completed task never completes twice")
	manager.register_collision(int(first["slot"]))
	_check_eq(
		manager.get_task(int(first["slot"]))["collisions"], int(first["collisions"]),
		"collisions after completion are ignored"
	)

	var early = TaskManagerScript.new()
	early.start_round([1], 3, 180000, 0)
	early.set_running(true, 0)
	var early_done: Array = []
	early.task_completed.connect(func(slot: int, result: Dictionary) -> void: early_done.append(slot))
	early.update_tasks(0.3, _parked_cars(early, 0.0), 300)
	early.update_tasks(0.3, _parked_cars(early, 100.0), 600)
	_check(early_done.is_empty(), "leaving the spot or speeding up prevents completion")

	var slow = TaskManagerScript.new()
	slow.start_round([2], 1, 180000, 0)
	slow.set_running(true, 0)
	var slow_done: Array = []
	slow.task_completed.connect(func(slot: int, result: Dictionary) -> void: slow_done.append(slot))
	slow.update_tasks(0.6, _parked_cars(slow, 100.0), 600)
	_check(slow_done.is_empty(), "a fast roll through the spot does not complete")

	var held = TaskManagerScript.new()
	held.start_round([1, 2], 1, 180000, 0)
	held.set_running(true, 0)
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
	paused.set_running(false, 0)
	var paused_done: Array = []
	paused.task_completed.connect(func(slot: int, result: Dictionary) -> void: paused_done.append(slot))
	paused.update_tasks(0.6, _parked_cars(paused, 0.0), 600)
	_check(paused_done.is_empty(), "no completion is detected while paused")

	var resumed = TaskManagerScript.new()
	resumed.start_round([1], 1, 180000, 1000)
	resumed.set_running(true, 1000)
	var resumed_done: Array = []
	var resumed_results: Array = []
	resumed.task_completed.connect(
		func(slot: int, result: Dictionary) -> void:
			resumed_done.append(slot)
			resumed_results.append(result)
	)
	resumed.update_tasks(0.3, _parked_cars(resumed, 0.0), 1300)
	resumed.set_running(false, 1300)
	resumed.set_running(true, 5300)
	resumed.update_tasks(0.5, _parked_cars(resumed, 0.0), 5800)
	_check_eq(resumed_done, [1], "the task completes after resume with a fresh hold")
	var resumed_first: Dictionary = resumed_results[0]
	_check_eq(
		resumed_first["completionMs"],
		800,
		"paused time is excluded from the completion duration",
	)

	var hits = TaskManagerScript.new()
	hits.start_round([1], 1, 180000, 0)
	var hit_events: Array = []
	hits.collisions_changed.connect(
		func(slot: int, count: int) -> void: hit_events.append([slot, count])
	)
	hits.register_collision(1)
	hits.register_collision(1)
	_check_eq(hit_events, [[1, 1], [1, 2]], "collisions emit live counts")
	_check_eq(int(hits.get_task(1)["collisions"]), 2, "collisions accumulate per slot")

	for node in [manager, early, slow, held, kicked, paused, resumed, hits]:
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
	var hud := scene.find_child("Hud", true, false)
	_check(hud != null, "the main scene has a HUD overlay")

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

	var environment := scene.find_child("Environment", true, false)
	_check(environment != null, "the main scene has an environment body")
	if environment != null:
		var solids := (environment as Node).find_children("*", "CollisionShape2D", true, false)
		_check(solids.size() >= 6, "the environment has walls plus two MFD planters")
		var positions: Array = []
		for solid in solids:
			positions.append((solid as Node2D).position)
		_check(positions.has(Vector2(240, 540)), "the west planter matches the MFD baseline")
		_check(positions.has(Vector2(1680, 540)), "the east planter matches the MFD baseline")

	var task_manager := scene.find_child("TaskManager", true, false)
	_check(task_manager != null, "the main scene has a task manager node")
	scene.free()


func _test_parking_lot_tileset() -> void:
	var builder = load("res://scripts/parking_lot_tileset_builder.gd")
	if builder != null and not ResourceLoader.exists("res://tilesets/parking_lot.tres"):
		builder.build(
			"res://assets/placeholders/large_parking_lot/",
			builder.placeholder_files(),
			"res://tilesets/parking_lot.tres",
		)
	var tile_set := load("res://tilesets/parking_lot.tres") as TileSet
	_check(tile_set != null, "parking_lot.tres loads")
	if tile_set == null:
		return

	_check_eq(tile_set.tile_size, Vector2i(48, 48), "parking tiles are 48x48")
	_check_eq(tile_set.get_source_count(), 8, "the tileset has eight atlas sources")
	_check_eq(tile_set.get_physics_layers_count(), 1, "the tileset has one physics layer")
	_check_eq(
		tile_set.get_physics_layer_collision_layer(0),
		1,
		"parking tiles target the environment collision layer",
	)
	_check_eq(
		tile_set.get_physics_layer_collision_mask(0),
		2,
		"parking tiles detect the car collision layer",
	)

	var expected := ["A2-1", "A2-2", "A4-walls", "B-01", "B-02", "B-03", "B-04", "B-05"]
	var names: Array[String] = []
	for index in tile_set.get_source_count():
		var source := tile_set.get_source(tile_set.get_source_id(index)) as TileSetAtlasSource
		_check(source != null, "source %d is an atlas source" % index)
		if source == null:
			continue
		names.append(source.resource_name)
		_check(source.texture != null, "source %s has a texture" % source.resource_name)
		_check_eq(
			source.texture_region_size,
			Vector2i(48, 48),
			"source %s uses 48x48 regions" % source.resource_name,
		)
		if source.texture != null:
			_check_eq(
				source.texture.get_width() % 48,
				0,
				"source %s width is a 48px multiple" % source.resource_name,
			)
			_check_eq(
				source.texture.get_height() % 48,
				0,
				"source %s height is a 48px multiple" % source.resource_name,
			)
	_check_eq(names, expected, "atlas sources are named in generation order")

	var a2 := _find_atlas_source(tile_set, "A2-2")
	_check(a2 != null, "the A2-2 ground source exists")
	if a2 != null and a2.texture != null:
		_check(
			a2.texture.resource_path.ends_with(
				"assets/placeholders/large_parking_lot/a2_2.png"
			),
			"the committed tileset uses the placeholder A2-2 sheet",
		)
		_check(a2.has_tile(Vector2i(1, 3)), "the plain ground tile exists")
		_check(a2.has_tile(Vector2i(6, 3)), "the kerb tile exists")

	var b01 := _find_atlas_source(tile_set, "B-01")
	_check(b01 != null, "the B-01 parking source exists")
	if b01 != null:
		_check(b01.has_tile(Vector2i(0, 2)), "the first bay top tile exists")
		_check(b01.has_tile(Vector2i(3, 3)), "the last bay bottom tile exists")


func _test_parking_lot_map() -> void:
	var packed := load("res://scenes/map/parking_lot.tscn") as PackedScene
	_check(packed != null, "parking_lot.tscn loads")
	if packed == null:
		return

	var map = packed.instantiate()
	var tile_map: TileMapLayer = map.get_tile_map_layer()
	_check(tile_map != null, "the parking map has a TileMapLayer")
	if tile_map == null:
		map.free()
		return

	_check_eq(tile_map.position, ParkingLotMapScript.MAP_OFFSET, "the map is centred vertically")
	_check_eq(
		ParkingLotMapScript.MAP_SIZE * 48,
		Vector2i(1920, 1056),
		"the 40x22 map fills the 1920px width",
	)

	var environment := map.get_node_or_null("Environment") as StaticBody2D
	_check(environment != null, "the demo map has an environment body")
	if environment != null:
		_check_eq(environment.collision_layer, 1, "the map boundary uses the environment layer")
		_check_eq(environment.collision_mask, 2, "the map boundary detects cars")
		var boundary_shapes := environment.get_children()
		var shape_count := 0
		for child in boundary_shapes:
			if child is CollisionShape2D and child.shape != null:
				shape_count += 1
		_check_eq(shape_count, 4, "the map boundary has four collision shapes")

	map.build()
	_check(tile_map.tile_set != null, "the parking map has a TileSet")
	if tile_map.tile_set == null:
		map.free()
		return
	_check_eq(
		tile_map.get_cell_source_id(Vector2i(0, 0)),
		ParkingLotMapScript.SOURCE_A2_2,
		"the border uses the A2-2 source",
	)
	_check_eq(
		tile_map.get_cell_atlas_coords(Vector2i(0, 0)),
		ParkingLotMapScript.KERB_COORDS,
		"the border uses the kerb tile",
	)
	_check_eq(
		tile_map.get_cell_atlas_coords(Vector2i(1, 1)),
		ParkingLotMapScript.GROUND_COORDS,
		"the interior uses the plain ground tile",
	)

	for bay in ParkingLotMapScript.BAY_COUNT:
		var column: int = ParkingLotMapScript.BAY_FIRST_COLUMN + bay
		var top_row: int = ParkingLotMapScript.BAY_TOP_ROW
		_check_eq(
			tile_map.get_cell_source_id(Vector2i(column, top_row)),
			ParkingLotMapScript.SOURCE_B_01,
			"bay %d uses the B-01 source" % (bay + 1),
		)
		_check_eq(
			tile_map.get_cell_atlas_coords(Vector2i(column, top_row)),
			ParkingLotMapScript.BAY_TOP_COORDS + Vector2i(bay, 0),
			"bay %d top marking is correct" % (bay + 1),
		)
		_check_eq(
			tile_map.get_cell_atlas_coords(Vector2i(column, top_row + 1)),
			ParkingLotMapScript.BAY_BOTTOM_COORDS + Vector2i(bay, 0),
			"bay %d bottom marking is correct" % (bay + 1),
		)

	var spawns := map.get_node_or_null("Spawns") as Node2D
	_check(spawns != null, "the parking map has spawn markers")
	if spawns != null:
		_check_eq(spawns.get_child_count(), 4, "there are four spawn markers")

	var bays := map.get_node_or_null("Bays") as Node2D
	_check(bays != null, "the parking map has bay markers")
	if bays != null:
		_check_eq(
			bays.get_child_count(),
			ParkingLotMapScript.BAY_COUNT,
			"every bay has a marker",
		)
		for bay in ParkingLotMapScript.BAY_COUNT:
			var marker := bays.get_node_or_null("Bay%d" % (bay + 1)) as Marker2D
			_check(marker != null, "bay %d has a marker" % (bay + 1))
			if marker != null:
				var center := Vector2(
					(ParkingLotMapScript.BAY_FIRST_COLUMN + bay + 0.5) * 48.0,
					(ParkingLotMapScript.BAY_TOP_ROW + 1) * 48.0,
				)
				_check_eq(marker.position, center, "bay %d marker sits in its bay" % (bay + 1))

	_check(
		ParkingLotMapScript.should_use_local(true, true),
		"an installed local TileSet is preferred",
	)
	_check(
		not ParkingLotMapScript.should_use_local(true, false),
		"a local TileSet without purchased sheets is stale",
	)
	_check(
		not ParkingLotMapScript.should_use_local(false, true),
		"a missing local TileSet falls back to committed",
	)
	_check(
		not ParkingLotMapScript.should_use_local(false, false),
		"a missing local TileSet without purchased sheets falls back",
	)

	map.free()


func _test_parking_lot_art_swap() -> void:
	_check_eq(
		ParkingLotArt.REAL_FILES.size(),
		8,
		"every atlas source has a purchased file mapping",
	)
	_check_eq(
		ParkingLotArt.REAL_FILES.get("A2-1", ""),
		"Tile_A2-1 (2).png",
		"the A2-1 mapping matches the purchased file",
	)
	_check_eq(
		ParkingLotArt.REAL_FILES.get("B-05", ""),
		"tile-B-05.png",
		"the B-05 mapping matches the purchased file",
	)
	_check_eq(ParkingLotArt.apply(null), 0, "a missing TileSet is ignored")

	var tile_set := load("res://tilesets/parking_lot.tres") as TileSet
	_check(tile_set != null, "the tileset loads for the art-swap test")
	if tile_set == null:
		return

	var before: Array = []
	for index in tile_set.get_source_count():
		var source := tile_set.get_source(tile_set.get_source_id(index)) as TileSetAtlasSource
		before.append(source.texture if source != null else null)

	var swapped := ParkingLotArt.apply(
		tile_set,
		"res://assets/__missing_parking_lot__/",
	)
	_check_eq(
		swapped,
		0,
		"missing purchased sheets leave the placeholder textures in place",
	)

	for index in tile_set.get_source_count():
		var source := tile_set.get_source(tile_set.get_source_id(index)) as TileSetAtlasSource
		if source != null:
			_check_eq(
				source.texture,
				before[index],
				"source %d keeps its placeholder texture" % index,
			)


func _find_atlas_source(tile_set: TileSet, source_name: String) -> TileSetAtlasSource:
	for index in tile_set.get_source_count():
		var source := tile_set.get_source(tile_set.get_source_id(index)) as TileSetAtlasSource
		if source != null and source.resource_name == source_name:
			return source
	return null


func _test_round_clock() -> void:
	var clock := RoundClock.new()
	_check_eq(clock.phase, RoundClock.Phase.READY, "the clock starts ready")
	_check_eq(clock.remaining_ms, 180000, "the clock starts full")

	clock.start()
	_check_eq(clock.phase, RoundClock.Phase.PLAYING, "start enters playing")
	_check_eq(clock.remaining_ms, 180000, "start resets the timer")

	clock.tick(1.0)
	_check_eq(clock.remaining_ms, 179000, "one second elapses")

	clock.tick(-1.0)
	_check_eq(clock.remaining_ms, 179000, "negative ticks are ignored")
	clock.tick(INF)
	_check_eq(clock.remaining_ms, 179000, "non-finite ticks are ignored")

	clock.pause()
	_check_eq(clock.phase, RoundClock.Phase.PAUSED, "pause enters paused")
	clock.tick(10.0)
	_check_eq(clock.remaining_ms, 179000, "paused time does not elapse")

	clock.resume()
	_check_eq(clock.phase, RoundClock.Phase.PLAYING, "resume enters playing")

	clock.reset()
	_check_eq(clock.phase, RoundClock.Phase.READY, "reset returns to ready")
	_check_eq(clock.remaining_ms, 180000, "reset restores the full time")

	var emitted: Array = []
	clock.time_changed.connect(func(remaining_ms: int) -> void: emitted.append(remaining_ms))
	clock.start()
	_check_eq(emitted, [180000], "start emits the reset time")
	clock.reset()
	_check_eq(emitted, [180000, 180000], "reset emits the restored time")

	clock.configure(60000)
	_check_eq(clock.round_length_ms, 60000, "configure applies the server duration")
	_check_eq(clock.remaining_ms, 60000, "configure resets the idle timer")
	_check_eq(emitted, [180000, 180000, 60000], "configure emits the new time")
	clock.start()
	clock.tick(10.0)
	clock.configure(120000)
	_check_eq(clock.remaining_ms, 50000, "configure while playing keeps the running timer")

	var finished := [false]
	clock.finished.connect(func() -> void: finished[0] = true)
	clock.start()
	clock.tick(180.0)
	_check_eq(clock.remaining_ms, 0, "the timer clamps at zero")
	_check_eq(clock.phase, RoundClock.Phase.FINISHED, "expiry enters finished")
	_check(finished[0], "expiry emits finished")

	clock.show_results()
	_check_eq(clock.phase, RoundClock.Phase.RESULTS, "finished shows results")

	var early_clock := RoundClock.new()
	early_clock.start()
	early_clock.show_results()
	_check_eq(early_clock.phase, RoundClock.Phase.RESULTS, "an early finish shows results")
	early_clock.free()

	clock.reset()
	clock.round_length_ms = 5000
	clock.start()
	_check_eq(clock.remaining_ms, 5000, "a custom length is used")
	clock.tick(10.0)
	_check_eq(clock.remaining_ms, 0, "oversized ticks clamp at zero")

	clock.free()


func _test_score_model() -> void:
	_check_eq(ScoreModel.compute_time_score(0), 100, "zero elapsed time scores 100")
	_check_eq(ScoreModel.compute_time_score(12400), 63, "12.4 s scores 63")
	_check_eq(ScoreModel.compute_time_score(18900), 44, "18.9 s scores 44")
	_check_eq(ScoreModel.compute_time_score(30000), 10, "30 s reaches the time floor")
	_check_eq(ScoreModel.compute_time_score(60000), 10, "slow parks keep the time floor")
	_check_eq(ScoreModel.compute_center_score(0.0), 20, "a centered park scores 20")
	_check_eq(ScoreModel.compute_center_score(48.0), 0, "half-bay offset scores 0")
	_check_eq(ScoreModel.compute_angle_score(0.0), 20, "aligned facing scores 20")
	_check_eq(ScoreModel.compute_angle_score(10.0), 20, "10 degrees keeps full angle score")
	_check_eq(ScoreModel.compute_angle_score(35.0), 0, "35 degrees scores 0")
	_check_eq(ScoreModel.compute_penalty(0), 0, "a clean park has no penalty")
	_check_eq(ScoreModel.compute_penalty(1), 10, "one hit costs 10")
	_check_eq(ScoreModel.compute_penalty(9), 30, "the penalty caps at 30")
	_check_eq(ScoreModel.rating_for(40), "perfect", "35-40 rates perfect")
	_check_eq(ScoreModel.rating_for(25), "great", "25-34 rates great")
	_check_eq(ScoreModel.rating_for(15), "good", "15-24 rates good")
	_check_eq(ScoreModel.rating_for(0), "parked", "0-14 rates parked")

	var scores := ScoreModel.new()
	_check(not scores.register(0), "slot 0 is rejected by the score model")
	_check(not scores.register(5), "slot 5 is rejected by the score model")
	_check(scores.register(1), "slot 1 registers")
	_check(scores.register(2), "slot 2 registers")
	_check(scores.register(3), "slot 3 registers")
	_check(scores.has_player(1), "player 1 is registered")
	_check_eq(scores.total(1), 0, "an unfinished player scores zero")
	_check_eq(scores.tasks_completed(1), 0, "an unfinished player has no tasks")
	_check_eq(scores.total(99), 0, "unknown players score zero")
	_check(not scores.record_completion(99, 1, 1000, 0.0, 0.0, 0), "unknown slots cannot complete")

	# Worked examples from the MFD baseline: 103 = 63 time + 40 precision at
	# 12.4 s, and 74 = 44 time + 40 precision - 10 for one hit at 18.9 s.
	_check(scores.record_completion(2, 12, 12400, 0.0, 0.0, 0), "a clean centered park records")
	_check(scores.record_completion(1, 11, 18900, 0.0, 0.0, 1), "a slow park with one hit records")
	_check_eq(scores.total(2), 103, "worked example: clean park scores 103")
	_check_eq(scores.total(1), 74, "worked example: slow park with one hit scores 74")
	_check(not scores.record_completion(2, 12, 13000, 0.0, 0.0, 0), "a completed task records exactly once")
	_check_eq(scores.total(2), 103, "a repeated completion changes nothing")

	_check_eq(scores.ranking(), [2, 1, 3], "ranking orders completed first, then score")
	_check_eq(scores.winner_slot(), 2, "the winner is the first ranked completing player")

	var empty := ScoreModel.new()
	empty.register(1)
	empty.register(2)
	_check_eq(empty.ranking(), [1, 2], "unfinished players rank by slot")
	_check_eq(empty.winner_slot(), 0, "no completing player means no winner")

	var task_payload: Dictionary = scores.task_result_payload(2)
	_check_eq(task_payload["taskId"], 12, "task payload carries the task id")
	_check_eq(task_payload["points"], 103, "task payload carries points")
	_check_eq(task_payload["timeScore"], 63, "task payload carries the time score")
	_check_eq(task_payload["precisionScore"], 40, "task payload carries precision")
	_check_eq(task_payload["collisionPenalty"], 0, "task payload carries the penalty")
	_check_eq(task_payload["rating"], "perfect", "task payload carries the rating")
	_check_eq(task_payload["totalScore"], 103, "task payload carries the total")
	_check(scores.task_result_payload(3).is_empty(), "unfinished slots have no task payload")

	var round_payload: Dictionary = scores.round_result_payload(7)
	_check_eq(round_payload["roundId"], 7, "round payload carries the round id")
	_check_eq(round_payload["winnerSlot"], 2, "round payload carries the winner")
	_check_eq(
		round_payload["scores"],
		[
			{"slot": 2, "score": 103, "tasks": 1, "collisions": 0},
			{"slot": 1, "score": 74, "tasks": 1, "collisions": 1},
			{"slot": 3, "score": 0, "tasks": 0, "collisions": 0},
		],
		"round payload lists every registered slot in rank order",
	)

	var ties := ScoreModel.new()
	ties.register(1)
	ties.register(2)
	ties.record_completion(1, 11, 5000, 0.0, 0.0, 1)
	ties.record_completion(2, 12, 5000, 0.0, 0.0, 0)
	_check_eq(ties.ranking(), [2, 1], "equal scores rank fewer collisions first")

	var timing := ScoreModel.new()
	timing.register(1)
	timing.register(2)
	timing.record_completion(1, 11, 9000, 0.0, 0.0, 0)
	timing.record_completion(2, 12, 9000, 24.0, 20.0, 1)
	# Slot 1: time 73 + precision 40 = 113. Slot 2: time 73 + center 10 +
	# angle 12 - 10 = 85. Faster completion must not outrank a higher score.
	_check_eq(timing.total(1), 113, "centered fast park scores 113")
	_check_eq(timing.total(2), 85, "off-center park with a hit scores 85")
	_check_eq(timing.ranking(), [1, 2], "higher score outranks faster completion")

	var incomplete_last := ScoreModel.new()
	incomplete_last.register(1)
	incomplete_last.register(2)
	incomplete_last.register(3)
	incomplete_last.record_completion(3, 13, 29000, 47.0, 34.0, 9)
	# Slot 3: time max(10, 100-87)=13, center 20-floor(19.58)=1, angle
	# floor(20*1/25)=0, precision 1, penalty 30 -> max(0, 13+1-30)=0.
	# Completed with 0 points still outranks unfinished players.
	_check_eq(incomplete_last.total(3), 0, "a sloppy park can score zero")
	_check_eq(incomplete_last.tasks_completed(3), 1, "a zero-point park still completed")
	_check_eq(incomplete_last.ranking()[0], 3, "a completed task outranks unfinished players")

	var invalid := ScoreModel.new()
	invalid.register(1)
	_check(not invalid.record_completion(1, 11, -5, 0.0, 0.0, 0), "negative time is rejected")
	_check(not invalid.record_completion(1, 11, 1000, INF, 0.0, 0), "infinite offset is rejected")
	_check(not invalid.record_completion(1, 11, 1000, 0.0, NAN, 0), "NaN angle is rejected")
	_check(not invalid.record_completion(1, 11, 1000, 0.0, 0.0, -1), "negative collisions are rejected")
	_check_eq(invalid.total(1), 0, "rejected completions score nothing")

	var kicked := ScoreModel.new()
	kicked.register(1)
	kicked.register(2)
	kicked.record_completion(1, 11, 5000, 0.0, 0.0, 0)
	kicked.unregister(2)
	_check(not kicked.has_player(2), "a kicked slot leaves the model")
	_check_eq(kicked.ranking(), [1], "a kicked slot is excluded from results")

	var live := ScoreModel.new()
	live.register(1)
	live.register(2)
	live.register(3)
	_check(live.set_collisions(1, 2), "live hits are recorded for an unfinished slot")
	_check(live.set_collisions(2, 0), "zero hits are recorded")
	_check_eq(live.collisions(1), 2, "live hits are readable")
	_check(not live.set_collisions(9, 1), "live hits for an unknown slot are rejected")
	_check(not live.set_collisions(1, -1), "negative live hits are rejected")
	# Both unfinished: fewer hits must rank first, then the lower slot.
	live.set_collisions(3, 1)
	_check_eq(live.ranking(), [2, 3, 1], "unfinished players rank fewer hits first")
	_check(live.record_completion(1, 11, 5000, 0.0, 0.0, 2), "completion keeps its final hit count")
	_check(not live.set_collisions(1, 5), "live hits never overwrite a completed task")
	_check_eq(live.collisions(1), 2, "completed hits stay frozen")


func _test_hud_scene() -> void:
	var packed := load("res://scenes/ui/hud.tscn") as PackedScene
	_check(packed != null, "hud.tscn loads")
	if packed == null:
		return

	var hud = packed.instantiate()
	root.add_child(hud)
	_check(hud.get_node_or_null("TimerLabel") != null, "the HUD has a timer label")
	_check(hud.get_node_or_null("Scores") != null, "the HUD has a scores container")
	_check(hud.get_node_or_null("StateLabel") != null, "the HUD has a state line")
	_check(
		hud.get_node_or_null("ResultsPanel") != null,
		"the HUD has a results panel",
	)
	_check(
		hud.get_node_or_null("ResultsPanel/Box/TitleLabel") != null,
		"the results panel has a title",
	)
	_check(
		hud.get_node_or_null("ResultsPanel/Box/WinnerLabel") != null,
		"the results panel has a winner line",
	)
	_check(
		hud.get_node_or_null("ResultsPanel/Box/Rows") != null,
		"the results panel has a rows container",
	)
	_check(
		hud.get_node_or_null("ResultsPanel/Box/PromptLabel") != null,
		"the results panel has a next-round prompt",
	)
	var box := hud.get_node_or_null("ResultsPanel/Box") as VBoxContainer
	_check(box != null, "results content sits in a layout container")
	if box != null:
		var order: Array = []
		for child in box.get_children():
			order.append(str(child.name))
		_check_eq(
			order,
			["TitleLabel", "WinnerLabel", "Rows", "PromptLabel"],
			"results stack title, winner, rows, prompt without overlap",
		)

	var clock := RoundClock.new()
	hud.bind(clock)
	hud.set_state("Drive to your spot")
	_check_eq(hud.state_label.text, "Drive to your spot", "state copy is shown")
	hud.update_board([
		{"slot": 1, "score": 120, "status": "Parked", "hits": 1},
		{"slot": 2, "score": 0, "status": "A-02", "hits": 0},
		{"slot": 3, "score": 0, "status": "Reconnecting", "hits": 2},
	])
	_check_eq(hud.scores_box.get_child_count(), 3, "the board lists active slots only")
	var first_row := hud.scores_box.get_child(0) as HBoxContainer
	var first_text := (first_row.get_child(1) as Label).text
	_check(first_text.contains("P1"), "board rows name the slot")
	_check(first_text.contains("120"), "board rows show the score")
	_check(first_text.contains("Parked"), "board rows show the status")
	_check(first_text.contains("1 hit"), "board rows use the singular hit word")
	var third_row := hud.scores_box.get_child(2) as HBoxContainer
	_check(
		(third_row.get_child(1) as Label).text.contains("2 hits"),
		"board rows use the plural hit word",
	)
	hud.update_board([
		{"slot": 1, "score": 5, "status": "Parked", "hits": 0},
		"bad-row",
		{"slot": 9, "score": 5, "status": "Parked", "hits": 0},
		{"score": 5},
	])
	_check_eq(hud.scores_box.get_child_count(), 1, "malformed board rows are skipped")

	hud.flash_notice("P1 parked!")
	_check_eq(hud.state_label.text, "P1 parked!", "completion flashes a notice")
	hud._process(1.0)
	_check_eq(hud.state_label.text, "P1 parked!", "the notice survives one second")
	hud._process(1.5)
	_check_eq(
		hud.state_label.text, "Drive to your spot", "the notice restores the base copy"
	)

	hud._on_time_changed(90000)
	_check_eq(hud.timer_label.text, "01:30", "timer formats as MM:SS")
	hud._on_time_changed(-1000)
	_check_eq(hud.timer_label.text, "00:00", "negative time formats as zero")

	_check(
		not hud.get_node("ResultsPanel").visible,
		"results start hidden",
	)
	hud.show_results("Round over", "Winner: P2", ["1. P2   103   Parked   0 hits   12.4 s"])
	var panel_labels := hud.get_node("ResultsPanel/Box/Rows").get_child_count()
	_check_eq(panel_labels, 1, "results list one row per player")
	_check_eq(hud.title_label.text, "Round over", "results show the title")
	_check_eq(hud.winner_label.text, "Winner: P2", "results show the winner")
	_check(
		hud.get_node("ResultsPanel").visible,
		"show_results reveals the panel",
	)
	hud.hide_results()
	_check(
		not hud.get_node("ResultsPanel").visible,
		"hide_results hides the panel for the next round",
	)

	hud.free()
	clock.free()


func _test_scriptless_hud_tolerance() -> void:
	# A scratch scene or stale import may leave a plain CanvasLayer named Hud
	# with none of the display methods. The round must run without errors.
	OS.set_environment("PD_GAME_TOKEN", "test-scriptless-hud")
	var parent := Node.new()
	var bare := CanvasLayer.new()
	bare.name = "Hud"
	parent.add_child(bare)
	var si = ServerInputScript.new()
	parent.add_child(si)
	root.add_child(parent)
	# _ready() is deferred for nodes added during SceneTree._initialize(), so
	# bind the HUD explicitly. The later engine _ready re-run is idempotent.
	si._setup_round_ui()
	_check(si._hud != null, "a scriptless Hud node is still found")
	_check(not si._hud.has_method("bind"), "a plain CanvasLayer has no bind")
	si._hud_call("bind", [si._clock])
	si._hud_call("set_state", ["Drive to your spot"])
	si._hud_call("update_board", [[]])
	si._hud_call("show_results", ["Round over", "Winner: none", []])
	si._hud_call("hide_results", [])
	_check(true, "display calls with a scriptless Hud are skipped")
	parent.free()


func _test_debug_drive() -> void:
	# F12 starts a serverless local round for slot 1 driven by the arrow keys,
	# so scoring can be tested with no server or controller.
	OS.set_environment("PD_GAME_TOKEN", "test-debug-drive")
	var parent := Node.new()
	var tm = TaskManagerScript.new()
	tm.name = "TaskManager"
	parent.add_child(tm)
	var si = ServerInputScript.new()
	parent.add_child(si)
	root.add_child(parent)
	_check(not si._connected, "the debug game starts disconnected")

	var f12 := InputEventKey.new()
	f12.pressed = true
	f12.keycode = KEY_F12
	si._unhandled_input(f12)
	_check(si._debug_drive, "F12 enables debug drive")
	_check_eq(si._phase, "playing", "debug drive starts a local round")
	_check(si._clock.phase == RoundClock.Phase.PLAYING, "debug drive runs the clock")
	_check_eq(tm.task_count(), 1, "debug drive assigns one task to slot 1")
	_check(tm.get_task(1)["spotLabel"] != "", "debug drive assigns a real target")

	Input.action_press("ui_right")
	Input.action_press("ui_up")
	_check_eq(si.get_input(1), Vector2(1, 1), "arrows steer slot 1 in debug drive")
	_check_eq(si.get_input(2), Vector2.ZERO, "other slots stay neutral in debug drive")
	Input.action_release("ui_right")
	Input.action_release("ui_up")

	si._physics_process(1.0)
	_check_eq(si._clock.remaining_ms, 179000, "debug drive ticks the clock")

	var parked_at := int(tm.get_task(1)["assignedMs"]) + 12400
	tm.update_tasks(0.6, _parked_cars(tm, 0.0), parked_at)
	_check_eq(si._scores.total(1), 103, "a debug park scores the task")
	_check_eq(si._phase, "results", "parking the only slot ends the debug round")
	_check_eq(si._scores.winner_slot(), 1, "slot 1 wins the debug round")

	si._unhandled_input(f12)
	_check(not si._debug_drive, "F12 again leaves debug drive")
	_check_eq(si._phase, "ready", "leaving debug drive returns to the lobby")
	_check(si._scores.ranking().is_empty(), "leaving debug drive clears scores")

	si._connected = true
	si._unhandled_input(f12)
	_check(not si._debug_drive, "F12 is refused while connected")
	si._connected = false

	parent.free()


func _test_main_round_lifecycle() -> void:
	OS.set_environment("PD_GAME_TOKEN", "test-lifecycle")
	var packed := load("res://scenes/main.tscn") as PackedScene
	_check(packed != null, "main.tscn loads for the lifecycle test")
	if packed == null:
		return

	# Adding the scene to the tree runs every _ready: cars wire to the
	# shared input, the task manager wires car collisions, and the server
	# input creates its clock, scores, and HUD binding.
	var main = packed.instantiate()
	root.add_child(main)
	# Same deferred-_ready note as above: bind explicitly for the assertions.
	# The engine _ready re-run afterwards is idempotent.
	var si = main.get_node("ServerInput")
	si._setup_round_ui()
	var tm = main.get_node("TaskManager")
	var hud = main.get_node("Hud")
	_check(si._clock != null, "the game owns a round clock")
	_check_eq(si._clock.phase, RoundClock.Phase.READY, "the clock starts ready")
	_check_eq(hud.timer_label.text, "03:00", "the timer starts full")
	_check_eq(hud.state_label.text, "Press Enter to start", "attract shows the start prompt")
	_check(
		not hud.get_node("ResultsPanel").visible,
		"results start hidden in the scene",
	)

	si._handle_player_joined({"slot": 1, "label": "P1", "color": "red"})
	si._handle_player_joined({"slot": 2, "label": "P2", "color": "blue"})
	si._handle_host_command({"command": "start", "roundId": 1})
	_check_eq(si._phase, "playing", "host start enters playing")
	_check(si._clock.phase == RoundClock.Phase.PLAYING, "host start runs the clock")
	_check_eq(hud.state_label.text, "Drive to your spot", "playing shows the drive prompt")
	_check_eq(si._scores.ranking(), [1, 2], "round 1 registers both slots")
	_check(
		tm.get_task(1)["spotIndex"] != tm.get_task(2)["spotIndex"],
		"round 1 targets are exclusive",
	)
	_check_eq(hud.scores_box.get_child_count(), 2, "the board lists both slots")

	# Both park centered 12.4 s after assignment: 63 time + 40 precision = 103
	# each. The timestamp is relative to the real assignedMs so wall-clock
	# startup time cannot skew the worked example.
	var parked_at := int(tm.get_task(1)["assignedMs"]) + 12400
	tm.update_tasks(0.6, _parked_cars(tm, 0.0), parked_at)
	_check_eq(si._scores.total(1), 103, "slot 1 scores the parked task")
	_check_eq(si._scores.total(2), 103, "slot 2 scores the parked task")
	# The final park ends the round in the same call, so the results prompt
	# wins over the completion flash (non-final flashes are covered separately
	# in _test_hud_scene).
	_check_eq(hud.state_label.text, "Round over", "the final park ends the round")
	_check(hud.get_node("ResultsPanel").visible, "all-completed ends the round early")
	_check_eq(hud.winner_label.text, "Winner: P1", "equal scores break by slot")
	_check_eq(si._phase, "results", "the game reports results")
	_check(si._clock.phase == RoundClock.Phase.RESULTS, "the clock stops in results")

	# Reset returns to lobby with a zeroed board; a new round rotates targets.
	si._handle_host_command({"command": "reset"})
	_check_eq(si._phase, "ready", "host reset returns to ready")
	_check(not hud.get_node("ResultsPanel").visible, "reset hides results")
	_check_eq(si._clock.remaining_ms, 180000, "reset restores the timer")
	_check_eq(hud.timer_label.text, "03:00", "the timer shows full after reset")
	_check_eq(hud.scores_box.get_child_count(), 2, "the lobby keeps connected rows")
	si._handle_host_command({"command": "start", "roundId": 2})
	_check(tm.get_task(1)["spotIndex"] != 0, "round 2 rotates the starting spot")

	# Expiry with no park: zero scores, no winner, every active slot listed.
	# Hits during the round still reach the results and the ranking.
	tm.register_collision(2)
	tm.register_collision(2)
	tm.register_collision(2)
	si._clock.tick(180.0)
	_check_eq(hud.timer_label.text, "00:00", "expiry zeroes the timer")
	_check(hud.get_node("ResultsPanel").visible, "expiry shows results")
	_check_eq(hud.winner_label.text, "Winner: none", "no completing player means no winner")
	_check_eq(si._scores.total(1), 0, "an unparked player scores zero")
	_check_eq(si._scores.collisions(2), 3, "unfinished hits survive expiry")
	_check_eq(si._scores.ranking()[0], 1, "fewer hits rank first without a park")
	_check(
		str(si._result_lines()[1]).contains("3 hits"),
		"results show unfinished hits",
	)

	# A start without the server roundId never enters playing.
	si._handle_host_command({"command": "reset"})
	si._handle_host_command({"command": "start", "roundId": 0})
	_check_eq(si._phase, "ready", "a start without roundId stays ready")
	_check(si._clock.phase == RoundClock.Phase.READY, "a start without roundId keeps the clock ready")

	# Round 3: slot 2 disconnects but keeps its task; slot 1 parks and the
	# round ends early without waiting for the timer.
	si._handle_host_command({"command": "start", "roundId": 3})
	si._handle_player_left({"slot": 2, "reason": "disconnect"})
	tm.update_tasks(0.6, _parked_cars(tm, 0.0), 8000)
	_check(hud.get_node("ResultsPanel").visible, "one connected park ends the round early")
	_check_eq(si._scores.winner_slot(), 1, "the connected park wins")
	_check_eq(si._scores.tasks_completed(2), 0, "the disconnected slot keeps no score")
	_check_eq(si._scores.total(2), 0, "the disconnected slot scores zero")

	main.free()


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
