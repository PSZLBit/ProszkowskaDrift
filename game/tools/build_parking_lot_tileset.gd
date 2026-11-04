extends SceneTree
## Builds the parking-lot TileSet from the committed placeholder atlases.
##
## The purchased "Large Parking Lot" textures stay out of Git (D-34). This
## script turns the placeholder sheets into one reusable TileSet with a 48x48
## grid. At runtime, game/scripts/parking_lot_art.gd swaps the placeholder
## textures for the purchased ones when they are installed locally.
##
## Run from the repository root:
##   godot --headless --path game --script res://tools/build_parking_lot_tileset.gd

const TILE_SIZE := Vector2i(48, 48)
const PLACEHOLDER_DIR := "res://assets/placeholders/large_parking_lot/"
const OUTPUT_PATH := "res://tilesets/parking_lot.tres"

const SHEETS := [
	{"name": "A2-1", "file": "a2_1.png"},
	{"name": "A2-2", "file": "a2_2.png"},
	{"name": "A4-walls", "file": "a4_walls.png"},
	{"name": "B-01", "file": "b_01.png"},
	{"name": "B-02", "file": "b_02.png"},
	{"name": "B-03", "file": "b_03.png"},
	{"name": "B-04", "file": "b_04.png"},
	{"name": "B-05", "file": "b_05.png"},
]


func _initialize() -> void:
	var result := _build()
	if result != OK:
		push_error("Failed to build parking-lot TileSet: error %d." % result)
		quit(1)
		return
	print("Parking-lot TileSet written to %s." % OUTPUT_PATH)
	quit(0)


func _build() -> Error:
	var tile_set := TileSet.new()
	tile_set.tile_size = TILE_SIZE
	tile_set.add_physics_layer()
	tile_set.set_physics_layer_collision_layer(0, 1)
	tile_set.set_physics_layer_collision_mask(0, 2)

	for index in SHEETS.size():
		var sheet: Dictionary = SHEETS[index]
		var texture := load(PLACEHOLDER_DIR + str(sheet["file"])) as Texture2D
		if texture == null:
			push_error("Missing placeholder texture: %s" % sheet["file"])
			return ERR_FILE_NOT_FOUND

		var source := TileSetAtlasSource.new()
		source.resource_name = str(sheet["name"])
		source.texture = texture
		source.texture_region_size = TILE_SIZE

		var columns := texture.get_width() / TILE_SIZE.x
		var rows := texture.get_height() / TILE_SIZE.y
		for y in rows:
			for x in columns:
				source.create_tile(Vector2i(x, y))

		tile_set.add_source(source, index)

	DirAccess.make_dir_recursive_absolute(ProjectSettings.globalize_path("res://tilesets"))
	return ResourceSaver.save(tile_set, OUTPUT_PATH)
