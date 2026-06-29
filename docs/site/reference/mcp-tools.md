# MCP Tools Reference

PieKBS exposes three MCP tools for agents.

## kb_search

Search the knowledge base with a keyword or phrase.

**Parameters:**

| Parameter | Type | Required | Description |
|---|---|---|---|
| `query` | string | yes | Search keyword or phrase |
| `kind` | string | no | Filter page kind: `source-note`, `concept`, `comparison`, `decision` |
| `layer` | string | no | Filter layer: `wiki`, `raw`, or `schema` |
| `limit` | number | no | Maximum results (default 10) |

**Returns:** Ranked list of matching pages, each with:
- `id` — page identifier for use with `kb_page`
- `title`, `snippet` — preview of the matched content
- `kind`, `layer` — page classification
- `related` — linked documents for graph navigation

**Example:**

```json
{
  "query": "RAG retrieval augmented generation",
  "limit": 5
}
```

## kb_add

Add a text document to the knowledge base.

**Parameters:**

| Parameter | Type | Required | Description |
|---|---|---|---|
| `filename` | string | yes | Path relative to `raw/`. Use any subdirectory structure (e.g. `references/article.md`, `converted/report.md`). Use `converted/` prefix for agent-extracted PDF/Word/Excel/EPUB content. |
| `content` | string | yes | File content (Markdown or plain text) |
| `source_url` | string | no | Original source URL, written as a comment at the top of the file |
| `overwrite` | boolean | no | Overwrite if file already exists (default false) |

Writes content to `raw/<filename>` and triggers incremental indexing. Distillation runs asynchronously in the background.

**Example:**

```json
{
  "filename": "references/my-article.md",
  "content": "# My Article\n\nContent here...",
  "source_url": "https://example.com/article"
}
```

## kb_page

Fetch full content of one or more pages by ID.

**Parameters:**

| Parameter | Type | Required | Description |
|---|---|---|---|
| `ids` | array | yes | Document IDs from `kb_search` results (1–5) |
| `full` | boolean | no | Return complete untruncated text (only with single ID) |

**Returns:** Full Markdown content of each requested page.

**Example:**

```json
{
  "ids": ["source-notes/my-doc", "concepts/rag-overview"],
  "full": false
}
```
