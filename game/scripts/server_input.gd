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
signal host_command_received(command: String, slot: int)
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


func _ready() -> void:
	if "--dev-auto-start" in OS.get_cmdline_user_args():
		dev_auto_start_round = true

	game_token = game_token.strip_edges()
	if game_token.is_empty():
		game_token = OS.get_environment(token_env_var).strip_edges()
	if game_token.is_empty():
		_fatal("missing_token", "Set game_token or %s before starting the game." % token_env_var)
		return
	_connect()


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


func _unhandled_input(event: InputEvent) -> void:
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
	config_received.emit(_config)
	_maybe_send_game_ready()


func _maybe_send_game_ready() -> void:
	if not _connected or _game_ready_sent or not _got_game_hello or not _got_config:
		return

	if _send("game_ready", {"gameVersion": "0.1.0", "mapVersion": "proszkowska-1"}):
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
	print(
		"[ServerInput] player_joined slot=%d label=%s color=%s."
		% [slot, str(data.get("label", "")), str(data.get("color", ""))]
	)
	slot_joined.emit(slot, str(data.get("label", "")), str(data.get("color", "")))


func _handle_player_left(data: Dictionary) -> void:
	var slot := int(data.get("slot", 0))
	if slot < 1 or slot > GameProtocol.MAX_PLAYERS:
		_log_once("bad_left", "[ServerInput] Ignoring player_left with slot %d." % slot)
		return

	var reason := str(data.get("reason", ""))
	_store.clear_slot(slot)
	_saw_input.erase(slot)
	print("[ServerInput] player_left slot=%d reason=%s." % [slot, reason])
	slot_left.emit(slot, reason)


func _handle_host_command(data: Dictionary) -> void:
	var command := str(data.get("command", ""))
	var slot := int(data.get("slot", 0))
	print("[ServerInput] host_command %s slot=%d." % [command, slot])

	match command:
		"start":
			_start_accepted = true
			_phase = "playing"
			clear_input()
		"pause":
			_phase = "paused"
			clear_input()
		"resume":
			_phase = "playing"
		"reset":
			_phase = "ready"
			clear_input()
		"end":
			_phase = "ended"
			clear_input()
		"kick":
			_store.clear_slot(slot)
			_saw_input.erase(slot)

	host_command_received.emit(command, slot)


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


func _tick_timers(delta: float) -> void:
	if not _game_ready_sent:
		return

	_heartbeat_accum += delta
	while _heartbeat_accum >= HEARTBEAT_INTERVAL_SEC:
		_heartbeat_accum -= HEARTBEAT_INTERVAL_SEC
		if not _send("heartbeat", {"gameTimeMs": Time.get_ticks_msec()}):
			return

	if not dev_auto_start_round or _start_accepted:
		return
	if _phase not in ["attract", "ready", "results"]:
		return

	var now := Time.get_ticks_msec()
	if now < _next_auto_start_at_ms:
		return
	_next_auto_start_at_ms = now + DEV_AUTO_START_RETRY_MS
	_send_host_request("start")


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
