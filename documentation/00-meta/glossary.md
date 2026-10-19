# Glossary

Status: Draft
Last updated: 2026-10-19

Single definition place for domain terms. Use these spellings in code, docs, and UI.

| Term | Definition | Notes |
|------|------------|-------|
| System | The complete local setup: controllers + server + game + display. | — |
| Controller | Android tablet app used as a game input device. | One per player; input only, no game rendering. |
| Player | Person holding a controller in a session. | No account, no name. Identified by slot color and `P1`–`P4`. |
| Host | Person operating the server laptop and the display. | Starts, pauses, resumes, resets, and ends rounds; manages slots. Does not play in MVP. |
| Spectator | Person watching the big screen without a controller. | No interaction, no data. |
| Server | Go process that owns the session, slots, input validation, reconnects, and host commands. | Separate from the game process; see D-22. |
| Game | Godot 4 module that owns simulation, collisions, parking detection, scoring, and rendering. | Trusted local process; swappable behind the protocol. |
| Display / Screen | TV or monitor connected to the server laptop showing the game. | Whole map visible at once. |
| Session | One server run from startup to shutdown, containing zero or more rounds. | Holds transient state only; discarded at the end. |
| Round | One timed parking time attack followed by a results screen. | Initial MVP length: 180 s. |
| Lobby / Ready | Pre-round state in which tablets connect and receive slots. | Host starts the round from this state. |
| Slot | One of the four player places in a session. | Assigned by the server; has a color and `P1`–`P4` label. |
| Device token | Random identifier stored on a tablet and used to reclaim its slot after a reconnect. | Not a name, account, or personal profile. |
| Input event | Real-time controller message containing steering and throttle values. | Sent at 20–30 Hz while playing; validated by the server. |
| Parking task | Assignment to park a car in one target spot within the task time window. | Five per player per MVP round. |
| Target spot | Parking spot currently assigned to one player. | Exclusive among active players; highlighted in the player's color. |
| Time score | Part of the task score that decreases as the player takes longer. | `max(10, 100 - 3 * seconds)`. |
| Precision score | Bonus for center alignment and facing angle. | 0–40 points; see game design. |
| Collision penalty | Point deduction for hitting static environment geometry during a task. | −10 per collision, maximum −30. |
| Attract mode | Idle display state with title, instructions, and a start prompt. | Used between groups at the festival. |
| Minimum Festival Demo (MFD) | Smallest playable scope acceptable for the festival stand. | Four tablets, one task type, score, reconnect, runbook. |
| MVP | First complete playable scope: five tasks, precision, results, attract mode, recovery. | Not implemented yet. |
| Post-MVP | Work explicitly excluded from the first playable version. | Racing, drift, extra task types, web host panel. |
| Local network | Same-site network shared by controllers, server, and display. | No internet play for MVP. |
| Disconnect | Loss of a controller connection. | Car stops; slot is reserved; reconnect resumes control. |
| Reconnect | Controller reconnects and reclaims its slot using its device token. | Automatic; no QR or manual code in the normal flow. |
| Spike | Timeboxed technical experiment allowed to resolve a TBD. | Maximum one week; throwaway code; results documented (D-18). |
| Flanki Organizer | Original tournament-management concept, superseded by D-06 on 2026-10-16. | Historical only; old files removed from the working tree. |

## Rules

- Add a new term here before using it in other docs or code.
- Do not redefine a term in another file — link here instead.
- Do not reintroduce Flanki tournament terms without a new decision.
