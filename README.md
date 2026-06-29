<div align="center">
  <img src="docs/readme/logo.png" width="128" alt="PieKBS"><br>
  <h1>PieKBS</h1>
  <p>A knowledge search engine for agents — distill raw docs into structured Markdown wiki, search and read via MCP</p>
  <p>
    <strong>English</strong> |
    <a href="docs/readme/README.zh-CN.md">简体中文</a> |
    <a href="docs/readme/README.zh-TW.md">繁體中文</a> |
    <a href="docs/readme/README.ru.md">Русский</a> |
    <a href="docs/readme/README.de.md">Deutsch</a> |
    <a href="docs/readme/README.fr.md">Français</a> |
    <a href="docs/readme/README.es.md">Español</a> |
    <a href="docs/readme/README.ko.md">한국어</a>
  </p>
  <p>
    <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="MIT License"></a>
    <a href="https://github.com/pieteams/piekbs/releases"><img src="https://img.shields.io/github/v/release/pieteams/piekbs" alt="Release"></a>
    <img src="https://img.shields.io/badge/go-1.25+-00ADD8.svg" alt="Go Version">
    <img src="https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-blue" alt="Platform">
  </p>
</div>

PieKBS is a local-first knowledge search engine for agents. It distills raw documents into a structured, reviewable Markdown wiki, then exposes two MCP tools — `kb_search` and `kb_page` — that let agents search and deep-read at their own pace.

![PieKBS Screenshot](docs/readme/image-001.png)

## Design Philosophy

PieKBS is built around one observation: **agents use external knowledge tools the same way humans use search engines** — they issue multiple queries from different angles, follow links, and synthesize their own conclusions. They do not want a pre-packaged answer; they want the raw materials to form their own.

This means PieKBS's job is not to answer questions. It is to make sure that when an agent searches for something, it finds the right documents — and can read them in full.

```text
piekbs-kb/
  raw/                  Source of truth — original materials in any format.
                        Drop files here; the watcher auto-distills them.

  wiki/                 Structured Markdown knowledge layer (LLM-maintained).
    source-notes/       One distilled note per raw document. FTS search target.
    concepts/           Cross-document synthesis: concepts and methodologies.
    comparisons/        Cross-document synthesis: side-by-side comparisons.
    decisions/          Cross-document synthesis: technical decisions.
    _draft/             Synthesized pages with < 2 sources (not indexed yet).

  schema/               KB-local authoring rules and page templates.
                        Edit these to customize the distilled page format.

  index/                Generated artifacts (SQLite FTS index, query logs).
                        Managed automatically — do not edit manually.
```

## How Agents Use PieKBS

Agents interact with PieKBS through three MCP tools:

**`kb_search(query, limit?)`** — Search with a keyword or phrase. Returns up to 5 source-notes and 3 concept/comparison/decision pages per call. Each result includes a `related` field listing linked documents for navigation. Use multiple searches with different keywords to cover a topic from multiple angles.

**`kb_page(ids, full?)`** — Fetch full content of one or more pages by ID (from `kb_search` results). Pass up to 5 IDs to scan several documents at once, or `full=true` with a single ID to get the complete untruncated text.

**`kb_add(filename, content, source_url?)`** — Add a text document to the knowledge base. Writes content to `raw/<filename>` and triggers incremental indexing. Distillation runs asynchronously in the background. Use the `converted/` prefix for agent-extracted PDF/Word/Excel/EPUB content.

The recommended agent workflow:

```text
kb_search("keyword A")          → discover relevant documents
kb_search("keyword B")          → cover a different angle
kb_page(["id1", "id2", "id3"]) → deep-read the most relevant ones
Agent synthesizes its own answer from what it found
```

Agents are expected to search iteratively, follow `related` links, cross-verify across sources, and form their own conclusions. PieKBS does not generate answers.

## PieKBS vs RAG

Traditional RAG retrieves context and hands it to the LLM to answer. PieKBS hands the agent raw materials and lets the agent do the reasoning.

```text
RAG:       user question → retrieve context → LLM answers
PieKBS:  agent searches → agent reads → agent synthesizes
```

| | RAG | PieKBS |
|---|---|---|
| Knowledge form | Implicit (vectors or chunks) | Explicit (Markdown, auditable) |
| Agent role | Passive receiver of context | Active searcher and reader |
| Answer source | System-generated | Agent-synthesized |
| Auditable | No | Yes — git diff, lint, conflict links |
| Multi-hop reasoning | LLM-dependent | Graph expansion via `related` links |
| Embedding | Required | Not required (pure FTS) |

