# MCP Server

PieKBS exposes KB tools via the MCP protocol. Two transport modes are supported.

## HTTP Mode (Recommended)

One PieKBS process shared by all agents — Claude Code, Cursor, VS Code (Copilot), Windsurf, and others.

**Start PieKBS:**

```bash
export PIEKBS_KB=/path/to/piekbs-kb
piekbs serve
```

**Configure each agent:**

```json
{
  "mcpServers": {
    "piekbs": {
      "type": "http",
      "url": "http://127.0.0.1:8766/mcp",
      "headers": {
        "x-api-key": "${PIEKBS_API_KEY}"
      }
    }
  }
}
```

`x-api-key` corresponds to `server.api_key` in `config.yaml`. Omit `headers` if no api_key is set.

## stdio Mode

For hosted environments (Hermes, OpenClaw, etc.) where PieKBS runs as a subprocess.

```json
{
  "mcpServers": {
    "piekbs": {
      "type": "stdio",
      "command": "/path/to/piekbs",
      "args": ["stdio"],
      "env": {
        "PIEKBS_KB": "/path/to/your-kb"
      }
    }
  }
}
```

The KB directory is created automatically on first launch. No manual `init` needed.

## Available Tools

| Tool | Description |
|---|---|
| `kb_search` | FTS keyword search, returns ranked results with related links |
| `kb_page` | Fetch full page content by ID |
| `kb_add` | Add a text document to the KB; triggers incremental indexing and async distillation |

Admin operations (`status`, `reindex`, `lint`) are available via the Web UI or CLI.
