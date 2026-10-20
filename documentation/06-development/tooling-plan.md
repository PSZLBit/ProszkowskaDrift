# Tooling and Quality Gate Plan

Status: Draft
Last updated: 2026-10-19

Planned developer tooling for the project. Nothing here is implemented beyond
the repository hygiene files under "Available now". A tool enters the repository
only with a **trigger**, an **owner**, and — if it changes the stack or the merge
gate — a **decision entry** (D-22, D-27, D-28).

## Available Now

| Tool | Purpose | Notes |
|------|---------|-------|
| `.editorconfig` | Consistent indentation and line endings across editors | Go tabs, GDScript tabs, Kotlin 4 spaces |
| `.gitattributes` | LF normalization and binary-file handling | Protects Windows/Linux collaboration |
| `.gitignore` | Keep builds, local config, secrets, and generated files out | Update per component after scaffold |
| `.github/CODEOWNERS` | Default review routing | Component placeholders to fill when roles are assigned |
| `.githooks/pre-push` | Block direct pushes to `main` | Install with `git config core.hooksPath .githooks` |

## Formatting and Linting

Add per component, when the component exists.

| Component | Tool | Trigger | Notes |
|-----------|------|---------|-------|
| Go server | `gofmt`, `go vet`; optionally `golangci-lint` | `server/` scaffold | Start with gofmt + vet; add more linters only if they catch real issues |
| Kotlin controller | `ktlint` or `detekt` | `controller-android/` scaffold | Versions must work with the chosen AGP/Kotlin; do not add AndroidX/Compose tooling |
| GDScript | `gdformat` / `gdlint` (gdtoolkit) | `game/` scaffold | Optional; keep Python tooling out of the shipped game |
| Markdown | `markdownlint` + link checker | When the docs volume grows | Keep it fast; optional and not a merge blocker at first |
| YAML/JSON | Editor/schema validation | With the first config file | Avoid a Node-based toolchain unless it earns its place |

## Git Hooks

- `.githooks/pre-commit`: run formatters/linters on changed files only, once the
  toolchains exist. It must stay fast (a few seconds). Document an emergency
  bypass; do not make a hook required for correctness.
- `.githooks/pre-push`: already blocks direct pushes to `main`; keep it.
- Do not add a hook before its tool is configured. A missing binary must not
  break commits.

## Commit Message Validation

- Goal: make the `<type>(<scope>): <summary>` convention (D-28) automatic.
- Preferred order: a small shell or Go script run by CI, or `commitlint` if the
  team accepts a dev-only Node tool. Prefer the script; do not add Node to the
  product.
- Trigger: first CI workflow (M1). Until then, review the convention manually.
- Validate messages on pull requests. Do not block GitHub-generated merge
  commits or the initial docs commits if they predate the rule.

## CI Quality Gates

Planned for the first component scaffold (M1), free tier only:

1. Go job: `go test -race ./...`, `go vet ./...`, build.
2. Android job: Gradle `assembleDebug` with cached dependencies; upload the APK.
3. Godot job: headless logic tests when the test addon is chosen; export only on
   tags if CI time becomes a problem.
4. Docs job: optional Markdown link check.
5. Cache Go modules, Gradle, and Godot downloads where possible.
6. Required checks for `main`: Go tests/vet and the Android build. Add Godot
   tests when configured.

No containers, cloud runners, or paid services. If a check cannot run, the PR
must say so (Definition of Done in `workflow.md`).

## Coverage

- No percentage threshold at MVP. Coverage is a diagnostic, not a gate.
- If a threshold is added later, apply it only to pure-logic packages (scoring,
  task assignment, session/slots, validation, protocol) and keep it low enough
  to avoid test theater.
- Prefer covering the decisions and edge cases listed in `testing.md`.

## Dependency and Security Checks

- Dependabot for Go modules, Gradle, and GitHub Actions (free for the repo).
- `go vuln check ./...` in the Go CI job once `server/` exists.
- Review every Android dependency against API 19 before adding it (ADR-0002).
- Never commit secrets, tokens, personal data, or local configuration.

## Release and Packaging

Planned for M3 (festival-ready):

- Version stamping for the Go server and the Godot export.
- A build script that produces one festival folder: server binary, game export,
  config example, launcher, and a logs directory.
- Checksums for the exported artifacts.
- A launcher smoke test: start, health check, stop, restart.
- An APK build and install checklist for the four tablets.

## Deliberately Not Planned for MVP

| Item | Reason |
|------|--------|
| Coverage threshold as a merge gate | Would force low-value tests in a small team |
| Docker/containers | No service fleet; the laptop runs two local processes |
| Cloud CI runners or paid services | Free tier is enough; no cloud dependency (D-22) |
| Code-signing infrastructure | Manual APK install on four team-owned tablets |
| MDM / full kiosk provisioning | API 19 has no screen pinning; the supervised stand is accepted (ADR-0002) |
| Binary WebSocket protocol | JSON is enough for four players; revisit only if D-23 measurements fail |
| Auto-deploy or release automation | Manual festival packaging with a checklist is safer |

## Adding a New Tool

Before adding anything to the repository:

1. Name the trigger (component scaffold, failed measurement, repeated manual
   work).
2. Name the owner and where the tool runs (local hook, CI, manual before event).
3. Check the free tier and platform support (API 19, Linux/Windows).
4. If it changes the stack or the merge gate, add a dated decision/ADR first.
5. Update this file and `06-development/testing.md` in the same change.
