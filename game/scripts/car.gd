class_name Car
extends CharacterBody2D
## Top-down arcade car for the parking game (D-19).
##
## Movement follows the documented arcade model in
## documentation/02-game-domain/game-design.md ("Player and Car"):
## throttle and brake/reverse, steering that becomes less effective at very
## low speed, friction, and a maximum speed. No drift mechanic in MVP.
##
## Input comes from an injected source that provides normalized `steer` and
## `throttle` values through `get_input(slot)`. The controller input scheme is
## still TBD (D-14); the scaffold wires every car to the shared `ServerInput`.

## Player slot this car follows, 1-4. The server input store is keyed by slot.
@export_range(1, 4, 1) var slot := 1

## Sprite-sheet frame used by this car's placeholder body.
@export var sprite_frame := 0

## Optional label override; when empty the car shows "P<slot>".
@export var label_text := ""

## Turn rate at full forward speed, in degrees per second.
@export var max_turn_speed_deg := 220.0

## Speed limits in pixels per second. Provisional tuning values; the final
## values are TBD after the first map playtest (game-design.md).
@export var max_speed := 400.0
@export var max_reverse_speed := 200.0

## Acceleration and braking as px/s^2.
@export var acceleration := 500.0
@export var brake_acceleration := 800.0

## Deceleration when neither accelerating nor braking, px/s^2.
@export var coast_friction := 250.0

## Node that provides (steer, throttle) through get_input(). Leave empty for
## neutral input.
@export var input_source: Node

## Steering input: -1 (left) to 1 (right).
var steer := 0.0

## Throttle input: -1 (brake/reverse) to 1 (throttle).
var throttle := 0.0

## Signed forward speed in px/s; positive = forward, negative = reverse.
var _speed := 0.0


func _ready() -> void:
	var sprite := get_node_or_null("Sprite2D") as Sprite2D
	if sprite != null:
		sprite.frame = sprite_frame

	var label := get_node_or_null("Label") as Label
	if label != null:
		label.text = label_text if not label_text.is_empty() else "P%d" % slot


func _physics_process(delta: float) -> void:
	var input := _gather_input()
	steer = input.x
	throttle = input.y

	_apply_throttle(delta)
	_apply_steering(delta)

	var forward := Vector2.RIGHT.rotated(rotation)
	velocity = forward * _speed
	move_and_slide()

	# Hitting the static environment reduces speed (game-design.md). Never
	# destroys the car or ends the round. Counting collisions for the current
	# task is the job of the future task/scoring module, not this script.
	if get_slide_collision_count() > 0:
		_speed *= 0.5


## Returns the current input as (steer, throttle), each in [-1, 1].
## A missing source or a source without get_input(slot) means neutral input.
func _gather_input() -> Vector2:
	if input_source == null or not input_source.has_method("get_input"):
		return Vector2.ZERO

	var value: Variant = input_source.call("get_input", slot)
	if value is Vector2:
		return value
	return Vector2.ZERO


## Adjusts the signed forward speed from the throttle value.
func _apply_throttle(delta: float) -> void:
	if throttle > 0.0:
		# Accelerate toward the throttle-scaled target speed.
		_speed = move_toward(_speed, max_speed * throttle, acceleration * delta)
	elif throttle < 0.0:
		# Brake toward zero first, then reverse.
		_speed = move_toward(
			_speed, -max_reverse_speed * -throttle, brake_acceleration * delta
		)
	else:
		# Coast to a stop.
		_speed = move_toward(_speed, 0.0, coast_friction * delta)


## Turns the car based on the steer value scaled by signed forward speed.
## At zero speed there is no turning ("less effective at very low speed"),
## and turning is naturally reversed while backing up, as a car does.
## Direction of travel and heading are kept in sync (no drift).
func _apply_steering(delta: float) -> void:
	var speed_ratio := _speed / max_speed
	rotation += deg_to_rad(steer * max_turn_speed_deg * speed_ratio * delta)
