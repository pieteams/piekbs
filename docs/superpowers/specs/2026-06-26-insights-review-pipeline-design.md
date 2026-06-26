# Insights 推荐日志流水线设计

## 背景

WikiLoop 知识库当前只能通过人工放置文件到 `raw/` 来增长。引入 `kb_add` 工具后，外部 Agent 可以在对话结束时把洞察写入 `raw/insights/`，作为推荐日志。

`raw/insights/` 定位是**推荐日志目录**，不是正式知识来源：
- Agent 觉得有价值就写，写错了也没关系
- 定期有 worker 扫描，处理完立即删除（无论采用还是跳过）
- 不投入过大资源，整体是轻量的"有则用，无则丢"机制
- worker 审核采用**严格标准**：宁可不用，不要用错

---

## insights 对知识库的价值

基于知识库实际数据，insights 主要弥补以下两类不足：

| 不足类型 | 现状 | insights 能补充什么 |
|---------|------|-------------------|
| **文章关系不足** | 518 篇 source-note，related_to 仅 39 条 | Agent 发现两篇文档有关联但未互指 |
| **综合不足** | 55% 综合页仅引用 ≤2 篇 source-note | Agent 综合多篇得出现有综合页未覆盖的新角度 |

**不适合写 insights 的内容：** 对已有 concept/comparison/decision 页的重新组织——这类内容写入只会造成冗余。

### 反馈可靠性说明

insights 的内容来源是 **Agent 任务驱动下的自发判断**，不是人类有意识的质量评判，两者可靠性有本质差异：

| 来源 | 可靠性 | 原因 |
|------|--------|------|
| 人类显式判断（"这两篇有关联"） | 高 | 有意识的质量评判 |
| Agent 任务驱动的判断 | 中低 | 可能受 FTS 排名偏差、标题匹配影响，不代表真实关联质量 |

**具体风险：**
- Agent 判断"A 和 B 有关联"——可能只是因为两篇文档都命中了同一个关键词，并非真正的语义关联
- Agent 综合结论——可能是对已有 comparison 页的重新表述，Agent 自身无法可靠判断是否"新角度"
- 外部补充——Agent 使用训练知识补充，存在知识截止日期和幻觉风险

**因此 InsightWorker 的 LLM 审核是必要的第二道过滤**，不能直接信任 Agent 写入的内容。审核标准必须严格，不确定时一律 skip——这正是"宁可不用，不要用错"原则的来源。

**与权重优化的边界：** insights 的价值是**补充知识内容**（新的 related_to 关系、新的综合角度、外部事实），而不是**调整检索权重**。用 Agent 写 insights 的频率来调整文档的 authority 或 related confidence 是不可靠的——Agent 写了某条 insights 不代表对应文档质量更高（参见探索文档 §32.6）。

---

## insights 文件格式

Agent 回答一个问题后，把所有发现写入**同一个文件**，按发现类型分 section，只写有内容的 section：

```markdown
# [问题主题] 查询洞察

## 跨文档关联发现
<!-- 发现两篇文档有关联，但 related_to 未互指 -->
- [wiki/source-notes/A.md] 和 [wiki/source-notes/B.md] 在 X 概念上有直接关联

## 综合结论
<!-- 综合 ≥3 篇文档，得出现有综合页没有覆盖的新角度 -->
综合 A、B、C 三篇关于 X 的内容，得出：...

## 外部补充
<!-- 使用了知识库以外的信息（训练知识、外部搜索） -->
论文 X 的实验数据显示：...

## 引用来源
- [wiki/source-notes/...]
```

---

## 整体数据流

