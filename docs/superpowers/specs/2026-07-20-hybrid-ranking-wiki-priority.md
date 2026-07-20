# Hybrid 排名修复：Wiki 优先级

> Issue: [#25](https://github.com/pieteams/piekbs/issues/25)
> 日期: 2026-07-20
> 状态: 草稿

## 问题

MCP `kb_search` 的混合搜索路径（`HybridRank`）只按 `HybridScore` 排序，不含 `wiki_priority` 和 `coverage`。导致：

- 蒸馏的 wiki 笔记排在 raw 文件之下，即使 wiki 笔记覆盖更多关键词
- 时间衰减让新的 raw 文件胜过旧的 wiki 笔记

## 设计目标

- Wiki 笔记始终优先于 raw 文件（`wiki_priority`）
- 覆盖更多关键词的结果优先（`coverage`）
- `HybridScore` 作为最终 tie-breaker（替代 `FTSRank`）
- 改动最小，复用现有 `sortResults` 逻辑

## 设计

### 核心变更

在 `internal/kb/search.go` 的 hybrid 路径中，将 `sort.Slice` 替换为 4 键排序：

**当前代码**（line 462-465）：
```go
sort.Slice(results, func(i, j int) bool {
    return results[i].HybridScore > results[j].HybridScore
})
```

**替换为**：
```go
sort.SliceStable(results, func(i, j int) bool {
    a, b := results[i], results[j]
    if a.WikiPriority != b.WikiPriority {
        return a.WikiPriority > b.WikiPriority
    }
    if a.MatchPhase != b.MatchPhase {
        return a.MatchPhase < b.MatchPhase
    }
    if a.Coverage != b.Coverage {
        return a.Coverage > b.Coverage
    }
    return a.HybridScore > b.HybridScore
})
```

排序键：WikiPriority → MatchPhase → Coverage → HybridScore（替代 FTSRank）。

### Coverage 计算

hybrid 路径的 `HybridRank` 函数当前不计算 `Coverage`。需要在排序前计算：

```go
// Compute Coverage for hybrid results.
for i := range results {
    if len(keywords) > 0 {
        text := strings.ToLower(results[i].Title + " " + results[i].Snippet)
        count := 0
        for _, kw := range keywords {
            if strings.Contains(text, strings.ToLower(kw)) {
                count++
            }
        }
        results[i].Coverage = count
    }
}
```

`HybridRank` 函数签名需要新增 `keywords []string` 参数。

### 函数签名变更

**当前**：
```go
func HybridRank(fts []SearchResult, graph map[string]float64, conflicts []Conflict, embedder Embedder) []SearchResult
```

**改为**：
```go
func HybridRank(fts []SearchResult, graph map[string]float64, conflicts []Conflict, embedder Embedder, keywords []string) []SearchResult
```

所有调用方需要传入 `keywords`。

### 排序逻辑提取（可选优化）

`sortResults` 和新的 hybrid 排序逻辑相同（只是最后 tie-breaker 不同）。可以提取为通用函数：

```go
// sortWithPriority sorts results by WikiPriority → MatchPhase → Coverage → finalKey.
func sortWithPriority(results []SearchResult, keywords []string, finalKey func(a, b SearchResult) bool) {
    // Compute Coverage
    for i := range results {
        if len(keywords) > 0 {
            text := strings.ToLower(results[i].Title + " " + results[i].Snippet)
            count := 0
            for _, kw := range keywords {
                if strings.Contains(text, strings.ToLower(kw)) {
                    count++
                }
            }
            results[i].Coverage = count
        }
    }
    sort.SliceStable(results, func(i, j int) bool {
        a, b := results[i], results[j]
        if a.WikiPriority != b.WikiPriority {
            return a.WikiPriority > b.WikiPriority
        }
        if a.MatchPhase != b.MatchPhase {
            return a.MatchPhase < b.MatchPhase
        }
        if a.Coverage != b.Coverage {
            return a.Coverage > b.Coverage
        }
        return finalKey(a, b)
    })
}
```

调用方：
- `sortResults` → `sortWithPriority(results, keywords, func(a, b SearchResult) bool { return a.FTSRank < b.FTSRank })`
- `HybridRank` → `sortWithPriority(results, keywords, func(a, b SearchResult) bool { return a.HybridScore > b.HybridScore })`

## 文件变更

| 文件 | 变更 |
|------|------|
| `internal/kb/search.go` | `HybridRank` 签名加 `keywords`，排序改为 4 键 |
| `internal/kb/search.go` | 提取 `sortWithPriority`（可选） |
| `internal/kb/search_test.go` | 新增测试验证 wiki 优先级 |

## 测试

- Wiki 笔记在 hybrid 结果中排在 raw 文件之上
- Coverage 高的结果排在 Coverage 低的结果之上
- HybridScore 作为 tie-breaker（相同 wiki/coverage 时，分数高的排前面）
