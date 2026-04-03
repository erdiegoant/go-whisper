# Phase 12 — MCP Server (Pro)

> **Goal:** Let any MCP-compatible client (Claude Desktop, Cursor, local agents) query
> GoWhisper's transcript history and act on it.
> "Speak → GoWhisper transcribes → agent reads your voice notes → files a ticket."

## Transport

`stdio` — the MCP server runs as a subprocess spawned by the MCP client. GoWhisper
exposes it as a subcommand: `gowhisper mcp`.

## SDK

`github.com/modelcontextprotocol/go-sdk` — handles JSON-RPC 2.0 framing so the
implementation focuses on tool definitions and SQLite queries only.

## History package extensions required

The MCP tools wrap `internal/history`. Before implementing tools, add these methods to
`internal/history/history.go`:

| Method | Purpose |
| ------ | ------- |
| `ByID(id int64) (Entry, error)` | Single entry fetch |
| `Search(query string) ([]Entry, error)` | Full-text search |
| `ByDate(from, to time.Time) ([]Entry, error)` | Date range filter |

Also add **FTS5** support in the migration: create a `transcriptions_fts` virtual table
keyed to `raw_text` + `processed_text`, with insert/delete triggers to keep it in sync.
`modernc.org/sqlite` supports FTS5 out of the box.

## Tools to expose

| Tool | Description |
| ---- | ----------- |
| `get_recent_transcripts` | Last N transcripts. Optional `mode` filter. Default N=10. |
| `search_transcripts` | Full-text search against transcript content (FTS5). |
| `get_transcripts_by_date` | Filter by `today`, `this_week`, or an ISO date range. |
| `get_transcript_by_id` | Fetch a single transcript by ID. |

**Response struct:** Tool responses use a filtered struct — do **not** include `PromptUsed`
in MCP responses by default. It bloats agent context with verbose system prompt text.

```go
type MCPEntry struct {
    ID             int64
    Timestamp      string
    ModeName       string
    RawText        string
    ProcessedText  string
    DurationMs     int64
    Language       string
}
```

## MCP Resources

Design the server so Resources can be added later without a rewrite. Transcript history
is a natural fit for a Resource (readable, subscribable). Not required for v1, but don't
paint the architecture into a corner.

## Package structure

```
internal/
  mcp/
    server.go        //go:build pro  — server init, stdio transport, tool registration
    tools.go         //go:build pro  — tool handler functions (thin wrappers over history pkg)
    stub.go          //go:build !pro — no-op Start() for free builds
cmd/gowhisper/
  main.go            — adds `mcp` subcommand: if args[1] == "mcp" { mcp.Start() }
```

## Claude Desktop config (what users add to `claude_desktop_config.json`)

```json
{
  "mcpServers": {
    "gowhisper": {
      "command": "gowhisper",
      "args": ["mcp"]
    }
  }
}
```

Requires `gowhisper` to be in `$PATH` (see Phase 11 "Add CLI to Path" feature). During
development, use the absolute path to the built binary.

### `gowhisper mcp --install`

Add a `--install` flag to the `mcp` subcommand that auto-writes (or updates) the
`gowhisper` entry in `~/Library/Application Support/Claude/claude_desktop_config.json`.
Prints the path and a note to restart Claude Desktop.

## Unit tests

`internal/history` and `internal/mcp/tools.go` are pure Go with no hardware dependency —
write unit tests for both. Use a temporary in-memory SQLite DB for history tests.

## Example agent interactions

- "Summarize everything I dictated today"
- "Turn my last voice note into a GitHub issue"
- "Find all transcripts where I mentioned the API refactor"
- "Draft a standup from my dictations since yesterday"
