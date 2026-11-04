extends SceneTree
## Builds the parking-lot TileSet.
##
## Two outputs are produced:
##
##   * `game/tilesets/parking_lot.tres` is always built from the committed
##     placeholder atlases. It is the CI/fresh-clone baseline and is committed.
##   * `game/tilesets/parking_lot.local.tres` is built from the purchased
##     "Large Parking Lot" sheets when they are installed locally. It is
##     gitignored (D-34) and is preferred by the map and editor when present,
##     so the editor shows the real art without redistributing it.
##
## Run from the repository root:
##   godot --headless --path game --import
##   godot --headless --path game --script res://tools/build_parking_lot_tileset.gd

const TILE_SIZE := Vector2i(48, 48)
const PLACEHOLDER_DIR := "res://assets/placeholders/large_parking_lot/"
const PURCHASED_DIR := "res://assets/Large Parking Lot/"
const COMMITTED_OUTPUT := "res://tilesets/parking_lot.tres"
const LOCAL_OUTPUT := "res://tilesets/parking_lot.local.tres"

const SHEETS := [
	{"name": "A2-1", "placeholder": "a2_1.png", "purchased": "Tile_A2-1 (2).png"},
	{"name": "A2-2", "placeholder": "a2_2.png", "purchased": "Tile_A2-2.png"},
	{"name": "A4-walls", "placeholder": "a4_walls.png", "purchased": "Auto-tile-A4-walls-3.png"},
	{"name": "B-01", "placeholder": "b_01.png", "purchased": "tile-B-01.png"},
	{"name": "B-02", "placeholder": "b_02.png", "purchased": "tile-B-02.png"},
	{"name": "B-03", "placeholder": "b_03.png", "purchased": "tile-B-03.png"},
	{"name": "B-04", "placeholder": "b_04.png", "purchased": "tile-B-04.png"},
	{"name": "B-05", "placeholder": "b_05.png", "purchased": "tile-B-05.png"},
]


func _initialize() -> void:
	DirAccess.make_dir_recursive_absolute(ProjectSettings.globalize_path("res://tilesets"))

	var purchased_available := _purchased_available()

	var result := _build(PLACEHOLDER_DIR, _placeholder_files(), COMMITTED_OUTPUT)
	if result != OK:
		push_error("Failed to build committed parking-lot TileSet: error %d." % result)
		quit(1)
		return
	print("Committed parking-lot TileSet written to %s." % COMMITTED_OUTPUT)

	if purchased_available:
		result = _build(PURCHASED_DIR, _purchased_files(), LOCAL_OUTPUT)
		if result != OK:
			push_error("Failed to build local parking-lot TileSet: error %d." % result)
			quit(1)
			return
		print("Local (purchased) parking-lot TileSet written to %s." % LOCAL_OUTPUT)
	else:
		print("Purchased sheets not found; skipped %s." % LOCAL_OUTPUT)

	quit(0)


func _purchased_available() -> bool:
	return ResourceLoader.exists(PURCHASED_DIR + SHEETS[0]["purchased"])


func _placeholder_files() -> Dictionary:
	var files := {}
	for sheet in SHEETS:
		files[sheet["name"]] = sheet["placeholder"]
	return files


func _purchased_files() -> Dictionary:
	var files := {}
	for sheet in SHEETS:
		files[sheet["name"]] = sheet["purchased"]
	return files


func _build(base_dir: String, files: Dictionary, output_path: String) -> Error:
	var tile_set := TileSet.new()
	tile_set.tile_size = TILE_SIZE
	tile_set.add_physics_layer()
	tile_set.set_physics_layer_collision_layer(0, 1)
	tile_set.set_physics_layer_collision_mask(0, 2)

	for index in SHEETS.size():
		var sheet: Dictionary = SHEETS[index]
		var name: String = sheet["name"]
		var file_name: String = files[name]
		var texture := load(base_dir + file_name) as Texture2D
		if texture == null:
			push_error("Missing texture: %s" % (base_dir + file_name))
			return ERR_FILE_NOT_FOUND

		var source := TileSetAtlasSource.new()
		source.resource_name = name
		source.texture = texture
		source.texture_region_size = TILE_SIZE

		var columns := texture.get_width() / TILE_SIZE.x
		var rows := texture.get_height() / TILE_SIZE.y
		for y in rows:
			for x in columns:
				source.create_tile(Vector2i(x, y))

		tile_set.add_source(source, index)

	return ResourceSaver.save(tile_set, output_path)
