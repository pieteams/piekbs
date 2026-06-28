---
layout: home

hero:
  name: "WikiLoop"
  text: "Knowledge Search Engine for Agents"
  tagline: Distill raw docs into structured Markdown wiki. Search and read via MCP.
  actions:
    - theme: brand
      text: Get Started
      link: /getting-started/what-is-wikiloop
    - theme: alt
      text: Quick Start
      link: /getting-started/quick-start
    - theme: alt
      text: GitHub
      link: https://github.com/jasen215/wikiloop

features:
  - title: Agent-Native Search
    details: Three MCP tools — kb_search, kb_page, and kb_add — let agents search, read, and write knowledge, just like a human uses and builds a knowledge base.
  - title: Auditable Knowledge
    details: All knowledge is explicit Markdown — git diff, lint, and review every change. No black-box vectors.
  - title: No Embedding Required
    details: Pure SQLite FTS5 with BM25 scoring. Fast, offline, zero infrastructure.
  - title: Auto-Distill Pipeline
    details: Drop any file into raw/. The watcher automatically distills it into structured source-notes and rebuilds the index.
---
