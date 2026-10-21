# Development Workflow

Status: Decided
Last updated: 2026-10-19

## Monorepo Layout

- `server/` — Go session server, WebSocket endpoints, validation, supervisor,
  and launcher scripts. (Not yet scaffolded.)
- `game/` — Godot 4 project: map, simulation, scoring, HUD, display. (Not yet
  scaffolded.)
- `controller-android/` — minimal Kotlin Android controller for API 19. (Not yet
  scaffolded.)
- `documentation/` — source of truth and decision history.
- Tests live next to the code they test; no separate top-level test tree.

The four top-level component directories are fixed by D-07. Additional files
live inside the relevant component.

## Team and Responsibilities

The team has four people. Roles are responsibilities, not exclusive ownership.

| Role | Primary responsibility |
|------|------------------------|
| Product / PM | Backlog, acceptance criteria, sprint cadence, risk register, testing coordination, festival operations, demo script |
| Backend / DevOps | Go server, protocol, session/slots, reconnects, CI, packaging, network, process supervision |
| Game | Godot gameplay, map, movement, collisions, parking detection, scoring, HUD, attract/results |
| Controller / UI | Kotlin controller app, touch controls, feedback, setup screen, visual consistency across controller and display |
| QA (shared) | Everyone tests; PM coordinates the test plan, regression checklist, and acceptance sign-off |

No one merges their own unreviewed change. The PM can review process/docs and
coordinate testing; code review requires a developer.

## Agile Cadence

- **One-week sprints.** The team has unknown and variable availability; short
  sprints keep scope small and feedback frequent.
- **Sprint planning:** 30 minutes at the start of the week. Select a small batch
  from `Ready`, confirm the sprint goal, and check capacity.
- **Weekly demo:** 15 minutes at the end of the week. Show working behavior on
  the real hardware where possible.
- **Retrospective:** 15 minutes after the demo. One thing to keep, one to change,
  one action with an owner.
- **Daily check-in:** asynchronous in the team chat. What moved, what is blocked,
  what is next. No mandatory meeting.
- **Backlog refinement:** continuous; the PM keeps the next two weeks of items
  `Ready`.
- **WIP limit:** one or two active items per person. Finish before starting.
- **Capacity:** record actual availability per sprint for the first two sprints;
  use observed throughput instead of invented velocity.

## GitHub Projects

Use GitHub Projects as the single work board, with issues and milestones.

Board: <https://github.com/orgs/PSZLBit/projects/1>.

**Fields:**

- `Status`: Backlog, Ready, In Progress, Review, Test, Done.
- `Priority`: P0 (blocks demo), P1 (MVP), P2 (nice-to-have).
- `Size`: S, M, L (no more than one L per sprint per person).
- `Area`: server, game, controller, docs, ops, process.
- `Sprint`: one-week iteration.
- `Owner`: one person accountable.
- `Risk`: optional flag for items with hardware/network uncertainty.

Implementation note: GitHub has no custom people field, so `Owner` is the
single assignee on an item. `Risk` is a `Yes`/`No` flag; leave it blank when
the item is not flagged.

**Views:**

- Board by `Status`.
- Table with all fields for planning.
- Roadmap by milestone.
- My Work filtered by owner.
- Bugs filtered by `bug`.

**Milestones:**

- `M0 — Decisions and spikes`
- `M1 — Vertical slice`
- `M2 — MVP`
- `M3 — Festival-ready`
- `M4 — Post-MVP`

**Issue types:** feature, bug, spike, chore, docs. Each issue has a clear
outcome and acceptance criteria. Link every PR to an issue.

## Git and Pull Requests

- `main` is protected. No direct pushes.
- Install the local hook after cloning:

  ```sh
  git config core.hooksPath .githooks
  ```

- Enable GitHub branch protection/rules for `main`: require a pull request,
  at least one review, and green CI.
- Branch names: `feat/<short-name>`, `fix/<short-name>`, `docs/<short-name>`,
  `spike/<short-name>`, `chore/<short-name>`.
- Commit convention: `<type>(<scope>): <summary>`, for example
  `feat(server): validate input rate`. Types: `feat`, `fix`, `docs`, `test`,
  `chore`, `spike`. Scopes: `server`, `game`, `controller`, `docs`, `ops`.
  Summary in imperative English; explain why in the body when it is not obvious.
