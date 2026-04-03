# Phase 15 — Notarization & DMG Packaging (Pro)

> **Goal:** Pro users download a `.dmg` that opens without a Gatekeeper warning.

- Apple Developer Program enrollment ($99/year)
- Code signing with Developer ID Application certificate
- `xcrun notarytool` submission and stapling
- DMG layout: GoWhisper.app + Applications shortcut alias
- CI target in Makefile: `make release-pro` → signed → notarized → DMG
