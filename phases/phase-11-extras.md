# Phase 11 — Optional Extras (Post-MVP)

**Status:** 🔄 Partial — History log and Auto-update models done; Push to Talk, Mouse shortcut, Transcribe from file, and Streaming deferred
**Depends on:** Phase 9

Nice-to-haves once everything is solid. Each feature ships independently.

## Features

| Feature | Description | Effort |
|---|---|---|
| Push to Talk | Hold to record, release to transcribe (alternative to toggle) | Low |
| History log | Save all transcriptions with timestamps to a local SQLite file | Medium |
| Transcribe from file | Drag an audio file onto the tray icon to transcribe it | Medium |
| Streaming transcription | Show text appearing in real time as you speak | High (research spike) |
| Auto-update models | CLI command to download newer GGML models from Hugging Face | Low |
| Mouse shortcut | Tap to toggle, or hold and release for push-to-talk | Medium |

---

## Push to Talk
> Requires key-up events. If using `golang.design/x/hotkey`, this is not supported — switch to `robotgo` for this feature only, or implement via a CGo hook.

- [ ] Research key-up support in current hotkey library
- [ ] Implement hold-to-record, release-to-transcribe as an alternative recording mode
- [ ] Add config option: `recording_mode: toggle | push_to_talk` (default: `toggle`)
- [ ] The two modes are mutually exclusive — `toggle_recording` hotkey maps to whichever is active

## History Log

- [ ] Add SQLite dependency: `modernc.org/sqlite` (pure Go, no CGo)
- [ ] Create schema:
  ```sql
  CREATE TABLE transcriptions (
      id INTEGER PRIMARY KEY,
      timestamp TEXT NOT NULL,
      mode_name TEXT,
      prompt_used TEXT,       -- store actual prompt text, not just mode name (prompts change over time)
      raw_text TEXT,
      processed_text TEXT,
      duration_ms INTEGER,
      language TEXT
  );
  ```
- [ ] Write entry after each successful paste
- [ ] Add `gowhisper history` CLI subcommand to tail/search the log
- [ ] Add "Copy from history" option in Phase 9's menubar dropdown

## Transcribe from File

- [ ] Register app as drag destination on the menubar icon
- [ ] Accept **WAV and AIFF** natively (pure Go decoders available)
- [ ] Optionally support MP3/M4A if `ffmpeg` is present at runtime — detect with `which ffmpeg`, do not make it a hard dependency
- [ ] Convert accepted formats to 16kHz mono float32 before passing to Whisper
- [ ] Show progress in the floating panel (Phase 9b) during long file transcriptions

## Streaming Transcription

> **Research spike required before committing to this feature.**

- [ ] Investigate whether the whisper.cpp Go bindings expose the streaming/partial results API (`whisper_full_with_state`)
- [ ] If not exposed: assess effort to add it via raw CGo — this may be substantial
- [ ] If viable: implement a floating text overlay that updates in real time as segments arrive
- [ ] Finalize and paste on toggle press (stop recording)

## Auto-Update Models

- [ ] Add `download-model` CLI subcommand: `gowhisper download-model --size small`
- [ ] Pull from Hugging Face (`ggerganov/whisper.cpp` repo)
- [ ] Show download progress bar in terminal
- [ ] Verify SHA256 checksum before replacing existing model file
- [ ] Store downloaded models in `models_dir` from config

## Mouse Shortcut

- [ ] Implement a global mouse button listener (requires `robotgo` or CGo hook)
- [ ] Config: `mouse_shortcut: enabled: false` with `button: side` as default
- [ ] Tap to toggle, hold+release for push-to-talk (same dual behavior as shown in Superwhisper)

---

## Notes

- All features here are **independent** — implement in any order, or not at all
- History schema stores `prompt_used` (full text) not just `mode_name` — mode prompts can change, making name-only entries unreproducible
- Streaming transcription is the highest-risk item — do the research spike before estimating effort
- Push-to-talk requires a different hotkey library than toggle (`golang.design/x/hotkey` does not expose key-up) — plan for that dependency swap
