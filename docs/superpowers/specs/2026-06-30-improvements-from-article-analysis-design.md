# PieKBS 改进设计：基于文章分析的借鉴方向

**来源分析：** 四篇参考文章（2026-06-29 入库）  
**讨论日期：** 2026-06-30  
**状态：** 待 Review

---

## 背景

基于以下四篇文章的阅读分析，识别出 PieKBS 可借鉴的方向：

1. **RAG评测，做不好等于蒙着眼睛改代码**（RichardFyoung）
2. **chao-rag-wiki：775篇收藏塞进4MB向量库**（鸟窝/smallnest）
3. **hermes 搭建知识库系统**（微信公众号）
4. **向量检索 vs 仓图 vs 知识图谱**（Chyris，介绍 codebase-memory-mcp）

---

## 现状梳理

### 检索现状

- **检索链路**：FTS5（AND-first → OR fallback）→ RRF 融合 → 时间衰减 + authority 加权 + graph_boost → 分层返回（source-note quota + synth quota）
- **向量搜索**：embeddings 表已有，向量已摘除（embed --full 命令存在但未启用），FTS-only 模式
- **Tokenizer**：FTS5 `tokenize='trigram'`，中文 3-gram 窗口，存在词边界切断问题
- **RRF**：已实现，`k=60`（标准值）

### 评测现状

- **eval 框架**：`eval/eval_piekbs.py`，使用 RAGAS 四指标（Faithfulness、Answer Relevancy、Context Precision、Context Recall）
- **题库**：`questions_v2.json`（12 道，静态）
- **当前最佳基准**（v3 题集，2026-06-23，有向量）：AR=1.000，CP=0.536，CR=0.567，Hit Rate=0.583，MRR=0.165
- **eval 触发方式**：手动运行离线脚本，无 CI 集成
- **Context Recall 瓶颈**：CR=0.567，主要受 trigram tokenizer 和向量搜索状态影响

### 去重现状

- **按路径去重**：`upsertDocument` 按 `content_hash` 检测同路径内容变更
- **文件名保护**：`KBAdd` 默认拒绝覆盖已存在文件（`overwrite=false`）
- **source_uri 字段**：`documents` 表已有，解析时从 frontmatter 读取，但 `KBAdd` 写入时不做查重

---

## 改进方向一览

| 优先级 | 事项 | 依据来源 | 现状 |
|---|---|---|---|
| P1 | eval 本地规范化（大功能前手动跑 + baseline 对比） | 第一篇 | 已有脚本，无规范 |
| P1 | gse 中文分词替代 trigram（提升 Context Recall） | 第一篇间接支撑 | 已在 2026-06-28 设计文档中规划为 P2-A |
| P2 | source_uri 去重（kb_add 写入前查重） | 第二篇 chunk 哈希去重思路迁移 | 字段已有，逻辑未实现 |
| P3 | EvalOps 闭环（线上采样 → 自动更新题库） | 第一篇 | 未做 |

---

## P1-A：eval 本地规范化

### 问题

当前 eval 是完全手动的离线脚本，改动后没有标准流程要求跑 eval，改参数全凭感觉。  
CI 不适合跑 eval（原因：LLM 调用、耗时 5-15 分钟、无 API key）。

### 设计

**操作规范（不改代码，只改流程）：**

1. **触发条件**：满足以下任一时，合并前必须本地跑一次 eval：
   - 修改 `internal/kb/search.go`（排序逻辑、权重、RRF）
   - 修改 `internal/kb/index.go`（tokenizer、embedding 相关）
   - 修改 `internal/distill/`（prompt 变更）
   - 修改 `internal/synthesize/`（综合页生成逻辑）

2. **运行方式**：
   ```bash
   cd /path/to/pie-kbs
   python3 eval/eval_piekbs.py
   ```

3. **通过标准**（对比 `eval/baseline_result.json`）：
   - Context Recall (CR) 不低于 baseline - 0.05
   - Hit Rate 不低于 baseline - 0.083（即不掉一个命中）
   - 任一指标相对 baseline 提升 > 0.05，更新 `baseline_result.json`

4. **PR 描述要求**：大功能 PR 需附上 eval 结果截图或 JSON 片段，格式：
   ```
   eval: AR=x.xxx CP=x.xxx CR=x.xxx Hit Rate=x.xxx MRR=x.xxx
   ```

**不改动内容：**
- CI 不加任何 LLM eval 步骤
- `go test ./...` 仍是 CI 唯一的质量门禁

