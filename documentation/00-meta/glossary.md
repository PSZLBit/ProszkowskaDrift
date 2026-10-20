# Glossary

Status: Draft
Last updated: 2026-10-16

Single definition place for domain terms. Use these spellings in code, docs, and UI.

| Term | Definition | Notes |
|------|------------|-------|
| System | The local real-time play setup: controllers + server + game + display. | — |
| Controller | Android tablet app (Kotlin native) used as a game input device. | One per player, 1–4 per session. |
| Player | Person holding a controller in a session. | No account. |
| Host | Person operating the server PC. | Opens lobby, manages slots, starts/ends game. |
| Server | PC program that owns the session, receives inputs, drives the game, renders the display. | Runs on the screen-connected PC; separate from the game. |
| Game | Swappable module driven by the server through a stable interface. | Concrete game is TBD. |
| Display / Screen | Big screen connected to the server PC showing game output. | — |
| Session | One local play instance from lobby open to end. | Holds only transient slot/connection/game state. |
| Lobby | Joinable session state advertising a join code/QR on the local network. | — |
| Slot | One of the 1–4 player places in a session. | Assigned by the server. |
| Input event | Real-time message from a controller to the server. | Exact set is TBD. |
| Local network | Same-site network (e.g. WiFi) controllers and server share. | No internet play for MVP. |
| Disconnect | Loss of a controller connection; rejoin behavior is TBD. | See `02-game-domain/domain-overview.md`. |

Archived Flanki tournament terms (competition, bracket, seeding, bye, walkover,
competitor, organizer, administrator, WCA, GDPR date-of-birth handling) are
superseded — see `archive/README.md`.

## Rules

- Add a new term here before using it in other docs or code.
- Do not redefine a term in another file — link here instead.
