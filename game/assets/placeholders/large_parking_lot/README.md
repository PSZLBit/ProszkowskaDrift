# Placeholder Parking Lot Atlases

These PNGs stand in for the purchased "Large Parking Lot" tileset, which is not
committed to Git (D-34). They are original project-created placeholders with the
same pixel dimensions and 48x48 grid as the purchased sheets, so atlas
coordinates are identical.

`game/tools/build_parking_lot_tileset.gd` generates
`game/tilesets/parking_lot.tres` from these files. At runtime,
`game/scripts/parking_lot_art.gd` swaps the placeholders for the purchased
textures when they are installed under `game/assets/Large Parking Lot/` (D-35).

Do not replace these files with the purchased art or commit the purchased
folder. Regenerate the TileSet after changing a placeholder:

```sh
godot --headless --path game --import
godot --headless --path game --script res://tools/build_parking_lot_tileset.gd
```
