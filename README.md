# GoWhisper

A Superwhisper-inspired voice dictation and translation app for macOS, built in Go. Press a hotkey, speak, press it again — your words are transcribed and pasted into whatever app you're using. Runs fully locally with no cloud dependency.

## Features

- **Toggle recording** — press ⌥Space to start, press again to stop and paste
- **Cancel recording** — press Esc to discard mid-recording, nothing is pasted
- **Cycle modes** — press ⌥⇧K to switch between Standard, Translate, and custom modes
- **ES → EN translation** — Whisper's native translation, no LLM needed
- **LLM post-processing** — optional Claude API integration for cleanup, formatting, and custom prompts
- **Custom modes** — define your own prompts in `config.yaml`, cycle through them with a hotkey or pick from the tray menu
- **Hot-reloadable config** — change hotkeys or models without restarting
- **No cloud, no subscription** — everything runs on your machine

## Stack

| Component | Technology |
|---|---|
| Language | Go |
| Speech-to-text | [whisper.cpp](https://github.com/ggerganov/whisper.cpp) (Metal GPU accelerated) |
| Audio capture | [malgo](https://github.com/gen2brain/malgo) (miniaudio — no Homebrew required) |
| Global hotkeys | [golang.design/x/hotkey](https://pkg.go.dev/golang.design/x/hotkey) |
| Clipboard | CGo + NSPasteboard (AppKit) + CGEventPost |
| Menubar icon | [fyne.io/systray](https://github.com/fyne-io/systray) |
| LLM post-processing | Claude API via `net/http` (optional, key via env or config) |
| Native macOS UI | [DarwinKit](https://github.com/progrium/darwinkit) (Phase 9) |
| Config | `config.yaml` with live file watching (Phase 5) |

## Hotkeys (default)

| Action | Shortcut |
|---|---|
| Toggle recording | ⌥Space |
| Cancel recording | Esc |
| Cycle mode | ⌥⇧K |

All hotkeys are configurable in `config.yaml` (Phase 5).

## Requirements

- macOS 13.0+
- [Xcode Command Line Tools](https://developer.apple.com/xcode/resources/) — `xcode-select --install`
- [cmake](https://cmake.org) — `brew install cmake`
- Claude API key (optional, for LLM transcript cleanup — set `ANTHROPIC_API_KEY` or add to `config.yaml`)

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

Compiles whisper.cpp into static libraries inside `third_party/whisper.cpp/build/`. Only needed once (or after updating the submodule).

### 3. Download a model

```bash
make download-model
```

Downloads `ggml-small.bin` (~465MB) to `~/.config/gowhisper/models/`. Recommended for a good balance of speed and accuracy with Spanish and English.

| Model | Size | Notes |
|---|---|---|
| tiny | ~75MB | Fastest, lower accuracy |
| small | ~465MB | Recommended |
| medium | ~1.5GB | Most accurate, slower |

### 4. Run

```bash
# Development (faster — no compile step)
make dev

# Or build a binary and run it
make run
```

On first launch, macOS will prompt for:
- **Microphone access** — required to capture your voice
- **Accessibility access** — required for global hotkeys and simulated paste (System Settings → Privacy & Security → Accessibility)

> **Note:** When using `make dev`, mic access is granted to your terminal app, not the binary. If recording captures only silence, open System Settings → Privacy & Security → Microphone and ensure your terminal is listed and enabled. Use `make rectest` to verify mic access before running the full app.

The app lives in your menubar. You'll see `⚫ Standard` when idle.

## Usage

1. Press **⌥Space** — icon changes to `🔴 Standard`, recording starts
2. Speak
3. Press **⌥Space** again — icon changes to `⏳ Standard`, transcription runs
4. Text is pasted into whatever window was active — icon returns to `⚫ Standard`

Press **Esc** at any point while recording to cancel (nothing is pasted).

## Configuration

Config lives at `~/.config/gowhisper/config.yaml` (created automatically on first launch):

```yaml
model: small
language: auto
models_dir: "~/.config/gowhisper/models"
max_recording_seconds: 120

claude:
  api_key: ""             # leave empty to use ANTHROPIC_API_KEY env var
  model: "claude-haiku-4-5-20251001"
  timeout_seconds: 15

hotkeys:
  toggle_recording: "option+space"
  cancel_recording: "esc"
  change_mode: "option+shift+k"

# Custom modes — omit this block to use the built-in Standard + Translate defaults.
# modes:
#   - name: Standard
#     language: auto
#   - name: Formal
#     language: auto
#     prompt: "Rewrite this transcript in a formal professional tone. Preserve all technical terms. Return only the result."
#   - name: Bullets
#     language: auto
#     prompt: "Convert this dictation into a concise bullet point list. Return only the result."
```

All changes are applied live on save — no restart required.

## Makefile Targets

```bash
make whisper          # Build whisper.cpp static libraries (once)
make build            # Compile the Go binary into GoWhisper.app
make run              # Build and run the compiled binary
make dev              # Run directly with go run (faster for development)
make test             # Run all tests
make install          # Install GoWhisper.app to /Applications
make download-model   # Download ggml-small.bin to ~/.config/gowhisper/models/
make rectest          # Record 5s to /tmp/rectest.wav — diagnose mic access (DEV="name" to pick device)
make clean            # Remove build artifacts
```

## Project Structure

```
cmd/gowhisper/        # Main entry point and event loop
cmd/rectest/          # Standalone mic recording test (5s WAV capture)
internal/
  audio/              # Mic capture, recording state machine, device selection
  transcribe/         # Whisper.cpp integration, TranscribeRequest
  hotkey/             # Global hotkeys (toggle always-on, Esc only while recording)
  clipboard/          # NSPasteboard save/restore + Cmd+V simulation via CGo
  config/             # Config loading and file watcher (Phase 5)
  llm/                # Claude API HTTP client for transcript cleanup
  ui/                 # Menubar tray icon + microphone device submenu
third_party/
  whisper.cpp/        # whisper.cpp source (git submodule)
GoWhisper.app/        # macOS app bundle with Info.plist
phases/               # Development plan (phase-by-phase)
```

## Development Status

| Phase | Description | Status |
|---|---|---|
| 1 | Audio capture | ✅ Done |
| 2 | Whisper.cpp integration | ✅ Done |
| 3 | Hotkey & clipboard | ✅ Done |
| 4 | Translation flow | ✅ Done |
| 5 | Config & shortcuts | ✅ Done |
| 6 | LLM transcript cleanup (Claude API) | ✅ Done |
| 7 | Custom modes | ✅ Done |
| 8 | Polish & reliability | Not started |
| 9 | Native macOS UI (DarwinKit) | Not started |
| 10 | Optional extras | Not started |

## License

MIT
