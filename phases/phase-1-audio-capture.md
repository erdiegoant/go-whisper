# Phase 1 — Foundation & Audio Capture

**Status:** [x] Done

Get audio from the microphone into Go as raw samples whisper.cpp can consume.

## Architecture Note (Read Before Coding)

The app must be Cocoa-compatible from day one (required for Phase 9's DarwinKit UI). Design the main loop accordingly:
- `macos.RunApp` (DarwinKit) owns the **main thread** — it must never block
- All audio capture, hotkey listening, and transcription run in **goroutines**
- Goroutines communicate back to the main thread via **channels**

Even in early phases before DarwinKit is wired in, structure the code this way to avoid a full rewrite at Phase 9.

## Project Structure

Set up this layout in step 1:

```
cmd/gowhisper/main.go
internal/audio/
internal/transcribe/
internal/clipboard/
internal/hotkey/
internal/config/
internal/llm/
internal/ui/
models/
GoWhisper.app/
  Contents/
    Info.plist
    MacOS/
    Resources/
```

## Steps

- [ ] 1. Initialize Go module and project structure (layout above)
- [ ] 2. Scaffold `GoWhisper.app` bundle with `Info.plist` declaring:
  - `NSMicrophoneUsageDescription` — required for mic permission on macOS
  - `NSAccessibilityUsageDescription` — required for global hotkeys and simulated paste
  - `LSUIElement = true` — hides the app from the Dock (menubar-only app)
- [ ] 3. Add `github.com/gen2brain/malgo` (miniaudio Go bindings) — single-header C library that compiles directly into the binary, no Homebrew or system install required; uses CoreAudio on macOS under the hood
- [ ] 4. Implement mic capture — record float32 samples at 16kHz mono (Whisper's required format)
- [ ] 5. Handle sample rate mismatch — request 16kHz from miniaudio; if the device doesn't natively support it, miniaudio will resample automatically via its built-in resampler
- [ ] 6. Implement a simple recording state machine:
  - `IDLE` → `RECORDING` → `PROCESSING` → `IDLE`
  - Recording starts and stops manually (toggle); no automatic silence detection
  - Cancel transitions `RECORDING` → `IDLE` and discards the buffer
- [ ] 7. Write captured samples to an in-memory `[]float32` buffer (no disk I/O needed)
- [ ] 8. Test: record 5 seconds and dump raw samples to verify 16kHz mono float32 format is correct

## Deliverable

A Go program that captures mic input and produces a `[]float32` sample buffer on demand (start/stop controlled externally).

## Notes

- Whisper requires 16kHz mono float32 PCM — enforce this at the capture layer, not the transcription layer
- `malgo` (miniaudio) compiles its C source inline — no system install needed, fully self-contained binary
- miniaudio handles resampling automatically when the device doesn't natively support 16kHz
- No VAD / automatic silence detection in this phase — recording is always started and stopped manually. VAD is a post-MVP extra (Phase 10).
- The `.app` bundle is required for macOS permission dialogs to work correctly — a bare binary cannot request mic access
