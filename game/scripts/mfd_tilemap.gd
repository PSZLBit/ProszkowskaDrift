class_name MfdTilemap
extends Node2D
## MFD visual map for proszkowska-mfd-1 (D-38, D-40).
##
## Paints the 40x22 48px lot into the child TileMapLayer from the generated
## TileSet (`game/tools/build_parking_lot_tileset.gd`), so no TileSet or
## purchased art is committed (D-34, D-39). Ground tiles follow the 16px visual
## pass divided by three; markers (spawns, bays, planters) already match
## `mfd_map.gd`, the single implementation source for MFD geometry. Parking
## detection stays authoritative: MFD spots are not 48px-aligned, so bay tile
## art can sit up to half a cell off the spot rect. At runtime
## `ParkingLotArt` swaps the placeholders for the purchased sheets (D-35).

const MfdMapData := preload("res://scripts/mfd_map.gd")

const TILESET_PATH := "res://tilesets/parking_lot.tres"
const TILESET_LOCAL_PATH := "res://tilesets/parking_lot.local.tres"

const MAP_SIZE := Vector2i(40, 22)
const MAP_OFFSET := Vector2(0, 12)

## Atlas source indices in `parking_lot.tres`, in generation order.
const SOURCE_A2_1 := 0
const SOURCE_B_01 := 3

## A2-1 ground tiles: the 16px visual pass used (0,0), (3,0), (6,0), (9,0),
## (0,9), (3,9); the generated TileSet uses a 48px grid, so coordinates are
## divided by three.
const LOT_COORDS := [Vector2i(0, 0), Vector2i(1, 0)]
const RING_COORDS := Vector2i(2, 0)
const KERB_COORDS := Vector2i(3, 0)
const GRASS_COORDS := [Vector2i(0, 3), Vector2i(1, 3)]

## Corner grass islands, 2x2 blocks each, in block coordinates.
const GRASS_ISLANDS := [Vector2i(2, 2), Vector2i(36, 2), Vector2i(2, 18), Vector2i(36, 18)]
const RING_WIDTH := 2

## B-01 bay designs: design A uses columns 0-1, design B columns 2-3; even
## bays use A, odd bays B. A bay is 2 wide x 3 tall cells: one top row and a
## repeated bottom row.
const BAY_TOP_ROW := 2
const BAY_BOTTOM_ROW := 3
const BAY_BASE_COLUMNS := [0, 2]
const BAY_CELLS := Vector2i(2, 3)


func _tileset_path() -> String:
	return ParkingLotTilesetBuilder.preferred_tileset_path(TILESET_PATH, TILESET_LOCAL_PATH)


## Pure selection rule for tests: a local TileSet is only preferred when the
## file exists and the purchased sheets are installed.
static func should_use_local(local_exists: bool, purchased_present: bool) -> bool:
	return ParkingLotMap.should_use_local(local_exists, purchased_present)


func _ready() -> void:
	var tile_map := get_tile_map_layer()
	if tile_map == null:
		push_error("MfdTilemap needs a TileMapLayer child.")
		return
	if tile_map.tile_set == null:
		tile_map.tile_set = load(_tileset_path()) as TileSet
	ParkingLotArt.apply(tile_map.tile_set)
	build()


## Fills the TileMapLayer with the MFD layout. Safe to call more than once.
func build() -> void:
	var tile_map := get_tile_map_layer()
	if tile_map == null:
		push_error("MfdTilemap needs a TileMapLayer child.")
		return
	if tile_map.tile_set == null:
		tile_map.tile_set = load(_tileset_path()) as TileSet
	if tile_map.tile_set == null:
		push_error("Could not load %s." % _tileset_path())
		return

	tile_map.clear()
	for y in MAP_SIZE.y:
		for x in MAP_SIZE.x:
			var block := Vector2i(x, y)
			var coords: Vector2i
			if _is_grass(block):
				coords = GRASS_COORDS[(x + y) % GRASS_COORDS.size()]
			elif _is_border(x, y):
				coords = KERB_COORDS
			elif _is_ring(x, y):
				coords = RING_COORDS
			elif (x * 5 + y * 3) % 7 == 0:
				coords = LOT_COORDS[1]
			else:
				coords = LOT_COORDS[0]
			tile_map.set_cell(block, SOURCE_A2_1, coords)

	for spot in MfdMapData.spot_count():
		_paint_bay(tile_map, spot)

	for bump_raw in MfdMapData.SPEED_BUMPS:
		_paint_bump(tile_map, bump_raw)


## Paints one 2x3 bay over the spot rect. The rect is authoritative for
## detection; its top-left corner is floored to the 48px grid.
func _paint_bay(tile_map: TileMapLayer, spot: int) -> void:
	var rect: Rect2 = MfdMapData.spot_rect(spot)
	var origin := Vector2i(
		int(floor(rect.position.x / 48.0)),
		int(floor((rect.position.y - MAP_OFFSET.y) / 48.0))
	)
	var base_column: int = BAY_BASE_COLUMNS[spot % BAY_BASE_COLUMNS.size()]
	for dy in BAY_CELLS.y:
		var row := BAY_TOP_ROW if dy == 0 else BAY_BOTTOM_ROW
		for dx in BAY_CELLS.x:
			tile_map.set_cell(
				origin + Vector2i(dx, dy),
				SOURCE_B_01,
				Vector2i(base_column + dx, row)
			)


func get_tile_map_layer() -> TileMapLayer:
	return get_node_or_null("TileMapLayer") as TileMapLayer


## Paints one 96x96 bump strip as a solid dark ground band across the road,
## contrasting with the lighter ring tiles in both the committed placeholder
## and purchased art. (An alternating checker was tried first and read as
## random speckles instead of a deliberate strip.) The detection rectangle
## is authoritative; cells are floored to the grid.
func _paint_bump(tile_map: TileMapLayer, bump: Rect2) -> void:
	var x0 := int(floor(bump.position.x / 48.0))
	var x1 := int(floor((bump.position.x + bump.size.x - 1.0) / 48.0))
	var y0 := int(floor((bump.position.y - MAP_OFFSET.y) / 48.0))
	var y1 := int(floor((bump.position.y + bump.size.y - 1.0 - MAP_OFFSET.y) / 48.0))
	for cy in range(y0, y1 + 1):
		for cx in range(x0, x1 + 1):
			tile_map.set_cell(Vector2i(cx, cy), SOURCE_A2_1, LOT_COORDS[0])


func _is_border(x: int, y: int) -> bool:
	return x == 0 or y == 0 or x == MAP_SIZE.x - 1 or y == MAP_SIZE.y - 1


func _is_ring(x: int, y: int) -> bool:
	var edge := mini(mini(x, MAP_SIZE.x - 1 - x), mini(y, MAP_SIZE.y - 1 - y))
	return edge >= 1 and edge <= RING_WIDTH


func _is_grass(block: Vector2i) -> bool:
	for island_raw in GRASS_ISLANDS:
		var island: Vector2i = island_raw
		if Rect2(island, Vector2i(2, 2)).has_point(block):
			return true
	return false
