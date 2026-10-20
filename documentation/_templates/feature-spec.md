# Feature Spec: <Name>

Status: Draft
Last updated: YYYY-MM-DD
Owner: Name

## Goal

One paragraph: what user problem does this solve?

## Users and Roles

Who can use it (Player / Host / Spectator)? Link to `03-players-sessions/roles-sessions.md`.

## Requirements

- [ ] Requirement 1 (link to `04-product/requirements.md` ID when possible).
- [ ] Requirement 2.

## Domain Rules

Round states, scoring rules, validation rules, edge cases. Link to
`02-game-domain/game-design.md` and the relevant decision. Do not invent a
mechanic that is not in the game design or decision log.

## Authorization

Who can do what, enforced on the server. See `03-players-sessions/roles-sessions.md`.

## Privacy

What session data is touched; confirm no accounts or personal data are retained.
See D-25.

## Target Hardware

Which component is affected (server, game, controller, ops) and which real
hardware/network must be used for verification (Android 4.4.2 tablet, target
laptop/TV, router).

## Test Approach

Automated tests, manual hardware checks, and the acceptance criteria that prove
the feature. Link to `06-development/testing.md`.

## Docs Update Checklist

- [ ] Glossary updated if new terms were added.
- [ ] Requirements updated.
- [ ] Decision log or ADR updated if a decision changed.
- [ ] `Last updated` dates bumped.
