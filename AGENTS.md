# Agent Instructions

## Project Context

Flanki Organizer is an early-stage web app for managing Flanki competitions, including competitor registration, organizer tools, announcements, and single-elimination bracket generation. The technology stack and development workflow have not yet been selected.

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
- Do not assume or introduce a framework, database, hosting platform, or other major technology choice; the team is still deciding the stack. A `TBD` entry in the decision log blocks implementation of that topic.
- Keep changes focused and avoid unrelated refactors or new dependencies unless the task requires them.
- Treat account details, especially dates of birth, as personal information. Avoid exposing them unnecessarily in UI, logs, or public competition views.
- Enforce 18+ eligibility handling without leaking dates of birth; see `documentation/03-users-permissions/roles-permissions-gdpr.md`.
- Keep authorization boundaries clear between competitors, competition organizers, and site administrators. Do not rely on UI visibility alone to protect privileged actions; enforce checks on the server.
- Do not describe planned features as implemented unless the code supports them.
- Add or update focused tests for behavior changes. Check the repository for its current test and build commands before running verification; report any checks that could not be run.
- Preserve existing user changes and do not discard unrelated work.
