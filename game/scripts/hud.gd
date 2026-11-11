class_name Hud
extends CanvasLayer
## MFD display HUD and results overlay (D-38, issue #47).
##
## A pure view. `game/scripts/server_input.gd` owns the clock, the scores, and
## the round state, and pushes every change explicitly: timer text follows the
## bound clock, the scoreboard follows update_board rows, and the state line
## follows set_state with short-lived flash_notice overlays. Layout follows the
## MFD HUD copy and minimum text sizes in the game design document.

const MfdMap := preload("res://scripts/mfd_map.gd")

## Seconds a flash notice (for example "P1 parked!") stays on the state line.
const NOTICE_SEC := 2.0

var timer_label: Label
var scores_box: VBoxContainer
var state_label: Label
var results_panel: Control
var title_label: Label
var winner_label: Label
var results_rows: VBoxContainer
var prompt_label: Label

var _clock: RoundClock
var _rows: Dictionary = {}
var _state_base := "Press Enter to start"
var _notice_left := 0.0


func _ready() -> void:
	_ensure_nodes()
	if state_label != null:
		state_label.text = _state_base


func _process(delta: float) -> void:
	if _notice_left <= 0.0:
		return
	_notice_left -= delta
	if _notice_left <= 0.0:
		_notice_left = 0.0
		_ensure_nodes()
		if state_label != null:
			state_label.text = _state_base


## Binds the round clock for live timer text. Called by the round owner.
func bind(clock: RoundClock) -> void:
	_ensure_nodes()
	if _clock != null and _clock.time_changed.is_connected(_on_time_changed):
		_clock.time_changed.disconnect(_on_time_changed)
	_clock = clock
	if _clock != null:
		_clock.time_changed.connect(_on_time_changed)
		_on_time_changed(_clock.remaining_ms)


## Replaces the scoreboard rows. Each row is {slot, score, status, hits} and
## renders as "P1   120   Parked        1 hit". Only active slots are passed in.
func update_board(rows: Array) -> void:
	_ensure_nodes()
	if scores_box == null:
		return
	for slot in _rows.keys():
		var old_row := _rows[slot] as Control
		if old_row != null:
			scores_box.remove_child(old_row)
			old_row.free()
	_rows.clear()
	for entry_raw in rows:
		if not (entry_raw is Dictionary):
			continue
		var entry: Dictionary = entry_raw
		var slot := int(entry.get("slot", 0))
		if slot < 1 or slot > 4:
			continue
		var row := HBoxContainer.new()
		row.add_theme_constant_override("separation", 8)
		var chip := ColorRect.new()
		chip.custom_minimum_size = Vector2(24, 24)
		chip.color = MfdMap.slot_color(slot)
		row.add_child(chip)
		var label := Label.new()
		label.add_theme_font_size_override("font_size", 28)
		var hits := int(entry.get("hits", 0))
		label.text = "P%d   %d   %-10s   %s" % [
			slot,
			int(entry.get("score", 0)),
			str(entry.get("status", "")),
			"1 hit" if hits == 1 else "%d hits" % hits,
		]
		row.add_child(label)
		scores_box.add_child(row)
		_rows[slot] = row


## Sets the persistent state line copy.
func set_state(text: String) -> void:
	_state_base = text
	_notice_left = 0.0
	_ensure_nodes()
	if state_label != null:
		state_label.text = text


## Shows a short-lived state line overlay, restoring the base copy after
## NOTICE_SEC seconds.
func flash_notice(text: String) -> void:
	_ensure_nodes()
	_notice_left = NOTICE_SEC
	if state_label != null:
		state_label.text = text


## Shows the results panel: title, winner line, one ranked row per player as
## "1. P2   103   Parked   0 hits   12.4 s" ("-" time when not completed),
## and the next-round prompt.
func show_results(title: String, winner_line: String, lines: Array) -> void:
	_ensure_nodes()
	if results_panel != null:
		results_panel.visible = true
	if title_label != null:
		title_label.text = title
	if winner_label != null:
		winner_label.text = winner_line
	if results_rows != null:
		for child in results_rows.get_children():
			results_rows.remove_child(child)
			child.free()
		for line_raw in lines:
			var line := Label.new()
			line.add_theme_font_size_override("font_size", 28)
			line.text = str(line_raw)
			results_rows.add_child(line)


func hide_results() -> void:
	_ensure_nodes()
	if results_panel != null:
		results_panel.visible = false


func _on_time_changed(remaining_ms: int) -> void:
	_ensure_nodes()
	if timer_label != null:
		timer_label.text = _format_time(remaining_ms)


func _ensure_nodes() -> void:
	if timer_label == null:
		timer_label = get_node_or_null("TimerLabel") as Label
	if scores_box == null:
		scores_box = get_node_or_null("Scores") as VBoxContainer
	if state_label == null:
		state_label = get_node_or_null("StateLabel") as Label
	if results_panel == null:
		results_panel = get_node_or_null("ResultsPanel") as Control
	if title_label == null and results_panel != null:
		title_label = results_panel.get_node_or_null("TitleLabel") as Label
	if winner_label == null and results_panel != null:
		winner_label = results_panel.get_node_or_null("WinnerLabel") as Label
	if results_rows == null and results_panel != null:
		results_rows = results_panel.get_node_or_null("Rows") as VBoxContainer
	if prompt_label == null and results_panel != null:
		prompt_label = results_panel.get_node_or_null("PromptLabel") as Label


func _format_time(ms: int) -> String:
	var total_seconds := maxi(0, ms) / 1000
	var minutes := total_seconds / 60
	var seconds := total_seconds % 60
	return "%02d:%02d" % [minutes, seconds]
