# Schema 版本独立设计

> 日期: 2026-07-20
> 状态: 草稿
> 类型: hotfix — 修正 schema 版本与应用版本耦合问题

## 问题

当前设计中，`schema_version` 和 `distill_version` 都存储应用版本号（如 `"0.4.8"`），与 `version.Version` 比较。这导致：

1. **应用升级但 schema 未变时，误报过时** — 如 v0.4.8 → v0.5.0 只改了 WebUI，schema 和 distill prompt 完全没变，但所有笔记被标记为过时
2. **无法区分"schema 变更"和"distill prompt 变更"** — 两者混为一谈
3. **语义不清晰** — `schema_version: "0.4.8"` 看不出 schema 实际改了几次

## 设计目标

- Schema 版本独立于应用版本，用简单整数表示
- 笔记记录 distill 时的 schema 版本
- 通过整数比较（`<`）判断是否需要重新蒸馏
- 最小化变更范围，不影响现有功能

## 设计

### 1. Schema 版本存储

**新增嵌入式文件**：`internal/kbinit/schema/VERSION`

内容为纯整数，如 `2`。随其他 schema 文件一起嵌入（`//go:embed schema/*`）。

**读取方式**：

```go
// internal/kbinit/init.go
func readSchemaVersion() (int, error) {
    data, err := schemaFS.ReadFile("schema/VERSION")
    if err != nil {
        return 0, err
    }
    var v int
    _, err = fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &v)
    return v, err
}
```

**config.yaml 变更**：

```yaml
schema_version: 2    # 整数，不再是应用版本字符串
```

`Config.SchemaVersion` 类型从 `string` 改为 `int`。

**向后兼容迁移**：

现有 config.yaml 的 `schema_version` 是字符串（如 `"0.4.8"`）。迁移逻辑：
- 如果值是纯数字 → 转为 int
- 如果值是非数字字符串（如 `"0.4.8"`）→ 视为 `0`（未版本化），下次升级时会更新

### 2. Init 和 UpgradeSchema 变更

**Init(kbRoot, force)**：

```go
func Init(kbRoot string, force bool) error {
    // ... 创建目录 ...

    if err := writeSchemaFiles(kbRoot, force); err != nil {
        return err
    }

    // 读取嵌入式 schema 版本
    schemaVer, err := readSchemaVersion()
    if err != nil {
        return fmt.Errorf("read schema version: %w", err)
    }

    cfg, err := config.Load(kbRoot)
    if err != nil {
        return fmt.Errorf("load config: %w", err)
    }

    // 仅在首次初始化或 force 时写入版本
    if cfg.SchemaVersion == 0 || force {
        cfg.SchemaVersion = schemaVer
        if err := config.Save(kbRoot, cfg); err != nil {
            return fmt.Errorf("save schema_version: %w", err)
        }
    }

    return nil
}
```

**UpgradeSchema(kbRoot)**：

```go
func UpgradeSchema(kbRoot string) ([]string, int, error) {
    oldCfg, err := config.Load(kbRoot)
    if err != nil {
        return nil, 0, fmt.Errorf("load config: %w", err)
    }
    oldVersion := oldCfg.SchemaVersion

    if err := writeSchemaFiles(kbRoot, true); err != nil {
        return nil, oldVersion, err
    }

    // 读取新的 schema 版本
    newVersion, err := readSchemaVersion()
    if err != nil {
        return nil, oldVersion, fmt.Errorf("read schema version: %w", err)
    }

    // 更新 config
    oldCfg.SchemaVersion = newVersion
    if err := config.Save(kbRoot, oldCfg); err != nil {
        return nil, oldVersion, fmt.Errorf("save config: %w", err)
    }

    // 收集更新文件列表
    schemaDir := filepath.Join(kbRoot, "schema")
    var updated []string
    _ = fs.WalkDir(schemaFS, "schema", func(path string, d fs.DirEntry, err error) error {
        if err != nil || d.IsDir() {
            return err
        }
        rel, _ := filepath.Rel(schemaDir, filepath.Join(kbRoot, path))
        updated = append(updated, rel)
        return nil
    })

    // Reindex
    if _, err := kb.KBReindex(kbRoot, false); err != nil {
        return updated, oldVersion, fmt.Errorf("reindex: %w", err)
    }

    return updated, oldVersion, nil
}
```

### 3. 笔记级版本标记

**distill 注入变更**：

`injectDistillVersion` 改名为 `injectSchemaVersion`，注入的字段名从 `distill_version` 改为 `schema_version`。

```go
// internal/distill/distill.go
func injectSchemaVersion(text string, ver int) string {
    // ... (逻辑同 injectDistillVersion，但值是整数) ...
    newFM := strings.TrimRight(fm, "\n") + "\nschema_version: " + strconv.Itoa(ver)
    // ...
}
```

**DistillFile 调用变更**：

```go
// 读取 config 中的 schema 版本
cfg, _ := config.Load(kbRoot)
generated = injectSchemaVersion(generated, cfg.SchemaVersion)
```

**frontmatter 示例**：

```yaml
---
type: source-note
title: "..."
sources:
  - raw/article.md
schema_version: 2
---
```

### 4. 数据库层变更

**documents 表**：

`distill_version TEXT` 列改名为 `schema_version INTEGER`。

```sql
CREATE TABLE IF NOT EXISTS documents (
    ...
    schema_version INTEGER DEFAULT 0  -- 0 = 未版本化（旧笔记）
);
```

**迁移逻辑**（`migrateDescription` 中）：

