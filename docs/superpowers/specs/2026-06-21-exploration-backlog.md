# WikiLoop 技术探索待验证清单

记录从文章分析、评估实验中发现的有价值方向，逐一验证后更新状态。

**状态说明：** 🔲 待验证 | ✅ 已验证有效 | ❌ 验证无效 | ⏳ 进行中

---

## 🎯 优先测试方向（按价值×可行性排序）

| 优先级 | 方向 | 预期效果 | 改动量 |
|---|---|---|---|
| ~~P0~~ | ~~15.1 synonyms 字段~~（蒸馏时扩词） | ✅ 已完成（SYNONYMS RULE，2026-06-24） | — |
| ~~P0~~ | ~~7.1 多索引并行 BM25 + RRF~~ | ✅ 已完成（multiKindFTS + SearchLayered 已实现） | — |
| ~~P1~~ | ~~15.2 Query Expansion~~（查询时扩词） | ✅ 已完成（serverInstructions 强制扩词指引，Agent 自发中英文双查询，2026-06-24） | — |
| ~~P1~~ | ~~15.3 验证：关掉向量，只用 FTS+扩词~~ | ✅ 已完成（向量模块已删除） | — |
| ~~P1~~ | ~~6.2 contentHash 增量 embed~~ | ❌ 已关闭（embed 模块已删除） | — |
| ~~P2~~ | ~~1.1 Cross-Encoder Reranker~~ | ❌ 已关闭（向量模块已删除） | — |
| ~~P2~~ | ~~10.0 goformer 双模式~~ | ❌ 已关闭（向量模块已删除，Windows 通过纯 Go 构建支持） | — |
| **P2** | **12.1 嵌入 Excel 提取** | 企业文档完整性 | 小 |
| **P2** | **16.1 LiteParse 替代 markitdown** | 表格解析准确率 | 小（替换工具）|
| ~~P2~~ | ~~2.5 元数据质量优先~~ | ✅ 已完成（timestamp 强制提取 + tags 领域分类约束，2026-06-24） | — |
| **P3** | **17.1 IdeaBlock 语义分块** | chunk 质量提升 | 中 |
| **P3** | **17.2 gotreesitter 纯 Go 代码解析** | 代码文档切分 | 中（仅代码场景）|
| ~~P2~~ | ~~18.1 LEANN 向量存储压缩~~ | ❌ 已关闭（向量模块已删除） | — |
| ~~P2~~ | ~~19.1 SAG SQL 动态多跳检索~~ | ✅ 已完成（key_claims 实体 claim-only 关联，写入 document_tags，TagExpand hop1 填充 related 字段，2026-06-25）| — |
| **P3** | **19.2 SeedER 多跳图扩展** | 多跳问答召回提升 | 大 |
| **P2** | **32.1 查询反哺 Wiki（Query 回流）** | 对话洞察沉淀回 raw/，知识自动增长 | 短期零代码；长期小 |
| **P2** | **32.2 红链知识缺口信号** | lint 输出被引用但未创建的概念缺口清单 | 小（lint.go）|
| **P3** | **32.3 用户反馈驱动 related 权重** | EvoRAG 思路，读取日志提升被读文档权重 | 中（依赖 32.1）|
| **P3** | **32.5 时序意图检索 boost** | 搜索时感知"最新/当前"等时序词，新文档加权 | 小（与 25.1 合并）|
| **P3** | **32.6 EvoRAG 反馈权重优化** | 待 query_log 完善后再评估；当前 Agent 行为信号不可靠 | 前置：完善 query_log（小）|
| **P3** | **13.1 查询反哺 Wiki** | 知识自动增长 | 中 |
| ~~P2~~ | ~~22.5 Synthesized 聚类合并（Draft 门槛）~~ | ✅ 已完成（draftThreshold=2，_draft/ 机制，2026-06-24） | — |
| ~~P1~~ | ~~23.7 MCP 接口重设计：kb_search + kb_page~~ | ✅ 已完成（kb_page 新接口 + SearchLayered 分层返回，2026-06-24） | — |
| **P3** | **3.1 PageIndex 树搜索** | 结构化文档精准搜索 | 大 |
| **P4** | **27.1 鸿蒙 PC 平台支持** | 原生运行于 HarmonyOS NEXT PC | 大（等待 Go 官方支持）|
| ~~P2~~ | ~~28.1 kb_ingest MCP 工具~~ | ✅ 已完成（kb_add + stdio 子命令，2026-06-25） | — |
| ~~P2~~ | ~~29.1 蒸馏队列（持久化 + 并发 + 重试）~~ | ✅ 已完成（SQLite 队列 + 3 worker goroutine + 指数退避，2026-06-25） | — |
| ~~P2~~ | ~~25.1 知识衰退字段（review_after）~~ | 降级为 P3，现阶段价值有限（见 §25.1 分析） | — |
| ~~P2~~ | ~~25.2 MCP 工具分档（agent/admin）~~ | ✅ 已完成（MCP 只暴露 kb_search/kb_page/kb_add，admin 工具只在 WebUI/CLI）| — |
| **P2** | **26.1 WITH RECURSIVE 多跳图查询** | Agent 能追踪 2-3 跳外的关联文档（⚠️ 依赖 19.1 先完成）| 小（改 graph.go GraphExpand） |
| **P2** | **26.2 预训练词向量 bake-in 轻量向量** | 零依赖重引入向量能力，30MB 内 | 中（embed + SQLite 自定义函数） |
| **P3** | **25.3 normalized_hash 语义去重** | 防止 FTS 索引膨胀 | 小（升级 content_hash 语义） |
| ~~P2~~ | ~~31.1 kb 业务逻辑统一层（kbsvc）~~ | ✅ 已完成（service.go + KBError + 薄包装，2026-06-25，PR #14）| — |

---

---

## 一、检索质量提升

### 1.1 Cross-Encoder Reranker（ONNX）
**状态：** ❌ 已关闭（向量模块已删除，不再适用）
**来源：** 第6篇（Marco）、第4篇（小小寰宇）
**预期效果：** CP 从 0.186 → 0.35+（基于 Cross-Encoder vs Bi-Encoder 的实验数据：Top-1 准确率从 45% → 78%）
**实现思路：**
- 模型：`BAAI/bge-reranker-v2-m3`（中文优化，ONNX 格式，~280MB）或 `ms-marco-MiniLM-L-6-v2`（英文，~80MB）
- 放置：`<WIKILOOP_KB>/models/rerank/`，复用现有 ONNX Runtime
- 调用点：`biEncoderRerank` 替换为 Cross-Encoder 打分
**触发条件：** Phase 1 Bi-Encoder Rerank 评估后 CP < 0.25 时升级

---

### 1.2 父子索引（Parent-Child Chunking）
**状态：** 🔲 待验证（已分析认为对当前场景价值有限）
**来源：** 第4篇（灵机遣记）
**预期效果：** 解决"检索精准但上下文不完整"的问题，CR 可能提升 5-10%
**实现思路：**
- 子 chunk：当前分块（section 级，用于向量检索）
- 父 chunk：整篇文档（用于返回给 LLM 的上下文）
- 命中子 chunk → 取父文档完整内容作为 Snippet 返回
**注意：** Claude Code 可以直接 Read 文件，对纯 MCP 客户端价值更大

---

### 1.3 置信度判断（Confidence Gating）
**状态：** 🔲 待验证
**来源：** 第4篇（小小寰宇），客服场景阈值 0.4-0.5
**预期效果：** 减少低质量回答，提升 Faithfulness（目前 0.710）
**实现思路：**
- `BuildContext` 检查返回的 `HybridScore` 均值
- 低于阈值（0.02）时在返回结果中加入 `"confidence": "low"` 标记
- MCP instructions 中指示 Agent：confidence=low 时明确告知用户"知识库信息不足"
**改动量：** 小（仅改 context.go 和 server instructions）

---

### 1.4 Query Rewriting / Expansion
**状态：** 🔲 待验证
**来源：** 第3篇（片涵沫 Advanced RAG）、第6篇（Marco）
**预期效果：** 对模糊查询，CR 可能提升 10-15%
**实现思路：**
- `kb_context` 接到问题后，先用 LLM 改写为"检索友好"形式再搜索
- 或生成多个同义查询，MultiQuery 合并结果
**代价：** 每次 context 调用多一次 LLM 调用，延迟增加 2-5 秒

---

### 1.5 MMR 参数优化（多样性约束）
**状态：** 🔲 待验证（当前 maxPerDoc = limit/5 效果不佳）
**来源：** 第4篇（小小寰宇），MMR α=0.7-0.9
**预期效果：** 在保持精度的同时提升结果多样性
**实现思路：**
- 调整 `VecSearch` 的 `maxPerDoc` 计算方式
- 或在 `biEncoderRerank` 阶段加入 MMR 重排（相关性 × α - 已选集合相似度 × (1-α)）

---

## 十八、向量存储优化

### 18.1 LEANN 向量存储压缩（97% 存储降低）★★★
**状态：** ❌ 已关闭（向量模块已删除，不再适用）
**来源：** UC Berkeley Sky Computing Lab，九秋拾序文章，开源项目
**核心技术：** 按需重算（on-demand recomputation）+ HNSW 图剪枝
- 不存完整向量，检索时从原始文本临时重算嵌入
- 只保留 HNSW 图的关键边，砍掉冗余邻接表
- 效果：201GB → 6GB（97% 压缩），精度不降
- 实测：78万封邮件仅占 78MB，3.8万条浏览器记录仅 6MB
**对 WikiLoop 的价值：** 当前 chromem-go 需要 1-2GB 内存加载 469 篇 source-notes，LEANN 可降至几十 MB，彻底解决内存问题，同时保留向量语义搜索能力
**研究结论（2026-06-22）：**
- 仅有 Python + C++ 核心，无 Go 接口，集成需要 Python sidecar，违背 WikiLoop 零依赖原则
- 115 Stars，非常早期，生产稳定性存疑
- 压缩优势在百万级文档才显著，WikiLoop 469 篇规模收益有限
- 更适合 WikiLoop 的内存优化：等待 LLM 扩词方案验证后直接删除向量（15.3）
**项目：** github.com/skypilot-org/leann

---

## 十九、图检索增强

### 19.1 SAG：SQL 动态超边多跳检索★★★
**状态：** ✅ 已完成（2026-06-25）
**来源：** 袋鼠帝文章，Zleap-AI 开源，GitHub 1.3K Stars，刚开源
**项目：** https://github.com/Zleap-AI/SAG
**论文：** SQL-Retrieval Augmented Generation with Query-Time Dynamic Hyperedges（arxiv.org/abs/2606.15971）
**Benchmark：** HotpotQA/2WikiMultiHop/MuSiQue 三个多跳基准，9 项 Recall@K 赢 8 项，平均 Recall@2 提升 **11.16%**；已在 5 亿级数据验证，线上延迟秒级
**核心思路：**
- 入库时：LLM 把文档片段总结为一句话 **event**，再提取 **entity**（event 负责语义，entity 负责 SQL 索引）
- 查询时：SQL 动态 JOIN 把共享同一 entity 的 event 串联，实时组装多跳路径
- 同时跑向量语义搜索，两路结果合并
**解决问题：** 传统 RAG/向量搜索的多跳断链问题。例如"A收购B，张三是A的CTO，张三后来去哪了"——三个文档无字面关联，普通检索必然断链，SAG 通过实体 JOIN 串联
**对比 GraphRAG：** GraphRAG 离线预构建图谱（成本极高，新文档要重建）；SAG 查询时动态构建，成本低，维护简单

**WikiLoop 实现进展（2026-06-25）：**

已实现 `document_tags` 表 + `TagExpand` SQL JOIN，但实验过程中发现关键设计决策：

**目标区分：**
- **问题 1（扩词召回）**：搜 "Karpathy" 能找到 LLM Wiki 文章 → 已通过 SYNONYMS RULE + Agent 扩词解决 ✅
- **问题 2（related 导航）**：读完一篇文章，related 字段推荐相关文章，减少 Agent 额外搜索 → 当前在解决

**三种 tag 来源的实验结论：**

