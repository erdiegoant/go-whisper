# Phase 6 — Ollama LLM Post-Processing

**Status:** [x] Done
**Depends on:** Phase 5

Add optional AI cleanup and formatting on top of the raw transcript.

## Steps

- [ ] 1. Spin up Ollama locally with a small model — `llama3.2:3b` or `qwen2.5:3b` are fast and lightweight
- [ ] 2. At startup (when `ollama.enabled: true`), health-check `GET {endpoint}/api/tags`:
  - If Ollama is not running: log a clear warning, disable all LLM-backed modes, fall back to raw for those modes
  - Do not fail the whole app — Whisper-only modes still work
- [ ] 3. Implement an Ollama HTTP client in Go using the **chat endpoint**:
  - `POST {endpoint}/api/chat` with `stream: true`
  - Use a system prompt field (not appended to user message) — better instruction following for formatting tasks
  - Consume the token stream as it arrives — enables real-time UI updates in Phase 9 and mid-stream timeout cancellation
- [ ] 4. Create a `Mode` struct:
  ```go
  type Mode struct {
      Name        string
      SystemPrompt string
      LLMEnabled  bool
      Model       string  // optional override; empty = use global ollama.model
      Temperature float32 // optional override; 0 = use model default
  }
  ```
- [ ] 5. Implement the default "Cleanup" mode — removes filler words, fixes punctuation, preserves meaning
- [ ] 6. Pipe transcript through LLM only when `mode.LLMEnabled == true` — raw transcription stays the fast default path
- [ ] 7. Guard: if transcript is empty, skip Ollama entirely (short-circuit before any HTTP call)
- [ ] 8. Implement timeout handling using `context.WithTimeout`:
  - Default timeout: **10 seconds** (configurable via `ollama.timeout_seconds`)
  - If timeout is hit mid-stream: cancel the context, fall back to the raw transcript, log a warning
- [ ] 9. Implement `translate_to_spanish` mode — the only translation direction requiring Ollama (EN→ES):
  - System prompt: `"Translate the following text to Spanish. Return only the translation."`
  - Note: ES→EN uses Whisper's native translation (Phase 4), which is higher quality — do not duplicate it here
- [ ] 10. Test: dictate a messy sentence with filler words, confirm cleanup mode polishes it
- [ ] 11. Test: kill Ollama mid-transcription, confirm raw transcript is pasted and a warning is logged

## Deliverable

A `Process(text string, mode Mode) string` function that optionally enhances the transcript, with graceful fallback to the raw text on any error or timeout.

## Notes

- Use `/api/chat` not `/api/generate` — system prompt support gives better results for structured output tasks
- `stream: true` is important: non-streaming blocks until full generation, making timeout detection delayed; streaming lets you cancel instantly at the timeout boundary
- Default timeout is 10s (not 3s) — a 3B model cold-starts its first token in 1–3s on average hardware
- `translate_to_spanish` via Ollama covers the EN→ES gap; ES→EN is always Whisper-native (Phase 4)
- Ollama endpoint is configurable (`ollama.endpoint`) to support remote Ollama instances
