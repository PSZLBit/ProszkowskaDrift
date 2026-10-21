# Documentation — Source of Truth

Status: Decided
Last updated: 2026-10-21

> All project artifacts are in English. This folder is the source of truth.
> Code implements what is described here. Chat history does not override this folder.

Published copy: <https://pszlbit.github.io/ProszkowskaDrift/>. The files in this
folder remain the source of truth.

## Status Legend

- `Draft` — under discussion, do not implement as final.
- `Decided` — agreed by the team, ready to implement.
- `Implemented` — agreed and reflected in code.
- `TBD` — open question, explicitly not decided yet.

Every document must start with:

```text
Status: Draft | Decided | Implemented
Last updated: YYYY-MM-DD
```

## Router — What To Read First

Always read this file plus `00-meta/documentation-rules.md` and
`00-meta/glossary.md` first.

| Task | Read first |
|------|------------|
| Any coding task | This file + module file(s) + glossary + open decisions in `05-architecture-decisions/decision-log.md` + `06-development/architecture.md` + `06-development/protocol.md` |
| New feature / requirement | `01-vision-scope/vision-scope.md`, `04-product/requirements.md`, relevant file in `02-game-domain/` and `03-players-sessions/` |
| Game rules, scoring, round flow | `02-game-domain/domain-overview.md`, `02-game-domain/game-design.md` |
| Players, slots, permissions, reconnect, privacy | `03-players-sessions/roles-sessions.md` |
| Tech stack, transport, engine, hosting | `05-architecture-decisions/decision-log.md`, ADRs, `06-development/architecture.md` |
| Protocol messages | `06-development/protocol.md` |
| Server implementation and operations | `06-development/server.md`, `06-development/protocol.md` |
| Git, tests, Agile, AI workflow | `06-development/workflow.md`, `06-development/testing.md`, `06-development/roadmap.md` |
| Tooling, hooks, CI quality gates | `06-development/tooling-plan.md` |
| Festival setup and recovery | `06-development/festival-operations.md` |

## Map

- `00-meta/` — how to write docs, glossary, history notes.
- `01-vision-scope/` — vision, MVP scope, success criteria, non-goals.
- `02-game-domain/` — game concept, detailed game design, session lifecycle,
  real-time constraints.
- `03-players-sessions/` — player/host/spectator roles, slots, connection and
  reconnect rules, session privacy.
- `04-product/` — functional and non-functional requirements per component.
- `05-architecture-decisions/` — decision log, ADRs, retained stack analysis.
- `06-development/` — architecture, protocol, workflow, testing, tooling plan,
  roadmap, festival runbook.
- `_templates/` — templates for ADR, decision entry, feature spec.

## Update Rule

After every code or decision change, update the relevant document(s), bump
`Last updated`, and add an entry to `05-architecture-decisions/decision-log.md`
if a decision changed. A change without a docs update is incomplete.

Superseded decisions stay in the log. Do not delete history.

## Current Decisions at a Glance

- Game: top-down 2D parking time attack (D-19).
- Stack: Go server + Godot 4 game + minimal Kotlin controller (D-22).
- Controller: Android 4.4.2, API 19, no Compose/OkHttp 4, AndroidX only if
  proven compatible (D-20).
- Connection: fixed tablets auto-connect and reclaim slots (D-21).
- Real-time: 60 FPS, 20–30 Hz input, ≤100 ms target (D-23).
- State: in-memory only, no personal data (D-25).
- Delivery: one-week sprints, GitHub Projects, DoR/DoD, CI (D-27).
- Working agreement: English artifacts, small regular commits with
  conventional prefixes, test-first for pure logic (D-28).
- Commit cadence: commit each coherent step on a task branch; push, merge, and
  PR only on explicit request (D-29).
- Docs publishing: read-only MkDocs site on GitHub Pages; `documentation/`
  remains the source of truth (D-30).
- Delivery calendar: festival 2026-11-24; festival-ready target 2026-11-18;
  MFD committed, MVP stretch (D-31).
- Server baseline: Go + gorilla/websocket, in-memory session, in-process game
  supervisor, protocol v1 clarified (D-32).
- Open: controller input scheme (D-14), Windows/Linux festival laptop, map/tuning
  details, and hardware specs.
