# Game Design — Proszkowska Drift MVP

Status: Draft
Last updated: 2026-10-19

This document defines the first playable game. Core rules are agreed in D-19;
numeric values are initial tuning targets and should be changed only after a
playtest, with the reason recorded here.

## Design Goals

- Understandable within 10 seconds: drive, park, score.
- A complete round in about three minutes, suitable for a festival queue.
- Spectator-friendly: the whole map, all players, and the score are visible at
  once.
- Skill expression through speed, precision, and clean driving, not through
  complex controls.
- Small enough for a four-person student team with no prior game-dev experience.

## Round at a Glance

| Item | MVP value (initial) |
|---|---|
| Players | 1–4, independent |
| Round length | 180 s |
| Tasks per player | 5 |
| Parking spots on map | 24 |
| Task mode | Park in the assigned target spot |
| Car collisions | Static environment only; no car-to-car collisions |
| Display | Fixed full-map view, 16:9 reference resolution 1920×1080 |
| Results screen | About 15 s, then host starts the next round |

If fewer than four tablets are connected, the round starts with the available
players. Empty slots are allowed. A solo round is a valid time attack.

## Map

- One fixed, stylized top-down parking lot inspired by the campus area on
  Proszkowska Street. It is not an exact map of the real location.
- 24 marked parking spots with different difficulty: open spots, tighter spots,
  angled spots, and spots with a required facing arrow.
- Static obstacles: building walls, barriers, cones, planters, and parked
  non-player cars. No moving traffic in MVP.
- Lanes wide enough for casual players to pass each other. Cars may overlap
  visually because they do not collide; use distinct colors and outlines.
- A perimeter loop road around the lot. It exists so a future racing mode can
  reuse the map; it is not a race track in MVP.
- The whole map fits on screen. No camera scrolling in MVP.
- The map is data-driven enough to move spots and obstacles without rewriting
  game logic.

## Player and Car

- Each player has one car with a distinct color and a visible `P1`–`P4` label.
- Car movement is arcade-style, not a realistic vehicle simulation:
  - throttle and brake/reverse,
  - steering that becomes less effective at very low speed,
  - friction and a maximum speed,
  - no drift mechanic in MVP.
- The car collides with static environment geometry. A collision reduces speed
  and is counted for the current task. Cars do not collide with each other.
- A finished player's car remains on the map but cannot score again until the
  next round. It may be rendered semi-transparent and does not block other cars.

## Task Assignment

- At round start, each player receives the first of five target spots.
- A target spot is exclusive among active players: no two players are assigned
  the same spot at the same time.
- Spots are chosen from the 24-spot pool. A player does not receive the same spot
  twice in one round unless the pool is exhausted.
- A completed or skipped spot returns to the pool and can be assigned to another
  player.
- The target is highlighted in the player's color on the display and described
  on the controller by spot number, color, and a simple direction hint.
- A required-facing arrow means the car should end the task facing that
  direction for the full angle bonus. Facing the other way is allowed but scores
  lower; there is no separate reverse-parking rule in MVP.

## Movement and Collision Rules

- Input is the latest validated `steer` and `throttle` from the controller.
- If no input arrives for 500 ms, the car coasts to a stop; it does not keep the
  last throttle forever.
- Environment collisions are detected by the game's physics/collision layer.
  Each new collision while a task is active adds one collision count. Continuous
  contact does not count every frame.
- A collision never destroys the car and never ends the round.
- Car-to-car collision is disabled by collision layers. Visual overlap is
  acceptable.

## Parking Detection

A task is complete when all of the following are true:

1. All four corners of the car's rectangle are inside the target spot polygon.
2. The car's speed is below the "parked" threshold.
3. The conditions hold for 0.5 s continuously.

On completion, the game records:

- task completion time,
- center offset from the spot center,
- angle difference from the spot's preferred direction,
- number of collisions during the task.

If the task is not completed within 45 s, it is automatically skipped with zero
task points. The player receives the next task. This prevents a stuck player from
blocking the festival queue.

## Scoring (Initial Tuning)

Each task produces a score:

```text
time_score      = max(10, 100 - 3 * seconds_since_task_assigned)
precision_score = center_score + angle_score          # 0..40
collision_penalty = min(30, 10 * collisions)          # 0, 10, 20, 30
task_score      = max(0, time_score + precision_score - collision_penalty)
```

Precision components:

- `center_score`: 20 points when the car center is at the spot center, falling
  linearly to 0 at half the spot's shorter dimension.
- `angle_score`: 20 points within 10° of the preferred direction, falling
  linearly to 0 at 35° or more.
