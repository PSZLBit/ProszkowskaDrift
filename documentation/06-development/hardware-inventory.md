# Hardware Inventory

Status: Draft
Last updated: 2026-10-28

Inventory of the four festival tablets and any spares, collected for Sprint 0
(S0-02, issue #10). Devices are labelled `T1`–`T4`; slot labels `P1`–`P4` are
assigned only during festival provisioning.

**Collection status:** all four tablets are recorded and share the same
platform. Static specifications were read from Settings and the device
specification. Hardware checks (Wi-Fi stability, charging under load, battery
health, touch, readability) were not performed and are deferred to the S0-03
controller spike (#11) and the S0-06 measurements (#14).

## Privacy Rules

- No owner names, accounts, serial numbers, IMEI, MAC addresses, or Android IDs.
- No photos of device labels or serial stickers.
- Record only model codes and anonymous device labels.
- Do not store Wi-Fi passwords in this file.

## Method

1. Record static specs from Settings and the device specification.
2. Treat every tablet as the same platform: the team confirmed that T1–T4 have
   identical hardware and software.
3. Record assumptions explicitly; do not present untested behavior as verified.
4. Defer Wi-Fi, charging, battery-health, touch, and readability checks to the
   spike that needs them (#11, #14, #17).
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
| Battery health | Not measured | Not measured | Not measured | Not measured | TBD |
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
| Camera(s) | TBD |
| Sensors (accelerometer, gyroscope, etc.) | TBD |
| Hardware buttons | TBD |
| Charger rating | TBD |

## Per-Device Condition and Tests

The static platform data above applies to all four tablets. The table below
records device-specific condition and the checks that are still open.

| Check | T1 | T2 | T3 | T4 |
|---|---|---|---|---|
| Physical condition (cracks, dead pixels, port, buttons) | Not inspected | Not inspected | Not inspected | Not inspected |
| Touch / multi-touch test | Not tested | Not tested | Not tested | Not tested |
| Screen readability at arm's length | Not tested | Not tested | Not tested | Not tested |
| Battery health | Not measured | Not measured | Not measured | Not measured |
| Battery level / voltage / temperature | Not measured | Not measured | Not measured | Not measured |
| Charging while screen on, 30 min | Not tested | Not tested | Not tested | Not tested |
| Wi-Fi idle, 30 min | Not tested; assumed working | Not tested; assumed working | Not tested; assumed working | Not tested; assumed working |
| Wi-Fi off/on reconnect | Not tested | Not tested | Not tested | Not tested |
| Keep-awake while charging | TBD | TBD | TBD | TBD |
| Developer options / USB debugging | TBD | TBD | TBD | TBD |
| Boots to home after reboot | TBD | TBD | TBD | TBD |
| Risk flag | Medium | Medium | Medium | Medium |

### Risk Notes

- **RAM:** 800 MB is reported. This is a low-end device for a controller app;
  validate app stability and touch latency in S0-03 (#11). Keep the controller
  UI and WebSocket client minimal.
- **Wi-Fi:** 2.4 GHz only. This matches the architecture baseline, but the
  30-minute stability and reconnect checks have not been run. They must pass
  before the festival; treat the current state as an assumption, not evidence.
- **Battery:** rated capacity is 4000 mAh, but health and charging under load
  are unknown. Verify that each tablet charges while the screen is on in #11;
  a tablet that loses charge under load is a festival blocker.
- **Display:** 7" 600×1024 TFT. Readability at arm's length and touch-target
  size must be validated with the real controller layout (#18).
- **Storage:** 4.91 GB free is sufficient for the APK and logs.

## Acceptance Checks

| Check | Pass condition | T1 | T2 | T3 | T4 |
|---|---|---|---|---|---|
| API level | Android 4.4.2 (API 19) confirmed | Pass | Pass | Pass | Pass |
| Display | Landscape, readable at arm's length | Not tested | Not tested | Not tested | Not tested |
| Touch | Buttons respond; multi-touch works; no stuck touches | Not tested | Not tested | Not tested | Not tested |
| Wi-Fi | Connected for 30 min without drops | Assumed; not tested | Assumed; not tested | Assumed; not tested | Assumed; not tested |
| Reconnect | Wi-Fi off/on or app restart reconnects | Not tested | Not tested | Not tested | Not tested |
| Charging | Charges while screen is on and app is running | Not tested | Not tested | Not tested | Not tested |
| Heat | No shutdown or severe throttling in 30 min | Not tested | Not tested | Not tested | Not tested |
| Kiosk readiness | Developer options and launcher change available | TBD | TBD | TBD | TBD |
| Boot | Starts to home after reboot | TBD | TBD | TBD | TBD |

## Risk Flags

Flag a device as **High** if any of these is true:

- API level is not 19.
- RAM is below 512 MB.
- Battery health is `Poor`, `Overheat`, or `Dead`.
- The battery level drops while charging with the screen on.
- Wi-Fi drops during the 30-minute idle test.
- Touch or multi-touch has dead zones or stuck touches.

Flag a device as **Medium** if any of these is true:

- Reported RAM is below 1 GB.
- A required hardware check has not been performed yet.
- The screen is small or has cosmetic damage that does not affect touch.
- The charging port or cable is loose.
- Storage free space is below 100 MB.

2.4 GHz-only Wi-Fi is acceptable: it is the documented compatibility baseline
in `06-development/architecture.md`.

## Spares and Replacements

- Spare tablet: TBD.
- Spare charger: TBD.
- Spare micro-USB cable: TBD.
- Replacement option and cost: TBD.

## Follow-ups

- Install the controller APK on all four tablets when S0-03 (#11) produces it.
- Run the 30-minute Wi-Fi, reconnect, and charging checks during #11 and #14.
- Measure latency and readability in S0-06 (#14).
- Update the hardware matrix in `06-development/testing.md` (#17).
- Feed the device labels into the D-14 input-scheme playtest (#18).
- Record any High risk in the roadmap risk register.
- Update ADR-0002 validation evidence when the hardware checks are complete.

## Collection Log

| Date | Device | What was recorded | By role |
|---|---|---|---|
| 2026-10-28 | T1–T4 | Model, Android/API, kernel, build, storage, RAM, display, Wi-Fi band, USB, battery capacity | QA/Controller |
