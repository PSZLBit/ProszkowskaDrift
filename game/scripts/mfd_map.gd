extends RefCounted
## MFD map data for proszkowska-mfd-1 (D-38).
##
## Single implementation source is documentation/02-game-domain/game-design.md
## ("MFD Baseline"). Task and scoring logic must read coordinates from here and
## must not hard-code them. Coordinates are viewport pixels, origin top-left.

const MAP_VERSION := "proszkowska-mfd-1"

## Player car collision rectangle, in pixels.
const CAR_SIZE := Vector2(73.75, 32.5)

## Parking bays: 96x144 px, preferred facing north (-90 degrees).
const SPOTS := [
	{"label": "A-01", "center": Vector2(384, 540), "size": Vector2(96, 144), "facing_deg": -90.0},
	{"label": "A-02", "center": Vector2(768, 540), "size": Vector2(96, 144), "facing_deg": -90.0},
	{"label": "A-03", "center": Vector2(1152, 540), "size": Vector2(96, 144), "facing_deg": -90.0},
	{"label": "A-04", "center": Vector2(1536, 540), "size": Vector2(96, 144), "facing_deg": -90.0},
]

## Spawn points on the bottom road, facing north.
const SPAWNS := {
	1: {"pos": Vector2(672, 1000), "facing_deg": -90.0},
	2: {"pos": Vector2(864, 1000), "facing_deg": -90.0},
	3: {"pos": Vector2(1056, 1000), "facing_deg": -90.0},
	4: {"pos": Vector2(1248, 1000), "facing_deg": -90.0},
}

## Static obstacle centers (96x96 planters). Boundary walls are separate.
const PLANTER_CENTERS := [Vector2(240, 540), Vector2(1680, 540)]
const PLANTER_SIZE := Vector2(96, 96)

## Slot colors for beacons and HUD, matching the protocol slot order.
const SLOT_COLORS := {
	1: Color(1.0, 0.25, 0.25),
	2: Color(0.35, 0.55, 1.0),
	3: Color(0.35, 1.0, 0.5),
	4: Color(1.0, 0.9, 0.3),
}


static func spot_count() -> int:
	return SPOTS.size()


static func get_spot(index: int) -> Dictionary:
	return SPOTS[posmod(index, SPOTS.size())]


static func spot_rect(index: int) -> Rect2:
	var spot: Dictionary = get_spot(index)
	var center: Vector2 = spot["center"]
	var size: Vector2 = spot["size"]
	return Rect2(center - size / 2.0, size)


static func spot_label(index: int) -> String:
	return str(get_spot(index)["label"])


static func spot_facing_deg(index: int) -> float:
	return float(get_spot(index)["facing_deg"])


static func spawn_pos(slot: int) -> Vector2:
	if not SPAWNS.has(slot):
		return Vector2.ZERO
	return SPAWNS[slot]["pos"]


static func spawn_facing_deg(slot: int) -> float:
	if not SPAWNS.has(slot):
		return -90.0
	return float(SPAWNS[slot]["facing_deg"])


static func planter_rects() -> Array:
	var rects: Array = []
	for center_raw in PLANTER_CENTERS:
		var center: Vector2 = center_raw
		rects.append(Rect2(center - PLANTER_SIZE / 2.0, PLANTER_SIZE))
	return rects


static func slot_color(slot: int) -> Color:
	return SLOT_COLORS.get(slot, Color.WHITE)
