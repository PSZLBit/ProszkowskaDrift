extends RefCounted
## Parser and validator for the server-to-game protocol (protocol.md).
##
## The parser is deliberately small and pure so the game can test it headlessly
## without a WebSocket connection. It requires the protocol envelope; the Sprint
## 0 bare-InputData fallback was removed before the game became a scaffold.

const VERSION := 1
const MAX_PLAYERS := 4
const MAX_ROUND_DURATION_MS := 3_600_000
const MAX_TASKS_PER_PLAYER := 100

## Errors that a reconnect cannot fix. The server uses the same values as the
## WebSocket close reason when it rejects the connection before or during the
## handshake.
const FATAL_ERROR_CODES := ["auth_failed", "kicked", "protocol_mismatch"]


## Parses one WebSocket text frame into an envelope dictionary.
## Returns {ok: true, type, data, seq} or {ok: false, error, fatal}.
static func parse(text: String) -> Dictionary:
	var json := JSON.new()
	if json.parse(text) != OK:
		return _error("malformed JSON: %s" % json.get_error_message())

	var message: Variant = json.data
	if typeof(message) != TYPE_DICTIONARY:
		return _error("message is not a JSON object")
	return parse_envelope(message)


## Validates an already parsed envelope. Missing or unsupported versions are
## fatal because reconnecting cannot fix a protocol mismatch.
static func parse_envelope(envelope: Dictionary) -> Dictionary:
	var version_raw: Variant = envelope.get("v", 0)
	if typeof(version_raw) != TYPE_INT and typeof(version_raw) != TYPE_FLOAT:
		return _error("missing protocol version", true)

	var version := int(version_raw)
	if version != VERSION:
		return _error("unsupported protocol version %d" % version, true)

	var type_raw: Variant = envelope.get("type", "")
	if typeof(type_raw) != TYPE_STRING or str(type_raw).is_empty():
		return _error("missing or invalid message type")

	var type := str(type_raw)
	var raw_data: Variant = envelope.get("data", {})
	if typeof(raw_data) != TYPE_DICTIONARY:
		return _error("%s without an object data payload" % type)

	return {
		"ok": true,
		"type": type,
		"data": raw_data,
		"seq": envelope.get("seq", null),
	}


## True when the game must stop reconnecting instead of retrying.
static func is_fatal_error(code: String) -> bool:
	return code in FATAL_ERROR_CODES


## Checks game_hello.protocolVersion before the game accepts the server.
static func validate_game_hello(data: Dictionary) -> Dictionary:
	var version_raw: Variant = data.get("protocolVersion", null)
	if typeof(version_raw) != TYPE_INT and typeof(version_raw) != TYPE_FLOAT:
		return _error("game_hello is missing protocolVersion", true)

	var version := int(version_raw)
	if version != VERSION:
		return _error(
			"game_hello protocolVersion %d does not match %d" % [version, VERSION],
			true,
		)
	return {"ok": true}


## Validates the configuration values used to size the session.
static func validate_config(data: Dictionary) -> Dictionary:
	var round_raw: Variant = data.get("roundDurationMs", null)
	if typeof(round_raw) != TYPE_INT and typeof(round_raw) != TYPE_FLOAT:
		return _error("config is missing roundDurationMs", true)
	var round_ms := int(round_raw)
	if round_ms <= 0 or round_ms > MAX_ROUND_DURATION_MS:
		return _error("config roundDurationMs is out of range: %d" % round_ms, true)

	var tasks_raw: Variant = data.get("tasksPerPlayer", null)
	if typeof(tasks_raw) != TYPE_INT and typeof(tasks_raw) != TYPE_FLOAT:
		return _error("config is missing tasksPerPlayer", true)
	var tasks := int(tasks_raw)
	if tasks <= 0 or tasks > MAX_TASKS_PER_PLAYER:
		return _error("config tasksPerPlayer is out of range: %d" % tasks, true)

	var players_raw: Variant = data.get("maxPlayers", null)
	if typeof(players_raw) != TYPE_INT and typeof(players_raw) != TYPE_FLOAT:
		return _error("config is missing maxPlayers", true)
	var players := int(players_raw)
	if players < 1 or players > MAX_PLAYERS:
		return _error("config maxPlayers is out of range: %d" % players, true)

	return {
		"ok": true,
		"roundDurationMs": round_ms,
		"tasksPerPlayer": tasks,
		"maxPlayers": players,
	}


## Validates one forwarded input sample. Slot and sequence are required because
## the game must map input to the correct car and drop duplicates.
static func parse_input(data: Dictionary) -> Dictionary:
	var slot_raw: Variant = data.get("slot", null)
	if typeof(slot_raw) != TYPE_INT and typeof(slot_raw) != TYPE_FLOAT:
		return _error("input is missing slot")
	var slot := int(slot_raw)
	if slot < 1 or slot > MAX_PLAYERS:
		return _error("input slot is out of range: %d" % slot)

	var seq_raw: Variant = data.get("seq", null)
	if typeof(seq_raw) != TYPE_INT and typeof(seq_raw) != TYPE_FLOAT:
		return _error("input is missing seq")
	var seq := int(seq_raw)
	if seq < 0:
		return _error("input seq must be non-negative")

	var steer := read_axis(data, "steer")
	if not bool(steer["ok"]):
		return steer
	var throttle := read_axis(data, "throttle")
	if not bool(throttle["ok"]):
		return throttle

	return {
		"ok": true,
		"slot": slot,
		"seq": seq,
		"steer": float(steer["value"]),
		"throttle": float(throttle["value"]),
	}


## Reads one optional axis, treating missing/null as neutral. Non-numeric and
## non-finite values are rejected; valid values are clamped defensively.
static func read_axis(data: Dictionary, key: String) -> Dictionary:
	if not data.has(key):
		return {"ok": true, "value": 0.0}

	var value: Variant = data[key]
	if value == null:
		return {"ok": true, "value": 0.0}
	if typeof(value) != TYPE_FLOAT and typeof(value) != TYPE_INT:
		return _error("%s is not numeric" % key)

	var axis := float(value)
	if not is_finite(axis):
		return _error("%s is not finite" % key)
	return {"ok": true, "value": clampf(axis, -1.0, 1.0)}


static func _error(message: String, fatal := false) -> Dictionary:
	return {"ok": false, "error": message, "fatal": fatal}
