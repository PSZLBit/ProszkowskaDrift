extends Node
## Single game connection to the Go server.
##
## The server keeps one /ws/game connection, so this node owns the connection and
## exposes a slot-keyed input store to all four cars. A separate ServerInput per
## car would replace the previous game connection and leave only one car alive.
##
## The game token is supplied by the launcher/configuration; PD_GAME_TOKEN is
## the development path until S0-07 injects a per-launch token.

const GameProtocol := preload("res://scripts/game_protocol.gd")
const SlotInputStore := preload("res://scripts/slot_input_store.gd")
const MfdMap := preload("res://scripts/mfd_map.gd")

const HEARTBEAT_INTERVAL_SEC := 1.0
const CONNECT_TIMEOUT_MS := 5000
const RECONNECT_BASE_MS := 1000
const RECONNECT_MAX_MS := 30000
const DEV_AUTO_START_DELAY_MS := 1000
const DEV_AUTO_START_RETRY_MS := 2000

signal connection_state_changed(connected: bool)
signal slot_joined(slot: int, label: String, color: String)
signal slot_left(slot: int, reason: String)
signal config_received(config: Dictionary)
signal host_command_received(command: String, slot: int, round_id: int)
signal task_completed(slot: int, result: Dictionary)
signal fatal_error(code: String, message: String)

## Base WebSocket URL. Do not put the token in this URL; use game_token or the
## PD_GAME_TOKEN environment variable so the token is not committed.
@export var server_url := "ws://127.0.0.1:8080/ws/game"

## Game token from the launcher/configuration. Empty falls back to token_env_var.
@export var game_token := ""
@export var token_env_var := "PD_GAME_TOKEN"

## Development-only convenience. Production starts a round with the hosted game
## keys (Enter); leave this false for the scaffold default.
@export var dev_auto_start_round := false

var _socket: WebSocketPeer
var _store = SlotInputStore.new()

var _connected := false
var _exiting := false
var _fatal_reason := ""
var _connect_started_ms := 0
var _reconnect_at_ms := 0
var _reconnect_attempt := 0

var _got_game_hello := false
var _got_config := false
var _game_ready_sent := false
var _heartbeat_accum := 0.0
var _phase := "attract"
var _start_accepted := false
var _next_auto_start_at_ms := 0
var _saw_input: Dictionary = {}
var _logged_types: Dictionary = {}
var _config: Dictionary = {}
var _active_slots: Dictionary = {}
var _round_id := 0

## MFD round state owned by the game (D-38). The clock counts down while the
## phase is playing, the scores hold one task result per active slot, and the
## HUD renders whatever this node pushes. The server stays the phase and slot
## authority; the game never starts, pauses, or resets on its own.
var _clock: RoundClock
var _scores: ScoreModel
var _hud: Hud


func _ready() -> void:
	_setup_round_ui()

	if "--dev-auto-start" in OS.get_cmdline_user_args():
		dev_auto_start_round = true

	game_token = game_token.strip_edges()
	if game_token.is_empty():
		game_token = OS.get_environment(token_env_var).strip_edges()
	if game_token.is_empty():
		_fatal("missing_token", "Set game_token or %s before starting the game." % token_env_var)
		return

	var env_server_url := OS.get_environment("PD_GAME_URL").strip_edges()
	if not env_server_url.is_empty():
		server_url = env_server_url

	_connect()


## Creates the round clock and score model, binds the HUD, and arms the expiry
## path. Runs before the token check so headless scenes still get the UI.
func _setup_round_ui() -> void:
	_clock = RoundClock.new()
	_scores = ScoreModel.new()
	_clock.finished.connect(_on_clock_finished)
	_hud = _find_hud()
	_hud_call("bind", [_clock])
	_show_lobby([])


func _find_hud():
	var parent := get_parent()
	if parent == null:
		return null
	return parent.get_node_or_null("Hud")


func _hud_call(method: String, args: Array) -> void:
	if _hud == null:
		return
	_hud.callv(method, args)


func _exit_tree() -> void:
	_exiting = true
	if _socket != null:
		_socket.close(1000, "game_exit")
		_socket = null


