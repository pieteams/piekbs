<div align="center">
  <img src="docs/readme/logo.png" width="128" alt="WikiLoop"><br>
  <h1>WikiLoop</h1>
  <p>Agent-native LLM Wiki (Karpathy-style) RAG: distill raw docs → structured Markdown wiki → hybrid search via MCP</p>
  <p><a href="docs/readme/README.zh-CN.md">中文文档</a></p>
  <p>
    <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="MIT License"></a>
    <a href="https://github.com/jasen215/wikiloop/releases"><img src="https://img.shields.io/github/v/release/jasen215/wikiloop" alt="Release"></a>
    <img src="https://img.shields.io/badge/go-1.25+-00ADD8.svg" alt="Go Version">
    <img src="https://img.shields.io/badge/platform-macOS%20%7C%20Linux-blue" alt="Platform">
  </p>
</div>

WikiLoop is a local-first LLM Wiki knowledge base for agents. It is designed to help agents and humans maintain sourced, reviewable, Markdown-based knowledge from local raw materials.

![WikiLoop Screenshot](docs/readme/image-001.png)

## Core Idea

```text
wikiloop-kb/
  raw/      source of truth; original materials and converted near-full-text derivatives
  wiki/     structured Markdown knowledge maintained by agents and humans
  schema/   KB-local rules, templates, citation rules, and workflows
  index/    generated search/index artifacts
```

WikiLoop is not a long-term memory system. User preferences, project habits, and cross-session memories should stay in external memory providers. WikiLoop manages external knowledge sources and the wiki layer derived from them.

## WikiLoop vs RAG

Traditional RAG pipes raw documents directly into a vector store — knowledge is implicit, hidden in embeddings, and not human-reviewable. WikiLoop adds an explicit distillation step:

```text
RAG:       raw docs → embed → vector store → LLM retrieval
WikiLoop:  raw docs → wiki (human/agent) → FTS + graph + embed (optional)
```

| | RAG | WikiLoop |
|---|---|---|
| Knowledge form | Implicit (vectors) | Explicit (Markdown) |
| Auditable | No | Yes — git diff, lint, conflict links |
| Cold-start cost | Low | Higher (wiki authoring required) |
| Knowledge decay | Hard to detect | Easy — lint + citations |
| Multi-hop reasoning | LLM-dependent | Graph expansion (explicit) |
| Embedding | Required | Optional enhancement |

The trade-off: WikiLoop requires a distillation step (human or agent), but yields knowledge that is controllable, traceable, and incrementally maintainable.

