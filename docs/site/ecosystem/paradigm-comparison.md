# Paradigm Comparison: RAG vs LLM Wiki

Two fundamentally different approaches to building AI knowledge infrastructure.

## Pipeline Overview

```
RAG Pipeline
────────────────────────────────────────────────────────
Raw docs → chunk → embed → vector store
                                         ↘ [query time]
                              query → ANN search → LLM → answer


LLM Wiki (Knowledge Compilation)
────────────────────────────────────────────────────────
Raw docs → LLM compiles → structured wiki pages → FTS index
                                                           ↘ [query time]
                                    query → FTS / graph → read page → answer
```

## Dimension-by-Dimension Comparison

| Dimension | RAG | LLM Wiki |
|---|---|---|
| **Core operation** | Retrieve at query time | Compile at write time |
| **Storage** | Vector DB + raw chunks | Structured Markdown + SQLite FTS |
| **Search** | ANN similarity search | FTS5 BM25 + graph traversal |
| **Knowledge form** | Implicit (vectors) | Explicit (readable Markdown) |
| **Auditability** | Low | High (git diff, lint) |
| **Multi-hop reasoning** | LLM-dependent | Via `related` graph links |
| **Embedding required** | Yes | No (pure FTS) |
| **Knowledge accumulation** | None (static index) | Compounds over time |
| **Token cost at query time** | High (raw chunks passed to LLM) | Low (pre-compiled page) |
| **Latency** | Higher (ANN + LLM) | Lower (FTS sub-millisecond) |
| **Infrastructure** | Vector DB required | SQLite only |
| **Human readability** | No (vectors opaque) | Yes (plain Markdown) |
| **Best for** | Broad doc Q&A, real-time/live data | Long-term knowledge, agent memory, research |

## When to Use Each

**Choose RAG when:**
- You need to query a large, dynamic corpus that changes frequently
- Real-time or near-real-time data ingestion is required
- Semantic similarity across heterogeneous documents matters most
- You don't need to maintain or review the knowledge layer

**Choose LLM Wiki when:**
- Knowledge needs to accumulate and improve over time
- Auditability and human review of knowledge are important
- Agents need to search and reason across a curated knowledge base
- You want to minimize query-time token costs
- You want offline-capable, zero-infrastructure search

## They Are Not Mutually Exclusive

Many production systems combine both:
- **RAG** for broad retrieval across large, changing corpora
- **LLM Wiki** as a curated, high-quality knowledge layer for known domains

Example hybrid: raw ingestion via RAG pipeline → high-confidence results compiled into wiki pages for long-term reuse.

---

See also: [RAG Technologies](/ecosystem/rag-technologies) · [LLM Wiki Technologies](/ecosystem/llm-wiki-technologies)
