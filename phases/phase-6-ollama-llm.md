# Phase 6 — LLM Transcript Cleanup (Claude API)

**Status:** [x] Done (commit 1fac1cd)
**Depends on:** Phase 5

Add optional AI cleanup on top of every raw transcript using the Claude API. No external programs required — just a REST call and an API key.

## What was built

- [x] `internal/llm/claude.go` — stdlib-only HTTP client (`net/http` + `encoding/json`, no SDK)
- [x] `claude:` config section replaces the original `ollama:` plan — `api_key`, `model`, `timeout_seconds`
- [x] API key resolved from `claude.api_key` in config first, then `ANTHROPIC_API_KEY` env var
- [x] Cleanup applied after every transcription when a key is available — regardless of active mode
- [x] Falls back to raw transcript silently on any API error or timeout
- [x] `context.WithTimeout` used per-call (default 15s, configurable)

## What changed from the original plan

- **Ollama dropped entirely** — no Ollama client, no `ollama:` config, no health-check, no streaming
- **No "Cleanup" mode** — cleanup is not a mode; it runs universally when the key is set
- **No `LLMEnabled` / `SystemPrompt` on `Mode` struct** — Mode stays simple (Name, Language, Translate)
- **No EN→ES translation** — not implemented, not planned
- **No streaming** — text is pasted once after full processing

## Cleanup system prompt

Tuned for dictation to AI agents, Claude Code, and Slack:
- Fixes punctuation, capitalization, grammar
- Removes filler words (um, uh, like, you know, etc.)
- Preserves technical terms, CLI flags, code identifiers, API names, agent names exactly as spoken

## Config

```yaml
claude:
  api_key: ""             # leave empty to use ANTHROPIC_API_KEY env var
  model: "claude-haiku-4-5-20251001"
  timeout_seconds: 15
```

## Notes

- Haiku 4.5 is fast (~100–300ms) and cheap for short cleanup tasks
- Claude Code subscription does not cover API usage — requires a separate API key from console.anthropic.com
- If no key is configured, the app behaves exactly as before (raw transcript pasted)
