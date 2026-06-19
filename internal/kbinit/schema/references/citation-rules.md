# Citation Rules

WikiLoop treats `raw/` as the source of truth and `wiki/` as a derived knowledge layer.

## Required Source Links

Every wiki page must include `sources` in frontmatter:

```yaml
---
title: "Agent Memory Systems"
kind: comparison
sources:
  - raw/articles/mem0.md
  - raw/articles/graphiti.md
updated_at: 2026-06-13
---
```

Use paths relative to the KB root.

## Raw Verification Required

Verify against raw sources before answering or updating wiki pages when the content involves:

- Numbers, dates, versions, prices, or licenses.
- Legal, compliance, privacy, or security conclusions.
- Conflicting or uncertain wiki claims.
- User-requested citations or provenance.
- A wiki page with missing or weak sources.
- A planned modification to an existing wiki page.

## Source Priority

1. Original raw source, such as PDF, HTML snapshot, image, exported file, or original Markdown.
2. Converted near-full-text derivative under `raw/converted/` or `raw/webpages/<source>/`.
3. `wiki/source-notes/` page that cites the raw source.
4. Synthesized wiki page such as concept, comparison, or decision.

Do not treat converted text as more authoritative than the original source.

## Answer Format

When answering from the KB, mention the wiki page and raw source paths that support the answer. If the raw source was not checked because the question did not require verification, say the answer is based on wiki pages only.