| source | 来源 | 数量 | 精度 |
|--------|------|------|------|
| `tag` | frontmatter tags（LLM 概括主题词） | 13288 条 | 低（RAG/AI 等泛词造成噪音）|
| `entity` | 正文全文 `【name\|type】` 标注 | 4249 条 | 低（Marp/Qmd 等弱相关词混入）|
| `claim` | **key_claims 段落**里的实体标注 | ~3194 条（待写入）| **高（LLM 认为核心论点里的实体）**|

**hop2 实验结论：** 无论 5% 还是 2% IDF 阈值，hop2 结果全为噪音（碳合规文章出现在 Karpathy related 里）。根本原因：tag 多跳是无语义的共现关系，跳 2 次等于随机游走。hop1 + claim 实体才是正确方向。

**已完成改动（2026-06-25）：** `upsertDocumentTags` 改为只从 `key_claims` 提取实体写入 source='claim'，完全放弃 source='tag' 和 source='entity'。写入量从 17537 降至 2206 条，hop1 related 质量从"碳合规、0成本上线"变成"AI知识库分叉、Hermes Agent"。

**IDF 过滤现状：** graph.go 里保留了 `idfFilter`（5% 阈值），但在 claim-only 模式下 **实际不起作用**——最高频 claim 实体（OpenClaw）只出现在 23 篇，远低于 5% 阈值（104 篇）。保留代码作为防护，等知识库增长到 2000+ 篇 claim 实体时才会生效。

**WebUI 与 MCP 不一致：** WebUI `/api/search` 用 `kb.Search()`（无 related），MCP `kb_search` 用 `kb.SearchLayered()`（有 related + TagExpand）。需要统一为 `SearchLayered`——见待办。

**后续：** 19.1 完成后，`related_to` 链接密度大幅提升，**26.1 WITH RECURSIVE 多跳图查询**的价值才能真正体现——建议先做 19.1，再做 26.1。

### 19.2 SeedER 多跳图扩展★★
**状态：** 🔲 长期探索
**来源：** PaperRAG 文章，知识图谱检索论文
**核心思路：** 知识图谱里答案和问题文本"不像"但在关系路径上。传统 dense retrieval 需要 Ω(|V|) 维度才能覆盖多跳组合，SeedER 改为局部迭代扩展：从锚点出发，学"每步沿哪类边走"，只需 O(log|V|) 表示规模
**对 WikiLoop 的价值：** 当前 `GraphExpand` 只做1层邻居扩展，答案在2-3跳之外时会漏掉
**与 19.1 的关系：** SAG（SQL JOIN）是更务实的实现，SeedER 是更学术的方向，先做 SAG

---

## 十七、分块与代码解析

### 17.1 IdeaBlock 语义分块★★★
**状态：** 🔲 待验证
**来源：** Blockify 开源项目（第2篇文章）
**背景：** 传统按字符数/标题切分会拦腰截断语义，IdeaBlock 用 LLM 判断语义边界，把一个"完整想法"作为一个 chunk。解决：内容被切断、重复向量、版本混乱三大问题。
**对 WikiLoop 的应用：**
- distill 时，Key Facts 每条按语义完整单元组织（已有 key_claims 字段，进一步规范）
- `chunkDoc()` 当前按 `##` 标题切分，可增加"语义完整性"判断：如果一个 section 内容跨越多个主题，进一步细分
- synthesize 生成 concept 页时，避免把不同主题强行合并进一个 chunk

---

### 17.2 gotreesitter 纯 Go 代码解析★★★
**状态：** 🔲 待验证
**来源：** polarisxu（第3篇），项目 https://github.com/smacker/go-tree-sitter 或 gotreesitter
**背景：** Tree-sitter 是现代编辑器（Neovim/Helix/Zed/GitHub）的解析引擎，但 Go 绑定需要 CGO。gotreesitter 用 22 万行纯 Go 重写，206 种语言，零 CGO，增量解析比 CGO 绑定快 69 倍。
**对 WikiLoop 的价值：**
- 代码文档（如小米智能制造代码仓库）按函数/类边界切分，而非暴力字符切割
- 实现探索清单 2.1 的"代码文档按函数/类边界切分"，零 CGO 依赖
- 解决了 Windows 兼容问题（无 CGO）
**优先级：** 企业代码仓库入库场景有用，通用文档场景暂不需要

---

## 二、内容质量提升

### 2.1 Chunk 策略：语义分块优化
**状态：** 🔲 待验证
**来源：** 第11篇（李源炳），Markdown 按标题层级切分
**预期效果：** 减少语义破坏，改善 chunk 质量
**实现思路：**
- 当前已按 `##` 切分；考虑同时支持 `#` 和 `###` 层级
- 短文档（<1000字）不分块（已实现）
- 对代码文档按函数/类边界切分

---

### 2.5 元数据质量优先★★
**状态：** ✅ 已完成（timestamp 强制提取 + tags 领域分类约束，2026-06-24）
**来源：** 德国埃森大学 IKIM 论文（第5篇），Agentic RAG 生产环境实测
**核心发现：** 文档元数据（时间、类型、关系）不准确时，RAG 效果大幅下降。**元数据质量 > 检索算法优化**。
**WikiLoop 现状：** source-note 的 `timestamp` 字段很多是默认值（2024-01-01），不准确；`tags` 质量参差不齐。
**改进方向：**
- distill 时强制提取并验证 `timestamp`（从原文日期提取，不用默认值）
- `tags` 要求包含领域分类词（如 `RAG`、`主数据`、`智能制造`），而非随机关键词
- 加 `doc_type` 字段区分：`技术规范`、`分析文章`、`项目文档`、`会议纪要`
- 元数据准确后，搜索时可加 `doc_type` 过滤，精度大幅提升

---

### 2.2 文档时效性管理
**状态：** 🔲 待验证
**来源：** 第4篇（小小寰宇），时间衰减 λ=0.01-0.1
**预期效果：** 避免旧文档误导，Faithfulness 可能提升
**当前状态：** 已实现 30 天线性衰减 recencyBoost（Phase 1），效果待评估
**待验证：** 衰减函数是否应改为指数衰减（`e^(-λ * days)`），λ 最优值

---

### 2.3 Distill 模板质量规范（已做）
**状态：** ✅ 已实现
**内容：** source-note/concept/comparison 模板加入 Key Facts 质量约束，buildSystemPrompt 注入 references

---

### 2.4 Source-note Key Claims 结构化提取
**状态：** 🔲 待验证
**来源：** UnWeaver 实体索引思路
**预期效果：** 提升跨文档多跳问答的召回率（Phase 3）
**实现思路：**
- distill prompt 要求 `key_claims` 字段包含结构化实体描述
- 聚合同名实体，建立实体→文档映射

---

## 三、架构方向

### 3.1 无向量 RAG（文档树 + LLM 导航）
**状态：** 🔲 长期探索
**来源：** 第5篇（Bono保罗）+ PageIndex/VectifyAI（GitHub 33k stars，FinanceBench 98.7%）
**核心思路：** 文档→语义树（递归 JSON，含页码/章节摘要）→ LLM 推理导航定位章节，完全不用向量数据库
**WikiLoop 天然适配：** wiki 层本身就是层级结构（concepts/comparisons/source-notes），index.md 是现成"目录树"，可直接用 LLM 导航
**实现参考：** PageIndex 开源（Python），核心逻辑：`run_pageindex.py --pdf_path` 生成树，Agent 查询时用 `PageIndexClient` 做树搜索
**适用场景：** 结构清晰的领域知识库（财报、技术规范、API 文档），精准问答优于语义搜索
**预期效果：** 对结构清晰文档，CP 可达 0.5+
**代价：** 每次检索多 1-2 次 LLM 调用

---

### 3.2 Agentic RAG（Agent 自主决策检索）
**状态：** 🔲 长期探索
**来源：** 第3篇（片涵沫），第7篇（Jameszyh）
**适用场景：** 复杂多跳问题，Agent 自主决定是否检索、检索几次
**预期效果：** 复杂问题 CR 大幅提升
**代价：** 架构改动大，延迟高

---

### 3.3 混合架构（Vector + Graph）
**状态：** 🔲 中期探索（Phase 3 实体索引的延伸）
**来源：** 第3篇（片涵沫 Graph RAG），HyGRAG（WWW 2026）
**适用场景：** 关系推理型问题（A 和 B 有什么关系？）
**预期效果：** 多跳问题 CR 可达 0.80+

---

## 十三、知识进化与反馈

### 13.1 查询反哺 Wiki（GBrain 启发）★★
**状态：** 🔲 待验证
**来源：** GBrain（Garry Tan，2026-04 开源）+ 第1篇文章分析
**背景：** GBrain 核心机制之一：Agent 回答问题后，高质量答案自动写回知识图谱，实现"知识复利"——越用越聪明。WikiLoop 目前蒸馏只是单向（raw→wiki），没有"查询→wiki"的反向路径。
**方案：**
- `kb_context` 工具加一个可选参数 `record=true`
- Agent 认为答案有价值时，调用 `kb_context(question, record=true)`，触发将答案 append 到相关 source-note 的 `key_claims` 或生成新的 concept 页
- 或：MCP 新增 `kb_record` 工具，专门让 Agent 把好的推理结果写回 wiki
**预期效果：** wiki 层随使用自动增长，不依赖新文档入库
**注意：** 需要防止错误信息污染 wiki，需要置信度门槛

---

## 十四、向量后端扩展

### 14.1 Redis RediSearch 向量后端★
**状态：** ❌ 已关闭（向量模块已删除，不再适用）
**来源：** OhMyGo 项目（第2篇），Go + RediSearch 实现 RAG
**背景：** Redis Stack 内置 RediSearch，支持向量索引（HNSW）+ FTS，无 CGO，纯 HTTP/TCP 协议
**优点：** 无 CGO，Windows 原生，有 ANN 索引，不全量加载内存
**缺点：** 需要外部 Redis 服务（非本地优先），用户需要额外部署
**方案：** 实现 `Embedder` 接口的 `RedisVecStore`，config.yaml 加 `vector.backend: redis`
**适合场景：** 已有 Redis 基础设施的团队/企业用户，KB 规模大（>5000文档）

---

## 十五、LLM 扩词替代向量搜索

> **2026-06-22 架构决策：方向确认，开始实施。向量搜索相关探索（6.2、11.2、10.0 goformer）全部降优先级。**

### 决策依据（2026-06-22）

| 维度 | 向量搜索 | LLM 扩词 |
|---|---|---|
| 语义覆盖 | bge-small-zh 512维小模型 | LLM 语义理解更强 |
| 查询延迟 | +5ms | **0ms**（蒸馏时离线完成）|
| 额外 LLM 调用 | 无 | **无**（Agent 调 MCP 本身就是 LLM）|
| 内存 | 1-2GB | 0 |
| CGO 依赖 | 有（daulet/tokenizers）| 无 |
| Windows 支持 | 需要 goformer 双模式 | 原生支持 |
| 精确编码（M17、表名）| 弱 | 强 |

**向量搜索无胜出维度，LLM 扩词全面占优。**

---

### 15.1 蒸馏时 synonyms 字段（Index-time Expansion）★★★
**状态：** ✅ 已完成（SYNONYMS RULE 加入 prompt，第一条 key_claim 为专用术语索引，2026-06-24）
**核心思路：** 蒸馏时让 LLM 提取同义词/缩写/中英文对照，存入 source-note frontmatter 的 `synonyms` 字段，全部进 FTS5 索引。搜索时用户搜"召回率"自动命中"recall rate"、"CR"、"Context Recall"。
**零运行时代价：** LLM 蒸馏文档时顺便提取，不增加任何额外调用或延迟。
**实现：** `internal/kbinit/schema/templates/source-note.md` 加 synonyms 字段，FTS 索引自动覆盖，distill prompt 新增 synonyms 提取要求。

---

