# Decision Log

Status: Decided
Last updated: 2026-10-16

Register of all team decisions. Newest entries on top.
For larger decisions, create an ADR file in this folder using `_templates/adr.md`
and link it here.

| Date       | ID   | Decision                                                                                                                                  | Status  | Owner |
| ---------- | ---- | ----------------------------------------------------------------------------------------------------------------------------------------- | ------- | ----- |
| 2026-10-16 | D-17 | Server language, game engine, and controller-to-server transport protocol are not selected yet. Do not introduce them without a decision. | TBD     | Team  |
| 2026-10-16 | D-16 | Latency target and update/tick rate are TBD. Do not invent numbers in code or docs.                                                       | TBD     | Team  |
| 2026-10-16 | D-15 | Disconnect, rejoin, late join, and host-crash handling are TBD.                                                                           | TBD     | Team  |
| 2026-10-16 | D-14 | Controller input set and per-controller feedback are TBD.                                                                                 | TBD     | Team  |
| 2026-10-16 | D-13 | Game concept, rules, and win condition are TBD. Do not implement game logic until `02-game-domain/domain-overview.md` reaches `Decided`.  | TBD     | Team  |
| 2026-10-16 | D-12 | No accounts and no retained personal data. Old account/role/18+/GDPR-DOB model is archived. Sessions hold transient state only.           | Decided | Team  |
| 2026-10-16 | D-11 | Server and game are separate so the game can be swapped via a stable game interface.                                                      | Decided | Team  |
| 2026-10-16 | D-10 | One session hosts 1–4 players, one Android tablet per player.                                                                             | Decided | Team  |
| 2026-10-16 | D-09 | Local-network only (e.g. shared WiFi). Tablets join via lobby code/QR. No internet play for MVP.                                          | Decided | Team  |
| 2026-10-16 | D-08 | Controller app is Android-only, Kotlin native, in `controller-android/`.                                                                  | Decided | Team  |
| 2026-10-16 | D-07 | Monorepo layout: `server/`, `game/`, `controller-android/`, `documentation/`.                                                             | Decided | Team  |
| 2026-10-13 | D-05 | All project artifacts (docs, code, comments, commits) are in English.                                                                     | Decided | Team  |
| 2026-10-13 | D-04 | `documentation/` is the source of truth; agents use read-first / update-after protocol.                                                   | Decided | Team  |
| 2026-10-13 | D-01 | Tech stack, database, and hosting are not selected yet. Do not introduce them without a decision.                                         | TBD     | Team  |

## Rules

- A `TBD` row blocks implementation of that topic.
- Changing a `Decided` row requires a new dated entry, not silent editing.
- Link the ADR file in the Decision column when one exists.
