# MCP 工具参考

PieKBS 为 Agent 暴露三个 MCP 工具。

## kb_search

用关键词或短语搜索知识库。

**参数：**

| 参数 | 类型 | 必填 | 描述 |
|---|---|---|---|
| `query` | string | 是 | 搜索关键词或短语 |
| `kind` | string | 否 | 过滤页面类型：`source-note`、`concept`、`comparison`、`decision` |
| `layer` | string | 否 | 过滤层：`wiki`、`raw`、`schema` |
| `limit` | number | 否 | 最大结果数（默认 10） |

**返回：** 按相关度排序的匹配页面列表，每条包含：
- `id` — 页面标识符，用于 `kb_page`
- `title`、`snippet` — 匹配内容预览
- `kind`、`layer` — 页面分类
- `related` — 关联文档，用于图谱导航

## kb_add

向知识库添加文本文档。

**参数：**

| 参数 | 类型 | 必填 | 描述 |
|---|---|---|---|
| `filename` | string | 是 | 相对于 `raw/` 的路径，支持任意子目录结构（如 `references/article.md`、`converted/report.md`）。Agent 提取的 PDF/Word/Excel/EPUB 内容使用 `converted/` 前缀。 |
| `content` | string | 是 | 文件内容（Markdown 或纯文本） |
| `source_url` | string | 否 | 原始来源 URL，写入文件顶部注释 |
| `overwrite` | boolean | 否 | 文件已存在时是否覆盖（默认 false） |

将内容写入 `raw/<filename>` 并触发增量索引。提炼在后台异步运行。

## kb_page

通过 ID 获取一个或多个页面的完整内容。

**参数：**

| 参数 | 类型 | 必填 | 描述 |
|---|---|---|---|
| `ids` | array | 是 | `kb_search` 结果中的文档 ID（1–5 个） |
| `full` | boolean | 否 | 返回完整不截断文本（仅对单个 ID 有效） |

**返回：** 每个请求页面的完整 Markdown 内容。