1. 如果 `distill_version` 列存在且 `schema_version` 不存在 → `ALTER TABLE documents RENAME COLUMN distill_version TO schema_version`（SQLite 3.25.0+ 支持，modernc.org/sqlite 满足）
2. 如果两者都不存在 → `ALTER TABLE documents ADD COLUMN schema_version INTEGER DEFAULT 0`
3. 回填：扫描 wiki source-notes frontmatter，如果有 `schema_version`（整数）→ UPDATE；如果有 `distill_version`（字符串，旧格式）→ 视为 0

**upsertDocument 变更**：

```go
// internal/kb/index.go
var schemaVersion int
if v, ok := parsed.RawFM["schema_version"]; ok {
    if n, ok := v.(int); ok {
        schemaVersion = n
    }
}
// INSERT/UPDATE 中使用 schemaVersion
```

### 5. 查询接口变更

**FindOutdatedNotes**：

```go
// internal/kb/outdated.go
func FindOutdatedNotes(db *sql.DB, currentSchemaVersion int) ([]string, error) {
    if currentSchemaVersion == 0 {
        return nil, nil  // 未版本化，不触发过时检测
    }
    rows, err := db.Query(
        "SELECT path, schema_version FROM documents WHERE layer='wiki' AND kind='source-note'")
    // ...
    for rows.Next() {
        var p string
        var v int
        if err := rows.Scan(&p, &v); err != nil {
            return nil, err
        }
        if v < currentSchemaVersion {
            paths = append(paths, p)
        }
    }
    return paths, rows.Err()
}
```

**版本语义**：
- `0` = 未版本化（旧笔记）→ 视为过时
- `1` = schema 版本 1 → 如果当前是 2，则过时
- `2` = schema 版本 2 → 如果当前是 2，则不过时

### 6. WebUI 变更

**Schema 升级横幅**：

```go
// internal/webui/api.go — handleSchemaStatus
writeJSON(w, map[string]interface{}{
    "current_version": version.Version,
    "schema_version":  cfg.SchemaVersion,
    "outdated":        cfg.SchemaVersion < embeddedSchemaVersion,
})
```

`embeddedSchemaVersion` 从嵌入式 `schema/VERSION` 读取，启动时缓存。

**过时笔记横幅**：

```go
// handleDistillOutdated
writeJSON(w, map[string]interface{}{
    "count":           len(paths),
    "current_version": cfg.SchemaVersion,  // 整数
})
```

**Lint 增强**：

```go
// KBLint 中
outdated, err := FindOutdatedNotes(db, cfg.SchemaVersion)
```

### 7. 发布流程变更

每次发布时，检查 schema 文件是否有变更：
- 有变更 → `schema/VERSION` +1，提交
- 无变更 → 不动 `schema/VERSION`

这比当前设计（每次发布都触发过时检测）更精确。

## 文件变更

| 文件 | 变更内容 |
|------|----------|
| `internal/kbinit/schema/VERSION` | **新增** — schema 版本号 |
| `internal/kbinit/init.go` | 读取 VERSION 文件，`SchemaVersion` 改为 int |
| `internal/config/config.go` | `SchemaVersion` 类型从 `string` 改为 `int` |
| `internal/config/config.go` | 迁移逻辑：非数字字符串 → 0 |
| `internal/distill/distill.go` | `injectDistillVersion` → `injectSchemaVersion`，注入整数 |
| `internal/kb/schema.go` | `distill_version TEXT` → `schema_version INTEGER` |
| `internal/kb/db.go` | 迁移：RENAME COLUMN 或 ADD COLUMN + 回填 |
| `internal/kb/index.go` | `upsertDocument` 提取 `schema_version` 整数 |
| `internal/kb/outdated.go` | `FindOutdatedNotes` 参数改为 int，比较改为 `<` |
| `internal/kb/service.go` | `KBLint` 调用 `FindOutdatedNotes(db, cfg.SchemaVersion)` |
| `internal/webui/api.go` | `handleSchemaStatus` 用嵌入式版本比较 |
| `internal/webui/api.go` | `handleDistillOutdated` 返回 int 版本 |
| `internal/webui/api.go` | `handleDistillRefreshOutdated` 读取 config 版本 |

## 实施顺序

1. **schema/VERSION 文件** — 新增嵌入式文件 + `readSchemaVersion()` 函数
2. **Config 迁移** — `SchemaVersion` 类型 string → int + 向后兼容迁移
3. **Init/UpgradeSchema** — 使用 `readSchemaVersion()` 写入 config
4. **distill 注入** — `injectSchemaVersion` 替换 `injectDistillVersion`
5. **数据库迁移** — RENAME COLUMN + 回填 + upsertDocument
6. **查询接口** — `FindOutdatedNotes` 参数和比较逻辑
7. **WebUI** — 横幅和 API 端点适配
8. **Lint** — 适配新接口

## 测试

- `readSchemaVersion()` 正确读取嵌入式版本
- Config 迁移：`"0.4.8"` → `0`，`"2"` → `2`
- `injectSchemaVersion` 注入整数到 frontmatter
- 数据库迁移：`distill_version` → `schema_version`，旧值转为 0
- `FindOutdatedNotes(db, 2)` 返回 `schema_version < 2` 的笔记
- `FindOutdatedNotes(db, 0)` 返回 nil（未版本化不触发）
- WebUI 横幅：嵌入式版本 > config 版本时显示升级提示
- Lint：`outdated_distill` 警告基于 schema 版本比较
