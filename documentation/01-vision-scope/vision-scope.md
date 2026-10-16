# Vision and Scope

Status: Draft
Last updated: 2026-10-16

## Vision

A local real-time play system: players use Android tablets as controllers,
a PC server receives the inputs, drives a swappable game, and shows it
on a big screen everyone in the room can see.

## Context

One session hosts 1–4 players, each with one Android tablet (Kotlin native),
joined to a server PC over the local network via lobby code/QR.
The server and the game are separate so games can be swapped without
rewiring controllers, networking, or display.

## MVP Scope (Planned, Not Implemented)

- Host opens a lobby on the server PC; lobby shows a join code/QR.
- 1–4 Android tablets join over the local network and take player slots.
- Tablets send real-time inputs to the server; input set is TBD.
- Server routes inputs to the game and renders game output on the connected screen.
- Host can start/end the game and manage player slots.
- One swappable-game slot with a TBD first game.

## Non-Goals for MVP

- No accounts, no persistent profiles.
- No internet play; local network only.
- No iOS controllers.
- No fixed game design yet — game concept is TBD and blocks game logic.

## Open Questions (TBD)

- Game concept, rules, and win condition — see `02-game-domain/domain-overview.md`.
- Controller input set and feedback — TBD.
- Disconnect, rejoin, late join, and host-crash handling — TBD.
- Latency target and transport protocol — see `05-architecture-decisions/decision-log.md`.
