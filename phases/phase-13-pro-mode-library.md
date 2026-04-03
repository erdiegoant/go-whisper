# Phase 13 — Pro Mode Library (Pro)

> **Goal:** Ship 6 curated modes out of the box so Pro users get immediate value without
> touching `config.yaml`.

| Mode | Description |
| ---- | ----------- |
| Standard | Raw transcription, language auto-detected (same as free) |
| ES → EN | Speak Spanish, get professionally rewritten English output |
| PR Description | Dictate context, get a formatted GitHub PR body |
| Standup | Speak yesterday/today/blockers, get a Slack-ready standup |
| Bug Report | Describe a bug verbally, get a structured report |
| Meeting Notes | Stream of consciousness → clean bullet-point summary |
| Formal Email | Casual dictation → professional email draft |

These ship as Go constants in `internal/modes/builtin_pro.go` (build tag: `pro`).
The free build only has Standard and ES→EN in `builtin_free.go`.
