# GoWhisper — Development Plan

A Superwhisper-inspired voice dictation and translation app for macOS, built in Go using
whisper.cpp, with optional LLM cleanup via Claude API or Ollama.

---

## Product Tiers

GoWhisper ships as two editions compiled from the **same repository** using Go build tags.

### Free (open source)
- Distributed via **GitHub Releases** as an unsigned binary
- Developer-friendly setup (clone, cmake, make whisper, download model)
- Core dictation, translation, custom modes via `config.yaml`, transcription history
- This is the top-of-funnel — it stays open source indefinitely

### Pro — $39 one-time
- Distributed as a **notarized `.dmg`** via Gumroad / Lemon Squeezy
- No Gatekeeper warning, first-launch onboarding, model download built-in
- Adds everything in the Pro feature set (see phases below)
- Compiled with `-tags pro`; Pro packages are no-ops in free builds via stub files
- No license key enforcement at v1 — the notarized DMG experience is the value

**No subscriptions. No cloud requirement. One price, own it forever.**

---

## Build Tag Strategy

Pro features live in files guarded by `//go:build pro`. Every Pro package has a matching
stub file guarded by `//go:build !pro` so the codebase compiles cleanly either way.

```
internal/
  mcp/
    server.go       //go:build pro   ← real MCP server
    stub.go         //go:build !pro  ← no-op Start()
  modes/
    builtin_pro.go  //go:build pro   ← full curated mode library
    builtin_free.go //go:build !pro  ← Standard + Translate only
  ui/
    settings_pro.go //go:build pro   ← settings window
    settings_free.go //go:build !pro ← tray only (current behaviour)
```

Makefile targets:
```makefile
build-free:    go build ./cmd/gowhisper
build-pro:     go build -tags pro ./cmd/gowhisper
release-pro:   go build -tags pro -ldflags="-s -w" ./cmd/gowhisper
               # → package into .app bundle → notarize → DMG
```

---

## Progress Tracker

| Phase | Title | Edition | Status |
| ----- | ----- | ------- | ------ |
| 1  | Foundation & Audio Capture              | Free | ✅ Done |
| 2  | Whisper.cpp Integration                 | Free | ✅ Done |
| 3  | Hotkey & Clipboard                      | Free | ✅ Done |
| 4  | Translation Flow                        | Free | ✅ Done |
| 5  | Config & Shortcut Customization         | Free | ✅ Done |
| 6  | LLM Transcript Cleanup (Claude API)     | Free | ✅ Done |
| 7  | Custom Modes                            | Free | ✅ Done |
| 8  | Polish & Reliability                    | Free | ✅ Done |
| 9  | Native macOS UI (SwiftUI)               | Pro  | ⏸ Postponed |
| 10 | Local LLM Backend (Ollama)              | Free | ✅ Done |
| 11 | Optional Extras                         | Free | 🔄 In progress — retention ✅, clear ✅, CLI path ✅, help/history CLI ✅, transcribe from file ⏳ |
| 12 | MCP Server                              | Pro  | 🔜 Next |
| 13 | Pro Mode Library                        | Pro  | 🔜 Planned |
| 14 | Settings UI Window                      | Pro  | 🔜 Planned |
| 15 | Notarization & DMG Packaging            | Pro  | 🔜 Planned |
| 16 | First-launch Onboarding                 | Pro  | 🔜 Planned |
| 17 | Distribution & Store Setup              | Pro  | 🔜 Planned |

---

## Stack

| Layer | Technology |
| ----- | ---------- |
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
| MCP Server (Pro) | `github.com/modelcontextprotocol/go-sdk` over stdio |
| Settings UI (Pro) | AppKit via DarwinKit (light usage) |
