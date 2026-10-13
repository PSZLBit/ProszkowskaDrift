# Vision and Scope

Status: Draft
Last updated: 2026-10-13

## Vision

Flanki Organizer is a web app for organizing Flanki tournaments at speedcubing competitions.
It is inspired by the WCA competition platform and brings registrations, announcements,
and single-elimination brackets into one place that works on the go.

## Context

Participant counts at major championships grew from about 30 (Worlds 2023, Incheon)
to 100 (Euros 2024, Pamplona) and 150 (Euros 2026). Manual bracket management
does not scale to that size.

## MVP Scope (Planned, Not Implemented)

- Create an account with name, private date of birth, and country.
- Browse announced competitions and register to compete.
- Create and manage competitions as an organizer.
- Generate single-elimination tournament brackets.
- Administrators manage the website and announce competitions.

## Non-Goals for MVP

- No multi-format brackets (only single-elimination).
- No payment handling.
- No integration with external competition systems.
- No native mobile apps.

## Open Questions (TBD)

- Exact Flanki team size and match rules — must be defined in `02-flanki-domain/domain-overview.md`.
- Minimum age enforcement flow — see `03-users-permissions/roles-permissions-gdpr.md`.
- Technology stack, hosting, database — must be decided in `05-architecture-decisions/decision-log.md`.
