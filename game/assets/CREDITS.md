# Asset Credits

This file records the source and licence of project assets, including
third-party assets under `game/assets/` and first-party assets used by the
controller app and documentation. Update it whenever an asset is added or
replaced.

## Parking Lot Top-Down Pixel Art Tileset ("Large Parking Lot")

- Author: Cute SCKR — <https://comshadow.itch.io>
- Source: <https://comshadow.itch.io/parking-lot-top-down-pixel-art-tileset>
- Licence: purchased copy. The itch.io terms allow personal and commercial use
  in game projects, prohibit reselling or redistributing the assets as
  standalone resources, and state that attribution is appreciated but not
  required.
- Intended use: the tiles are used through `game/tilesets/parking_lot.tres`
  and the demo map `game/scenes/map/parking_lot.tscn`. Task integration is
  tracked by #46; the demo map is not final map geometry.
- AI disclosure: the itch.io page marks the pack as AI-assisted.
- Files: `game/assets/Large Parking Lot/*.png`, eight tiles. The folder is not
  committed to Git (D-34); each team member obtains the purchased copy and
  places it there locally. Godot regenerates the `.import` metadata on import.
- Repository policy: `/game/assets/Large Parking Lot/` is ignored by
  `.gitignore` because the purchase terms prohibit redistributing the assets as
  standalone resources. Do not remove the ignore rule or force-add the files.
- Purchase record: held by the project team. Do not commit receipts or personal
  data.
- Placeholder atlases: `game/assets/placeholders/large_parking_lot/` contains
  original project-created placeholder sheets with the same pixel dimensions
  and 48×48 grid as the purchased files. They are committed so the TileSet can be
  generated in CI and on fresh clones. `game/scripts/parking_lot_art.gd` swaps in
  the purchased textures at runtime when the local folder is present (D-35).
- Generated tilesets: `game/tilesets/` is gitignored. Running
  `game/tools/build_parking_lot_tileset.gd` writes `parking_lot.tres` from the
  placeholders and `parking_lot.local.tres` from the purchased sheets when they
  are installed. CI runs the generator before the tests.

## Controller App Logo and Banner

- Source: original assets created for this project by the repository owner for
  PR #79.
- Licence: project-owned. These files are not third-party assets and are not
  redistributed as standalone resources.
- AI disclosure: no AI tools were used to create these assets.
- Files: `controller-android/app/src/main/res/drawable/logo.png`,
  `controller-android/app/src/main/res/drawable/banner.png`,
  density-specific `mipmap-*/ic_launcher*.png`,
  `documentation/assets/banner.png`, `documentation/assets/logo.png`, and
  `documentation/img/favicon.ico` (derived from the logo).

## Pending provenance

These files predate this record and still need source and licence details; see
issue #35:

- `game/assets/2D_TOPDOWN_PIXELART_CARS.png`
- `game/assets/image_2024-03-02_131934629.png`
