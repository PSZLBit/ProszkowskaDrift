# ADR-0004: Server Round Identifier in `host_command start`

Status: Decided
Last updated: 2026-11-10
Date: 2026-11-10
Owner: Team

## Context

The server is the phase authority: it validates host requests and tells the
game when a round starts, pauses, resumes, resets, or ends (D-24). The MFD
baseline (D-38) requires deterministic per-round task assignment on the game
side: the starting spot rotates per round (`start = (roundId - 1) mod 4`),
task IDs derive from the round (`taskId = roundId * 10 + (spotIndex + 1)`),
and `round_result` reports the round. Previously `host_command start` carried
only the command, so the game had no server-owned round identifier to base
these values on.

## Options Considered

1. **Game-local round counter (rejected).** A counter kept by the game
   diverges from the server after a game crash and restart, a dropped
   connection, or a missed command. That breaks task-ID uniqueness across
   rounds and the fairness of the rotating assignment.
2. **Server-assigned `roundId` in `host_command start`, additive in protocol
   version 1 (selected).** The value arrives exactly when the round is
   accepted, so assignment, task IDs, and results share one authoritative
   identifier with no extra round trip.
3. **Separate round-announcement message (rejected).** A second message needs
   its own ordering and duplication handling for no benefit over a field on
   the command the game already applies.

## Decision

- `host_command` with `command: start` carries `roundId`: the server's new
  round identifier, an integer starting at 1 and incrementing once per
  accepted start.
- The game uses `roundId` for the assignment rotation, `taskId` derivation,
  and `round_result`. It must not keep its own round counter.
- The field is additive in protocol version 1: other commands omit it, the
  envelope and unknown-field policy are unchanged, and no version bump is
  required. The server remains the phase authority; `roundId` does not
  transfer timer ownership to the game.

## Consequences

- The MFD implementation (#46) adds the field on the server side and the
  game-side handling with validation: a `start` without a valid `roundId`
  is rejected under the existing protocol rules.
- Controllers are unaffected: this is a server-to-game message and the
  controller protocol is unchanged.
- If another round-scoped server-to-game field is needed later, extend this
  ADR with a dated note instead of recording a competing decision.

## Validation Status (2026-11-10)

- Documented only. Server emission, game handling, and tests are pending in
  the MFD implementation (#46). No code sends or reads `roundId` yet.

## Links

- D-38 (MFD gameplay baseline)
- [Game design](../02-game-domain/game-design.md)
- [Protocol](../06-development/protocol.md)
- [ADR-0001](adr-0001-mvp-architecture.md), [ADR-0003](adr-0003-session-lifecycle-and-recovery.md)
