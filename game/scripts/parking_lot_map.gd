class_name ParkingLotMap
extends Node2D
## Demo map for the prepared parking-lot TileSet.
##
## The TileSet is generated from the committed placeholder atlases by
## `game/tools/build_parking_lot_tileset.gd`. In the editor and in CI the map
## shows the placeholders; locally, `ParkingLotArt` swaps in the purchased
## sheets when they exist (D-34). The layout uses a 48x48 grid and a 40x22 map
## (1920x1056 px) centred vertically in the 1920x1080 viewport.
##
## This scene is the base for the parking task (#46). It deliberately contains
## no cars, scoring, or input wiring.

const TILESET_PATH := "res://tilesets/parking_lot.tres"
const TILESET_LOCAL_PATH := "res://tilesets/parking_lot.local.tres"


func _tileset_path() -> String:
	if ResourceLoader.exists(TILESET_LOCAL_PATH):
		# A local TileSet is only usable when the purchased sheets are
		# installed and the file actually loads. Otherwise it is a stale
		# artifact referencing missing textures; loading it fails and the
		# map would render blank, so fall back to the committed path.
		var local := load(TILESET_LOCAL_PATH) as TileSet
		if local != null and ParkingLotTilesetBuilder.purchased_available():
			return TILESET_LOCAL_PATH
		push_warning("Ignoring stale local TileSet %s." % TILESET_LOCAL_PATH)
	if not ResourceLoader.exists(TILESET_PATH):
		_generate_tileset()
	return TILESET_PATH


## Pure selection rule for tests: a local TileSet is only preferred when the
## file exists and the purchased sheets are installed.
static func should_use_local(local_exists: bool, purchased_present: bool) -> bool:
	return local_exists and purchased_present


func _generate_tileset() -> void:
	var builder := load("res://scripts/parking_lot_tileset_builder.gd")
	if builder == null:
		return
	var base_dir := "res://assets/placeholders/large_parking_lot/"
	var files: Dictionary = builder.placeholder_files()
	builder.build(base_dir, files, TILESET_PATH)

const MAP_SIZE := Vector2i(40, 22)
const MAP_OFFSET := Vector2(0, 12)

## Atlas source indices in `parking_lot.tres`, in generation order.
const SOURCE_A2_2 := 1
const SOURCE_B_01 := 3

## Ground and kerb tiles from A2-2, and parking-bay tiles from B-01.
const GROUND_COORDS := Vector2i(1, 3)
const KERB_COORDS := Vector2i(6, 3)
const BAY_TOP_COORDS := Vector2i(0, 2)
const BAY_BOTTOM_COORDS := Vector2i(0, 3)

## Four bays along the top edge, starting at map column 3.
const BAY_FIRST_COLUMN := 3
const BAY_TOP_ROW := 2
const BAY_COUNT := 4


func _ready() -> void:
	var tile_map := get_tile_map_layer()
	if tile_map == null:
		push_error("ParkingLotMap needs a TileMapLayer child.")
		return
	if tile_map.tile_set == null:
		tile_map.tile_set = load(_tileset_path()) as TileSet
	ParkingLotArt.apply(tile_map.tile_set)
	build()


## Fills the TileMapLayer with the demo layout. Safe to call more than once.
func build() -> void:
	var tile_map := get_tile_map_layer()
	if tile_map == null:
		push_error("ParkingLotMap needs a TileMapLayer child.")
		return
	if tile_map.tile_set == null:
		tile_map.tile_set = load(_tileset_path()) as TileSet
	if tile_map.tile_set == null:
		push_error("Could not load %s." % _tileset_path())
		return

	tile_map.clear()
	for y in MAP_SIZE.y:
		for x in MAP_SIZE.x:
			var coords := KERB_COORDS if _is_border(x, y) else GROUND_COORDS
			tile_map.set_cell(Vector2i(x, y), SOURCE_A2_2, coords)

	for bay in BAY_COUNT:
		var column := BAY_FIRST_COLUMN + bay
		tile_map.set_cell(
			Vector2i(column, BAY_TOP_ROW),
			SOURCE_B_01,
			BAY_TOP_COORDS + Vector2i(bay, 0),
		)
		tile_map.set_cell(
			Vector2i(column, BAY_TOP_ROW + 1),
			SOURCE_B_01,
			BAY_BOTTOM_COORDS + Vector2i(bay, 0),
		)


func get_tile_map_layer() -> TileMapLayer:
	return get_node_or_null("TileMapLayer") as TileMapLayer


func _is_border(x: int, y: int) -> bool:
	return x == 0 or y == 0 or x == MAP_SIZE.x - 1 or y == MAP_SIZE.y - 1
