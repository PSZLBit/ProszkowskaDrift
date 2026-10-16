# Roles, Permissions, and GDPR

Status: Draft
Last updated: 2026-10-13

## Roles

- `Competitor` — registered user. Can browse competitions and register for them.
  Can view own registration and public brackets.
- `Organizer` — can create and manage own competitions, manage registrations,
  generate brackets, and record results.
- `Administrator` — site-level. Can manage the website and announce competitions
  to competitors. Can manage users and competitions when needed.

A user can hold multiple roles. Every privileged action must be authorized on the server.
Never rely on UI visibility alone.

## Permission Matrix (Draft)

| Action | Competitor | Organizer | Administrator |
|--------|:----------:|:---------:|:-------------:|
| Browse announced competitions | yes | yes | yes |
| Register for competition | yes (self only) | yes | yes |
| Create competition | no | yes | yes |
| Edit own competition | no | yes (own only) | yes |
| Manage registrations | no | yes (own only) | yes |
| Generate bracket | no | yes (own only) | yes |
| Record match result | TBD | yes (own only) | yes |
| Announce competition site-wide | no | no | yes |
| View private date of birth | self only | no | only for age verification, never listed |

`TBD` rows must be decided before implementation.

## GDPR and 18+ Policy

Flanki is an alcohol-related game. This app must be GDPR-compliant:

- Date of birth is private personal data. Never expose it in public competition views,
  brackets, APIs, or logs. Show at most an `18+ verified` flag where needed.
- Enforce 18+ eligibility at registration. Exact verification flow is TBD;
  do not store more data than needed.
- Minimize personal data: name, country, and age-verification status only, unless the team decides otherwise via ADR.
- Provide account deletion and data export paths in the future requirements.
- Treat all account details as personal information in UI, API responses, and logs.
