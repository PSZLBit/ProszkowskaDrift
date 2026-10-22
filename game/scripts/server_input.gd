extends Node
## Sprint 0 spike (D-18): receive controller input from the Go server.
##
## This is a single-car spike, not the final protocol layer. It connects to the
## server's /ws/game endpoint, completes the game handshake, sends heartbeats,
## and exposes the latest steer/throttle sample to the car.
##
## The implemented server forwards ForwardedInputData:
##   {"slot":1,"seq":7,"steer":0.5,"throttle":1.0}
## The controller-side InputData shape is also accepted:
##   {"steer":0.5,"throttle":1.0,"ts":123456}
## Slot, seq, and ts are ignored because the spike drives the one car in the
## scene. Missing or null axes are treated as 0.

const PROTOCOL_VERSION := 1
const INPUT_TIMEOUT_MS := 500
const HEARTBEAT_INTERVAL_SEC := 1.0
const RECONNECT_DELAY_MS := 1000
const CONNECT_TIMEOUT_MS := 5000
const AUTO_START_DELAY_MS := 1000
const AUTO_START_RETRY_MS := 2000

## The token is part of the URL for this spike; the node does not validate it.
@export var server_url := "ws://127.0.0.1:8080/ws/game?token=dev-token"

## Sends host_request start until the server accepts it. Useful for manual
## testing because the server forwards input only while the phase is playing.
@export var auto_start_round := true

var _socket: WebSocketPeer
var _connected := false
var _connect_started_ms := 0
var _reconnect_at_ms := 0

var _got_game_hello := false
var _got_config := false
var _game_ready_sent := false
var _heartbeat_accum := 0.0

var _steer := 0.0
var _throttle := 0.0
var _last_input_ms := -1
var _saw_input := false

var _start_accepted := false
var _next_start_at_ms := 0
var _logged_types: Dictionary = {}


func _ready() -> void:
	_connect()


func _exit_tree() -> void:
	if _socket != null:
		_socket.close(1000, "game_exit")
		_socket = null


func _process(delta: float) -> void:
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
			_on_closed()


## Returns the latest input as (steer, throttle), each in [-1, 1].
## Returns neutral input when no sample arrived within the last 500 ms.
func get_input() -> Vector2:
	if _last_input_ms < 0:
		return Vector2.ZERO
	if Time.get_ticks_msec() - _last_input_ms >= INPUT_TIMEOUT_MS:
		return Vector2.ZERO
	return Vector2(_steer, _throttle)


## Clears the current sample immediately, for example on player_left or pause.
func clear_input() -> void:
	_clear_input()


func _connect() -> void:
	_reconnect_at_ms = 0
	_reset_connection()
	_socket = WebSocketPeer.new()
	_connect_started_ms = Time.get_ticks_msec()

	var err := _socket.connect_to_url(server_url)
	if err != OK:
		_socket = null
		_reconnect_at_ms = Time.get_ticks_msec() + RECONNECT_DELAY_MS
		push_warning("[ServerInput] Cannot connect to %s (error %d)." % [_safe_url(), err])
		return

	print("[ServerInput] Connecting to %s" % _safe_url())


func _on_open() -> void:
	_connected = true
	_got_game_hello = false
	_got_config = false
	_game_ready_sent = false
	_heartbeat_accum = 0.0
	_start_accepted = false
	_next_start_at_ms = 0
	_saw_input = false
	_logged_types.clear()
	print("[ServerInput] Connected.")


func _on_closed() -> void:
	if _socket == null:
		return

	var code := _socket.get_close_code()
	var reason := _socket.get_close_reason()
	var was_connected := _connected
	_socket = null
	_reset_connection()
	_reconnect_at_ms = Time.get_ticks_msec() + RECONNECT_DELAY_MS

	if was_connected:
		print("[ServerInput] Disconnected (code %d, reason %s). Retrying." % [code, reason])
	else:
		push_warning(
			"[ServerInput] Connection failed (code %d, reason %s). Retrying."
			% [code, reason]
		)


func _read_packets() -> void:
	while _socket != null and _socket.get_available_packet_count() > 0:
		var packet := _socket.get_packet()
		if not _socket.was_string_packet():
			continue
		_handle_text(packet.get_string_from_utf8())


func _handle_text(text: String) -> void:
	var json := JSON.new()
	if json.parse(text) != OK:
		push_warning("[ServerInput] Ignoring malformed JSON: %s" % json.get_error_message())
		return

	var message = json.data
	if typeof(message) != TYPE_DICTIONARY:
		push_warning("[ServerInput] Ignoring non-object message.")
		return

	var envelope: Dictionary = message
	var type := ""
	var data: Dictionary = {}

	if envelope.has("type"):
		var version := int(envelope.get("v", 0))
		if version != PROTOCOL_VERSION:
			_log_once("version_%d" % version, "[ServerInput] Ignoring protocol version %d." % version)
			return
		type = str(envelope.get("type", ""))
		var raw_data = envelope.get("data", {})
		if typeof(raw_data) != TYPE_DICTIONARY:
			_log_once("bad_data_%s" % type, "[ServerInput] Ignoring %s without object data." % type)
			return
		data = raw_data
	else:
		# Spike fallback: accept a bare InputData object without the envelope.
		type = "input"
		data = envelope

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
	_got_game_hello = true
	print("[ServerInput] game_hello (protocol %d)." % int(data.get("protocolVersion", 0)))
	_maybe_send_game_ready()


