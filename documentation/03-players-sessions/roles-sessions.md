# Players, Sessions, and Permissions

Status: Draft
Last updated: 2026-10-16

There are no accounts. A play session is local and short-lived.

## Roles

- `Player` — one person with one Android tablet controller. Can join a lobby via code/QR, send game inputs, and leave.
- `Host` — the person operating the server PC connected to the screen. Can open/close the lobby, start/end the game, and remove a player slot.W

## Permission Matrix (Draft)

| Action                 |            Player             | Host |
| ---------------------- | :---------------------------: | :--: |
| Join lobby via code/QR |     yes (free slot only)      | n/a  |
| Send game input        | yes (own slot, while playing) |  no  |
| Open / close lobby     |              no               | yes  |
| Start / end game       |              no               | yes  |
| Remove a player slot   |              no               | yes  |

Every privileged action (slot assignment, kick, start/end) must be enforced
by the server. Never rely on controller UI visibility alone.

## Session Data

- Session holds only what is needed to play: player slots, connection handles,
  and game state. Content is TBD alongside the game concept.
- Session data is discarded when the session ends.
