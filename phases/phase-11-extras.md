# Phase 11 — Optional Extras (Post-MVP)

**Status:** 🔄 Partial — History log and Auto-update models done; Transcribe from file deferred

Nice-to-haves once everything is solid. Each feature ships independently.

## Features

| Feature | Description | Effort | Status |
|---|---|---|---|
| History log | Save all transcriptions with timestamps to a local SQLite file | Medium | ✅ Done |
| Auto-update models | Download newer GGML models from Hugging Face via tray menu | Low | ✅ Done |
| Transcribe from file | Drag an audio file onto the tray icon to transcribe it | Medium | ⏳ Pending |

---

## Transcribe from File

**Output:** copy result to clipboard, same as the live recording flow. No new mental model. If the transcription needs to be retrieved later it is already saved to the history log.

### Input
- [ ] Register app as drag destination on the menubar icon
- [ ] Accept **WAV and AIFF** natively (pure Go decoders available — no extra deps)
- [ ] Optionally support MP3/M4A if `ffmpeg` is present at runtime — detect with `exec.LookPath("ffmpeg")`, do not make it a hard dependency; if absent, show a tray notification explaining the limitation
- [ ] Convert accepted formats to 16kHz mono float32 PCM before handing to Whisper (same format live recording already produces)
- [ ] Reject unsupported formats with a tray notification

### Processing
- [ ] Reuse the existing `transcribe.Transcriber` — no new transcription path needed
- [ ] Apply the active mode (language, translate, prompt) exactly as live recording does
- [ ] Run LLM cleanup if enabled, same as live recording
- [ ] Write entry to history log (`history.Log`) so the result is always retrievable

### Output
- [ ] Copy final text to clipboard via existing `clipboard` package
- [ ] Show a tray notification: `"Transcribed: <first 60 chars>…"`
- [ ] Tray icon shows `⏳ <mode>` during processing, returns to idle when done

### Tray state during file transcription
- Reuse the same idle → processing → idle state machine already used for live recording
- Concurrent file transcription + live recording should be blocked — if a recording is in progress, drop the drag with a notification

---

## Notes

- All features here are **independent** — implement in any order, or not at all
- History schema stores `prompt_used` (full text) not just `mode_name` — mode prompts can change, making name-only entries unreproducible