```
Agent 对话结束
  └─ kb_add(filename="insights/YYYY-MM-DD-<slug>.md", content="...")
       └─ 写入 raw/insights/
            └─ 不触发 distill_queue（跳过正式蒸馏，不做 FTS 索引）

InsightWorker（定时，每周一次）
  └─ 扫描 raw/insights/ 所有文件
       └─ 对每个文件：
            ├─ FTS 搜索知识库，找相关 source-note（≤5 篇）作为参考
            ├─ 调 LLM 严格审核：对知识库是否有确定的增量价值？
            │   （宁可不用，不要用错）
            │
            ├─ 有确定价值
            │    ├─ 写入 raw/reviewed/<category>/<slug>.md
            │    ├─ distill_queue 入队（触发正式蒸馏）
            │    └─ 删除 raw/insights/<slug>.md（立即）
            │
            └─ 无价值 / 不确定 / 重复
                 └─ 删除 raw/insights/<slug>.md（立即）
```

---

## 组件设计

### 1. `internal/distill/insights_worker.go`（新文件，~80 行）

```go
// RunInsightWorker 每周扫描一次 raw/insights/，有价值的内容 promote 到 raw/reviewed/。
func RunInsightWorker(ctx context.Context, cfg Config, kbRoot string)

// reviewInsightFile 审核单个文件。
// 返回 (skip, category, formattedContent, err)
// skip=true 时 category/formattedContent 为空，文件直接删除。
func reviewInsightFile(cfg Config, kbRoot string, path string) (bool, string, string, error)
```

**LLM prompt 结构：**

系统提示：
```
你是知识库质量审核员。评估从外部 Agent 对话中提取的洞察，
判断对知识库是否有增量价值。
```

用户提示：
```
## 待审核洞察
<insights 文件全文>

## 知识库参考资料（FTS 搜索结果，≤5 篇）
<source-note 标题 + description + snippet>

## 审核标准

判断以下任一条件是否成立：
1. 发现了两篇文档的关联，但知识库中未建立 related_to 链接
2. 综合结论有新角度，现有 concept/comparison/decision 页未覆盖
3. 包含知识库以外的具体数据或事实

不符合以上任一条件，或有任何不确定时 → skip（宁可不用，不要用错）

输出 JSON：
{
  "skip": true 或 false,
  "reason": "一句话",
  "category": "references | insights-synthesis | decisions（skip=false 时填）",
  "formatted_content": "标准 Markdown 全文（skip=false 时填，否则空字符串）"
}

formatted_content 需包含 YAML frontmatter：
  type: source-note
  title, description, tags, doc_type, authority(2-3), resource:"", sources:["__RAW_SOURCE__"], timestamp
  origin: insights   # 标记来源，区别于人工放入的文件，便于审计和清理
```

### 2. `internal/distill/distill.go`（改动）

`FindNewFiles` 跳过 `raw/insights/` 路径，防止被普通蒸馏 worker 处理：

```go
if strings.HasPrefix(rel, "insights"+string(filepath.Separator)) {
    return nil
}
```

### 3. `internal/watcher/watcher.go`（改动）

`raw/insights/` 路径**跳过所有处理**（不索引、不蒸馏）：

```go
if strings.HasPrefix(rel, "insights/") {
    return // 跳过，由 InsightWorker 定期处理
}
// 原有逻辑
```

---

## 文件改动汇总

| 文件 | 类型 | 改动说明 |
|------|------|----------|
| `internal/distill/insights_worker.go` | 新建 | InsightWorker：每周扫描 + LLM 严格审核 + promote + 立即删除 |
| `internal/distill/distill.go` | 改动 | `FindNewFiles` 跳过 `raw/insights/` |
| `internal/watcher/watcher.go` | 改动 | `raw/insights/` 跳过所有处理（不索引不蒸馏） |
| `internal/mcp/server.go` | 已完成 | serverInstructions KNOWLEDGE CAPTURE 段 |

**不需要：**
- ~~`insights_queue` 表~~（无状态机）
- ~~schema.go / db.go 改动~~
- ~~service.go 改动~~

---

## 关键约束

**insights 不触发蒸馏也不做 FTS 索引：** `FindNewFiles`、watcher、FTS 索引均跳过 `raw/insights/`，文件仅供 InsightWorker 读取。

**InsightWorker 每周一次：** insights 是低优先级后台任务，不需要实时处理。

**处理完立即删除：** 无论 promote 还是跳过，文件处理后立即删除。无需定时清理任务，无需记录历史。

