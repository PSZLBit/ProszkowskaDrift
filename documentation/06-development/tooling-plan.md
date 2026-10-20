# Tooling and Quality Gate Plan

Status: Draft
Last updated: 2026-10-20

Developer tooling for the project. A tool enters the repository only with a
**trigger**, an **owner**, and — if it changes the stack or the merge gate — a
**decision entry** (D-22, D-27, D-28, D-30).

## Available Now

| Tool | Purpose | Notes |
|------|---------|-------|
| `.editorconfig` | Consistent indentation and line endings across editors | Go tabs, GDScript tabs, Kotlin 4 spaces |
| `.gitattributes` | LF normalization and binary-file handling | Protects Windows/Linux collaboration |
| `.gitignore` | Keep builds, local config, secrets, and generated files out | Update per component after scaffold |
| `.github/CODEOWNERS` | Default review routing | Component placeholders to fill when roles are assigned |
| `Makefile` | Local developer setup on Linux/macOS and Windows | Requires GNU Make; `make setup` installs hooks; add targets only when their toolchains exist |
| `.githooks/pre-push` | Block direct pushes to `main` | Installed by `make setup` |

`make setup` is the single local setup entry point. It is owned by the team and
triggered by repository onboarding. It must not install component toolchains
implicitly; those get their own targets when their tools are triggered.

Every Makefile target must work on Linux/macOS and Windows, or contain explicit
OS-specific branches. Keep target names identical and test both paths when
adding a target.

## Documentation Site

The project documentation is built with MkDocs and published to GitHub Pages
from the `main` branch (D-30).

| Item | Purpose | Notes |
|------|---------|-------|
| `mkdocs.yml` | Site configuration | Root config; `docs_dir: documentation`, `site_url` set for the project Pages path |
| `requirements-docs.txt` | Reproducible docs toolchain | Pins `mkdocs==1.6.1`; install with `pip install -r requirements-docs.txt` |
| `.github/workflows/docs.yml` | Build and deploy | `mkdocs build --strict` on docs changes; deploys `main` to Pages |

- Published URL: <https://pszlbit.github.io/ProszkowskaDrift/>.
- `documentation/` remains the source of truth; the published site is a
  read-only copy.
- `site/` is generated output; do not edit or commit it.
- Pages uses **Source: GitHub Actions**.

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
- `.githooks/pre-push`: blocks direct pushes to `main`; `make setup` installs
  it. The hook is a local convenience; GitHub rules remain the enforcement.
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
4. Docs job: `mkdocs build --strict` when documentation, the MkDocs config, or
   its requirements change.
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
| Paid cloud services | Free-tier GitHub Actions and Pages are enough; no paid cloud dependency (D-22) |
| Code-signing infrastructure | Manual APK install on four team-owned tablets |
| MDM / full kiosk provisioning | API 19 has no screen pinning; the supervised stand is accepted (ADR-0002) |
| Binary WebSocket protocol | JSON is enough for four players; revisit only if D-23 measurements fail |
| Auto-deploy of game or server releases | Manual festival packaging with a checklist is safer |

## Adding a New Tool

Before adding anything to the repository:

1. Name the trigger (component scaffold, failed measurement, repeated manual
   work).
2. Name the owner and where the tool runs (local hook, CI, manual before event).
3. Check the free tier and platform support (API 19, Linux/Windows).
4. If it changes the stack or the merge gate, add a dated decision/ADR first.
5. Update this file and `06-development/testing.md` in the same change.