func _process(delta: float) -> void:
	if _exiting or not _fatal_reason.is_empty():
		return

	if _socket == null:
		if _reconnect_at_ms > 0 and Time.get_ticks_msec() >= _reconnect_at_ms:
			_connect()
		return

	_socket.poll()
	match _socket.get_ready_state():
		WebSocketPeer.STATE_CONNECTING:
			if Time.get_ticks_msec() - _connect_started_ms >= CONNECT_TIMEOUT_MS:
				_schedule_reconnect("connect timed out")
		WebSocketPeer.STATE_OPEN:
			if not _connected:
				_on_open()
			_read_packets()
			if _socket != null:
				_tick_timers(delta)
				_tick_round(delta)
		WebSocketPeer.STATE_CLOSING:
			pass
		WebSocketPeer.STATE_CLOSED:
			# Drain a final error frame before handling the close. The server
			# sends auth_failed/protocol_mismatch and then closes immediately.
			_read_packets()
			if _socket != null:
				_on_closed()


## Returns the latest input for one car as (steer, throttle), each in [-1, 1].
## The store applies the 500 ms per-slot timeout.
func get_input(slot: int) -> Vector2:
	return _store.get_input(slot, Time.get_ticks_msec())


## Clears all current samples, for example when the connection drops.
func clear_input() -> void:
	_store.clear_all()
	_saw_input.clear()


## True for a fresh Q key press. Q quits the local game process, so it is
## checked before the connection and host-authority guards.
static func is_quit_event(event: InputEvent) -> bool:
	if not (event is InputEventKey):
		return false

	var key := event as InputEventKey
	return key.pressed and not key.echo and key.keycode == KEY_Q


func _unhandled_input(event: InputEvent) -> void:
	if is_quit_event(event):
		print("[ServerInput] Quit key pressed; exiting.")
		get_tree().quit()
		return

	if not _connected or not _game_ready_sent:
		return
	if not (event is InputEventKey):
		return

	var key := event as InputEventKey
	if not key.pressed or key.echo:
		return

	match key.keycode:
		KEY_ENTER, KEY_KP_ENTER:
			if _phase == "paused":
				_send_host_request("resume")
			elif _phase in ["attract", "ready", "results"]:
				_send_host_request("start")
		KEY_P:
			if _phase == "paused":
				_send_host_request("resume")
			elif _phase == "playing":
				_send_host_request("pause")
		KEY_R:
			if _phase in ["playing", "paused", "results"]:
				_send_host_request("reset")


func _connect() -> void:
	if _exiting or not _fatal_reason.is_empty():
		return

	_reconnect_at_ms = 0
	_reset_connection_state()
	_socket = WebSocketPeer.new()
	_connect_started_ms = Time.get_ticks_msec()

	var err := _socket.connect_to_url(_build_url())
	if err != OK:
		_socket = null
		_schedule_reconnect("connect_to_url failed (error %d)" % err)
		return
	print("[ServerInput] Connecting to %s" % _safe_url())


func _on_open() -> void:
	_connected = true
	_got_game_hello = false
	_got_config = false
	_game_ready_sent = false
	_heartbeat_accum = 0.0
	_start_accepted = false
	_next_auto_start_at_ms = 0
	_phase = "attract"
	_saw_input.clear()
	_logged_types.clear()
	_store.clear_all()
	print("[ServerInput] Connected.")
	connection_state_changed.emit(true)


func _on_closed() -> void:
	if _socket == null:
		return

	var code := _socket.get_close_code()
	var reason := _socket.get_close_reason()
	var was_connected := _connected
	_socket = null
	_connected = false
	_reset_connection_state()

	if _exiting:
		return
	if GameProtocol.is_fatal_error(reason):
		_fatal(reason, "server closed the connection (%s)" % reason)
		return
	if not _fatal_reason.is_empty():
		return

	connection_state_changed.emit(false)
	if was_connected:
		print("[ServerInput] Disconnected (code %d, reason %s)." % [code, reason])
	else:
		push_warning("[ServerInput] Connection failed (code %d, reason %s)." % [code, reason])
	_schedule_reconnect("connection closed")


func _read_packets() -> void:
	while _socket != null and _socket.get_available_packet_count() > 0:
		var packet := _socket.get_packet()
		if not _socket.was_string_packet():
			continue
		_handle_text(packet.get_string_from_utf8())


