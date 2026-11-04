extends SceneTree
## Builds the parking-lot TileSet (D-34, D-35).
##
## The committed `parking_lot.tres` is built from the placeholder atlases and is
## gitignored along with the rest of `game/tilesets/`; run this script (or let
## CI run it) to (re)generate it. When the purchased sheets are installed, a
## `parking_lot.local.tres` is also written from them and preferred by the map.
##
## Run from the repository root:
##   godot --headless --path game --import
##   godot --headless --path game --script res://tools/build_parking_lot_tileset.gd

const ParkingLotTilesetBuilder := preload("res://scripts/parking_lot_tileset_builder.gd")

const PLACEHOLDER_DIR := "res://assets/placeholders/large_parking_lot/"
const PURCHASED_DIR := "res://assets/Large Parking Lot/"
const COMMITTED_OUTPUT := "res://tilesets/parking_lot.tres"
const LOCAL_OUTPUT := "res://tilesets/parking_lot.local.tres"


func _initialize() -> void:
	DirAccess.make_dir_recursive_absolute(ProjectSettings.globalize_path("res://tilesets"))

	var result := ParkingLotTilesetBuilder.build(
		PLACEHOLDER_DIR,
		ParkingLotTilesetBuilder.placeholder_files(),
		COMMITTED_OUTPUT,
	)
	if result != OK:
		push_error("Failed to build committed parking-lot TileSet: error %d." % result)
		quit(1)
		return
	print("Committed parking-lot TileSet written to %s." % COMMITTED_OUTPUT)

	if ParkingLotTilesetBuilder.purchased_available():
		result = ParkingLotTilesetBuilder.build(
			PURCHASED_DIR,
			ParkingLotTilesetBuilder.purchased_files(),
			LOCAL_OUTPUT,
		)
		if result != OK:
			push_error("Failed to build local parking-lot TileSet: error %d." % result)
			quit(1)
			return
		print("Local (purchased) parking-lot TileSet written to %s." % LOCAL_OUTPUT)
	else:
		print("Purchased sheets not found; skipped %s." % LOCAL_OUTPUT)

	quit(0)
