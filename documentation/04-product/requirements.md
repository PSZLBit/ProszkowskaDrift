# Product Requirements

Status: Draft
Last updated: 2026-10-16

All items below are planned and not implemented unless marked otherwise.
Old Flanki account/competition/bracket requirements are archived —
see `archive/README.md`.

## Functional Requirements

Controller (`controller-android/`, Kotlin native, Android only):

1. `FR-C01` Lobby join: a tablet can discover/join a lobby over the local network via code/QR and take a free player slot (1–4 slots).
2. `FR-C02` Input send: a joined tablet can send real-time game inputs to the server. Exact input set is TBD.
3. `FR-C03` Leave/rejoin: a tablet can leave cleanly; rejoin behavior after disconnect is TBD.

Server (`server/`, runs on screen-connected PC):

4. `FR-S01` Lobby: the host can open/close a lobby; the lobby presents a join code/QR on the screen.
5. `FR-S02` Slots: the server assigns one slot per controller (max 4), rejects joins when full, and lets the host remove slots.
6. `FR-S03` Input routing: the server receives controller inputs, validates the sender slot, and passes them to the active game. Invalid senders are rejected server-side.
7. `FR-S04` Display: the server renders game output on the connected screen.
8. `FR-S05` Game swap: the server drives the game through a stable game interface so games can be swapped without changing controller or networking code. Interface is TBD until the stack is chosen.

Game (`game/`, swappable):

9. `FR-G01` First game: one playable game behind the server game interface. Game concept and rules are TBD — no game logic until `02-game-domain/domain-overview.md` reaches `Decided`.

## Non-Functional Requirements

- `NFR-01` Server-authoritative: every privileged action (slots, kick, start/end, input validity) is enforced by the server, not the controller UI.
- `NFR-02` Local-network operation; no accounts and no retained personal data.
- `NFR-03` English-only UI and content for MVP.
- `NFR-04` Latency target and update rate: TBD.
- `NFR-05` Disconnect/rejoin and host-crash behavior: TBD.

## Out of Scope for MVP

Internet play, iOS controllers, accounts/profiles, multiple simultaneous games.
See `01-vision-scope/vision-scope.md`.
