# Phase 3 — Hotkey & Clipboard

**Status:** [ ] Not Started
**Depends on:** Phase 2

Make the app feel like a system utility — press a key, speak, press again, text appears.

## Recording Model

**Toggle + Cancel** (matching Superwhisper's default behavior):

| Action | Default Hotkey | Behavior |
|---|---|---|
| Toggle Recording | ⌥Space (Option+Space) | First press starts recording. Second press stops recording and triggers transcription. |
| Cancel Recording | Esc | Discards the active recording buffer and returns to IDLE — nothing is transcribed or pasted. |
| Change Mode | ⌥⇧K (Option+Shift+K) | Cycles to the next mode. Works at any time, including while recording. |

No push-to-talk. No automatic stop-on-silence. Recording always starts and stops manually.

## Steps

- [ ] 1. Add global hotkey listener using `golang.design/x/hotkey`
  - Key-down events only are sufficient for toggle behavior — no key-up needed
- [ ] 2. Implement the three hotkeys above with hardcoded defaults (they become configurable in Phase 5):
  - Toggle: `hotkey.KeySpace` + `hotkey.ModOption`
  - Cancel: `hotkey.KeyEscape` (no modifier)
  - Change Mode: `hotkey.KeyK` + `hotkey.ModOption + hotkey.ModShift`
- [ ] 3. Check `AXIsProcessTrustedWithOptions` at startup — if Accessibility access is not granted, show a prompt and open System Settings > Privacy > Accessibility. Without this, hotkeys and simulated paste will silently fail.
- [ ] 4. Implement the recording state machine in `internal/hotkey/`:
  - Toggle press in `IDLE` → start audio capture goroutine, transition to `RECORDING`
  - Toggle press in `RECORDING` → stop capture, hand buffer to transcription goroutine, transition to `PROCESSING`
  - Cancel press in `RECORDING` → stop capture, discard buffer, transition to `IDLE`
  - Ignore toggle/cancel presses while in `PROCESSING` state (transcription in flight)
- [ ] 5. Implement clipboard injection using `golang.design/x/clipboard`:
  - **Before** writing: save current clipboard contents
  - Write transcribed text to clipboard
  - Simulate **Cmd+V** (not Ctrl+V) to paste into the active window
  - **After** paste event: restore original clipboard contents
- [ ] 6. Add a minimal menubar tray icon using `github.com/fyne-io/systray` as a placeholder — gets replaced in Phase 9
  - Show a static icon in `IDLE` state
  - Show a different icon (e.g. filled circle) while `RECORDING`
  - Show a spinner or different icon while `PROCESSING`
- [ ] 7. Test: press ⌥Space, dictate a sentence in English, press ⌥Space again, confirm it pastes into a text editor
- [ ] 8. Test: press ⌥Space, dictate something, press Esc, confirm nothing is pasted and clipboard is unchanged

## Deliverable

Background app that listens for toggle/cancel hotkeys, records, transcribes, and pastes into any active window. Cancel discards cleanly.

## Notes

- `golang.design/x/hotkey` is sufficient — push-to-talk (key-up) is not needed for toggle recording
- Use `fyne-io/systray` (not `getlantern/systray`) — it is the actively maintained fork with fewer macOS issues
- Accessibility permission is not just a warning — without it, `golang.design/x/hotkey` registers the hotkey but presses are silently swallowed on macOS
- Clipboard save/restore prevents clobbering the user's in-progress copy — important for daily use
- macOS paste shortcut is Cmd+V, not Ctrl+V
- The Change Mode hotkey (⌥⇧K) just cycles the current mode index for now — full mode system comes in Phase 7
