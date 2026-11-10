# Agent Instructions

## Project Context

Proszkowska Drift (working title) is a local multiplayer arcade parking game.
One to four players use four team-owned Samsung tablets (Android 4.4.2, API 19)
as controllers. A laptop runs the game and shows it on a TV/monitor. Players
complete a sequence of parking tasks in three-minute rounds and score points for
speed, precision, and clean driving. No accounts, no internet play, no personal
data. Drift and racing are post-MVP.

Monorepo layout (D-07): `server/` (Go), `game/` (Godot 4),
`controller-android/` (minimal Kotlin, API 19), `documentation/`.

## Current Decisions

- D-19: top-down 2D parking time attack; 1–4 independent players; fixed map;
  five tasks per player; environment collisions only.
- D-20: Android 4.4.2 controller; minimal Kotlin; plain Views; no Compose, no
  OkHttp 4, and no AndroidX unless a library is proven API 19 compatible;
  compatibility is a Sprint 0 gate, with a minimal Java app as fallback.
- D-21: fixed tablets auto-connect to a configured server and reclaim slots with
  a random device token; QR/manual address is only a setup fallback.
- D-22: Go server + Godot 4 game + minimal Kotlin controller; WebSocket + JSON;
  server owns session/input authority, game owns simulation/scoring; in-memory.
- D-23: 60 FPS display, 20–30 Hz input, ≤100 ms input-to-visible target.
- D-24: a disconnected car stops and its slot is reserved; all-disconnected
  pauses; late join is between rounds; game/server crashes restart to `Attract`.
- D-25: in-memory state only; no accounts, names, photos, or personal data.
- D-27: one-week sprints, weekly demo/retro, GitHub Projects, DoR/DoD, peer
  review, CI, AI-assistance disclosure.
- D-28: working agreement: English artifacts, small regular commits with
  conventional prefixes, test-first for pure logic, tests or documented manual
  hardware checks for behavior changes.
- D-29: on a task branch, commit each coherent step without waiting; pushing,
  merging, and opening a PR still require an explicit request.
- D-30: project docs are published as a read-only MkDocs site on GitHub Pages;
  `documentation/` remains the source of truth.
- D-14 is still TBD: the controller input scheme must be chosen by playtesting
  on the real tablets. Do not hard-code a scheme as final.

Decisions are recorded in `documentation/05-architecture-decisions/decision-log.md`.
A `TBD` row blocks production implementation; only a timeboxed spike (D-18) may
explore it.

## Language

All project artifacts are in English (D-05): documentation, code, comments,
commit messages, issue and PR titles/descriptions, review comments, branch
names, and UI strings. Do not add non-English content to the repository.

## Documentation Is the Source of Truth

`documentation/` is the source of truth. Code implements what is documented there.

Read-first protocol before any task:

1. Read `documentation/README.md` (router) plus
   `documentation/00-meta/documentation-rules.md` and
   `documentation/00-meta/glossary.md`.
2. Read the module file(s) for the task from the router table. Most code tasks
   also need `06-development/architecture.md` and `06-development/protocol.md`.
3. Check `documentation/05-architecture-decisions/decision-log.md` for open
   `TBD` items.
4. Only then inspect code.

Update-after protocol: after every behavior or decision change, update the
relevant `documentation/` file(s), bump `Last updated`, and add a decision-log
entry when a decision changed. A change without a docs update is incomplete.

Truth hierarchy: `documentation/` (`Decided` / `Implemented`) > code > chat
discussion. If docs and code disagree, fix one of them with a dated update
instead of leaving them inconsistent.

## Documentation Site

