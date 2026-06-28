# LLM Wiki Technologies

Core technologies behind the LLM Wiki (Knowledge Compilation) paradigm — knowledge is pre-compiled at write time, not retrieved at query time.

```
Raw docs → LLM distills → wiki pages → [query time] → FTS / graph → read compiled page
```

> Coined by Andrej Karpathy: use LLM as a compiler that transforms raw materials into a persistent, structured, auditable knowledge layer.

## 1. Knowledge Compilation

| Technology | Notes |
|---|---|
| LLM as compiler | LLM reads raw docs and writes structured wiki pages — pre-compiled, not retrieved at query time. |
| Distillation pipeline | Raw doc → LLM extracts key claims, entities, aliases, related links → source-note page (WikiLoop pattern). |
| Tiered compilation | Compile frequently-used docs first (Tier 0–3). Sage Wiki: auto-promote on 3 hits, auto-demote after 90 days inactive. |
| Incremental update | Only re-process changed documents — avoids full rebuild cost. |
| [STORM](https://github.com/stanford-oval/storm) (Stanford OVAL) | Multi-perspective Wiki generation agent: simulates different viewpoints → retrieves → generates hierarchical outline → writes cited article. |
| Co-STORM | Collaborative variant of STORM — builds dynamic knowledge map during research to guide compilation direction. |
| Agentic compilation loop | Agent loop: retrieve → draft → evaluate → re-retrieve → regenerate until knowledge is stable. |

## 2. Knowledge Representation

| Technology | Notes |
|---|---|
| Structured Markdown | All knowledge stored as plain Markdown — human-readable, git-diffable, auditable. |
| Source-note pages | One distilled page per raw document. Contains `key_claims`, entity annotations `【entity\|type】`, `related_to` / `supports` / `contradicts` links. |
| Concept pages | Cross-document synthesis of a concept or methodology. |
| Comparison pages | Side-by-side comparison of approaches, tools, or paradigms. |
| Decision pages | Technical decision records (ADR format): context + options + decision + consequences. |
| Ontology graph | Typed entity-relation graph built during compilation. Sage Wiki: 8 built-in relation types (`implements`, `contradicts`, `trades_off`, …). |
| Schema / Templates | Authoring rules and page templates that guide LLM compilation style, customizable per KB. |

## 3. Indexing & Search

| Technology | Notes |
|---|---|
| SQLite FTS5 + BM25 | Core search engine — no vector model needed. Sub-millisecond full-text search. Used by WikiLoop, Sage Wiki, TreeSearch. |
| Alias expansion | Key terms indexed with aliases and cross-language equivalents to maximize FTS recall. |
| Graph traversal | `related_to` links enable multi-hop navigation (similar to wiki page links). BFS expansion on search results. |
| Hybrid (FTS + vector + graph) | Sage Wiki: FTS5 (411µs) + vector (81ms) + ontology graph (1µs) merged via RRF. |
| Chapter-tree indexing | Preserves document H1/H2/H3 hierarchy — structure-preserving alternative to flat chunking. |

## 4. Knowledge Quality & Maintenance

| Technology | Notes |
|---|---|
| Git version control | All wiki pages in git — full history, diffs, blame. Knowledge changes are auditable. |
| Lint / health checks | Validate frontmatter, broken source links, missing citations. `wikiloop lint`. |
| Conflict detection | `contradicts` links surface disagreements between sources for human review. |
| Draft staging | Pages with < 2 source references quarantined in `_draft/` — not indexed until verified. |
| Knowledge gap analysis | `wikiloop synthesize --gaps` identifies topics with insufficient coverage. |
| Entity deduplication | Identify different expressions of the same concept and merge into a single node. |

## 5. Agent Interface (MCP)

| Technology | Notes |
|---|---|
| [MCP protocol](https://modelcontextprotocol.io) | Model Context Protocol — Anthropic open standard for exposing tools/resources to AI agents. Supports stdio + HTTP transports. |
| `kb_search` | FTS keyword search, returns ranked results with `related` links for graph navigation. |
| `kb_page` | Fetch full page content by ID. Supports batch (up to 5 IDs) or `full=true` for untruncated text. |
| MCP Resources | Read-only resources (URI form): wiki pages, graph schema, raw documents. |
| Iterative search pattern | Agent issues multiple queries from different angles, follows `related` links, synthesizes own answer. |
| Sage Wiki MCP tools | 17 tools: 6 read, 9 write, 2 composite — agents can directly write and compile knowledge. |

## 6. File Conversion (Input Layer)

| Technology | Notes |
|---|---|
| markitdown (Microsoft) | Converts PDF, Word, Excel, PPT, HTML to Markdown before distillation. |
| `raw/converted/` pattern | Agent-extracted content written directly here — skips conversion, goes straight to distillation. |
| File watcher | Auto-detects new/changed files in `raw/`, triggers convert → distill → index pipeline. |

---

See also: [RAG Technologies](/ecosystem/rag-technologies) · [Paradigm Comparison](/ecosystem/paradigm-comparison)
