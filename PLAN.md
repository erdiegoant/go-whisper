# GoWhisper — Development Plan

A Superwhisper-inspired voice dictation and translation app for macOS, built in Go using whisper.cpp, Ollama, and DarwinKit.

## Progress Tracker

| Phase | Title | Status | File |
|---|---|---|---|
| 1 | Foundation & Audio Capture | [x] Done | [phase-1-audio-capture.md](phases/phase-1-audio-capture.md) |
| 2 | Whisper.cpp Integration | [x] Done | [phase-2-whisper-integration.md](phases/phase-2-whisper-integration.md) |
| 3 | Hotkey & Clipboard | [ ] Not Started | [phase-3-hotkey-clipboard.md](phases/phase-3-hotkey-clipboard.md) |
| 4 | Translation Flow | [ ] Not Started | [phase-4-translation.md](phases/phase-4-translation.md) |
| 5 | Config & Shortcut Customization | [ ] Not Started | [phase-5-config-shortcuts.md](phases/phase-5-config-shortcuts.md) |
| 6 | Ollama LLM Post-Processing | [ ] Not Started | [phase-6-ollama-llm.md](phases/phase-6-ollama-llm.md) |
| 7 | Custom Modes | [ ] Not Started | [phase-7-custom-modes.md](phases/phase-7-custom-modes.md) |
| 8 | Polish & Reliability | [ ] Not Started | [phase-8-polish.md](phases/phase-8-polish.md) |
| 9 | Native macOS UI (DarwinKit) | [ ] Not Started | [phase-9-native-macos-ui.md](phases/phase-9-native-macos-ui.md) |
| 10 | Optional Extras (Post-MVP) | [ ] Not Started | [phase-10-extras.md](phases/phase-10-extras.md) |

**Statuses:** `[ ] Not Started` → `[~] In Progress` → `[x] Done`

---

## Milestones

- **Phases 1–4** → Working MVP. Dictate and translate with a hotkey, pastes into any window.
- **Phase 5** → Fully configurable hotkeys and settings.
- **Phases 6–7** → Superwhisper paid-tier features (Modes), running locally and free.
- **Phase 8** → Production-quality, daily-driveable.
- **Phase 9** → Looks and feels like a real native Mac app.
- **Phase 10** → Post-MVP extras (history, streaming, file drop).

---

## Stack

| Layer | Technology |
|---|---|
| Language | Go |
| STT Engine | whisper.cpp (via Go bindings) |
| Audio Capture | miniaudio via `github.com/gen2brain/malgo` (no system install, CoreAudio-backed) |
| Hotkeys | `golang.design/x/hotkey` or `github.com/robotn/robotgo` |
| Clipboard | `golang.design/x/clipboard` |
| LLM Post-processing | Ollama (local, optional) |
| Native macOS UI | DarwinKit (`github.com/progrium/darwinkit`) |
| Config | `config.yaml` with file watcher (`github.com/fsnotify/fsnotify`) |

---

## How to Use This Plan

- **Start a phase:** read its file in `phases/`, mark status `[~] In Progress` in the table above.
- **Finish a phase:** mark `[x] Done` in the table, move to the next phase file.
- **Only load the current phase file** — do not read ahead phases unless checking dependencies.
- **Current phase to start:** Phase 1 — [phase-1-audio-capture.md](phases/phase-1-audio-capture.md)
