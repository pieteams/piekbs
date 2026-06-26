# Red Links Detection Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend `kb_lint` to detect and clean broken links in the `links` table, and expose knowledge-gap "red links" (concept names with no wiki page) via `red_links.json` and a WebUI panel.

**Architecture:** Two SQL queries do all the work — one SELECT to classify broken links, one DELETE to clean them. Go aggregates concept-name results into `wiki/index/red_links.json`. The WebUI reads this file via two new API endpoints and shows a dismissable list on the home page.

**Tech Stack:** Go 1.25+, SQLite FTS5, build tag `fts5`, existing `kb` package patterns.

## Global Constraints

- Build tag: all Go files in `internal/kb/` and `internal/webui/` use `//go:build fts5`
- All tests run with: `rtk go test -tags fts5 ./internal/kb/... ./internal/webui/...`
- No new dependencies; no CGO
- `LintWarning.Kind` new values: `"broken_related"` and `"missing_concept"` (exact strings)
- `red_links.json` path: `{kbRoot}/wiki/index/red_links.json` (exact)
- JSON field names: `concept`, `count`, `referenced_by` (exact, lowercase)
- API routes: `GET /api/red-links` and `DELETE /api/red-links` with query param `?concept=<name>` (exact)
- `lint` has a side-effect: it mutates the `links` table and writes `red_links.json` — this must be documented in code comments
- Placeholder pattern for SQL classification: target starts with `#`, `[`, or is empty after trim → `placeholder`; contains `/` → `path`; otherwise → `concept`

---

### Task 1: `cleanBrokenLinks` in `lint.go` + tests

**Files:**
- Modify: `internal/kb/lint.go`
- Modify: `internal/kb/lint_test.go`

**Interfaces:**
- Produces:
  ```go
  type RedLink struct {
      Concept      string   `json:"concept"`
      Count        int      `json:"count"`
      ReferencedBy []string `json:"referenced_by"`
  }

  // cleanBrokenLinks scans the links table for broken related_to/supports/contradicts
  // entries and deletes them. Returns concept-name broken links as RedLink slice
  // (for red_links.json), path-broken-link warnings, and counts of deleted rows.
  // Side effect: deletes rows from links table.
  func cleanBrokenLinks(db *sql.DB) (redLinks []RedLink, warnings []LintWarning, brokenPaths int, placeholders int, err error)
  ```

- [ ] **Step 1: Write failing tests**

Add to `internal/kb/lint_test.go`:

```go
// TestCleanBrokenLinks_Placeholder verifies placeholder entries are silently
// deleted with no warning — these are template noise, not actionable gaps.
func TestCleanBrokenLinks_Placeholder(t *testing.T) {
	dir := setupTestKB(t)
	db, err := OpenDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Insert a source doc so FK is satisfied.
	db.Exec(`INSERT OR IGNORE INTO documents (id,path,layer,kind,title,description,content,content_hash,updated_at,authority,doc_timestamp) VALUES ('wiki/a.md','wiki/a.md','wiki','source-note','A','','body','h',1,3,0)`)
	// Placeholder target
	db.Exec(`INSERT INTO links (source_doc_id,target_doc_id,relation,confidence) VALUES ('wiki/a.md','[]  # related wiki pages','related_to',1.0)`)

	redLinks, warnings, _, placeholders, err := cleanBrokenLinks(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(redLinks) != 0 {
		t.Errorf("want 0 redLinks, got %d", len(redLinks))
	}
	if len(warnings) != 0 {
		t.Errorf("want 0 warnings for placeholder, got %v", warnings)
	}
	if placeholders != 1 {
		t.Errorf("want placeholders=1, got %d", placeholders)
	}

	// Verify row was deleted.
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM links WHERE relation='related_to'`).Scan(&n)
	if n != 0 {
		t.Errorf("want 0 links after cleanup, got %d", n)
	}
}

