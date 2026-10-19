# Agent Instructions

## Project Context

Local real-time play system: Android tablets (Kotlin native) act as controllers, a PC server receives inputs, drives a swappable game, and renders it on a connected big screen. One session hosts 1–4 players on a local network via lobby code/QR. No accounts. Monorepo layout: `server/`, `game/`, `controller-android/`, `documentation/`. Game concept, controller input set, session edge cases, latency target, and server/game/transport stack are TBD and block implementation.

## Language

All project artifacts — documentation, code, comments, commit messages, UI strings — are in English.

## Documentation Is the Source of Truth

`documentation/` is the source of truth. Code implements what is documented there.

Read-first protocol before any task:

1. Read `documentation/README.md` (router) plus `documentation/00-meta/documentation-rules.md` and `documentation/00-meta/glossary.md`.
2. Read the module file(s) for the task from the router table.
3. Check `documentation/05-architecture-decisions/decision-log.md` for `TBD` items.
4. Only then inspect code.

Update-after protocol: after every behavior or decision change, update the relevant `documentation/` file(s), bump `Last updated`, and add a decision-log entry when a decision changed. A change without a docs update is incomplete.

Truth hierarchy: `documentation/` (`Decided` / `Implemented`) > code > chat discussion. If docs and code disagree, fix one of them with a dated update instead of leaving them inconsistent.

## Working Guidelines

- Inspect the repository before making changes and follow its existing structure and conventions.
- Do not assume or introduce a server language, game engine, transport protocol, database, hosting platform, or other major technology choice; the team is still deciding the stack. A `TBD` entry in the decision log blocks implementation of that topic.
- Keep changes focused and avoid unrelated refactors or new dependencies unless the task requires them.
- Sessions hold transient state only (player slots, connection handles, game state). Do not store accounts or personal data; discard session data when the session ends unless a decision says otherwise.
- Keep authorization boundaries clear between players and the host. Do not rely on controller UI visibility alone to protect privileged actions; enforce slot, kick, start/end, and input-validity checks on the server; see `documentation/03-players-sessions/roles-sessions.md`.
- Do not describe planned features as implemented unless the code supports them.
- Add or update focused tests for behavior changes. Check the repository for its current test and build commands before running verification; report any checks that could not be run.
- Preserve existing user changes and do not discard unrelated work.
