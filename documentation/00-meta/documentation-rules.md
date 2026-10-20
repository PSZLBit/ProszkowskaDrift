# Documentation Rules

Status: Decided
Last updated: 2026-10-19

## Principles

1. `documentation/` is the source of truth. Code follows docs, not the reverse.
2. One topic per file. Keep files small so agents can read them fully.
3. No duplication. Define a term once in `00-meta/glossary.md` and link to it.
4. Mark uncertainty explicitly with `TBD`. Never silently invent missing facts.
5. All files, code, comments, and commit messages are in English.

## File Header

Every file starts with:

```text
Status: Draft | Decided | Implemented
Last updated: YYYY-MM-DD
```

## Lifecycle

- New idea: `Draft`.
- Team agreement: change to `Decided`, add entry to `05-architecture-decisions/decision-log.md`.
- Code merged that reflects the doc: change to `Implemented`.
- A decision change adds a new dated row or ADR. Superseded rows stay in the
  log; never delete decision history.
- A `TBD` row blocks production implementation. A timeboxed spike may explore it
  under D-18; spike code is throwaway and is not merged as production.

## Read-First Protocol for Agents

1. Read `documentation/README.md` (router).
2. Read `00-meta/documentation-rules.md` (this file) and `00-meta/glossary.md`.
3. Read the module file(s) for the task (see router table). Code tasks also
   need `06-development/architecture.md` and `06-development/protocol.md`.
4. Check `05-architecture-decisions/decision-log.md` for open `TBD` items.
5. Only then inspect code. If there is no code yet, check
   `06-development/roadmap.md` for the current milestone.

## Update-After Protocol for Agents

1. Implement the change in a focused way.
2. Update the relevant `documentation/` file(s) and bump `Last updated`.
3. If a decision changed, add a row to the decision log or a new ADR.
4. Never describe a planned feature as implemented unless code supports it.

## Hierarchy of Truth

1. `documentation/` files with status `Decided` or `Implemented`.
2. Code.
3. Chat or issue discussion.

If docs and code disagree, fix code or update docs with a dated decision — do not leave them inconsistent.
