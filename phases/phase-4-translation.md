# Phase 4 — Translation Flow

**Status:** [ ] Not Started
**Depends on:** Phase 3

Make ES↔EN translation work seamlessly as a separate mode.

## Steps

- [ ] 1. Translation is a **mode**, not a separate toggle hotkey — it fits naturally into the mode cycle introduced in Phase 3 (Change Mode: ⌥⇧K). No new hotkey needed.
- [ ] 2. Add a `translate` mode to the hardcoded mode list for now (Phase 7 makes this config-driven):
  - When active: `SetLanguage("es")` + `SetTranslate(true)` for ES→EN
  - When inactive: normal dictation mode
- [ ] 3. Implement auto language detection option — `ctx.SetLanguage("auto")` lets Whisper detect the input language automatically
  - Note: "auto" + translate quirk — if the input is already English and translate is on, Whisper still runs the translation pathway (slower, occasionally degrades quality). Prefer explicit `"es"` when you know the source language.
- [ ] 4. For ES→EN: `SetLanguage("es")` + `SetTranslate(true)` — this is Whisper's native path, higher quality than LLM translation
- [ ] 5. For EN→ES: Whisper only translates *into* English natively — flag in mode config as `llm: true`; handled in Phase 6 via Ollama prompt
- [ ] 6. Show the active mode name in the tray icon (e.g. update the placeholder systray tooltip or title)
- [ ] 7. Test: activate translate mode with ⌥⇧K, record Spanish speech, confirm English text is pasted
- [ ] 8. Test: press Esc during translate mode recording, confirm nothing is pasted

## Deliverable

Two working modes accessible via the mode cycle hotkey — dictate (transcribe as-is) and translate (ES→EN). Cancel works correctly in both modes.

## Notes

- Whisper's built-in translation only goes TO English — EN→ES requires Ollama (Phase 6, `translate_to_spanish` mode)
- ES→EN via Whisper is higher quality than ES→EN via LLM prompt — always prefer the native path
- No dedicated translate toggle hotkey — ⌥⇧K cycles through all modes including translate
