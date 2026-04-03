# Phase 14 — Settings UI Window (Pro)

> **Goal:** Non-technical users can configure GoWhisper without editing `config.yaml`.

- Basic native window (AppKit via DarwinKit or a minimal CGo wrapper)
- Tabs: General, Modes, Hotkeys, LLM, MCP
- Modes tab: add/edit/delete custom modes with a name + prompt textarea
- All changes write back to `config.yaml` (hot-reload already handles the rest)
- Free build keeps tray-only configuration