- `documentation/` remains the source of truth; `documentation/README.md` is the
  router. The published site (<https://pszlbit.github.io/ProszkowskaDrift/>) is
  a read-only copy.
- The site is built from root `mkdocs.yml`; `site/` is generated output and must
  not be committed or edited.
- Run `mkdocs build --strict` after documentation changes. CI builds it on pull
  requests and deploys `main` to GitHub Pages.

## Collaboration Rules

Detailed process: `documentation/06-development/workflow.md`. Detailed testing:
`documentation/06-development/testing.md`. These rules are binding for agents.

### Issues and PRs

- Work from a GitHub issue. Link the PR with `Closes #<n>`.
- One PR does one thing. Include what changed, why, and how it was tested.
- Disclose AI assistance and state what the author verified.
- Use the PR template. Never describe planned behavior as implemented.
- Keep the board accurate and write blockers in the issue or PR, not only in
  chat.

### Commits

- English, imperative summary, conventional prefix:
  `<type>(<scope>): <summary>`.
- Types: `feat`, `fix`, `docs`, `test`, `chore`, `spike`.
- Scopes: `server`, `game`, `controller`, `docs`, `ops`.
- Commit small and regularly: one logical change per commit, and commit each
  coherent step instead of batching days of work.
- Never bundle unrelated refactors, formatting, or generated files.
- Never commit secrets, tokens, personal data, generated builds, or local
  configuration. If a secret is committed, report it; do not rewrite shared
  history yourself.
- No direct commits to `main`. No force-push and no history rewrite on shared
  branches.
- Commit each coherent step on a task branch without waiting for a separate
  instruction. Pushing, merging, and opening a PR require an explicit request
  (D-29).
- Keep branches short-lived: days, not weeks.

### Way of Working

- Work in small vertical slices. Prefer a working end-to-end path over a broad
  unfinished layer. Demo on the real target hardware every week when the work
  touches the controller, game, or network.
- **Test-first for pure logic:** scoring, task assignment, session/slot logic,
  input validation, protocol parsing. Red → green → refactor.
- **Manual check first for UI, visual, and hardware behavior:** write the check
  in the issue or in `testing.md`, then implement and run it on the real device.
- No behavior change without an automated test or a documented manual check.
- Bug fix: add a regression test when practical.
- Spikes are exempt from test-first but must record what was measured and the
  conclusion (D-18).
- Stop and ask when blocked, when a `TBD` blocks the work, or when scope is
  ambiguous. Do not invent a decision or hard-code a `TBD`.
- Prefer the simplest solution that fits the documented design. No speculative
  abstraction, new dependency, or new runtime without a decision.
- Definition of Done: `documentation/06-development/workflow.md`. Do not mark an
  item done before docs, tests, review, CI, and applicable hardware checks are
  complete.

## Working Guidelines

- Inspect the repository before making changes and follow its existing structure
  and conventions.
- Do not introduce a new server language, game engine, transport, database,
  hosting platform, or major library. The MVP stack is fixed by D-22 and
  ADR-0001. A change requires a new dated decision/ADR.
- Do not introduce Rust, Next.js runtime code, a cloud service, or a database
  into the MVP. Keep the stack to Go, GDScript, and Kotlin.
- Respect Android 4.4.2 (API 19). Check every Android dependency against API 19;
  prefer no dependency over a risky one. See ADR-0002.
- The server is authoritative for session actions and input authenticity. The
  game is the trusted local simulation/scoring module. Controllers never talk to
  the game directly.
- Sessions hold transient state only: slots, device tokens, connection handles,
  game state, scores. Discard at session end. Do not store accounts, names,
  photos, Android IDs, or other personal data.
- Keep authorization boundaries clear: slot assignment, kick, start/pause/reset/
  end, and input validity are enforced on the server. Never rely on controller UI
  visibility.
- Do not implement Post-MVP features (racing, drift, extra task types, web host
  panel) without a new decision. Keep the perimeter loop in the map, but do not
  build racing.
- Use a timeboxed spike (D-18) only to resolve a specific TBD; spike code is
  throwaway and is not merged as production.
- Add or update focused tests for behavior changes. Check the repository for its
  current test and build commands before running verification; report any checks
  that could not be run.
- Keep commits small, focused, and in English. Follow
  `documentation/06-development/workflow.md`.
- Do not add tooling before its trigger; follow
  `documentation/06-development/tooling-plan.md`.
- Preserve existing user changes and do not discard unrelated work.
- AI-assisted code is allowed. Do not merge code that was not reviewed,
  understood, and tested. Never paste secrets, tokens, or personal data into an
  external model.
- Do not describe planned features as implemented unless code supports them.
