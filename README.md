# Real-Time Play System

Local multiplayer system: Android tablets act as controllers, a PC server receives inputs, drives a swappable game, and renders it on a connected big screen.

One session hosts 1–4 players on the local network. The host opens a lobby on the server PC, players join via code/QR, and the server routes validated inputs to the active game. No accounts.

## Components

- `server/` — session ownership, input handling, game driving, big-screen display. (Not yet scaffolded; stack TBD.)
- `game/` — swappable game module behind the server game interface. (Concept TBD.)
- `controller-android/` — Kotlin native Android tablet controller. (Input set TBD.)
- `documentation/` — source of truth, including `archive/` for superseded Flanki Organizer docs.

## Planned Features

- Host opens a lobby with join code/QR on the server screen.
- 1–4 tablets join over the local network and take player slots.
- Tablets send real-time inputs; server validates slots and passes inputs to the game.
- Game output renders on the connected screen.
- Server/game separation allows swapping games without changing controllers or networking.

## Technology

To be decided with the team. See [`05-architecture-decisions/decision-log.md`](documentation/site/05-architecture-decisions/decision-log.md) — `TBD` rows block implementation.

## Getting Started

Development is at an early stage. Setup instructions will be added after the team chooses the technology stack per component.

## Project Status

Project is now in planning stage.

## License

To be determined.
