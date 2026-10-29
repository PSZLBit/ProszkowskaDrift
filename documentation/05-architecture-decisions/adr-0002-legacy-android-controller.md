# ADR-0002: Legacy Android Controller

Status: Decided
Last updated: 2026-10-28
Date: 2026-10-19
Owner: Team

## Context

The team owns four identical Samsung tablets running **Android 4.4.2 (API 19)**.
They are the only controllers available. The tablets are old, batteries vary in
condition, and the team has no prior Android development experience. The app
must be installable without Google Play, must stay connected to a local Go
server, and must be usable by visitors at a festival stand.

Android 4.4.2 rules out most modern Android choices:

- Compose requires a newer minimum SDK.
- Many AndroidX libraries require API 21 or newer.
- OkHttp 4 requires API 21; Flutter, React Native, and .NET MAUI also require
  API 21 or newer.
- Screen pinning (`startLockTask`) was introduced in Android 5.0 and is not
  available.
- The system WebView is very old and unsuitable for a modern web controller.

## Options Considered

1. **Minimal native Kotlin app (selected).** `minSdk 19`, plain Android Views,
   no Compose, no AndroidX, minimal dependencies. Small surface area; direct
   control over WebSocket, keep-awake, immersive mode, and boot behavior.
2. **Minimal native Java app.** Similar compatibility, but Kotlin is the
   project's decided controller language (D-08) and is more concise. Keep as a
   fallback if the Kotlin toolchain cannot target API 19 reliably.
3. **Web controller in the old Android browser/WebView.** No app install, but
   unreliable WebSocket behavior, old rendering, no keep-awake/screen pinning,
   and poor kiosk control. Not acceptable as the primary path.
4. **Godot Android export.** Adds a full game runtime for four buttons, larger
   APK, and uncertain API 19 support. Rejected.
5. **Replace the tablets.** Best long-term reliability, but conflicts with the
   goal of using the available hardware and adds cost. Keep as a last-resort
   fallback if the spike fails.

## Decision

Build a **minimal Kotlin Android controller** with these constraints:

- `minSdk 19`, landscape orientation, one Activity.
- Plain Android Views or a custom `View`; **no Compose, no AppCompat, no
  AndroidX** unless a specific library is proven to support API 19.
- WebSocket via `Java-WebSocket` or OkHttp 3.12 (not OkHttp 4); JSON via the
  platform `org.json` parser.
- Input sent at 20–30 Hz while a round is active. Reconnect with exponential
  backoff and a stored random device token.
- Keep screen awake, immersive/fullscreen mode, no lock screen, app registered
  as a home launcher, and a `BOOT_COMPLETED` receiver to relaunch after a reboot.
- Server address configured once by the team; no per-player QR flow. A hidden
  setup screen is acceptable.
- Large touch targets, high contrast, player color, current task, score, and
  round time. No map rendering.

**Kiosk limitations:** Android 4.4.2 has no screen pinning. The stand must rely
on the app as the default launcher, immersive mode, disabled lock screen,
keep-awake while charging, and team supervision. This is acceptable for a
supervised festival stand but not a fully locked kiosk.

## Consequences

- The controller UI must stay simple; no modern animation or design system.
- Dependencies must be checked against API 19 before adding them.
- APK installation and updates are manual via `adb` before the event.
- Tablets must remain plugged in during the festival; battery condition must be
  tested.
- The app cannot be fully locked down without newer Android or device-owner
  provisioning.
- The server must handle reconnects gracefully because the tablets may sleep,
  drop Wi-Fi, or be restarted by visitors.

## Validation Gate (Sprint 0)

1. Build the minimal app with the chosen Android Gradle Plugin/Kotlin versions.
2. Install it on all four tablets and confirm it launches after reboot.
3. Send WebSocket input to the Go server from all four tablets simultaneously.
4. Measure touch responsiveness and input-to-display latency.
5. Confirm charging, screen-on behavior, and immersive mode during a 30-minute
   test.
6. Record device specs and the results in
   [`hardware-inventory.md`](../06-development/hardware-inventory.md).

If Kotlin/AGP cannot target API 19 reliably, try a minimal Java app. If the
native path fails entirely, record a new decision before replacing the tablets
or moving to a web controller.

## Links

- D-20 (Android 4.4.2 constraint), D-08 (Kotlin controller), D-21 (connection
  model)
- `05-architecture-decisions/adr-0001-mvp-architecture.md`
- `06-development/architecture.md`
- `03-players-sessions/roles-sessions.md`
