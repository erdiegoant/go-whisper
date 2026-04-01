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

- [ ] Register app as drag destination on the menubar icon
- [ ] Accept **WAV and AIFF** natively (pure Go decoders available)
- [ ] Optionally support MP3/M4A if `ffmpeg` is present at runtime — detect with `which ffmpeg`, do not make it a hard dependency
- [ ] Convert accepted formats to 16kHz mono float32 before passing to Whisper
- [ ] Decide output behaviour (see note below)

> **Open question — output behaviour:** two options:
> - **Clipboard** — consistent with the live recording flow; works well for short clips
> - **Text file** — saved alongside the source audio; better for long recordings like meetings
>
> A middle ground: copy to clipboard for short files (under ~60s), write a `.txt` next to the source for longer ones.

---

## Notes

- All features here are **independent** — implement in any order, or not at all
- History schema stores `prompt_used` (full text) not just `mode_name` — mode prompts can change, making name-only entries unreproducible
