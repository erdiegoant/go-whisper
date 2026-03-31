# Phase 7 — Custom Modes

**Status:** [x] Done
**Depends on:** Phase 6

Let the user define their own modes in `config.yaml` — custom names, custom system prompts for Claude cleanup, custom Whisper language/translate settings.

## Current state (entering Phase 7)

Two hardcoded modes in `internal/mode/mode.go`:
- **Standard** — `Language: "auto"`, `Translate: false`
- **Translate** — `Language: "es"`, `Translate: true`

Claude cleanup always runs when `ANTHROPIC_API_KEY` is set, using the global `llm.CleanupPrompt` regardless of mode. There is no per-mode prompt override yet.

## Goal

Load modes from `config.yaml` instead of hardcoding them. Each mode can optionally override the cleanup system prompt sent to Claude. Cycling (⌥⇧K), tray title, and state persistence all continue to work.

## Config structure

```yaml
modes:
  - name: Standard
    language: auto
    translate: false
    # no prompt — uses the built-in CleanupPrompt

  - name: Translate
    language: es
    translate: true
    # no prompt — CleanupPrompt applied to the English output

  - name: Formal
    language: auto
    translate: false
    prompt: "Rewrite this transcript in a formal professional tone. Preserve all technical terms, CLI flags, and code identifiers exactly. Return only the result."

  - name: Bullets
    language: auto
    translate: false
    prompt: "Convert this dictation into a concise bullet point list. Preserve technical terms. Return only the result."

  - name: Code Comment
    language: auto
    translate: false
    prompt: "Format this as a developer code comment or docstring. Preserve all identifiers and technical terms. Return only the result."
```

When `prompt` is set, it replaces `llm.CleanupPrompt` for that mode's Claude call. When absent, the default CleanupPrompt is used.

## Steps

- [x] 1. Add `Prompt string` field to `Mode` struct in `internal/mode/mode.go`
- [x] 2. Add `modesRaw` to `internal/config/config.go`:
  ```go
  type modeRaw struct {
      Name      string `yaml:"name"`
      Language  string `yaml:"language"`
      Translate bool   `yaml:"translate"`
      Prompt    string `yaml:"prompt"`
  }
  ```
  Add `Modes []modeRaw` to the `raw` struct.
- [x] 3. Add `Modes() []mode.Mode` accessor to `config.Manager` — converts `[]modeRaw` to `[]mode.Mode`, applying defaults (`Language: "auto"` when empty)
- [x] 4. On startup, if `modes:` is absent or empty in config, fall back to the two hardcoded defaults (Standard + Translate) so the app works out of the box without a modes block
- [x] 5. Update `mode.Manager` to hold a `[]Mode` slice instead of relying on the package-level `All`:
  - `NewManager(modes []Mode) *Manager`
  - Track active mode by **name**, not index — on config reload, look up the current name in the new list; fall back to first mode if not found
- [x] 6. Wire config reload: when `OnChange` fires, rebuild `modeManager` with the new modes list and update the tray title via `setModeCh <- ""`
- [x] 7. In the transcription goroutine, use `m.Prompt` if non-empty, otherwise fall back to `llm.CleanupPrompt`
- [x] 8. `state.json` mode persistence — `SetByName` handles unknown names gracefully (falls back to index 0 on reload)
- [x] 9. Update `writeDefaults` in config to include a commented-out example modes block so users know the format
- [x] 10. Add a **Mode** submenu to the tray:
  - Lists every mode by name
  - Active mode has a checkmark (✓) via `item.Check()`
  - Each item's tooltip shows: `"Standard — auto transcription"`, `"Translate — ES→EN (Whisper native)"`, or first ~60 chars of the custom prompt
  - Clicking a mode activates it immediately (same as cycling to it with ⌥⇧K)
  - `AddModeMenu` returns an update func; event loop calls it when active mode changes
- [x] 11. Tests written for `internal/mode` (10 cases), `internal/config` (keyparse, state, parseModes, applyDefaults, combosEqual), `internal/llm` (CleanupPrompt, Process with fake server)
- [x] 12. All tests pass: `go test ./internal/mode/... ./internal/config/... ./internal/llm/...`

## Deliverable

Modes fully driven by `config.yaml`. Hardcoded `All` slice removed. Users can add unlimited modes with custom prompts without touching code. Mode picker in the tray menubar lists all modes with checkmarks and prompt previews as tooltips.

## Notes

- No `llm: false` flag — Claude cleanup always runs when the API key is set, regardless of mode. The `prompt` field only controls *which* system prompt is used.
- No `translate_to_spanish` mode — EN→ES is not planned
- No per-mode model or temperature overrides — those were Ollama-specific concepts, not needed with Claude API
- The two built-in modes (Standard, Translate) become the config defaults, not special-cased code paths
- Mode names are user-defined strings — no reserved names