**严格审核，宁可不用：** LLM prompt 中明确要求——不确定时输出 skip=true。有价值的判断标准必须满足其中一条：发现未记录的文档关联、综合出现有综合页未覆盖的新角度、包含知识库外的具体可验证数据。

**FTS 参考资料 ≤5 篇：** prompt 长度控制，取 BM25 top-5 摘要（title + description + snippet）。

**LLM API 不可用时静默跳过：** 不重试，下次周期再处理剩余文件。

---

## serverInstructions 引导（已完成）

`internal/mcp/server.go` 的 KNOWLEDGE CAPTURE 段已引导 Agent：
- 触发条件：发现文档关联未互指、综合出新角度、使用了外部信息
- 不写条件：回答完全来自已有综合页（最常见情况）
- 格式：一文件三 section，只写有内容的 section

---

## 已知风险与待解决问题（暂不实施）

以下问题在当前设计中存在，已识别但暂不处理，待知识库规模增长后再评估是否需要干预。

### 风险 1：过拟合——关系网络偏向高频查询话题

**问题：** Agent 写 insights 的频率取决于用户问什么问题。同一对文档被反复问到，related_to 关系被 promote；从未被问到的文档对，关系永远不会被发现。

**结果：** related_to 网络逐渐成为"查询分布的影子"，而不是"文档内容本身的语义结构"。

**缓解（已有）：** InsightWorker 严格审核、蒸馏二次过滤。

**待解决：** 无话题多样性约束，同一话题可以无限积累 promote。

---

### 风险 2：连接泛化不足

**问题：** insights 机制只能覆盖**被查询到的知识路径**。知识库里存在但从未被问到的关联，永远不会通过 insights 被发现。这是结构性局限，无法通过改进 InsightWorker 解决。

**影响范围：** 低频话题的 related_to 稀疏问题不会因为 insights 机制而改善。

**长期方向：** 离线批量分析文档相似度（不依赖查询），作为补充（参见探索文档 §26.1 WITH RECURSIVE 多跳）。

---

### 风险 3：综合页话题分布偏移

**问题：** 用户集中问某类话题时，该话题的 insights 大量积累，promote 后生成大量相关综合页，知识库综合层向高频话题倾斜，低频话题综合覆盖越来越薄。

**缓解（已有）：** `origin: insights` 标记可以追踪哪些综合页来自 insights 流水线。

**待解决方案（暂不实施）：**
- InsightWorker promote 前检查 `raw/reviewed/` 同话题已有内容数量，超过阈值提高 skip 门槛
- `kb_lint` 统计 `origin: insights` 内容占比，超过一定比例时告警

---

### 风险 4：promote 内容无法反向校正

**问题：** 一旦 insights 内容 promote 进 `raw/reviewed/` 并触发蒸馏，没有自动撤回机制。如果某条关系或结论是错的，只能靠 `kb_lint` 人工发现后手动删除。

**缓解（已有）：** `origin: insights` 标记使人工审计可定向进行——`kb_lint` 可以列出所有 `origin: insights` 的文档供人工复核。

**待解决：** 目前 `kb_lint` 尚未实现对 `origin` 字段的检测，需要后续补充。

---

### 总结：当前设计的安全边界

| 风险 | 严重程度 | 当前缓解 | 是否需要立即处理 |
|------|---------|---------|----------------|
| 关系过拟合 | 中 | 严格审核 + 低频处理 | 否，规模小时影响有限 |
| 连接泛化不足 | 低 | 结构性局限，接受 | 否 |
| 话题分布偏移 | 中 | origin 标记可追踪 | 否，待规模增长后观察 |
| 无反向校正 | 低 | origin 标记 + 人工审计 | 否，目前 promote 量极少 |

**结论：** 当前知识库规模（518 篇 source-note）下，这些风险的实际影响可忽略。InsightWorker 的严格审核是主要保护。待 insights 来源内容积累到可观规模（如 50+ 篇）后，再评估是否需要补充话题配额和自动告警机制。
