# Development Workflow

Status: Draft
Last updated: 2026-10-13

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

No test runner is configured yet (stack TBD). Once chosen, document the exact
test and build commands here. Until then, report which checks could not be run.

## Working With AI Agents

- Agents must follow `documentation/README.md` and `00-meta/documentation-rules.md`.
- Agents must inspect the repository before making changes and follow existing structure.
- Agents must not assume a framework, database, or hosting platform.
- All generated files, code, comments, and docs must be in English.
