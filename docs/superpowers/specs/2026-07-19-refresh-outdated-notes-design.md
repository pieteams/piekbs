# 重跑过时笔记 设计（简化版）

> Issue: [#26](https://github.com/pieteams/piekbs/issues/26)（#4 + #5）
> 日期: 2026-07-19
> 状态: 草稿（v3 — 简化版）
> 前置依赖: schema 版本管理（PR #35，已合入）

## 问题

PR #35 在 distill 输出中注入了 `distill_version` 字段，但：
1. **无法批量定位过时笔记** — 没有查询接口，只能逐个读文件检查
2. **无法重跑过时笔记** — 旧笔记的 prompt/schema 可能已过时，需要重新 distill
3. **Lint 检查不到版本过时** — 只检查 missing_field 和 broken_source

## 范围

1. documents 表新增 `distill_version` 列 + 迁移
2. 重跑过时笔记（WebUI 横幅 + 入队重跑）
3. Lint 新增 `outdated_distill` 检查项

## 与 v2 的差异

v2 设计包含以下不必要的复杂度，本版本去掉了：

| 去掉的部分 | 原因 |
|---|---|
| `semver` 依赖 | `distill_version` 只由 `injectDistillVersion` 写入，值就是 `version.Version`。只需判断是否相等，不需要语义化版本排序。字符串 `!=` 即可。 |
| `outdatedCount` 缓存 | `FindOutdatedNotes` 是简单 SELECT，对几百条记录的 KB 来说很快，直接查询即可。缓存引入了"何时刷新"的额外问题。 |
| `Lint` 函数签名变更 | `KBLint`（service.go:283）已经通过 `GlobalDB` 拿到了 DB，直接在 `KBLint` 里加 outdated 检查，不需要改 `Lint(kbRoot)` 的签名。 |
| raw 路径推导逻辑 | 只用 `sources` 字段。sources 是 `DistillFile` 写入的，正常 distill 过的笔记都有。为空则跳过。 |

## 设计

### 1. documents 表新增 distill_version 列

在 `internal/kb/schema.go` 的 documents 表定义中新增：

```sql
CREATE TABLE IF NOT EXISTS documents (
    id TEXT PRIMARY KEY,
    path TEXT NOT NULL,
    layer TEXT NOT NULL,
    kind TEXT,
    title TEXT,
    description TEXT,
    content TEXT NOT NULL,
    content_hash TEXT NOT NULL,
    source_uri TEXT,
    updated_at INTEGER NOT NULL,
    authority INTEGER NOT NULL DEFAULT 3,
    doc_timestamp INTEGER NOT NULL DEFAULT 0,
    distill_version TEXT  -- NULL = 未记录版本（旧笔记）
);
```

**迁移逻辑**（在 `internal/kb/db.go` 的 `migrateDescription` 中追加）：

1. `ALTER TABLE documents ADD COLUMN distill_version TEXT`（如果列不存在）
2. 扫描 `wiki/source-notes/` 下所有 .md 文件
3. 对每个文件，用 `ParseMarkdownFile` 解析 frontmatter
4. 如果 `RawFM["distill_version"]` 存在，`UPDATE documents SET distill_version = ? WHERE path = ?`
5. 没有该字段的行保持 NULL

**版本语义**：
- `NULL` = 没有记录版本（旧笔记或手写页面）→ 视为过时
- `"0.4.7"` = 有明确版本的笔记 → 与当前版本字符串比较

**写入时机**：在 `internal/kb/index.go` 的 `upsertDocument` 函数中，解析 frontmatter 后提取 `distill_version` 并写入 documents 表。这样 `DistillFile` 完成后调用 `IndexFiles` 时会自动写入版本，无需手动 UPDATE。

INSERT/UPDATE 语句中加入 `distill_version` 字段。

**查询接口**：

```go
// FindOutdatedNotes 返回 wiki source-notes 中 distill_version 为 NULL
// 或与 currentVersion 不同的路径。
// currentVersion 为 "dev" 时返回 nil（开发模式不触发过时检测）。
func FindOutdatedNotes(db *sql.DB, currentVersion string) ([]string, error) {
    if currentVersion == "dev" {
        return nil, nil
    }
    rows, err := db.Query(
        "SELECT path, distill_version FROM documents WHERE layer='wiki' AND kind='source-note'")
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var paths []string
    for rows.Next() {
        var p string
        var v sql.NullString
        if err := rows.Scan(&p, &v); err != nil {
            return nil, err
        }
        if !v.Valid || v.String != currentVersion {
            paths = append(paths, p)
        }
    }
    return paths, rows.Err()
}
```

无需 semver 库，纯字符串比较。

### 2. 重跑过时笔记

**WebUI 横幅**（与 schema 升级横幅并列）：

```
⚠ 3 个笔记需要更新 — distill 版本已过期。 [重跑过时笔记] [忽略]
```

**API 端点**：

- `GET /api/distill/outdated` — 返回 `{count: N, current_version: "..."}`
- `POST /api/distill/refresh-outdated` — 将过时笔记入队重跑

**重跑逻辑**（`POST /api/distill/refresh-outdated`）：

1. 调用 `FindOutdatedNotes(db, version.Version)` 获取过时路径列表
2. 对每个路径：
   - 读取 frontmatter 中的 `sources` 字段（列表），取第一个元素作为 raw 路径（格式如 `raw/foo/bar.md`）
   - 如果 sources 为空或第一个元素为空 → 跳过，记录警告
   - 如果 raw 文件存在（相对于 kbRoot）→ 删除旧的 wiki 文件（`os.Remove`），将 raw 路径去掉 `raw/` 前缀后插入 `distill_queue` 表（格式如 `foo/bar.md`，与 `Enqueue` 一致）
   - 如果 raw 文件不存在 → 删除孤立的 wiki 文件，从 documents 表中删除记录，记录警告，跳过
3. 返回 `{enqueued: N, cleaned: C, errors: [...]}`

**worker pool** 自动消费队列，重新 distill 每个文件（此时会注入新的 `distill_version`）。

**注意**：入队前必须删除旧的 wiki 文件，否则 `FindNewFiles` 会认为文件已存在而跳过。

**入队方式**：直接向 `distill_queue` 表 INSERT（与 `Enqueue` 函数相同的 SQL），而不是调用 `Enqueue`（因为 `Enqueue` 内部调用 `FindNewFiles`，而我们需要精确控制入队哪些文件）。

```sql
INSERT OR IGNORE INTO distill_queue (path, status, retry_count, queued_at, updated_at)
VALUES (?, 'pending', 0, ?, ?)
```

**并发安全**：

在 Server 中用 mutex 保护 refresh 操作，防止多次点击导致竞态：

```go
type Server struct {
    kbRoot     string
    importLark func(...)
    refreshMu  sync.Mutex  // 保护 refresh-outdated 操作
}
```

`TryLock` 失败时返回 409。前端点击按钮后禁用按钮，显示"处理中..."。

### 3. Lint 增强

在 `internal/kb/service.go` 的 `KBLint` 函数中（已有 DB 访问），新增 outdated 检查：

```go
// KBLint 中，cleanBrokenLinks 之后追加：
outdated, err := FindOutdatedNotes(db, version.Version)
if err == nil {
    for _, p := range outdated {
        warnings = append(warnings, LintWarning{
            Path:   p,
            Kind:   "outdated_distill",
            Detail: "distill version missing or outdated",
        })
    }
}
```

不需要改 `Lint(kbRoot)` 的签名。CLI 输出示例：

```
⚠ outdated_distill: wiki/source-notes/article.md — distill version missing or outdated
```

### 4. WebUI 横幅

在 `internal/webui/static/index.html` 中，在 schema 升级横幅下方添加过时笔记横幅，复用相同的模式（display:none + JS 检查 + 按钮交互）。

JS 逻辑：页面加载时调用 `GET /api/distill/outdated`，count > 0 时显示横幅。点击"重跑过时笔记"调用 `POST /api/distill/refresh-outdated`。

### 5. 错误处理

- **部分失败继续处理**：单个文件处理失败时记录警告并继续，最终返回 `{enqueued: N, cleaned: C, errors: [...]}`
- **文件删除失败**：跳过该文件，记录错误，不入队
- **数据库查询失败**：返回 500，不执行任何删除或入队操作

## 文件变更

| 文件 | 变更内容 |
|------|----------|
| `internal/kb/schema.go` | documents 表新增 `distill_version` 列 |
| `internal/kb/db.go` | 迁移逻辑（添加列 + 从 frontmatter 回填） |
| `internal/kb/index.go` | `upsertDocument` 中提取并写入 `distill_version` |
| `internal/kb/service.go` | `KBLint` 中新增 `outdated_distill` 检查 |
| `internal/webui/api.go` | 新增 `handleDistillOutdated` + `handleDistillRefreshOutdated` |
| `internal/webui/server.go` | 注册新路由 + Server 结构体新增 `refreshMu` |
| `internal/webui/static/index.html` | 过时笔记横幅 |

**不变的文件**：
- `internal/kb/lint.go` — 签名不变
- `internal/distill/distill.go` — 无需改动
- `cmd/piekbs/main.go` — `runLint` 调用不变（`KBLint` 内部处理）
- `go.mod` — 无新依赖

## 实施顺序

1. **数据库列 + 迁移** — schema.go + db.go
2. **IndexFiles 集成** — index.go 中自动写入 distill_version
3. **查询接口** — `FindOutdatedNotes` + WebUI API 端点
4. **重跑逻辑** — `POST /api/distill/refresh-outdated`（含并发保护）
5. **WebUI 横幅** — 前端展示 + 交互
6. **Lint 增强** — `KBLint` 中新增 `outdated_distill` 检查

## 测试

- 迁移：已有 wiki 文件的 distill_version 正确填充
- 版本比较：`FindOutdatedNotes` 正确处理字符串比较（相等则不过时，不同则过时）
- NULL 处理：distill_version 为 NULL 的笔记被视为过时
- dev 版本：`version.Version == "dev"` 时返回空列表
- 重跑：过时文件入队后 worker 重新 distill，新版本写入
- 孤立文件：raw 文件已删除的过时笔记应跳过并记录警告
- 并发：多次点击"重跑"按钮，第二次应返回 409
- Lint：`outdated_distill` 警告正确生成

## 后续规划

- #5 补充：中文源→英文笔记的语言规则违反检测
- #5 补充：`key_claims` 缺少 CJK alias 的检测
