# Schema 版本独立设计

> 日期: 2026-07-20
> 状态: 草稿
> 类型: hotfix — 修正 schema 版本与应用版本耦合问题

## 问题

当前 `schema_version` 和 `distill_version` 都存储应用版本号（如 `"0.4.8"`），与 `version.Version` 比较。应用升级但 schema 未变时会误报过时。

## 设计目标

- Schema 版本独立于应用版本，用简单整数表示
- 笔记记录 distill 时的 schema 版本
- 整数比较（`<`）判断是否需要重新蒸馏
- 最小化变更，向后兼容

## 设计

### 1. Schema 版本存储

**新增嵌入式文件**：`internal/kbinit/schema/VERSION`，内容为纯整数（如 `2`）。

**读取函数**：

```go
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

**Config 变更**：`SchemaVersion` 保持 `string` 类型，新增解析函数：

```go
func (c Config) SchemaVersionInt() int {
    v, err := strconv.Atoi(c.SchemaVersion)
    if err != nil {
        return 0
    }
    return v
}
```

现有 `"0.4.8"` → 解析失败 → 返回 0（未版本化）。`"2"` → 返回 2。

### 2. Init 和 UpgradeSchema

**Init**：调用 `readSchemaVersion()` 写入 config，仅在 `SchemaVersion == "0" || force` 时。

**UpgradeSchema**：调用 `readSchemaVersion()` 更新 config，返回值从 `string` 改为 `int`。

### 3. distill 注入

`injectDistillVersion` 改名为 `injectSchemaVersion`，字段名改为 `schema_version`，值为整数字符串。

```go
func injectSchemaVersion(text string, ver int) string {
    // 逻辑同 injectDistillVersion
    newFM := strings.TrimRight(fm, "\n") + "\nschema_version: " + strconv.Itoa(ver)
    // ...
}
```

DistillFile 中改为读取 `cfg.SchemaVersionInt()`。

### 4. 数据库迁移

**迁移逻辑**：

```sql
-- 1. 删除旧列（如果存在）
ALTER TABLE documents DROP COLUMN distill_version;
-- 2. 新增整数列
ALTER TABLE documents ADD COLUMN schema_version INTEGER DEFAULT 0;
```

注：SQLite 3.35.0+ 支持 DROP COLUMN（modernc.org/sqlite 满足）。

**回填**：扫描 wiki source-notes frontmatter，如果有 `schema_version`（int）→ UPDATE；如果有 `distill_version`（string，旧格式）→ 视为 0。

**upsertDocument**：提取 `schema_version` 整数写入。

### 5. 查询接口

**FindOutdatedNotes**：

```go
func FindOutdatedNotes(db *sql.DB, currentSchemaVersion int) ([]string, error) {
    if currentSchemaVersion == 0 {
        return nil, nil  // 未版本化，不触发
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

版本语义：`0` = 未版本化（过时），`1` < `2` → 过时，`2` == `2` → 不过时。

### 6. WebUI

**Schema 升级横幅**：`cfg.SchemaVersionInt() < embeddedSchemaVersion` 时显示。

**过时笔记横幅**：`count` 返回 `FindOutdatedNotes(db, cfg.SchemaVersionInt())` 的结果数。

**Lint**：`FindOutdatedNotes(db, cfg.SchemaVersionInt())`。

### 7. 发布流程

Schema 文件有变更 → `schema/VERSION` +1；无变更 → 不动。

## 文件变更

| 文件 | 变更 |
|------|------|
| `internal/kbinit/schema/VERSION` | **新增** |
| `internal/kbinit/init.go` | `readSchemaVersion()` + Init/UpgradeSchema 适配 |
| `internal/config/config.go` | `SchemaVersionInt()` 解析函数 |
| `internal/distill/distill.go` | `injectSchemaVersion` 替换 `injectDistillVersion` |
| `internal/kb/schema.go` | `distill_version` → `schema_version` |
| `internal/kb/db.go` | RENAME + 回填迁移 |
| `internal/kb/index.go` | `upsertDocument` 提取 `schema_version` |
| `internal/kb/outdated.go` | `FindOutdatedNotes` 参数改为 int |
| `internal/kb/service.go` | `KBLint` 适配 |
| `internal/webui/api.go` | 横幅和 API 适配 |

## 实施顺序

1. **schema/VERSION + Config 适配** — 新增文件 + `readSchemaVersion()` + `SchemaVersionInt()`
2. **Init/UpgradeSchema + distill 注入** — 使用新版本逻辑
3. **数据库迁移 + 查询接口** — RENAME + 回填 + `FindOutdatedNotes` 适配
4. **WebUI + Lint** — 横幅和 API 适配

## 测试

- `readSchemaVersion()` 读取正确
- `SchemaVersionInt()`：`"0.4.8"` → 0，`"2"` → 2
- `injectSchemaVersion` 注入整数字符串
- 数据库迁移：RENAME + 旧值转 `"0"`
- `FindOutdatedNotes(db, 2)` 返回版本 < 2 的笔记
- `FindOutdatedNotes(db, 0)` 返回 nil
- WebUI 横幅：嵌入式版本 > config 版本时显示
