# Agent 如何使用 PieKBS

Agent 通过两个 MCP 工具与 PieKBS 交互。

## kb_search

```
kb_search(query, limit?)
```

用关键词或短语搜索。每次调用最多返回 5 个 source-note 和 3 个 concept/comparison/decision 页面。每条结果包含 `related` 字段，列出关联文档供导航。

用**不同关键词**多次搜索，从多个角度覆盖同一主题。不要重复相同查询 — 换关键词或换角度。

## kb_page

```
kb_page(ids, full?)
```

通过 ID（来自 `kb_search` 结果）获取一个或多个页面的完整内容。一次最多传 5 个 ID，或对单个 ID 传 `full=true` 获取完整不截断的文本。

## 推荐工作流

```text
kb_search("关键词 A")              → 发现相关文档
kb_search("关键词 B")              → 换角度覆盖
kb_page(["id1", "id2", "id3"])    → 深度阅读最相关的文档
Agent 从找到的内容中自行综合答案
```

Agent 应该：
- 用不同关键词迭代搜索
- 跟随 `related` 链接进行多跳推理
- 跨来源交叉验证
- 自行形成结论

PieKBS 提供**原始材料** — 不生成答案。

## 查询扩展

搜索前，将查询扩展为别名、缩写和跨语言等价词。FTS 使用精确词项匹配。

示例：
- `"召回率"` → 同时搜索 `"recall"`、`"Context Recall"`、`"CR"`
- `"提炼"` → 同时搜索 `"distill"`、`"extract"`、`"summarize"`
