# Hardware Inventory

Status: Draft
Last updated: 2026-10-28

Inventory of the four festival tablets and any spares, collected for Sprint 0
(S0-02, issue #10). Devices are labelled `T1`–`T4`; slot labels `P1`–`P4` are
assigned only during festival provisioning.

**Collection status:** 1 of 4 tablets recorded (T1 static data only). Hardware
checks and the remaining tablets are still open.

## Privacy Rules

- No owner names, accounts, serial numbers, IMEI, MAC addresses, or Android IDs.
- No photos of device labels or serial stickers.
- Record only model codes and anonymous device labels.
- Do not store Wi-Fi passwords in this file.

## Method

1. Charge each tablet to 100% and let it cool to room temperature.
2. Record static specs from Settings, the service menu, and the model spec sheet.
3. Run the checks below under identical conditions for all devices.
4. Run the 30-minute Wi-Fi and charging checks with the festival router,
   charger, and cable.
5. Mark unknown values `TBD`; never guess.

Tools used: Settings, Samsung service menus `*#0*#` and `*#*#4636*#*#`, router
client list. Optional ADB commands:

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
| Model number | SM-T210 | TBD | TBD | TBD | TBD |
| Manufacturer | Samsung | TBD | TBD | TBD | TBD |
| Android version | 4.4.2 | TBD | TBD | TBD | TBD |
| API level | 19 | TBD | TBD | TBD | TBD |
| RAM | TBD | TBD | TBD | TBD | TBD |
| Internal storage total / free | 8.00 GB / 4.91 GB | TBD | TBD | TBD | TBD |
| Display resolution / density | TBD | TBD | TBD | TBD | TBD |
| Wi-Fi bands (2.4 / 5 GHz) | TBD | TBD | TBD | TBD | TBD |
| Battery health | TBD | TBD | TBD | TBD | TBD |
| Risk flag | TBD | TBD | TBD | TBD | TBD |

## Device Details

### T1

| Field | Value |
|---|---|
| Model number | SM-T210 |
| Manufacturer | Samsung |
| Android version / API level | 4.4.2 / 19 |
| Kernel version | 3.4.5-4081557 dpi@SWDD5907 #1 Tue Feb 3 15:14:55 KST 2015 |
| Build number | KOT49H.T210XXBOB1 |
| CPU / ABI | TBD |
| RAM | TBD |
| Internal storage total | 8.00 GB |
| System storage | 2.72 GB |
| Internal storage used | 293 MB |
| Cached data | 2.92 MB |
| Miscellaneous files | 91.13 MB |
| Internal storage available | 4.91 GB |
| Display resolution / density / diagonal | TBD |
| Wi-Fi bands | TBD |
| Camera(s) | TBD |
| Sensors (accelerometer, gyroscope, etc.) | TBD |
| Hardware buttons | TBD |
| Charging port / cable type | TBD |
| Charger rating | TBD |
| Screen condition (cracks, dead pixels, ghost touch) | TBD |
| Touch / multi-touch test | TBD |
| Battery health | TBD |
| Battery level / voltage / temperature at start | TBD |
| Charging while screen on, 30 min | TBD |
| Wi-Fi idle, 30 min | TBD |
| Wi-Fi off/on reconnect | TBD |
| Keep-awake while charging | TBD |
| Developer options / USB debugging | TBD |
| Boots to home after reboot | TBD |
| Risk flag | TBD |
| Notes | Static data recorded from Settings on 2026-10-28. Hardware checks pending. |

### T2

Not recorded yet. Copy the T1 table when the device is inspected.

### T3

Not recorded yet. Copy the T1 table when the device is inspected.

### T4

Not recorded yet. Copy the T1 table when the device is inspected.

### Spare

| Field | Value |
|---|---|
| Available | TBD |
| Model number | TBD |
| Android version / API level | TBD |
| Condition | TBD |
| Notes | TBD |

## Acceptance Checks

| Check | Pass condition | T1 | T2 | T3 | T4 |
|---|---|---|---|---|---|
| API level | Android 4.4.2 (API 19) confirmed | Pass | TBD | TBD | TBD |
| Display | Landscape, readable at arm's length | TBD | TBD | TBD | TBD |
| Touch | Buttons respond; multi-touch works; no stuck touches | TBD | TBD | TBD | TBD |
| Wi-Fi | Connected for 30 min without drops | TBD | TBD | TBD | TBD |
| Reconnect | Wi-Fi off/on or app restart reconnects | TBD | TBD | TBD | TBD |
| Charging | Charges while screen is on and app is running | TBD | TBD | TBD | TBD |
| Heat | No shutdown or severe throttling in 30 min | TBD | TBD | TBD | TBD |
| Kiosk readiness | Developer options and launcher change available | TBD | TBD | TBD | TBD |
| Boot | Starts to home after reboot | TBD | TBD | TBD | TBD |

## Risk Flags

Flag a device as **High** if any of these is true:

- API level is not 19.
- RAM is below 1 GB.
- Battery health is `Poor`, `Overheat`, or `Dead`.
- The battery level drops while charging with the screen on.
- Wi-Fi drops during the 30-minute idle test.
- Touch or multi-touch has dead zones or stuck touches.

Flag a device as **Medium** if any of these is true:

- 5 GHz Wi-Fi is unsupported.
- The screen is small or has cosmetic damage that does not affect touch.
- The charging port or cable is loose.
- Storage free space is below 100 MB.

## Spares and Replacements

- Spare tablet: TBD.
- Spare charger: TBD.
- Spare micro-USB cable: TBD.
- Replacement option and cost: TBD.

## Follow-ups

- Record T2–T4 and all hardware checks.
- Install the controller APK on all four tablets when S0-03 (#11) produces it.
- Use these devices for the latency and Wi-Fi measurements in S0-06 (#14).
- Update the hardware matrix in `06-development/testing.md` (#17).
- Feed the device labels into the D-14 input-scheme playtest (#18).
- Record any High risk in the roadmap risk register.
- Update ADR-0002 validation evidence when the inventory is complete.

## Collection Log

| Date | Device | What was recorded | By role |
|---|---|---|---|
| 2026-10-28 | T1 | Model, Android version, kernel, build, storage | QA/Controller |
