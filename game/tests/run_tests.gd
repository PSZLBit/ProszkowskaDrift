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
