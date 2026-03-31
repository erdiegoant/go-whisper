# Phase 9 — Native macOS UI (DarwinKit)

**Status:** ⏸ Postponed — work in progress on branch `feature/swiftui-rewrite` (see bottom of that branch's version of this file for details)
**Depends on:** Phase 8

Replace the placeholder tray icon with a proper native macOS UI. **Do not start until Phase 8 is stable.**

## Pre-Flight: DarwinKit API Verification

Before writing any Phase 9 code, verify these specific DarwinKit APIs are implemented (not stubs):
- [ ] `NSStatusItem` with variable-width title (for `● Cleanup` label next to icon)
- [ ] `NSPanel` with `NonactivatingPanelMask` (floating panel that never steals focus)
- [ ] `NSStatusItem` icon animation (pulse effect while recording)
- [ ] `NSMenu` with dynamic items (for mode switching dropdown)

If any are stubs, those specific parts will need raw CGo/Objective-C bridging — plan for it before starting.

## Architecture Reminder

All DarwinKit/AppKit calls **must happen on the main thread**. The goroutine architecture designed in Phase 1 handles this:
- Audio, hotkey, and transcription goroutines send state updates via channels
- The main thread (running `macos.RunApp`) consumes those channels and drives all UI updates
- Never call AppKit from a goroutine directly

---

## 9a — Menubar Icon

- [ ] 1. Add DarwinKit (`github.com/progrium/darwinkit`) to `go.mod`
- [ ] 2. Replace the `fyne-io/systray` placeholder with a native `NSStatusBar` item
- [ ] 3. Show current mode name next to the icon (e.g. `● Standard`, `● Translate`, `● Formal`)
- [ ] 4. Add a native AppKit dropdown menu with:
  - Mode list (tap to switch — replaces the hotkey cycle for mouse users)
  - Separator
  - "Open Config File" (opens `config.yaml` in default editor)
  - "Quit"
- [ ] 5. Animate the icon (pulse/fill effect) while `RECORDING` state is active
- [ ] 6. Show a spinner or different icon while `PROCESSING` state is active

## 9b — Floating Recording Popup

- [ ] 1. Create a borderless `NSPanel` that floats above all other windows
- [ ] 2. Show it immediately on toggle press (hotkey), hide it automatically after paste completes
- [ ] 3. Display three states visually inside the panel:
  - `RECORDING`: mic waveform animation (audio level bars driven by the capture goroutine via channel)
  - `PROCESSING`: spinner while Whisper transcription and Claude cleanup are running
  - `DONE`: brief checkmark on successful paste (~1s), then dismiss
- [ ] 4. Show current mode name inside the popup (e.g. "Standard", "Translate → EN", or the custom mode name)
- [ ] 5. Position bottom-center of screen, matching Superwhisper's style
- [ ] 6. Make it dismissible with Esc (cancel: discard buffer, hide panel, return to IDLE)
- [ ] 7. Set `panel.setSharingType(.none)` — exclude from screen recordings (privacy: hides dictated passwords from captures)
- [ ] 8. Set collection behavior: `[.canJoinAllSpaces, .fullScreenAuxiliary]` — prevents panel appearing in Mission Control / Exposé

## Panel Setup Example

```go
import (
    "github.com/progrium/darwinkit/macos"
    "github.com/progrium/darwinkit/macos/appkit"
    "github.com/progrium/darwinkit/macos/foundation"
)

macos.RunApp(func(app appkit.Application, delegate *appkit.ApplicationDelegate) {
    frame := foundation.Rect{Size: foundation.Size{Width: 300, Height: 80}}

    panel := appkit.NewWindowWithContentRectStyleMaskBackingDefer(
        frame,
        appkit.BorderlessWindowMask|appkit.NonactivatingPanelMask,
        appkit.BackingStoreBuffered,
        false,
    )
    panel.SetLevel(appkit.FloatingWindowLevel) // always on top
    panel.SetOpaque(false)
    panel.Center()
    panel.MakeKeyAndOrderFront(nil)
})
```

## 9c — Settings Panel (Optional)

- [ ] 1. Build a simple native `NSWindow` settings panel
- [ ] 2. Show config options as native form controls — dropdowns for model/language, text fields for hotkeys
- [ ] 3. Write changes back to `config.yaml` on save — the Phase 5 file watcher picks them up automatically (hot-reload)

## Deliverable

A polished, native-feeling macOS app indistinguishable from a Swift-built tool.

## Notes

- `NonactivatingPanelMask` is critical — the floating panel must never steal focus from the app the user is dictating into
- `setSharingType(.none)` prevents screen-recording tools from capturing the panel contents (passwords, sensitive dictation)
- 9c (Settings Panel) is optional — direct `config.yaml` editing with hot-reload is sufficient for MVP
- DarwinKit is CGo-based — expect longer compile times and macOS-only builds (no cross-compilation)
