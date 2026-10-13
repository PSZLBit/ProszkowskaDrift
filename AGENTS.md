# Agent Instructions

## Project Context

Flanki Organizer is an early-stage web app for managing Flanki competitions, including competitor registration, organizer tools, announcements, and single-elimination bracket generation. The technology stack and development workflow have not yet been selected.

## Working Guidelines

- Inspect the repository before making changes and follow its existing structure and conventions.
- Do not assume or introduce a framework, database, hosting platform, or other major technology choice; the team is still deciding the stack.
- Keep changes focused and avoid unrelated refactors or new dependencies unless the task requires them.
- Treat account details, especially dates of birth, as personal information. Avoid exposing them unnecessarily in UI, logs, or public competition views.
- Keep authorization boundaries clear between competitors, competition organizers, and site administrators. Do not rely on UI visibility alone to protect privileged actions.
- Do not describe planned features as implemented unless the code supports them.
- Add or update focused tests for behavior changes. Check the repository for its current test and build commands before running verification; report any checks that could not be run.
- Preserve existing user changes and do not discard unrelated work.