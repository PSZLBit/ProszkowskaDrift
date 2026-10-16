# Flanki Domain Overview

Status: Draft
Last updated: 2026-10-13

## Game Rules (TBD — Team Must Define)

The team has not yet fixed the official Flanki rules for this app.
Before implementing match logic, decide and document:

- Team size and roster rules (substitutions allowed?).
- Equipment (bottles, ball, field layout).
- Match win condition and scoring.
- Fouls, rematches, and dispute handling.

Do not implement scoring logic until this section reaches `Decided`.

## Tournament Lifecycle (Draft)

1. `Announced` — administrator or organizer publishes competition details.
2. `Registration open` — competitors register.
3. `Registration closed` — organizer freezes the participant list.
4. `Bracket generated` — organizer generates single-elimination bracket.
5. `In progress` — results are recorded per round.
6. `Completed` — winner is determined and published.

State transitions must be explicit in code once decided. No silent transitions.

## Single-Elimination Rules (Draft)

- Knockout: one loss eliminates a team.
- Participant count not a power of two: use byes. Seeding method is TBD
  (options: random, registration order, manual by organizer).
- Walkover: if a team does not show up, the opponent advances. Record it as a walkover, not as a played score.
- Third-place match: TBD whether MVP needs it.
- Score input: TBD who can enter results (organizer only vs. organizer + teams).

## Scale Notes

Design bracket generation and result entry to stay usable at 150 participants on a phone.
