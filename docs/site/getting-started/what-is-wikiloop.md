# What is WikiLoop

WikiLoop is a local-first knowledge search engine for agents. It distills raw documents into a structured, reviewable Markdown wiki, then exposes three MCP tools — `kb_search`, `kb_page`, and `kb_add` — that let agents search, read, and write knowledge at their own pace.

## Design Philosophy

WikiLoop is built around one observation: **agents use external knowledge tools the same way humans use search engines** — they issue multiple queries from different angles, follow links, and synthesize their own conclusions. They do not want a pre-packaged answer; they want the raw materials to form their own.

This means WikiLoop's job is not to answer questions. It is to make sure that when an agent searches for something, it finds the right documents — and can read them in full.

## WikiLoop vs RAG

| | RAG | WikiLoop |
|---|---|---|
| Knowledge form | Implicit (vectors or chunks) | Explicit (Markdown, auditable) |
| Agent role | Passive receiver of context | Active searcher and reader |
| Answer source | System-generated | Agent-synthesized |
| Auditable | No | Yes — git diff, lint, conflict links |
| Multi-hop reasoning | LLM-dependent | Graph expansion via `related` links |
| Embedding | Required | Not required (pure FTS) |

## KB Structure

```text
wikiloop-kb/
  raw/              Source of truth — drop files here, watcher auto-distills.
  wiki/
    source-notes/   One distilled note per raw document. FTS search target.
    concepts/       Cross-document synthesis: concepts and methodologies.
    comparisons/    Cross-document synthesis: side-by-side comparisons.
    decisions/      Cross-document synthesis: technical decisions.
    _draft/         Pages with < 2 sources (not indexed yet).
  schema/           KB-local authoring rules and page templates.
  index/            Generated artifacts (SQLite FTS index). Do not edit.
```

## Ecosystem

- [RAG Knowledge Base Systems](/ecosystem/rag-systems) — Frameworks, platforms, and vector databases in the RAG space
- [LLM Wiki Systems](/ecosystem/llm-wiki-systems) — AI-powered wiki platforms, PKM tools, and agent memory frameworks

## Learn More

- [Installation](/getting-started/installation)
- [Quick Start](/getting-started/quick-start)
- [How Agents Use WikiLoop](/guide/how-agents-use)
