# Phase 8 — Polish & Reliability

**Status:** [x] Complete
**Depends on:** Phase 7

Make it production-quality for daily use.

## What's already done (don't redo)

- **Makefile** — all targets exist: `build`, `run`, `dev`, `test`, `install`, `download-model`, `rectest`, `whisper`, `clean`
- **Accessibility permission check** — handled at startup (Phase 3)
- **Hotkey conflict detection** — `warnConflicts` logs warnings (Phase 5)
- **Claude API unavailable** — already falls back to raw transcript with a log line (Phase 6)
- **Cleanup toggle** — tray menu item, persisted in state.json (Phase 7 session)

## Steps

- [x] 1. **Audio feedback** — short system sound on recording start/stop
  - Start: `afplay /System/Library/Sounds/Tink.aiff`
  - Stop/paste: `afplay /System/Library/Sounds/Pop.aiff`
  - Cancel: `afplay /System/Library/Sounds/Basso.aiff`
  - Run each async (goroutine) so playback never blocks recording
  - Add `sound_enabled: true` to config (default on); respect it before playing

- [x] 2. **Desktop notifications** — show transcribed text after paste
  - Use `osascript -e 'display notification ...'` (no entitlement needed, works out of the box)
  - Truncate to first 100 chars in the notification subtitle
  - Add `notifications_enabled: true` to config; skip if false
  - Useful when the paste landed in the wrong window

- [x] 3. **Long recording chunking** — handle recordings over 30s gracefully
  - Chunk audio into 25s segments with a 5s overlap (prevents mid-word cuts at boundaries)
  - Transcribe each chunk, deduplicate the overlapping tail/head between consecutive results, concatenate
  - Respect `max_recording_seconds` from config as hard cap (default: 120s)
  - Log chunk count and total duration when chunking kicks in

- [x] 4. **Structured logging to file**
  - Use `log/slog` (Go 1.21 stdlib) with JSON handler writing to `~/.config/gowhisper/gowhisper.log`
  - Add log rotation: cap at 10MB, keep 3 files — use `gopkg.in/natefinish/lumberjack.v2`
  - Respect `log_level: info/debug` from config
  - Keep existing `log.Printf` calls or replace with slog — consistency matters more than perfection

- [x] 5. **Error handling gaps**
  - Mic unavailable at `capturer.Start()` → log + stay IDLE (already partially handled; verify message is clear)
  - Model file missing → exit with message pointing to `make download-model` (currently crashes with an opaque error)
  - Config file unwritable → log warning, continue with in-memory state

- [x] 7. **Tests for all new code in this phase**
  - `sound_enabled` / `notifications_enabled` config parsing via `applyDefaults` patterns
  - Chunking logic (pure function — no CGo dependency): split/overlap/deduplicate
  - Any new config fields added to state.go

## Concrete test scenarios (manual)

- [ ] 10 English sentences across Standard and any custom modes
- [ ] Spanish dictation in Translate mode — confirm ES→EN output
- [ ] Rapid double-press of ⌥Space — second press must be ignored during PROCESSING
- [ ] Press Esc while recording — buffer discarded, nothing pasted
- [ ] Disable cleanup from tray, dictate — confirm raw transcript pasted (no Claude call)
- [ ] Record longer than 30s — confirm chunking produces coherent stitched output
- [ ] Change hotkey in config while running — confirm new hotkey works without restart
- [ ] Remove API key from config while running — confirm graceful fallback to raw transcript

## Deliverable

Stable, daily-driveable app: audio cues, desktop notifications, long-recording support, file logging, and all new logic covered by unit tests.

## Notes

- No Ollama — Claude API only; remove any remaining Ollama references found in code
- `afplay` is a macOS built-in at `/usr/bin/afplay` — no Homebrew dependency
- `osascript` notifications don't require entitlements, unlike `UNUserNotificationCenter` (avoid CGo/framework complexity)
- Chunking overlap at 25+5s is a heuristic — tune if real-world results show seam artifacts
- Log file path should be printed to stdout on first launch so users can find it
- **After completing this phase: audit all packages for untested logic and backfill tests**