WikiLoop wiki bundles are conformant with [OKF v0.1](https://github.com/GoogleCloudPlatform/knowledge-catalog/tree/main/okf).

## Installation

Download the latest release for your platform:

| Platform | File |
|---|---|
| macOS Apple Silicon | `WikiLoop-<version>-darwin-arm64.dmg` |
| Linux x86_64 | `wikiloop-<version>-linux-amd64.tar.gz` |
| Linux ARM64 | `wikiloop-<version>-linux-arm64.tar.gz` |

> **Windows** is not currently available as a prebuilt binary. The build requires
> [libtokenizers](https://github.com/daulet/tokenizers) which has no Windows prebuilt and
> requires Rust + CGO toolchain compatibility that is still being resolved.
> Windows users can build from source with MinGW-w64 and Rust installed.

**macOS:** Open the DMG and drag WikiLoop to Applications. The app runs as a menubar icon.

**Linux:**
```bash
tar -xzf wikiloop-<version>-linux-amd64.tar.gz -C /path/to/install/
sudo ln -sf /path/to/install/wikiloop /usr/local/bin/wikiloop
```

### Download Embedding Model

Vector search requires the `bge-small-zh` ONNX model (53 MB), downloaded separately:

**[⬇ Download bge-small-zh.tar.gz](https://github.com/jasen215/wikiloop/releases/tag/models-v1)**

```bash
# Extract to your KB's models directory
tar -xzf bge-small-zh.tar.gz -C $WIKILOOP_KB/models/
```

After extraction: `$WIKILOOP_KB/models/bge-small-zh/model.onnx` should exist.
FTS search works without the model; vector search requires it.

## Building from Source

Requires Go 1.25+ with CGO enabled.

```bash
./scripts/build.sh [version] [target...]
```

| Target | Output | Platform |
|---|---|---|
| `darwin-arm64` | `dist/WikiLoop-<version>-darwin-arm64.dmg` | macOS Apple Silicon |
| `linux-amd64` | `dist/wikiloop-<version>-linux-amd64.tar.gz` | Linux x86_64 |
| `linux-arm64` | `dist/wikiloop-<version>-linux-arm64.tar.gz` | Linux ARM64 |

```bash
./scripts/build.sh 1.2.0               # all platforms
./scripts/build.sh 1.2.0 linux-amd64   # single target
```

**Dependencies:**

- Linux targets: `brew install FiloSottile/musl-cross/musl-cross`
- macOS DMG: `brew install create-dmg` (optional, skipped if absent)

Each tar.gz includes the binary only. The DMG is a drag-to-install macOS app bundle with system tray support. Embedding models are downloaded separately (see above).

## Repository Structure

```text
wikiloop/
  cmd/wikiloop/        # main entry point
  internal/
    kb/                # indexing, FTS, vector search
    mcp/               # MCP server (stdio + HTTP)
    embed/             # ONNX embedding (bge-small-zh)
    watcher/           # file watcher for auto-reindex
    distill/           # LLM distillation pipeline
    convert/           # raw file conversion
    service/           # OS service manager (launchd / systemd)
    webui/             # web UI
    tray/              # macOS system tray (darwin only)
    config/            # KB config (config.yaml)
  scripts/
    build.sh           # multi-platform build script
    Info.plist         # macOS app bundle metadata
    wikiloop.icns      # app icon
  docs/
```

## Schema & Templates

`wikiloop init` populates the KB's `schema/` directory with bundled authoring
rules and page templates (sourced from `internal/kbinit/schema/`):

- `schema/templates/`: Markdown templates for source-note / concept / comparison / decision pages.
- `schema/references/`: authoring rules — page types, citation rules, conflict rules, directory structure, maintenance loops.

The distill/synthesize prompts read these templates, so editing them customizes
the generated wiki format per-KB.

## Knowledge Base Instance

The default knowledge base instance name is:

```text
wikiloop-kb/
```

This is only a convention. Users can place a KB anywhere and name it differently.

Minimum structure:

```text
wikiloop-kb/
  raw/
  wiki/
  schema/
```

Recommended full structure is documented in `schema/references/kb-directory-structure.md` (created by `wikiloop init`).

## Quick Start

```bash
export WIKILOOP_KB=/path/to/your-kb

wikiloop index          # build/update index
wikiloop embed          # generate embeddings (optional)
wikiloop status         # index stats
wikiloop search "query"
wikiloop context "question"
wikiloop lint
```

Full options: `wikiloop --help`.

## Command Reference

All commands accept a global `--kb <path>` flag (defaults to `$WIKILOOP_KB`, then `~/wikiloop-kb`).

| Command | Description |
|---|---|
| `wikiloop init [--force]` | Scaffold KB dirs and copy bundled schema/templates. `--force` overwrites existing schema files. |
| `wikiloop serve` | Start the long-running server: HTTP MCP (`/mcp`) + Web UI + file watcher. The default when no subcommand is given. |
| `wikiloop index` | Build/update the FTS index from `wiki/` and `raw/` markdown. |
| `wikiloop embed [--full]` | Generate vector embeddings for documents. `--full` drops and rebuilds the vector store. Requires ONNX runtime. |
| `wikiloop search <query>` | FTS keyword search; prints ranked hits with paths and snippets. |
| `wikiloop context <question>` | Build a context bundle (relevant pages + sources) for a question. |
| `wikiloop distill` | (runs inside `serve`/watcher) Convert new `raw/` files into `wiki/source-notes/` via LLM. Not a standalone subcommand — triggered automatically. |
| `wikiloop synthesize [--topic X] [--full]` | Generate concept/comparison/decision pages from source-notes. Incremental by default; `--full` reprocesses all; `--topic` limits to tag/title matches. |
| `wikiloop synthesize --gaps --topic X` | Knowledge-gap analysis for a topic; writes report to `index/gaps/<slug>.md`. |
| `wikiloop import-lark <URL>` | Import a Lark/Feishu Wiki document and expand embedded Base tables into searchable local datasets. Requires authenticated `lark-cli`. |
| `wikiloop lint` | Health-check wiki pages: missing frontmatter fields, broken source links. |
| `wikiloop status` | Print index stats (document/embedding counts, index size). |
| `wikiloop service <install\|uninstall\|start\|stop\|status\|logs>` | Manage the OS service (launchd / systemd). |

**LLM config** (`config.yaml` under KB root, `distill` section) is required for `distill` and `synthesize`. See the MCP Server section for the format; `api_type` selects `openai` (default) or `anthropic`.

DeepSeek uses the OpenAI-compatible API:

```yaml
distill:
  base_url: "https://api.deepseek.com"
  model: "deepseek-chat"
  api_type: "openai"
```

Keep the token out of `config.yaml` when possible:

```bash
export WIKILOOP_DISTILL_TOKEN="your-api-key"
```

The settings page also has a **Use DeepSeek** preset. Base URLs with or without
the trailing `/v1` are accepted.

### Import a Lark/Feishu Wiki page

```bash
wikiloop import-lark "https://example.larkoffice.com/wiki/..."
```

Embedded Base tables are fetched page by page. The document is saved as
Markdown under `raw/lark/`. Original tables are preserved as `.snapshot.tsv`
files for audit. WikiLoop indexes one `records-deduplicated.txt` dataset,
deduplicating exact links and repeated title + nickname submissions while
retaining same-title submissions from different users.

### synthesize workflow: from raw sources to topic summaries

Typical flow: Agent collects or generates content → auto-distill → on-demand synthesis per topic.

Raw sources can be anything:
- Articles fetched by an Agent (web pages, WeChat, RSS)
- Documents dropped by the user (converted PDFs, local notes)
- Research reports or analysis files generated by an Agent
- Any markdown file

**Step 1 — content enters KB (automatic)**

Drop markdown into `raw/` (organized by source, e.g. `raw/articles/`, `raw/papers/`, `raw/reports/`);
the `wikiloop serve` watcher auto-runs distill + index + embed.

**Step 2 — synthesize by topic (on-demand, Agent-triggered)**

```bash
# Summarize all source-notes tagged/titled with "chip industry"
wikiloop synthesize --topic "芯片产业"

# Force full re-run (ignore incremental cache)
wikiloop synthesize --topic "芯片产业" --full

# No topic: process all new/changed source-notes across topics
wikiloop synthesize

# Knowledge-gap analysis: what's missing on this topic?
wikiloop synthesize --gaps --topic "芯片产业"
# → index/gaps/zhi-pian-chan-ye.md
```

**`--topic` matching**: case-insensitive substring match on source-note `title` or `tags`.

**Output types**:

| Type | Directory | Threshold |
|---|---|---|
| concept | `wiki/concepts/` | ≥3 source-notes share a concept/pattern |
| comparison | `wiki/comparisons/` | ≥2 source-notes worth contrasting |
| decision | `wiki/decisions/` | ≥2 source-notes support a technical judgment |

## Auto-index Service

Monitors the KB directory for changes and automatically triggers `index + embed`. Supports macOS (launchd) and Linux (systemd).

```bash
# Install and start system service
wikiloop service install --kb /path/to/your-kb

# Check status / uninstall
wikiloop service status
wikiloop service uninstall
```

Logs are written to `{WIKILOOP_KB}/index/watcher.log`.

## MCP Server

WikiLoop exposes KB tools via the MCP protocol. Two deployment scenarios are supported.

Available tools: `kb_search`, `kb_context`, `kb_status`, `kb_reindex`, `kb_lint`

---

### Scenario 1: Local Multi-Agent Sharing

For multiple local agents (Claude Code, Cursor, etc.) sharing a single KB instance.

HTTP mode is recommended: one WikiLoop process shared by all agents, avoiding index conflicts that can occur when each agent spawns its own stdio process.

**Step 1: Set KB path and start WikiLoop**

The default KB directory is `~/wikiloop-kb`. Override with an environment variable:

```bash
export WIKILOOP_KB=/path/to/wikiloop-kb
```

After installation, create a symlink for CLI access:

```bash
# macOS
sudo ln -sf /Applications/WikiLoop.app/Contents/MacOS/wikiloop /usr/local/bin/wikiloop

# Linux
sudo ln -sf /path/to/wikiloop /usr/local/bin/wikiloop
```

Start the service:

```bash
wikiloop serve
```

> On macOS, you can also double-click WikiLoop.app to launch it as a menubar icon.
> The app automatically reads `WIKILOOP_*` variables from `~/.zshenv`, `~/.bashrc`,
> and other shell rc files, so behavior is identical to CLI invocation.

`serve` provides both HTTP MCP (`/mcp`) and Web UI. Port is configured in `config.yaml` under `server` (default: 8766).

**Step 2: Configure HTTP MCP in each agent**

Add to `~/.claude.json` under `mcpServers`:

```json
{
  "mcpServers": {
    "wikiloop": {
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

---

### Scenario 2: Hosted Agent Environments (Hermes / OpenClaw, etc.)

In hosted environments, the container cannot access the user's local processes. Install WikiLoop on the **persistent volume** in the agent's environment and invoke it locally via stdio.

Example using Alibaba Cloud NAS-mounted OpenClaw/Hermes (mount point `/root/.openclaw`):

**1. Install to persistent volume (one-time):**

```bash
tar -xzf wikiloop-linux-amd64.tar.gz -C /root/.openclaw/wikiloop/
chmod +x /root/.openclaw/wikiloop/wikiloop

# Copy models into the KB directory
mkdir -p /root/.openclaw/wikiloop-kb/models
cp -r /root/.openclaw/wikiloop/models/* /root/.openclaw/wikiloop-kb/models/
```

After a container rebuild, the NAS volume is remounted and both binary and KB data are preserved — no reinstallation needed.

**2. MCP configuration:**

Claude Code (`~/.claude.json`):

```json
{
  "mcpServers": {
    "wikiloop": {
      "command": "/root/.openclaw/wikiloop/wikiloop",
      "args": ["serve"],
      "env": {
        "WIKILOOP_KB": "/root/.openclaw/wikiloop-kb"
      }
    }
  }
}
```

Hermes (`mcp_servers` in agent config):

```yaml
mcp_servers:
  wikiloop:
    command: /root/.openclaw/wikiloop/wikiloop
    args: [serve]
    env:
      WIKILOOP_KB: /root/.openclaw/wikiloop-kb
```

OpenClaw uses the same format as Hermes — refer to the platform documentation.
