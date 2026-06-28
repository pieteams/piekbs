# LLM Wiki Systems

A curated list of AI-powered wiki platforms, knowledge management tools, and LLM-driven knowledge base systems. Last updated: 2025.

## Wiki Platforms with AI

| Project | Company | Stars | Open Source | Description |
|---|---|---|---|---|
| [Outline](https://github.com/outline/outline) | Outline HQ | 29k+ | BSL 1.1 | Team wiki with OpenAI/Claude/Ollama support, semantic search, AI writing assistant, and RAG Q&A. |
| [AppFlowy](https://github.com/AppFlowy-IO/AppFlowy) | AppFlowy Inc. | — | AGPL | Open-source Notion alternative. Local-first, supports LLM plugins. |
| [Docmost](https://github.com/docmost/docmost) | Community | — | AGPL | Emerging open-source wiki similar to Confluence/Notion. AI features on the roadmap. |
| [Wiki.js](https://github.com/requarks/wiki) | Requarks | — | AGPL | Modular wiki platform. No native AI; extensible via APIs. v3.0 roadmap includes AI features. |
| [BookStack](https://github.com/BookStackApp/BookStack) | Dan Brown | — | MIT | Simple self-hosted documentation wiki. No native AI; API available for LLM integration. |
| [Confluence](https://www.atlassian.com/software/confluence) | Atlassian | — | Commercial | Enterprise wiki standard. Atlassian Intelligence + Rovo AI: AI summaries, natural language search. |
| [Notion AI](https://www.notion.so) | Notion Labs | — | Commercial | All-in-one workspace. AI layer: writing, summaries, Q&A, database auto-fill, AI Agents (2025). |

## Enterprise Knowledge Platforms

| Product | Company | Open Source | Description |
|---|---|---|---|
| [Onyx (Danswer)](https://github.com/onyx-dot-app/onyx) | Onyx (YC W23) | MIT (community) | Connects 50+ sources (Confluence/Notion/Slack). Self-hosted RAG enterprise Q&A, any LLM. ⭐13k+ |
| [Glean](https://www.glean.com) | Glean Technologies | No | Enterprise AI search across 100+ integrations. Builds org knowledge graph with permission-awareness. |
| [Guru](https://www.getguru.com) | Guru Technologies | No | AI knowledge management. Guru AI answers questions in real time from internal docs. Slack/Teams integration. |
| [Tettra](https://www.tettra.com) | Tettra | No | Slack-native internal wiki. Kai AI answers directly in Slack. Good at knowledge gap detection. |
| [Document360](https://www.document360.com) | Kovai.co | No | Internal/external docs platform. Eddy AI: search Q&A and embedded customer support chatbot. |
| [Slite](https://slite.com) | Slite | No | AI-driven document management. Auto-tagging, search, summaries. Good for small/mid teams. |

## Personal Knowledge Management (PKM) with LLM

| Project | Company | Stars | Open Source | Description |
|---|---|---|---|---|
| [Logseq](https://github.com/logseq/logseq) | Logseq | 30k+ | AGPL | Outline + graph PKM. DB version supports stronger AI integration; community plugins for LLM. |
| [SiYuan Note](https://github.com/siyuan-note/siyuan) | Community | 20k+ | AGPL | Block-level editor, local-first. Community plugins support AI summaries and RAG. |
| [Foam](https://github.com/foambubble/foam) | Community | — | MIT | Roam Research-style PKM inside VS Code. LLM integration via Continue.dev. |
| [Trilium Notes](https://github.com/zadam/trilium) | Community | — | AGPL | Hierarchical self-hosted personal knowledge base. Community LLM integration scripts available. |
| [Obsidian](https://obsidian.md) | Obsidian | — | Partial (plugins MIT) | Local-first Markdown knowledge graph. Smart Connections / Copilot plugins support Ollama/OpenAI RAG. |
| [Mem.ai](https://mem.ai) | Mem Labs | — | No | AI-native notes. Mem X auto-organizes and links notes. Fully LLM-driven. |
| [Capacities](https://capacities.io) | Capacities | — | No | Object-type PKM. LLM writing assistant and intelligent search (2025). |
| [Google NotebookLM](https://notebooklm.google.com) | Google | — | No (free) | LLM-native notebook. Upload docs → auto summaries, podcast generation, Q&A. Deep Gemini integration. |

## Agent Memory Frameworks

Systems focused on giving AI agents persistent, cross-session memory — distinct from document Q&A.

| Project | Company | Stars | Open Source | Description |
|---|---|---|---|---|
| [Cognee](https://github.com/topoteretes/cognee) | Community | — | Apache 2.0 | Open-source AI memory platform. Auto-builds knowledge graphs; `remember` / `recall` / `forget` / `improve` API. Vector search + graph reasoning for cross-session Agent memory. |
| [Mem0](https://github.com/mem0ai/mem0) | Mem0 AI | — | Apache 2.0 | Persistent memory layer for AI agents and assistants. Stores and retrieves user preferences and context across sessions. |
| [Letta](https://github.com/letta-ai/letta) | Letta AI | — | Apache 2.0 | Stateful agents with persistent memory backed by Neo4j. Formerly MemGPT. |
| [MemOS](https://github.com/MemTensor/MemOS) | MemTensor | — | Apache 2.0 | Memory operating system for agents. Active memory management, hybrid retrieval, skill evolution. MemReader model for memory extraction. |
| [Engram](https://github.com/Gentleman-Programming/engram) | Community | — | MIT | Lightweight, local-first persistent memory system for AI coding agents. Alternative to Mem0+Qdrant or Letta+Neo4j. |

## Research & Specialized Projects

| Project | Company | Stars | Description |
|---|---|---|---|
| [WikiChat](https://github.com/stanford-oval/WikiChat) | Stanford OVAL | — | Uses Wikipedia to reduce LLM hallucination. Retrieves and verifies before generating answers. |
| [LLM Wiki](https://github.com/nashsu/llm_wiki) | nashsu (community) | — | Desktop app (Tauri) inspired by Karpathy's LLM Wiki idea. Incrementally builds a persistent, interlinked wiki from your documents. Obsidian-compatible output. "Human curation, LLM maintenance." |
| [GBrain](https://github.com/garrytan/gbrain) | Garry Tan (YC CEO) | — | Knowledge operating system. Handles multi-source, multimedia input (audio transcription, etc.) and continuously updates a structured knowledge graph. Built for Agent long-term memory. |
| [Sage Wiki](https://github.com/joneyao/sage-wiki) | Community | — | Go single-binary LLM Wiki implementation. Tiered compilation (supports 100k+ docs), hybrid search (FTS + vector + ontology graph), 17 MCP tools. Cuts token usage by up to 95% vs naive RAG. |
| [Walnut](https://github.com/wimham/walnut) | Community | — | Local-first AI knowledge management Agent. BYOK mode (OpenAI/Claude/any provider). Auto-organizes docs and proactively recommends related notes during writing. |
| [WikiLoop](https://github.com/jasen215/wikiloop) | — | — | Local-first knowledge search engine for agents. Distills raw docs into structured Markdown wiki; search and read via MCP. |
| [Mintlify](https://mintlify.com) | Mintlify | — | Auto-generates developer docs/wiki from code. AI-driven documentation platform. |