### 15.2 Query Expansion（MCP system prompt 方案）★★★
**状态：** ✅ 已完成（serverInstructions 加入强制扩词指引，实测 Agent 自发中英文多角度查询，2026-06-24）
**关键洞察：** Agent 调用 `kb_context` MCP 工具本身就是 LLM 在思考，无需额外 LLM 调用。
**实现方式（零代码改动）：** 在 MCP server instructions / kb_context tool description 里加入查询扩展指引，Agent 在调用前自动展开查询词。
**蒸馏时 synonyms 是静态扩词；查询时 Agent 自主扩词是动态扩词。两者互补。**

---

### 15.3 废弃向量搜索
**状态：** ✅ 已完成（2026-06-24 向量模块已删除）
**内容：**
- 删除 chromem-go 依赖（-1-2GB 内存）
- 删除 daulet/tokenizers（解决 Windows CGO）
- 删除 internal/embed/ 相关代码
- 6.2 contentHash、11.2 Ollama embedder、10.0 goformer 全部作废

---

## 四、评估体系

### 4.1 Context Relevancy 指标
**状态：** 🔲 待添加
**来源：** 第6篇（Marco），语句级相关性，无需 ground truth
**价值：** 唯一不需要参考答案的指标，可用于生产监控
**实现思路：** 在 `eval_wikiloop.py` 中加入第四个评估指标

---

### 4.2 MRR（Mean Reciprocal Rank）
**状态：** 🔲 待添加
**来源：** 第6篇（Marco），第8篇（老梁agent），RRF Fused 的 MRR=0.68
**价值：** 比 CR 更关注"第一个正确答案在第几位"，补充召回排序质量
**实现思路：** 评估时记录 ground truth 首次出现位置

---

### 4.3 扩大评估题集（10→30 题）
**状态：** 🔲 待做
**原因：** 当前 10 题评估方差较大（LLM 打分波动约 ±0.05），增加到 30 题可降低噪声

---

### 4.4 版本回归测试★★
**状态：** 🔲 待做
**来源：** 第7篇（瑰夏森林的小屋）企业级 RAG 自动化评估体系
**背景：** 每次改动后只跑单次评估，无法知道"新版本有没有把原来答对的问题改坏"（Regression）
**方案：**
- `eval/README.md` 已有历史对比表，但缺少自动化对比
- 评估脚本加 `--compare baseline` 参数，自动对比当前结果与 baseline，输出每题的变化（新通过/新失败）
- 每次 commit 后跑评估，记录 per-question 级别的变化日志
**优先级：** 在题集稳定（4.3 完成）后做

---

### 4.5 Hallucination Rate 单独统计★
**状态：** 🔲 待添加
**来源：** 第4篇（Marco）RAG 生成答案评估
**方案：** 评估脚本加一个指标：答案中有多少声明无法从 context 找到支撑（即幻觉比例）
**实现：** Faithfulness < 0.5 的题目单独统计，报告幻觉率 = 幻觉题数 / 总题数

---

## 六、向量存储与内存优化

### 6.1 SQLite BLOB 替换 chromem-go
**状态：** ❌ 不推荐（2026-06-22 分析）
**结论：** SQLite BLOB 全表扫描 cosine，无 ANN 索引，文档增多后查询性能急剧下降（5000篇约 500ms-2s）。精度无损失，但性能是退步而非进步。
**正确方向：** 控制嵌入文档数量（已实施：只嵌 source-notes 469篇）+ 考虑 lancedb（见 6.3）

---

### 6.3 lancedb 替换 chromem-go
**状态：** ❌ 不推荐（2026-06-22 研究）
**研究结论：**
- Go SDK（`lancedb/lancedb-go`）仅 56 stars，非纯 Go，**强 CGO 依赖**（需预编译 liblancedb_go.a），和 WikiLoop 避免 CGO 的方向相悖
- API 模型基于 Apache Arrow，与 WikiLoop 简洁 VecStore 接口差异大，迁移成本中高
- 469 篇 source-notes 规模下 chromem-go 内存 1-2GB 可控（已限制只嵌入 source-notes）
- ANN 索引在小规模（<1000文档）优势不明显

**当前最优策略：** 继续 chromem-go + 控制嵌入文档范围（只嵌 source-notes），内存可控

**备选关注：**
- `smhanov/syzgydb`：纯 Go 磁盘型向量库，但 11 stars 过低，不成熟
- 若 KB 增长到 5000+ 篇 source-notes，再重新评估 Qdrant（外部服务）或 sqlite-vec（CGO 但成熟）

---

### 6.2 contentHash 增量 embed（跳过未变更文档）★★
**状态：** ❌ 已关闭（embed 模块已删除）
**来源：** CodeGraph、GitNexus 的增量索引设计
**问题：** 每次 `embed --full` 重建所有向量，1882 篇文档需要几十分钟
**方案：** documents 表已有 `content_hash` 字段，embed 时对比 hash，内容未变则跳过嵌入计算
**预期效果：** 增量 embed 从几十分钟降到几秒（只处理变更文档）
**改动范围：** `internal/kb/vector.go` 的 `EmbedDocuments`（已有 incremental 模式，需确认 hash 比对逻辑）

---

## 七、检索质量提升（补充）

### 7.1 多索引并行 BM25 + RRF 融合★★★
**状态：** ✅ 已完成（multiKindFTS + SearchLayered，2026-06-24）
**来源：** GitNexus（多 FTS 索引并行）、codebase-memory-mcp（min-cosine 策略）
**问题：** 当前 FTS 和向量是串行的，wiki_pages/source-notes/concepts 混在一起检索
**方案：**
- wiki 层按 kind 分索引：source-notes、concept、comparison、decision 各自建 FTS5
- 并行查询四个索引，每个取 top-N，RRF 合并
- 向量搜索并行运行，最终 RRF 三路融合
**预期效果：** CR 从 0.567 提升到 0.70+（GitNexus 实测 RRF 多索引融合显著优于单索引）

---

### 7.2 links 表加 confidence 分数★
**状态：** 🔲 待验证
**来源：** GitNexus 的 `CodeRelation` 表设计（type + confidence + reason）
**问题：** 当前 links 表只有 relation 类型，没有置信度，GraphBoost 无法区分强弱关联
**方案：** links 表加 `confidence REAL DEFAULT 1.0` 列，蒸馏时根据 LLM 的确定性填入
**预期效果：** GraphBoost 可以加权，减少低置信度边引入的噪声

---

## 八、LLM 调用质量

### 8.1 tiktoken-go 控制 distill/synthesize prompt 长度★
**状态：** 🔲 待验证
**来源：** https://github.com/pkoukk/tiktoken-go（纯 Go，无 CGO，~2k star，更成熟）或 https://github.com/tiktoken-go/tokenizer（同类备选）
**背景：** distill 和 synthesize 把大量文档内容拼进 prompt，没有 token 计数，容易触发 context-too-long 或 429。tiktoken 是 OpenAI BPE tokenizer 的纯 Go 实现，可在调用 LLM 前预估 token 数。
**注意：** tiktoken 只适用于 GPT 系列模型的 token 计数（BPE），不能替代 daulet/tokenizers（bge-small-zh embedding 用的 WordPiece）
**用途：**
- `internal/distill/distill.go`：拼 prompt 前计算 token 数，超限时自动截断 rawContent
- `internal/synthesize/generate.go`：拼多篇 source-notes 时按 token 预算分批，避免单次超限
- `eval/eval_wikiloop.py`：评估脚本 LLM 调用前预估 token，动态调整 max_tokens
**改动量：** 小（只在拼 prompt 处加 token 计数判断）
**依赖：** `github.com/pkoukk/tiktoken-go`（纯 Go，无 CGO，更成熟，~2k star）

---

## 十六、文档解析增强

### 16.1 LiteParse 替代/补充 markitdown★★
**状态：** 🔲 待验证
**来源：** wsleepybear 文章（第5篇），GitHub 7300+ stars，v2.0.3，活跃维护
**背景：** markitdown 遇到复杂排版（多列、嵌入 Excel、表格）经常翻车；LiteParse 提供精确文本位置（bounding box），表格结构保留更完整
**优势：**
- 保留表格结构，不会把嵌入 Excel 转成图片占位符
- 提供 bounding box 信息（文字在页面的位置），有助于 Agent 理解表格/图表
- 支持 PDF + Office 格式，比 markitdown 更精确
**对 WikiLoop 的价值：** 直接解决 M27 字段属性表格无法提取的问题，对企业内部技术文档尤其有价值
**项目地址：** https://github.com/VikParuchuri/marker（LiteParse 的上游）或搜 LiteParse
**评估方式：** 用同一批 docx 文件分别用 markitdown 和 LiteParse 转换，对比表格提取质量

---

## 十二、文档转换增强

### 12.1 Office 文档转换增强（嵌入 Excel 提取 + 旧格式支持）★★
**状态：** ✅ 已验证可行（2026-06-22）

#### 问题背景
飞书文档导出 docx 时，内嵌电子表格变成 OLE Object（图片），markitdown/pandoc 均无法识别，只显示为图片占位符 `![](data:image/png;base64...)`。

#### 嵌入 Excel 提取方案（docx/pptx）
docx/pptx 本质是 zip，嵌入 Excel 存在 `word/embeddings/Microsoft_Excel_WorksheetX.xlsx`，可直接提取后用 markitdown 转为 Markdown 表格，替换原文中图片占位符：

```python
import zipfile, subprocess, tempfile, os

def extract_embedded_xlsx(docx_path, md_path):
    with zipfile.ZipFile(docx_path) as z:
        for name in z.namelist():
            if 'embeddings' in name and '.xlsx' in name:
                data = z.read(name)
                with tempfile.NamedTemporaryFile(suffix='.xlsx', delete=False) as f:
                    f.write(data)
                    tmp = f.name
                result = subprocess.run(['markitdown', tmp], capture_output=True, text=True)
                excel_md = result.stdout.strip()
                os.unlink(tmp)
                # 替换 md 中的图片占位符
                with open(md_path, 'r') as f:
                    content = f.read()
                # 清理 NaN（空单元格）和转义符
                excel_md = excel_md.replace('| NaN ', '|  ').replace('\\_', '_')
                # 插入表格内容（替换第一个图片占位符）
                content = content.replace('![](data:image/png;base64...)', excel_md, 1)
                with open(md_path, 'w') as f:
                    f.write(content)
```

**清理规则：**
- NaN → 空：`sed 's/| NaN /|  /g'`
- 转义符：`sed 's/\\\_/_/g'`

#### 各格式支持情况

| 格式 | markitdown 支持 | 嵌入对象提取 | 备注 |
|---|---|---|---|
| docx | ✅ | ✅ 已验证 | 嵌入 xlsx 可提取 |
| xlsx / xls | ✅ | — | 直接转表格 |
| pptx | ✅ | ⚠️ 待测试 | 同为 zip，预计可用 |
| pdf | ✅ | ❌ | 图片表格无法还原 |
| doc / ppt / xls（旧版二进制）| ❌ | ❌ | 需先转新格式 |
| wps / et / dps | ❌ | ❌ | 需先转新格式 |

#### 旧格式预处理（用户手动执行）
旧版二进制格式（.doc/.ppt/.xls）和 WPS 格式需先用 LibreOffice 转换为 Office Open XML：

```bash
# 批量转换旧版 Word/PowerPoint/Excel
libreoffice --headless --convert-to docx *.doc
libreoffice --headless --convert-to pptx *.ppt
libreoffice --headless --convert-to xlsx *.xls

# WPS 格式
libreoffice --headless --convert-to docx *.wps
libreoffice --headless --convert-to xlsx *.et
libreoffice --headless --convert-to pptx *.dps
```
转换后放入 `raw/` 目录，WikiLoop 的 convert 流程会自动处理。

**改动范围：** `internal/convert/` 新增 docx/pptx 后处理步骤（嵌入对象提取）

---

## 九、架构验证与参考

