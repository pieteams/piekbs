---
type: source-note
title: ""
description: ""   # REQUIRED: 1-2 sentences. Must include Chinese keywords matching the source language PLUS ≥2 specific technical terms/numbers. Example: "介绍UnWeaver框架如何在不构建知识图谱的情况下实现细粒度RAG检索，索引Token消耗仅为GraphRAG的1/10。"
tags: []          # REQUIRED: 3-6 domain classification tags (e.g. RAG, 主数据, 智能制造, 数据治理). NOT random keywords.
doc_type: ""      # REQUIRED: one of: 技术文章 | 白皮书 | 技术规范 | 项目文档 | 会议纪要 | 分析报告 | 教程 | 开源项目 | 产品文档
authority: 3      # REQUIRED: 1-5 信息权威度
                  # 5 = 一手资料（官方文档、论文、项目作者本人写的）
                  # 4 = 权威机构（大厂技术博客、知名研究团队实测报告）
                  # 3 = 专业分析（有数据支撑的深度技术文章）
                  # 2 = 二手转述（解读/介绍他人工作的文章、公众号转发）
                  # 1 = 泛泛内容（无具体数据、纯观点、营销文章）
resource: ""
sources:
  - raw/
timestamp: ""     # REQUIRED: ISO 8601 date extracted from the document itself (publication date, article date, report date).
                  # Look for: 发布日期, 文章日期, 报告时间, byline date, footer date, URL date pattern (YYYY-MM-DD or YYYY/MM).
                  # If multiple dates found, use the earliest (creation/publication date).
                  # ONLY use today's date as last resort if NO date exists anywhere in the document.
key_claims: []  # REQUIRED: 5-8 specific, searchable claims in the source's original language. Each must be a complete factual statement with concrete terms/numbers. Cover all major points — more claims = better recall coverage.
               # ALIAS RULE: Inline ALL known aliases, abbreviations, and cross-language equivalents directly in each claim.
               # BAD:  "CR 偏低需要优化"
               # GOOD: "Context Recall（CR，召回率，检索覆盖率）偏低（0.32），需通过扩大 wiki 覆盖度优化"
related_to: []
contradicts: []
supports: []
---

# Title

## Source

- Path: `raw/`
- Type:
- Author:
- Published:
- Imported:

## Summary

<!-- 2-4 paragraphs covering ALL major points. Must include:
  - Specific technical terms, tool names, algorithm names
  - Concrete numbers, metrics, percentages where present
  - Key comparisons or contrasts made in the source
  Do NOT omit facts just to be brief. -->

## Key Facts

<!-- REQUIRED quality rules for each bullet:
  - Must contain at least one specific term, name, number, or metric
  - Must be a complete factual statement that directly answers a question
  - GOOD: "UnWeaver reduces GraphRAG indexing token cost by 10-17x while matching its accuracy"
  - BAD: "UnWeaver is more efficient than GraphRAG"
  - GOOD: "ChromaDB suits prototypes under 100K vectors; Qdrant suits production 100K-1M with filtering"
  - BAD: "Different vector databases suit different use cases"

  ALIAS RULE (MANDATORY for search coverage):
  Inline ALL known aliases, abbreviations, and cross-language equivalents directly in each claim.
  - BAD:  "CR 偏低需要优化"
  - GOOD: "Context Recall（CR，召回率，检索覆盖率）偏低（0.32），需通过扩大 wiki 覆盖度优化"
  - BAD:  "使用 FTS 检索"
  - GOOD: "使用 FTS（Full-Text Search，全文检索，BM25）检索，配合 RRF（倒数排名融合）合并结果"
  This ensures FTS search finds the document regardless of which term the user queries.

  ENTITY RULE (MANDATORY for knowledge graph):
  Mark named entities inline using 【entity|type】 format. Types: 人物|组织|产品|技术|概念|项目|地点
  - GOOD: "【Karpathy|人物】提出的【LLM Wiki|概念】采用三层架构，由【Anthropic|组织】等团队验证"
  - GOOD: "【bge-small-zh|产品】（【BAAI|组织】出品）在【WikiLoop|项目】中用于向量嵌入，维度512"
  - BAD:  直接写名称不标注类型
  This enables cross-document entity linking and multi-hop retrieval.

  STRUCTURED DOCUMENT RULE (tables, numbered lists, entity catalogs):
  If the source contains numbered/coded items (e.g. M01-M43, API list, field catalog),
  ALL items MUST be preserved — do NOT summarize, merge, or omit any entry.
  Each item must retain: code/ID, name, source system, storage table names, and any specific technical identifiers.
  Partial preservation is a critical quality failure.
-->

## Important Quotes Or Evidence

<!-- Include direct quotes that contain specific claims, numbers, or definitions.
     At minimum 1-2 quotes if the source contains quotable facts. -->

## Terms

<!-- Define every significant abbreviation, acronym, or technical term.
     Include the full expansion + one-line definition. -->

## Limitations

## Related Pages
