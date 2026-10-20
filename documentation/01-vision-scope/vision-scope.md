# Vision and Scope

Status: Decided
Last updated: 2026-10-19

## Vision

Proszkowska Drift (working title) is a local multiplayer arcade parking game.
One to four players use old Android tablets as controllers while the game runs
on a laptop and is shown on a TV or monitor. Each round is a short time attack:
players drive around a stylized campus parking lot on Proszkowska Street,
complete a sequence of parking tasks, and earn points for speed, precision, and
clean driving.

The game is built for a student team and a science-club festival stand. It must
be easy to understand in seconds, playable in short rounds, readable to
spectators, and independent of internet access or accounts.

## Context

- The team has four identical Samsung tablets running Android 4.4.2 (API 19).
  They are owned by the team and can be provisioned freely, but they are old and
  impose hard compatibility limits.
- The game runs on a laptop connected to a TV or monitor. The display shows the
  whole map at once.
- Tablets are controllers only. They send input and show simple feedback; they
  do not render the game.
- The planned stack is Go for the session server, Godot 4 for the game and
  display, and a minimal Kotlin Android controller (`05-architecture-decisions/adr-0001-mvp-architecture.md`).
- No accounts, no personal profiles, no internet play. Session data is
  transient.
- "Drift" and a racing mode are working-title ideas. They are post-MVP and must
  not delay the parking core (`02-game-domain/game-design.md`).

## MVP Scope (Planned, Not Implemented)

- One fixed, stylized top-down map based on the campus parking area on
  Proszkowska Street.
- 1–4 players per session, one tablet per player. Players complete tasks
  independently and can see each other on the shared map.
- Three-minute rounds with a shared timer. Each player receives a sequence of
  five parking tasks.
- Arcade car movement with collisions against the static environment only. Cars
  do not collide with each other.
- Scoring for task completion time, parking precision, and collisions. Results
  screen after every round.
- Tablets show controls, current task, score, round time, and connection state.
- Server owns session, slots, input validation, reconnects, and host commands.
- Host can start, pause, resume, reset, and end a round, and can kick or disable
  a slot.
- Fixed tablets connect automatically to the configured server and reclaim their
  slot after a reconnect. QR or manual address entry is only a setup fallback.
- Attract mode between rounds and a simple results screen.
- A festival runbook: network, kiosk, power, spare hardware, failure recovery.

## Success Criteria

- `Minimum Festival Demo` (MFD): four tablets connect, four cars move on the
  shared map, one parking task works, scores are visible, reconnect works, and
  the full setup starts in under 15 minutes.
- `MVP`: five tasks per player, precision scoring, three-minute rounds, results,
  attract mode, automatic recovery from game/server restarts, and a stable
  30-minute festival rehearsal with four players.
- Process: weekly demo, documented decisions, reviewed pull requests, CI green,
  and focused tests for game rules and networking.
- Performance target: 60 FPS display, 20–30 Hz controller input, and no more than
  100 ms from input to visible response (D-23).

## Non-Goals for MVP

- Internet play, accounts, persistent profiles, personal data, player names.
- iOS controllers, phones as controllers, bring-your-own devices.
- 3D graphics, realistic vehicle physics, drifting as a core mechanic.
- Car-to-car collisions or PvP collisions.
- Map editor, multiple maps, user-generated content.
- Racing mode, extra quest types, delivery mechanics, police/chase mechanics.
- High-fidelity art, licensed assets, music production.
- Public leaderboards with names or photos.

## Open Questions

- Controller input scheme (D-14) — decide by playtesting on the real tablets.
- Exact map layout, number of spots, and obstacle placement.
- Festival date, expected attendance, and available setup time.
- Laptop and TV/monitor specifications.
- Final scoring and round-length tuning after playtests.
- Whether to show anonymous best scores for the event (D-25 allows only
  anonymous, in-memory scores).
