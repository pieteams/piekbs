# Knowledge Pipeline

Raw documents flow through a distillation pipeline before agents can search them.

## Step 1 — Distill (automatic)

Drop any Markdown file into `raw/`. The `piekbs serve` watcher automatically runs distill + index.

The LLM extracts structured source-notes into `wiki/source-notes/`, including:
- `key_claims` with inlined aliases and cross-language equivalents (ALIAS RULE) — ensures FTS matches all query variants
- Named entity annotations in `【entity|type】` format
- `related_to`, `supports`, `contradicts` links — powers the `related` field in search results
- `authority` (1–5) and `doc_type` metadata

## Step 2 — Synthesize (on-demand)

```bash
piekbs synthesize --topic "RAG"
```

Generates concept / comparison / decision pages from source-notes when enough sources on a topic accumulate.

Pages with fewer than 2 source references go to `wiki/<type>/_draft/` and are not indexed until more sources are added.

```bash
# Knowledge-gap analysis
piekbs synthesize --gaps --topic "RAG"
```

## Step 3 — Search

Agents use `kb_search` + `kb_page` via MCP. Search is pure FTS (SQLite FTS5 with BM25 scoring). No vector model required.

## File Support

| Format | Processing |
|---|---|
| `.md`, `.txt` | Direct distillation |
| PDF, Word, Excel, PPT, HTML | Converted via `markitdown`, then distilled |
| Agent-converted content | Write to `raw/converted/` to skip conversion step |

Install `markitdown` to enable binary file support:

```bash
pip install markitdown
```