### 9.0 Go RAG 实现参考（知乎文章验证）
**状态：** ✅ 已验证方向正确
**来源：** 知乎文章《RAG的教程还是Python的丰富呀，咱们也想办法给Go生态做做贡献吧》（2025-12-10）
**结论：** 外部独立实现与 WikiLoop 架构高度吻合，验证了我们的技术路线正确
**对 WikiLoop 的具体启示：**
1. **ONNX + daulet/tokenizers 是目前 Go 生态最成熟的嵌入方案**——和 WikiLoop 一致，无需改变
2. **MockONNXEmbedding 降级策略**：models/ 目录无模型时自动降级到 FTS-only 模式，而非报错退出（WikiLoop preflightCheck 已有类似逻辑，可加强）
3. **混合检索降级**：`混合搜索未找到 → 回退到纯向量搜索`，和 WikiLoop 的 compressResults fallback 思路一致
4. **中文 n-gram 分词** 和 WikiLoop FTS5 trigram tokenizer 互相印证

---

## 十一、嵌入加速与替代后端

### 11.1 ONNX CoreML Execution Provider（已验证，❌ 不适用）
**状态：** ❌ 已验证无效（2026-06-22 实测）
**结论：** CoreML EP 对 bge-small-zh（24M参数小模型）反而更慢
- CPU only：469篇 1分20秒（~10ms/条）
- CoreML EP：469篇 4分39秒（~29ms/条），慢 3.5x
- 原因：小模型的 CPU→GPU 数据传输开销 + 算子编译开销 > 计算节省
- CoreML 适合大模型（>100M参数）或大批量并行推理，不适合当前场景

---

### 11.2 Ollama Embedder 后端（解决 Windows CGO）★★
**状态：** 🔲 待验证
**来源：** Ollama HTTP API + github.com/richardwooding/ollamaembed（零依赖）
**背景：** Ollama 官方支持 bge-m3（中文），Go 调用无 CGO 依赖
**方案：** 实现 `Embedder` 接口的 `OllamaEmbedder`，通过 HTTP 调用本地 Ollama 服务
**集成方式：** config.yaml 加 `embedding.backend: ollama`，`makeEmbedder()` 根据配置选择后端
**优缺点：**
- ✅ 零 CGO，Windows 原生，50 行代码
- ❌ 需要用户预先启动 `ollama serve`（外部依赖）
- bge-small-zh 需 Modelfile 导入；官方 bge-m3 更好但 568MB 较大
**适合场景：** 已使用 Ollama 的用户作为可选后端

---

## 十、平台兼容性

### 10.0 goformer 双模式方案（已验证，待实现）★★★
**状态：** ❌ 已关闭（向量模块已删除，Windows 通过纯 Go 构建支持）
**来源：** https://github.com/MichaelAyles/goformer（MIT，纯 Go）

**验证结论：**
- goformer + bge-small-zh（512维）完全兼容，单条/批量一致性 cosine=1.000000
- bge-small-zh 实际就是 512 维（meta.json 和 ONNX 输出均为 512 维）
- 实测速度：单条 242ms，469 篇全量 embed 约 2 分钟（ONNX 约 2 秒，慢 50x）
- 速度差异本质：纯 Go 矩阵乘法 vs C++ SIMD（AVX2/NEON），无法短期优化

**决策：双模式平衡方案**
- **默认模式（-tags fts5）**：保留 ONNX + daulet/tokenizers，快（~5ms/条），需要 CGO，macOS/Linux
- **纯 Go 模式（-tags fts5,goformer）**：goformer，慢（~242ms/条），零 CGO，Windows 原生支持
- embed 是离线操作，2 分钟可接受；检索速度不受影响（向量已预存）

**实现方案：**
- `internal/embed/onnx.go` 加 `//go:build fts5 && !goformer` 保留 ONNX 路径
- 新建 `internal/embed/goformer.go`（`//go:build fts5 && goformer`）实现 goformer 路径
- `internal/embed/cgo_link.go` 加 `//go:build fts5 && !goformer`
- `scripts/build.sh` windows 目标改为 `CGO_ENABLED=0 go build -tags fts5,goformer`
- 模型目录需要 model.safetensors（可由 pytorch_model.bin 一键转换）

**模型文件说明：**
- bge-small-zh 需要 safetensors 格式：`python convert_to_safetensors.py` 从 pytorch_model.bin 转换
- bge-small-zh-v1.5 已有官方 safetensors，可直接使用（512维，质量更好）
**来源：** https://github.com/knights-analytics/hugot
**背景：** WikiLoop Windows 构建失败根因是 daulet/tokenizers（Rust CGO），hugot 提供了纯 Go tokenizer + ONNX 推理的完整方案
**方案：**
- 用 hugot 的 `featureExtraction` pipeline 替换 `daulet/tokenizers` + `yalue/onnxruntime_go`
- hugot 内部用纯 Go tokenizer，只依赖 onnxruntime（有官方 Windows 预编译包）
- bge-small-zh 本身已是 ONNX 格式，**直接复用**，无需重新导出
**改动范围：**
- `internal/embed/onnx.go`：换成 hugot featureExtraction pipeline
- `internal/embed/cgo_link.go`：删除（不再需要 libtokenizers.a）
- `go.mod`：删 `daulet/tokenizers`，加 `knights-analytics/hugot`
**预期效果：** Windows 构建成功，CGO 依赖从2个降为1个（只剩 onnxruntime）
**风险：** hugot 还处于早期阶段，需验证与 bge-small-zh 的兼容性；tokenize 结果需与现有保持一致

### 9.1 modernc.org/sqlite 替换 go-sqlite3
**状态：** 🔲 待验证（深度分析完成，见下）
**来源：** CodeGraph 项目分析 + 深度代码分析（2026-06-22）
**结论：** 可行，低风险，但 CGO 无法完全消除

**关键发现：**
- 只需改 `internal/kb/db.go` 的两行：import 和 driver name（`"sqlite3"` → `"sqlite"`）
- WikiLoop 只用标准 `database/sql` 接口，无 go-sqlite3 私有 API
- FTS5 trigram tokenizer 在 modernc v1.25+ 已支持
- **CGO 仍然必须**：`daulet/tokenizers`（Rust libtokenizers.a）仍需 CGO_ENABLED=1
- Windows 构建失败根因是 tokenizers，不是 sqlite，替换后 Windows 问题仍存在
- 性能影响：FTS5 查询慢 10-20%（可接受）
- 二进制体积基本持平

**值得做的理由：** 降低构建复杂度（不再需要 C 编译器编译 sqlite3.c）
**改动范围：** 仅 `internal/kb/db.go`（2行）+ `go.mod`（换依赖）

---

## 五、已验证结论

| 日期 | 方向 | 结论 |
|---|---|---|
| 2026-06-20 | Bi-Encoder Rerank | CP +5.2%，CR 未达预期（0.320 < 0.40），Bi-Encoder 提升有限 |
| 2026-06-20 | 无分块（整文档嵌入） | CR 0.380 > 分块的 0.320，但 CP 下降。分块有助精度但伤召回 |
| 2026-06-20 | MMR 去重约束 | CR 0.300，低于 baseline，当前参数不合适 |
| 2026-06-20 | Phase 2 synthesize（12→18 篇） | CP 0.282（+13.2%），CR 0.390，wiki 覆盖度是根本瓶颈 |
| 2026-06-21 | kb_context limit 5→10 | 待评估（基于第6篇数据预期 CR +14%） |

---

## 二十、竞品参考与借鉴

### 20.1 WeKnora（腾讯开源）★★★
**状态：** ✅ 已研究（2026-06-22）
**项目：** https://github.com/Tencent/WeKnora，Go，16.7K Stars，极活跃，v0.6.2
**定位：** 企业级全栈 RAG 平台，多租户 + Web UI + IM 集成 + Cloud 托管
**与 WikiLoop 关系：** 互补而非竞争。WeKnora 是企业级平台，WikiLoop 是本地优先极简 MCP 原生知识层

**核心架构亮点：**
- **Wiki Mode**：Agent 自动从 raw 文档生成互链 Markdown Wiki + 知识图谱可视化——和 WikiLoop raw→wiki 蒸馏几乎一样，证明我们方向正确
- **原生 MCP**：stdio/SSE/HTTP 三传输，v0.6.1 GA，支持 human-in-the-loop 工具审批
- **混合检索**：BM25 稀疏 + Dense 向量 + GraphRAG 三路 fan-out + Reranker
- **parent-child chunking**：自适应3级分块，命中子 chunk 返回父文档完整内容
- **可观测性**：Langfuse 全链路追踪，蒸馏进度时间线

**WikiLoop 可借鉴：**
1. Wiki 页间链接生成算法（蒸馏时自动建立 related_to 关系）
2. MCP multi-transport：增加 stdio 传输（当前只有 HTTP）
3. GraphRAG fan-out 路由策略（与 19.1 SAG 结合）

---

## 二十一、多语言支持（长期方向，勿删）

> ⚠️ 长期方向，不因短期未实现而删除。WikiLoop 当前定位中文知识库，以下为支持英文用户的演进路径。

### 21.1 多语言用户支持★★
**状态：** 🔲 长期探索（2026-06-23 分析）
**背景：** 当前架构以中文知识库为主，synthesized page 标题为中文，英文用户通过 ALIAS RULE 内联英文词可基本使用，但体验不是最优。
**现状分析：**
- FTS 查询：英文用户能命中（ALIAS RULE 内联英文词，synthesized page description 有英文术语）
- 向量搜索：bge-small-zh 中文优化模型，英文查询向量质量下降
- Hit Rate：synthesized page 中文标题对英文查询 MRR 稍低

**演进路径（按优先级）：**
1. **title_en 字段**（极小改动）：synthesized page frontmatter 加 `title_en` 字段存英文副标题，FTS 索引同时覆盖中英文标题
2. **多语言嵌入模型**（中等改动）：bge-small-zh → bge-m3（支持中英文，568MB，性能稍慢），解决向量语义搜索的跨语言问题
3. **lang 字段**（极小改动）：frontmatter 加 `lang: zh|en|mixed`，支持按语言过滤查询
4. **双语 synthesized page**（大改动）：同一知识点生成中英文两份 synthesized page，独立索引

**行业惯例：** 方案1（融合，中英文混排）是最符合行业实践的做法，Elasticsearch 推荐"synonym at index time"，即在文档级别内联同义词，无需多索引。WikiLoop 的 ALIAS RULE 已是这个方向的最佳实践。

**当前决策：** 保持现有中文为主+双语内联的架构，待有英文用户需求时优先实现 title_en 字段。

---

## 二十二、WeKnora 技术借鉴（腾讯开源，Go，16.9K Stars）

> 来源：代码级分析 https://github.com/Tencent/WeKnora，2026-06-23

### 22.1 WikiBoost 数值参考★★
**状态：** 🔲 待验证
**WeKnora 实现：** `wikiBoostFactor = 1.3`，在 CHUNK_RERANK 阶段后对 `ChunkType == "wiki_page"` 的 chunk 乘以 1.3 倍
**WikiLoop 现状：** `synthesizedBoost = 1.0/(rrfK+1) * 0.8` ≈ 0.008 的绝对加分
**参考价值：** WeKnora 用乘法而非加法，效果更稳定（不依赖 RRF 基础分的大小）。可考虑改为乘法：`score *= 1.3`
**风险：** 乘法 boost 可能让低质量 synthesized page 也排到前面，需要配合 description 长度过滤

---

### 22.2 Query Expansion 条件触发★★
**状态：** 🔲 待验证
**WeKnora 实现：** 只在初始召回数量低于阈值时才触发 query expansion，goroutine 并行扩词查询，semaphore 限制并发数=16
**WikiLoop 现状：** MCP instructions 静态指引 Agent 扩词，无条件触发
**参考价值：** 条件触发更高效——召回已足够时跳过扩词，节省 LLM 推理时间
**实现路径：** `BuildContext` 内部判断 `len(results) < threshold`，不足时触发第二轮扩词查询

---

### 22.3 RRF 向量/关键词权重分离★
**状态：** ❌ 已关闭（去向量化后仅剩 FTS，此项无意义）
**WeKnora 实现：** `vectorWeight/(k+vectorRank) + keywordWeight/(k+keywordRank)`，k/vectorWeight/keywordWeight 均可配置
**WikiLoop 现状：** 向量和 FTS 权重相同（都是 1.0/(60+rank)）
**参考价值：** 针对不同查询类型（精确匹配 vs 语义搜索）动态调整权重，CP 和 CR 更灵活
**改动量：** 中（config.yaml 加配置项，search.go 改 RRF 公式）

