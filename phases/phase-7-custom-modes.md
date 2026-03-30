# Phase 7 — Custom Modes

**Status:** [ ] Not Started
**Depends on:** Phase 6

Let the user define their own formatting modes — the equivalent of Superwhisper's paid Modes feature, running locally and free.

## Steps

- [ ] 1. Define modes in `config.yaml` as a list of name + system prompt pairs:

```yaml
modes:
  - name: raw
    llm: false
    prompt: ""
  - name: cleanup
    llm: true
    prompt: "Clean up this transcript. Remove filler words, fix punctuation, keep the meaning intact. Return only the result."
  - name: formal
    llm: true
    prompt: "Rewrite this in a formal professional tone. Return only the result."
  - name: bullets
    llm: true
    prompt: "Convert this dictation into a concise bullet point list. Return only the result."
    temperature: 0.2      # optional: override global model temperature
  - name: code_comment
    llm: true
    prompt: "Format this as a developer code comment or docstring. Return only the result."
  - name: translate_to_spanish
    llm: true
    prompt: "Translate the following text to Spanish. Return only the translation."
  - name: translate_to_english
    llm: false            # uses Whisper native (Phase 4), not Ollama
    prompt: ""
```

- [ ] 2. The `change_mode` hotkey (⌥⇧K, defined in config) cycles through the mode list in order — circular, wraps back to first after last
- [ ] 3. Track active mode by **name**, not by list index:
  - On config reload, look up the current mode name in the new list
  - If not found (mode was removed or renamed), fall back to the first mode in the new list
  - This prevents silent mode jumps when the user edits config while the app is running
- [ ] 4. Modes marked `llm: false` skip Ollama entirely — zero added latency
- [ ] 5. Guard: if transcript is empty string after Whisper, skip mode processing and paste entirely, log a debug message
- [ ] 6. Users add unlimited custom modes without touching code — just add entries to `config.yaml`
- [ ] 7. Test each built-in mode with sample dictation in English and Spanish
- [ ] 8. Test: rename a mode in config while app is running, confirm active mode falls back gracefully

## Deliverable

Full mode system working. User can cycle modes with ⌥⇧K and add custom ones in `config.yaml`.

## Notes

- Mode cycle is circular: raw → cleanup → formal → bullets → code_comment → translate_to_spanish → translate_to_english → raw → ...
- `raw` and `translate_to_english` are `llm: false` — no Ollama call at all
- Per-mode optional overrides: `model` (override global `ollama.model`) and `temperature` (override model default)
- Mode name shown in tray icon (placeholder text for now; proper animated display in Phase 9)
- The `translate_to_english` mode delegates to Whisper's native translation, not Ollama — the mode struct's `LLMEnabled: false` signals the transcription layer to use `SetTranslate(true)` instead
