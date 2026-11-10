# Delivery Roadmap

Status: Decided
Last updated: 2026-11-10

Festival: Tuesday 2026-11-24, confirmed with the organizers on 2026-10-21.
Internal festival-ready target: Wednesday 2026-11-18. The days from 2026-11-19
to 2026-11-23 are reserved for fixes, setup, and the final rehearsal; no new
features start after 2026-11-18.

Sprint 0 started on Monday 2026-10-19 and the festival is five weeks away. The
original seven-week plan does not fit, so the Minimum Festival Demo (MFD) is the
committed scope and the full MVP is a stretch goal. Cut scope by moving items to
Post-MVP, not by extending the MVP or the festival date.

## Calendar

| Sprint | Dates | Milestone and focus |
|--------|-------|---------------------|
| 0 | 2026-10-19 – 2026-10-25 | M0: decisions and spikes |
| 1 | 2026-10-26 – 2026-11-01 | M1: first half — missed; see Sprint 1 review |
| 2 | 2026-11-02 – 2026-11-08 | M1: single-tablet vertical slice (re-baselined 2026-11-03) |
| 3 | 2026-11-09 – 2026-11-15 | M2: four-player MFD acceptance |
| 4 | 2026-11-16 – 2026-11-22 | M3: festival-ready; checkpoint 2026-11-18 |
| Final setup | 2026-11-23 | Venue setup, final rehearsal, freeze |
| Festival | 2026-11-24 | Public event |

Each sprint ends with a demo and retrospective. Sprint 4 keeps the normal
one-week cadence, but its M3 checkpoint is Wednesday 2026-11-18; the rest of the
sprint is buffer for fixes and rehearsal.

## Progress Review — 2026-11-03

