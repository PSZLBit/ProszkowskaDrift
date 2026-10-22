class_name Car
extends CharacterBody2D
## Top-down arcade car for the parking game (D-19).
##
## Movement follows the documented arcade model in
## documentation/02-game-domain/game-design.md ("Player and Car"):
## throttle and brake/reverse, steering that becomes less effective at very
## low speed, friction, and a maximum speed. No drift mechanic in MVP.
##
## Control scheme is a stub for now (D-14 is still TBD). Real input will be
## the latest validated `steer` and `throttle` forwarded by the server
## (game-design.md, "Movement and Collision Rules"); until then the car only
## coasts to a stop.

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

## Steering input: -1 (left) to 1 (right).
var steer := 0.0

## Throttle input: -1 (brake/reverse) to 1 (throttle).
var throttle := 0.0

## Signed forward speed in px/s; positive = forward, negative = reverse.
var _speed := 0.0


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
## Stub: neutral input only. Do not hard-code a final control scheme here;
## the scheme is not chosen yet (D-14). A keyboard fallback may live here for
## manual testing only. Once real input arrives from the server, a 500 ms
## staleness watchdog (input stops -> car coasts to stop) is added here too.
func _gather_input() -> Vector2:
	# Keyboard example for manual testing; uncomment when trying movement:
	return Vector2(
	 	Input.get_axis("ui_left", "ui_right"),
	 	Input.get_axis("ui_down", "ui_up"),
	 )
	# return Vector2.ZERO


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
