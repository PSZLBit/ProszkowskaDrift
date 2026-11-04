class_name Hud
extends CanvasLayer
## Minimal HUD and results overlay shell for the vertical slice (#47).
##
## Layout and values are provisional; final readability is verified on the
## target TV. See `06-development/testing.md`.

var timer_label: Label
var task_label: Label
var results_panel: Control
var results_label: Label
var score_labels: Array[Label] = []


func _ensure_nodes() -> void:
	if timer_label == null:
		timer_label = get_node_or_null("TimerLabel") as Label
	if task_label == null:
		task_label = get_node_or_null("TaskLabel") as Label
	if results_panel == null:
		results_panel = get_node_or_null("ResultsPanel") as Control
	if results_label == null and results_panel != null:
		results_label = results_panel.get_node_or_null("ResultsLabel") as Label
	if score_labels.is_empty():
		var scores_node := get_node_or_null("Scores")
		if scores_node != null:
			for child in scores_node.get_children():
				if child is Label:
					score_labels.append(child as Label)

var clock: RoundClock
var scores: ScoreModel


func _ready() -> void:
	_ensure_nodes()
	if clock == null:
		clock = RoundClock.new()
		add_child(clock)
	if scores == null:
		scores = ScoreModel.new()
		for slot in range(1, 5):
			scores.register(slot)
	bind(clock, scores)
	clock.start()


func _process(delta: float) -> void:
	if clock != null and clock.phase == RoundClock.Phase.PLAYING:
		clock.tick(delta)


func bind(new_clock: RoundClock, new_scores: ScoreModel) -> void:
	_ensure_nodes()
	if clock != null and clock.time_changed.is_connected(_on_time_changed):
		clock.time_changed.disconnect(_on_time_changed)
	if clock != null and clock.phase_changed.is_connected(_on_phase_changed):
		clock.phase_changed.disconnect(_on_phase_changed)
	clock = new_clock
	scores = new_scores
	if clock != null:
		clock.time_changed.connect(_on_time_changed)
		clock.phase_changed.connect(_on_phase_changed)
		_on_time_changed(clock.remaining_ms)
	update_scores()


func _on_time_changed(remaining_ms: int) -> void:
	_ensure_nodes()
	timer_label.text = _format_time(remaining_ms)


func _on_phase_changed(phase: int) -> void:
	if clock != null and phase == int(RoundClock.Phase.RESULTS):
		show_results()


func update_scores() -> void:
	_ensure_nodes()
	if scores == null:
		return
	for slot in range(1, 5):
		var value := 0
		if scores.has_player(slot):
			value = scores.total(slot)
		score_labels[slot - 1].text = "P%d: %d" % [slot, value]


func set_task_state(text: String) -> void:
	_ensure_nodes()
	task_label.text = text


func show_results() -> void:
	_ensure_nodes()
	if scores == null:
		return
	results_panel.visible = true
	var lines: Array[String] = []
	var rank := 1
	for slot in scores.ranking():
		lines.append("%d. P%d - %d" % [rank, slot, scores.total(slot)])
		rank += 1
	results_label.text = "\n".join(lines)


func hide_results() -> void:
	_ensure_nodes()
	results_panel.visible = false


func _format_time(ms: int) -> String:
	var total_seconds := ms / 1000
	var minutes := total_seconds / 60
	var seconds := total_seconds % 60
	return "%02d:%02d" % [minutes, seconds]
