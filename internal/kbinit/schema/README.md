---
name: piekbs
description: Use this skill when maintaining, querying, initializing, or reviewing a local-first PieKBS knowledge base. This skill should trigger for tasks involving raw/, wiki/, schema/, source notes, cited wiki pages, KB ingestion, KB maintenance loops, or a piekbs-kb directory, even if the user does not explicitly say "PieKBS".
---

# PieKBS

Use this skill to maintain a local-first LLM Wiki knowledge base. Keep raw sources authoritative, wiki pages structured and cited, and every knowledge update tied to a repeatable loop.

## Core Model

```text
piekbs-kb/
  raw/      source of truth; do not modify raw sources
  wiki/     LLM-maintained Markdown knowledge layer
  schema/   KB-local rules and workflows
  index/    generated artifacts; optional after Phase 1
```

## Start Here

1. For KB structure or initialization, read `references/kb-directory-structure.md`.
2. For source links or verification, read `references/citation-rules.md`.
3. For duplicate, stale, uncertain, or conflicting information, read `references/conflict-rules.md`.
4. For page selection, read `references/page-types.md`.
5. For workflow execution, read `references/maintenance-loops.md`.

## Operating Rules

- Do not modify files under `raw/`.
- Do not put full raw source copies into `wiki/`.
- Create or update `wiki/source-notes/` for one-source notes.
- Create or update `wiki/concepts/`, `wiki/comparisons/`, and `wiki/decisions/` only when knowledge is reusable across questions or sources.
- Verify against raw sources for numbers, dates, versions, prices, licenses, legal claims, security claims, conflicts, or user-requested citations.
- Prefer updating an existing related wiki page over creating a near-duplicate page.

## Templates

Use files under `templates/` when creating new wiki pages:

- `templates/source-note.md`
- `templates/concept.md`
- `templates/comparison.md`
- `templates/decision.md`
