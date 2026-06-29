# CLI Reference

All commands accept a global `--kb <path>` flag (defaults to `$PIEKBS_KB`, then `~/piekbs-kb`).

## Commands

| Command | Description |
|---|---|
| `piekbs init [--force]` | Scaffold KB dirs and copy bundled schema/templates |
| `piekbs serve` | Start the long-running server: HTTP MCP + Web UI + file watcher. Default when no subcommand is given. |
| `piekbs index` | Build/update the FTS index from `wiki/` and `raw/` markdown |
| `piekbs search <query>` | FTS keyword search; prints ranked hits with paths and snippets |
| `piekbs synthesize [--topic X] [--full]` | Generate concept/comparison/decision pages from source-notes |
| `piekbs synthesize --gaps --topic X` | Knowledge-gap analysis for a topic |
| `piekbs import-lark <URL>` | Import a Lark/Feishu Wiki page into `raw/lark/` |
| `piekbs lint` | Health-check wiki pages: missing frontmatter, broken source links |
| `piekbs status` | Print index stats (document counts, index size) |
| `piekbs service <install\|uninstall\|start\|stop\|status\|logs>` | Manage the OS service (launchd / systemd) |

## System Service

To make PieKBS start on boot and run in the background:

```bash
piekbs service install --kb /path/to/your-kb
piekbs service status
piekbs service uninstall
```

Logs: `{PIEKBS_KB}/index/watcher.log`