func _handle_text(text: String) -> void:
	var parsed := GameProtocol.parse(text)
	if not bool(parsed["ok"]):
		if bool(parsed.get("fatal", false)):
			_fatal("protocol_error", str(parsed["error"]))
		else:
			_log_once("bad_message", "[ServerInput] Ignoring message: %s" % str(parsed["error"]))
		return

	var type := str(parsed["type"])
	var data: Dictionary = parsed["data"]

	match type:
		"game_hello":
			_handle_game_hello(data)
		"config":
			_handle_config(data)
		"input":
			_handle_input(data)
		"player_joined":
			_handle_player_joined(data)
		"player_left":
			_handle_player_left(data)
		"host_command":
			_handle_host_command(data)
		"host_denied":
			_handle_host_denied(data)
		"error":
			_handle_error(data)
		_:
			_log_once("unknown_%s" % type, "[ServerInput] Ignoring unknown message type %s." % type)


func _handle_game_hello(data: Dictionary) -> void:
	var result := GameProtocol.validate_game_hello(data)
	if not bool(result["ok"]):
		_fatal("protocol_mismatch", str(result["error"]))
		return

	_got_game_hello = true
	print("[ServerInput] game_hello (protocol %d)." % GameProtocol.VERSION)
	_maybe_send_game_ready()


func _handle_config(data: Dictionary) -> void:
	var result := GameProtocol.validate_config(data)
	if not bool(result["ok"]):
		_fatal("invalid_config", str(result["error"]))
		return

	_got_config = true
	_config = result
	print(
		"[ServerInput] config round=%dms tasks=%d maxPlayers=%d."
		% [
			int(result["roundDurationMs"]),
			int(result["tasksPerPlayer"]),
			int(result["maxPlayers"]),
		]
	)
	if _clock != null:
		_clock.configure(int(result["roundDurationMs"]))
	config_received.emit(_config)
	_maybe_send_game_ready()


func _maybe_send_game_ready() -> void:
	if not _connected or _game_ready_sent or not _got_game_hello or not _got_config:
		return

	if _send("game_ready", {"gameVersion": "0.1.0", "mapVersion": MfdMap.MAP_VERSION}):
		_game_ready_sent = true
		_heartbeat_accum = 0.0
		_reconnect_attempt = 0
		_next_auto_start_at_ms = Time.get_ticks_msec() + DEV_AUTO_START_DELAY_MS
		print("[ServerInput] game_ready sent.")


func _handle_input(data: Dictionary) -> void:
	var result := GameProtocol.parse_input(data)
	if not bool(result["ok"]):
		_log_once("bad_input", "[ServerInput] Ignoring input: %s" % str(result["error"]))
		return

	var slot := int(result["slot"])
	if not _store.set_input(
		slot,
		float(result["steer"]),
		float(result["throttle"]),
		int(result["seq"]),
		Time.get_ticks_msec(),
	):
		return

	if not _saw_input.has(slot):
		_saw_input[slot] = true
		print(
			"[ServerInput] slot %d input steer=%.3f throttle=%.3f."
			% [slot, float(result["steer"]), float(result["throttle"])]
		)


func _handle_player_joined(data: Dictionary) -> void:
	var slot := int(data.get("slot", 0))
	if slot < 1 or slot > GameProtocol.MAX_PLAYERS:
		_log_once("bad_join", "[ServerInput] Ignoring player_joined with slot %d." % slot)
		return

	# A reconnect re-sends player_joined and resets the controller sequence.
	_store.clear_slot(slot)
	_saw_input.erase(slot)
	_active_slots[slot] = true
	print(
		"[ServerInput] player_joined slot=%d label=%s color=%s."
		% [slot, str(data.get("label", "")), str(data.get("color", ""))]
	)
	slot_joined.emit(slot, str(data.get("label", "")), str(data.get("color", "")))
	_task_manager_call("set_slot_connected", [slot, true])
	_refresh_board()


func _handle_player_left(data: Dictionary) -> void:
	var slot := int(data.get("slot", 0))
	if slot < 1 or slot > GameProtocol.MAX_PLAYERS:
		_log_once("bad_left", "[ServerInput] Ignoring player_left with slot %d." % slot)
		return

	var reason := str(data.get("reason", ""))
	_store.clear_slot(slot)
	_saw_input.erase(slot)
	_active_slots.erase(slot)
	print("[ServerInput] player_left slot=%d reason=%s." % [slot, reason])
	slot_left.emit(slot, reason)
	_task_manager_call("set_slot_connected", [slot, false])
	_refresh_board()
	_check_early_finish()


