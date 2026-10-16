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
| New feature / requirement | `01-vision-scope/vision-scope.md`, `04-product/requirements.md`, relevant file in `02-flanki-domain/` and `03-users-permissions/` |
| Bracket / tournament logic | `02-flanki-domain/domain-overview.md` |
| Accounts, roles, authz, age, privacy | `03-users-permissions/roles-permissions-gdpr.md` |
| Tech stack, hosting, DB, framework | `05-architecture-decisions/decision-log.md` — if status is `TBD`, do not introduce a choice |
| Git, tests, AI workflow | `06-development/workflow.md` |

## Map

- `00-meta/` — how to write docs, glossary.
- `01-vision-scope/` — vision, MVP scope, non-goals.
- `02-flanki-domain/` — Flanki rules, tournament lifecycle, single-elimination.
- `03-users-permissions/` — roles, permission matrix, GDPR and 18+ policy.
- `04-product/` — functional requirements, user stories.
- `05-architecture-decisions/` — decision log and ADRs. Undecided stack lives here.
- `06-development/` — git workflow, testing, working with AI agents.
- `_templates/` — templates for ADR, decision entry, feature spec.

## Update Rule

After every code or decision change, update the relevant document(s), bump `Last updated`,
and add an entry to `05-architecture-decisions/decision-log.md` if a decision changed.
A change without a docs update is incomplete.
