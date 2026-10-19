# Players, Sessions, and Permissions

Status: Decided
Last updated: 2026-10-19

There are no accounts. A play session is local and short-lived. Players are
identified by slot color and `P1`–`P4`; no names or personal profiles are stored.

## Roles

- `Player` — one person holding one of the four team-owned Android tablets.
  Can send input for their slot, see their task/score/time, and leave. A player
  cannot start, pause, reset, or kick.
- `Host` — the person operating the server laptop and the display. Starts,
  pauses, resumes, resets, and ends rounds; resets or kicks slots; can end the
  session. The host does not play in MVP.
- `Spectator` — watches the big screen. No controller, no interaction, no data.

## Permission Matrix

| Action | Player | Host |
|--------|:------:|:----:|
| Connect / reclaim own slot | yes (device token) | n/a |
| Send input for own slot | yes (while playing) | no |
| View task, score, round time | own slot only | all slots on display |
| Start / pause / resume / reset / end round | no | yes |
| Kick or reset a slot | no | yes |
| End session | no | yes |
| Start servers or game process | no | yes |

Every privileged action is enforced by the server. Controller UI visibility is
never used as authorization.

## Fixed Tablet Connection Model

- The four tablets are team hardware, not visitor-owned devices. They connect
  automatically to the configured server address (D-21).
- Each tablet stores a random device token in app storage. The server maps the
  token to a slot. The token is not a name, account, or personal profile.
- On reconnect, the tablet sends the same token and reclaims its slot. The host
  can reset a slot if a tablet is replaced or the token is lost.
- QR scanning or manual address entry exists only as a setup fallback. It is not
  part of the normal player flow.
- A tablet that connects for the first time while no slot is free is rejected
  and shown a "session full" state.

## Session and Round Data

- Session: slots, device tokens, connection handles, heartbeat state, current
  round phase, and scores.
- Game: map state, car state, tasks, timers, and collision counts.
- All state is in memory. It is discarded when the session ends or the server
  restarts.
- Optional event scores are anonymous and in-memory only; no names, photos, or
  persistent history (D-25).
- Logs must not contain personal data. Device tokens may appear as truncated
  identifiers in diagnostics.

## Disconnect and Reconnect

- A controller disconnect stops that player's car immediately. The slot remains
  reserved for the session.
- The controller reconnects automatically with backoff. It sends its device
  token and resumes control of the same slot. If the round is still running,
  control resumes; if not, the player waits for the next round.
- If all controllers disconnect, the round pauses after a 10 s grace period.
  The host can resume, reset, or end the session.
- Late join is allowed between rounds only. A new player starts the next round
  with zero score.
- If the game process crashes, the server restarts it and returns to `Attract`.
  The current round is lost. Tablets reconnect automatically.
- If the server crashes, the supervisor restarts server and game. The current
  round is lost. Tablets reconnect automatically.
- Host pause freezes the timer, input, and scoring. Resume uses a short
  countdown. Host reset abandons the round and returns to `Ready`.

## Host Controls (MVP)

- `Enter` — start round / confirm next round.
- `P` — pause or resume.
- `R` — reset current round to `Ready`.
- `1`–`4` — kick or disable the corresponding slot after confirmation.
- `Esc` — leave fullscreen or open the host quit prompt.

A web host panel is post-MVP. The host operates the same laptop as the display.

## Authorization and Privacy Boundaries

- The server validates the sender's device token, slot, round phase, input
  ranges, and rate. Inputs from an unassigned or kicked slot are rejected.
- The game accepts input only from the server, never directly from controllers.
- Controllers receive only their own task/score/time and public round state. Do
  not leak other players' private task assignments.
- No accounts, no names, no dates of birth, no persistent profiles, no photos.
- Session data is discarded at session end. If an anonymous event score is
  shown, it must not be linkable to a person.

## Session End

The host ends the session or closes the server. The server stops the game,
closes all controller connections, clears in-memory state, and writes only
non-personal diagnostic logs. Tablets return to the disconnected/ready state.
