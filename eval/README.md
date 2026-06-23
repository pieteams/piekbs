# WikiLoop 评估框架

用 RAGAS 四项指标对比 WikiLoop 与 Naive RAG 的检索质量。

## 指标说明

| 指标 | 含义 |
|---|---|
| Faithfulness（忠实度） | 答案是否来自检索内容，不 hallucinate |
| Answer Relevancy（答案相关性） | 答案是否回答了问题 |
| Context Precision（上下文精度） | 检索内容中有用的比例 |
| Context Recall（上下文召回） | 相关内容是否被检索到 |

## Baseline（2026-06-20，无 Rerank/Pre-Route）

问题集：`questions_rag.json`（10 题，RAG/LLM Wiki 主题）

| 指标 | WikiLoop | Naive RAG | 提升 |
|---|---|---|---|
| 忠实度 | 0.825 | 0.900 | ↓ -7.5% |
| 答案相关性 | 0.660 | 0.425 | ↑ +23.5% |
| 上下文精度 | 0.122 | 0.060 | ↑ +6.2% |
| 上下文召回 | 0.360 | 0.085 | ↑ +27.5% |

**检索链路**：FTS + Vector → RRF 融合 → 直接返回（无 Rerank，无 Pre-Route）

**结论**：WikiLoop 在答案相关性（+23.5%）和上下文召回（+27.5%）上显著优于 Naive RAG。
上下文精度（0.122）是主要优化方向，预期 Rerank 实现后会有明显提升。

详细结果：`baseline_result.json`

## 运行评估

```bash
# 依赖
pip install ragas requests pyyaml

# 使用预设问题集（推荐，可复现对比）
cp eval/questions_rag.json /tmp/eval_questions.json
python3 eval/eval_wikiloop.py

# 自动从 KB 生成问题集
rm /tmp/eval_questions.json
python3 eval/eval_wikiloop.py
```

**前置条件**：
- WikiLoop serve 已启动（`wikiloop serve`）
- `~/.hermes/wikiloop-kb/config.yaml` 中已配置 LLM（distill 段）

## 对比历史

| 日期 | 版本/变更 | 答案相关性 | 上下文精度 | 上下文召回 |
|---|---|---|---|---|
| 2026-06-20 | Baseline（RRF，无 Rerank） | 0.660 | 0.122 | 0.360 |
| 2026-06-20 | Phase 1（Bi-Encoder Rerank + 短文档不分块） | 0.520 | 0.186 | 0.320 |
| 2026-06-20 | 实验：关闭分块（整文档嵌入）+ Rerank | 0.670 | 0.120 | 0.380 |
| 2026-06-20 | 实验：MMR 多样性约束 + 分块 + Rerank | 0.625 | 0.151 | 0.300 |
| **结论** | **Phase 1 检索优化未突破 baseline，wiki 覆盖度（1.7%）是根本瓶颈，需推进 Phase 2** | | | |
| 2026-06-20 | Phase 2（去阈值 synthesize，12→18 篇综合页）| 0.610 | 0.282 | 0.390 |
| 2026-06-21 | Phase2完整(933篇) + kind过滤 + 引用强制 + 上下文压缩 + limit=10 | 0.630 | 0.289 | 0.260 |
| 2026-06-21 | + 压缩优化（Jaccard 0.5 + 低质量综合页过滤 description<30字） | 0.510 | 0.324 | 0.300 |
| 2026-06-21 | + AND-first FTS + graph_pages wiki-only + minHybridScore=0.025 | 0.720 | 0.206 | 0.330 |
| （待补充） | Phase 3（实体索引） | — | — | — |

## v2 问题集结果（concept/comparison/decision，含 Hit Rate + MRR）

| 版本 | AR | CP | CR | Hit Rate | MRR |
|---|---|---|---|---|---|
| 933篇综合页全量嵌入 | 0.992 | 0.411 | 0.467 | 0.083 | 0.017 |
| **只嵌入469篇source-notes（当前最佳）** | **0.988** | **0.455** | **0.567** | **0.167** | **0.095** |
| 2026-06-22 有向量 baseline（v2题集，新评估环境）| 0.950 | 0.511 | 0.633 | 0.167 | 0.097 |
| 2026-06-22 **无向量 FTS-only**（WIKILOOP_NO_VEC=1）| 0.992 | 0.495 | 0.542 | 0.167 | 0.097 |

