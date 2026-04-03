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

## Package structure

Add to the existing `internal/mode/` package (do **not** create a new `internal/modes/`
package):

```
internal/mode/
  mode.go           — existing Mode / Manager types (unchanged)
  builtin_free.go   //go:build !pro — Standard + ES→EN only
  builtin_pro.go    //go:build pro  — all 7 modes above
```

Each file exports a `BuiltinModes() []Mode` function. `Manager` uses this to seed its
default mode list when no custom modes are configured in `config.yaml`.
