# RAG Technologies

Core technologies behind the RAG (Retrieval-Augmented Generation) pipeline — from raw document to generated answer.

```
Raw docs → chunk → embed → store → [query time] → retrieve → LLM → answer
```

## 1. Document Parsing

| Technology | Representative Tools | Notes |
|---|---|---|
| Multi-format parsing | [Unstructured.io](https://unstructured.io), [Docling](https://github.com/DS4SD/docling) (IBM) | Layout-aware extraction from PDF/Word/HTML |
| PDF parsing | PyMuPDF, PDFPlumber | Low-level text/table/layout extraction |
| LLM-enhanced parsing | LlamaParse | Cloud API for complex PDFs with tables and figures |
| PDF → Markdown | [Marker](https://github.com/VikParuchuri/marker), [MinerU](https://github.com/opendatalab/MinerU) | High-fidelity conversion, preserves structure |
| OCR cloud services | Azure Document Intelligence, AWS Textract | Enterprise-grade form and table recognition |

## 2. Chunking Strategies

| Strategy | Tools | Notes |
|---|---|---|
| Fixed-length chunking | LangChain `CharacterTextSplitter` | Simple, breaks semantic boundaries |
| Recursive splitting | LangChain `RecursiveCharacterTextSplitter` | Respects paragraph → sentence → word hierarchy |
| Semantic chunking | LlamaIndex `SemanticSplitter` | Detects topic boundaries via embedding similarity |
| Chapter-tree indexing | [he-wiki-rag](https://github.com/liuhe37186/he-wiki-rag) | Preserves H1/H2/H3 hierarchy + breadcrumb path |
| Parent-child chunking | LlamaIndex | Retrieve small chunks, return parent as context |
| Proposition chunking | Custom + LLM | Split into atomic facts — highest precision, expensive |
| Late Chunking | Jina AI | Embed full document first, then split embeddings |
| RAPTOR | LlamaIndex | Recursive tree summaries for hierarchical retrieval |
| Document-tree reasoning | [PageIndex](https://github.com/vectifyai/vectify) | LLM traverses summary tree — no vectors needed, 98.7% on FinanceBench |

## 3. Embedding Models

| Model | Provider | Notes |
|---|---|---|
| text-embedding-3-large/small | OpenAI | General-purpose, 3072-dim large variant |
| Embed v3 | Cohere | Optimized for retrieval, 1024-dim |
| BGE-M3 | BAAI (open source) | Multilingual, multi-granularity, strong Chinese |
| E5-Mistral-7B | Microsoft (open source) | High-dim, top MTEB ranking |
| NV-Embed-v2 | NVIDIA (open source) | MTEB leader among open models |
| nomic-embed-text | Nomic (open source) | Fully open, local-friendly |
| jina-embeddings-v3 | Jina AI (open source) | Supports Late Chunking |

## 4. Vector Databases

| Database | Type | Notes |
|---|---|---|
| [Qdrant](https://github.com/qdrant/qdrant) | Open source | Rust, high-performance, strong filtering. 100k–10M scale. |
| [Milvus](https://github.com/milvus-io/milvus) | Open source | Billion-scale distributed. Commercial: Zilliz Cloud. |
| [Weaviate](https://github.com/weaviate/weaviate) | Open source | Native hybrid search (vector + BM25) |
| [Chroma](https://github.com/chroma-core/chroma) | Open source | Lightweight, embedded, best for prototyping |
| [pgvector](https://github.com/pgvector/pgvector) | Extension | Vector search inside PostgreSQL |
| [LanceDB](https://github.com/lancedb/lancedb) | Open source | Arrow format, embedded, serverless-friendly |
| Pinecone | Managed SaaS | Serverless, zero ops |

## 5. Retrieval Strategies

| Strategy | Tools | Notes |
|---|---|---|
| Dense retrieval | All vector DBs (ANN) | Cosine/dot-product semantic similarity |
| Sparse retrieval (BM25) | Elasticsearch, OpenSearch, Tantivy | Term-frequency keyword matching |
| Hybrid retrieval | Weaviate, Qdrant, RRF algorithm | Dense + sparse, merged via Reciprocal Rank Fusion |
| Graph-augmented retrieval | [GraphRAG](https://github.com/microsoft/graphrag), [LightRAG](https://github.com/HKUDS/LightRAG) | Entity/relation graph for multi-hop reasoning |
| Vector Graph RAG | Community | Triples vectorized instead of graph DB — 96.3% on HotpotQA |
| Agentic RAG (A-RAG) | Custom + LLM | Agent autonomously chooses `keyword_search` / `semantic_search` / `chunk_read` tools |

## 6. Query Optimization

| Technique | Tools | Notes |
|---|---|---|
| HyDE | LangChain, LlamaIndex | Generate hypothetical answer, use its embedding to retrieve |
| Multi-query / Query rewriting | LangChain `MultiQueryRetriever` | LLM rewrites to multiple sub-questions |
| Step-Back Prompting | LangChain | Abstract specific question to general before retrieval |
| Self-RAG | Research implementation | LLM self-evaluates retrieval quality, decides whether to re-retrieve |
| CRAG | Research implementation | Falls back to web search for low-confidence retrievals |

## 7. Reranking

| Tool | Notes |
|---|---|
| Cohere Rerank | Cross-encoder commercial reranker |
| BGE Reranker (BAAI) | Open-source cross-encoder, strong multilingual |
| FlashRank | Lightweight open-source reranker for local deployment |

## 8. Orchestration Frameworks

| Framework | Notes |
|---|---|
| [LangChain](https://github.com/langchain-ai/langchain) | Most popular modular RAG pipeline framework |
| [LlamaIndex](https://github.com/run-llama/llama_index) | Data-centric RAG, richest chunking/retrieval strategies |
| [Haystack](https://github.com/deepset-ai/haystack) | Enterprise-grade NLP/RAG pipeline |
| [DSPy](https://github.com/stanfordnlp/dspy) | Programmatic LLM optimization, replaces hand-written prompts |

## 9. Evaluation

| Tool | Notes |
|---|---|
| [RAGAS](https://ragas.io) | Most popular RAG eval — faithfulness, context recall, answer relevancy |
| [DeepEval](https://github.com/confident-ai/deepeval) | Unit-test style RAG evaluation |
| [TruLens](https://www.trulens.org) | Production monitoring via RAG Triad |
| LangSmith | LangChain's tracing and eval platform |
| Key metrics | Faithfulness, Context Recall, Answer Relevancy, MRR, NDCG |

---

See also: [LLM Wiki Technologies](/ecosystem/llm-wiki-technologies) · [Paradigm Comparison](/ecosystem/paradigm-comparison)
