# Maintenance Loops

PieKBS works by repeating small knowledge maintenance loops. Do not treat the KB as a place to dump summaries.

## Ingest Loop

Use this loop when new material appears under `raw/`.

1. Identify the raw source path.
2. Check whether converted text exists under `raw/converted/` or `raw/webpages/<source>/`.
3. Search `wiki/source-notes/` for an existing note about the same source.
4. Create or update one source note.
5. Update related concept, comparison, or decision pages only when the knowledge is reusable.
6. Check that every updated wiki page has source links.
7. Record uncertainty or conflict instead of smoothing it over.

## Query Loop

Use this loop when answering a user question from a KB.

1. Search or inspect relevant wiki pages first.
2. Prefer source notes and synthesized pages over raw full text for initial understanding.
3. Verify raw sources when citation rules require it.
4. Answer with clear source paths.
5. If no suitable page exists, say the KB has a gap instead of inventing one.

## Maintenance Loop

Use this loop when pages are stale, duplicate, weakly sourced, or contradictory.

1. Identify the affected wiki pages.
2. Read their cited raw sources.
3. Decide whether to update, merge, split, or mark conflict.
4. Preserve sourced historical information when useful.
5. Re-check frontmatter source links.
6. Keep the page focused on one page type.

## Phase 1 Lint Checklist

Before finishing a wiki update, manually check:

- The page has YAML frontmatter.
- `title`, `kind`, `sources`, and `updated_at` are present.
- Every `sources` path is relative to the KB root.
- No raw full-text copy was pasted into `wiki/`.
- Any uncertainty or conflict is explicit.

## Phase 2 CLI Commands

When the CLI is available (`piekbs`), prefer these over manual directory scanning:

- After ingesting new files: `piekbs index`
- Before answering a question: `piekbs context "<question>"`
- To search by keyword: `piekbs search "<query>"`
- To validate the KB: `piekbs lint`
- To check index state: `piekbs status`

Set `PIEKBS_KB=/path/to/piekbs-kb` to avoid typing `--kb` on every command.
