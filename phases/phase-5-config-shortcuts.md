# Phase 5 — Config & Shortcut Customization

**Status:** [x] Done
**Depends on:** Phase 3

Make the app fully configurable without recompiling, including hotkeys.

## Steps

- [x] 1. `~/.config/gowhisper/config.yaml` auto-created on first launch with defaults:

```yaml
model: small
language: auto
models_dir: "~/.config/gowhisper/models"
max_recording_seconds: 120
log_level: info

claude:
  api_key: ""
  model: "claude-haiku-4-5-20251001"
  timeout_seconds: 15

hotkeys:
  toggle_recording: "option+space"
  cancel_recording: "esc"
  change_mode: "option+shift+k"
```

- [x] 2. `internal/config/` package: YAML loading, `internal/config/keyparse.go` parses hotkey strings into `hotkey.Modifier`/`hotkey.Key`
- [x] 3. fsnotify hot-reload (200ms debounce) — hotkeys and model rebind/swap on save without restart
- [x] 4. Conflict detection (log warning) and invalid key handling (keep previous binding)
- [x] 5. Model change on config reload: `Transcriber.Swap()` holds mutex, waits for in-flight transcription
- [x] 6. `state.json` for runtime state (`last_mode`, `last_language`) — never written back to config.yaml
- [x] 7. `--config` CLI flag for custom config path
- [x] 8. "Open Config" tray menu item — opens config.yaml in the default system app

## Deliverable

`internal/config/Manager` with `Combos()`, `ModelPath()`, `ClaudeConfig()`, `OnChange()`, `Dir()`. Hot-reload works for hotkeys and model.

## Notes

- `state.json` lives alongside `config.yaml` in the config dir
- fsnotify watches the specific config file only — state.json changes don't trigger a reload
- `claude.api_key` falls back to `ANTHROPIC_API_KEY` env var if empty
