class_name ParkingLotArt
extends RefCounted
## Swaps the committed placeholder atlases for the purchased "Large Parking Lot"
## textures when they are installed locally (D-34).
##
## The purchased pack stays out of Git. Placeholder sheets have the same pixel
## dimensions and 48x48 grid as the purchased sheets, so atlas coordinates stay
## valid after a swap. Sources whose purchased file is missing keep the
## placeholder texture, which keeps fresh clones and CI working.

## Local folder that holds the purchased sheets. It is ignored by Git.
const REAL_DIR := "res://assets/Large Parking Lot/"

## TileSet source name -> purchased file name.
const REAL_FILES := {
	"A2-1": "Tile_A2-1 (2).png",
	"A2-2": "Tile_A2-2.png",
	"A4-walls": "Auto-tile-A4-walls-3.png",
	"B-01": "tile-B-01.png",
	"B-02": "tile-B-02.png",
	"B-03": "tile-B-03.png",
	"B-04": "tile-B-04.png",
	"B-05": "tile-B-05.png",
}


## Replaces placeholder textures with purchased ones found under `base_dir`.
## Returns the number of atlas sources that were swapped. Sources without a
## matching installed file or with a different size keep their current texture.
static func apply(tile_set: TileSet, base_dir := REAL_DIR) -> int:
	if tile_set == null:
		return 0

	var swapped := 0
	for index in tile_set.get_source_count():
		var source_id := tile_set.get_source_id(index)
		var source := tile_set.get_source(source_id) as TileSetAtlasSource
		if source == null:
			continue

		var file_name: String = REAL_FILES.get(source.resource_name, "")
		if file_name.is_empty():
			continue

		var path := base_dir.path_join(file_name)
		if not ResourceLoader.exists(path):
			continue

		var texture := load(path) as Texture2D
		if texture == null:
			push_warning("Could not load purchased texture: %s" % path)
			continue

		var placeholder_size := Vector2.ZERO
		if source.texture != null:
			placeholder_size = source.texture.get_size()
		if placeholder_size != texture.get_size():
			push_warning(
				"Purchased texture %s is %s, expected %s; keeping the placeholder."
				% [path, texture.get_size(), placeholder_size]
			)
			continue

		source.texture = texture
		swapped += 1

	return swapped
