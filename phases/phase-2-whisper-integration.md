# Phase 2 — Whisper.cpp Integration

**Status:** [x] Done
**Depends on:** Phase 1

Wire the audio buffer into the whisper.cpp Go bindings.

## Steps

- [x] 1. Clone whisper.cpp as a git submodule and build static libraries via `make whisper`
- [x] 2. Add `github.com/ggerganov/whisper.cpp/bindings/go` to `go.mod`
- [x] 3. Download a GGML model file — `ggml-small.bin` (~465MB) via `make download-model`, stored in `~/.config/gowhisper/models/`
- [x] 4. Load the model at startup and keep it in memory — do not reload per transcription
- [x] 5. `TranscribeRequest` struct with `Samples []float32`, `Language string`, `Translate bool`
- [x] 6. Protect the whisper context with a mutex — `whisper.Context` is not thread-safe
- [x] 7. Guard against empty/blank-audio results — return error, do not proceed to clipboard
- [x] 8. ES→EN translation via `ctx.SetTranslate(true)` + `ctx.SetLanguage("es")`
- [x] 9. Disable token timestamps and progress callbacks to reduce CGo overhead
- [x] 10. `Swap(modelPath string)` method for hot-swapping the model on config reload (Phase 5)

## Deliverable

`internal/transcribe/` with `Transcriber` — `New`, `Transcribe`, `Swap`, `Close`. Concurrency-safe.

## Notes

- Model sizes: tiny (~75MB), small (~465MB, recommended), medium (~1.5GB)
- Model path always resolved from `models_dir` config, never hardcoded
- `Swap` holds the mutex so in-flight transcription completes before the model is replaced
