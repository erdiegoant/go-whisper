# Phase 15 — Notarization & DMG Packaging (Pro)

> **Goal:** Pro users download a `.dmg` that opens without a Gatekeeper warning.

## Version string

Embed a version at build time via ldflags. Add to `cmd/gowhisper/main.go`:

```go
var version = "dev" // overridden by -ldflags "-X main.version=1.x.x"
```

Expose via `gowhisper --version` flag. Used in user support, update checking, and DMG
filename (`GoWhisper-1.0.0-pro.dmg`).

## Makefile targets

Add the following targets (none currently exist for Pro builds):

```makefile
build-free:
	go build -ldflags "-X main.version=$(VERSION)" ./cmd/gowhisper

build-pro:
	go build -tags pro -ldflags "-X main.version=$(VERSION)" ./cmd/gowhisper

release-pro:
	go build -tags pro -ldflags "-s -w -X main.version=$(VERSION)" -o GoWhisper.app/Contents/MacOS/gowhisper ./cmd/gowhisper
	# → codesign → notarize → create DMG

install-cli:
	ln -sf /Applications/GoWhisper.app/Contents/MacOS/gowhisper /usr/local/bin/gowhisper
```

## Signing & notarization

- Apple Developer Program enrollment ($99/year)
- Code signing with Developer ID Application certificate
- `xcrun notarytool` submission and stapling
- DMG layout: GoWhisper.app + Applications shortcut alias
