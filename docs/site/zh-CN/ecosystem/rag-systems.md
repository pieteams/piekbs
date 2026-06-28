# RAG 知识库系统

主流 RAG（检索增强生成）框架、平台和向量数据库精选列表。最后更新：2025年。

## RAG 框架

面向开发者的 RAG 管道构建库。

| 项目 | 公司 | Stars | 协议 | 简介 |
|---|---|---|---|---|
| [LangChain](https://github.com/langchain-ai/langchain) | LangChain Inc.（美国，a16z 投资） | 90k+ | MIT | LLM 应用开发的事实标准框架，模块化 Chain 架构支持 RAG；配套 LangGraph 支持 Agentic RAG，LangSmith 提供可观测性。 |
| [LlamaIndex](https://github.com/run-llama/llama_index) | LlamaIndex Inc.（美国） | 38k+ | MIT | 专为 RAG 设计的数据框架，提供 100+ 数据连接器、分层检索、多模态 RAG；文档 Q&A 场景最佳选择。 |
| [Haystack](https://github.com/deepset-ai/haystack) | deepset（德国） | 17k+ | Apache 2.0 | 企业级端到端 LLM/RAG Pipeline 框架，v2 于 2024 年重新设计；内置评估工具，生产就绪。 |

## 一体化知识库平台

| 项目 | 公司 | Stars | 协议 | 简介 |
|---|---|---|---|---|
| [Dify](https://github.com/langgenius/dify) | LangGenius（中国） | 45k+ | Apache 2.0 | 可视化拖拽式 LLM 应用平台，内置 RAG Pipeline 和知识库管理，支持 100+ 模型。 |
| [RAGFlow](https://github.com/infiniflow/ragflow) | InfiniFlow（中国） | 30k+ | Apache 2.0 | 基于深度文档理解的 RAG 引擎，擅长处理复杂 PDF/表格/图表，支持引用溯源。 |
| [FastGPT](https://github.com/labring/FastGPT) | labring（中国） | 20k+ | Apache 2.0 | 基于 LLM 的知识库平台，内置可视化 Flow 工作流编排，TypeScript/Next.js 构建，支持团队协作。 |
| [AnythingLLM](https://github.com/Mintplex-Labs/anything-llm) | Mintplex Labs（美国） | 30k+ | MIT | 全栈 LLM 桌面/服务端应用，将任意文档转为 RAG 聊天机器人；多用户、本地+云端 LLM、多向量数据库。 |
| [MaxKB](https://github.com/1panel-dev/MaxKB) | 飞致云 FIT2CLOUD（中国） | — | Apache 2.0 | 基于 RAG 的企业知识库问答机器人，支持多 LLM 后端，与 1Panel/JumpServer 同生态。 |
| [Quivr](https://github.com/QuivrHQ/quivr) | Quivr HQ（法国） | 36k+ | Apache 2.0 | 定位"AI 第二大脑"的个人知识助手，支持多格式文件 RAG 检索，支持云端和本地部署。 |
| [PrivateGPT](https://github.com/zylon-ai/private-gpt) | Zylon（隐私 AI 创业） | 54k+ | Apache 2.0 | 100% 本地化隐私优先的文档问答工具，全程离线运行无需联网。 |
| [Kotaemon](https://github.com/Cinnamon/kotaemon) | Cinnamon AI（越南/日本） | 18k+ | Apache 2.0 | 简洁可定制的文档 RAG 聊天工具，Gradio UI，支持多向量检索、引用展示和 GraphRAG 集成。 |
| [Onyx（原 Danswer）](https://github.com/onyx-dot-app/onyx) | Onyx（YC W23） | 13k+ | MIT | 连接 50+ 数据源（Confluence/Notion/Slack 等），自托管 RAG 企业问答平台，支持任意 LLM。 |
| [WeKnora](https://github.com/Tencent/WeKnora) | 腾讯（中国） | — | Apache 2.0 | 腾讯开源的企业知识平台，"Wiki Mode" 将 Karpathy 的 LLM-as-Compiler 范式产品化。混合检索（向量+BM25+知识图谱）、多租户 RBAC、内建 Agent、IM 原生集成。 |

## 向量数据库

| 项目 | 公司 | Stars | 协议 | 简介 |
|---|---|---|---|---|
| [Chroma](https://github.com/chroma-core/chroma) | Chroma（美国，YC 孵化） | 16k+ | Apache 2.0 | 轻量级嵌入式向量数据库，RAG 原型开发最易上手的选择。 |
| [Weaviate](https://github.com/weaviate/weaviate) | Weaviate B.V.（荷兰，融资 $50M+） | 11k+ | BSD | 带 GraphQL API 的向量数据库，支持混合搜索和多模态检索。 |
| [Qdrant](https://github.com/qdrant/qdrant) | Qdrant（德国，Series A $28M） | 21k+ | Apache 2.0 | Rust 编写的高性能向量搜索引擎，生产级 RAG 过滤能力极强。 |
| [Milvus](https://github.com/milvus-io/milvus) | Zilliz（中美双栖，融资 $113M+） | 30k+ | Apache 2.0 | 云原生向量数据库，支持十亿级向量规模；商业版为 Zilliz Cloud。 |
| [pgvector](https://github.com/pgvector/pgvector) | 社区（Supabase 主要贡献） | — | PostgreSQL | PostgreSQL 向量扩展，支持 HNSW/IVFFlat 索引；已用 Postgres 的团队零成本接入向量检索。 |

## 托管向量数据库服务

| 产品 | 公司 | 简介 |
|---|---|---|
| [Pinecone](https://www.pinecone.io) | Pinecone Systems（美国，融资 $100M+） | 全托管 Serverless 向量数据库，零运维，最成熟的托管 RAG 存储方案，闭源。 |
| [Zilliz Cloud](https://zilliz.com) | Zilliz（Milvus 母公司） | Milvus 的全托管云服务，企业级 SLA 和安全性。 |

## Graph RAG

| 项目 | 公司 | Stars | 简介 |
|---|---|---|---|
| [GraphRAG](https://github.com/microsoft/graphrag) | Microsoft Research | 19k+ | 通过知识图谱社区摘要回答复杂多跳问题。 |
| [LightRAG](https://github.com/HKUDS/LightRAG) | 香港大学数据智能实验室 | 15k+ | 图向量双索引轻量 RAG 框架，支持局部/全局双模式检索，2024 年底兴起。 |
| [UnWeaver](https://github.com/whyhow-ai/unweaver) | 社区 | — | 在 VectorRAG 基础上增加实体级信息拆解与聚合，无需构建显式知识图谱即可在事实正确性上超越 GraphRAG。 |
| [TrustGraph](https://github.com/trustgraph-ai/trustgraph) | 社区 | — | 知识图谱与 RAG 平台，将结构化图谱推理与向量检索结合。 |

## 无向量 RAG

不依赖 Embedding 相似度的替代检索方案。

| 项目 | 公司 | Stars | 简介 |
|---|---|---|---|
| [PageIndex](https://github.com/vectifyai/vectify) | VectifyAI | 29k+ | 用语义摘要树替代向量数据库，检索即推理（LLM 树搜索），在 FinanceBench 上达到 98.7% 准确率。 |
| [Memvid](https://github.com/olow304/memvid) | 社区 | — | 单个 `.mv2` 文件取代整个 RAG 管道和向量数据库，亚毫秒级检索。 |
| [OhMyGo](https://github.com/masnun/ohmygo) | 社区 | — | 基于 Go 语言实现的 AI 知识库平台，集成聊天、RAG 问答、图像识别和语音合成。 |
| [he-wiki-rag](https://github.com/liuhe37186/he-wiki-rag) | 社区 | — | 章节树索引保留 Markdown H1/H2/H3 层级，避免固定切块的语义丢失。向量+BM25+Cross-Encoder Rerank 三阶段混合检索管线。 |

## 传统搜索引擎（RAG 扩展）

| 产品 | 公司 | 简介 |
|---|---|---|
| [Elasticsearch](https://www.elastic.co) | Elastic N.V.（纽交所上市） | 老牌企业搜索引擎，通过 ESRE 和 ELSER 稀疏向量模型扩展 RAG，亿级规模混合搜索成熟。 |
| [OpenSearch](https://opensearch.org) | AWS（Amazon） | Elasticsearch 的 AWS 开源分叉，深度集成 Amazon Bedrock 知识库，Neural Sparse Retrieval 支持云原生 RAG。 |
