# Phase 11 — Optional Extras (Post-MVP)

**Status:** 🔄 Partial — Most extras done; transcribe from file deferred

Nice-to-haves once everything is solid. Each feature ships independently.

## Features

| Feature | Description | Effort | Status |
|---|---|---|---|
| History log | Save all transcriptions with timestamps to a local SQLite file | Medium | ✅ Done |
| Auto-update models | Download newer GGML models from Hugging Face via tray menu | Low | ✅ Done |
| History retention | Prune history to a configurable max via `max_history_entries` | Low | ✅ Done |
| Clear history | "Clear History" tray menu item — wipes all SQLite entries | Low | ✅ Done |
| Add CLI to Path | Tray menu item (shown only if binary isn't in $PATH) | Low | ✅ Done |
| CLI help & history | `gowhisper help` and `gowhisper history [n]` subcommands | Low | ✅ Done |
| Transcribe from file | `gowhisper transcribe <file>` CLI subcommand | Medium | ⏳ Pending |

---

## History Retention

Add `max_history_entries` to `config.yaml` (default: 500). On every `history.Add()` call,
prune entries beyond the limit:

```sql
DELETE FROM transcriptions
WHERE id NOT IN (
  SELECT id FROM transcriptions ORDER BY timestamp DESC LIMIT ?
)
```

---

## Clear History

Add a "Clear History" item to the tray menu (below the history list). On click:
- Show a confirmation notification or use `NSAlert` (CGo) before wiping
- Run `DELETE FROM transcriptions`
- Refresh the history submenu to show empty state

---

## Add CLI to Path

On launch, check if `gowhisper` is resolvable via `exec.LookPath("gowhisper")`. If not,
show a tray menu item "Add CLI to Path". On click, create the symlink:

```sh
ln -sf /Applications/GoWhisper.app/Contents/MacOS/gowhisper /usr/local/bin/gowhisper
```

If `/usr/local/bin` requires elevated permissions, use `osascript` to prompt for sudo:

```sh
osascript -e 'do shell script "ln -sf ..." with administrator privileges'
```

Remove the menu item once the symlink is confirmed present.

Also add a `make install-cli` Makefile target for developer setups:
```makefile
install-cli:
	ln -sf /Applications/GoWhisper.app/Contents/MacOS/gowhisper /usr/local/bin/gowhisper
```

---

## Transcribe from File

**Delivery:** `gowhisper transcribe <file>` CLI subcommand (consistent with existing
`download-model` subcommand pattern). Optionally add a tray menu item "Transcribe File…"
that opens `NSOpenPanel` (CGo file picker) and invokes the same logic.

**Output:** Copy result to clipboard, same as the live recording flow. Result is saved to
history log — no new mental model needed.

### Input
- [ ] Add `transcribe` subcommand in `cmd/gowhisper/main.go` (alongside `download-model`)
- [ ] Accept **WAV and AIFF** natively (pure Go decoders — no extra deps)
- [ ] Optionally support MP3/M4A if `ffmpeg` is present at runtime — detect with
  `exec.LookPath("ffmpeg")`, do not make it a hard dependency; if absent, print a clear
  error explaining the limitation
- [ ] Convert accepted formats to 16kHz mono float32 PCM before handing to Whisper
  (same format live recording already produces)
- [ ] Reject unsupported formats with a clear error message

### Processing
- [ ] Reuse the existing `transcribe.Transcriber` — no new transcription path needed
- [ ] Apply the active mode (language, translate, prompt) exactly as live recording does
- [ ] Run LLM cleanup if enabled, same as live recording
- [ ] Write entry to history log (`history.Log`) so the result is always retrievable

### Output
- [ ] Copy final text to clipboard via existing `clipboard` package
- [ ] Print result to stdout (useful when invoked from terminal)
- [ ] Show a tray notification if the app is running: `"Transcribed: <first 60 chars>…"`

---

## Notes

- All features here are **independent** — implement in any order, or not at all
- History schema stores `prompt_used` (full text) not just `mode_name` — mode prompts can
  change, making name-only entries unreproducible
