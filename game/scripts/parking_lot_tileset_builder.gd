class_name ParkingLotTilesetBuilder
extends RefCounted
## Shared logic for generating the parking-lot TileSet (D-34, D-35).
##
## Used by `game/tools/build_parking_lot_tileset.gd` and by the map/tests to
## regenerate the (gitignored) TileSet on demand when it is missing.

const TILE_SIZE := Vector2i(48, 48)

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


static func build(base_dir: String, file_map: Dictionary, output_path: String) -> Error:
	var tile_set := TileSet.new()
	tile_set.tile_size = TILE_SIZE
	tile_set.add_physics_layer()
	tile_set.set_physics_layer_collision_layer(0, 1)
	tile_set.set_physics_layer_collision_mask(0, 2)

	for index in SHEETS.size():
		var sheet: Dictionary = SHEETS[index]
		var name: String = sheet["name"]
		var file_name: String = file_map[name]
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

	DirAccess.make_dir_recursive_absolute(ProjectSettings.globalize_path("res://tilesets"))
	return ResourceSaver.save(tile_set, output_path)


static func placeholder_files() -> Dictionary:
	var files := {}
	for sheet in SHEETS:
		files[sheet["name"]] = sheet["placeholder"]
	return files


static func purchased_files() -> Dictionary:
	var files := {}
	for sheet in SHEETS:
		files[sheet["name"]] = sheet["purchased"]
	return files


static func purchased_available() -> bool:
	return ResourceLoader.exists("res://assets/Large Parking Lot/" + SHEETS[0]["purchased"])
