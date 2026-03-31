# Phase 4 — Translation Flow

**Status:** [x] Done
**Depends on:** Phase 3

Add ES→EN translation as a second mode alongside Standard dictation.

## Steps

- [x] 1. Translation is a **mode** cycled by ⌥⇧K — no separate hotkey needed
- [x] 2. `internal/mode/` package: `Mode` struct (`Name`, `Language`, `Translate`), `All` slice, `Manager` with `Current`/`Next`/`SetByName`
- [x] 3. Two modes: **Standard** (`Language="auto"`, `Translate=false`) and **Translate** (`Language="es"`, `Translate=true`)
- [x] 4. Mode snapshotted at recording-stop time — a mid-flight mode change doesn't affect the in-progress transcription
- [x] 5. Active mode name shown in tray title (e.g. `⚫ Standard`, `🔴 Translate`)
- [x] 6. Last-used mode persisted to `state.json` and restored on next launch (implemented in Phase 5)

## Deliverable

Two working modes via ⌥⇧K. Tray title reflects active mode. Cancel works in both modes.

## Notes

- Whisper native translation only goes TO English — ES→EN is high quality via this path
- EN→ES is not implemented and not planned
- Mode name was changed from "Raw" to "Standard" during implementation