PieKBS bundles are conformant with [OKF v0.1](https://github.com/GoogleCloudPlatform/knowledge-catalog/tree/main/okf).

## Knowledge Pipeline

Raw documents flow through a distillation pipeline before agents can search them:

**Step 1 — Distill (automatic)**

Drop any Markdown file into `raw/`. The `piekbs serve` watcher automatically runs distill + index. The LLM extracts structured source-notes into `wiki/source-notes/`, including:
- `key_claims` with inlined aliases and cross-language equivalents (ALIAS RULE) — ensures FTS matches all query variants
- Named entity annotations in `【entity|type】` format
- `related_to`, `supports`, `contradicts` links — powers the `related` field in search results
- `authority` (1–5) and `doc_type` metadata

**Step 2 — Synthesize (on-demand)**

```bash
piekbs synthesize --topic "RAG"
```

Generates concept / comparison / decision pages from source-notes when enough sources on a topic accumulate. Pages with fewer than 2 source references go to `wiki/<type>/_draft/` and are not indexed until more sources are added.

**Step 3 — Search**

Agents use `kb_search` + `kb_page` via MCP. Search is pure FTS (SQLite FTS5 with BM25 scoring). No vector model required.

## Installation

Download the latest release:

| Platform | File |
|---|---|
| macOS Apple Silicon (ARM64) | `PieKBS-<version>-macos-arm64.dmg` |
| Linux x86_64 | `piekbs-<version>-linux-amd64.tar.gz` |
| Linux ARM64 | `piekbs-<version>-linux-arm64.tar.gz` |
| Windows x86_64 | `piekbs-<version>-windows-amd64.zip` |

> **macOS Intel (x86_64):** No pre-built release. GitHub Actions dropped the Intel macOS runner in April 2025. Build from source on your Intel Mac: `CGO_ENABLED=1 go build -tags fts5 -o piekbs ./cmd/piekbs/`

**macOS:** Open the DMG and drag PieKBS to Applications. The app runs as a menubar icon.

**Linux:**
```bash
tar -xzf piekbs-<version>-linux-amd64.tar.gz -C /path/to/install/
sudo ln -sf /path/to/install/piekbs /usr/local/bin/piekbs
```

**Windows:** Extract the zip and run `piekbs.exe serve` (or `piekbs.exe stdio` for MCP). Add the directory to `PATH` for convenience. No CGO required — pure Go binary.

**HarmonyOS PC (community, experimental):** PieKBS is not officially released for HarmonyOS PC. However, since the core binary requires no CGO (pure Go + SQLite), it can be built natively on HarmonyOS using the community [Harmonybrew](https://harmonybrew.dev) package manager. See [ohos_go_cgo](https://github.com/ohos-go/ohos_go_cgo) for a guide on setting up Go + CGO on HarmonyOS PC.

```bash
# On HarmonyOS PC (after installing Go via Harmonybrew)
CGO_ENABLED=0 go build -tags fts5 -o piekbs ./cmd/piekbs/
piekbs serve
```

## Building from Source

Requires Go 1.25+. No CGO required.

```bash
# macOS / Linux
go build -tags fts5 -o piekbs ./cmd/piekbs/

# Windows
go build -tags fts5 -o piekbs.exe ./cmd/piekbs/
```

Or use the multi-platform build script:

```bash
./scripts/build.sh [version] [target...]
```

| Target | Output | Platform |
|---|---|---|
| `darwin-arm64` | `dist/PieKBS-<version>-macos-arm64.dmg` | macOS Apple Silicon |
| `linux-amd64` | `dist/piekbs-<version>-linux-amd64.tar.gz` | Linux x86_64 |
| `linux-arm64` | `dist/piekbs-<version>-linux-arm64.tar.gz` | Linux ARM64 |
| `windows-amd64` | `dist/piekbs-<version>-windows-amd64.zip` | Windows x86_64 |

## Repository Structure

```text
piekbs/
  cmd/piekbs/        # main entry point
  internal/
    kb/                # FTS indexing, search, graph expansion, page fetch
    mcp/               # MCP server (stdio + HTTP)
    watcher/           # file watcher for auto-distill + reindex
    distill/           # LLM distillation pipeline
    synthesize/        # concept/comparison/decision page generation
    convert/           # raw file conversion
    service/           # OS service manager (launchd / systemd)
    webui/             # web UI
    tray/              # macOS system tray (darwin only)
    config/            # KB config (config.yaml)
  scripts/
    build.sh           # multi-platform build script
```

## Schema & Templates

`piekbs init` populates the KB's `schema/` directory with bundled authoring rules and page templates:

- `schema/templates/`: Markdown templates for source-note / concept / comparison / decision pages.
- `schema/references/`: authoring rules — page types, citation rules, conflict rules, directory structure.

The distill/synthesize prompts read these templates, so editing them customizes the generated wiki format per-KB.

## Quick Start

```bash
export PIEKBS_KB=/path/to/your-kb

piekbs init           # scaffold KB dirs and copy schema/templates
piekbs serve          # start server: MCP + Web UI + file watcher
piekbs index          # build/update FTS index
piekbs status         # index stats
piekbs lint           # health-check wiki pages
```

## Command Reference

All commands accept a global `--kb <path>` flag (defaults to `$PIEKBS_KB`, then `~/piekbs-kb`).

| Command | Description |
|---|---|
| `piekbs init [--force]` | Scaffold KB dirs and copy bundled schema/templates. |
| `piekbs serve` | Start the long-running server: HTTP MCP (`/mcp`) + Web UI + file watcher. Default when no subcommand is given. |
| `piekbs index` | Build/update the FTS index from `wiki/` and `raw/` markdown. |
| `piekbs search <query>` | FTS keyword search; prints ranked hits with paths and snippets. |
| `piekbs synthesize [--topic X] [--full]` | Generate concept/comparison/decision pages from source-notes. |
| `piekbs synthesize --gaps --topic X` | Knowledge-gap analysis for a topic. |
| `piekbs import-lark <URL>` | Import a Lark/Feishu Wiki page and its embedded tables into `raw/lark/`. Requires a logged-in `lark-cli`. |
| `piekbs lint` | Health-check wiki pages: missing frontmatter fields, broken source links. |
| `piekbs status` | Print index stats (document counts, index size). |
| `piekbs service <install\|uninstall\|start\|stop\|status\|logs>` | Manage the OS service (launchd / systemd). |

**LLM config** (`config.yaml` under KB root, `distill` section) is required for `distill` and `synthesize`.

## MCP Server

PieKBS exposes KB tools via the MCP protocol.

**Available tools:** `kb_search`, `kb_page`, `kb_add`

Admin operations (`status`, `reindex`, `lint`) are available via the Web UI or CLI (`piekbs status`, `piekbs index`, `piekbs lint`).

---

### Scenario 1: Local Multi-Agent Sharing

HTTP mode is recommended: one PieKBS process shared by all agents — Claude Code, Cursor, VS Code (Copilot), Windsurf, Trae, Codex, Hermes, OpenClaw, and others.

**Step 1: Start PieKBS**

```bash
export PIEKBS_KB=/path/to/piekbs-kb
piekbs serve
```

> On macOS, double-click PieKBS.app to launch as a menubar icon.

**Step 2: Configure HTTP MCP in each agent**

Add to `~/.claude.json` under `mcpServers`:

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

---

### Scenario 2: Hosted Agent Environments

In hosted environments (Hermes, OpenClaw, etc.), install PieKBS on the persistent volume and invoke via **stdio** — PieKBS starts as a subprocess of the agent host, with the watcher running in the background automatically.

Example (NAS-mounted OpenClaw/Hermes, mount point `/root/.openclaw`):

**1. Install to persistent volume (one-time):**

```bash
tar -xzf piekbs-linux-amd64.tar.gz -C /root/.openclaw/piekbs/
chmod +x /root/.openclaw/piekbs/piekbs
```

**2. Install markitdown (recommended):**

markitdown enables conversion of PDF, Word, Excel, PPT, and HTML files to Markdown before distillation. Without it, only `.md` and `.txt` files are distilled; binary files are indexed by filename only.

```bash
pip install markitdown
# verify
markitdown --version
```

> Verified working on OpenClaw/Hermes (path: `/root/.openclaw/workspace/bin/markitdown`). Add `workspace/bin` to PATH or set the full path in your environment.

If markitdown is unavailable, agents can extract text themselves (using LLM vision or other tools) and write the result directly to `$PIEKBS_KB/raw/converted/<slug>.md` — the watcher picks it up automatically.

**3. MCP configuration:**

Hermes (`mcp_servers` in agent config):

```yaml
mcp_servers:
  piekbs:
    command: /root/.openclaw/piekbs/piekbs
    args: [stdio]
    env:
      PIEKBS_KB: /root/.openclaw/piekbs-kb
      PATH: /root/.openclaw/workspace/bin:/usr/local/bin:/usr/bin:/bin
```

The KB directory is created automatically on first launch. No manual `init` needed.

**4. Adding content to the knowledge base:**

Agents with `write_file` access can write directly into the KB — the watcher detects changes and triggers indexing and distillation automatically.

| Content type | Write to |
|---|---|
| Articles, notes, references (Markdown/text) | `$PIEKBS_KB/raw/<your-category>/<slug>.md` |
| Agent-converted PDF / Word / Excel / EPUB content | `$PIEKBS_KB/raw/converted/<slug>.md` |

Files in `raw/converted/` are treated as already-converted and go straight to distillation, skipping the markitdown step. All other paths under `raw/` are processed through the full pipeline (convert → index → distill).

Organize subdirectories however makes sense for your content — PieKBS does not enforce a fixed structure under `raw/`.

## System Service (optional)

`piekbs serve` includes a built-in watcher that automatically monitors the KB directory, triggers distill, and rebuilds the index. No additional setup required.

To make PieKBS **start on boot and run in the background**, install it as a system service (macOS launchd / Linux systemd):

```bash
piekbs service install --kb /path/to/your-kb
piekbs service status
piekbs service uninstall
```

Logs: `{PIEKBS_KB}/index/watcher.log`
