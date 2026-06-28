# 什么是 WikiLoop

WikiLoop 是一个面向 Agent 的本地优先知识搜索引擎。它将原始文档提炼为结构化、可审计的 Markdown 知识库，然后通过三个 MCP 工具 — `kb_search`、`kb_page` 和 `kb_add` — 让 Agent 按自己的节奏搜索、阅读和写入知识。

## 设计理念

WikiLoop 基于一个核心观察：**Agent 使用外部知识工具的方式和人使用搜索引擎完全一样** — 从不同角度发出多次查询、跟随链接、自行综合结论。它们不需要一个预先打包好的答案，而是需要形成自己结论所需的原始材料。

因此 WikiLoop 的工作不是回答问题，而是确保当 Agent 搜索某个内容时，能找到正确的文档，并能完整阅读。

## WikiLoop vs RAG

| | RAG | WikiLoop |
|---|---|---|
| 知识形式 | 隐式（向量或 chunk） | 显式（Markdown，可审计） |
| Agent 角色 | 被动接收上下文 | 主动搜索和阅读 |
| 答案来源 | 系统生成 | Agent 自行综合 |
| 可审计 | 否 | 是 — git diff、lint、冲突链接 |
| 多跳推理 | 依赖 LLM | 通过 `related` 链接图扩展 |
| Embedding | 必需 | 不需要（纯 FTS） |

## 知识库结构

```text
wikiloop-kb/
  raw/              原始材料 — 放入文件，watcher 自动提炼。
  wiki/
    source-notes/   每个原始文档对应一个提炼笔记，FTS 搜索目标。
    concepts/       跨文档综合：概念与方法论。
    comparisons/    跨文档综合：方案横向对比。
    decisions/      跨文档综合：技术决策记录。
    _draft/         来源不足 2 个的页面（暂不索引）。
  schema/           知识库本地的写作规则和页面模板。
  index/            生成产物（SQLite FTS 索引），勿手动编辑。
```

## 生态

- [RAG 知识库系统](/zh-CN/ecosystem/rag-systems) — RAG 领域的框架、平台和向量数据库
- [LLM Wiki 系统](/zh-CN/ecosystem/llm-wiki-systems) — AI 驱动的 Wiki 平台、PKM 工具和 Agent 记忆框架

## 了解更多

- [安装](/zh-CN/getting-started/installation)
- [快速入门](/zh-CN/getting-started/quick-start)
- [Agent 如何使用 WikiLoop](/zh-CN/guide/how-agents-use)
