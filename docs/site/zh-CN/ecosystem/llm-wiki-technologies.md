# LLM Wiki 关键技术

LLM Wiki（知识编译范式）背后的核心技术——知识在写入时预编译，查询时直接读取。

```
原始文档 → LLM 提炼 → Wiki 页面 → [查询时] → FTS / 图遍历 → 读取编译页面 → 答案
```

> 由 Andrej Karpathy 提出：将 LLM 用作编译器，把原始材料转化为持久化、结构化、可审计的知识层。

## 1. 知识提炼 / 编译

| 技术 | 说明 |
|---|---|
| LLM 作为编译器 | LLM 读取原始文档，写入结构化 Wiki 页面——查询时直接读取编译产物。 |
| 提炼管道 | 原始文档 → LLM 提取关键主张、实体、别名、关联链接 → source-note 页面（WikiLoop 模式）。 |
| 分层编译 | 优先编译高频文档（Tier 0–3）。Sage Wiki：命中 3 次自动升级，90 天不活跃自动降级。 |
| 增量更新 | 仅处理变更文档，避免全量重建成本。 |
| [STORM](https://github.com/stanford-oval/storm)（Stanford OVAL） | 多视角 Wiki 生成 Agent：模拟不同视角提问 → 检索 → 生成层级大纲 → 撰写带引用的完整词条。 |
| Co-STORM | STORM 的协作变体，研究过程中构建动态知识图谱指导编译方向。 |
| Agentic 编译循环 | Agent 循环：检索 → 草稿 → 评估 → 补充检索 → 再生成，直到知识稳定。 |

## 2. 标准与格式

| 标准 | 说明 |
|---|---|
| [OKF v0.1](https://github.com/GoogleCloudPlatform/knowledge-catalog/tree/main/okf)（Open Knowledge Format，开放知识格式） | Google Cloud 对 "LLM Wiki" 想法的工程化规格。知识库是包含 YAML frontmatter 的 Markdown 文件目录，强调知识的结构化、可携带性和工具无关性。WikiLoop 知识库与 OKF v0.1 兼容。 |
| YAML frontmatter Schema | 知识单元的"身份证"：`type`（sop / metric / template / case / decision / risk / glossary）、`title`、`description`、`source`、`tags`、`updated_at`、`status`（active / outdated / draft）。 |
| 知识单元 + 关联 | 每个知识单元是独立的 Markdown 文件；关联关系用 Markdown 链接表达而非文件夹层级——形成供 Agent 导航的知识地图。 |

## 3. 知识表示

| 技术 | 说明 |
|---|---|
| 结构化 Markdown | 所有知识以纯文本 Markdown 存储——人类可读，可 git diff，可审计。 |
| Source-note 页面 | 每个原始文档对应一个提炼笔记，包含 `key_claims`、实体标注 `【实体\|类型】`、`related_to` / `supports` / `contradicts` 链接。 |
| Concept 页面 | 对某一概念或方法论的跨文档综合描述。 |
| Comparison 页面 | 方案、工具或范式的横向对比。 |
| Decision 页面 | 技术决策记录（ADR 格式）：背景 + 选项 + 决策 + 影响。 |
| 本体图谱（Ontology Graph） | 编译过程中构建的类型化实体-关系图谱。Sage Wiki 内置 8 种关系类型（`implements`、`contradicts`、`trades_off` 等）。 |
| Schema / 模板 | 指导 LLM 编译风格的写作规则和页面模板，每个 KB 可自定义。 |

## 3. 索引与搜索

| 技术 | 说明 |
|---|---|
| SQLite FTS5 + BM25 | 核心搜索引擎——无需向量模型，亚毫秒级全文检索。WikiLoop、Sage Wiki、TreeSearch 均采用。 |
| 别名扩展 | 关键术语索引时内嵌别名和跨语言等价词，最大化 FTS 召回率。 |
| 图遍历 | `related_to` 链接实现多跳导航（类似 Wiki 页面链接）。搜索结果进行 BFS 扩展。 |
| 混合检索（FTS + 向量 + 图） | Sage Wiki：FTS5（411µs）+ 向量（81ms）+ 本体图谱（1µs），通过 RRF 融合。 |
| 章节树索引 | 保留文档 H1/H2/H3 层级——平铺切块的结构保留替代方案。 |

## 4. 知识质量与维护

| 技术 | 说明 |
|---|---|
| Git 版本控制 | 所有 Wiki 页面在 Git 中——完整历史、diff、blame，知识变更完全可审计。 |
| Lint / 健康检查 | 验证 frontmatter、断开的 source 链接、缺失引用。`wikiloop lint`。 |
| 冲突检测 | `contradicts` 链接将来源间的分歧暴露出来，供人工审核。 |
| Draft 暂存 | 来源少于 2 个的页面隔离到 `_draft/`——验证补充后才进入索引。 |
| 知识空白分析 | `wikiloop synthesize --gaps` 识别覆盖不足的主题。 |
| 实体去重 / 解析 | 识别同一概念的不同表述并合并为单一节点。 |

## 5. Agent 接口（MCP）

| 技术 | 说明 |
|---|---|
| [MCP 协议](https://modelcontextprotocol.io) | Model Context Protocol——Anthropic 开放标准，向 AI Agent 暴露工具/资源。支持 stdio + HTTP 双传输。 |
| `kb_search` | FTS 关键词搜索，返回带 `related` 链接的排序结果，用于图谱导航。 |
| `kb_page` | 通过 ID 获取完整页面内容，支持批量（最多 5 个 ID）或 `full=true` 获取不截断文本。 |
| `kb_add` | 向知识库添加文本文档，写入 `raw/<filename>`，触发增量索引，后台异步运行提炼。 |
| MCP Resources | 只读资源（URI 形式）：Wiki 页面、图谱 Schema、原始文档。 |
| 迭代搜索模式 | Agent 从不同角度发出多次查询，跟随 `related` 链接，自行综合答案。 |
| Sage Wiki MCP 工具 | 17 个工具：6 读、9 写、2 复合——Agent 可直接写入和编译知识。 |

## 6. 文件转换（输入层）

| 技术 | 说明 |
|---|---|
| markitdown（Microsoft） | 将 PDF、Word、Excel、PPT、HTML 转为 Markdown 后再提炼。 |
| `raw/converted/` 模式 | Agent 提取的内容直接写入此目录——跳过转换，直接进入提炼流程。 |
| 文件监听器（Watcher） | 自动检测 `raw/` 下的新文件/变更文件，触发转换 → 提炼 → 索引全流程。 |

---

另见：[RAG 关键技术](/zh-CN/ecosystem/rag-technologies) · [范式对比](/zh-CN/ecosystem/paradigm-comparison)