Sprint 0 closed partial with eight carry-over issues: the Go server slice,
decisions, CI, and project setup landed, but the four-tablet path, latency
measurement, and Android toolchain proof did not. Sprint 1 missed its M1
first-half goal; only the game scaffold landed (PR #31). M1 first half was not
achieved, and the controller and parking loop did not start.

D-33 re-baselines delivery: Sprint 2 is a single-tablet end-to-end vertical
slice, Sprint 3 is the four-player MFD acceptance run, and Sprint 4 is
festival-ready work. The MFD is committed; the full MVP remains a stretch. The
MFD (2026-11-15) and festival-ready (2026-11-18) dates do not change.

The detailed reviews, Sprint 2 plan, component status, and open risks are in
[`sprint-reviews.md`](sprint-reviews.md).

## Scope Tiers

| Tier | Goal | Must work |
|------|------|-----------|
| Minimum Festival Demo (MFD) — committed | A visitor can play one short round | 4 tablets connect, 4 cars move, one parking task, score, results, reconnect, setup/runbook |
| MVP — stretch | Complete first playable game | 5 tasks, precision/collision scoring, 3-minute rounds, attract/results, auto-recovery, stable 4-player rehearsal |
| Post-MVP | Extra value | MVP items not finished before the festival, racing, drift, extra task types, web host panel, richer art/audio |

## Milestones

### M0 — Decisions and Spikes (Sprint 0: 2026-10-19 – 2026-10-25)

Goal: prove the risky parts before building the game.

- Record decisions/ADRs for stack, Android 4.4.2, connection model, game rules,
  and latency targets.
- Set up GitHub Projects, issue/PR templates, DoR/DoD, branch protection, and
  the first CI skeleton.
- Collect tablet specs: model, API level, RAM, display, Wi-Fi bands, battery
  health, camera, sensors.
- Build a throwaway Kotlin app that sends WebSocket input to Go on all four
  tablets.
- Start Godot on the target laptop, connect it to Go, and render four moving
  cars on the TV.
- Measure input-to-display latency, Wi-Fi stability, charging, and readability.
- Confirm the Go supervisor can start/restart Godot.
- Write the spike results into the decision log/docs.

**Exit criteria:** four tablets move four cars through Go on the target
display; the measured path is understood; the Android toolchain is proven or a
fallback decision is recorded.

### M1 — Vertical Slice (Sprints 1–2: 2026-10-26 – 2026-11-08)

Goal: one complete round with one task type on one real tablet.

- Fixed map with spots, spawn points, static obstacles, and a perimeter loop.
- Cars for slots 1–4, arcade movement, environment collisions, no car-to-car
  collisions.
- One target spot per active player, exclusive assignment, parking detection.
- Time score + basic precision + collision count, live score, 3-minute timer.
- Placeholder HUD and results screen.
- Go session/slots/input validation; controller connect/reconnect; host start.
- Continuous playtest on one real tablet.

**Exit criteria:** one real tablet connects, drives, completes one parking task,
and shows a basic score and result on the target laptop; reconnect works; the
team can demo it.

Sprint 1 (2026-10-26 – 2026-11-01) delivered only the game scaffold. The
four-player M1 exit is now part of M2.

### M2 — Minimum Festival Demo (Sprint 3: 2026-11-09 – 2026-11-15)

Goal: a visitor can play one short round on the real hardware.

Committed MFD work:

- Four-player validation of the M1 slice on the real tablets and display.
- Four tablets connect and reclaim slots; four cars move.
- One parking task per player with exclusive assignment and parking detection.
- Time score plus basic precision and collision count; live score and 3-minute
  timer.
- Results screen and host start.
- Basic reconnect: a disconnected car stops and its slot is reserved.
- Setup/runbook draft and a playtest on the real hardware.

Stretch (only after the committed work is stable):

- Five tasks per player, task pool, auto-skip, finish bonus, tie-breakers.
- Full precision scoring and result ratings.
- Attract mode, pause/reset/kick.
- All-disconnected pause and game/server restart recovery.
- Controller task/score/time feedback and setup screen.

**Exit criteria:** four people play a full MFD round on the real tablets and
display; scores and reconnects work; the team can demo it. Stretch items are not
an exit criterion and move to Post-MVP if unfinished.

### M3 — Festival-Ready (Sprint 4: 2026-11-16 – 2026-11-22; target 2026-11-18)

Goal: the stand can run without a developer for the committed MFD.

- Festival launcher and packaging (Go + Godot + config + logs).
- Kiosk-like tablet setup: home launcher, immersive, keep-awake, boot start
  (where API 19 permits; document the fallback otherwise).
- Dedicated router/hotspot, static addressing, firewall, no internet dependency.
- Attract/queue flow, reset between groups, printed instructions.
- Core failure drills: tablet, game, and server; router and TV as time allows.
- 30–60 minute endurance test; spare hardware and power plan.
- Full rehearsal with people who have never played.
- Freeze on 2026-11-18; 2026-11-19–22 is only for fixes and rehearsal.

**Exit criteria:** a non-developer host can run three consecutive MFD groups
using the runbook; core recovery drills pass. The 2026-11-23 final setup
confirms a cold start from packed boxes.

### M4 — Post-MVP (After 2026-11-24)

Only after the festival, or after the committed MFD is stable and time remains
before the festival:

- MVP stretch items not completed before 2026-11-18.
- Racing mode on the perimeter loop.
- Drift mechanic.
- Extra task types.
- Web host panel.
- Anonymous event leaderboard, if privacy-safe.
- Art/audio polish.

## Sprint 0 Backlog (Historical)

M0 closed on 2026-11-03 with carry-over. Outcomes and owners are in
[`sprint-reviews.md`](sprint-reviews.md).

| Item | Area | Owner role | Output |
|------|------|------------|--------|
| S0-00 Confirm festival date and internal target | process | PM | Calendar, target date, scope tier |
| S0-01 Record decisions and ADRs | docs | Backend/PM | Decision log, ADRs |
| S0-02 Collect tablet specs and battery health | ops | QA/Controller | Hardware inventory |
| S0-03 Minimal Kotlin WebSocket app on all 4 tablets | controller | Controller/UI | APK + test notes |
| S0-04 Go WebSocket server with fake controller | server | Backend | Running server + tests |
| S0-05 Godot game scaffold connects to Go and renders four slot-keyed cars | game | Game | Scaffold + headless tests; target-laptop/TV check pending (#13) |
| S0-06 Measure latency and Wi-Fi stability | ops | Backend/QA | Measurements + verdict |
| S0-07 Supervisor starts/restarts Godot | server | Backend | Crash-recovery demo |
| S0-08 GitHub board, templates, DoR/DoD, CI skeleton | process | PM/Backend | Project setup |
| S0-09 Test plan and first hardware checklist | docs | QA | `testing.md` updates |

M0 closed on 2026-11-03 with carry-over; the original gate was not met.

## Sprint 2 Backlog (Current)

Sprint 2 goal: single-tablet end-to-end vertical slice (D-33). The full plan,
exit criteria, and risks are in [`sprint-reviews.md`](sprint-reviews.md).

| Issue | Work | Owner | Priority | Status |
|-------|------|-------|----------|--------|
| #10 | Tablet specs and battery health | ZofiaDan | P0 | Done (PR #70) |
| #11 | Kotlin WebSocket spike on one tablet | TracerDS | P0 | In Progress |
| #45 | Define the MFD gameplay baseline | Brzana | P0 | In Progress |
| #46 | One parking task with exclusive assignment and detection | Brzana | P0 | Ready |
| #47 | Score, timer, and results overlay | Brzana | P0 | Ready |
| #17 | Test plan and hardware checklist | ZofiaDan | P1 | Ready |
| #18 | D-14 choose controller input scheme | TracerDS | P0 | Ready |
| #25 | Queue backpressure and slow-client close | GreenDude5 | P1 | In Progress |
| #51 | Reconcile documentation and board drift | GreenDude5 | P1 | Ready |

## Epic Backlog

### E1 — Session and Network (Go)

- Session lifecycle and slots.
- Device-token connect/reconnect.
- Input validation, rate limiting, heartbeat.
- Host commands and authorization.
- Game process supervision and launcher.
- Structured logging and diagnostics.
- Integration tests with fake game/controllers.

### E2 — Controller (Kotlin, API 19)

- Project scaffold with minSdk 19; no Compose/OkHttp 4; avoid AndroidX unless
  API 19 compatible.
- WebSocket connect/reconnect.
- Touch controls and input loop.
- Task/score/time/state UI.
- Setup/diagnostics screen.
- Kiosk-like behavior: home, immersive, keep awake, boot start.
- APK build and install procedure.

### E3 — Game (Godot)

- Project scaffold and fullscreen display.
- Map, spots, obstacles, spawn points, perimeter loop.
- Car movement and environment collisions.
- Task assignment, target highlight, auto-skip.
- Parking detection and scoring.
- HUD, results, attract mode.
- Pause/reset/kick and reconnect states.
- Godot tests for pure logic.

### E4 — Festival Operations

- Hardware inventory and spares.
- Router/hotspot configuration.
- Tablet provisioning checklist.
- Launcher and recovery.
- Runbook and failure playbook.
- Queue/throughput and reset flow.
- Full rehearsal.

### E5 — Process and Documentation

- Decision log/ADRs maintained.
- DoR/DoD and templates.
- Weekly demo/retro notes.
- Test plan and hardware matrix.
- Risk register updates.
- AI-assistance disclosure in PRs.

## Risks

| Risk | Likelihood | Impact | Mitigation | Owner |
|------|------------|--------|------------|-------|
| Kotlin/AGP cannot target API 19 reliably | Medium | High | Sprint 0 spike; fallback to minimal Java app; last resort replace tablets | Controller/Backend |
| Tablets too slow or batteries fail | Medium | High | Test all four; keep plugged; identify spare/replacement options | QA/Backend |
| Wi-Fi instability at venue | High | High | Dedicated router/hotspot; 2.4 GHz; AP isolation off; rehearsal; plan B | Backend |
| No Android/game-dev experience | High | Medium | AI-assisted small increments; spikes; peer review; keep scope small | Team |
| Godot↔Go integration friction | Medium | Medium | Thin protocol; fake clients; vertical slice early | Backend/Game |
| Scope creep (racing, drift, extra quests) | High | High | Scope tiers; Post-MVP list; PM gate | PM |
| Five weeks vs seven-week plan | High | High | MFD committed, MVP stretch; weekly scope gate; cut scope, not dates; M3 target 2026-11-18 with 2026-11-19–23 buffer | PM |
| Sprint 0 spikes not proven by 2026-10-25 | High | High | Run S0-03, S0-04, S0-05, and S0-06 in parallel; narrow to one controller→Go→Godot path; daily check; if unproven, skip M1 scope and go straight to MFD | Backend/Game/Controller |
| Sprint 1 delivered only the game scaffold; the controller/game vertical slice is unproven | High | High | Single-tablet Sprint 2; cut four-player scope; daily check; MFD only | Game/Controller |
| Game delivery depends on one owner on the critical path | High | High | Baseline first; pair GreenDude5 or TracerDS on game logic; cut polish | Team |
| Unknown availability | High | Medium | One-week sprints; small items; measure throughput; cut scope | PM |
| Hardware/TV/laptop not available | Medium | High | Confirm owner and specs; test early; backup monitor/laptop | Backend/QA |
| AI-generated code quality | Medium | Medium | Review, tests, reproduce, no blind merges | Team |
| Personal data leaks into scores/logs | Low | High | No names; random tokens; truncate logs; clear between groups | PM/Backend |

## Dependencies and Gates

- M2 cannot start until one real tablet completes the M1 vertical slice by
  2026-11-08.
- M3 cannot start until the MFD acceptance criteria pass by 2026-11-15;
  operations preparation may start earlier in Sprint 3.
- M3 target is 2026-11-18. From 2026-11-19, no new features; the buffer is for
  fixes, setup, and rehearsal.
- The festival is 2026-11-24. If the MFD is not ready by 2026-11-15, cut to the
  smallest demo that fits the MFD must-work list; do not extend the MVP or the
  festival date.
- Any change to stack, transport, or game rules requires a new decision/ADR.

## Capacity Notes

The team has four people with unknown weekly availability. Three sprints remain
from Sprint 2. Record planned vs actual hours from Sprint 0 and use the observed
throughput to plan, not a guessed velocity. If throughput is below the plan, cut
MFD items, not the buffer. Protect one weekly demo slot even in a bad week; a
small demo keeps the project honest.

Observed throughput: Sprint 0 delivered the server slice and project setup;
Sprint 1 delivered only the game scaffold. Plan by observed throughput, not
optimism.
