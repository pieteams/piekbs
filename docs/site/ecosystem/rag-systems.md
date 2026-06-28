# RAG Knowledge Base Systems

A curated list of RAG (Retrieval-Augmented Generation) frameworks, platforms, and vector databases. Last updated: 2025.

## RAG Frameworks

Developer-facing libraries for building RAG pipelines.

| Project | Company | Stars | License | Description |
|---|---|---|---|---|
| [LangChain](https://github.com/langchain-ai/langchain) | LangChain Inc. (US, a16z) | 90k+ | MIT | De facto standard LLM app framework. Modular chain architecture for RAG; LangGraph for agentic RAG, LangSmith for observability. |
| [LlamaIndex](https://github.com/run-llama/llama_index) | LlamaIndex Inc. (US) | 38k+ | MIT | Data framework purpose-built for RAG. 100+ connectors, hierarchical retrieval, multimodal RAG. Best for document Q&A. |
| [Haystack](https://github.com/deepset-ai/haystack) | deepset (Germany) | 17k+ | Apache 2.0 | Enterprise end-to-end LLM/RAG pipeline framework. v2 redesigned in 2024; built-in evaluation, production-ready. |

## All-in-One Knowledge Base Platforms

| Project | Company | Stars | License | Description |
|---|---|---|---|---|
| [Dify](https://github.com/langgenius/dify) | LangGenius (China) | 45k+ | Apache 2.0 | Visual drag-and-drop LLM app platform with built-in RAG pipeline and knowledge base management. Supports 100+ models. |
| [RAGFlow](https://github.com/infiniflow/ragflow) | InfiniFlow (China) | 30k+ | Apache 2.0 | RAG engine based on deep document understanding. Excels at complex PDFs, tables, and charts. Supports citation tracing. |
| [FastGPT](https://github.com/labring/FastGPT) | labring (China) | 20k+ | Apache 2.0 | LLM knowledge base platform with visual Flow workflow orchestration. TypeScript/Next.js, team collaboration support. |
| [AnythingLLM](https://github.com/Mintplex-Labs/anything-llm) | Mintplex Labs (US) | 30k+ | MIT | Full-stack desktop/server LLM app. Turns any document into a RAG chatbot. Multi-user, local+cloud LLM, multi-vector-DB. |
| [MaxKB](https://github.com/1panel-dev/MaxKB) | FIT2CLOUD (China) | — | Apache 2.0 | Enterprise RAG Q&A bot supporting multiple LLM backends. From the 1Panel/JumpServer ecosystem. |
| [Quivr](https://github.com/QuivrHQ/quivr) | Quivr HQ (France) | 36k+ | Apache 2.0 | "AI second brain" personal knowledge assistant. Multi-format RAG retrieval, cloud and local deployment. |
| [PrivateGPT](https://github.com/zylon-ai/private-gpt) | Zylon (privacy AI) | 54k+ | Apache 2.0 | 100% local, privacy-first document Q&A. Fully offline. One of the earliest viral local RAG projects. |
| [Kotaemon](https://github.com/Cinnamon/kotaemon) | Cinnamon AI (Vietnam/Japan) | 18k+ | Apache 2.0 | Clean, customizable document RAG chat tool with Gradio UI. Multi-vector retrieval, citation display, GraphRAG integration. |
| [Onyx (Danswer)](https://github.com/onyx-dot-app/onyx) | Onyx (YC W23) | 13k+ | MIT | Connects 50+ sources (Confluence/Notion/Slack). Self-hosted RAG enterprise Q&A platform, any LLM. |
| [WeKnora](https://github.com/Tencent/WeKnora) | Tencent (China) | — | Apache 2.0 | Enterprise knowledge platform open-sourced by Tencent. "Wiki Mode" productizes Karpathy's LLM-as-Compiler pattern. Hybrid retrieval (vector + BM25 + knowledge graph), multi-tenant RBAC, built-in Agent, IM integration. |

## Vector Databases

| Project | Company | Stars | License | Description |
|---|---|---|---|---|
| [Chroma](https://github.com/chroma-core/chroma) | Chroma (US, YC) | 16k+ | Apache 2.0 | Lightweight embedded vector database. Easiest to get started with for RAG prototyping. |
| [Weaviate](https://github.com/weaviate/weaviate) | Weaviate B.V. (Netherlands, $50M+) | 11k+ | BSD | Vector database with GraphQL API. Supports hybrid search and multimodal retrieval. |
| [Qdrant](https://github.com/qdrant/qdrant) | Qdrant (Germany, $28M Series A) | 21k+ | Apache 2.0 | High-performance vector search engine written in Rust. Excellent filtering for production RAG. |
| [Milvus](https://github.com/milvus-io/milvus) | Zilliz (US/China, $113M+) | 30k+ | Apache 2.0 | Cloud-native vector database for billion-scale vectors. Commercial: Zilliz Cloud. |
| [pgvector](https://github.com/pgvector/pgvector) | Community (Supabase contributor) | — | PostgreSQL | PostgreSQL vector extension with HNSW/IVFFlat index. Zero-cost vector retrieval if already on Postgres. |

## Managed Vector Database Services

| Product | Company | Description |
|---|---|---|
| [Pinecone](https://www.pinecone.io) | Pinecone Systems (US, $100M+) | Fully managed serverless vector database. Zero ops, most mature managed RAG storage option. Closed source. |
| [Zilliz Cloud](https://zilliz.com) | Zilliz (Milvus parent) | Managed cloud service for Milvus. Enterprise SLA and security. |

## Graph RAG

| Project | Company | Stars | Description |
|---|---|---|---|
| [GraphRAG](https://github.com/microsoft/graphrag) | Microsoft Research | 19k+ | Graph-based RAG via community summaries. Answers complex multi-hop questions. |
| [LightRAG](https://github.com/HKUDS/LightRAG) | HKU Data Intelligence Lab | 15k+ | Dual graph-vector index. Supports local/global dual-mode retrieval. Emerged late 2024. |
| [UnWeaver](https://github.com/whyhow-ai/unweaver) | Community | — | Entity-level decomposition + aggregation on top of VectorRAG. Beats GraphRAG in factual correctness without building an explicit knowledge graph. |
| [TrustGraph](https://github.com/trustgraph-ai/trustgraph) | Community | — | Knowledge graph + RAG platform. Combines structured graph reasoning with vector retrieval. |

## Vector-Free RAG

Alternative retrieval approaches that don't rely on embedding similarity.

| Project | Company | Stars | Description |
|---|---|---|---|
| [PageIndex](https://github.com/vectifyai/vectify) | VectifyAI | 29k+ | Semantic summary tree replaces vector database. Retrieval = reasoning via LLM tree search. 98.7% accuracy on FinanceBench. |
| [Memvid](https://github.com/olow304/memvid) | Community | — | Single `.mv2` file replaces the entire RAG pipeline and vector database. Sub-millisecond retrieval. |
| [OhMyGo](https://github.com/masnun/ohmygo) | Community | — | Go-based AI knowledge platform integrating chat, RAG Q&A, image recognition, and speech synthesis. |
| [he-wiki-rag](https://github.com/liuhe37186/he-wiki-rag) | Community | — | Chapter-tree indexing preserves Markdown H1/H2/H3 hierarchy to avoid semantic loss from fixed-length chunking. Vector + BM25 + Cross-Encoder Rerank pipeline. |

## Traditional Search Engines (RAG-extended)

| Product | Company | Description |
|---|---|---|
| [Elasticsearch](https://www.elastic.co) | Elastic N.V. (NYSE: ESTC) | Enterprise search extended with ESRE and ELSER sparse vectors. Mature hybrid search (BM25 + vector) at billion-scale. |
| [OpenSearch](https://opensearch.org) | AWS (Amazon) | AWS open-source Elasticsearch fork. Deep Amazon Bedrock integration. Neural Sparse Retrieval for cloud-native RAG. |
