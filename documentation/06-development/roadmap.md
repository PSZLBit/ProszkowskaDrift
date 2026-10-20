# Delivery Roadmap

Status: Draft
Last updated: 2026-10-19

Dates are relative because the festival date and team availability are not fixed
yet. Once the festival date is known, place the milestones on the calendar and
cut scope by moving items to Post-MVP, not by extending the MVP.

## Scope Tiers

| Tier | Goal | Must work |
|------|------|-----------|
| Minimum Festival Demo (MFD) | A visitor can play one short round | 4 tablets connect, 4 cars move, one parking task, score, results, reconnect, setup/runbook |
| MVP | Complete first playable game | 5 tasks, precision/collision scoring, 3-minute rounds, attract/results, auto-recovery, stable 4-player rehearsal |
| Post-MVP | Extra value | Racing, drift, extra task types, web host panel, richer art/audio |

## Milestones

### M0 — Decisions and Spikes (Week 1)

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
  rectangles on the TV.
- Measure input-to-display latency, Wi-Fi stability, charging, and readability.
- Confirm the Go supervisor can start/restart Godot.
- Write the spike results into the decision log/docs.

**Exit criteria:** four tablets move four rectangles through Go on the target
display; the measured path is understood; the Android toolchain is proven or a
fallback decision is recorded.

### M1 — Vertical Slice (Weeks 2–3)

Goal: one complete round with one task type.

- Fixed map with spots, spawn points, static obstacles, and a perimeter loop.
- Four cars, arcade movement, environment collisions, no car-to-car collisions.
- One target spot per player, exclusive assignment, parking detection.
- Time score + basic precision + collision count, live score, 3-minute timer.
- Placeholder HUD and results screen.
- Go session/slots/input validation; controller connect/reconnect; host start.
- Continuous playtest on the real hardware.

**Exit criteria:** four people play a full round on the real tablets and display;
scores and reconnects work; the team can demo it.

### M2 — MVP (Weeks 4–5)

Goal: the complete MVP described in the requirements.

- Five tasks per player, task pool, auto-skip, finish bonus, tie-breakers.
- Full precision scoring and result ratings.
- Attract mode, results screen, pause/reset/kick.
- Robust reconnect, all-disconnected pause, game/server restart.
- Controller task/score/time feedback and setup screen.
- Test suite for Go and Godot logic; CI green.
- First internal tournament/playtest.

**Exit criteria:** MVP acceptance criteria pass on hardware; no P0/P1 defects;
docs and decision log updated.

### M3 — Festival-Ready (Weeks 6–7)

Goal: the stand can run without a developer.

- Festival launcher and packaging (Go + Godot + config + logs).
- Kiosk-like tablet setup: home launcher, immersive, keep-awake, boot start.
- Dedicated router/hotspot, static addressing, firewall, no internet dependency.
- Attract/queue flow, reset between groups, printed instructions.
- Failure drills: tablet, game, server, router, TV.
- 60-minute endurance test; spare hardware and power plan.
- Full rehearsal with people who have never played.

**Exit criteria:** a non-developer host can run three consecutive groups using
the runbook; recovery drills pass.

### M4 — Post-MVP (After Festival)

Only after the MVP is stable:

- Racing mode on the perimeter loop.
- Drift mechanic.
- Extra task types.
- Web host panel.
- Anonymous event leaderboard, if privacy-safe.
- Art/audio polish.

## Sprint 0 Backlog (Start Here)

| Item | Area | Owner role | Output |
|------|------|------------|--------|
| S0-01 Record decisions and ADRs | docs | Backend/PM | Decision log, ADRs |
| S0-02 Collect tablet specs and battery health | ops | QA/Controller | Hardware inventory |
| S0-03 Minimal Kotlin WebSocket app on all 4 tablets | controller | Controller/UI | APK + test notes |
| S0-04 Go WebSocket server with fake controller | server | Backend | Running server + tests |
| S0-05 Godot connects to Go and renders 4 rectangles | game | Game | Fullscreen demo |
| S0-06 Measure latency and Wi-Fi stability | ops | Backend/QA | Measurements + verdict |
| S0-07 Supervisor starts/restarts Godot | server | Backend | Crash-recovery demo |
| S0-08 GitHub board, templates, DoR/DoD, CI skeleton | process | PM/Backend | Project setup |
| S0-09 Test plan and first hardware checklist | docs | QA | `testing.md` updates |

Do not start M1 until S0-03, S0-04, S0-05, and S0-06 have a clear result.

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
| Unknown availability | High | Medium | One-week sprints; small items; measure throughput; cut scope | PM |
| Festival date unknown | High | High | Get the date; plan relative milestones; set internal target | PM |
| Hardware/TV/laptop not available | Medium | High | Confirm owner and specs; test early; backup monitor/laptop | Backend/QA |
| AI-generated code quality | Medium | Medium | Review, tests, reproduce, no blind merges | Team |
| Personal data leaks into scores/logs | Low | High | No names; random tokens; truncate logs; clear between groups | PM/Backend |

## Dependencies and Gates

- M1 cannot start until M0 proves the controller→Go→Godot path.
- M2 cannot start until M1 has a playable round on real hardware.
- M3 cannot start until the MVP acceptance criteria pass.
- Festival date determines the final cut line. If time is short, ship MFD and
  move everything else to Post-MVP.
- Any change to stack, transport, or game rules requires a new decision/ADR.

## Capacity Notes

The team has four people with unknown weekly availability. For the first two
sprints, record planned vs actual hours and completed items. Use the observed
throughput to plan, not a guessed velocity. Protect one weekly demo slot even in
a bad week; a small demo keeps the project honest.