func _handle_config(data: Dictionary) -> void:
	_got_config = true
	print(
		"[ServerInput] config round=%dms tasks=%d maxPlayers=%d."
		% [
			int(data.get("roundDurationMs", 0)),
			int(data.get("tasksPerPlayer", 0)),
			int(data.get("maxPlayers", 0)),
		]
	)
	_maybe_send_game_ready()


func _maybe_send_game_ready() -> void:
	if not _connected or _game_ready_sent or not _got_game_hello or not _got_config:
		return

	if _send("game_ready", {"gameVersion": "0.1.0", "mapVersion": "proszkowska-1"}):
		_game_ready_sent = true
		_heartbeat_accum = 0.0
		_next_start_at_ms = Time.get_ticks_msec() + AUTO_START_DELAY_MS
		print("[ServerInput] game_ready sent.")


func _handle_input(data: Dictionary) -> void:
	_steer = _read_axis(data, "steer")
	_throttle = _read_axis(data, "throttle")
	_last_input_ms = Time.get_ticks_msec()

	if not _saw_input:
		_saw_input = true
		print(
			"[ServerInput] first input steer=%.3f throttle=%.3f."
			% [_steer, _throttle]
		)


func _handle_player_joined(data: Dictionary) -> void:
	print(
		"[ServerInput] player_joined slot=%d label=%s color=%s."
		% [
			int(data.get("slot", 0)),
			str(data.get("label", "")),
			str(data.get("color", "")),
		]
	)


func _handle_player_left(data: Dictionary) -> void:
	print(
		"[ServerInput] player_left slot=%d reason=%s."
		% [int(data.get("slot", 0)), str(data.get("reason", ""))]
	)
	_clear_input()


func _handle_host_command(data: Dictionary) -> void:
	var command := str(data.get("command", ""))
	print("[ServerInput] host_command %s." % command)

	if command == "start":
		_start_accepted = true
	elif command in ["pause", "reset", "end", "kick"]:
		_clear_input()


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
	push_warning(
		"[ServerInput] server error code=%s message=%s."
		% [str(data.get("code", "")), str(data.get("message", ""))]
	)


func _tick_timers(delta: float) -> void:
	if not _game_ready_sent:
		return

	_heartbeat_accum += delta
	if _heartbeat_accum >= HEARTBEAT_INTERVAL_SEC:
		_heartbeat_accum -= HEARTBEAT_INTERVAL_SEC
		_send("heartbeat", {"gameTimeMs": Time.get_ticks_msec()})

	if not auto_start_round or _start_accepted:
		return

	var now := Time.get_ticks_msec()
	if now < _next_start_at_ms:
		return

	_next_start_at_ms = now + AUTO_START_RETRY_MS
	if _send("host_request", {"command": "start"}):
		print("[ServerInput] host_request start sent.")


func _read_axis(data: Dictionary, key: String) -> float:
	if not data.has(key):
		return 0.0

	var value = data[key]
	if value == null:
		return 0.0
	if typeof(value) != TYPE_FLOAT and typeof(value) != TYPE_INT:
		_log_once("bad_axis_%s" % key, "[ServerInput] Ignoring non-numeric %s." % key)
		return 0.0

	var axis := float(value)
	if not is_finite(axis):
		_log_once("nonfinite_%s" % key, "[ServerInput] Ignoring non-finite %s." % key)
		return 0.0
	return clampf(axis, -1.0, 1.0)


func _send(type: String, data: Dictionary) -> bool:
	if _socket == null or _socket.get_ready_state() != WebSocketPeer.STATE_OPEN:
		return false

	var message := {
		"v": PROTOCOL_VERSION,
		"type": type,
		"data": data,
	}
	var err := _socket.send_text(JSON.stringify(message))
	if err != OK:
		_schedule_reconnect("send %s failed (error %d)" % [type, err])
		return false
	return true


func _clear_input() -> void:
	_steer = 0.0
	_throttle = 0.0
	_last_input_ms = -1
	_saw_input = false


func _reset_connection() -> void:
	_connected = false
	_got_game_hello = false
	_got_config = false
	_game_ready_sent = false
	_heartbeat_accum = 0.0
	_start_accepted = false
	_next_start_at_ms = 0
	_clear_input()


func _schedule_reconnect(reason: String) -> void:
	push_warning("[ServerInput] %s; reconnecting." % reason)
	if _socket != null:
		_socket.close()
		_socket = null
	_reset_connection()
	_reconnect_at_ms = Time.get_ticks_msec() + RECONNECT_DELAY_MS


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
