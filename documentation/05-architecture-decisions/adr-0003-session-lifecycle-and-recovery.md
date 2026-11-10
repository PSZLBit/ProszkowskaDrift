# ADR-0003: Session Lifecycle, Reconnect, and Recovery

Status: Decided
Last updated: 2026-11-10
Date: 2026-11-09
Owner: Team

## Context

The festival uses four fixed, team-owned Android tablets as controllers and one
laptop running the Go server and Godot game. There are no accounts and no
internet play (D-12, D-21, D-25). Visitors may sleep, unplug, or restart a
tablet, and the venue Wi-Fi can drop. The system must recover without a developer
at the keyboard, while keeping slots stable for the current session.

This ADR records the connection and recovery model already decided in D-21 and
D-24. It does not change their behavior.

## Options Considered

1. **Lobby code or QR join per session (D-09, superseded).** More setup work on
   old cameras and tablets; a visitor-facing join flow adds no value for fixed
   team hardware.
2. **Auto-connect with a stored random token and slot reclaim (selected).**
   Tablets connect to the configured server automatically, identify themselves
   with a random token, and reclaim the same slot after a reconnect. QR or manual
   address entry remains a setup fallback only.
3. **Manual host re-pairing after every disconnect.** Simple to reason about,
   but it requires the host to intervene during a round and does not meet the
   unsupervised festival requirement.

## Decision

- Each tablet stores a random device token in app storage. The token is not a
  name, account, Android ID, serial number, or other personal data.
- The server maps the token to a slot and reserves that slot for the session.
  A reconnect with the same token reclaims the slot; a new device cannot take
  it while it is reserved.
- A disconnected controller's car coasts to a stop after the 500 ms input
  timeout. If the round is still running, control resumes when the same tablet
  reconnects.
- If all controllers disconnect, the round pauses after a 10 s grace period.
  The host can resume, reset, or end the session.
- Late join is allowed between rounds only. A new player starts the next round
  with zero score.
- A game crash returns the system to `Attract`; the server restarts Godot and
  the interrupted round is lost. A server crash also returns to `Attract`; the
  festival launcher restarts the server and game, and the round is lost. Tablets
  reconnect automatically. The Godot supervisor is tracked in #15; the
  server-crash launcher is part of the M3 festival runbook (#50).
- The host can kick a slot when a tablet is replaced or its token is lost. A
  kicked token is blocked for the rest of the session.
- QR or manual address entry is only a setup fallback, never the normal player
  flow.

## Consequences

- The server owns slot assignment, token validation, heartbeat tracking, and
  reconnect policy. Controller UI visibility is never an authorization check.
- The controller must persist one random token and reconnect with backoff. This
  is production controller work (#48).
- The server and game need explicit tests for disconnect, reconnect, all-
  disconnected pause, late join, kick, and crash recovery. Current hardware
  checks are pending in #49.
- The supervisor is required for unattended recovery; it is tracked in #15.
- No personal data is introduced: tokens are random, may appear truncated in
  diagnostics, and are discarded at session end.

## Validation Status (2026-11-10)

- Server token/slot reclaim and the all-disconnected pause path are implemented
  and covered by server tests.
- Controller auto-connect, token persistence, and reconnect are not merged; they
  are merged in PR #63 and tracked by #11 and #48.
- The Godot supervisor is in review in PR #59; the server-crash launcher remains
  M3 work in #50. Neither is proven by a manual crash drill yet.
- Real-tablet reconnect, Wi-Fi loss, and charging behavior have not been tested;
  they are tracked by #10, #14, and #49.

The decision remains accepted while validation is carried over. If a hardware
or supervisor check fails, record a new dated decision; do not silently replace
the connection model.

## Links

- D-21 (auto-connect and slot reclaim), D-24 (disconnect and recovery), D-15
  (superseded)
- [Players, sessions, and permissions](../03-players-sessions/roles-sessions.md)
- [Runtime architecture](../06-development/architecture.md)
- [Protocol](../06-development/protocol.md)
- [Testing strategy](../06-development/testing.md)
- [ADR-0001](adr-0001-mvp-architecture.md), [ADR-0002](adr-0002-legacy-android-controller.md)