---

### 22.5 Synthesized Pages 聚类合并（Draft 门槛机制）★★
**状态：** ✅ 已完成（draftThreshold=2，_draft/ 机制，2026-06-24）
**问题：** 当前 synthesize 对每篇 source-note 独立生成 concept/comparison/decision 页，导致同话题页面数量与 source-note 成正比，内容碎片化。
**方案：**
1. **FTS 查找已有页**：synthesize 一篇新文档时，用其 tags/key_claims 关键词搜索 `concepts/`、`comparisons/`、`decisions/` 已有页面
   - 命中 → 把新 source-note 追加到已有页的 sources 列表，触发重新生成（素材增加）
   - 未命中 → 新建草稿页，写入 `concepts/_draft/`（或 `comparisons/_draft/`、`decisions/_draft/`）
2. **草稿门槛**：`_draft/` 子目录下的页面不纳入索引；当某草稿页 sources 数量达到阈值（建议 3 篇）时，移到父目录，触发重新 synthesize + 索引
3. **索引跳过规则**：`IndexFiles` 跳过路径含 `/_draft/` 的文件

**优点：**
- 综合页真正跨文档，内容密度更高
- 低频话题（<3篇）不污染索引，避免单篇低质量综合页
- 直接复用 FTS，无需向量聚类
- 偶发话题归错簇影响小，可接受

**改动范围：** `internal/synthesize/plan.go`（查找已有页逻辑）、`internal/kb/index.go`（跳过 `_draft/`）、synthesize 触发脚本
**依赖：** 需先稳定现有 synthesize 流程

---

---

## 二十三、Agent 使用行为分析（2026-06-23 实测）

通过让独立 Agent 用 MCP 回答 8 道知识库问题，观察其查询日志，得出以下结论。

### 23.1 Agent 实际查询行为

**两种主要模式：**

| 模式 | 代表题目 | 行为描述 |
|---|---|---|
| **先 context 后 search** | Q3向量数据库、Q4 Agentic RAG | 先用完整问题调 kb_context，再用提取关键词做 kb_search 补充 |
| **先大量 search 后 context** | Q6召回优化、Q8章节树、Q9记忆、Q10 LLM Wiki | 先用 10-19 次 kb_search 覆盖各子话题，最后一次 kb_context 汇总 |

模式B（后6题）的扩词质量极高，Agent 自发做了查询扩展——把一个大问题拆成每个子话题分别搜索，等于手动实现了 Multi-Query。

**重复调用问题：** Q3 同一查询词调了 3 次，Q5 同一问题调了 4 次，是 Agent 对结果不确定时的重试行为，浪费 token 但无实质收益。

**实际命中情况：** 全部 8 题都命中了相关 source-note，回答质量高，综合了多篇文档内容。

---

### 23.2 六大核心问题与方向

**Q1：怎么引导 Agent 使用系统**
- Agent 会自发阅读 serverInstructions，但执行不稳定（Q3 没扩词，Q6 扩词很好）
- 当前 serverInstructions 有"强制扩词"指引，但 Agent 是否执行取决于它对任务难度的判断
- **方向：** serverInstructions 加更具体的使用模式示例（先 search 多角度，再 context 汇总）

**Q2：Agent 是怎么使用系统的**
- 简单问题：1次 context + 少量 search
- 复杂问题：大量 search 预扫各子话题 + 1次 context 汇总
- 结论：Agent 把 kb_search 当"精确探针"，kb_context 当"最终汇总"，是合理用法
- **问题：** 同一查询重复调用，缺乏"已有足够 context"的判断

**Q3：针对 Agent 习惯，如何让查询更高效**
- Agent 会自发拆分子话题搜索，FTS 需要支持精确关键词命中
- ALIAS RULE（蒸馏时扩词）让 source-note 能被多种查询词命中，对 Agent 的多角度搜索帮助很大
- **方向：** serverInstructions 加"同一查询不重复调用"规则；支持 Agent 批量 search 后一次 context

**Q4：深度 vs 广度如何反馈给 Agent**
- 当前 kb_context 返回混合结果（source-note + concept + comparison），Agent 能感知但不能主动控制深度
- 复杂综合问题 Agent 倾向拿更多文档（20-26篇），简单问题拿 5-12 篇
- **方向：** kb_context 返回结果可以附带"已覆盖子话题列表"，让 Agent 知道哪些角度已经有了，不用重复搜

**Q5：文件关系、重要程度对 Agent 是否有帮助**
- authority 字段已在 search 结果里返回，Agent 回答中确实优先引用 authority=3 的 decision/concept 页
- graph_pages（关联文档）Agent 也会参考，对扩展相关话题有帮助
- **方向：** 在结果里暴露 `sources` 数量（代表跨文档覆盖度），让 Agent 判断综合页质量

**Q6：concept/comparison/decision 对 Agent 是否有用**
- **有用，但使用方式不同于预期：** Agent 不是直接"找到综合页就满足"，而是把综合页作为"话题地图"继续展开搜索
- comparison/decision 页的决策树结构对 Agent 生成结构化回答帮助很大（Q5/Q12 的回答都有决策树）
- concept 页的定义和分类给 Agent 提供了术语框架
- **结论：** concept/comparison/decision 价值确认，但作用是"导航索引"而非"终点"

---

### 23.3 向量搜索在 Agent 实际使用中的边际贡献

评估脚本测出向量贡献显著（Hit Rate +0.250，CP +0.303），但这是**每题只调一次 kb_context** 的场景。

Agent 实际行为是：先 10+ 次 kb_search 覆盖各子话题，最后一次 kb_context 汇总。这两种场景下向量的价值不同：

- kb_search 本身走 FTS + 向量混合，Agent 多角度搜索时 FTS 已能覆盖大部分相关文档
- 最后一次 kb_context 里的向量，主要补充 FTS 语义偏差较大时漏掉的文档
- **推断：** Agent 多次 search 预扫后，向量的边际贡献比评估脚本测出来的小

**待验证：** 用 Agent 实际的查询序列（10+ 次 search + 1 次 context）跑有/无向量对比，而非评估脚本的单次 context。

---

### 23.4 猜想：Agent 把 WikiLoop 当搜索引擎用

从实测行为看，Agent 调用 WikiLoop MCP 的模式和人用搜索引擎几乎一致：

1. **先广度扫描**：多个不同关键词组合搜索，覆盖话题各角度（等同于搜索引擎多次查询）
2. **结果驱动迭代**：看到结果后调整关键词，精准定位（等同于搜索引擎翻页/换词）
3. **最后汇总**：拿到足够素材后，一次性综合生成答案（等同于人读完多个页面后写总结）

**猜想（待验证）：** Agent 调用任何外部知识查询工具，底层行为模式都遵循"搜索引擎思维"——广度优先探索 + 关键词迭代 + 最终综合。这不是 WikiLoop 特有的现象，而是 Agent 的通用外部检索范式。

**推论：**
- WikiLoop 应该把自己当搜索引擎来优化，而不只是 RAG context provider
- FTS 精确匹配的重要性 ≥ 向量语义匹配（Agent 主动做了扩词，语义gap已被查询迭代弥补）
- ALIAS RULE（蒸馏时把同义词/英文变体内嵌到 key_claims）对 Agent 的价值极高——Agent 换词搜索时能命中同一篇文档
- concept/comparison/decision 综合页的价值在于"让 Agent 一次拿到结构化的话题地图"，减少迭代次数

**学术支撑（猜想基本成立）：**

这个行为模式在学术界已有大量研究印证：

