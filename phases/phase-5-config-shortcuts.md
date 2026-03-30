# Phase 5 — Config & Shortcut Customization

**Status:** [ ] Not Started
**Depends on:** Phase 3

Make the app fully configurable without recompiling, including hotkeys.

## Steps

- [ ] 1. Create `~/.config/gowhisper/config.yaml` with sane defaults:

```yaml
model: small              # tiny | small | medium
language: auto            # auto | es | en
models_dir: "~/.config/gowhisper/models"
max_recording_seconds: 120  # hard cap before forced stop (no VAD)
log_level: info             # debug | info | warn | error
notifications_enabled: true

ollama:
  enabled: false
  endpoint: "http://localhost:11434"
  model: llama3.2:3b
  timeout_seconds: 10

hotkeys:
  toggle_recording: "option+space"
  cancel_recording: "esc"
  change_mode: "option+shift+k"

modes:
  default: raw
```

- [ ] 2. Implement config loading at startup — parse hotkey strings into combos the `golang.design/x/hotkey` listener understands
- [ ] 3. Implement **hotkey rebinding at runtime** — when config changes on disk, reload hotkeys without restarting using `github.com/fsnotify/fsnotify`
  - Unregister old hotkeys, register new ones. Document that there is a brief gap (~ms) during the swap.
- [ ] 4. Validate hotkey config on load:
  - Detect conflicts (two actions mapped to the same key combo) — log a warning, last definition wins
  - Detect invalid key strings — log an error and keep the previous binding
- [ ] 5. Handle model change on config reload:
  - If `model:` value changes, unload the current model from memory and load the new one
  - Resolve model path as: `{models_dir}/ggml-{model}.bin`
- [ ] 6. Persist last-used mode and language to `~/.config/gowhisper/state.json` (NOT back into `config.yaml`):
  ```json
  { "last_mode": "cleanup", "last_language": "es" }
  ```
  Read on startup, restore to last state.
- [ ] 7. Add a `--config` CLI flag to point to a custom config file path
- [ ] 8. Test: change `toggle_recording` in config, confirm the new hotkey works without restarting
- [ ] 9. Test: change `model: small` to `model: tiny`, confirm the model swaps without restart

## Deliverable

Fully configurable app where all hotkeys are user-defined and hot-reloadable from `config.yaml`.

## Notes

- Default hotkeys match Superwhisper's defaults: ⌥Space toggle, Esc cancel, ⌥⇧K mode change
- `option+space` avoids the `ctrl+shift+space` conflict with macOS input source switchers
- Last-used state in `state.json` keeps `config.yaml` clean and version-control friendly
- `models_dir` must be consistent between Phase 2 (model loading) and this config — absolute path, tilde expanded at load time
- Missing config fields from earlier phases that must be added here: `models_dir`, `max_recording_seconds`, `log_level`, `notifications_enabled`, `ollama.endpoint`
