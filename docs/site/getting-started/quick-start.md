# Quick Start

## 1. Initialize a Knowledge Base

```bash
export WIKILOOP_KB=/path/to/your-kb

piekbs init    # scaffold KB dirs and copy schema/templates
piekbs serve   # start server: MCP + Web UI + file watcher
```

> On macOS, double-click PieKBS.app to launch as a menubar icon.

## 2. Configure MCP in Your Agent

**HTTP mode** (recommended — one process shared by all agents):

Add to `~/.claude.json` under `mcpServers`:

```json
{
  "mcpServers": {
    "piekbs": {
      "type": "http",
      "url": "http://127.0.0.1:8766/mcp",
      "headers": {
        "x-api-key": "${WIKILOOP_API_KEY}"
      }
    }
  }
}
```

`x-api-key` corresponds to `server.api_key` in `config.yaml`. Omit `headers` if no api_key is set.

**stdio mode** (for hosted environments):

```json
{
  "mcpServers": {
    "piekbs": {
      "type": "stdio",
      "command": "/path/to/piekbs",
      "args": ["stdio"],
      "env": {
        "WIKILOOP_KB": "/path/to/your-kb"
      }
    }
  }
}
```

## 3. Add Content

Drop any Markdown file into `raw/`. The watcher automatically distills it.

```bash
cp my-notes.md $WIKILOOP_KB/raw/
# watcher detects the file and triggers distill + reindex automatically
```

## 4. Search

```bash
piekbs search "your query"
```

Or let your agent use the MCP tools:

```text
kb_search("keyword A")          → discover relevant documents
kb_search("keyword B")          → cover a different angle
kb_page(["id1", "id2"])         → deep-read the most relevant ones
```
