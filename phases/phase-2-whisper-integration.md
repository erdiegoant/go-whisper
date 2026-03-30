# Phase 2 — Whisper.cpp Integration

**Status:** [~] In Progress
**Depends on:** Phase 1

Wire the audio buffer into the whisper.cpp Go bindings.

## Steps

- [ ] 1. Clone whisper.cpp and build the static `libwhisper.a` library
  - Requires: Xcode Command Line Tools (`xcode-select --install`)
- [ ] 2. Add `github.com/ggml-org/whisper.cpp/bindings/go/pkg/whisper` to `go.mod`
- [ ] 3. Download a GGML model file — start with `ggml-small.bin` (~470MB)
  - Store in `{models_dir}` from config (default: `~/.config/gowhisper/models/`)
  - Resolve path absolutely at load time (a relative path breaks when the binary is installed to `/usr/local/bin`)
- [ ] 4. Load the model at startup and keep it in memory — do not reload per transcription
- [ ] 5. Use a `TranscribeRequest` struct as the function input (avoids breaking API changes when modes and languages are added in later phases):
  ```go
  type TranscribeRequest struct {
      Samples  []float32
      Language string // "auto" | "es" | "en"
      Translate bool   // true = translate TO English (Whisper native)
  }

  func Transcribe(req TranscribeRequest) (string, error)
  ```
- [ ] 6. Protect the whisper context with a mutex (or route all calls through a single-worker channel) — `whisper.Context` is not thread-safe; rapid double-press of the hotkey must not race
- [ ] 7. Guard against empty transcript: if Whisper returns `""` or whitespace-only, return early — do not proceed to clipboard or LLM
- [ ] 8. Implement translation mode — `ctx.SetTranslate(true)` + `ctx.SetLanguage("es")` for ES→EN
- [ ] 9. Disable progress callbacks to avoid CGo overhead
- [ ] 10. Test: transcribe a pre-recorded WAV file in both Spanish and English

## Example

```go
model, _ := whisper.New("/Users/you/.config/gowhisper/models/ggml-small.bin")
defer model.Close()

ctx, _ := model.NewContext()
ctx.SetLanguage("es")
ctx.SetTranslate(true) // ES → EN built-in

ctx.Process(samples, nil, nil, nil)

for {
    segment, err := ctx.NextSegment()
    if err != nil { break }
    fmt.Println(segment.Text)
}
```

## Deliverable

A function `Transcribe(req TranscribeRequest) (string, error)` that returns clean text, is concurrency-safe, and guards against empty output.

## Notes

- Model sizes: tiny (~75MB), small (~470MB, recommended), medium (~1.5GB)
- Model path: always resolved from `models_dir` config value, never hardcoded or relative
- Model hot-swap: when Phase 5's config watcher detects a `model:` change, it must call model.Close() and reload — this is coordinated via the same mutex that protects the context
- Concurrent transcription: the mutex means a second hotkey press during processing is queued, not dropped — but the PROCESSING state in Phase 3's state machine prevents a second recording from starting anyway, making this a belt-and-suspenders guard