func _handle_host_command(data: Dictionary) -> void:
	var command := str(data.get("command", ""))
	var slot := int(data.get("slot", 0))
	if command == "start":
		# ADR-0004: the server owns the round identifier. The game never
		# invents its own counter; without a valid roundId there is no task
		# assignment, so the round cannot start its tasks.
		var round_id := int(data.get("roundId", 0))
		if round_id <= 0:
			push_error("[ServerInput] host_command start without a valid roundId; skipping task assignment.")
			_phase = "playing"
			clear_input()
			host_command_received.emit(command, slot, 0)
			return
		_round_id = round_id
	print("[ServerInput] host_command %s slot=%d round=%d." % [command, slot, _round_id])

	match command:
		"start":
			_start_accepted = true
			_phase = "playing"
			clear_input()
			_start_task_round()
		"pause":
			_phase = "paused"
			clear_input()
			_task_manager_call("set_running", [false, Time.get_ticks_msec()])
			if _clock != null:
				_clock.pause()
			_show_paused()
		"resume":
			_phase = "playing"
			_task_manager_call("set_running", [true, Time.get_ticks_msec()])
			if _clock != null:
				_clock.resume()
			_show_playing()
		"reset":
			_phase = "ready"
			clear_input()
			_task_manager_call("set_running", [false, Time.get_ticks_msec()])
			_task_manager_call("clear_round", [])
			if _clock != null:
				_clock.reset()
			_scores.reset_round()
			_show_lobby(_connected_slots())
		"end":
			_phase = "ended"
			clear_input()
			_task_manager_call("set_running", [false, Time.get_ticks_msec()])
			_task_manager_call("clear_round", [])
			_active_slots.clear()
			if _clock != null:
				_clock.reset()
			_scores.reset_round()
			_show_lobby([])
		"kick":
			_store.clear_slot(slot)
			_saw_input.erase(slot)
			_active_slots.erase(slot)
			_scores.unregister(slot)
			_task_manager_call("release_slot", [slot])
			_refresh_board()

	host_command_received.emit(command, slot, _round_id)


## Starts one MFD round: every connected slot gets one exclusive task.
## Late join is between rounds (D-24), so no task is assigned mid-round.
func _start_task_round() -> void:
	var slots := _connected_slots()
	_scores.reset_round()
	for slot in slots:
		_scores.register(slot)
	var manager := _task_manager()
	if manager == null:
		return
	var duration_ms := int(_config.get("roundDurationMs", 180000))
	if _clock != null:
		_clock.configure(duration_ms)
		_clock.start()
	manager.call("start_round", slots, _round_id, duration_ms, Time.get_ticks_msec())
	manager.call("set_running", true, Time.get_ticks_msec())
	if not manager.is_connected("task_completed", _on_task_completed):
		manager.connect("task_completed", _on_task_completed)
	_show_playing()
	_send_playing_phase()


## Sorted list of currently connected slots.
func _connected_slots() -> Array:
	var slots: Array = []
	for slot_raw in _active_slots.keys():
		slots.append(int(slot_raw))
	slots.sort()
	return slots


## Records a completed task: score it, refresh the board, tell the server, and
## end the round early when every connected player has completed their task.
func _on_task_completed(slot: int, result: Dictionary) -> void:
	print(
		"[ServerInput] task_completed slot=%d task=%s offset=%.1f angle=%.1f collisions=%d."
		% [
			slot,
			str(result.get("taskId", "")),
			float(result.get("centerOffset", 0.0)),
			float(result.get("angleDiff", 0.0)),
			int(result.get("collisions", 0)),
		]
	)
	if _scores.record_completion(
		slot,
		int(result.get("taskId", 0)),
		int(result.get("completionMs", 0)),
		float(result.get("centerOffset", 0.0)),
		float(result.get("angleDiff", 0.0)),
		int(result.get("collisions", 0)),
	):
		_send("task_result", _task_result_payload(slot))
	_hud_call("flash_notice", ["P%d parked!" % slot])
	_refresh_board()
	_check_early_finish()
	task_completed.emit(slot, result)


