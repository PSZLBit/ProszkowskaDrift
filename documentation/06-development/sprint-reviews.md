# Sprint Reviews and Plans

Status: Decided
Last updated: 2026-11-10

This file is the canonical record of sprint reviews and plans (D-27). Sprints are
one week long and end with a demo and retrospective. Each review and plan is
written here first and mirrored in GitHub Discussions.

Statuses below are a snapshot from 2026-11-03, with factual corrections through
2026-11-10. The GitHub Project and issues are authoritative for live status.

- Board: <https://github.com/orgs/PSZLBit/projects/1>
- Sprint 0 review:
  <https://github.com/PSZLBit/ProszkowskaDrift/discussions/52>
- Sprint 1 review:
  <https://github.com/PSZLBit/ProszkowskaDrift/discussions/53>
- Sprint 2 plan:
  <https://github.com/PSZLBit/ProszkowskaDrift/discussions/54>
- Project progress and re-baseline:
  <https://github.com/PSZLBit/ProszkowskaDrift/discussions/55>

## Sprint 0 review (2026-10-19 – 2026-10-25)

Goal: M0 decisions and spikes. Exit criteria: four tablets move four rectangles
through Go on the target display; latency and the Android toolchain are
understood.

**Verdict:** M0 partial; exit criteria not met. There was no four-tablet path, no
latency measurement, and no Android toolchain proof. M0 was closed on 2026-11-03
with carry-over.

| Planned item | Issue | Current owner | Outcome |
|--------------|-------|---------------|---------|
| S0-00 Confirm festival date and internal target | #8 | — (closed) | Done 2026-10-21: festival date and delivery calendar recorded (D-31). |
| S0-01 Audit decisions and ADRs | #9 | ZofiaDan | Open; audit deliverable in review (PR #68, 2026-11-09). Carry-over. |
| S0-02 Collect tablet specs and battery health | #10 | ZofiaDan | Open; static inventory merged in PR #44 on 2026-11-03; hardware checks pending (#11/#14/#49). |
| S0-03 Kotlin WebSocket spike on four tablets | #11 | TracerDS | Open; PR #63 merged with a WebSocket client, device-token store, reconnect/backoff, and 34 host-JVM tests; real-tablet proof is pending. Carry-over. |
| S0-04 Go WebSocket server with fake controller | #12 | GreenDude5 | Done 2026-10-23 (PR #30): protocol envelope, slots, reconnect, input validation, timers, host command path, fake clients, and CI. |
| S0-05 Godot renders four rectangles from Go | #13 | ZofiaDan (historically Brzana) | Partial; the four-car game scaffold was merged later as PR #31 in Sprint 1 (D-32). Target laptop/TV evidence is still pending. |
| S0-06 Measure latency and Wi-Fi stability | #14 | ZofiaDan | Not started. Carry-over. |
| S0-07 Supervisor starts and restarts Godot | #15 | GreenDude5 | Open; PR #59 is in review and is not merged; no manual crash-recovery proof is recorded. Carry-over. |
| S0-08 GitHub project and workflow | #16 | GreenDude5 | Done 2026-10-21: project board, templates, workflow, MkDocs site, and local setup. |
| S0-09 Test plan and hardware checklist | #17 | ZofiaDan | Merged; PR #65 added the MFD test plan and hardware checklist. |
| D-14 Choose controller input scheme | #18 | TracerDS | Not started. Carry-over. |

Owner is the current assignee on 2026-11-03. Issue #13 was historically Brzana
and is now ZofiaDan for the display check.

Metrics:

- 8 PRs merged: #3, #4, #5, #6, #7, #19, #20, #30.
- 6 issues closed: #8, #12, #16, #21, #22, #24.
- 8 carry-over issues: #9, #10, #11, #13, #14, #15, #17, #18.

### What worked

- Test-first server work and CI.
- Decisions and ADRs recorded early.
- Small PRs and a visible board.

### What did not

- Hardware spikes were not executed.
- The controller branch did not become a PR during Sprint 0.
- The test plan stayed Draft.
- M0 was closed late (2026-11-03) with carry-over.

### Retrospective

| Keep | Change | Action (owner) |
|------|--------|----------------|
| Server test discipline and small PRs | Every spike branch must become a PR within the sprint | All carry-over moved to explicit owners and sprints on 2026-11-03 (Team) |
| Decisions and ADRs recorded early | Hardware checks need a named owner and a date | #13 owner corrected to ZofiaDan for the display check; hardware work assigned in Sprint 2 (ZofiaDan) |

## Sprint 1 review (2026-10-26 – 2026-11-01)

Goal: M1 first half — map, four cars, collisions, one task type, parking
detection, scoring/timer, HUD/results, controller connect/reconnect, host start,
and continuous playtest.

**Verdict:** goal missed; only the game scaffold landed. M1 first half was not
achieved.

