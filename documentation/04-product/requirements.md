# Product Requirements

Status: Draft
Last updated: 2026-10-13

All items below are planned and not implemented unless marked otherwise.

## Functional Requirements

1. `FR-01` Accounts: a user can create an account with name, date of birth (private), and country.
2. `FR-02` Competitions list: any visitor can browse announced competitions.
3. `FR-03` Registration: a competitor can register for a competition and view own registration.
4. `FR-04` Competition management: an organizer can create, edit, and cancel own competitions.
5. `FR-05` Bracket generation: an organizer can generate a single-elimination bracket from the participant list.
6. `FR-06` Results: an authorized user can record match results and advance winners. Exact authorized roles TBD.
7. `FR-07` Announcements: an administrator can announce competitions to competitors.

## Non-Functional Requirements

- `NFR-01` GDPR compliance for personal data, especially dates of birth.
- `NFR-02` Server-side authorization for every privileged action.
- `NFR-03` Usable on mobile during live events with 100+ participants.
- `NFR-04` English-only UI and content for MVP.

## Out of Scope for MVP

Multiple bracket formats, payments, external integrations, native apps.
See `01-vision-scope/vision-scope.md`.
