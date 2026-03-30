# Phase 8 — Polish & Reliability

**Status:** [ ] Not Started
**Depends on:** Phase 7

Make it production-quality for daily use.

## Steps

- [ ] 1. Add audio feedback — a short system sound on recording start and stop
  - Use `afplay /System/Library/Sounds/Tink.aiff` (start) and `afplay /System/Library/Sounds/Pop.aiff` (stop)
  - Run async so audio playback doesn't block the recording goroutine
- [ ] 2. Implement proper error handling throughout:
  - Mic not found or unavailable → alert via desktop notification, stay in IDLE
  - Model file missing or failed to load → log error and exit with a helpful message pointing to `make download-model`
  - Ollama unavailable → already handled in Phase 6 (graceful fallback)
  - Hotkey conflict detected in config → log warning, last definition wins
  - Accessibility permission not granted → prompt at startup (already handled in Phase 3)
- [ ] 3. Add a desktop notification showing the transcribed text after paste:
  - Request notification permission via `UNUserNotificationCenter` at first launch (required macOS 10.14+)
  - Fall back to `osascript -e 'display notification ...'` if permission not granted
  - Notification is useful when paste went to the wrong window
  - Respect `notifications_enabled` from config
- [ ] 4. Handle long recordings gracefully — chunk audio into overlapping segments:
  - Use **25-second chunks with a 5-second overlap** (not hard 30s splits — hard splits cut sentences mid-word)
  - Transcribe each chunk, deduplicate the overlapping region between consecutive chunks, concatenate results
  - Respect `max_recording_seconds` from config as the hard cap (default: 120s)
- [ ] 5. Add structured logging to `~/.config/gowhisper/gowhisper.log`:
  - Use `log/slog` (stdlib, Go 1.21+) with JSON output
  - Add log rotation: cap at 10MB, keep last 3 files — use `gopkg.in/natefinish/lumberjack.v2`
  - Respect `log_level` from config
- [ ] 6. Add VAD (Voice Activity Detection) as an enhancement to the manual toggle:
  - Implement simple energy-based VAD (RMS threshold on the audio buffer)
  - When enabled in config: auto-stop recording after N seconds of silence below the threshold
  - When disabled (default): recording only stops on toggle press — the original behavior
  - This is off by default and doesn't change the primary UX
- [ ] 7. Write a `Makefile` with the following targets:
  - `make build` — compile the binary and embed into `GoWhisper.app`
  - `make run` — build and run
  - `make test` — run all tests
  - `make install` — install app bundle to `/Applications`
  - `make download-model` — download the configured GGML model from Hugging Face (with SHA256 verification)
  - Add a pre-flight check: verify `portaudio` is installed via Homebrew, Xcode CLT present — fail with a helpful message if not
- [ ] 8. Concrete test scenarios (replace "test for a full day" with specific cases):
  - [ ] 10 English sentences across all modes
  - [ ] 10 Spanish sentences in translate_to_english mode
  - [ ] Rapid double-press of ⌥Space — confirm second press is ignored during PROCESSING
  - [ ] Press Esc while recording — confirm buffer is discarded and nothing is pasted
  - [ ] Kill Ollama mid-transcription — confirm raw transcript is pasted and warning is logged
  - [ ] Record longer than 30s — confirm chunking and stitching produce coherent output
  - [ ] Change hotkey in config while app is running — confirm new hotkey works

## Deliverable

Stable, daily-driveable app with working tray icon, audio feedback, notifications, long-recording support, and a complete Makefile.

## Notes

- Chunking overlap: 25s + 5s overlap prevents sentence-boundary cuts; always prefer overlap over hard splits
- Log rotation via `lumberjack` keeps the log file from growing unboundedly during daily use
- VAD added here (not Phase 1) — it's an enhancement on top of a working manual system, not a foundation requirement
- `UNUserNotificationCenter` must be called from the main thread — dispatch via the Cocoa main queue
- `make download-model` should pull from Hugging Face and verify SHA256 before replacing an existing model file