| Planned area | Outcome | Evidence |
|--------------|---------|----------|
| Fixed map, spawn points, obstacles, perimeter loop | Not started | No map work landed; the scaffold keeps placeholder boundary walls. |
| Four cars and environment collisions | Partial | PR #31: four slot-keyed cars, movement math, and headless tests; no real map or playtest. |
| One task type and parking detection | Not started | No task assignment or parking detection code. |
| Scoring, 3-minute timer, HUD, and results | Not started | No scoring, timer, HUD, or results code. |
| Controller connect, reconnect, and input | Not started | At Sprint 1 end, the controller scaffold was on an unmerged branch; no WebSocket, token, or reconnect (#11). PR #63 opened later. |
| Host start | Partial | Host keys exist in the game scaffold (PR #31); no start with a real tablet. |
| Continuous playtest on real hardware | Not started | No demo or playtest record. |
| Game design baseline (#32) | Not started | Issue #32 was Ready but not started. |

Metrics:

- 1 PR merged: #31.
- 1 PR opened in Sprint 1: #44 (merged 2026-11-03).
- 0 issues closed.
- 8 commits: 5 game commits by Brzana, 2 hardware-doc commits by ZofiaDan on the
  PR branch, and 1 merge commit.
- No demo or retrospective record found.

### What worked

- The game scaffold is tested and documented.
- CI and documentation quality stayed high.
- The hardware inventory is a useful start.

### What did not

- The Sprint 1 board had only one issue (#32), and it was not started.
- There was no sprint plan, demo, or retrospective record.
- The critical path (controller and parking loop) did not move.
- Work stayed on branches instead of becoming reviewable PRs.

### Retrospective

| Keep | Change | Action (owner) |
|------|--------|----------------|
| Small, tested increments and honest documentation | Plan every sprint on the board; start the critical path on day one; rebase and PR branch work | Re-baseline Sprint 2 to a single-tablet vertical slice (D-33, 2026-11-03) (Team) |

## Sprint 2 plan (2026-11-02 – 2026-11-08)

Goal: single-tablet end-to-end vertical slice. One real API 19 tablet connects
through Go, drives a car, gets one parking task, the game detects the park, and a
basic score/time/result is visible on the target laptop. Four-player validation
is not required this sprint.

### Scope and owners

| Issue | Work | Owner | Priority | Status |
|-------|------|-------|----------|--------|
| #10 | Tablet specs and battery health | ZofiaDan | P0 | In Progress |
| #11 | Kotlin WebSocket spike on one tablet | TracerDS | P0 | In Progress |
| #45 | Define the MFD gameplay baseline | Brzana | P0 | In Progress |
| #46 | One parking task with exclusive assignment and detection | Brzana | P0 | Ready |
| #47 | Score, timer, and results overlay | Brzana | P0 | Ready |
| #17 | Test plan and hardware checklist | ZofiaDan | P1 | In Review |
| #18 | D-14 choose controller input scheme | TracerDS | P0 | Ready |
| #25 | Queue backpressure and slow-client close | GreenDude5 | P1 | Done (PR #57) |
| #51 | Reconcile documentation and board drift | GreenDude5 | P1 | Closed (PR #56) |

### Out of scope

- Four tablets and the full four-player MFD run (Sprint 3, #49).
- Production controller (#48).
- Supervisor (#15, Sprint 3) and festival runbook and rehearsal (#50, Sprint 4).
- Full GDD (#32).
- Full five-task pool, attract mode, and Post-MVP features.

### Exit criteria

- One real tablet connects, drives, and reclaims its slot after a forced
  disconnect.
- One parking task is assigned, highlighted, detected, and completed exactly
  once.
- A basic 3-minute timer, score, and results view are visible.
- Manual checks are recorded in `testing.md`.
- The demo and retrospective are recorded at the end of the sprint.

### Risks and mitigations

| Risk | Mitigation |
|------|------------|
| Game is on the critical path with one owner (Brzana), and #45, #46, and #47 are all in this sprint | Do the baseline first; implement the smallest rules that satisfy it; pair GreenDude5 or TracerDS on game logic if the controller/server path is clear; cut polish, not the parking loop |
| The controller WebSocket approach and API 19 compatibility are still unproven | Spike on one real tablet early; reuse the existing server and fake controllers for integration; keep the minimal Java app fallback (D-20) open |
| Hardware access and team capacity are unknown | Confirm hardware dates with ZofiaDan; plan by observed throughput; move unfinished work to Sprint 3 |
| Scope creep into four-player features | Keep the single-tablet scope; four-player work is Sprint 3; gate new scope through the PM |

### Ownership rationale

- GreenDude5: server, CI, and process/docs work.
- Brzana: game and game design.
- TracerDS: controller, tooling, and Godot setup.
- ZofiaDan: hardware, QA, and documentation.

### Demo and retrospective

The demo and retrospective happen at the end of the sprint (2026-11-08). The
exact time is not fixed in this record.

## Project progress — 2026-11-03

### Timeline

- Festival-ready target: 2026-11-18, 15 days away.
- Festival: 2026-11-24, 21 days away.
- Buffer: 2026-11-19 to 2026-11-23 for fixes, setup, and final rehearsal.

### Component status

| Component | Status |
|-----------|--------|
| Server | S0-04 is done. Protocol, slots, reconnect, validation, timers, host path, and CI work. Supervisor (#15) is in review (PR #59). Backpressure (#25) and end-phase (#23) bugs are fixed by PR #57/#58. |
| Game | Scaffold is merged: four cars, protocol, 101 headless checks, and fullscreen. No map, task assignment, parking detection, scoring, HUD, or results. |
| Controller | API 19 plain-Views scaffold, WebSocket client, device-token store, and reconnect/backoff are merged (PR #63). Real-tablet proof is pending; production work is #48. |
| Ops | Static hardware inventory is merged in PR #44. Latency (#14) and display evidence (#13) are open. Supervisor (#15) is in review (PR #59); runbook and rehearsal (#50) are Sprint 4. |
| Docs | The decision log is strong. `game-design.md`, `testing.md`, `protocol.md`, and `festival-operations.md` are still Draft. Board drift was closed in #51; the requirements status follow-up is #67. |
| Board | Sprint 2 ended 2026-11-08; Sprint 3 started 2026-11-09. M0 is closed with carry-over. Every open issue has one owner, a milestone, a priority, and a sprint or backlog state. |

### Critical path

Controller WebSocket → game parking/scoring → one-tablet integration →
four-player MFD → festival hardening.

### Re-baseline (D-33)

- Sprint 1 missed its M1 first-half goal.
- Sprint 2 is the single-tablet vertical slice.
- Sprint 3 is the four-player MFD acceptance run.
- Sprint 4 is festival-ready work: runbook, failure drills, and rehearsal. The
  supervisor (#15) lands in Sprint 3 so it can be rehearsed. Freeze on
  2026-11-18.
- The MFD and festival dates do not change. The full MVP remains a stretch.

### Committed vs stretch vs Post-MVP

| Tier | Scope |
|------|-------|
| Committed (MFD) | Four tablets, four cars, one parking task, score, results, reconnect, setup/runbook. |
| Stretch (MVP) | Five tasks, full precision scoring, attract/results, auto-recovery, stable four-player rehearsal. |
| Post-MVP | Racing, drift, extra task types, web host panel. |

### Open decisions and unknowns

- D-14 controller input scheme (#18).
- OkHttp 3.12.x on API 19; PR #63 is merged, and real-tablet proof is pending
  (#11, #48).
- Windows vs Linux festival laptop.
- Router model and static-IP plan.
- Final map data format.
- Host second control device (Post-MVP).

### Documentation drift and corrections

- `game-design.md` is Draft while D-19 points to it; #45 defines the MFD baseline
  and #32 is deferred.
- `testing.md` is Draft; #17 owns the test plan.
- Root README and the documentation router still described Sprint 1; corrected
  on 2026-11-03.
- `workflow.md` milestone names did not match GitHub; corrected on 2026-11-03.
- #13 was Done on the board while the issue was open; corrected on 2026-11-03.
- `feat/controller-scaffold` did not become a PR during Sprint 0; PR #63 opened
  later on 2026-11-04. `feat/server-scaffold` is stale.
- PR #44 was merged on 2026-11-03; the earlier review request is closed.

### Risks

- Game delivery depends on one owner.
- The controller is merged; PR #63 is merged, and real-tablet WebSocket
  proof is pending.
- Hardware behavior, latency, and Wi-Fi are unmeasured.
- Team capacity is unknown.
- Scope creep and late integration.

### Next actions

| Date | Action | Owner |
|------|--------|-------|
| 2026-11-03 | Board re-baseline, sprint reviews, and documentation update | GreenDude5 |
| 2026-11-08 | Sprint 2 exit and demo/retrospective | Team |
| 2026-11-09 | Sprint 3 starts; four-player integration | Team |
| 2026-11-15 | MFD acceptance | Team |
| 2026-11-18 | Festival-ready freeze | Team |
| 2026-11-24 | Festival | Team |

## Update History

| Date | Change |
|------|--------|
| 2026-11-10 | Corrected PR #63/#59 and Sprint 2/Sprint 3 statuses; scoped the Sprint 1 controller outcome to the sprint end. |
| 2026-11-09 | Added the snapshot note and corrected merged/open status for #10, #17, #23, #25, #44, and #51. |
| 2026-11-03 | Added Sprint 0 and Sprint 1 reviews, Sprint 2 plan, and the D-33 progress review. |
