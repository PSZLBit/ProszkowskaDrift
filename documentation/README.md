# Documentation — Source of Truth

> All project artifacts are in English. This folder is the source of truth.
> Code implements what is described here. Chat history does not override this folder.

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

Always read this file plus `00-meta/documentation-rules.md` and `00-meta/glossary.md` first.

| Task | Read first |
|------|------------|
| Any coding task | This file + module file(s) below + glossary + open decisions in `05-architecture-decisions/decision-log.md` |
| New feature / requirement | `01-vision-scope/vision-scope.md`, `04-product/requirements.md`, relevant file in `02-game-domain/` and `03-players-sessions/` |
| Game / session logic | `02-game-domain/domain-overview.md` |
| Players, slots, session permissions, privacy | `03-players-sessions/roles-sessions.md` |
| Tech stack, transport, engine, hosting | `05-architecture-decisions/decision-log.md` — if status is `TBD`, do not introduce a choice |
| Git, tests, AI workflow | `06-development/workflow.md` |

## Map

- `00-meta/` — how to write docs, glossary.
- `01-vision-scope/` — vision, MVP scope, non-goals.
- `02-game-domain/` — game concept (TBD), session lifecycle, real-time constraints.
- `03-players-sessions/` — player/host roles, slot permissions, session privacy (no accounts).
- `04-product/` — functional requirements per component (controller/server/game).
- `05-architecture-decisions/` — decision log and ADRs. Undecided stack lives here.
- `06-development/` — monorepo layout, git workflow, testing, working with AI agents.
- `archive/` — superseded Flanki Organizer docs, kept for history only. Do not implement from these.
- `_templates/` — templates for ADR, decision entry, feature spec.

## Update Rule

After every code or decision change, update the relevant document(s), bump `Last updated`,
and add an entry to `05-architecture-decisions/decision-log.md` if a decision changed.
A change without a docs update is incomplete.
