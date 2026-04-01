# Phase 10 — Local LLM Backend (Ollama)

**Status:** ✅ Done
**Depends on:** Phase 8

Add Ollama as an optional local LLM backend for transcript cleanup. Users who don't have (or don't want) a Claude API key can run a local model with no internet round-trip and no cost.

## Goal

Keep Claude as-is. Add Ollama as a second backend behind the same internal interface. Config determines which one runs. If neither is configured, cleanup is skipped — exactly as today.

**Priority order:** `ollama.model` set → Ollama; else `claude.api_key` set → Claude; else no cleanup.

---

## Config changes

Add an `ollama:` block to `config.yaml`:

```yaml
ollama:
  model: "llama3.2:3b"              # any model pulled in Ollama
  host: "http://localhost:11434"    # optional — default shown
  timeout_seconds: 30
```

- If `ollama.model` is empty (the default), Ollama is disabled.
- `host` defaults to `http://localhost:11434` if omitted.
- Claude config remains unchanged; the two sections coexist.

---

## Step 1 — Extract a `Processor` interface (`internal/llm/llm.go`)

New file. Both backends implement this:

```go
// Processor cleans up a transcript using a language model.
type Processor interface {
    Process(systemPrompt, text string) (string, error)
}
```

The existing `*Client` in `claude.go` already satisfies this — no changes needed to that file.

---

## Step 2 — Ollama client (`internal/llm/ollama.go`)

```go
type OllamaClient struct {
    model   string
    host    string
    timeout time.Duration
    http    *http.Client
}

func NewOllama(model, host string, timeoutSeconds int) *OllamaClient
func (c *OllamaClient) Process(systemPrompt, text string) (string, error)
```

**API call:** `POST {host}/api/chat`

```json
{
  "model": "llama3.2:3b",
  "stream": false,
  "messages": [
    {"role": "system", "content": "<systemPrompt>"},
    {"role": "user",   "content": "Transcript:\n\n<text>"}
  ]
}
```

**Response shape:**
```json
{"message": {"role": "assistant", "content": "cleaned text"}}
```

If Ollama is not running or the model isn't pulled, the call fails and the caller falls back to the raw transcript (same behaviour as a Claude API error today).

---

## Step 3 — Config (`internal/config/config.go`)

Add `OllamaConfig` struct and `OllamaConfig()` getter:

```go
type OllamaConfig struct {
    Model          string `yaml:"model"`
    Host           string `yaml:"host"`
    TimeoutSeconds int    `yaml:"timeout_seconds"`
}
```

Defaults: `Host = "http://localhost:11434"`, `TimeoutSeconds = 30`.

---

## Step 4 — Wire up in `cmd/gowhisper/main.go`

Replace the single `llmClient *llm.Client` with `llmClient llm.Processor` (the interface). Build it:

```go
var llmClient llm.Processor
if oc := cfg.OllamaConfig(); oc.Model != "" {
    llmClient = llm.NewOllama(oc.Model, oc.Host, oc.TimeoutSeconds)
    log.Printf("llm: Ollama (%s @ %s)", oc.Model, oc.Host)
} else if cc := cfg.ClaudeConfig(); cc.APIKey != "" {
    llmClient = llm.New(cc.APIKey, cc.Model, cc.TimeoutSeconds)
    log.Printf("llm: Claude (%s)", cc.Model)
} else {
    log.Println("llm: no backend configured — cleanup disabled")
}
```

No other changes to the event loop — it already calls `llmClient.Process(...)`.

---

## Verification

```bash
# Install Ollama (if not already)
brew install ollama
ollama pull llama3.2:3b   # ~2GB download

# Add to config
ollama:
  model: "llama3.2:3b"

# Build and run
make dev

# Dictate something — transcript should be cleaned without any Claude API call
# Check log: should say "llm: Ollama (llama3.2:3b @ http://localhost:11434)"
```

**Test matrix:**
- [ ] Ollama running, model pulled → cleanup works, no Claude call
- [ ] Ollama not running → logs warning, raw transcript pasted (no crash)
- [ ] Both `ollama.model` and `claude.api_key` set → Ollama wins
- [ ] Neither set → cleanup skipped, raw transcript pasted
- [ ] `ollama.model` set but wrong model name → error logged, raw transcript pasted

---

## Files to change

| File | Change |
|---|---|
| `internal/llm/llm.go` | **New** — `Processor` interface |
| `internal/llm/ollama.go` | **New** — Ollama HTTP client |
| `internal/config/config.go` | Add `OllamaConfig` struct + getter, default host/timeout |
| `cmd/gowhisper/main.go` | `llmClient` type → `llm.Processor`; add Ollama selection logic |
| `README.md` | Document Ollama as optional backend |

No changes to `internal/llm/claude.go`, `claude_test.go`, or any other package.
