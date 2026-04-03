# Phase 14 — Settings UI Window (Pro)

> **Goal:** Non-technical users can configure GoWhisper without editing `config.yaml`.

## Approach decision (required before starting)

DarwinKit was abandoned — it is **not** a viable option. Choose one:

| Option | Pros | Cons |
| ------ | ---- | ---- |
| `github.com/webview/webview` (recommended) | Pure Go + HTML/CSS/JS; no CGo complexity; easy to iterate on UI | Adds a webview dep; non-native look |
| Minimal CGo + raw AppKit | Truly native; zero extra deps | Verbose, brittle, painful to build UI in |
| Defer to post-v1 | Ships faster; tray-only config is already functional | Pro users without YAML experience are stuck |

Recommendation: **webview**. The settings window is infrequently opened — native polish
is less important than shipping it at all.

## Tabs

- **General** — model selection, language, max recording length, sound/notifications
- **Modes** — add/edit/delete custom modes with name + prompt textarea
- **Hotkeys** — toggle, cancel, mode cycle
- **LLM** — Claude API key + model, Ollama host + model, cleanup toggle
- **MCP** — show MCP subcommand path; "Install Claude Desktop Config" button

## Implementation notes

- All changes write back to `config.yaml` directly
- Hot-reload is already wired — saving config triggers immediate apply, no restart needed
- Free build: keep tray-only configuration (no settings window)
- Build tag: `//go:build pro` for the window code; stub for free builds
