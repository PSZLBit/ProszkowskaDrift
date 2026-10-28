# Festival Operations Runbook

Status: Draft
Last updated: 2026-10-28

This is the operational plan for running Proszkowska Drift at a science-club
festival stand. The game must work without a developer at the keyboard.

## Schedule

- Festival: Tuesday 2026-11-24.
- Internal festival-ready target: Wednesday 2026-11-18.
- 2026-11-19 – 2026-11-23: fixes, venue setup, and final rehearsal; no new
  features.

## Operating Goal

- A group of 1–4 visitors can play a three-minute round with minimal
  explanation.
- A host can run consecutive groups for several hours.
- A tablet, game, server, network, or display failure is recoverable without
  restarting the whole event.
- No personal data is collected or retained.

## Hardware Checklist

Per-device specs and test results are recorded in
[hardware-inventory.md](hardware-inventory.md).

- [ ] Laptop with the festival build, charger, and HDMI cable/adapter.
- [ ] TV or monitor with power cable and remote.
- [ ] Dedicated Wi-Fi router or laptop hotspot.
- [ ] Four Samsung tablets, charged, with the controller APK installed.
- [ ] Four chargers and micro-USB cables long enough for play.
- [ ] Power strip(s) and extension cord; tape for cable management.
- [ ] Tablet stands or cases that keep the charging port accessible.
- [ ] Spare tablet if available; spare USB cable and charger.
- [ ] Spare laptop or a recovery USB with the build, if available.
- [ ] Printed quick instructions and a QR/link to this runbook (optional).
- [ ] Small toolkit, labels, and a notepad for issues.
- [ ] Backup monitor if the TV is not reliable.

## Network

- Use a dedicated router or laptop hotspot. Do not depend on venue Wi-Fi.
- SSID and password are set by the team; keep the password simple for setup
  devices but not trivial.
- Disable client/AP isolation.
- Use a fixed laptop address (static IP or DHCP reservation), for example
  `192.168.50.10`.
- Use DHCP reservations for the four tablets so their addresses are stable, or
  configure the server address once in each app.
- Prefer 2.4 GHz for compatibility with the old tablets; use 5 GHz only if all
  four support it and the venue is congested.
- No internet is required. If the router provides internet, do not depend on it.
- Allow the server port (default `8080`) through the laptop firewall on the
  private network only.

## Tablet Provisioning

Before the event, on each tablet:

- [ ] Install the controller APK via `adb install -r`.
- [ ] Enable developer options and USB debugging for maintenance.
- [ ] Set the controller app as the default home/launcher.
- [ ] Disable the lock screen; set screen timeout to never or maximum.
- [ ] Set brightness high enough for the venue; keep auto-brightness off.
- [ ] Connect to the dedicated network and set the server address once.
- [ ] Confirm the app starts after reboot.
- [ ] Disable notifications and non-essential accounts.
- [ ] Label each tablet `P1`–`P4` to match its slot/color.
- [ ] Confirm charging works while the app is running.

Android 4.4.2 has no screen pinning. The default-launcher + immersive approach is
not a full kiosk; a team member must supervise the tablets.

## Laptop and Display

- [ ] Disable sleep, screensaver, automatic updates, and notifications.
- [ ] Set the TV/monitor as the only or primary display; use duplicate/extend
  mode deliberately.
- [ ] Launch the game fullscreen at 1920×1080 (or the display's native
  resolution with letterboxing).
- [ ] Set TV picture mode to game/low-latency if available; disable overscan.
- [ ] Set volume to an audible but not painful level.
- [ ] Confirm the host keyboard shortcuts work while the game window is focused.
- [ ] Confirm the laptop stays plugged in and does not throttle.

## Setup Sequence (Target: Under 15 Minutes)

1. Place TV/monitor, laptop, router, and tablets on the stand.
2. Connect power: laptop, TV, router, tablet chargers. Tape cables down.
3. Connect HDMI; set the display input and resolution.
4. Power the router; confirm the SSID and that the laptop has its fixed address.
5. Power the laptop; start the festival launcher (server + game).
6. Confirm the game enters `Attract` on the TV.
7. Power the tablets; confirm each connects and receives its slot color/label.
8. Run a one-player smoke test: countdown, movement, one task, score.
9. Run a four-player smoke test if the team is available.
10. Place printed instructions and confirm the queue area.