---

## P1-B：gse 中文分词（已在 2026-06-28 文档规划）

详见 `docs/superpowers/specs/2026-06-28-sage-wiki-features-design.md` P2-A 章节。

**补充依据（来自第一篇）：** Context Recall 当前瓶颈（CR=0.567）有相当比例来自 trigram 的词边界切断——搜"机器学习"只能匹配包含"机器学"或"器学习"的 chunk，词级精确匹配缺失。gse 替代 trigram 是提升 CR 的直接路径，与第一篇"八成问题在检索"的判断一致。

**优先级建议：** 提升为 P1，在下一个大功能迭代中优先实现。

---

## P2-A：source_uri 去重

### 问题

`kb_add` 目前只按文件名去重。同一篇微信文章如果用不同文件名两次入库，会产生内容重复的文档占据搜索名额。

### 适用场景

- `kb_add` 时传入了 `source_url`（微信文章、网页等有 URL 来源）
- 同一 URL 多次触发入库

### 不适用场景

- 手动写的文档（无 source_url）
- raw/ 目录下不同作者写的同主题文章

### 设计

在 `KBAdd` 函数写文件前，增加 source_uri 查重：

```go
// 伪代码，在 os.WriteFile 之前
if sourceURL != "" {
    db, _ := OpenDB(kbRoot)
    var existingPath string
    err := db.QueryRow(
        "SELECT path FROM documents WHERE source_uri = ? LIMIT 1",
        sourceURL,
    ).Scan(&existingPath)
    db.Close()
    if err == nil {
        // 已存在相同 source_uri 的文档
        return nil, &KBError{
            Code: 409,
            Message: "document with same source already exists: " + existingPath,
        }
    }
}
```

**返回格式（新增 409 状态码）：**
```json
{
  "error": "document with same source already exists: raw/references/xxx.md"
}
```

**影响文件：**
- `internal/kb/service.go` — `KBAdd` 函数加查重逻辑
- `internal/mcp/handlers.go` — 透传 409 错误（已有 KBError 机制，无需改动）

**不改动内容：**
- schema 不变（`source_uri` 字段已有）
- 无 source_url 时行为不变

---

## P3-A：EvalOps 闭环（长期，暂不实现）

### 设计思路（备忘）

1. `piekbs serve` 记录真实 MCP 查询到 `index/query_log.jsonl`（已有 `AppendQueryLog`）
2. 定期从 query_log 采样高频/多样性问题，人工标注 expected_page 后加入题库
3. 题库版本化（`questions_v3.json` → `questions_v4.json`），每次更新同步更新 baseline
4. 长期目标：每月跑一次 eval，对比 baseline，识别退化

**当前不做的原因：** 题库静态（12 道）足以支撑当前规模的迭代，EvalOps 的价值在 KB 规模和查询量上来后才显著。

---

## 不借鉴的部分

| 方案 | 来源 | 不借鉴原因 |
|---|---|---|
| chao-rag-wiki 全文 RAG 模式 | 第二篇 | PieKBS 定位是"先 distill 再检索"，原文全量索引路线不同 |
| Codebase Memory MCP | 第四篇 | 代码图谱工具，与知识库场景无关 |
| Reasonix 双 AI 分工 | 第三篇 | 工程价值有限，不适用 PieKBS |
| CI eval 门禁 | 第一篇 DeepEval | LLM 依赖无法在 CI 中使用 |
| chunk 内容哈希去重 | 第二篇 | PieKBS 无分块，整文档入库，哈希去重不适用 |

---

## 实现顺序建议

```
Phase 1（即刻可做，零代码改动）：
  P1-A: eval 本地规范化 — 只需写 CONTRIBUTING.md 或 PR 模板，明确 eval 触发条件

Phase 2（下一个功能迭代中）：
  P1-B: gse 中文分词 — 参见 2026-06-28 设计文档，建议提升优先级
  P2-A: source_uri 去重 — 改动小，可随 P1-B 一起提交

Phase 3（长期）：
  P3-A: EvalOps 闭环 — KB 规模和查询量上来后再评估
```

---

## 关键约束

1. **CI 不引入 LLM**：eval 只在本地手动跑，CI 只跑 `go test`
2. **单二进制**：gse 通过 `embed.FS` 打包词典，P2-A source_uri 去重无外部依赖
3. **向后兼容**：source_uri 去重只在有 source_url 时生效，现有数据和调用方不受影响
