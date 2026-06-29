# 知识管道

原始文档在 Agent 能搜索之前，需经过提炼管道处理。

## 第一步 — 提炼（自动）

将任意 Markdown 文件放入 `raw/`，`piekbs serve` 的 watcher 自动运行提炼 + 索引。

LLM 将结构化 source-note 提取到 `wiki/source-notes/`，包含：
- `key_claims`：内嵌别名和跨语言等价词（ALIAS RULE）— 确保 FTS 匹配所有查询变体
- `【实体|类型】` 格式的命名实体标注
- `related_to`、`supports`、`contradicts` 链接 — 驱动搜索结果中的 `related` 字段
- `authority`（1–5）和 `doc_type` 元数据

## 第二步 — 综合（按需）

```bash
piekbs synthesize --topic "RAG"
```

当某个主题积累了足够多的 source-note 后，从中生成 concept / comparison / decision 页面。

来源少于 2 个的页面进入 `wiki/<type>/_draft/`，不会被索引，直到补充更多来源。

```bash
# 知识空白分析
piekbs synthesize --gaps --topic "RAG"
```

## 第三步 — 搜索

Agent 通过 MCP 使用 `kb_search` + `kb_page`。搜索是纯 FTS（SQLite FTS5 + BM25 评分），无需向量模型。

## 文件格式支持

| 格式 | 处理方式 |
|---|---|
| `.md`、`.txt` | 直接提炼 |
| PDF、Word、Excel、PPT、HTML | 通过 `markitdown` 转换后提炼 |
| Agent 转换的内容 | 写入 `raw/converted/` 跳过转换步骤 |

安装 `markitdown` 以支持二进制文件：

```bash
pip install markitdown
```
