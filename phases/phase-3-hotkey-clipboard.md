# Phase 3 — Hotkey & Clipboard

**Status:** [x] Done
**Depends on:** Phase 2

Make the app feel like a system utility — press a key, speak, press again, text appears.

## Hotkeys

| Action | Default | Behavior |
|---|---|---|
| Toggle Recording | ⌥Space | First press starts recording. Second press stops and transcribes. |
| Cancel Recording | Esc | Discards the active recording — nothing is transcribed or pasted. |
| Change Mode | ⌥⇧K | Cycles to the next mode. |

## Steps

- [x] 1. Global hotkey listener via `golang.design/x/hotkey` (key-down only)
- [x] 2. Toggle, Cancel, and Mode hotkeys registered; Cancel only active while recording
- [x] 3. `AXIsProcessTrustedWithOptions` accessibility check at startup — exit with instructions if not granted
- [x] 4. Event loop drives the state machine: Toggle in IDLE starts recording, Toggle in RECORDING stops and transcribes, Cancel in RECORDING discards
- [x] 5. Clipboard injection via CGo + NSPasteboard + CGEventPost (Cmd+V simulation): save current clipboard → write transcript → paste → restore original
- [x] 6. Menubar tray icon via `fyne.io/systray`: ⚫ idle, 🔴 recording, ⏳ processing — tray title shows active mode name
- [x] 7. Microphone device submenu in tray

## Deliverable

`internal/hotkey/` (`Manager` with `EnableCancel`/`DisableCancel`/`Rebind`), `internal/clipboard/` (CGo NSPasteboard + Cmd+V), `internal/ui/` (systray tray).

## Notes

- Clipboard save/restore prevents clobbering the user's in-progress copy
- Accessibility permission required — without it, hotkeys register but presses are silently swallowed
- Esc is registered only while recording; always-on Esc would conflict with other apps
