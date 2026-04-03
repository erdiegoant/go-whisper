# Phase 17 — Distribution & Store Setup (Pro)

> **Goal:** Accept payments and deliver the Pro DMG automatically.

## Store

**Gumroad** at launch (fastest to set up, ~10% fee).
Migrate to **Lemon Squeezy** if monthly volume justifies it (~5% fee, better VAT handling,
built-in license key API for future use).

## Flow

```
GitHub repo (public)
  └── Releases → free unsigned binary (developer setup, stays free forever)

Landing page (Carrd or GitHub Pages)
  └── "Download Free" → GitHub Releases
  └── "Buy Pro — $39" → Gumroad product page
        └── Payment → buyer receives .dmg download link via email
```

## Landing page must-haves

- One-line hook ("Your voice, queryable by AI agents")
- Free vs Pro comparison table
- MCP demo video (30 seconds — speak → Claude Desktop acts on it)
- Single CTA button: Buy Pro $39
