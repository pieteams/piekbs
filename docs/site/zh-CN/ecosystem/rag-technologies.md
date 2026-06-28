# RAG 关键技术

RAG（检索增强生成）管道背后的核心技术——从原始文档到生成答案的全链路。

```
原始文档 → 切块 → 嵌入 → 存储 → [查询时] → 检索 → LLM → 答案
```

## 1. 文档解析层

| 技术 | 代表工具 | 说明 |
|---|---|---|
| 多格式文档解析 | [Unstructured.io](https://unstructured.io)、[Docling](https://github.com/DS4SD/docling)（IBM） | 版面感知的 PDF/Word/HTML 多格式提取 |
| PDF 底层解析 | PyMuPDF、PDFPlumber | 文本/表格/布局结构提取 |
| LLM 增强解析 | LlamaParse | 处理含复杂表格和图形 PDF 的云端 API |
| PDF 转 Markdown | [Marker](https://github.com/VikParuchuri/marker)、[MinerU](https://github.com/opendatalab/MinerU) | 高保真格式转换，保留文档结构 |
| OCR 云服务 | Azure Document Intelligence、AWS Textract | 企业级表单和表格识别 |

## 2. 文本切块策略

| 策略 | 工具 | 说明 |
|---|---|---|
| 固定长度切块 | LangChain `CharacterTextSplitter` | 简单，但会破坏语义边界 |
| 递归字符切割 | LangChain `RecursiveCharacterTextSplitter` | 按段落→句子→单词层级递归切割 |
| 语义切块 | LlamaIndex `SemanticSplitter` | 通过 Embedding 相似度检测主题边界 |
| 章节树索引 | [he-wiki-rag](https://github.com/liuhe37186/he-wiki-rag) | 保留 H1/H2/H3 层级和面包屑路径 |
| 父子分块 | LlamaIndex | 检索小粒度子块，返回父块作为上下文 |
| 命题切块 | 自定义 + LLM | 拆分为原子事实——精度最高，代价最大 |
| Late Chunking | Jina AI | 先嵌入完整文档再切分 Embedding |
| RAPTOR | LlamaIndex | 递归树状摘要，构建层次化检索结构 |
| 文档树推理 | [PageIndex](https://github.com/vectifyai/vectify) | LLM 遍历摘要树检索，无需向量，FinanceBench 98.7% |

## 3. Embedding 模型

| 模型 | 提供方 | 说明 |
|---|---|---|
| text-embedding-3-large/small | OpenAI | 通用嵌入，large 版 3072 维 |
| Embed v3 | Cohere | 专为检索优化，1024 维 |
| BGE-M3 | BAAI（开源） | 多语言多粒度，中文表现优异 |
| E5-Mistral-7B | Microsoft（开源） | 高维度，MTEB 排名靠前 |
| NV-Embed-v2 | NVIDIA（开源） | 开源模型中 MTEB 领先 |
| nomic-embed-text | Nomic（开源） | 完全开源，本地友好 |
| jina-embeddings-v3 | Jina AI（开源） | 支持 Late Chunking |

## 4. 向量数据库

| 数据库 | 类型 | 说明 |
|---|---|---|
| [Qdrant](https://github.com/qdrant/qdrant) | 开源 | Rust 实现，高性能，强过滤，适合 10 万~千万规模 |
| [Milvus](https://github.com/milvus-io/milvus) | 开源 | 十亿级分布式，商业版：Zilliz Cloud |
| [Weaviate](https://github.com/weaviate/weaviate) | 开源 | 原生混合检索（向量 + BM25） |
| [Chroma](https://github.com/chroma-core/chroma) | 开源 | 轻量嵌入式，原型开发首选 |
| [pgvector](https://github.com/pgvector/pgvector) | 扩展 | PostgreSQL 原生向量搜索扩展 |
| [LanceDB](https://github.com/lancedb/lancedb) | 开源 | Arrow 格式，嵌入式，无服务器友好 |
| Pinecone | 托管 SaaS | Serverless，零运维 |

## 5. 检索策略

| 策略 | 工具 | 说明 |
|---|---|---|
| 密集检索 | 各向量数据库 ANN | 余弦/点积语义相似度近似最近邻搜索 |
| 稀疏检索（BM25） | Elasticsearch、OpenSearch、Tantivy | 基于词频统计的关键词匹配 |
| 混合检索 | Weaviate、Qdrant、RRF 算法 | 密集 + 稀疏，通过 RRF 倒数排名融合 |
| 图增强检索 | [GraphRAG](https://github.com/microsoft/graphrag)、[LightRAG](https://github.com/HKUDS/LightRAG) | 实体/关系图谱支持多跳推理 |
| Vector Graph RAG | 社区 | 三元组向量化替代图数据库，HotpotQA 96.3% |
| Agentic RAG（A-RAG） | 自定义 + LLM | Agent 自主选择 `keyword_search` / `semantic_search` / `chunk_read` 工具 |

## 6. 查询优化

| 技术 | 工具 | 说明 |
|---|---|---|
| HyDE | LangChain、LlamaIndex | 生成假设性答案，用其 Embedding 做检索 |
| Multi-Query 查询改写 | LangChain `MultiQueryRetriever` | LLM 将原问题改写为多个子问题扩大召回 |
| Step-Back Prompting | LangChain | 将具体问题抽象为通用问题再检索 |
| Self-RAG | 论文实现 | LLM 自评估检索质量，决定是否继续检索 |
| CRAG | 论文实现 | 低置信度时回退到网络搜索 |

## 7. Rerank 精排

| 工具 | 说明 |
|---|---|
| Cohere Rerank | 交叉编码器商业精排模型 |
| BGE Reranker（BAAI） | 开源交叉编码器，多语言表现强 |
| FlashRank | 轻量开源精排库，适合本地部署 |

## 8. 编排框架

| 框架 | 说明 |
|---|---|
| [LangChain](https://github.com/langchain-ai/langchain) | 最流行的模块化 RAG 管道框架 |
| [LlamaIndex](https://github.com/run-llama/llama_index) | 数据中心型 RAG，最丰富的切块/检索策略 |
| [Haystack](https://github.com/deepset-ai/haystack) | 企业级 NLP/RAG 管道框架 |
| [DSPy](https://github.com/stanfordnlp/dspy) | 通过编程方式优化 LLM 管道，替代手写 Prompt |

## 9. 评估框架

| 工具 | 说明 |
|---|---|
| [RAGAS](https://ragas.io) | 最流行的 RAG 评估框架——忠实度、上下文召回、答案相关性 |
| [DeepEval](https://github.com/confident-ai/deepeval) | 单元测试风格的 RAG 评估库 |
| [TruLens](https://www.trulens.org) | 基于 RAG Triad 的生产监控工具 |
| LangSmith | LangChain 官方链路追踪与评估平台 |
| 核心指标 | Faithfulness、Context Recall、Answer Relevancy、MRR、NDCG |

---

另见：[LLM Wiki 关键技术](/zh-CN/ecosystem/llm-wiki-technologies) · [范式对比](/zh-CN/ecosystem/paradigm-comparison)
