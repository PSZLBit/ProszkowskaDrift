# Development Workflow

Status: Draft
Last updated: 2026-10-19

## Monorepo Layout

- `server/` — PC program owning sessions, inputs, game driving, and display. (Not yet scaffolded; stack TBD per D-17.)
- `game/` — swappable game module behind the server game interface. (Concept TBD per D-13.)
- `controller-android/` — Kotlin native Android tablet controller. (Input set TBD per D-14.)
- `documentation/` — source of truth and decision history.

## Git

- Keep changes focused. Do not bundle unrelated refactors.
- Preserve existing user changes; do not discard unrelated work.
- Write commit messages in English.
- Commit and push only when explicitly requested.

## Definition of Done

A task is done only when:

1. Code or change is implemented in a focused way.
2. Relevant `documentation/` file(s) are updated with a new `Last updated` date.
3. Decision log is updated if a decision changed.
4. Focused tests are added or updated for behavior changes.
5. Checks that could not run are reported.

## Testing

No test runner is configured yet (stack TBD, see D-17). Once chosen, document the exact
test and build commands here per component (`server/`, `game/`, `controller-android/`).
Until then, report which checks could not be run.

## Working With AI Agents

- Agents must follow `documentation/README.md` and `00-meta/documentation-rules.md`.
- Agents must inspect the repository before making changes and follow existing structure.
- Agents must not assume a framework, database, hosting platform, transport protocol, or game engine (D-01, D-17).
- A `TBD` decision-log row blocks implementation of that topic.
- All generated files, code, comments, and docs must be in English.
