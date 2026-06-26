# 红链检测与断链清理设计（32.2）

## 背景

WikiLoop 的 `links` 表记录文档间的关系（`related_to`、`supports`、`contradicts`）。蒸馏时 LLM 写入这些字段，但存在三类质量问题：

1. **概念名断链**：LLM 写的是概念名（如 `"数字经济"`），不是文件路径，找不到对应文档
2. **路径断链**：路径格式正确，但对应文件还不存在（文档尚未创建/蒸馏）
3. **占位符断链**：蒸馏时未填实际值，留下注释或空数组（如 `[]  # related wiki pages`）

**实测数据（2026-06-26）：**
```
related_to  总计 39 条，全部断链
supports    总计  8 条，全部断链
contradicts 总计  2 条，全部断链
```

所有 links 记录均为断链，`GraphExpand` 和 `related` 字段完全无效。

---

## 设计目标

- `kb_lint` 检测三种断链并区分类型
- **三种断链最终都清理 links 表记录**（保持 links 表干净，GraphExpand 不走断链边）
- 概念名断链额外写入 `wiki/index/red_links.json` 作为知识缺口建议
- WebUI 展示红链列表，用户可逐条删除

---

## 三种断链的处理方式

| 类型 | 判断方式 | 处理 |
|------|---------|------|
| **占位符** | target_doc_id 含 `#`、`[]`、为空，或以空白字符开头 | 静默删除 links 记录 |
| **路径断链** | target_doc_id 含 `/`（路径格式），但 documents 表无对应 id | 删除 links 记录 |
| **概念名断链** | target_doc_id 不含 `/`（纯概念名），documents 表无对应 id | 写入 red_links.json + 删除 links 记录 |

三种情况最终都删除 links 记录，区别在于概念名额外保留到 JSON 供用户参考。

---

## 数据流

```
kb_lint（或 kb_reindex 后自动运行）
  └─ 扫描 links 表中 related_to / supports / contradicts
       └─ 对每条记录，检查 target_doc_id 是否存在于 documents 表
            ├─ 存在 → 跳过（正常链接）
            │
            ├─ 占位符 → 删除 links 记录（静默）
            │
            ├─ 路径断链（含 /）→ 删除 links 记录
            │    └─ lint warning: kind="broken_related", detail=target_doc_id
            │
            └─ 概念名断链（不含 /）
                 ├─ 追加到 red_links.json（概念名 + 引用来源 + 引用次数）
                 ├─ 删除 links 记录
                 └─ lint warning: kind="missing_concept", detail=concept_name
```

---

## red_links.json 格式

路径：`{kbRoot}/wiki/index/red_links.json`

```json
[
  {
    "concept": "数字经济",
    "count": 3,
    "referenced_by": [
      "wiki/source-notes/data_gov/xxx.md",
      "wiki/source-notes/data_gov/yyy.md"
    ]
  },
  {
    "concept": "数据要素政策",
    "count": 1,
    "referenced_by": [
      "wiki/source-notes/data_gov/zzz.md"
    ]
  }
]
```

- 按 `count` 降序排列（高频缺口排前面）
- 每次 `kb_lint` 运行时**重新生成**（覆盖旧文件），不追加
- 文件不存在时 WebUI 返回空列表，不报错

---

## 实现机制：纯 SQL，两次查询

断链比较是直接的字符串相等：`links.target_doc_id` 与 `documents.id`（文档路径）做 LEFT JOIN，匹配不到即为断链。

**第一步：SELECT 拿断链列表（分类）**

```sql
SELECT
    l.rowid,
    l.relation,
    l.source_doc_id,
    l.target_doc_id,
    CASE
        WHEN trim(l.target_doc_id) = ''
          OR l.target_doc_id LIKE '#%'
          OR l.target_doc_id LIKE '[%'      THEN 'placeholder'
        WHEN instr(l.target_doc_id, '/') > 0 THEN 'path'
        ELSE                                      'concept'
    END AS link_type
FROM links l
LEFT JOIN documents d ON d.id = l.target_doc_id
WHERE l.relation IN ('related_to', 'supports', 'contradicts')
  AND d.id IS NULL;
```

Go 代码遍历结果：
- `placeholder` → 静默跳过（只删除）
- `path` → 生成 `LintWarning{Kind: "broken_related"}`
- `concept` → 累计到 `RedLink` map，生成 `LintWarning{Kind: "missing_concept"}`