- A perfect park is fast, centered, aligned, and collision-free.

Bonuses:

- Completing all five tasks: `+100`.
- Remaining round time after finishing all tasks: `+5 per second`, capped at
  `+150`.
- Skipped task: 0 points for that task.

Round total is the sum of task scores plus bonuses. The maximum is roughly 950
points, but tuning may change this.

Tie-breakers, in order:

1. More completed tasks.
2. Fewer total collisions.
3. Shorter total time spent on completed tasks.

Display feedback after each task: `Perfect` (precision 35–40), `Great` (25–34),
`Good` (15–24), or `Parked` (0–14), followed by the points awarded.

## Round Flow

```text
Attract
  -> Ready        (host starts; tablets connected)
  -> Countdown    (3, 2, 1)
  -> Playing      (180 s)
  -> Results      (about 15 s)
  -> Attract or next Ready
```

- The host can start only when at least one player slot is connected.
- Pause freezes the timer, car movement, and scoring. Resume continues with a
  short countdown.
- Reset abandons the current round, clears scores, and returns to `Ready`.
- If all players finish their five tasks, the round ends early and results are
  shown.
- If the timer reaches zero, the round ends immediately; cars stop accepting
  input.
- The next round starts only after the host confirms.

## Display and HUD

- Fixed full-map view with a scoreboard showing player color, label, score,
  completed tasks, and current task status.
- Large round timer.
- Target-spot highlight in the player's color, with an on-map arrow or beacon so
  players can find it from a distance.
- Task result popups near the player's car and in the scoreboard.
- Clear "Paused", "Reconnecting", "Finished", and "Round over" states.
- High contrast and large shapes; readable from several metres away.
- The display is the only place that renders the game.

## Controller Feedback

The tablet is strictly a controller. It shows:

- connection state and assigned player color/label,
- current task (spot number/color and a short direction hint),
- current score and round time,
- round state (`Ready`, `Countdown`, `Playing`, `Paused`, `Finished`, `Results`),
- large touch controls,
- a reconnect overlay when the connection drops.

No map rendering, no camera, no game state simulation on the tablet.

## Attract and Results

- `Attract`: game title, one-line instruction, "Press ENTER to start", control
  hint, and optional anonymous best scores for the event.
- `Results`: winner, final scores, tasks completed, collisions, and a short
  "next round" prompt.
- No names, photos, or personal data. Players are identified by slot color and
  `P1`–`P4`.
- The host can clear scores between groups.

## Edge Cases

- One player: solo time attack; same rules.
- Two or three players: same rules; remaining slots idle.
- Player finishes early: waits; finish bonus already applied.
- Player disconnects: car stops, slot is reserved, reconnect resumes control.
- All players disconnect: pause after a 10 s grace period; host resumes or
  resets.
- Late join: allowed between rounds only; the new player starts the next round
  with zero score.
- Host pause during results: ignored; results screen has its own timer.
- Game or server crash: see D-24; the round is lost and the system returns to
  `Attract`.
- Two cars overlap: no physical response; render with outlines and optional
  transparency.

## Post-MVP (Not Implemented)

- Racing mode on the perimeter loop, with checkpoints and laps.
- Drift mechanic and skid effects.
- Additional task types: delivery/pickup, parallel parking, timed bonus spots,
  obstacle courses.
- Anonymous event leaderboard, if it remains free of personal data.
- A web host panel for a second operator device.
- Better art, audio, and particle effects.

These require a new decision or a new game-design section. They must not leak
into the MVP implementation.

## Tuning Table

| Parameter | Initial value | Change after |
|---|---|---|
| Round length | 180 s | First 4-player playtest |
| Tasks per player | 5 | First 4-player playtest |
| Task time decay | 100 → 10 over 30 s | Playtest |
| Auto-skip | 45 s | Playtest |
| Precision window | ±10° full, ±35° zero | Playtest |
| Collision penalty | −10 each, max −30 | Playtest |
| Finish bonus | +100 + 5/s, cap +150 | Playtest |
| Parked speed threshold | TBD in game code | Controller/game spike |
| Car max speed | TBD in game code | First map playtest |

## Open Questions

- Exact control scheme (D-14): buttons, analog slider, or virtual joystick.
- Exact map layout and the number of spots after the first sketch.
- Whether a task can be manually skipped by the player in MVP, or only
  auto-skipped.
- Whether finished cars should be hidden, ghosted, or parked in a finish area.
- Whether the display should show a minimap for target finding or only on-map
  beacons.
