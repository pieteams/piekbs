# How Agents Use PieKBS

Agents interact with PieKBS through two MCP tools:

## kb_search

```
kb_search(query, limit?)
```

Search with a keyword or phrase. Returns up to 5 source-notes and 3 concept/comparison/decision pages per call. Each result includes a `related` field listing linked documents for navigation.

Use multiple searches with **different keywords** to cover a topic from multiple angles. Do NOT repeat the same query — switch keywords or topic angle.

## kb_page

```
kb_page(ids, full?)
```

Fetch full content of one or more pages by ID (from `kb_search` results). Pass up to 5 IDs to scan several documents at once, or `full=true` with a single ID to get the complete untruncated text.

## Recommended Workflow

```text
kb_search("keyword A")          → discover relevant documents
kb_search("keyword B")          → cover a different angle
kb_page(["id1", "id2", "id3"]) → deep-read the most relevant ones
Agent synthesizes its own answer from what it found
```

Agents are expected to:
- Search iteratively with varied keywords
- Follow `related` links for multi-hop reasoning
- Cross-verify across sources
- Form their own conclusions

PieKBS provides **materials** — it does not generate answers.

## Query Expansion

Before searching, expand the query into aliases, abbreviations, and cross-language equivalents. FTS uses exact term matching.

Examples:
- `"召回率"` → also search `"recall"`, `"Context Recall"`, `"CR"`
- `"distillation"` → also search `"extract"`, `"summarize"`, `"pipeline"`