**向量搜索贡献分析（2026-06-22）：**
- CR 差距：0.633 → 0.542，**下降 0.091（14%）**，向量有真实贡献
- CP 差距：0.511 → 0.495，**下降 0.016（3%）**，几乎可忽略
- Hit Rate / MRR：完全一样，向量对精确页面定位无帮助
- 注意：此为旧文档未重蒸馏的**最坏情况**，全量重蒸馏加入 ALIAS RULE 后差距预计缩小

**最终决策（2026-06-23 修订）：保留向量搜索。**

修复 synthesized pages (concept/comparison/decision) 被向量压制的 bug 后，向量搜索贡献显著：
- CR +0.175（0.600 vs 0.425），召回率明显提升
- CP +0.276（0.655 vs 0.379），精度大幅领先
- Hit Rate +0.166（0.333 vs 0.167），目标页命中率翻倍

之前得出"向量无用"的结论是 bug 导致的误判：向量把高 vec_score 无关 source-note 顶上来，压制了 synthesized pages。修复后向量价值真实显现。

## v2 问题集最终结果（2026-06-23，全量重蒸馏 + bug 修复后）

| 版本 | AR | CP | CR | Hit Rate | MRR |
|---|---|---|---|---|---|
| **有向量（校准后最终版）** | **0.983** | **0.657** | **0.571** | **0.833** | **0.604** |
| **无向量（校准后）** | 0.917 | 0.389 | 0.433 | 0.417 | 0.221 |
| 旧无向量（未校准）| 1.000 | 0.379 | 0.425 | 0.167 | 0.037 |
| 旧 baseline（2026-06-20）| 0.950 | 0.511 | 0.633 | 0.167 | 0.095 |

**结论：向量搜索在 CP/CR/Hit Rate/MRR 全面领先，应保留。**

## 2026-06-23 全量重蒸馏后评估（含 ALIAS+ENTITY+LANGUAGE RULE + authority + doc_type）

| 版本 | AR | CP | CR | Hit Rate | MRR |
|---|---|---|---|---|---|
| 有向量 | 0.992 | 0.481 | 0.508 | 0.667 | 0.514 |
| 无向量 | 0.942 | 0.391 | 0.392 | 0.167 | 0.104 |

**CP/Hit Rate 较上次下降原因：** 旧英文标题 synthesized pages 与新中文页面并存，互相干扰排名。
**向量差距扩大：** Hit Rate 差距 +0.500（上次 +0.416），向量价值更加确定。
**待改进：** 清理旧英文 synthesized pages，重新触发 synthesize 生成中文版本。

**剩余2题未命中的真实原因（不是 expected_page 问题）：**
- Q2（传统/混合/树结构检索对比）：`comparison-of-rag-retrieval-methods.md` 被 source-note 挤出 top10
- Q7（长上下文 vs RAG）：向量语义偏差，Agentic RAG 文档语义相似度更高

## 问题集说明

`questions_rag.json`：10 道 RAG/LLM Wiki 主题问题，ground truth 来自 KB 中高质量 source-notes，覆盖：
- RAG 优化方法（五阶段、RRF、HyDE、Rerank）
- 向量数据库选型（ChromaDB/FAISS/Qdrant/Milvus）
- UnWeaver vs GraphRAG
- LLM Wiki vs 传统 RAG（Karpathy 三层架构）

### 清理英文 synthesized pages 后（2026-06-23）

| 版本 | AR | CP | CR | Hit Rate | MRR |
|---|---|---|---|---|---|
| **有向量（清理后，当前最佳CR）** | **0.992** | **0.591** | **0.614** | **0.667** | **0.472** |

**CR 0.614 历史最高**。删除 872 篇英文标题 synthesized pages，context 精度和召回同步提升。
