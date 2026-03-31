# Phase 1 — Foundation & Audio Capture

**Status:** [x] Done

Get audio from the microphone into Go as raw samples whisper.cpp can consume.

## Steps

- [x] 1. Initialize Go module and project structure
- [x] 2. Scaffold `GoWhisper.app` bundle with `Info.plist` declaring `NSMicrophoneUsageDescription`, `NSAccessibilityUsageDescription`, `LSUIElement = true`
- [x] 3. Add `github.com/gen2brain/malgo` (miniaudio Go bindings) — single-header C library, no Homebrew required, uses CoreAudio on macOS
- [x] 4. Implement mic capture — record float32 samples at 16kHz mono (Whisper's required format)
- [x] 5. miniaudio resamples automatically if the device doesn't natively support 16kHz
- [x] 6. Recording state machine: `IDLE → RECORDING → PROCESSING → IDLE`. Cancel discards buffer and returns to IDLE.
- [x] 7. In-memory `[]float32` buffer — no disk I/O
- [x] 8. `cmd/rectest/` — standalone 5-second WAV capture tool to verify mic access and sample format

## Deliverable

`internal/audio/` package with `Capturer` — start/stop/cancel recording, device selection, state machine.

## Notes

- Whisper requires 16kHz mono float32 PCM — enforced at the capture layer
- The `.app` bundle is required for macOS permission dialogs to work correctly
- No VAD / automatic silence detection — recording is always started and stopped manually
