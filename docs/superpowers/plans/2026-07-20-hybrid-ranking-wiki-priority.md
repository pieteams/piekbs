# Hybrid 排名修复 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix hybrid ranking so wiki notes always rank above raw files, using 4-key sort (WikiPriority → MatchPhase → Coverage → HybridScore).

**Architecture:** Extract `sortWithPriority` helper that computes Coverage and sorts by 4 keys. Replace `sort.Slice` in both `SearchLayered` and `HybridRank` with this helper. Add `keywords` parameter to `HybridRank`.

**Tech Stack:** Go, SQLite

## Global Constraints

- WikiPriority is always the first sort key
- Coverage is computed from keywords matching Title+Snippet
- HybridScore replaces FTSRank as the final tie-breaker
- `sortResults` becomes a thin wrapper around `sortWithPriority`

---

### Task 1: Extract `sortWithPriority` and fix `SearchLayered`

**Files:**
- Modify: `internal/kb/search.go` (extract helper, update `sortResults` and `SearchLayered`)
- Test: `internal/kb/search_test.go`

**Interfaces:**
- Produces: `sortWithPriority(results []SearchResult, keywords []string, finalKey func(a, b SearchResult) bool)`
- `sortResults` becomes a wrapper calling `sortWithPriority` with `FTSRank` as finalKey

- [ ] **Step 1: Add `sortWithPriority` function**

In `internal/kb/search.go`, add before `sortResults`:

```go
// sortWithPriority sorts results by WikiPriority → MatchPhase → Coverage → finalKey.
// Coverage is computed by counting keyword hits in Title+Snippet.
func sortWithPriority(results []SearchResult, keywords []string, finalKey func(a, b SearchResult) bool) {
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

- [ ] **Step 2: Rewrite `sortResults` as wrapper**

Replace the existing `sortResults` function:

```go
func sortResults(results []SearchResult, keywords []string) {
	sortWithPriority(results, keywords, func(a, b SearchResult) bool {
		return a.FTSRank < b.FTSRank
	})
}
```

- [ ] **Step 3: Fix `SearchLayered` sorting**

In `SearchLayered`, replace the `sort.Slice` block (line 462-465) with:

```go
// Sort by WikiPriority → MatchPhase → Coverage → HybridScore.
keywords := strings.Fields(query)
sortWithPriority(results, keywords, func(a, b SearchResult) bool {
    return a.HybridScore > b.HybridScore
})
```

Note: `query` is already a parameter of `SearchLayered`. `strings.Fields` splits it into keywords.

- [ ] **Step 4: Write test for wiki priority**

In `internal/kb/search_test.go`, add:

```go
func TestSearchLayered_WikiPriority(t *testing.T) {
	dir := t.TempDir()
	db, err := OpenDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Insert a raw file and a wiki note with same content.
	db.Exec(`INSERT INTO documents (id, path, layer, kind, title, description, content, content_hash, updated_at, authority, doc_timestamp, schema_version)
		VALUES ('raw/test.md', 'raw/test.md', 'raw', '', 'Test RAG', '', 'RAG content', 'h1', 1, 3, 0, 0)`)
	db.Exec(`INSERT INTO documents (id, path, layer, kind, title, description, content, content_hash, updated_at, authority, doc_timestamp, schema_version)
		VALUES ('wiki/test.md', 'wiki/test.md', 'wiki', 'source-note', 'Test RAG', '', 'RAG content', 'h2', 1, 3, 0, 1)`)

	results, _, err := SearchLayered(db, dir, "RAG", nil, nil, 5, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) < 2 {
		t.Fatalf("expected >=2 results, got %d", len(results))
	}
	// Wiki should rank above raw.
	if results[0].Layer != "wiki" {
		t.Errorf("expected first result to be wiki, got %s", results[0].Layer)
	}
}
```

- [ ] **Step 5: Run tests**

```bash
rtk go test ./internal/kb/ -v -tags fts5 -run "TestSearchLayered|TestSortResults"
```

- [ ] **Step 6: Commit**

```bash
rtk git add internal/kb/search.go internal/kb/search_test.go
rtk git commit -m "fix(kb): use 4-key sort in SearchLayered hybrid path"
```

---

### Task 2: Fix `HybridRank` signature and sorting

**Files:**
- Modify: `internal/kb/search.go` (HybridRank signature + sorting)
- Modify: `internal/kb/search_test.go` (update test call)

**Interfaces:**
- `HybridRank(fts []SearchResult, graph map[string]float64, conflicts []Conflict, embedder Embedder, keywords []string) []SearchResult`

- [ ] **Step 1: Update `HybridRank` signature**

Change the function signature:

```go
func HybridRank(fts []SearchResult, graph map[string]float64, conflicts []Conflict, embedder Embedder, keywords []string) []SearchResult {
```

- [ ] **Step 2: Replace sorting in `HybridRank`**

Replace the `sort.Slice` block (line 530-533) with:

```go
sortWithPriority(results, keywords, func(a, b SearchResult) bool {
    return a.HybridScore > b.HybridScore
})
```

- [ ] **Step 3: Update test call**

In `internal/kb/search_test.go`, find the call to `HybridRank` and add the `keywords` parameter:

```go
result := HybridRank(fts, nil, nil, nil, []string{"test"})
```

- [ ] **Step 4: Run tests**

```bash
rtk go test ./internal/kb/ -v -tags fts5 -run TestHybridRank
```

- [ ] **Step 5: Commit**

```bash
rtk git add internal/kb/search.go internal/kb/search_test.go
rtk git commit -m "fix(kb): add keywords param to HybridRank for coverage sorting"
```

---

### Task 3: Full verification

- [ ] **Step 1: Run all tests**

```bash
rtk go test ./... -tags fts5
```

- [ ] **Step 2: Build and verify**

```bash
rtk go build -tags fts5 -o piekbs ./cmd/piekbs/
```