- Commit small and regularly: one logical change per commit, and commit as soon
  as a coherent step is complete. Do not leave days of work uncommitted. Never
  bundle unrelated refactors, formatting, or generated files.
- Push feature branches, never `main`. Do not force-push a shared branch and do
  not rewrite history after review.
- Keep branches short-lived: days, not weeks. Bring in `main` before review if
  needed, without rewriting published history.
- On a task branch, commit each coherent step without waiting for a separate
  instruction. Pushing, merging, and opening a pull request still require an
  explicit request (D-29).
- One PR should do one thing. Include what changed, why, how it was tested, and
  any AI assistance.
- Do not commit generated artifacts, APKs, logs, or local configuration.
- Merge only after review and CI. Delete the branch after merge.

## Working Agreement

- Work in small vertical slices that produce something demoable. Prefer a
  working end-to-end path over a broad unfinished layer.
- **Test-first for pure logic:** game rules, scoring, task assignment, session
  and slot logic, input validation, and protocol parsing. Write the failing test
  first, then the implementation.
- **Manual check first for UI, visual, and hardware behavior:** write the check
  in the issue or in `06-development/testing.md`, then implement and run it on
  the real hardware. Automated UI tests are not required for MVP.
- No behavior change without either an automated test or a documented manual
  verification. A bug fix adds a regression test when practical.
- Do not chase coverage percentages. Cover decisions, boundaries, and edge cases
  that can break the round or the festival stand.
- Stop and ask when blocked, when a TBD blocks the work, or when scope is
  ambiguous. Do not invent a decision, hard-code a TBD, or silently expand scope.
- Prefer the simplest solution that fits the documented design. Do not add
  speculative abstraction, a new dependency, or a new runtime without a decision.
- Keep the board accurate: update the issue status and write blockers in the
  issue/PR, not only in chat.
- Demo on the real target hardware every week when the work touches controller,
  game, or network behavior.

## Definition of Ready

An item can enter a sprint only when:

- The outcome and user value are clear.
- Acceptance criteria are written and testable.
- Dependencies and blocking decisions are resolved, or the item is explicitly a
  spike.
- Size is S or M; L is split.
- A test approach is named.
- One owner is assigned.
- For controller/game work, the target hardware is known.

## Definition of Done

A task is done only when:

1. The change is implemented and focused.
2. Focused automated tests are added or updated where behavior changed.
3. The change was manually verified on the target hardware when applicable
   (Android 4.4.2 tablet, target laptop/TV, or festival network).
4. Relevant `documentation/` files are updated with a new `Last updated` date.
5. The decision log or an ADR is updated if a decision changed.
6. A pull request is reviewed by at least one other person and CI is green.
7. The behavior is demoed or recorded.
8. No P0/P1 defect is introduced.
9. Checks that could not run are reported in the PR.

## Spikes

A TBD can block production implementation. To resolve uncertainty, a timeboxed
spike is allowed under D-18:

- Maximum one week; usually one to three days.
- Create a `spike` issue with the question, timebox, and expected output.
- Work on a `spike/...` branch. Spike code is throwaway and is not merged as
  production.
- Record the result in the decision log and the relevant doc, then close the
  spike.
- If the spike changes a decision, add a new dated decision; never edit history
  silently.

## AI-Assisted Work

AI assistance is expected and allowed. The human author remains accountable.

- State in the PR whether AI assisted and what was verified.
- Review AI-generated code like any other code; do not merge code you do not
  understand.
- Run tests and reproduce behavior; do not trust generated claims.
- Never paste secrets, tokens, personal data, or private repository content into
  an external model.
- Check licenses of generated or copied code and assets.
- Keep prompts and generated artifacts out of the repository unless they are
  useful documentation.

## Documentation Protocol

- `documentation/` is the source of truth. Read the router and the relevant
  module before work.
- Update docs in the same PR as the behavior change.
- Bump `Last updated`.
- Add a decision-log row for a decision change; use an ADR for larger decisions.
- Do not describe planned behavior as implemented.
- All artifacts are in English (D-05).

## Testing and CI

See `06-development/testing.md` for the strategy and
`06-development/tooling-plan.md` for the planned tooling and quality gates. The
planned CI runs Go tests/vet and builds the Android APK; Godot headless tests are
added once the project exists. Use free-tier GitHub Actions and cache
dependencies. Report any check that cannot run.
