# WikiLoop KB Directory Structure

Default knowledge base instance name:

```text
wikiloop-kb/
```

The name is only a convention. Users may choose any local directory as the KB root.

## Standard Structure

```text
wikiloop-kb/
  raw/
    articles/
    papers/
    webpages/
      wechat/
    images/
    data/
    code/
    converted/

  wiki/
    source-notes/   ← core, auto-created by init
    concepts/       ← core, auto-created by init
    comparisons/    ← core, auto-created by init
    decisions/      ← core, auto-created by init

  schema/
    AGENTS.md
    wiki-structure.md
    page-templates.md
    ingestion-workflow.md
    citation-rules.md
    conflict-rules.md

  index/
    kb.sqlite
    gaps/           ← generated gap analysis reports (not wiki pages)
```

## Directory Roles

- `raw/`: authoritative original sources and converted near-full-text derivatives.
- `wiki/`: structured, cited Markdown knowledge maintained by agents and humans.
- `schema/`: KB-local rules, templates, citation rules, and maintenance workflow docs.
- `index/`: generated search artifacts such as SQLite, FTS, vector, or graph indexes.

## Raw Source Rules

- Do not modify original raw files.
- Put converted near-full-text derivatives under `raw/converted/` or beside webpage snapshots.
- Put webpage snapshots under `raw/webpages/<source>/`.
- For WeChat articles, prefer saving `.html`, extracted `.md`, and `.meta.json` together.

## Wiki Rules

- Put one-source notes under `wiki/source-notes/`.
- Put reusable concepts under `wiki/concepts/`.
- Put tradeoff pages under `wiki/comparisons/`.
- Put technical judgments under `wiki/decisions/`.
- Do not store raw full-text copies in `wiki/`.

## Note on Karpathy's Original Design

In Karpathy's original llm-wiki, **Overview / Entities / How-to / Timelines** are
*sections inside each article*, not subdirectories. WikiLoop uses
`source-notes/concepts/comparisons/decisions/` as the four core subdirectories,
which is a community evolution better suited to knowledge-base use cases.

Gap analysis reports (`wikiloop synthesize --gaps`) are generated artifacts and
belong in `index/gaps/`, not in `wiki/`.

## Phase 1 Minimum

```text
wikiloop-kb/
  raw/
  wiki/
  schema/
```

`index/` can be absent until CLI indexing exists.