// TestCleanBrokenLinks_PathAndConcept verifies path broken links produce
// broken_related warnings and concept broken links produce missing_concept
// warnings and are collected as RedLinks.
func TestCleanBrokenLinks_PathAndConcept(t *testing.T) {
	dir := setupTestKB(t)
	db, err := OpenDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	db.Exec(`INSERT OR IGNORE INTO documents (id,path,layer,kind,title,description,content,content_hash,updated_at,authority,doc_timestamp) VALUES ('wiki/src.md','wiki/src.md','wiki','source-note','S','','body','h',1,3,0)`)
	// Path broken link
	db.Exec(`INSERT INTO links (source_doc_id,target_doc_id,relation,confidence) VALUES ('wiki/src.md','wiki/concepts/missing-page.md','related_to',1.0)`)
	// Concept broken link (referenced twice from different sources)
	db.Exec(`INSERT OR IGNORE INTO documents (id,path,layer,kind,title,description,content,content_hash,updated_at,authority,doc_timestamp) VALUES ('wiki/src2.md','wiki/src2.md','wiki','source-note','S2','','body','h2',1,3,0)`)
	db.Exec(`INSERT INTO links (source_doc_id,target_doc_id,relation,confidence) VALUES ('wiki/src.md','数字经济','related_to',1.0)`)
	db.Exec(`INSERT INTO links (source_doc_id,target_doc_id,relation,confidence) VALUES ('wiki/src2.md','数字经济','supports',1.0)`)

	redLinks, warnings, brokenPaths, _, err := cleanBrokenLinks(db)
	if err != nil {
		t.Fatal(err)
	}
	if brokenPaths != 1 {
		t.Errorf("want brokenPaths=1, got %d", brokenPaths)
	}
	if len(redLinks) != 1 || redLinks[0].Concept != "数字经济" || redLinks[0].Count != 2 {
		t.Errorf("want 1 redLink '数字经济' count=2, got %+v", redLinks)
	}
	if !findWarning(warnings, "wiki/src.md", "broken_related", "wiki/concepts/missing-page.md") {
		t.Errorf("expected broken_related warning; got %+v", warnings)
	}
	if !findWarning(warnings, "wiki/src.md", "missing_concept", "数字经济") {
		t.Errorf("expected missing_concept warning; got %+v", warnings)
	}

	// All 3 broken rows deleted.
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM links`).Scan(&n)
	if n != 0 {
		t.Errorf("want 0 links after cleanup, got %d", n)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
rtk go test -tags fts5 ./internal/kb/ -run "TestCleanBrokenLinks" -v
```

Expected: FAIL — `cleanBrokenLinks` undefined.

- [ ] **Step 3: Implement `RedLink` type and `cleanBrokenLinks` in `lint.go`**

Add after the existing `isFMFieldAbsent` function:

```go
// RedLink represents a concept name referenced in related_to/supports/contradicts
// that has no corresponding wiki page. Used to surface knowledge gaps.
type RedLink struct {
	Concept      string   `json:"concept"`
	Count        int      `json:"count"`
	ReferencedBy []string `json:"referenced_by"`
}

// cleanBrokenLinks scans links table for broken related_to/supports/contradicts
// entries and deletes them all. Returns:
//   - redLinks: concept-name broken links (no "/" in target) for red_links.json
//   - warnings: LintWarnings for path broken links (broken_related) and concept
//     broken links (missing_concept)
//   - brokenPaths: count of path-format broken links deleted
//   - placeholders: count of placeholder entries deleted (silent)
//
// Side effect: deletes broken rows from the links table.
func cleanBrokenLinks(db *sql.DB) ([]RedLink, []LintWarning, int, int, error) {
	const selectSQL = `
SELECT l.rowid, l.relation, l.source_doc_id, l.target_doc_id,
  CASE
    WHEN trim(l.target_doc_id) = ''
      OR l.target_doc_id LIKE '#%'
      OR l.target_doc_id LIKE '[%'       THEN 'placeholder'
    WHEN instr(l.target_doc_id, '/') > 0 THEN 'path'
    ELSE                                      'concept'
  END AS link_type
FROM links l
LEFT JOIN documents d ON d.id = l.target_doc_id
WHERE l.relation IN ('related_to', 'supports', 'contradicts')
  AND d.id IS NULL`

	rows, err := db.Query(selectSQL)
	if err != nil {
		return nil, nil, 0, 0, err
	}
	defer rows.Close()

	conceptMap := make(map[string]*RedLink)
	var warnings []LintWarning
	brokenPaths, placeholders := 0, 0

	for rows.Next() {
		var rowid int64
		var relation, sourceDID, targetDID, linkType string
		if err := rows.Scan(&rowid, &relation, &sourceDID, &targetDID, &linkType); err != nil {
			return nil, nil, 0, 0, err
		}
		switch linkType {
		case "placeholder":
			placeholders++
		case "path":
			brokenPaths++
			warnings = append(warnings, LintWarning{
				Path:   sourceDID,
				Kind:   "broken_related",
				Detail: targetDID,
			})
		case "concept":
			rl, ok := conceptMap[targetDID]
			if !ok {
				rl = &RedLink{Concept: targetDID}
				conceptMap[targetDID] = rl
			}
			rl.Count++
			rl.ReferencedBy = append(rl.ReferencedBy, sourceDID)
			warnings = append(warnings, LintWarning{
				Path:   sourceDID,
				Kind:   "missing_concept",
				Detail: targetDID,
			})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, 0, 0, err
	}

	// Delete all broken rows in one statement.
	const deleteSQL = `
DELETE FROM links
WHERE relation IN ('related_to', 'supports', 'contradicts')
  AND id NOT IN (
    SELECT l.id FROM links l
    JOIN documents d ON d.id = l.target_doc_id
    WHERE l.relation IN ('related_to', 'supports', 'contradicts')
  )`
	if _, err := db.Exec(deleteSQL); err != nil {
		return nil, nil, 0, 0, err
	}

	// Convert concept map to sorted slice (by count desc).
	redLinks := make([]RedLink, 0, len(conceptMap))
	for _, rl := range conceptMap {
		redLinks = append(redLinks, *rl)
	}
	sort.Slice(redLinks, func(i, j int) bool {
		return redLinks[i].Count > redLinks[j].Count
	})
	return redLinks, warnings, brokenPaths, placeholders, nil
}
```

Also add `"sort"` to the import block in `lint.go`.

- [ ] **Step 4: Run tests to verify they pass**

```bash
rtk go test -tags fts5 ./internal/kb/ -run "TestCleanBrokenLinks" -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
rtk git add internal/kb/lint.go internal/kb/lint_test.go
rtk git commit -m "feat(kb): add cleanBrokenLinks — classify and delete broken related_to/supports/contradicts"
```

---

### Task 2: Wire `cleanBrokenLinks` into `KBLint` in `service.go` + write `red_links.json`

**Files:**
- Modify: `internal/kb/service.go`
- Modify: `internal/kb/service_test.go`

**Interfaces:**
- Consumes:
  ```go
  func cleanBrokenLinks(db *sql.DB) ([]RedLink, []LintWarning, int, int, error)
  ```
- Produces (updated `LintResult`):
  ```go
  type LintResult struct {
      Warnings     []LintWarning `json:"warnings"`
      Count        int           `json:"count"`
      RedLinks     []RedLink     `json:"red_links"`
      BrokenLinks  int           `json:"broken_links"`
      Placeholders int           `json:"placeholders"`
  }
  ```
- Produces (`red_links.json` at `{kbRoot}/wiki/index/red_links.json`)

- [ ] **Step 1: Write failing test**

Add to `internal/kb/service_test.go`:

```go
// TestKBLint_CleansBrokenLinks verifies that KBLint deletes broken links from
// the links table and writes red_links.json with concept-name gaps.
func TestKBLint_CleansBrokenLinks(t *testing.T) {
	dir := setupTestKB(t)
	db, err := OpenDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Insert a source doc and a concept-name broken link.
	db.Exec(`INSERT OR IGNORE INTO documents (id,path,layer,kind,title,description,content,content_hash,updated_at,authority,doc_timestamp) VALUES ('wiki/src.md','wiki/src.md','wiki','source-note','S','','body','h',1,3,0)`)
	db.Exec(`INSERT INTO links (source_doc_id,target_doc_id,relation,confidence) VALUES ('wiki/src.md','数字经济','related_to',1.0)`)
	db.Close()

	result, err := KBLint(dir)
	if err != nil {
		t.Fatal(err)
	}
	if result.RedLinks == nil || len(result.RedLinks) != 1 {
		t.Fatalf("want 1 RedLink, got %+v", result.RedLinks)
	}
	if result.RedLinks[0].Concept != "数字经济" {
		t.Errorf("want concept '数字经济', got %q", result.RedLinks[0].Concept)
	}

	// red_links.json must exist and be valid.
	jsonPath := filepath.Join(dir, "wiki", "index", "red_links.json")
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("red_links.json not written: %v", err)
	}
	var links []kb.RedLink
	if err := json.Unmarshal(data, &links); err != nil {
		t.Fatalf("invalid red_links.json: %v", err)
	}
	if len(links) != 1 || links[0].Concept != "数字经济" {
		t.Errorf("unexpected red_links.json content: %s", data)
	}
}
```

Note: this test is in package `kb` (internal), adjust import accordingly — the test file already uses `package kb`.

- [ ] **Step 2: Run test to verify it fails**

```bash
rtk go test -tags fts5 ./internal/kb/ -run "TestKBLint_CleansBrokenLinks" -v
```

Expected: FAIL — `LintResult` has no `RedLinks` field.

- [ ] **Step 3: Update `LintResult` in `service.go`**

Replace:
```go
// LintResult is returned by KBLint.
type LintResult struct {
	Warnings []LintWarning `json:"warnings"`
	Count    int           `json:"count"`
}
```

With:
```go
// LintResult is returned by KBLint.
type LintResult struct {
	Warnings     []LintWarning `json:"warnings"`
	Count        int           `json:"count"`
	RedLinks     []RedLink     `json:"red_links"`
	BrokenLinks  int           `json:"broken_links"`
	Placeholders int           `json:"placeholders"`
}
```

- [ ] **Step 4: Update `KBLint` in `service.go`**

Replace the existing `KBLint` function:

```go
// KBLint runs deterministic health checks over wiki pages and cleans broken
// links from the links table. Side effect: deletes broken related_to/supports/
// contradicts rows and writes wiki/index/red_links.json.
func KBLint(kbRoot string) (*LintResult, error) {
	AppendQueryLog(kbRoot, "kb_lint", "")

	// File-level lint (missing fields, broken sources).
	warnings, err := Lint(kbRoot)
	if err != nil {
		return nil, &KBError{Code: 500, Message: err.Error()}
	}
	if warnings == nil {
		warnings = []LintWarning{}
	}

	// Broken-link cleanup (requires DB).
	db, err := OpenDB(kbRoot)
	if err != nil {
		// Non-fatal: return file warnings even if DB unavailable.
		return &LintResult{Warnings: warnings, Count: len(warnings)}, nil
	}
	defer db.Close()

	redLinks, blWarnings, brokenPaths, placeholders, err := cleanBrokenLinks(db)
	if err != nil {
		return nil, &KBError{Code: 500, Message: "clean broken links: " + err.Error()}
	}
	warnings = append(warnings, blWarnings...)
	if redLinks == nil {
		redLinks = []RedLink{}
	}

	// Write red_links.json (overwrite each run).
	if err := writeRedLinks(kbRoot, redLinks); err != nil {
		return nil, &KBError{Code: 500, Message: "write red_links.json: " + err.Error()}
	}

	return &LintResult{
		Warnings:     warnings,
		Count:        len(warnings),
		RedLinks:     redLinks,
		BrokenLinks:  brokenPaths,
		Placeholders: placeholders,
	}, nil
}

// writeRedLinks writes redLinks as JSON to wiki/index/red_links.json,
// sorted by count descending. Creates the directory if needed.
func writeRedLinks(kbRoot string, redLinks []RedLink) error {
	dir := filepath.Join(kbRoot, "wiki", "index")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(redLinks)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "red_links.json"), data, 0o644)
}
```

Add `"encoding/json"` to the import block in `service.go` if not already present.

- [ ] **Step 5: Run tests**

```bash
rtk go test -tags fts5 ./internal/kb/ -v 2>&1 | tail -20
```

Expected: all tests PASS.

- [ ] **Step 6: Commit**

```bash
rtk git add internal/kb/service.go internal/kb/service_test.go
rtk git commit -m "feat(kb): wire cleanBrokenLinks into KBLint, write red_links.json"
```

---

### Task 3: WebUI API endpoints + frontend panel

**Files:**
- Modify: `internal/webui/api.go`
- Modify: `internal/webui/server.go`
- Modify: `internal/webui/static/index.html`

**Interfaces:**
- Consumes:
  - `GET /api/red-links` → reads `{kbRoot}/wiki/index/red_links.json`
  - `DELETE /api/red-links?concept=<name>` → removes one entry from JSON

- [ ] **Step 1: Add API handlers in `api.go`**

Add after `handleLint`:

```go
// handleRedLinks serves GET and DELETE for the red_links.json knowledge-gap list.
// GET /api/red-links — returns {"red_links":[...],"count":N}
// DELETE /api/red-links?concept=<name> — removes one concept from red_links.json
func (s *Server) handleRedLinks(w http.ResponseWriter, r *http.Request) {
	jsonPath := filepath.Join(s.kbRoot, "wiki", "index", "red_links.json")
	switch r.Method {
	case http.MethodGet:
		data, err := os.ReadFile(jsonPath)
		if os.IsNotExist(err) {
			writeJSON(w, map[string]interface{}{"red_links": []interface{}{}, "count": 0})
			return
		}
		if err != nil {
			kbErrToHTTP(w, &kb.KBError{Code: 500, Message: err.Error()})
			return
		}
		var links []kb.RedLink
		if err := json.Unmarshal(data, &links); err != nil {
			kbErrToHTTP(w, &kb.KBError{Code: 500, Message: "parse red_links.json: " + err.Error()})
			return
		}
		writeJSON(w, map[string]interface{}{"red_links": links, "count": len(links)})

	case http.MethodDelete:
		concept := strings.TrimSpace(r.URL.Query().Get("concept"))
		if concept == "" {
			kbErrToHTTP(w, &kb.KBError{Code: 400, Message: "concept query param required"})
			return
		}
		data, err := os.ReadFile(jsonPath)
		if os.IsNotExist(err) {
			writeJSON(w, map[string]interface{}{"ok": true})
			return
		}
		if err != nil {
			kbErrToHTTP(w, &kb.KBError{Code: 500, Message: err.Error()})
			return
		}
		var links []kb.RedLink
		if err := json.Unmarshal(data, &links); err != nil {
			kbErrToHTTP(w, &kb.KBError{Code: 500, Message: err.Error()})
			return
		}
		filtered := links[:0]
		for _, l := range links {
			if l.Concept != concept {
				filtered = append(filtered, l)
			}
		}
		out, _ := json.Marshal(filtered)
		if err := os.WriteFile(jsonPath, out, 0o644); err != nil {
			kbErrToHTTP(w, &kb.KBError{Code: 500, Message: err.Error()})
			return
		}
		writeJSON(w, map[string]interface{}{"ok": true})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
```

Add `"os"` to the import block in `api.go` if not already present.

- [ ] **Step 2: Register route in `server.go`**

In `server.go`, find the route registrations block and add after the lint route:

```go
mux.HandleFunc("/api/red-links", s.handleRedLinks)
```

- [ ] **Step 3: Add red-links panel to `index.html`**

In `index.html`, find the `lint-results` div and add after it:

```html
<div id="red-links-panel" style="margin-top:12px;display:none">
  <div style="font-weight:600;margin-bottom:6px;font-size:13px">
    知识缺口 / Knowledge Gaps
    <span id="red-links-count" style="font-weight:400;color:#888;font-size:12px"></span>
  </div>
  <div id="red-links-list"></div>
</div>
```

In the `doLint()` JavaScript function, after the existing lint result rendering, add:

```javascript
// Load red links after lint.
loadRedLinks();
```

Add a new function after `doLint()`:

```javascript
async function loadRedLinks() {
    const panel = document.getElementById('red-links-panel');
    const countEl = document.getElementById('red-links-count');
    const listEl = document.getElementById('red-links-list');
    try {
        const res = await fetch('/api/red-links');
        const data = await res.json();
        const links = data.red_links || [];
        if (links.length === 0) { panel.style.display = 'none'; return; }
        panel.style.display = '';
        countEl.textContent = '(' + links.length + ')';
        listEl.innerHTML = links.map(l =>
            `<div style="display:flex;align-items:center;gap:8px;margin:3px 0;font-size:12px">
              <span style="color:#cf222e">⬤</span>
              <span style="flex:1">${escHtml(l.concept)}</span>
              <span style="color:#888">引用 ${l.count} 次</span>
              <button onclick="deleteRedLink(${JSON.stringify(l.concept)})"
                style="padding:2px 8px;font-size:11px;border:1px solid #d0d7de;border-radius:4px;cursor:pointer;background:#fff">
                忽略
              </button>
            </div>`
        ).join('');
    } catch(e) { panel.style.display = 'none'; }
}

async function deleteRedLink(concept) {
    await fetch('/api/red-links?concept=' + encodeURIComponent(concept), {method:'DELETE'});
    loadRedLinks();
}
```

Also call `loadRedLinks()` in the initial page load (after `loadStatus()`):

```javascript
loadRedLinks();
```

- [ ] **Step 4: Build and verify**

```bash
rtk go build -tags fts5 ./cmd/wikiloop/
```

Expected: compiles with no errors.

- [ ] **Step 5: Commit**

```bash
rtk git add internal/webui/api.go internal/webui/server.go internal/webui/static/index.html
rtk git commit -m "feat(webui): add /api/red-links endpoints and knowledge-gap panel"
```

---

## Self-Review Checklist

- [x] **Spec coverage:** All three task areas from spec covered (cleanBrokenLinks, KBLint wiring, WebUI)
- [x] **Placeholder scan:** No TBD/TODO — all code blocks complete
- [x] **Type consistency:** `RedLink` defined in Task 1, used in Tasks 2 and 3 consistently; `LintResult.RedLinks []RedLink` matches throughout
- [x] **SQL DELETE correctness:** Uses `id NOT IN (SELECT l.id ... JOIN documents ...)` — selects only valid links to keep, deletes the rest
- [x] **Side effect documented:** `cleanBrokenLinks` and `KBLint` have comments stating the mutation side effect
- [x] **API route:** `DELETE /api/red-links?concept=<name>` matches spec exactly
- [x] **File path:** `wiki/index/red_links.json` matches spec exactly
- [x] **JSON fields:** `concept`, `count`, `referenced_by` match spec exactly
