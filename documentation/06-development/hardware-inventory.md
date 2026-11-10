# Hardware Inventory

Status: Draft
Last updated: 2026-11-10

Inventory of the four festival tablets and any spares, collected for Sprint 0
(S0-02, issue #10). Devices are labelled `T1`–`T4`; slot labels `P1`–`P4` are
assigned only during festival provisioning.

**Collection status:** all four tablets are recorded and share the same
platform. Static specifications were read from Settings and the device
specification. The S0-02 short connectivity and charging check ran on
2026-11-10; the inventory owner reports a pass on T1–T4 with no blocker. The
30-minute Wi-Fi stability, latency, touch/readability, and spare checks remain
deferred to the measurements (#14) and the four-player MFD run (#49).

## Privacy Rules

- No owner names, accounts, serial numbers, IMEI, MAC addresses, or Android IDs.
- No photos of device labels or serial stickers.
- Record only model codes and anonymous device labels.
- Do not store Wi-Fi passwords in this file.

## Scope and Sensors

The controller app uses touch input and Wi-Fi only. Camera, accelerometer,
gyroscope, proximity/light sensors, battery-health telemetry, and any other
sensor not listed in this file are not read by the app, do not affect its
operation, and do not gate S0-02. Their rows are recorded as `Not required`
instead of `TBD`.

Battery health is kept only as a low-cost risk indicator for festival planning.
An unmeasured battery-health value does not block the controller; charging while
in use is covered by the operational check below.

## Method

1. Record static specs from Settings and the device specification.
2. Treat every tablet as the same platform: the team confirmed that T1–T4 have
   identical hardware and software.
3. Record assumptions explicitly; do not present untested behavior as verified.
4. Defer the 30-minute Wi-Fi, latency, touch, readability, and spare checks to
   the measurement and MFD steps that need them (#14, #49).
5. Mark unknown values `TBD`; never guess.

Tools used: Settings, device specification. Optional ADB commands for later
checks:

```text
adb shell getprop ro.product.model
adb shell getprop ro.build.version.sdk
adb shell cat /proc/meminfo
adb shell wm size
adb shell wm density
adb shell df /data
adb shell dumpsys battery
adb shell dumpsys sensorservice
```

## Summary

| Field | T1 | T2 | T3 | T4 | Spare |
|---|---|---|---|---|---|
| Model number | SM-T210 | SM-T210 | SM-T210 | SM-T210 | TBD |
| Manufacturer | Samsung | Samsung | Samsung | Samsung | TBD |
| Android version | 4.4.2 | 4.4.2 | 4.4.2 | 4.4.2 | TBD |
| API level | 19 | 19 | 19 | 19 | TBD |
| RAM | 800 MB | 800 MB | 800 MB | 800 MB | TBD |
| Internal storage total / free | 8.00 GB / 4.91 GB | 8.00 GB / 4.91 GB | 8.00 GB / 4.91 GB | 8.00 GB / 4.91 GB | TBD |
| Display | 7" 600×1024 TFT | 7" 600×1024 TFT | 7" 600×1024 TFT | 7" 600×1024 TFT | TBD |
| Wi-Fi bands | 2.4 GHz | 2.4 GHz | 2.4 GHz | 2.4 GHz | TBD |
| Battery rated capacity | 4000 mAh | 4000 mAh | 4000 mAh | 4000 mAh | TBD |
| Battery health | Not used; not measured | Not used; not measured | Not used; not measured | Not used; not measured | TBD |
| Risk flag | Medium | Medium | Medium | Medium | TBD |

## Common Platform (T1–T4)

| Field | Value |
|---|---|
| Model number | SM-T210 |
| Manufacturer | Samsung |
| Android version / API level | 4.4.2 / 19 |
| Kernel version | 3.4.5-4081557 dpi@SWDD5907 #1 Tue Feb 3 15:14:55 KST 2015 |
| Build number | KOT49H.T210XXBOB1 |
| CPU / ABI | TBD |
| RAM | 800 MB reported |
| Internal storage total | 8.00 GB |
| System storage | 2.72 GB |
| Internal storage used | 293 MB |
| Cached data | 2.92 MB |
| Miscellaneous files | 91.13 MB |
| Internal storage available | 4.91 GB |
| Display size | 7" |
| Display resolution | 600 × 1024 px |
| Display aspect ratio | 15.4:9 |
| Display panel | TFT |
| Touch | Multi-touch supported |
| Wi-Fi bands | 2.4 GHz |
| USB connector | micro-USB |
| Battery rated capacity | 4000 mAh |
| Camera(s) | Not required; not used by the controller app |
| Sensors (accelerometer, gyroscope, etc.) | Not required; not used by the controller app |
| Hardware buttons | TBD |
| Charger rating | TBD |

## Per-Device Condition and Tests

The static platform data above applies to all four tablets. The table below
records device-specific condition, the operational check that was run, and the
checks that are still open.

| Check | T1 | T2 | T3 | T4 |
|---|---|---|---|---|
| Short Wi-Fi connect + charging while in use (2026-11-10) | Pass (reported) | Pass (reported) | Pass (reported) | Pass (reported) |
| Physical condition (cracks, dead pixels, port, buttons) | Not inspected | Not inspected | Not inspected | Not inspected |
| Touch / multi-touch test | Not tested | Not tested | Not tested | Not tested |
| Screen readability at arm's length | Not tested | Not tested | Not tested | Not tested |
| Battery health | Not measured; not used | Not measured; not used | Not measured; not used | Not measured; not used |
| Battery level / voltage / temperature | Not measured | Not measured | Not measured | Not measured |
| Charging while screen on, 30 min | Not tested | Not tested | Not tested | Not tested |
| Wi-Fi idle, 30 min | Not tested; short check passed | Not tested; short check passed | Not tested; short check passed | Not tested; short check passed |
| Wi-Fi off/on reconnect | Not tested | Not tested | Not tested | Not tested |
| Keep-awake while charging | TBD | TBD | TBD | TBD |
| Developer options / USB debugging | TBD | TBD | TBD | TBD |
| Boots to home after reboot | TBD | TBD | TBD | TBD |
| Risk flag | Medium | Medium | Medium | Medium |

### Risk Notes

- **RAM:** 800 MB is reported. This is a low-end device for a controller app;
  validate app stability and touch latency in #48/#49. Keep the controller UI
  and WebSocket client minimal.
- **Wi-Fi:** 2.4 GHz only. This matches the architecture baseline. The short
  S0-02 connectivity check passed on 2026-11-10, but the 30-minute stability
  and reconnect checks have not been run; they must pass before the festival.
- **Battery:** rated capacity is 4000 mAh. The short check found no charging
  problem while the app was in use, and battery-health telemetry is not read by
  the app. The 30-minute charging and heat check remains in #14/#49; a tablet
  that loses charge under load is a festival blocker.
- **Display:** 7" 600×1024 TFT. Readability at arm's length and touch-target
  size must be validated with the real controller layout (#18).
- **Storage:** 4.91 GB free is sufficient for the APK and logs.

## Acceptance Checks

| Check | Pass condition | T1 | T2 | T3 | T4 |
|---|---|---|---|---|---|
| API level | Android 4.4.2 (API 19) confirmed | Pass | Pass | Pass | Pass |
| Connectivity + charging (short check) | Tablet connects over Wi-Fi and charges while the app is in use | Pass (reported) | Pass (reported) | Pass (reported) | Pass (reported) |
| Display | Landscape, readable at arm's length | Not tested | Not tested | Not tested | Not tested |
| Touch | Buttons respond; multi-touch works; no stuck touches | Not tested | Not tested | Not tested | Not tested |
| Wi-Fi | Connected for 30 min without drops | Short check passed; 30 min not tested | Short check passed; 30 min not tested | Short check passed; 30 min not tested | Short check passed; 30 min not tested |
| Reconnect | Wi-Fi off/on or app restart reconnects | Not tested | Not tested | Not tested | Not tested |
| Charging | Charges while screen is on and app is running | Pass (short check) | Pass (short check) | Pass (short check) | Pass (short check) |
| Heat | No shutdown or severe throttling in 30 min | Not tested | Not tested | Not tested | Not tested |
| Kiosk readiness | Developer options and launcher change available | TBD | TBD | TBD | TBD |
| Boot | Starts to home after reboot | TBD | TBD | TBD | TBD |

`Pass (reported)` means the inventory owner ran the check on 2026-11-10 and
reported the result; no per-device measurements were recorded. It satisfies the
short S0-02 check but not the 30-minute stability gate in S0-06 (#14).

## Risk Flags

Flag a device as **High** if any of these is true:

- API level is not 19.
- RAM is below 512 MB.
- The battery level drops while charging with the screen on.
- Wi-Fi drops during the 30-minute idle test.
- Touch or multi-touch has dead zones or stuck touches.

Battery-health telemetry is not read by the controller app (see
[Scope and Sensors](#scope-and-sensors), so a degraded or unmeasured health value
is not a device risk by itself; charging behavior is covered by the charging
check.

Flag a device as **Medium** if any of these is true:

- Reported RAM is below 1 GB.
- A required hardware check has not been performed yet.
- The screen is small or has cosmetic damage that does not affect touch.
- The charging port or cable is loose.
- Storage free space is below 100 MB.

2.4 GHz-only Wi-Fi is acceptable: it is the documented compatibility baseline
in `06-development/architecture.md`.

## Spares and Replacements

Owner: ZofiaDan. The spare set is tracked in #49.

- Spare tablet: TBD.
- Spare charger: TBD.
- Spare micro-USB cable: TBD.
- Replacement option: an Xbox 360/PC gamepad is the identified last-resort input
  option if the tablets cannot be used. It is not implemented and not part of
  the MVP: the server accepts controller WebSocket connections only, so a
  gamepad path needs a decision plus a host-side input bridge or a server change.
- Replacement cost: TBD; do not buy a gamepad unless the tablet path fails.

## Follow-ups

| Follow-up | Owner | Tracking |
|---|---|---|
| Install the controller APK on all four tablets | TracerDS | #48 |
| Run the 30-minute Wi-Fi, reconnect, and charging checks | ZofiaDan | #14, #49 |
| Measure latency and readability | ZofiaDan | #14 |
| Feed the device labels into the D-14 input-scheme playtest | TracerDS | #18 |
| Record any High risk in the roadmap risk register | ZofiaDan | #49 |
| Update ADR-0002 validation evidence when the checks are complete | ZofiaDan | #49, ADR-0002 |

## Collection Log

| Date | Device | What was recorded | By role |
|---|---|---|---|
| 2026-10-28 | T1–T4 | Model, Android/API, kernel, build, storage, RAM, display, Wi-Fi band, USB, battery capacity | QA/Controller |
| 2026-11-10 | T1–T4 | Short Wi-Fi connect and charging-while-in-use check; pass reported by the inventory owner | QA/Controller |