## Same payload the server forwards to controllers, with the slot added.
func _task_result_payload(slot: int) -> Dictionary:
	var payload := _scores.task_result_payload(slot)
	payload["slot"] = slot
	return payload


## The round timer expired. Stop the cars, freeze detection, and publish the
## results; the host starts the next round.
func _on_clock_finished() -> void:
	_end_round()


## Ends the round early when at least one player is connected and every
## connected player has completed their task. A disconnected slot does not
## block the early end.
func _check_early_finish() -> void:
	if _phase != "playing":
		return
	var slots := _connected_slots()
	if slots.is_empty():
		return
	var manager := _task_manager()
	if manager == null:
		return
	for slot in slots:
		var task: Dictionary = manager.call("get_task", slot)
		if task.is_empty() or str(task.get("state", "")) != "completed":
			return
	_end_round()


## Publishes the MFD results: stops input and detection, shows the results
## panel, and sends phase and round_result to the server.
func _end_round() -> void:
	if _phase != "playing":
		return
	_phase = "results"
	clear_input()
	_task_manager_call("set_running", [false, Time.get_ticks_msec()])
	if _clock != null:
		_clock.show_results()
	var winner := _scores.winner_slot()
	var winner_line := "Winner: P%d" % winner if winner > 0 else "Winner: none"
	_hud_call("set_state", ["Round over"])
	_hud_call("show_results", ["Round over", winner_line, _result_lines()])
	_send("phase", {"phase": "results", "roundId": _round_id, "timeLeftMs": 0})
	_send("round_result", _scores.round_result_payload(_round_id))


## Lobby/attract screen: static prompt, hidden results, zeroed board.
func _show_lobby(slots: Array) -> void:
	_hud_call("hide_results", [])
	_hud_call("set_state", ["Press Enter to start"])
	_refresh_board()


func _show_playing() -> void:
	_hud_call("hide_results", [])
	_hud_call("set_state", ["Drive to your spot"])
	_refresh_board()


func _show_paused() -> void:
	_hud_call("set_state", ["Paused"])


## Rebuilds the scoreboard rows for every registered slot: score, current
## spot or Parked, and hit count. Reconnecting players keep their row.
func _refresh_board() -> void:
	var rows: Array = []
	var manager := _task_manager()
	for slot in _scores.ranking():
		var status := ""
		var hits := _scores.collisions(slot)
		if _scores.tasks_completed(slot) > 0:
			status = "Parked"
		elif not bool(_active_slots.get(slot, false)):
			status = "Reconnecting"
		elif manager != null:
			var task: Dictionary = manager.call("get_task", slot)
			if not task.is_empty():
				status = str(task.get("spotLabel", ""))
				hits = int(task.get("collisions", 0))
		rows.append({
			"slot": slot,
			"score": _scores.total(slot),
			"status": status,
			"hits": hits,
		})
	_hud_call("update_board", [rows])


## Ranked result rows: "1. P2   103   Parked   0 hits   12.4 s".
func _result_lines() -> Array:
	var lines: Array = []
	var rank := 1
	for slot in _scores.ranking():
		var state_word := "Parked" if _scores.tasks_completed(slot) > 0 else "No park"
		var hits := _scores.collisions(slot)
		var time_text := "-"
		var completion := _scores.completion_time_ms(slot)
		if completion >= 0:
			time_text = "%.1f s" % (float(completion) / 1000.0)
		lines.append("%d. P%d   %d   %s   %s   %s" % [
			rank,
			slot,
			_scores.total(slot),
			state_word,
			"1 hit" if hits == 1 else "%d hits" % hits,
			time_text,
		])
		rank += 1
	return lines


func _task_manager():
	var parent := get_parent()
	if parent == null:
		return null
	return parent.get_node_or_null("TaskManager")


func _task_manager_call(method: String, args: Array) -> void:
	var manager := _task_manager()
	if manager == null:
		return
	manager.callv(method, args)


func _handle_host_denied(data: Dictionary) -> void:
	print(
		"[ServerInput] host_denied command=%s code=%s message=%s."
		% [
			str(data.get("command", "")),
			str(data.get("code", "")),
			str(data.get("message", "")),
		]
	)


