# Game Design — Proszkowska Drift MVP

Status: Draft
Last updated: 2026-11-04

The MFD subset is baselined in the "MFD Baseline" section below (D-35,
2026-11-04). The full GDD remains issue #32 and is deferred to the MVP
stretch/Post-MVP; the rest of this document stays Draft until those changes
land.

This document defines the first playable game. Core rules are agreed in D-19;
numeric values are initial tuning targets and should be changed only after a
playtest, with the reason recorded here. Sections after the MFD Baseline
describe the MVP/full game; where they differ, the MFD Baseline wins for the
MFD.

## Design Goals

- Understandable within 10 seconds: drive, park, score.
- A complete round in about three minutes, suitable for a festival queue.
- Spectator-friendly: the whole map, all players, and the score are visible at
  once.
- Skill expression through speed, precision, and clean driving, not through
  complex controls.
- Small enough for a four-person student team with no prior game-dev experience.

## MFD Baseline (Decided 2026-11-04)

Status: Decided for the Minimum Festival Demo (MFD). This section is the single
implementation source for #46 and #47 and is recorded as D-35. The rest of this
document remains Draft for the full GDD (#32); where an MVP section differs,
this section wins for the MFD.

Rules below are scoped as **MFD**, **MVP**, or **Post-MVP**. Subsection headings
carry the tier, and the table below summarizes the boundaries:

- **MFD** — required for the festival demo; #46 and #47 must implement it.
- **MVP** — part of the first complete game; not required for the MFD.
- **Post-MVP** — explicitly outside the first playable game.

### Scope and Tier Labels

| Area | MFD | MVP | Post-MVP |
|---|---|---|---|
| Players | 1–4 | 1–4 | — |
| Tasks per player | 1 | 5 | more task types |
| Round length | 180 s; ends early when all connected players finish | 180 s | — |
| Target spots | 4 (`A-01`–`A-04`) | 24 | more maps |
| Auto-skip | No | 45 s | — |
| Finish bonus | No | +100 +5/s, cap +150 | — |
| Scoring | time + basic precision + collisions | full precision + ratings | — |
| Controller task/score feedback | No; the display is the only HUD | Yes | — |
| Countdown / attract loop / host pause-reset-kick flows | No | Yes | — |
| Results | host starts the next round | about 15 s, then host | — |
| Racing / drift | No | No | Yes |

D-19 remains the MVP rule (five tasks, 180 s, time/precision/collision
scoring). The MFD is a deliberate subset under D-26, D-31, and D-33; this
baseline does not change D-19.

### MFD Map v1

The MFD uses one fixed map, version `proszkowska-mfd-1`. It is the first
revision of the single MVP map; the MVP adds spots and obstacles to the same
layout later, it does not add a second map. Coordinates below are viewport
pixels with the origin at the top-left. The 48 px art grid is a rendering
detail; detection uses the rectangles in the tables.

| Element | MFD value |
|---|---|
| Viewport | 1920×1080, fixed full-map view, no camera scrolling |
| Art grid | 48 px; 40×22 tiles centered vertically (12 px top and bottom) |
| Map version | `proszkowska-mfd-1` |
| Boundary | 20 px collision walls at the viewport edges; visual kerb |
| Perimeter road | 96 px wide drivable ring inside the kerb; future racing loop, no racing logic |
| Inner lot | x 116–1804, y 116–964 |
| Parking bays | 4, 96×144 px, labelled and centered as in the table below |
| Spawns | 4, on the bottom road as in the table below |
| Static obstacles | two 96×96 planters at `(240, 540)` and `(1680, 540)`; boundary walls |
| Target pool | 4 spots in the MFD; the MVP adds 20 more to the same map |

Parking bays:

| Spot | Center (px) | Size (px) | Preferred facing | Hint |
|---|---|---|---|---|
| `A-01` | `(384, 540)` | `96×144` | north (`-90°`) | `north` |
| `A-02` | `(768, 540)` | `96×144` | north (`-90°`) | `north` |
| `A-03` | `(1152, 540)` | `96×144` | north (`-90°`) | `north` |
| `A-04` | `(1536, 540)` | `96×144` | north (`-90°`) | `north` |

Spawns:

| Spawn | Position (px) | Facing | Slot |
|---|---|---|---|
| `S1` | `(672, 1000)` | north (`-90°`) | P1 |
| `S2` | `(864, 1000)` | north (`-90°`) | P2 |
| `S3` | `(1056, 1000)` | north (`-90°`) | P3 |
| `S4` | `(1248, 1000)` | north (`-90°`) | P4 |

Map rules:

- The whole map is visible at once; there is no camera scroll.
- The ring road and the inner lot are both drivable; there is no collision
  boundary between them.
- Cars collide with the boundary walls and the two planters only. Car-to-car
  collisions remain disabled.
- The bay art may use the tileset or simple painted shapes; the detection
  rectangle is authoritative.
- Map data (spot centers, sizes, facings, spawns, and obstacle rectangles) is
  data-driven. Task and scoring logic must not hard-code coordinates.
- CI and fresh clones use the committed placeholder sheets; the purchased
  tileset is installed locally only (D-34).
- The map keeps the perimeter loop required by the M1 map and FR-G10, but no
  racing logic.

### Task Assignment (MFD)

- At round start (the first `playing` frame), every connected slot receives one
  task. Late join is between rounds (D-24), so no task is assigned mid-round.
- The accepted `host_command start` carries the server's new `roundId`; the
  game uses that value rather than a local counter. The MFD implementation must
  add `roundId` to `host_command` (it is additive in protocol version 1).
- Active slots are processed in ascending slot order.
- The starting spot rotates per round: `start = (roundId - 1) mod 4`; the i-th
  active slot gets spot `(start + i) mod 4`. Assignment is deterministic and
  fair across rounds.
- A target spot is exclusive among active players; no two active players share
  a spot.
- A disconnected slot keeps its task and spot reserved. Reconnecting resumes the
  same task. A kicked slot releases its spot; the MFD does not reassign that
  spot mid-round.
- Task fields:
  - `taskId = roundId * 10 + (spotIndex + 1)`, where `spotIndex` is 0–3.
  - `spotLabel` is `A-01`–`A-04`.
  - `spotColor` is the player's slot color.
  - `hint` is the cardinal direction from the spawn to the spot (`north` for
    all MFD spots).
  - `deadlineMs` is the remaining round time at assignment (180000 at the
    start).
- There is no auto-skip and no manual skip in the MFD; the round timer ends the
  task.

### Parking Detection (MFD)

- The car footprint is its collision rectangle: 73.75×32.5 px. Transform its
  four corners by the car's position and rotation.
- A task completes when all of the following hold for 0.5 s continuously:
  1. all four corners are inside the target spot rectangle (the boundary counts
     as inside);
  2. the car's speed is at most 40 px/s.
- Any failed frame resets the hold timer to zero.
- Facing is not a completion gate. It only affects `angle_score`; a car facing
  the opposite way can still complete the task.
- Completion state machine: `assigned` → `completed`. Completion is recorded
  exactly once. A completed task cannot score again if the car leaves and
  re-enters the spot.
- On completion, record:
  - completion time in milliseconds from round start;
  - center offset: Euclidean distance from the car center to the spot center;
  - angle difference: absolute smallest difference between the car heading and
    the spot's preferred facing, 0–180°;
  - collisions during the task.
- Collisions: one count per contact episode with the static environment. A
  contact that persists across frames counts once; separation and re-contact
  count again. Collisions before assignment and after completion are not
  counted for the task.
- The car keeps the existing collision behavior: it is never destroyed, never
  ends the round, and its speed is reduced on contact.
- A disconnected car stops (input timeout or `player_left`); the task stays
  assigned. On reconnect, the same task and timer continue.
- No completion is detected while paused or in results.

### Scoring (MFD)

The MFD uses the existing time, precision, and collision formula for one task.
No finish bonus and no remaining-time bonus are awarded.

```text
elapsed_ms        = completion_time_ms - assignment_time_ms
time_score        = max(10, 100 - (3 * elapsed_ms) / 1000)          # integer division
center_score      = max(0, 20 - floor(20 * center_offset_px / 48))  # 48 = half of 96
angle_score       = 20                                if angle <= 10°
                    0                                 if angle >= 35°
                    floor(20 * (35 - angle) / 25)      otherwise
precision_score   = center_score + angle_score                      # 0..40
collision_penalty = min(30, 10 * collisions)                        # 0, 10, 20, 30
task_score        = max(0, time_score + precision_score - collision_penalty)
round_score       = task_score
```

- `center_offset_px` is the Euclidean distance between the car center and the
  spot center.
- `angle` is the absolute smallest difference between the car heading and the
  spot's preferred facing, in degrees (0–180).
- `48` is half the bay's shorter side (96 / 2).
- The perfect score is 140: 100 time + 40 precision and no collisions at zero
  elapsed time.
- A non-completing player scores 0 for the round.
- The existing rating mapping may be computed for `task_result.rating`; the MFD
  HUD and results do not display ratings (MVP).

Ranking and winner:

1. More completed tasks (0 or 1 in the MFD).
2. Higher score.
3. Fewer collisions.
4. Shorter completion time.
5. Lower slot number.

For the completion-time tie-breaker, a non-completing player counts as slower
than any completing player; between two non-completing players, use the lower
slot number. The winner is the first player in this order if at least one
player completed a task; otherwise the display shows `Winner: none`.

### Round Flow (MFD)

```text
Attract or Ready
  -> host Enter -> Playing (180 s)
  -> Results
  -> host Enter -> Playing ...
```

- `Attract` or `Ready`: the map and scoreboard are visible; the display shows
  `Press Enter to start`. The MFD has no animated attract loop; the server's
  initial `attract` phase uses the same static screen.
- Host Enter sends `host_request start` from either phase; the server validates
  it and returns `host_command start` with the new `roundId`. The server remains
  the phase authority.
- The MFD has no countdown phase. The round starts on the first `playing` frame;
  the game assigns tasks and starts the 180 s timer.
- `Playing`: validated input is applied, detection runs, and the timer counts
  down.
- The round ends when the timer reaches 0, or when at least one player is
  connected and every connected player has completed their task. A disconnected
  slot does not block the early end.
- `Results`: the results panel stays visible until the host starts the next
  round. Host Enter sends `host_request start`; the server accepts a start from
  `results` and returns `host_command start` with the next `roundId`. The game
  resets scores and tasks, then starts the next round.
- A single disconnect: the car stops, the slot and task stay reserved, and
  reconnect resumes control. All-disconnected pause follows D-24; the MFD
  acceptance check covers a single disconnect, not the all-disconnected pause.
- Countdown, an animated attract mode, and the full host pause/resume/reset/kick
  flows are MVP and are not MFD acceptance. The game must still apply the
  matching `host_command` without corrupting the round: `pause` freezes input
  and the timer, `resume` continues, `reset` returns to `Ready`, and `kick`
  clears the slot.
- Crash recovery follows D-24 and is festival-ready (M3) work, not MFD
  acceptance.

### HUD and Results (MFD)

Layout:

| Element | Content |
|---|---|
| Timer | top center, `MM:SS`, at least 48 px, high contrast |
| Scoreboard | top right, one row per slot: color chip, `P1`–`P4`, score, current spot/status, hit count |
| Target | filled translucent rectangle in the player's color, 4 px border, `A-01`–`A-04` label |
| State line | bottom center |
| Results panel | centered, about 600×480, title, winner, one row per active player, host prompt |

- The whole map is visible; there is no minimap.
- The display is the only renderer. The MFD tablet shows connection state and
  controls only; task/score/time feedback is MVP.
- Use the slot colors and labels from the protocol (`P1` red, `P2` blue, `P3`
  green, `P4` yellow).
- No names or personal data (D-25).

Exact copy:

| State | Copy |
|---|---|
| Attract / Ready | `Press Enter to start` |
| Playing | `Drive to your spot` |
| Player completes | `P1 parked!` for about 2 s |
| Player disconnects | `P1 reconnecting` |
| Round end | `Round over` |
| Winner | `Winner: P2` or `Winner: none` |
| Prompt | `Press Enter to start the next round` |

Scoreboard rows (examples):

```text
P1   120   Parked        1 hit
P2     0   A-02          0 hits
P3     0   Reconnecting  2 hits
P4     0   No park       3 hits
```

Results:

```text
Round over
Winner: P2

1. P2   138   Parked   0 hits   12.4 s
2. P1   112   Parked   1 hit    18.9 s
3. P3     0   No park  3 hits   -
4. P4     0   No park  0 hits   -

Press Enter to start the next round
```

- Rows are sorted by the ranking above; only active players are listed.
- `hits` is the UI word for collisions; use `1 hit` and `0 hits`.
- Completion time is one decimal place in seconds; `-` for a player who did not
  complete.
- The results panel does not auto-hide in the MFD; the host starts the next
  round. The MVP may add a ~15 s auto-return.
- Minimum text sizes: timer 48 px, score rows 28 px, results title 48 px, and
  results rows 28 px at 1920×1080. Verify readability on the target TV (#47
  manual check).

### Protocol and Config (MFD)

- `config` for the MFD: `roundDurationMs = 180000`, `tasksPerPlayer = 1`, and
  `maxPlayers = 4`.
- The server currently defaults `tasksPerPlayer` to 5. The MFD implementation
  (#46) must change the default and its tests to 1; the MVP restores 5 when the
  five-task rules land. The game must not ignore the config value.
- `game_ready.mapVersion` is `proszkowska-mfd-1`.
- `host_command start` carries the server's new `roundId`; the game uses it for
  task IDs, spot rotation, and `round_result`. This is an additive protocol
  field for #46.
- The game sends `task_result` on completion and `round_result` when the round
  ends, using the payload shapes in `06-development/protocol.md`. Sending the
  task to controllers (`player_feedback` from the game, then `task` from the
  server) is MVP controller feedback and is not required for the MFD.
- Task data fields: `taskId`, `spotLabel`, `spotColor`, `hint`, and
  `deadlineMs`; the display uses them directly.
- `task_result` fields: `taskId`, `points`, `timeScore`, `precisionScore`,
  `collisionPenalty`, `rating`, and `totalScore`. `rating` may use the existing
  precision mapping; the MFD does not display it.
- `round_result` fields: `roundId`, `winnerSlot`, and one score per active slot
  with `slot`, `score`, `tasks` (0 or 1), and `collisions`.
- `player_feedback` and controller task/score/time display are MVP, not MFD.
- The game sends `phase` with `timeLeftMs` while playing and `results` when the
  round ends; the server remains the phase authority.

### Tier Boundaries and Open Items

MFD rules that must not expand without a new decision:

- one task, four spots, no auto-skip, and no finish bonus;
- display-only HUD, with no controller task feedback;
- host start, with no countdown, animated attract loop, or host
  pause/reset/kick flow;
- single-disconnect handling; all-disconnected pause and crash recovery are
  MVP/M3 work.

MVP work (not MFD):

- five tasks and the 24-spot pool;
- auto-skip, finish bonus, full ratings, and final tuning;
- controller task/score/time feedback;
- countdown, animated attract mode, full pause/resume/reset/kick flows,
  all-disconnected pause, and crash recovery;
- about 15 s results auto-return;
- the final map art and the full obstacle layout.

Post-MVP:

- racing on the perimeter loop, drift, extra task types, anonymous leaderboard,
  web host panel, and richer art/audio.

Explicit open items (no silent TBDs):

- **D-14 / #18** controller input scheme remains TBD. This baseline uses only
  normalized `steer` and `throttle`; it does not choose buttons, a slider, or a
  virtual joystick.
- **#32** full GDD and the MVP map expansion to 24 spots. The MFD map is the
  first revision of the single fixed map, not a second map.
- Final tuning after playtests. Every number in this section is an initial
  Decided value; a change after a playtest is recorded in this document with
  the date and reason, and the Tuning Table is updated. No silent edits.
- **#48** production controller feedback is MVP; the MFD tablet shows controls
  and connection state only.

Dependencies: D-19, D-26, D-31, D-33, and D-34; #18, #32, #46, and #47; the
protocol in `06-development/protocol.md` and the requirements in
`04-product/requirements.md`.

## Round at a Glance (MVP)

The MFD overrides this table: one task per player, four target spots, no
auto-skip or finish bonus, and host-started results. See the MFD Baseline above.

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
- If no input arrives for 500 ms, the car coasts to a stop; it does not keep
  applying the last throttle command.
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

Each task produces a score (illustrative pseudocode, not executable):

```python
time_score        = max(10, 100 - 3 * seconds_since_task_assigned)
precision_score   = center_score + angle_score          # 0..40
collision_penalty = min(30, 10 * collisions)            # 0, 10, 20, 30
task_score        = max(0, time_score + precision_score - collision_penalty)
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
| Round length | 180 s (MFD and MVP) | First 4-player playtest |
| Tasks per player | 1 (MFD), 5 (MVP) | First 4-player playtest |
| MFD target spots | 4 (`A-01`–`A-04`) | First MFD playtest |
| MFD parking bay | 96×144 px | First MFD playtest |
| MFD task hold | 0.5 s | First MFD playtest |
| Task time decay | 100 → 10 over 30 s | Playtest |
| Auto-skip | 45 s (MVP) | Playtest |
| Precision window | ±10° full, ±35° zero | Playtest |
| Collision penalty | −10 each, max −30 | Playtest |
| Finish bonus | +100 + 5/s, cap +150 (MVP) | Playtest |
| Parked speed threshold | 40 px/s | First MFD playtest |
| Car max speed | 400 px/s | First map playtest |

## Open Questions

- Exact control scheme (D-14, #18): buttons, analog slider, or virtual
  joystick. The MFD Baseline uses only normalized `steer` and `throttle`.
- MVP map expansion to 24 spots and the full obstacle layout (#32). The MFD map
  is fixed at four spots in the MFD Baseline.
- Final tuning values after the first playtests; the initial values are in the
  MFD Baseline and the Tuning Table.
- Whether a task can be manually skipped by the player in MVP, or only
  auto-skipped.
- Whether finished cars should be hidden, ghosted, or parked in a finish area.
- Whether the display should show a minimap for target finding or only on-map
  beacons.
