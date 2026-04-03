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

## Tools to expose

| Tool | Description |
| ---- | ----------- |
| `get_recent_transcripts` | Last N transcripts. Optional `mode` filter. Default N=10. |
| `search_transcripts` | Full-text search against transcript content. |
| `get_transcripts_by_date` | Filter by `today`, `this_week`, or an ISO date range. |
| `get_transcript_by_id` | Fetch a single transcript by ID. |

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
      "command": "/Applications/GoWhisper.app/Contents/MacOS/GoWhisper",
      "args": ["mcp"]
    }
  }
}
```

## Example agent interactions

- "Summarize everything I dictated today"
- "Turn my last voice note into a GitHub issue"
- "Find all transcripts where I mentioned the API refactor"
- "Draft a standup from my dictations since yesterday"
