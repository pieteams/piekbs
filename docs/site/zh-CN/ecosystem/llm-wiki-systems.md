# LLM Wiki 系统

AI 驱动的 Wiki 平台、知识管理工具和 LLM 知识库系统精选列表。最后更新：2025年。

## 含 AI 功能的 Wiki 平台

| 项目 | 公司 | Stars | 是否开源 | 简介 |
|---|---|---|---|---|
| [Outline](https://github.com/outline/outline) | Outline HQ | 29k+ | BSL 1.1 | 团队协作 Wiki，支持 OpenAI/Claude/Ollama 多模型，内置语义搜索、AI 写作助手和 RAG 问答。 |
| [AppFlowy](https://github.com/AppFlowy-IO/AppFlowy) | AppFlowy Inc. | — | AGPL | Notion 开源替代，本地优先，支持 LLM 插件。 |
| [Docmost](https://github.com/docmost/docmost) | 社区 | — | AGPL | 新兴开源 Wiki，功能类 Confluence/Notion，AI 功能在路线图中。 |
| [Wiki.js](https://github.com/requarks/wiki) | Requarks | — | AGPL | 模块化 Wiki 平台，原生无 AI，可通过扩展接入 LLM，v3.0 路线图含 AI 特性。 |
| [BookStack](https://github.com/BookStackApp/BookStack) | Dan Brown | — | MIT | 简洁自托管文档 Wiki，无原生 AI，提供 API 可二次集成 LLM。 |
| [Confluence](https://www.atlassian.com/software/confluence) | Atlassian | — | 商业 | 企业 Wiki 标杆，集成 Atlassian Intelligence + Rovo AI，支持 AI 摘要和自然语言搜索。 |
| [Notion AI](https://www.notion.so) | Notion Labs | — | 商业 | 知名 All-in-One 工作空间，AI 层支持写作、摘要、Q&A、数据库自动填充，2025 年推出 AI Agent。 |

## 企业知识平台

| 产品 | 公司 | 是否开源 | 简介 |
|---|---|---|---|
| [Onyx（原 Danswer）](https://github.com/onyx-dot-app/onyx) | Onyx（YC W23） | MIT（社区版） | 连接 50+ 数据源（Confluence/Notion/Slack），自托管 RAG 企业问答平台，支持任意 LLM。⭐13k+ |
| [Glean](https://www.glean.com) | Glean Technologies | 否 | 企业级跨工具 AI 搜索（100+ 集成），构建组织知识图谱，权限感知，大型企业首选。 |
| [Guru](https://www.getguru.com) | Guru Technologies | 否 | AI 企业知识管理，Guru AI 支持实时从内部文档回答问题，集成 Slack/Teams。 |
| [Tettra](https://www.tettra.com) | Tettra | 否 | Slack 原生内部知识库，Kai AI 在 Slack 中直接回答问题，擅长知识空白检测。 |
| [Document360](https://www.document360.com) | Kovai.co | 否 | 内外部文档兼顾，Eddy AI 支持搜索问答和嵌入式客服聊天机器人。 |
| [Slite](https://slite.com) | Slite | 否 | AI 驱动文档管理，自动打标签、搜索、摘要，适合中小团队。 |

## 含 LLM 的个人知识管理工具（PKM）

| 项目 | 公司 | Stars | 是否开源 | 简介 |
|---|---|---|---|---|
| [Logseq](https://github.com/logseq/logseq) | Logseq | 30k+ | AGPL | 大纲+图谱 PKM，DB 版本支持更强 AI 集成，社区插件接入 LLM。 |
| [思源笔记 SiYuan](https://github.com/siyuan-note/siyuan) | 社区 | 20k+ | AGPL | 块级编辑器，本地优先，社区插件支持 AI 摘要与 RAG。 |
| [Foam](https://github.com/foambubble/foam) | 社区 | — | MIT | VS Code 内 Roam Research 风格 PKM，可通过 Continue.dev 接入 LLM。 |
| [Trilium Notes](https://github.com/zadam/trilium) | 社区 | — | AGPL | 层级式自托管个人知识库，社区有 LLM 集成脚本。 |
| [Obsidian](https://obsidian.md) | Obsidian | — | 部分开源（插件 MIT） | 本地优先 Markdown 知识图谱，Smart Connections/Copilot 插件支持 Ollama/OpenAI RAG。 |
| [Mem.ai](https://mem.ai) | Mem Labs | — | 否 | AI 原生笔记，Mem X 自动组织和关联笔记，完全基于 LLM 驱动。 |
| [Capacities](https://capacities.io) | Capacities | — | 否 | 对象型 PKM，2025 年集成 LLM 写作助手和智能搜索。 |
| [Google NotebookLM](https://notebooklm.google.com) | Google | — | 否（免费） | LLM 原生笔记，上传文档后自动摘要、生成播客、问答，深度 Gemini 集成。 |

## Agent 记忆框架

专注于为 AI Agent 提供跨会话持久记忆的系统，与文档问答定位不同。

| 项目 | 公司 | Stars | 是否开源 | 简介 |
|---|---|---|---|---|
| [Cognee](https://github.com/topoteretes/cognee) | 社区 | — | Apache 2.0 | 开源 AI 记忆平台，自动构建知识图谱，提供 `remember` / `recall` / `forget` / `improve` 四操作 API，向量搜索+图推理实现 Agent 跨会话持久记忆。 |
| [Mem0](https://github.com/mem0ai/mem0) | Mem0 AI | — | Apache 2.0 | 面向 AI Agent 和助手的持久记忆层，跨会话存储和检索用户偏好与上下文。 |
| [Letta](https://github.com/letta-ai/letta) | Letta AI | — | Apache 2.0 | 有状态 Agent，持久记忆由 Neo4j 支撑，前身为 MemGPT。 |
| [MemOS](https://github.com/MemTensor/MemOS) | MemTensor | — | Apache 2.0 | Agent 记忆操作系统，主动记忆管理、混合检索、技能进化，配套 MemReader 记忆提取模型。 |
| [Engram](https://github.com/Gentleman-Programming/engram) | 社区 | — | MIT | 轻量级本地优先的 AI 编码 Agent 持久记忆系统，是 Mem0+Qdrant 或 Letta+Neo4j 方案的轻量替代。 |

## 研究与特殊项目

| 项目 | 机构 | Stars | 简介 |
|---|---|---|---|
| [WikiChat](https://github.com/stanford-oval/WikiChat) | 斯坦福 OVAL 实验室 | — | 以 Wikipedia 为基础减少 LLM 幻觉，先检索验证再生成回答。 |
| [LLM Wiki](https://github.com/nashsu/llm_wiki) | nashsu（社区） | — | 基于 Karpathy 的 LLM Wiki 理念开发的 Tauri 桌面应用，增量式将文档构建为持久化、相互链接的 Wiki，Obsidian 兼容输出，"人类策展，LLM 维护"。 |
| [GBrain](https://github.com/garrytan/gbrain) | Garry Tan（YC CEO） | — | 知识操作系统。处理多源、多媒体输入（语音转录等），持续更新结构化知识图谱，为 Agent 提供长期记忆的运行层基础设施。 |
| [Sage Wiki](https://github.com/joneyao/sage-wiki) | 社区 | — | Go 单二进制 LLM Wiki 实现，分层编译支持 10 万+文档，混合搜索（FTS+向量+本体图谱），提供 17 个 MCP 工具，相比朴素 RAG 最高节省 95% token。 |
| [Walnut](https://github.com/wimham/walnut) | 社区 | — | 本地优先 AI 知识管理 Agent，BYOK 模式（OpenAI/Claude/任意供应商），自动整理文档，在写作时主动推荐关联历史笔记。 |
| [WikiLoop](https://github.com/jasen215/wikiloop) | — | — | 面向 Agent 的本地优先知识搜索引擎，将原始文档提炼为结构化 Markdown 知识库，通过 MCP 提供搜索和阅读能力。 |
| [Mintlify](https://mintlify.com) | Mintlify | — | 面向开发者的 AI 文档生成，自动从代码生成 Wiki/API 文档。 |
