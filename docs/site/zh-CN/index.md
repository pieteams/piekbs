---
layout: home

hero:
  name: "WikiLoop"
  text: "面向 Agent 的知识搜索引擎"
  tagline: 将原始文档提炼为结构化 Markdown 知识库，通过 MCP 供 Agent 搜索和阅读。
  actions:
    - theme: brand
      text: 快速开始
      link: /zh-CN/getting-started/what-is-wikiloop
    - theme: alt
      text: 入门指南
      link: /zh-CN/getting-started/quick-start
    - theme: alt
      text: GitHub
      link: https://github.com/jasen215/wikiloop

features:
  - title: Agent 原生搜索
    details: 三个 MCP 工具 — kb_search、kb_page 和 kb_add — 让 Agent 像人使用知识库一样迭代搜索、完整阅读、主动写入知识。
  - title: 可审计的知识
    details: 所有知识都是显式 Markdown — 可以 git diff、lint 和 review 每一次变更，没有黑盒向量。
  - title: 无需 Embedding
    details: 纯 SQLite FTS5 + BM25 评分，快速、离线、零基础设施。
  - title: 自动提炼管道
    details: 将任意文件放入 raw/ 目录，watcher 自动提炼为结构化 source-note 并重建索引。
---