## Host Runbook

### Starting a Group

1. Confirm all tablets show `Ready` and the correct player colors.
2. Give a 10-second explanation: "Drive to your colored spot, park inside it,
   score points. Three minutes."
3. Press `Enter` to start. The display counts down.
4. Stay at the laptop; watch for disconnects and stuck players.

### During a Round

- Do not press host keys accidentally; the game window must have focus.
- If a player is stuck, let auto-skip handle it or use the host skip/kick if
  implemented.
- If a tablet disconnects, wait for auto-reconnect; the car stops.
- If the display freezes, use the failure playbook.

### Ending a Round

1. Results appear automatically after the timer or when all tasks are done.
2. Announce the winner briefly.
3. Press `Enter` for the next group, or `R` to reset and return to `Attract`.
4. Confirm scores are cleared between groups.
5. Sanitize/wipe the tablets if required by the venue; at minimum, return them
   to the stand.

## Failure Playbook

| Symptom | First action | Recovery |
|---------|--------------|----------|
| One tablet disconnected | Wait 5 s for auto-reconnect | Reopen app; check Wi-Fi; if still broken, host kicks slot and continues with fewer players |
| Tablet app frozen | Force-stop and reopen the app | Reconnect; if it fails, reboot tablet and reopen |
| Game window frozen | Check server log; restart the game process | Supervisor should restart; otherwise use the launcher restart action |
| Server crashed | Check launcher window/log | Restart server; tablets reconnect; current round is lost |
| Router down | Check power and cables | Restart router; if it does not recover, switch to laptop hotspot and reconfigure tablets |
| TV black/no signal | Check HDMI and input | Switch cable/input; use backup monitor |
| Laptop overheating/throttling | Improve ventilation; reduce brightness | Move to backup laptop with the same build |
| No audio | Check TV volume and game audio | Continue silently; audio is optional |
| Power outage | Check power strip/breakers | Restart in setup order; current round is lost |

## Queue and Throughput

- Round: 3 minutes. Turnover: about 1 minute. Target: 12–15 groups per hour.
- Use a simple physical queue or a sign-up sheet with slot labels, no personal
  data beyond a first name if the venue requires it; prefer queue positions.
- If demand is high, shorten rounds to 2 minutes and/or reduce tasks to 4;
  record the change as a tuning decision, not a silent code change.
- Keep one team member free to talk to visitors and manage the queue.
- Do not let visitors handle the laptop or router.

## Privacy and Reset

- No accounts, names, photos, or persistent scores.
- Optional event scores are anonymous and in-memory; clear them between groups.
- Do not log device tokens in full; truncate them.
- Do not take photos of visitors without consent.
- At pack-up, close the server and discard session state.

## Event-Day Roles

| Role | Responsibility |
|------|----------------|
| Host/operator | Runs rounds, keyboard, queue, announcements |
| Floater/tech | Tablets, router, cables, recovery, spare hardware |
| Queue/PR | Explains the game, manages waiting visitors, answers questions |

With four team members, rotate roles every 30–45 minutes to avoid fatigue.

## Rehearsal Checklist

- [ ] Cold start from packed boxes in under 15 minutes.
- [ ] Four-player round with people who have never played.
- [ ] Disconnect/reconnect one tablet mid-round.
- [ ] Kill the game process; confirm restart and reconnect.
- [ ] Kill the server; confirm supervisor restart and reconnect.
- [ ] Simulate router failure; execute the hotspot fallback.
- [ ] Run continuously for 30–60 minutes; inspect heat, battery, logs.
- [ ] Reset between groups; confirm scores and tasks are cleared.
- [ ] Pack up in under 15 minutes.

## Open Operational Items

- Venue and available setup time (date confirmed: 2026-11-24).
- Laptop ownership and specifications.
- TV/monitor model, resolution, and input lag.
- Router model/budget and backup hotspot plan.
- Tablet stands, chargers, and cable lengths.
- Spare tablet/laptop availability.
- Venue rules for power, network, noise, and data.
