# GoWhisper — Development Plan

A Superwhisper-inspired voice dictation and translation app for macOS, built in Go using whisper.cpp, Ollama, and DarwinKit.

---

## Stack Overview

| Layer | Technology |
|---|---|
| Language | Go |
| STT Engine | whisper.cpp (via Go bindings) |
| Audio Capture | PortAudio (`github.com/gordonklaus/portaudio`) |
| Hotkeys | `golang.design/x/hotkey` or `github.com/robotn/robotgo` |
| Clipboard | `golang.design/x/clipboard` |
| LLM Post-processing | Ollama (local, optional) |
| Native macOS UI | DarwinKit (`github.com/progrium/darwinkit`) |
| Config | `config.yaml` with file watcher (`github.com/fsnotify/fsnotify`) |

---

## Phase 1 — Foundation & Audio Capture

Get audio from the microphone into Go as raw samples whisper.cpp can consume.

### Steps

1. Initialize Go module and project structure
2. Install and test PortAudio bindings (`github.com/gordonklaus/portaudio`)
3. Implement mic capture — record float32 samples at 16kHz mono (Whisper's required format)
4. Implement VAD (Voice Activity Detection) — detect silence to auto-stop recording instead of requiring a manual stop
5. Write audio to a temp WAV file or pass samples directly to a memory buffer
6. Test: record 5 seconds and dump raw samples to verify format is correct

**Deliverable:** A Go program that captures mic input and produces a `[]float32` sample buffer.

---

## Phase 2 — Whisper.cpp Integration

Wire the audio buffer into the whisper.cpp Go bindings.

### Steps

1. Clone whisper.cpp and build the static `libwhisper.a` library
2. Add `github.com/ggml-org/whisper.cpp/bindings/go/pkg/whisper` to `go.mod`
3. Download a GGML model file — start with `ggml-small.bin` (~470MB, good balance of speed and accuracy for ES/EN)
4. Load the model at startup and keep it in memory (don't reload on every transcription)
5. Implement transcription — pass `[]float32` samples, get segments back
6. Implement translation mode — `ctx.SetTranslate(true)` + `ctx.SetLanguage("es")` for ES→EN
7. Disable progress callbacks to avoid CGo overhead
8. Test: transcribe a pre-recorded WAV file in both Spanish and English

### Example

```go
model, _ := whisper.New("models/ggml-small.bin")
defer model.Close()

ctx, _ := model.NewContext()
ctx.SetLanguage("es")
ctx.SetTranslate(true) // ES → EN built-in, no Ollama needed

ctx.Process(samples, nil, nil, nil)

for {
    segment, err := ctx.NextSegment()
    if err != nil { break }
    fmt.Println(segment.Text)
}
```

**Deliverable:** A function `Transcribe(samples []float32, translate bool) (string, error)` that returns clean text.

---

## Phase 3 — Hotkey & Clipboard

Make the app feel like a system utility — press a key, speak, release, text appears.

### Steps

1. Add a global hotkey listener (`github.com/robotn/robotgo` or `golang.design/x/hotkey`)
2. Define two hotkeys with hardcoded defaults for now: one to start/stop recording, one to cycle modes — these become configurable in Phase 5
3. Implement push-to-talk behavior: hold to record, release to transcribe
4. Implement clipboard injection using `golang.design/x/clipboard`
5. After transcription, write text to clipboard and simulate Ctrl+V paste into the active window
6. Add a minimal menubar tray icon using `github.com/getlantern/systray` as a placeholder — gets replaced in Phase 9
7. Test: trigger hotkey, dictate a sentence in English, confirm it pastes into a text editor

**Deliverable:** Background app that listens for hotkey, records, transcribes, and pastes into any active window.

---

## Phase 4 — Translation Flow

Make ES↔EN translation work seamlessly as a separate mode.

### Steps

1. Add a second hotkey or toggle for "translate mode" vs "dictate mode"
2. Implement auto language detection — `ctx.SetLanguage("auto")` lets Whisper detect the input language
3. For ES→EN: `SetLanguage("es")` + `SetTranslate(true)`
4. For EN→ES: Whisper only translates *into* English natively — flag this for Phase 6 where Ollama handles it via a translate mode prompt
5. Show a small visual indicator in the tray icon when translate mode is active
6. Test: speak Spanish with translate mode on, confirm English text is pasted

**Deliverable:** Two working modes — dictate (transcribe as-is) and translate (ES→EN).

---

## Phase 5 — Config & Shortcut Customization

Make the app fully configurable without recompiling, including hotkeys.

### Steps

1. Create a `config.yaml` file with sane defaults:

```yaml
model: small          # tiny | small | medium
language: auto        # auto | es | en
ollama:
  enabled: false
  model: llama3.2:3b
  timeout_seconds: 3
hotkeys:
  record: "ctrl+shift+space"
  cycle_mode: "ctrl+shift+m"
  translate_toggle: "ctrl+shift+t"
modes:
  default: cleanup
```

2. Implement config loading at startup — parse hotkey strings into key combos the listener understands
3. Implement **hotkey rebinding at runtime** — when config changes on disk, reload hotkeys without restarting using `github.com/fsnotify/fsnotify`
4. Validate hotkey config on load — detect conflicts (two actions on the same key) and log a warning
5. Add model size selection with a comment in config explaining the speed/accuracy tradeoff:
   - `tiny` — fastest, least accurate (~75MB)
   - `small` — good balance (~470MB) ← recommended default
   - `medium` — most accurate, slower (~1.5GB)
6. Store last-used mode and language preference between restarts
7. Add a `--config` CLI flag to point to a custom config file path
8. Test: change the record hotkey in config, confirm the new shortcut works without restarting

**Deliverable:** Fully configurable app where all hotkeys are user-defined and hot-reloadable from `config.yaml`.

---

## Phase 6 — Ollama LLM Post-Processing

Add optional AI cleanup and formatting on top of the raw transcript.

### Steps

1. Spin up Ollama locally with a small model — `llama3.2:3b` or `qwen2.5:3b` are fast and lightweight
2. Implement an Ollama HTTP client in Go — `POST /api/generate`
3. Create a `Mode` struct:

```go
type Mode struct {
    Name         string
    SystemPrompt string
    LLMEnabled   bool
}
```

4. Implement a default "Cleanup" mode — removes filler words, fixes punctuation, preserves meaning
5. Pipe transcript through LLM only when a mode is active — raw transcription stays the fast default path
6. Implement EN→ES translation here as a mode with prompt: `"Translate the following text to Spanish, return only the translation"`
7. Add timeout handling — if Ollama takes more than the configured timeout, fall back to raw transcript and log a warning
8. Test: dictate a messy sentence with filler words, confirm cleanup mode polishes it

**Deliverable:** A `Process(text string, mode Mode) (string, error)` function that optionally enhances the transcript.

---

## Phase 7 — Custom Modes

Let the user define their own formatting modes — the equivalent of Superwhisper's paid Modes feature, running locally and free.

### Steps

1. Define modes in `config.yaml` as a list of name + system prompt pairs:

```yaml
modes:
  - name: cleanup
    llm: true
    prompt: "Clean up this transcript. Remove filler words, fix punctuation, keep the meaning intact. Return only the result."
  - name: formal
    llm: true
    prompt: "Rewrite this in a formal professional tone. Return only the result."
  - name: bullets
    llm: true
    prompt: "Convert this dictation into a concise bullet point list. Return only the result."
  - name: code_comment
    llm: true
    prompt: "Format this as a developer code comment or docstring. Return only the result."
  - name: translate_es
    llm: true
    prompt: "Translate the following text to Spanish. Return only the translation."
  - name: raw
    llm: false
    prompt: ""
```

2. The `cycle_mode` hotkey (defined in config) cycles through this list in order
3. Modes marked `llm: false` skip Ollama entirely — useful for a raw/fast mode
4. Users add unlimited custom modes without touching code
5. Test each built-in mode with sample dictation in English and Spanish

**Deliverable:** Full mode system working. User can cycle modes with a hotkey and add custom ones in `config.yaml`.

---

## Phase 8 — Polish & Reliability

Make it production-quality for daily use.

### Steps

1. Add audio feedback — a short system sound on recording start and stop
2. Implement proper error handling throughout:
   - Mic not found or unavailable
   - Model file missing or failed to load
   - Ollama unavailable (graceful fallback to raw transcript)
   - Hotkey conflict detected in config
3. Add a desktop notification showing the transcribed text after paste — useful if paste went to the wrong window
4. Handle long recordings gracefully — chunk audio into 30s segments and stitch the results
5. Add structured logging to `~/.config/gowhisper/gowhisper.log` for debugging transcription issues
6. Write a `Makefile` with the following targets:
   - `make build` — compile the binary
   - `make run` — build and run
   - `make test` — run all tests
   - `make install` — install to `/usr/local/bin`
   - `make download-model` — download the configured GGML model from Hugging Face
7. Test end-to-end across a full day of real usage

**Deliverable:** Stable, daily-driveable app with a working placeholder tray icon.

---

## Phase 9 — Native macOS UI (DarwinKit)

Replace the placeholder tray icon with a proper native macOS UI that matches the Superwhisper feel. Requires all previous phases to be stable before starting.

### 9a — Menubar Icon

1. Add DarwinKit (`github.com/progrium/darwinkit`) to `go.mod`
2. Replace the `systray` placeholder with a native `NSStatusBar` item
3. Show current mode name next to the icon (e.g. `● Cleanup`)
4. Add a native AppKit dropdown menu: switch modes, open config file, quit
5. Animate the icon (pulse effect) while recording is active

### 9b — Floating Recording Popup

1. Create a borderless `NSPanel` that floats above all other windows
2. Show it on hotkey press, hide it automatically after transcription completes
3. Display three states visually inside the panel:
   - 🎙 Mic waveform animation while recording
   - ⏳ Spinner while Whisper/Ollama is processing
   - ✓ Checkmark briefly on successful paste
4. Show the current mode name inside the popup (e.g. "Cleanup", "Translating...")
5. Position bottom-center of screen, matching Superwhisper's style
6. Make it dismissible with the Escape key

### Example Panel Setup

```go
import (
    "github.com/progrium/darwinkit/macos"
    "github.com/progrium/darwinkit/macos/appkit"
    "github.com/progrium/darwinkit/macos/foundation"
)

macos.RunApp(func(app appkit.Application, delegate *appkit.ApplicationDelegate) {
    frame := foundation.Rect{Size: foundation.Size{Width: 300, Height: 80}}

    panel := appkit.NewWindowWithContentRectStyleMaskBackingDefer(
        frame,
        appkit.BorderlessWindowMask|appkit.NonactivatingPanelMask,
        appkit.BackingStoreBuffered,
        false,
    )
    panel.SetLevel(appkit.FloatingWindowLevel) // always on top
    panel.SetOpaque(false)
    panel.Center()
    panel.MakeKeyAndOrderFront(nil)
})
```

### 9c — Settings Panel (Optional)

1. Build a simple native `NSWindow` settings panel
2. Show all config options as native form controls — dropdowns for model/language, text fields for hotkeys
3. Write changes back to `config.yaml` on save, triggering the file watcher from Phase 5 to hot-reload automatically

**Deliverable:** A polished, native-feeling macOS app indistinguishable from a Swift-built tool.

---

## Phase 10 — Optional Extras (Post-MVP)

Nice-to-haves once everything is solid.

| Feature | Description |
|---|---|
| History log | Save all transcriptions with timestamps to a local SQLite file |
| Transcribe from file | Drag and drop an audio file onto the tray icon to transcribe it |
| Streaming transcription | Show text appearing in real time as you speak instead of waiting |
| Auto-update models | CLI command to download newer GGML models from Hugging Face |

---

## Build Order Summary

| Phase | Priority | Effort | Depends On |
|---|---|---|---|
| 1 — Audio Capture | Must | Medium | — |
| 2 — Whisper Integration | Must | Medium | Phase 1 |
| 3 — Hotkey & Clipboard | Must | Low | Phase 2 |
| 4 — Translation | Must | Low | Phase 2 |
| 5 — Config & Shortcuts | Must | Low | Phase 3 |
| 6 — Ollama Cleanup | Should | Medium | Phase 5 |
| 7 — Custom Modes | Should | Low | Phase 6 |
| 8 — Polish | Should | Medium | Phase 7 |
| 9 — Native macOS UI | Should | High | Phase 8 |
| 10 — Extras | Nice | High | Phase 9 |

---

## Milestones

- **Phases 1–4** → Working MVP. Dictate and translate with a hotkey, pastes into any window.
- **Phase 5** → Fully configurable hotkeys and settings.
- **Phases 6–7** → Superwhisper paid-tier features (Modes), running locally and free.
- **Phase 8** → Production-quality, daily-driveable.
- **Phase 9** → Looks and feels like a real native Mac app.