**第二步：DELETE 一次性清理所有断链**

```sql
DELETE FROM links
WHERE relation IN ('related_to', 'supports', 'contradicts')
  AND id NOT IN (
      SELECT l.id FROM links l
      JOIN documents d ON d.id = l.target_doc_id
      WHERE l.relation IN ('related_to', 'supports', 'contradicts')
  );
```

整个逻辑只需两次 SQL，Go 层只做结果聚合和 JSON 写入。

---

## 组件设计

### 1. `internal/kb/lint.go`（改动）

新增 `LintWarning.Kind` 枚举值：
- `"broken_related"`：路径格式但文件不存在
- `"missing_concept"`：概念名格式，知识缺口

新增函数：

```go
// cleanBrokenLinks 扫描 links 表，清理三种断链。
// 返回概念名断链列表（用于写入 red_links.json）。
func cleanBrokenLinks(db *sql.DB) ([]RedLink, []LintWarning, int, int, error)
// 返回值：redLinks, warnings, brokenPathCount, placeholderCount, err

// RedLink 代表一个被引用但尚未创建页面的概念。
type RedLink struct {
    Concept      string   `json:"concept"`
    Count        int      `json:"count"`
    ReferencedBy []string `json:"referenced_by"`
}
```

### 2. `internal/kb/service.go`（改动）

`KBLint` 在现有 lint 逻辑后调用 `cleanBrokenLinks`，将概念名断链写入 `red_links.json`：

```go
func writeRedLinks(kbRoot string, links []RedLink) error {
    // 按 count 降序排列，写入 wiki/index/red_links.json
}
```

`LintResult` 新增字段：

```go
type LintResult struct {
    Warnings     []LintWarning `json:"warnings"`
    Count        int           `json:"count"`
    RedLinks     []RedLink     `json:"red_links"`      // 概念名缺口
    BrokenLinks  int           `json:"broken_links"`   // 路径断链清理数量
    Placeholders int           `json:"placeholders"`   // 占位符清理数量
}
```

### 3. `internal/webui/api.go`（改动）

新增两个接口：

**`GET /api/red-links`** — 读取 red_links.json 返回列表：
```json
{"red_links": [...], "count": 5}
```

**`DELETE /api/red-links/:concept`** — 从 red_links.json 删除指定概念名条目（用户确认不需要后删除）。

### 4. WebUI（改动）

在 lint 页面（或状态页面）增加红链区块：

```
知识缺口（Red Links）  5 个概念被引用但尚未创建页面

  数字经济        被引用 3 次  [删除]
  数据要素政策    被引用 1 次  [删除]
  数据治理框架    被引用 1 次  [删除]
  ...
```

用户点击"删除"：从 JSON 移除该条，不影响知识库其他内容。

---

## 文件改动汇总

| 文件 | 类型 | 改动说明 |
|------|------|----------|
| `internal/kb/lint.go` | 改动 | 新增 `cleanBrokenLinks`、`RedLink`，扩展 `LintWarning.Kind`；分类逻辑在 SQL CASE 里 |
| `internal/kb/service.go` | 改动 | `KBLint` 调用 `cleanBrokenLinks`，写 red_links.json，`LintResult` 新增字段 |
| `internal/webui/api.go` | 改动 | 新增 `GET/DELETE /api/red-links` |
| `internal/webui/static/` | 改动 | lint 页面增加红链展示区块 |

**不需要：**
- ~~新建数据库表~~（JSON 文件足够）
- ~~schema.go / db.go 改动~~

---

## 关键约束

**links 表最终全部清理：** 三种断链都删除 links 记录，保证 `GraphExpand` 和 TagExpand 不走无效边。

**red_links.json 每次重新生成：** 不追加，避免历史缺口无限积累。用户删除某条后，下次 lint 如果该概念仍被引用会重新出现；若对应文档已创建则不再出现。

**占位符静默清理：** 不产生 lint warning，避免噪音——占位符是蒸馏质量问题，不是用户需要关注的知识缺口。

**路径断链产生 warning：** 路径格式正确但文件不存在，说明蒸馏时引用了尚未创建的文档，值得用户知晓（可能需要补充蒸馏）。

**WebUI 删除只改 JSON：** 不影响 links 表（已经清理过了）、不影响 source-note 的原始 frontmatter 文件。
