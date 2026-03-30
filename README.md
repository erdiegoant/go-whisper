# GoWhisper

A Superwhisper-inspired voice dictation and translation app for macOS, built in Go. Press a hotkey, speak, press it again — your words are transcribed and pasted into whatever app you're using. Runs fully locally with no cloud dependency.

## Features

- **Toggle recording** — press to start, press again to stop and paste
- **Cancel recording** — discard a recording mid-way with Esc
- **ES → EN translation** — Whisper's native translation, no LLM needed
- **LLM post-processing** — optional Ollama integration for cleanup, formatting, and custom modes (formal tone, bullet points, code comments, etc.)
- **Custom modes** — define your own prompts in `config.yaml`, cycle through them with a hotkey
- **Hot-reloadable config** — change hotkeys or models without restarting
- **No cloud, no subscription** — everything runs on your machine

## Stack

| Component | Technology |
|---|---|
| Language | Go |
| Speech-to-text | [whisper.cpp](https://github.com/ggerganov/whisper.cpp) |
| Audio capture | [malgo](https://github.com/gen2brain/malgo) (miniaudio — no Homebrew required) |
| LLM post-processing | [Ollama](https://ollama.com) (local, optional) |
| Native macOS UI | [DarwinKit](https://github.com/progrium/darwinkit) |
| Config | `config.yaml` with live file watching |

## Hotkeys (default)

| Action | Shortcut |
|---|---|
| Toggle recording | ⌥Space |
| Cancel recording | Esc |
| Cycle mode | ⌥⇧K |

All hotkeys are configurable in `config.yaml`.

## Requirements

- macOS 13.0+
- [Xcode Command Line Tools](https://developer.apple.com/xcode/resources/) — `xcode-select --install`
- [cmake](https://cmake.org) — `brew install cmake`
- [Ollama](https://ollama.com) (optional, for LLM modes)

## Getting Started

### 1. Clone with submodules

```bash
git clone --recurse-submodules https://github.com/erdiegoant/go-whisper.git
cd go-whisper
```

### 2. Build whisper.cpp

```bash
make whisper
```

This compiles whisper.cpp into static libraries inside `third_party/whisper.cpp/build/`. Only needed once (or after updating the submodule).

### 3. Download a model

```bash
make download-model
```

Downloads `ggml-small.bin` (~465MB) to `~/.config/gowhisper/models/`. Recommended for a good balance of speed and accuracy with Spanish and English.

Available model sizes (configure in `config.yaml`):

| Model | Size | Speed | Accuracy |
|---|---|---|---|
| tiny | ~75MB | Fastest | Lower |
| small | ~465MB | Good | Recommended |
| medium | ~1.5GB | Slower | Highest |

### 4. Build and run

```bash
make run
```

On first launch, macOS will prompt for:
- **Microphone access** — required to capture your voice
- **Accessibility access** — required for global hotkeys and simulated paste (System Settings → Privacy & Security → Accessibility)

## Configuration

The config file lives at `~/.config/gowhisper/config.yaml` and is created on first run. Example:

```yaml
model: small
language: auto
models_dir: "~/.config/gowhisper/models"
max_recording_seconds: 120

ollama:
  enabled: false
  endpoint: "http://localhost:11434"
  model: llama3.2:3b
  timeout_seconds: 10

hotkeys:
  toggle_recording: "option+space"
  cancel_recording: "esc"
  change_mode: "option+shift+k"

modes:
  - name: raw
    llm: false
  - name: cleanup
    llm: true
    prompt: "Clean up this transcript. Remove filler words, fix punctuation, keep the meaning intact. Return only the result."
  - name: formal
    llm: true
    prompt: "Rewrite this in a formal professional tone. Return only the result."
  - name: bullets
    llm: true
    prompt: "Convert this dictation into a concise bullet point list. Return only the result."
```

Changes to `config.yaml` are picked up live — no restart needed.

## Makefile Targets

```bash
make whisper          # Build whisper.cpp static libraries (once)
make build            # Compile the Go binary into GoWhisper.app
make run              # Build and run
make test             # Run all tests
make install          # Install GoWhisper.app to /Applications
make download-model   # Download ggml-small.bin to ~/.config/gowhisper/models/
make clean            # Remove build artifacts
```

## Project Structure

```
cmd/gowhisper/        # Main entry point
internal/
  audio/              # Mic capture, recording state machine
  transcribe/         # Whisper.cpp integration
  hotkey/             # Global hotkey listener
  clipboard/          # Clipboard injection and paste
  config/             # Config loading and file watcher
  llm/                # Ollama HTTP client
  ui/                 # Native macOS UI (DarwinKit)
third_party/
  whisper.cpp/        # whisper.cpp source (git submodule)
GoWhisper.app/        # macOS app bundle
phases/               # Development plan (phase-by-phase)
```

## Development Status

| Phase | Description | Status |
|---|---|---|
| 1 | Audio capture | Done |
| 2 | Whisper.cpp integration | In Progress |
| 3 | Hotkey & clipboard | Not started |
| 4 | Translation flow | Not started |
| 5 | Config & shortcuts | Not started |
| 6 | Ollama LLM post-processing | Not started |
| 7 | Custom modes | Not started |
| 8 | Polish & reliability | Not started |
| 9 | Native macOS UI (DarwinKit) | Not started |
| 10 | Optional extras | Not started |

## License

MIT
