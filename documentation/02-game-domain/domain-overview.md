# Game Domain Overview

Status: Draft
Last updated: 2026-10-16

## Concept (TBD)

Brand-new real-time game, unrelated to the archived Flanki tournament model.
Genre, rules, win condition, and scoring are TBD (see D-13).
Do not implement game logic until this section reaches `Decided`.

## System Shape (Decided)

Three separate parts in one monorepo:

- `controller-android/` — Kotlin native app on Android tablets. One tablet per player, 1–4 players per session. Input set is TBD (see D-14).
- `server/` — runs on a PC connected to the big screen. Receives controller inputs, enforces session and game rules, drives the display. Server and game are separate so the game can be swapped out (see D-11).
- `game/` — swappable game module driven by the server. Concrete game is TBD.

## Session Lifecycle (Draft)

1. `Lobby open` — host opens a lobby on the server/PC. Lobby advertises a join code/QR for the local network.
2. `Joining` — 1–4 tablets join over the local network via code/QR. Server assigns player slots.
3. `Playing` — server routes inputs to the game, game state renders on the big screen (and optional per-controller feedback, TBD).
4. `Ended` — host ends the session or all players leave; slots are released.

Edge cases (disconnect, rejoin, host crash, late join, pause) are TBD (see D-15).
State transitions must be explicit in code once decided. No silent transitions.

## Real-Time Constraints (TBD)

Latency target, tick/update rate, and input protocol (transport TBD, see D-17)
will be defined here once decided. No invented numbers until then.