func _handle_error(data: Dictionary) -> void:
	var code := str(data.get("code", ""))
	var message := str(data.get("message", ""))
	if GameProtocol.is_fatal_error(code):
		_fatal(code, message)
		return
	push_warning("[ServerInput] server error code=%s message=%s." % [code, message])


## Advances the round clock while the accepted phase is playing. The clock only
## emits; starting, pausing, and resetting always come from host commands.
func _tick_round(delta: float) -> void:
	if _clock == null or _phase != "playing":
		return
	_clock.tick(delta)


func _tick_timers(delta: float) -> void:
	if not _game_ready_sent:
		return

	_heartbeat_accum += delta
	while _heartbeat_accum >= HEARTBEAT_INTERVAL_SEC:
		_heartbeat_accum -= HEARTBEAT_INTERVAL_SEC
		if not _send("heartbeat", {"gameTimeMs": Time.get_ticks_msec()}):
			return
		_send_playing_phase()

	if not dev_auto_start_round or _start_accepted:
		return
	if _phase not in ["attract", "ready", "results"]:
		return

	var now := Time.get_ticks_msec()
	if now < _next_auto_start_at_ms:
		return
	_next_auto_start_at_ms = now + DEV_AUTO_START_RETRY_MS
	_send_host_request("start")


## Reports the playing phase with the authoritative time left, so controllers
## see the round timer. Sent at 1 Hz alongside the heartbeat and immediately
## on round start. Other phases are broadcast by the server itself.
func _send_playing_phase() -> void:
	if _phase != "playing" or _clock == null:
		return
	_send("phase", {
		"phase": "playing",
		"timeLeftMs": _clock.remaining_ms,
		"roundId": _round_id,
	})


func _send_host_request(command: String, slot := 0) -> bool:
	var data := {"command": command}
	if slot > 0:
		data["slot"] = slot
	if not _send("host_request", data):
		return false
	print("[ServerInput] host_request %s sent." % command)
	return true


func _send(type: String, data: Dictionary) -> bool:
	if not _connected or _socket == null or _socket.get_ready_state() != WebSocketPeer.STATE_OPEN:
		return false

	var message := {
		"v": GameProtocol.VERSION,
		"type": type,
		"data": data,
	}
	var err := _socket.send_text(JSON.stringify(message))
	if err != OK:
		_schedule_reconnect("send %s failed (error %d)" % [type, err])
		return false
	return true


func _build_url() -> String:
	var separator := "&" if server_url.contains("?") else "?"
	return "%s%stoken=%s" % [server_url, separator, game_token.uri_encode()]


func _reset_connection_state() -> void:
	_connected = false
	_got_game_hello = false
	_got_config = false
	_game_ready_sent = false
	_heartbeat_accum = 0.0
	_phase = "attract"
	_start_accepted = false
	_next_auto_start_at_ms = 0
	_store.clear_all()
	_saw_input.clear()
	_active_slots.clear()
	if _clock != null:
		_clock.reset()
	_scores.reset_round()
	_show_lobby([])
	_task_manager_call("set_running", [false, Time.get_ticks_msec()])


func _schedule_reconnect(reason: String) -> void:
	if _exiting or not _fatal_reason.is_empty():
		return

	if _socket != null:
		_socket.close()
		_socket = null
	_reset_connection_state()

	var delay := mini(RECONNECT_BASE_MS * (1 << mini(_reconnect_attempt, 5)), RECONNECT_MAX_MS)
	_reconnect_attempt += 1
	_reconnect_at_ms = Time.get_ticks_msec() + delay
	push_warning(
		"[ServerInput] %s; retrying in %d ms." % [reason, delay]
	)


func _fatal(code: String, message: String) -> void:
	if not _fatal_reason.is_empty():
		return

	_fatal_reason = "%s: %s" % [code, message]
	_connected = false
	_store.clear_all()
	_saw_input.clear()
	if _socket != null:
		_socket.close(1008, code)
		_socket = null

	push_error("[ServerInput] Fatal %s. Not reconnecting." % _fatal_reason)
	connection_state_changed.emit(false)
	fatal_error.emit(code, message)


func _log_once(key: String, message: String) -> void:
	if _logged_types.has(key):
		return
	_logged_types[key] = true
	print(message)


func _safe_url() -> String:
	var query := server_url.find("?")
	if query >= 0:
		return server_url.substr(0, query)
	return server_url
