extends RefCounted
## MFD parking detection helpers (D-38).
##
## A task completes when all four car corners are inside the target rectangle
## (boundary counts as inside) and the car speed from velocity.length() stays
## at or below 40 px/s, both continuously for 0.5 s. Any failed frame resets
## the hold timer. Facing never gates completion; it only affects scoring
## in #47.

## Parked speed threshold in px/s, from velocity.length().
const SPEED_MAX := 40.0
## Required continuous hold time in seconds.
const HOLD_SEC := 0.5
## Tolerance for float comparisons at the tolerance boundary.
const EPSILON := 0.001


## Corners of the car rectangle (73.75x32.5) in world space.
static func car_corners(pos: Vector2, rotation_rad: float, size: Vector2) -> Array:
	var half := size / 2.0
	var local := [
		Vector2(-half.x, -half.y),
		Vector2(half.x, -half.y),
		Vector2(half.x, half.y),
		Vector2(-half.x, half.y),
	]
	var corners: Array = []
	for point_raw in local:
		var point: Vector2 = point_raw
		corners.append(pos + point.rotated(rotation_rad))
	return corners


## True when every corner is inside rect. The boundary counts as inside, so
## the check is inclusive on all four edges (unlike Rect2.has_point, which
## excludes the right/bottom edge).
static func all_inside(corners: Array, rect: Rect2) -> bool:
	for corner_raw in corners:
		var corner: Vector2 = corner_raw
		if corner.x < rect.position.x - EPSILON:
			return false
		if corner.x > rect.position.x + rect.size.x + EPSILON:
			return false
		if corner.y < rect.position.y - EPSILON:
			return false
		if corner.y > rect.position.y + rect.size.y + EPSILON:
			return false
	return true


static func speed_ok(speed: float) -> bool:
	return speed <= SPEED_MAX + EPSILON


## Absolute smallest angle difference in degrees, 0-180.
static func angle_diff_deg(car_deg: float, spot_deg: float) -> float:
	return absf(wrapf(car_deg - spot_deg, -180.0, 180.0))


## Tracks one continuous hold window. Returns true once the hold is satisfied.
## Any failed frame resets the timer to zero; completion is latched by the
## caller so a task completes exactly once.
class Hold:
	extends RefCounted

	var held_sec := 0.0

	func update(delta_sec: float, inside: bool, slow: bool) -> bool:
		if inside and slow:
			held_sec += delta_sec
		else:
			held_sec = 0.0
		return held_sec >= HOLD_SEC

	func reset() -> void:
		held_sec = 0.0