| 论文 | 机构/年份 | 核心发现 |
|---|---|---|
| [ReAct](https://arxiv.org/abs/2210.03629) | Google/Princeton 2022 | LLM Agent 交替"推理+行动"，每次检索后根据结果决定下一步查什么，与搜索引擎用户迭代行为一致 |
| [IRCoT](https://arxiv.org/abs/2212.10509) | Stony Brook NLP 2022 | 复杂问题下 LLM 把检索与推理交替进行（推理→检索→推理→再检索），是我们实测"先大量search预扫"模式的理论基础 |
| [MindSearch](https://arxiv.org/abs/2407.20183) | 上海AI Lab 2024 | LLM Agent 在复杂信息检索时"模仿人类搜索思维"——问题分解→多轮子查询→结果聚合，与实测10+次search高度吻合 |
| [Agentic RAG Survey](https://arxiv.org/abs/2501.09136) | 2025综述 | Agentic RAG 核心特征是 dynamic decision-making + iterative refinement + multi-step retrieval，与搜索引擎工作模式趋同 |
| [Search-o1](https://arxiv.org/abs/2501.05366) | 2025 | Agentic RAG 框架增强推理模型的迭代搜索能力 |

**根本原因：** 这不是偶然，是 RLHF/指令微调后 LLM 学会的通用外部知识获取范式。IRCoT/ReAct 等方法本质上是把搜索引擎的"查询迭代+结果消化+再查询"内化到 LLM 推理链里。

**对 WikiLoop 的核心推论：FTS 精确命中质量 > 向量语义匹配。** Agent 会主动做查询迭代来弥补语义 gap，而 FTS 精确性是它无法自己弥补的。

**Tool Learning 综述的直接印证（arxiv:2405.17935）：**

论文把 Tool Learning 分为四阶段：Task Planning → Tool Selection → Tool Calling → Response Generation，并明确分两种范式：
- **一步式**：一次规划后直接产出答案
- **迭代式**：多次调用工具，每步根据返回信息修正

我们实测 Agent 的行为（先大量 search 预扫各子话题，最后一次 context 汇总）正是**迭代式范式**的标准表现。

论文在 Tool Selection 阶段还专门提到：工具数量大时先用 **BM25/TF-IDF** 检索 Top-K 候选，再由 LLM 二次判断。WikiLoop 的 kb_search（FTS）扮演的正是这个"工具检索器"角色，kb_context 是 LLM 的最终汇总调用——和论文描述的两阶段工具选择完全一致。

**华为云工程实践的补充：**
- 工具函数建议 ≤20 个，太多降低选择准确率
- "能用代码完成的任务就不要调用大模型"——确定性任务优先走 FTS/BM25，不确定才走语义/向量
- 这进一步印证：FTS 是 Agent 工具调用的主力，向量是补充

**相关原始文档已入库：**
- `raw/wechat-tech/晴天/2026-06-23-tool-learning-llm-survey.md`（Tool Learning 综述）
- `raw/references/huaweicloud-tool-calling-best-practice.md`（华为云 Tool Calling 实践）

---

### 23.5 核心定位转变：WikiLoop 是搜索引擎，不是答案机

**传统 RAG 定位：** 返回最相关的 context，让 LLM 直接生成答案。追求 CP/CR/Hit Rate。

**新定位：** WikiLoop 是 Agent 的搜索引擎，不是答案提供者。

基于"Agent 本质是搜索引擎用户"这个前提（§23.4 已有学术支撑），WikiLoop 的目标不是每次 kb_context 都返回完美答案，而是：

> **保证 Agent 用任何角度的关键词来查，都能命中相关文档。Agent 自己会多次迭代、综合多篇内容生成答案。**

**这个定位下真正重要的只有两件事：**

1. **连接关系**：source-note 之间的 `related_to`、graph_pages，让 Agent 顺着线索往下走，一篇带出多篇
2. **扩词增强**：ALIAS RULE 把同义词/中英文变体/缩写内嵌到 key_claims，保证 Agent 换词查时能命中同一篇文档

**concept/comparison/decision 的价值重新定义：**
- 不是"给 Agent 一个完整答案页"
- 而是"给 Agent 一张话题地图"——让它一次拿到这个话题所有相关 source-note 的路径，减少迭代次数
- draft 机制（sources<2 不索引）的合理性也由此得到支撑：单篇综合页作为话题地图价值有限

### 23.6 纯 RAG vs Agentic RAG：被动接收 vs 主动验证

**纯 RAG（无工具，越来越少见）：**
- 检索 → 返回 top-k context → LLM 只能用给定 context + 内部参数知识生成答案
- LLM **没有搜索工具**，无法主动发起新查询
- 只能用内部知识对 context 产生质疑，但无法验证
- 答案质量完全依赖初次检索质量，检索错了答案就错

**Agentic RAG（有工具，现实主流）：**
- LLM 被赋予搜索工具（如 WikiLoop MCP、网络搜索）
- 可自主决定何时检索、检索什么、检索几次
- 主动搜索 → 获取候选材料 → **自己判断够不够** → 不够继续搜 → 综合生成答案
- 有验证意识——WikiLoop 返回结果后，仍会去网络搜索佐证
- 这正是实测观察到的行为：Agent 调 10+ 次 search，不满足于第一次结果

| | 纯 RAG（无工具） | Agentic RAG（有工具） |
|---|---|---|
| 检索发起 | 系统代劳，一次性 | LLM 自主决定何时检索 |
| 验证能力 | ❌ 只能用内部知识质疑 | ✅ 可主动搜索佐证 |
| 结果依赖 | 完全依赖初次检索质量 | 可迭代弥补检索不足 |
| 现实占比 | 越来越少 | 成为主流 |

**关键区别：**

纯 RAG 要求系统"给出正确答案"，Agentic RAG 要求系统"提供可被验证的材料"。Agent 自己会做：
1. **交叉验证**：WikiLoop 内部多篇 source-note 对比，看观点是否一致
2. **外部佐证**：网络搜索补充，WikiLoop 没有的内容从外部找
3. **置信度判断**：材料够了就综合生成，不够就继续找

**对 WikiLoop 的意义：**

- WikiLoop MCP 本身就是给 Agent 用的工具，凡是调用它的 LLM 都处于 Agentic 模式——"Agent 会主动验证"对所有 WikiLoop 用户都成立
- WikiLoop **不需要保证"返回内容一定正确完整"**，只需要保证"相关文档能被找到"
- source-note 的质量（authority 字段、来源可信度）比"返回结果的排名"更重要
- Agent 会自己做综合判断，我们只需确保高质量来源文档在搜索范围内能被命中
- **这进一步确认：扩词覆盖 > 排序精度**

---

**评估指标需要重新设计：**

现有 CP/CR/Hit Rate 测的是"单次 context 是否命中 expected_page"，这个方向可能本身就偏了。

真正应该测的是：**Agent 用不同关键词组合查询时，相关文档的覆盖率**
- 例如：同一个话题用 5 种不同表达方式查，有几种能命中相关 source-note？
- 这才反映 ALIAS RULE 和扩词的真实效果
- 待探索：设计"多查询覆盖率"指标替代或补充现有 Hit Rate

---

### 23.7 接口定位重新梳理：kb_search + kb_page
**状态：** ✅ 已完成（kb_page 新接口 + SearchLayered 分层返回，2026-06-24）

**当前两个接口返回的内容深度：**

`kb_search` 和 `kb_context` 返回的 wiki_pages 字段完全相同（SearchResult 结构体）：
- `title`：标题
- `description`：frontmatter 摘要（约100字）
- `snippet`：FTS 命中片段（约50字）
- 元数据：kind/authority/fts_rank/hybrid_score 等
- **无全文内容**

`kb_context` 额外返回：
- `raw_sources`：最多3篇引用原始文档（只有 id/path/title，无内容）
- `graph_pages`：最多3篇关联页（只有 id/path/title，无内容）

**问题：两个接口的内容深度几乎一样，区别只是 kb_context 多了 raw_sources 和 graph_pages。**

Agent 每次拿到的都是摘要+片段，无法读到全文。这解释了为什么 Q3/Q5 会反复查同一查询——不是找不到文档，而是拿到的内容太浅，Agent 不确定够不够用，于是重试。

**两个接口应有的定位差异：**

| 接口 | 当前定位 | 应有定位 |
|---|---|---|
| `kb_search` | 返回排序结果列表（摘要级） | **探针**：广度优先，覆盖各关键词角度，确认哪些文档存在 |
| `kb_context` | 返回聚合 bundle（同样摘要级） | **深读入口**：返回更完整内容，或提供进一步深读的路径 |

**接口描述的问题：**
- `kb_context` 参数名是 `question`，引导 Agent 传完整问题句——符合"给我答案"的 RAG 思维
- `kb_search` 参数名是 `query`，引导 Agent 传关键词——符合搜索引擎思维
- 两者定位差异在描述上不够清晰，Agent 容易把 kb_context 当"最终答案接口"反复调用

**接口重新设计方向（待实现）★★★**

基于"WikiLoop 是搜索引擎不是答案机"的定位，MCP 接口应简化为两个，对应搜索引擎的两个基本动作：

**`kb_search`（广度探索，重新强化）**
- 输入：query（关键词或自然语言）
- 返回：匹配文档列表，每篇包含 title/description/snippet/id
- **新增**：关联关系（related_to、graph_neighbors、conflicts）直接内嵌到每篇结果
- Agent 用来发现"有哪些相关文档"，覆盖各角度关键词

**`kb_page`（深度读取，新增）**
- 输入：id 或 slug（从 kb_search 结果的 id 字段获取）
- 返回：单页完整内容
  - 文本类（source-note/concept/comparison/decision）：完整 markdown 正文
  - 其他类型：title/path/id，Agent 自己判断如何处理
- Agent 在 kb_search 探索后，对感兴趣的文档深读

**`kb_context` 的去向**
- 当前 kb_context 做的聚合（bundle）是 Agent 应该自己做的事
- 可废弃，或保留作为兼容层（内部调用 kb_search + kb_page 组合）
- 搜索引擎只有"搜索"和"打开页面"两个动作，不需要"帮你打包好的答案包"

**迁移后 Agent 工作流：**
```
1. kb_search(query) × N   → 发现相关文档（广度）
2. kb_page(id) × M        → 深读感兴趣的文档（深度）
3. Agent 自己综合 → 生成答案
```

**预期效果：**
- Agent 不再重复调用同一查询（有了 kb_page 可以深读，不需要靠重试拿更多内容）
- 接口语义清晰，Agent 行为更可预测
- 完全符合搜索引擎使用习惯（ReAct/IRCoT 等框架的标准模式）

**短期可做的改进（不改接口，改描述）：**
- `kb_context` description 改为"返回材料供 Agent 综合，不是生成答案"
- serverInstructions 加规则："同一查询词不重复调用，换词或换角度继续探索"

---

### 23.3 评估框架 gap

实测回答质量远高于 Hit Rate=0.583 反映的水平，原因：
- 评估只看 top1 是否命中 expected_page（单篇 source-note）
- Agent 实际拿到 12-26 篇文档综合回答，不依赖单篇命中
- **结论：** 现有 Hit Rate/MRR 衡量的是"精确单篇定位能力"，不能反映"综合回答质量"
- **待探索：** 是否需要补充"相关文档集合命中率"（top10 里有几篇相关的）替代单篇 Hit Rate

---

### 22.4 Wiki Ingest 批次防抖★
**状态：** 🔲 长期探索（企业级场景）
**WeKnora 实现：** 文档上传后 30 秒防抖再触发批次；Redis 分布式锁防止并发；批次最多处理 5 个文档；失败最多重试 5 次，永久失败归档到 dead_letters
**WikiLoop 现状：** 简单文件 watcher，无防抖无锁
**参考价值：** 高并发文档入库时稳定性更好，但依赖 Redis，不符合 WikiLoop 本地优先原则
**当前决策：** 不适用（WikiLoop 单机本地，无需分布式锁）

---

## 二十四、OKF v0.1 字段规范对照（2026-06-24）

OKF（Open Knowledge Format）v0.1 是 WikiLoop 遵循的上游规范。**原则：优先使用 OKF 标准字段，规范没有的才自定义。**

规范来源：https://github.com/GoogleCloudPlatform/knowledge-catalog/tree/main/okf

### 24.1 OKF 标准字段全集

**必填（REQUIRED）：**

| 字段 | 类型 | 说明 | WikiLoop 现状 |
|---|---|---|---|
| `type` | string | 概念类型（如 source-note、concept、comparison、decision） | ✅ 已实现 |

**推荐（RECOMMENDED，按优先级）：**

| 字段 | 类型 | 说明 | WikiLoop 现状 |
|---|---|---|---|
| `title` | string | 人类可读标题 | ✅ 已实现 |
| `description` | string | 单句摘要，用于搜索片段和预览 | ✅ 已实现 |
| `resource` | URI | 底层资产的唯一标识 URI | ✅ 已实现（存原始 URL/citation） |
| `tags` | list[string] | 横向分类标签 | ✅ 已实现 |
| `timestamp` | ISO 8601 | 最后有意义变更时间 | ✅ 已实现（已加强提取约束） |

**可选扩展（OKF 允许任意自定义字段）：**

| 字段 | 类型 | 说明 | WikiLoop 现状 |
|---|---|---|---|
| `okf_version` | string | 声明遵循的 OKF 版本 | 🔲 未实现（可加到 bundle root index.md） |

### 24.2 WikiLoop 自定义扩展字段

以下字段不在 OKF 标准中，是 WikiLoop 根据 Agent 搜索引擎定位自定义的：

| 字段 | 类型 | 用途 | 适用页面 |
|---|---|---|---|
| `key_claims` | list[string] | 核心观点，内嵌 ALIAS（同义词/中英文对照） | source-note |
| `authority` | int 1-5 | 来源权威度，用于搜索排序加权 | source-note |
| `doc_type` | string | 文档类型（技术文章/白皮书/项目文档等） | source-note |
| `sources` | list[path] | 引用的原始文档路径（指向 raw/） | source-note、concept 等 |
| `related_to` | list[path] | 显式关联文档（驱动 kb_search 的 related 字段） | 所有类型 |
| `supports` | list[path] | 支持/佐证的文档 | source-note |
| `contradicts` | list[path] | 与之矛盾的文档 | source-note |
| `source_count` | int | 综合页来源数量，用于 _draft/ 门槛机制 | concept/comparison/decision |

### 24.3 待考虑的 OKF 规范功能（尚未实现）

| OKF 功能 | 说明 | 优先级 |
|---|---|---|
| `# Citations` 章节 | 正文末尾列出外部引用来源（OKF §8） | P3 |
| `index.md` 目录文件 | 每级目录的概念清单，用于渐进式披露（OKF §6） | 🔲（已有部分实现） |
| `log.md` 变更日志 | 按日期记录知识库变更历史（OKF §7） | ✅（已有 wiki/log.md）|
| `okf_version` 声明 | 在 bundle root index.md 的 frontmatter 声明版本 | P3 |

---

## 二十五、Engram vs WikiLoop 架构对比（2026-06-24）

**项目：** https://github.com/Gentleman-Programming/engram（4600 stars，Go + SQLite + FTS5 + MCP，12MB 单二进制）

**定位差异：** Engram 是 Agent 跨 session 短期记忆层；WikiLoop 是长期结构化知识库。两者不竞争，但技术选型几乎一致，Engram 的成功独立验证了 WikiLoop 的 SQLite+FTS5 架构决策。

### 25.1 架构对比概览

| 维度 | Engram | WikiLoop | 结论 |
|---|---|---|---|
| 存储 | SQLite + FTS5（BM25） | SQLite + FTS5（trigram + RRF） | 一致，WikiLoop 多维排名更成熟 |
| MCP | stdio，19 个工具（agent/admin 分档） | HTTP + stdio，5 个工具 | Engram 分档设计值得借鉴 |
| 排名 | BM25 + BM25Floor 过滤 | RRF + wiki_priority + authority + GraphBoost | WikiLoop 更成熟 |
| FTS 查询 | sanitizeFTS（每词加双引号）| AND-first / OR-fallback 两阶段 | WikiLoop 更精细 |
| 多类型并发 | 无 | multiKindFTS + goroutine + RRF 合并 | WikiLoop 独有优势 |
| 知识衰退 | `review_after`（按类型自动过期：决策6月、策略12月） | 无 | **Engram 有，WikiLoop 缺失** |
| 去重 | `normalized_hash`（15分钟窗口内相同内容计数而非重复写） | `content_hash`（仅用于 reindex 跳过） | **Engram 更完善** |
| Git Sync | SQLite → gzip JSONL chunks → git push | 无（依赖用户手动 git push wiki/） | Engram 有，WikiLoop 天然 markdown 已可 git |
| 冲突处理 | 完整状态机（pending/judged/supersedes/not_conflict）+ `mem_judge` 工具 | `ConflictLinks()` 展示冲突对，无判决工具 | Engram 更完整 |

### 25.2 值得 WikiLoop 借鉴的三条建议

**~~P2~~ → P3 — 知识衰退字段（`review_after`）**
- Engram 按知识类型自动计算过期时间，到期触发 review
- WikiLoop 建议：`documents` 表加 `reviewed_at INTEGER` + `review_after INTEGER`，`kb_lint` 附带检测过期页
- 价值：避免 AI 读到过时的 decision/concept 页，尤其技术决策类页面
- **2026-06-26 分析结论：现阶段价值有限，降为 P3 暂不实施。**
  原因：① `doc_timestamp` 已有，搜索时已做时间衰减 boost；② 知识库规模小（170 篇 decision），人工 review 成本低于自动化；③ WikiLoop 知识时效性远低于 Engram 的记忆时效性，6-12 月过期规则不适合；④ 蒸馏 prompt 没有 `review_after` 填写规则，实际会大量空值，lint 报警无意义。
  真正有用的做法：仅对 `doc_type: 技术规范` 和 `wiki/decisions/` 加检测，需先改蒸馏 prompt，改动范围超出 lint.go 单文件。

**P2 — MCP 工具分档（`--tools=agent/admin`）**
- Engram 将 19 个工具拆为 agent 档（常用读写）和 admin 档（维护），减少 agent context 噪音
- WikiLoop 建议：`kb_reindex`/`kb_lint`/`kb_status` 归入 admin 档，agent 默认只看 `kb_search`/`kb_page`
- 价值：serverInstructions 更简洁，agent 不会误调维护工具

**P3 — `normalized_hash` 语义去重**
- Engram 对 content normalize 后 SHA-256，相同内容不重复写行
- WikiLoop 建议：升级 `content_hash` 为去重键，相同内容更新时间戳而非重写行
- 价值：防止 FTS 索引膨胀，尤其 watcher 频繁触发时

---

## 二十七、鸿蒙 PC 平台支持（2026-06-25）

**状态：** 🔲 长期探索（等待 Go 官方支持）

**背景：** HarmonyOS NEXT PC 基于自研微内核，不再兼容 Linux，Go 官方目前无 `GOOS=harmonyos` target。

**现状分析：**

| 方案 | 可行性 | 说明 |
|---|---|---|
| 原生 Go 编译 | ❌ 暂不可行 | Go 官方无鸿蒙 PC target，华为内部有但未开放 |
| Go → C/C++ .so | ❌ 很难 | 依赖完整 Go runtime 移植到鸿蒙微内核 |
| WebAssembly | ⚠️ 中等 | 鸿蒙 PC 浏览器支持 WASM，但 SQLite FTS5 需适配 |
| **浏览器访问 WebUI** | ✅ 立即可用 | WikiLoop 跑在 Mac/Linux，鸿蒙 PC 用浏览器访问 `http://IP:8766` |
| 等待 Go 官方支持 | 🔲 未来 | 鸿蒙生态扩张中，Go 支持只是时间问题 |

**短期建议：** 通过浏览器访问 WebUI 即可使用全部功能（搜索、文件管理、状态）；MCP 走 HTTP 协议，如果鸿蒙 PC 上有支持 MCP 的 AI 工具也可直接接入。

**触发条件：** Go 官方添加 `GOOS=harmonyos` target，或鸿蒙 NDK 有成熟的 Go 社区移植方案。

---

## 二十六、codebase-memory-mcp 架构研究（2026-06-24）

**项目：** https://github.com/DeusData/codebase-memory-mcp
**技术栈：** 纯 C，单二进制，零外部依赖
**定位：** 代码库知识图谱 MCP 服务器（不是知识文档库，但搜索架构高度可借鉴）

### 26.1 向量搜索实现

**完全本地，无网络调用，无 ONNX/Ollama：**

- **存储：** 纯 SQLite，自建 `node_vectors`（int8 量化节点向量）和 `token_vectors`（词向量 + IDF），自己用 C 实现 `cbm_cosine_i8` 函数注册到 SQLite，完全不依赖 sqlite-vec 等扩展
- **Embedding 计算：** 把 40856 个 token × 768 维 int8 量化向量（nomic-embed-code 模型离线蒸馏，约 30MB）直接编译进二进制（汇编 `.incbin`）；未知 token 用 XXH3 seed 生成稀疏随机向量（RI，8个非零位）兜底
- **查询方式：** `semantic_query` 接收关键词数组，每个词独立构建查询向量，取 **min-cosine**（所有词的最小值），确保候选结果同时与所有关键词相关（等价于 AND 语义的向量版本）
- **关键文件：** `src/store/store.c`（向量搜索）、`src/semantic/semantic.c`（RI + nomic 查找）、`vendored/nomic/code_vectors.h`（预训练词向量）

### 26.2 图搜索实现

**SQLite 属性图 + WITH RECURSIVE CTE，无专用图数据库：**

- **Schema：** `nodes`（id/label/name/qualified_name/file_path/行号/JSON properties）+ `edges`（source_id/target_id/type/JSON properties）+ `nodes_fts`（FTS5，unicode61 分词）
- **多跳追踪（trace_path）：** `cbm_store_bfs()` 用 SQLite `WITH RECURSIVE` CTE 实现 BFS，支持 inbound/outbound 双向，最大深度可配
- **语义搜索（search_graph）：** 先走 FTS5 BM25，fallback regex，可追加向量结果，三路合并
- **Cypher 查询（query_graph）：** 自实现完整 Cypher 引擎（lexer + parser + planner），把 Cypher MATCH 翻译为 SQL；变长路径 `[*..n]` 通过调用 BFS 展开
- **关键文件：** `src/store/store.c`（Schema + BFS）、`src/cypher/cypher.c`（Cypher 引擎）

### 26.3 对 WikiLoop 的价值（按优先级）

**P2 — WITH RECURSIVE CTE 实现多跳图查询**
- codebase-memory-mcp 用 SQL 递归 CTE 实现多跳 BFS，不需要图数据库
- WikiLoop 现在 GraphExpand 只做 1 跳，改成 `WITH RECURSIVE` 即可实现多跳，和 19.1 SAG 探索项高度契合
- 改动范围：`internal/kb/graph.go` 的 `GraphExpand` 函数
- 价值：Agent 找到一篇文章后能顺着关系链追踪到 2-3 跳外的相关文档
- **⚠️ 依赖 19.1 SAG 先完成：** 当前 `related_to` 只有 37 条（1678 篇文档），多跳收益极低。19.1 SAG 建立实体关联后，`related_to` 密度大幅提升，多跳才有实质价值。**建议先做 19.1，再做 26.1。**

**P2 — 预训练词向量 bake-in + RI fallback 的轻量向量方案**
- 把多语言预训练词向量（如 fastText multilingual，覆盖中英文）int8 量化后用 `//go:embed` 编译进 Go 二进制
- 未知词用随机稀疏向量兜底（Random Indexing）
- 在 SQLite 注册自定义余弦函数（CGO 或 pure Go 实现）
- **优势：** 零运行时依赖，冷启动无网络，比之前的 ONNX bge 模型方案轻 10 倍以上（30MB vs 90MB）
- 这是重新引入向量能力的最轻量方案，不需要 CGO，不需要 ONNX Runtime

**P3 — min-cosine 多词强制相关查询**
- `semantic_query` 接受词数组，取各词向量的 min-cosine 而非 avg
- 等价于 AND 语义：候选文档必须同时与所有词相关
- WikiLoop 可在向量层引入后，用此方式改善多关键词查询精度

---

## 二十八、托管 Agent 环境完整入库流程支持（2026-06-25）

### 背景

WikiLoop 以 MCP stdio 方式运行在 Hermes/OpenClaw 等托管 Agent 环境中：

```yaml
mcp_servers:
  wikiloop:
    command: /root/.openclaw/wikiloop/wikiloop
    args: [serve]
    env:
      WIKILOOP_KB: /root/.openclaw/wikiloop-kb
```

**当前 MCP 只暴露 `kb_search` 和 `kb_page`（只读）**，Agent 无法完成完整的知识入库流程。

### 当前断点

```
Agent 想添加文章
  → ❌ 无 MCP 工具写入 raw/
  → ❌ 无法触发 distill
  → ❌ 知识库不更新
```

Agent 要完成知识入库，需要 Bash 权限 + 手动调用 CLI，体验差，也不安全。

### 28.1 kb_ingest MCP 工具★★

**状态：** 🔲 待实现
**方案：** 在 MCP 中加一个 `kb_ingest` 工具，接受内容 + 路径，直接写入 raw/ 并让 watcher 自动触发后续流程：

```
Agent 调用: kb_ingest(content="...", path="wechat-tech/author/2026-06-25-title.md")
  → 写入 raw/wechat-tech/author/2026-06-25-title.md
  → watcher 感知文件变化
  → 自动触发 distill → index → synthesize
  → 知识库更新完成
```

**工具接口设计：**
- `path`：raw/ 下的相对路径（`wechat-tech/xxx.md`）
- `content`：文件内容（Markdown）
- 返回：`{ok: true, path: "raw/..."}`

**改动范围：** `internal/mcp/server.go` + `internal/mcp/handlers.go` 加一个工具，约 30 行

**依赖：** watcher 必须在运行（`wikiloop serve` 模式），否则需要同步调用 distill

### 28.2 WikiLoop Skill 文件（可选补充）

写一个 `wikiloop-skill.md` 指导 Agent 如何高效使用 WikiLoop MCP：
- 什么时候用 `kb_search` vs `kb_page`
- 如何用 `kb_ingest` 添加新知识
- 多角度搜索的最佳实践（QUERY EXPANSION）
- 推荐的工作流：搜索 → 深读 → 综合回答

**改动范围：** 新建 skill 文件，无代码改动

---

## 二十九、蒸馏队列（持久化 + 并发 + 重试）（2026-06-25）

### 背景

当前蒸馏流程：
```
raw/ 文件变化 → watcher debounce → FindNewFiles() → 串行逐一调用 LLM 蒸馏
```

**问题：**
- 一次放入大量文件（如 100 篇微信文章）时，串行处理慢、无进度、中途失败无重试
- 没有持久化，重启后重新扫描（会重复处理或遗漏）
- 无法查看蒸馏进度

### 29.1 蒸馏队列设计★★

**状态：** 🔲 待实现

**核心设计：**

```
raw/ 文件写入
  → 入队（SQLite distill_queue 表）
  → N 个 worker goroutine 并发处理
  → 失败自动重试（最多 5 次，指数退避）
  → 完成后写入 wiki/source-notes/
```

**Schema（SQLite 新增表）：**
```sql
CREATE TABLE distill_queue (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    path TEXT NOT NULL UNIQUE,      -- raw/ 相对路径
    status TEXT NOT NULL DEFAULT 'pending',  -- pending/processing/done/failed
    retry_count INTEGER NOT NULL DEFAULT 0,
    last_error TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);
```

**改动范围：**
- `internal/kb/schema.go`：加 `distill_queue` 表
- `internal/distill/queue.go`（新建）：入队、出队、重试逻辑
- `internal/distill/worker.go`（新建）：worker pool，N 个 goroutine 并发
- `internal/watcher/watcher.go`：文件变化时改为入队，而非直接触发 reindexFn
- `internal/mcp/handlers.go`：`handleKBStatus` 返回队列状态（pending/processing 数量）

**并发数建议：** 默认 3 个 worker（避免 LLM API 429），可通过 config.yaml 配置

**进度可见：** `kb_status` 返回 `distill_queue: {pending: 23, processing: 2, failed: 1}`

**与 WeKnora 的对比：**
- WeKnora：30s 防抖 + Redis 分布式锁 + 批次最多 5 文档
- WikiLoop：SQLite 队列（本地优先，无 Redis）+ N worker goroutine + 单文档粒度

**依赖：** 需要先完成 28.1 kb_ingest（托管环境入库场景更需要队列）

---

## 三十、内建轻量文档转换（无 markitdown 降级方案）

### 背景

markitdown 是推荐的转换工具，但在某些托管环境可能无法安装（无 pip、受限容器）。
当前降级行为：找不到 markitdown/pandoc → `convert.Run()` 跳过 → PDF/Word/Excel 只能蒸馏到文件名，内容缺失。

**各格式无工具时的现状：**

| 格式 | 有 markitdown | 无任何工具 |
|------|--------------|-----------|
| `.md` / `.txt` | ✅ 直接索引 | ✅ 直接索引 |
| `.pdf` | ✅ 文本提取 | ⚠️ 只有文件名 |
| `.docx` | ✅ 完整转换 | ⚠️ 只有文件名 |
| `.xlsx` | ✅ 表格转 MD | ⚠️ 只有文件名 |
| `.pptx` | ✅ 幻灯片转 MD | ⚠️ 只有文件名 |
| `.epub` | ✅ 完整转换 | ⚠️ 只有文件名 |

### 30.1 内建轻量转换器★★

**状态：** 🔲 待社区反馈后实现

**思路：** 利用 Go 标准库 + 纯 Go 依赖，在无外部工具时提供基础转换能力，不求完美，但能拿到主要文本内容。

**可行方案（纯 Go，无 CGO）：**

| 格式 | Go 库 | 能力 |
|------|-------|------|
| `.docx` / `.xlsx` / `.pptx` | `archive/zip` 标准库（本质是 zip） | 提取 XML 中的纯文本，表格结构丢失 |
| `.epub` | `archive/zip` 标准库（同为 zip） | 提取 HTML 章节文本 |
| `.pdf` | `github.com/pdfcpu/pdfcpu` 或 `github.com/ledongthuc/pdf` | 纯文本 PDF 可提取，扫描件无法处理 |

**降级优先级：**
```
markitdown（最优）
  → pandoc（次优）
  → 内建轻量转换（基础文本，无格式）
  → 只记录文件名/元信息（最差）
```

**改动范围：**
- `internal/convert/convert.go`：在 `FindConverter()` 返回空时，调用内建转换
- `internal/convert/builtin.go`（新建）：纯 Go 轻量转换实现

**触发条件：** 社区用户反馈"没有 markitdown 时 PDF/Word 内容无法蒸馏"达到 3 次以上，再实现。

**当前建议用户的替代方案：**
- 托管环境：`pip install markitdown`（openclaw/hermes 均已验证可安装）
- Agent 侧转换：Agent 用 LLM 视觉能力提取文本后通过 `kb_add` 写入 `raw/converted/`

---

## 三十一、业务逻辑统一层

### 31.1 kb 业务逻辑统一（kbsvc）★★

**状态：** 🔲 待实现

**背景：** 当前 MCP 和 WebUI 各自实现了业务逻辑：

```
MCP handlers.go    → handleKBSearch()  → kb.SearchLayered()
WebUI api.go       → handleSearch()    → kb.SearchLayered()（已统一，2026-06-25）

MCP handlers.go    → handleKBStatus()  → kb.LayerCounts() + distill.Stats()
WebUI api.go       → handleStatus()    → kb.LayerCounts() + distill.Stats()（重复）

MCP handlers.go    → handleKBPage()    → kb.FetchPages()
WebUI api.go       → （无对应接口）
```

**目标架构：** MCP 和 WebUI 都是协议包装器，底层调同一套业务函数：

```
internal/kb/service.go（新文件）
  Search(db, kbRoot, query, layer, kind, limit) → ([]SearchResult, []Conflict, error)
  Page(db, kbRoot, ids, full)                   → ([]PageResult, error)
  Status(db, kbRoot)                            → (*StatusResult, error)
  Lint(kbRoot)                                  → ([]LintWarning, error)
  Add(db, kbRoot, filename, content, sourceURL, overwrite) → (string, error)

MCP handlers.go → 仅做参数解析 + JSON 序列化，调 kb.Service*
WebUI api.go    → 仅做 HTTP 参数解析 + JSON 响应，调 kb.Service*
```

**好处：**
- 逻辑只写一次，WebUI/MCP 行为完全一致
- 新增协议（CLI、gRPC）只需加包装层
- 单元测试只测 service.go，不需要模拟 HTTP/MCP

**改动范围：**
- 新建 `internal/kb/service.go`（~100 行）
- `internal/mcp/handlers.go`：删除业务逻辑，改为薄包装
- `internal/webui/api.go`：同上

**依赖：** 无，可独立完成

---

## 三十二、RAG/检索前沿方向（2026-06-26 文献综述）

来源：8 篇微信文章综合分析，聚焦对 WikiLoop 有实际指导价值的方向。

### 32.1 查询反哺 Wiki（Query回流）★★★

**状态：** 🔲 待实现（对应原 13.1，更新描述）

**来源：** 两篇独立实践文章（Kain思维引擎、陈一豪）均指向同一结论；EvoRAG 论文提供学术支撑

**问题：** Agent 每次对话产生的有价值分析结果消失在对话历史里，下次相同问题仍从零开始。

**设计：**
- Hermes/OpenClaw Agent 完成一次 kb_search + kb_page 查询后，若产生新洞察，通过 `kb_add` 写入 `raw/insights/` 或更新已有 source-note 的 related_to 字段
- WikiLoop 蒸馏流水线自动处理：新的 raw 文件 → source-note → synthesize
- 参考 EvoRAG：Agent 读了某文档后找到答案 = 正向反馈，该文档在 related 中的权重应提升

**实现路径：** 分两步
1. **短期**：在 serverInstructions 里引导 Agent 主动调用 kb_add 写入对话洞察（零代码）
2. **长期**：query log 分析——Agent 搜索后紧接着 kb_page 读了哪些文档，作为隐式正向反馈信号，提升这些文档在 related 中的排序权重

**改动量：** 短期零代码；长期小（search.go 加权重调整逻辑）

---

### 32.2 红链作为知识缺口信号★★

**状态：** 🔲 待验证

**来源：** Kain思维引擎实践文章

**问题：** WikiLoop 目前不知道"哪些概念被频繁引用但还没有独立页面"。

**思路：**
- 蒸馏时 LLM 在 key_claims 里写到某个概念，但 wiki/ 里没有对应页面 → "红链"
- lint 时统计红链被引用次数，输出"知识缺口清单"：本月哪些概念被引用最多但尚未编译？
- 优先蒸馏高频红链概念

**实现：** `kb_lint` 新增 `missing_concept` 类型警告，扫描 source-note 的 related_to/supports 字段，找指向不存在文档的链接

**改动量：** 小（lint.go 加扫描逻辑）

---

### 32.3 用户反馈驱动 related 权重（EvoRAG 思路）★★

**状态：** 🔲 长期探索

**来源：** EvoRAG 论文（东北大学，准确率提升 7.34%）

**核心思路：** EvoRAG 把用户反馈"反传"到知识图谱三元组，经常出现在好答案路径的三元组权重上升，经常出现在坏答案路径的降权。

**WikiLoop 对应：**
- Agent 通过 kb_search 找到文档 A，然后 kb_page 精读并给出正确答案 → A 的 related 权重+
- Agent 搜索后没有继续精读（跳过）→ 该文档对该 query 贡献弱
- query log 已有记录（tool/query/related 字段），数据基础具备

**实现路径：**
- query log 分析：`kb_search` 后的 `kb_page` 调用，检查 related 里的文档是否被实际读取
- 被读取的文档在该 query 的语义领域里提升 document_tags 权重（source='feedback'）
- 类似 EvoRAG 的"关系抑制"：长期不被读取的 related 文档降权

**依赖：** 需要 32.1 Query 回流先建立反馈信号采集机制

**改动量：** 中（需要 query log 分析 + document_tags 权重机制）

---

### 32.4 GrepSeek 精确检索思路的验证★

**状态：** ✅ 已间接验证

**来源：** GrepSeek 论文（UMass，多跳 F1=0.5691，超越 BGE/Qwen3-Embedding）

**结论：** WikiLoop 用 SQLite FTS5 精确字符串匹配（而非向量 embedding）的技术选择被学术论文独立验证——精确匹配在多跳任务和实体关联上优于 dense retrieval。GrepSeek 的 DCI 接口（`rg | grep | head`）和 WikiLoop 的 FTS 检索在设计哲学上一致：可审计、可组合、结果可追踪。

**已做：** 无需额外实现，记录为方向验证。

---

### 32.5 EvoEmbedding 时序感知（对 25.1 的补充）★★

**状态：** 🔲 与 25.1 review_after 合并考虑

**来源：** EvoEmbedding 论文（南大，4B 干翻 12B，naive RAG 打败 agentic memory）

**关键发现：** 时序检索——模型遇到"firstly/lastly/最近"等时序关键词时，能感知对应历史阶段并在目标文档上达峰。这说明知识的时效性不只是"有没有 review_after 字段"，还包括"检索时能否感知用户意图里的时序需求"。

**对 WikiLoop 的扩展：**
- 25.1 review_after：已有 doc_timestamp，加 review_after 字段 + lint 检测（静态衰退）
- 动态时序：搜索时检测 query 里的时序意图词（最新/最近/当前/已过时），对 doc_timestamp 较新的文档给予额外 boost

**改动量：** 25.1 小；动态时序 boost 中（search.go 加时序意图检测）

---

### 32.6 EvoRAG 反馈权重优化——前置条件分析★★

**状态：** 🔲 待条件具备（当前信号不可靠，不适合现阶段实施）

**来源：** EvoRAG 论文（东北大学，+7.34%）+ EvoEmbedding 论文（南大）+ query_log 实测分析（2026-06-26）

**核心问题：人类反馈 vs Agent 自发行为的信号质量差异**

EvoRAG 的原始设计针对**人类反馈**——用户对答案打分（1-5），信号是有意识的质量评判，可靠。

WikiLoop 的 query_log 记录的是 **Agent 自发的 search→page 序列**，两者有本质差异：

| 维度 | 人类点击/阅读 | Agent 自发搜索+阅读 |
|------|------------|------------------|
| **意图** | 有意识判断有用性 | 任务驱动，不评判质量 |
| **可靠性** | 高 | 低（可能只是标题匹配好） |
| **噪音** | 误点、好奇心 | FTS 排名偏差、标题党 |
| **EvoRAG 适用性** | ✅ 直接对应 | ⚠️ 需要额外过滤 |

**Agent 读了某篇文档 ≠ 这篇文档质量高**——Agent 可能读了之后发现没用（负向反馈），但 query_log 里看起来和正向反馈完全相同。

**三个潜在优化方向及当前可行性：**

| 优化 | 思路 | 当前可行性 |
|------|------|----------|
| read_count 提升排序权重 | 被精读过的文档 BM25 加权 | ❌ 信号不可靠，可能强化错误排序 |
| related confidence 动态调整 | Agent 沿 related 读到的文档提升 confidence | ❌ Agent 阅读选择不代表关系有效 |
| dynamic_authority 校正 | 用阅读频率校正静态 authority 分 | ❌ authority 是来源权威度，不应被 Agent 行为改变 |

**真正可用的反馈信号（目前缺失）：**

- **来源 A**：用户对 Agent 回答的满意度（人类显式打分）——目前完全没有
- **来源 B**：Agent 在答案中**实际引用了哪些文档**（不只是读了）——query_log 目前记不到

**前置工作：完善 query_log**

在 MCP 响应里让 Agent 上报"哪些文档被实际引用到了答案里"，积累足够的引用数据后再做权重调整。具体做法：在 serverInstructions 里引导 Agent 在调用 `kb_add(insights/...)` 时附带 `cited_ids` 字段，记录本次回答实际引用的文档 ID。

**与 32.3 的关系：** 32.3（用户反馈驱动 related 权重）依赖人类反馈信号，本条分析说明 Agent 自发行为无法替代人类反馈——两者需要分开设计，不能混用。

**改动量：** 当前阶段仅需完善 query_log（小）；权重调整本身待信号具备后再评估。

