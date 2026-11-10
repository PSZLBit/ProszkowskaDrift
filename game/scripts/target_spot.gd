extends Node2D
## On-map target beacon for one parking spot (D-38, MFD baseline).
##
## Filled translucent rectangle in the player's slot color, 4 px border, and
## the spot label (A-01..A-04). The display is the only renderer; the beacon
## must be readable from the spectator distance at 1920x1080 fullscreen.

var spot_size := Vector2(96, 144)
var spot_label := ""
var slot_color := Color.WHITE

var _label: Label


func setup(label_text: String, color: Color, size: Vector2) -> void:
	spot_label = label_text
	slot_color = color
	spot_size = size
	_layout_label()
	if _label != null:
		_label.text = spot_label
	queue_redraw()


func _ready() -> void:
	_label = Label.new()
	_label.text = spot_label
	_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	_label.vertical_alignment = VERTICAL_ALIGNMENT_CENTER
	_layout_label()
	_label.add_theme_font_size_override("font_size", 36)
	_label.add_theme_color_override("font_color", Color.WHITE)
	_label.add_theme_color_override("font_outline_color", Color.BLACK)
	_label.add_theme_constant_override("outline_size", 8)
	add_child(_label)


## The beacon node sits at the spot center, so the label is a spot-sized
## rect centered on the local origin. Anchors are viewport-relative under a
## Node2D parent, so position/size are set directly instead.
func _layout_label() -> void:
	if _label == null:
		return
	_label.position = -spot_size / 2.0
	_label.size = spot_size


func _draw() -> void:
	var rect := Rect2(-spot_size / 2.0, spot_size)
	var fill := slot_color
	fill.a = 0.35
	draw_rect(rect, fill, true)
	draw_rect(rect, slot_color, false, 4.0)
