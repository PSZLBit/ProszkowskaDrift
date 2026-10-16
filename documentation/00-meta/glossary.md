# Glossary

Status: Draft
Last updated: 2026-10-13

Single definition place for domain terms. Use these spellings in code, docs, and UI.

| Term | Definition | Notes |
|------|------------|-------|
| Flanki | Polish team drinking game; tournaments run after speedcubing competitions. Exact rules TBD — see `02-flanki-domain/domain-overview.md`. | Keep rules in domain file, not here. |
| Competition | One announced Flanki event (place, date, organizer, participant list). | Inspired by WCA competition model. |
| Tournament | Single-elimination bracket run within a competition. | One competition may host one tournament for MVP. |
| Competitor | Registered user who can browse competitions and register to compete. | — |
| Organizer | User who can create and manage competitions and brackets. | — |
| Administrator | Site-level user who can manage the website and announce competitions. | — |
| Single-elimination | Knockout format: a team is out after one loss. | MVP format. |
| Bye | Automatic advance when participant count is not a power of two. | Needs seeding rule, TBD. |
| Seeding | Initial bracket placement method (e.g. random, registration order). | TBD. |
| Walkover | Win by default when the opponent does not show up or is disqualified. | TBD handling. |
| WCA | World Cube Association. Reference model for competitions, not a dependency. | — |
| GDPR | EU data protection regulation. Dates of birth are private personal data. | See `03-users-permissions/roles-permissions-gdpr.md`. |

## Rules

- Add a new term here before using it in other docs or code.
- Do not redefine a term in another file — link here instead.
