# GoWhisper — Development Plan

A Superwhisper-inspired voice dictation and translation app for macOS, built in Go using whisper.cpp, with optional LLM cleanup via Claude API or Ollama.

## Progress Tracker

| Phase | Title | Status |
|---|---|---|
| 1 | Foundation & Audio Capture | ✅ Done |
| 2 | Whisper.cpp Integration | ✅ Done |
| 3 | Hotkey & Clipboard | ✅ Done |
| 4 | Translation Flow | ✅ Done |
| 5 | Config & Shortcut Customization | ✅ Done |
| 6 | LLM Transcript Cleanup (Claude API) | ✅ Done |
| 7 | Custom Modes | ✅ Done |
| 8 | Polish & Reliability | ✅ Done |
| 9 | Native macOS UI (SwiftUI) | ⏸ Postponed |
| 10 | Local LLM Backend (Ollama) | ✅ Done |
| 11 | Optional Extras | 🔄 In progress — history ✅, transcribe from file ⏳ |

---

## Milestones

- **Phases 1–4** — Working MVP. Dictate and translate with a hotkey, pastes into any window. ✅
- **Phase 5** — Fully configurable hotkeys and settings. ✅
- **Phases 6–7** — Superwhisper paid-tier features (Modes + LLM cleanup), running locally and free. ✅
- **Phase 8** — Production-quality, daily-driveable. ✅
- **Phase 9** — Native SwiftUI UI. Postponed indefinitely — DarwinKit abandoned, SwiftUI rewrite deferred.
- **Phase 10** — Local LLM via Ollama. ✅
- **Phase 11** — Post-MVP extras. History log shipped. Transcribe from file planned.

---

## Stack

| Layer | Technology |
|---|---|
| Language | Go |
| STT Engine | whisper.cpp (CGo bindings, Metal GPU accelerated) |
| Audio Capture | miniaudio via `github.com/gen2brain/malgo` |
| Hotkeys | `golang.design/x/hotkey` |
| Clipboard | CGo + NSPasteboard + CGEventPost |
| LLM Post-processing | Claude API or Ollama — both via `net/http`, both optional |
| Menubar UI | `fyne.io/systray` |
| Config | `config.yaml` with `fsnotify` file watcher |
| History | SQLite via `modernc.org/sqlite` |
| Notifications | `NSUserNotificationCenter` via CGo |

---

## v1.0.0 — Shipped

First public release. Apple Silicon only (M1+), macOS 13+, unsigned.

Key features at release:
- Toggle recording with ⌥Space, cancel with Esc, cycle modes with ⌥⇧K
- Whisper.cpp transcription with Metal acceleration
- ES → EN translation natively via Whisper
- Optional LLM cleanup (Claude API or Ollama)
- Custom modes with per-mode prompts and a global prompt override
- Model management from the tray (download, switch, auto-update check)
- Transcription history in tray (SQLite-backed)
- Hot-reloadable config
- Graceful first-launch: prompts to download a model if none is installed

## What's Next

- **Transcribe from file** — drag an audio file onto the tray to transcribe it (see `phases/phase-11-extras.md`)
- **Phase 9 (Native UI)** — postponed, revisit when time allows
