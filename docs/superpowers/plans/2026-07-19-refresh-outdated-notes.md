# 重跑过时笔记 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enable batch detection and re-distillation of outdated wiki notes whose `distill_version` doesn't match the current binary version.

**Architecture:** Add `distill_version` column to the documents table, query it to find outdated notes, provide API endpoints for refresh, and surface staleness in both WebUI banner and Lint output.

**Tech Stack:** Go, SQLite (modernc.org/sqlite), net/http, embed (static files)

## Global Constraints

- No new external dependencies (no semver library)
- `Lint(kbRoot string)` signature unchanged — outdated check added inside `KBLint`
- `distill_version` values come from `version.Version` (injected by `injectDistillVersion`)
- `INSERT OR IGNORE` for idempotent queue inserts
- Frontend uses same banner pattern as existing `schema-banner`

---

### Task 1: Add `distill_version` column to schema and migration

**Files:**
- Modify: `internal/kb/schema.go:6-19` (documents table DDL)
- Modify: `internal/kb/db.go:62-92` (`OpenDB`) and `internal/kb/db.go:94-167` (`migrateDescription`)
- Test: `internal/kb/db_test.go`

**Interfaces:**
- Produces: `distill_version TEXT` column in documents table, nullable
- `migrateDescription(db *sql.DB, kbRoot string)` — new signature adds `kbRoot` for frontmatter backfill

- [ ] **Step 1: Update schema DDL**

In `internal/kb/schema.go`, add `distill_version TEXT` to the documents table definition (after `doc_timestamp`):

```sql
    doc_timestamp INTEGER NOT NULL DEFAULT 0,
    distill_version TEXT
);
```

- [ ] **Step 2: Change `migrateDescription` signature**

In `internal/kb/db.go`, change the signature from `migrateDescription(db *sql.DB) error` to `migrateDescription(db *sql.DB, kbRoot string) error`.

Update the call site in `OpenDB` (line 86):
```go
if err := migrateDescription(db, kbRoot); err != nil {
```

- [ ] **Step 3: Add `distill_version` migration and backfill**

Append to `migrateDescription` after the `doc_timestamp` block:

```go
// Migrate distill_version column.
hasDistillVersion := false
rows2, err := db.Query("PRAGMA table_info(documents)")
if err != nil {
    return err
}
defer rows2.Close()
for rows2.Next() {
    var cid int
    var name, ctype string
    var notnull int
    var dflt interface{}
    var pk int
    if err := rows2.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
        return err
    }
    if name == "distill_version" {
        hasDistillVersion = true
    }
}
if err := rows2.Err(); err != nil {
    return err
}
if !hasDistillVersion {
    if _, err := db.Exec("ALTER TABLE documents ADD COLUMN distill_version TEXT"); err != nil {
        return err
    }
    // Backfill from existing wiki source-notes frontmatter.
    notesDir := filepath.Join(kbRoot, "wiki", "source-notes")
    if _, statErr := os.Stat(notesDir); statErr == nil {
        filepath.Walk(notesDir, func(path string, info os.FileInfo, err error) error {
            if err != nil || info.IsDir() || filepath.Ext(path) != ".md" {
                return nil
            }
            data, readErr := os.ReadFile(path)
            if readErr != nil {
                return nil
            }
            parsed := ParseMarkdown(string(data))
            if v, ok := parsed.RawFM["distill_version"]; ok {
                if s, ok := v.(string); ok && s != "" {
                    rel, _ := filepath.Rel(kbRoot, path)
                    db.Exec("UPDATE documents SET distill_version = ? WHERE path = ?",
                        s, filepath.ToSlash(rel))
                }
            }
            return nil
        })
    }
}
```

- [ ] **Step 4: Write test for migration**

In `internal/kb/db_test.go`, add:

```go
func TestOpenDB_MigratesDistillVersionColumn(t *testing.T) {
    dir := t.TempDir()
    db, err := OpenDB(dir)
    if err != nil {
        t.Fatal(err)
    }
    defer db.Close()

    var found bool
    rows, _ := db.Query("PRAGMA table_info(documents)")
    defer rows.Close()
    for rows.Next() {
        var cid int
        var name, ctype string
        var notnull int
        var dflt interface{}
        var pk int
        rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk)
        if name == "distill_version" {
            found = true
        }
    }
    if !found {
        t.Error("distill_version column not found after migration")
    }
}
```

- [ ] **Step 5: Run tests**

```bash
rtk go test ./internal/kb/ -run TestOpenDB_Migrates -v -tags fts5
```

- [ ] **Step 6: Commit**

```bash
rtk git add internal/kb/schema.go internal/kb/db.go internal/kb/db_test.go
rtk git commit -m "feat(kb): add distill_version column with migration and backfill"
```

---

### Task 2: Write `distill_version` during indexing

**Files:**
- Modify: `internal/kb/index.go:119-172` (`upsertDocument` function)
- Test: `internal/kb/index_test.go`

**Interfaces:**
- Consumes: `distill_version` column from Task 1
- Produces: `distill_version` populated in DB whenever `IndexFiles` runs

- [ ] **Step 1: Extract `distill_version` from frontmatter**

In `internal/kb/index.go`, in `upsertDocument` after `parsed := ParseMarkdown(text)` (line 135), add:

```go
var distillVersion sql.NullString
if v, ok := parsed.RawFM["distill_version"]; ok {
    if s, ok := v.(string); ok && s != "" {
        distillVersion = sql.NullString{String: s, Valid: true}
    }
}
```

- [ ] **Step 2: Update INSERT/UPDATE SQL**

In the same function, update the SQL statement (line 154) to include `distill_version`:

```go
_, err = db.Exec(`
    INSERT INTO documents (id, path, layer, kind, title, description, content, content_hash, source_uri, updated_at, authority, doc_timestamp, distill_version)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    ON CONFLICT(id) DO UPDATE SET
        path=excluded.path, layer=excluded.layer, kind=excluded.kind,
        title=excluded.title, description=excluded.description, content=excluded.content,
        content_hash=excluded.content_hash, source_uri=excluded.source_uri,
        updated_at=excluded.updated_at, authority=excluded.authority,
        doc_timestamp=excluded.doc_timestamp, distill_version=excluded.distill_version
`, did, filepath.ToSlash(rel), layer, parsed.Kind, title, parsed.Description,
    text, h, nil, now, parsed.Authority, docTs, distillVersion)
```

- [ ] **Step 3: Write test**

In `internal/kb/index_test.go`, add:

```go
func TestUpsertDocument_StoresDistillVersion(t *testing.T) {
    dir := setupTestKB(t)
    db, err := OpenDB(dir)
    if err != nil {
        t.Fatal(err)
    }
    defer db.Close()

    content := "---\ntitle: Versioned Note\ndistill_version: 0.4.7\n---\nBody"
    os.WriteFile(filepath.Join(dir, "raw", "ver.md"), []byte(content), 0644)

    _, err = IndexFiles(db, dir)
    if err != nil {
        t.Fatal(err)
    }

    var v sql.NullString
    err = db.QueryRow("SELECT distill_version FROM documents WHERE id = ?", "raw/ver.md").Scan(&v)
    if err != nil {
        t.Fatal(err)
    }
    if !v.Valid || v.String != "0.4.7" {
        t.Errorf("expected 0.4.7, got %v", v)
    }
}
```

- [ ] **Step 4: Run tests**

```bash
rtk go test ./internal/kb/ -run TestUpsertDocument_StoresDistillVersion -v -tags fts5
```

- [ ] **Step 5: Commit**

```bash
rtk git add internal/kb/index.go internal/kb/index_test.go
rtk git commit -m "feat(kb): store distill_version during indexing"
```

---

### Task 3: Add `FindOutdatedNotes` query function

**Files:**
- Create: `internal/kb/outdated.go` (new file for `FindOutdatedNotes`)
- Test: `internal/kb/outdated_test.go`

**Interfaces:**
- Produces: `FindOutdatedNotes(db *sql.DB, currentVersion string) ([]string, error)`

- [ ] **Step 1: Create `internal/kb/outdated.go`**

```go
//go:build fts5

package kb

import (
    "database/sql"
)

// FindOutdatedNotes returns wiki source-notes whose distill_version is NULL
// or differs from currentVersion. Returns nil when currentVersion is "dev"
// (development mode skips staleness detection).
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

- [ ] **Step 2: Create `internal/kb/outdated_test.go`**

```go
//go:build fts5

package kb

import (
    "testing"
)

func TestFindOutdatedNotes_DevVersion(t *testing.T) {
    dir := setupTestKB(t)
    db, err := OpenDB(dir)
    if err != nil {
        t.Fatal(err)
    }
    defer db.Close()

    paths, err := FindOutdatedNotes(db, "dev")
    if err != nil {
        t.Fatal(err)
    }
    if paths != nil {
        t.Errorf("expected nil for dev version, got %v", paths)
    }
}

func TestFindOutdatedNotes_NullVersion(t *testing.T) {
    dir := setupTestKB(t)
    db, err := OpenDB(dir)
    if err != nil {
        t.Fatal(err)
    }
    defer db.Close()

    // Insert a wiki source-note without distill_version.
    db.Exec(`INSERT INTO documents (id, path, layer, kind, title, description, content, content_hash, updated_at, authority, doc_timestamp)
        VALUES ('wiki/old.md', 'wiki/old.md', 'wiki', 'source-note', 'Old', '', 'body', 'h1', 1, 3, 0)`)

    paths, err := FindOutdatedNotes(db, "1.0.0")
    if err != nil {
        t.Fatal(err)
    }
    if len(paths) != 1 || paths[0] != "wiki/old.md" {
        t.Errorf("expected [wiki/old.md], got %v", paths)
    }
}

func TestFindOutdatedNotes_MatchingVersion(t *testing.T) {
    dir := setupTestKB(t)
    db, err := OpenDB(dir)
    if err != nil {
        t.Fatal(err)
    }
    defer db.Close()

    db.Exec(`INSERT INTO documents (id, path, layer, kind, title, description, content, content_hash, updated_at, authority, doc_timestamp, distill_version)
        VALUES ('wiki/new.md', 'wiki/new.md', 'wiki', 'source-note', 'New', '', 'body', 'h2', 1, 3, 0, '1.0.0')`)

    paths, err := FindOutdatedNotes(db, "1.0.0")
    if err != nil {
        t.Fatal(err)
    }
    if len(paths) != 0 {
        t.Errorf("expected empty, got %v", paths)
    }
}

func TestFindOutdatedNotes_DifferentVersion(t *testing.T) {
    dir := setupTestKB(t)
    db, err := OpenDB(dir)
    if err != nil {
        t.Fatal(err)
    }
    defer db.Close()

    db.Exec(`INSERT INTO documents (id, path, layer, kind, title, description, content, content_hash, updated_at, authority, doc_timestamp, distill_version)
        VALUES ('wiki/old.md', 'wiki/old.md', 'wiki', 'source-note', 'Old', '', 'body', 'h3', 1, 3, 0, '0.3.0')`)

    paths, err := FindOutdatedNotes(db, "1.0.0")
    if err != nil {
        t.Fatal(err)
    }
    if len(paths) != 1 || paths[0] != "wiki/old.md" {
        t.Errorf("expected [wiki/old.md], got %v", paths)
    }
}
```

- [ ] **Step 3: Run tests**

```bash
rtk go test ./internal/kb/ -run TestFindOutdatedNotes -v -tags fts5
```

- [ ] **Step 4: Commit**

```bash
rtk git add internal/kb/outdated.go internal/kb/outdated_test.go
rtk git commit -m "feat(kb): add FindOutdatedNotes query"
```

---

### Task 4: Add WebUI API endpoints

**Files:**
- Modify: `internal/webui/api.go` (add two new handlers)
- Modify: `internal/webui/server.go:44-79` (register new routes + add `refreshMu`)
- Test: `internal/webui/api_test.go`

**Interfaces:**
- Consumes: `FindOutdatedNotes` from Task 3
- Produces: `GET /api/distill/outdated`, `POST /api/distill/refresh-outdated`

- [ ] **Step 1: Add `refreshMu` to Server struct**

In `internal/webui/server.go`, add `sync` import and update the struct:

```go
import (
    "context"
    "embed"
    "net/http"
    "sync"

    "github.com/pieteams/piekbs/internal/larkimport"
)

type Server struct {
    kbRoot     string
    importLark func(context.Context, string, string, string) (*larkimport.Result, error)
    refreshMu  sync.Mutex
}
```

- [ ] **Step 2: Register new routes**

In `Handler()` method (line 44), add after the schema routes:

```go
mux.HandleFunc("/api/distill/outdated", s.handleDistillOutdated)
mux.HandleFunc("/api/distill/refresh-outdated", s.handleDistillRefreshOutdated)
```

- [ ] **Step 3: Add `handleDistillOutdated` handler**

In `internal/webui/api.go`, add:

```go
// handleDistillOutdated returns the count of outdated source-notes. GET /api/distill/outdated
func (s *Server) handleDistillOutdated(w http.ResponseWriter, r *http.Request) {
    db, err := kb.GlobalDB(s.kbRoot)
    if err != nil {
        kbErrToHTTP(w, err)
        return
    }
    paths, err := kb.FindOutdatedNotes(db, version.Version)
    if err != nil {
        kbErrToHTTP(w, err)
        return
    }
    writeJSON(w, map[string]interface{}{
        "count":           len(paths),
        "current_version": version.Version,
    })
}
```

- [ ] **Step 4: Add `handleDistillRefreshOutdated` handler**

In `internal/webui/api.go`, add:

```go
// handleDistillRefreshOutdated enqueues outdated notes for re-distillation. POST /api/distill/refresh-outdated
func (s *Server) handleDistillRefreshOutdated(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "POST required", http.StatusMethodNotAllowed)
        return
    }
    if !s.refreshMu.TryLock() {
        w.WriteHeader(http.StatusConflict)
        writeJSON(w, map[string]interface{}{"error": "refresh already in progress"})
        return
    }
    defer s.refreshMu.Unlock()

    db, err := kb.GlobalDB(s.kbRoot)
    if err != nil {
        kbErrToHTTP(w, err)
        return
    }
    outdated, err := kb.FindOutdatedNotes(db, version.Version)
    if err != nil {
        kbErrToHTTP(w, err)
        return
    }

    var enqueued, cleaned int
    var errs []string
    now := time.Now().Unix()

    for _, notePath := range outdated {
        absPath := filepath.Join(s.kbRoot, filepath.FromSlash(notePath))
        data, readErr := os.ReadFile(absPath)
        if readErr != nil {
            errs = append(errs, notePath+": read error: "+readErr.Error())
            continue
        }
        parsed := kb.ParseMarkdown(string(data))

        // Get raw source path from frontmatter sources field.
        if len(parsed.Sources) == 0 || parsed.Sources[0] == "" {
            // No sources — orphan wiki file. Clean up.
            os.Remove(absPath)
            db.Exec("DELETE FROM documents WHERE id = ?", notePath)
            cleaned++
            continue
        }
        rawSource := parsed.Sources[0] // e.g. "raw/foo/bar.md"
        rawAbsPath := filepath.Join(s.kbRoot, filepath.FromSlash(rawSource))

        if _, statErr := os.Stat(rawAbsPath); os.IsNotExist(statErr) {
            // Raw file gone — orphan wiki file. Clean up.
            os.Remove(absPath)
            db.Exec("DELETE FROM documents WHERE id = ?", notePath)
            cleaned++
            continue
        }

        // Delete old wiki file, then enqueue raw for re-distill.
        if removeErr := os.Remove(absPath); removeErr != nil {
            errs = append(errs, notePath+": remove error: "+removeErr.Error())
            continue
        }
        // Strip "raw/" prefix for distill_queue path format.
        queuePath := strings.TrimPrefix(rawSource, "raw/")
        _, insertErr := db.Exec(
            `INSERT OR IGNORE INTO distill_queue (path, status, retry_count, queued_at, updated_at)
             VALUES (?, 'pending', 0, ?, ?)`,
            queuePath, now, now)
        if insertErr != nil {
            errs = append(errs, notePath+": enqueue error: "+insertErr.Error())
            continue
        }
        enqueued++
    }

    writeJSON(w, map[string]interface{}{
        "enqueued": enqueued,
        "cleaned":  cleaned,
        "errors":   errs,
    })
}
```

- [ ] **Step 5: Run tests**

```bash
rtk go test ./internal/webui/ -v -tags fts5
```

- [ ] **Step 6: Commit**

```bash
rtk git add internal/webui/api.go internal/webui/server.go
rtk git commit -m "feat(webui): add distill outdated API endpoints"
```

---

### Task 5: Add WebUI banner for outdated notes

**Files:**
- Modify: `internal/webui/static/index.html` (add banner HTML + JS)

**Interfaces:**
- Consumes: `GET /api/distill/outdated`, `POST /api/distill/refresh-outdated` from Task 4

- [ ] **Step 1: Add banner HTML**

In `internal/webui/static/index.html`, after the `schema-banner` div (after line 21), add:

```html
<div id="outdated-banner" style="display:none; background:#fff3e0; border-left:4px solid #ff9800; padding:8px 16px; margin:8px 0; border-radius:4px; font-size:14px;">
    <span id="outdated-banner-text"></span>
    <button id="outdated-refresh-btn" onclick="refreshOutdated()" style="margin-left:12px; padding:4px 12px; cursor:pointer;">重跑过时笔记</button>
    <button onclick="document.getElementById('outdated-banner').style.display='none'" style="margin-left:4px; padding:4px 12px; cursor:pointer;">忽略</button>
</div>
```

- [ ] **Step 2: Add JS functions**

In the `<script>` section of `index.html`, add:

```javascript
async function checkOutdated() {
    try {
        const resp = await fetch('/api/distill/outdated');
        const data = await resp.json();
        const banner = document.getElementById('outdated-banner');
        const text = document.getElementById('outdated-banner-text');
        if (data.count > 0) {
            text.textContent = `⚠ ${data.count} 个笔记需要更新 — distill 版本已过期。`;
            banner.style.display = 'block';
        } else {
            banner.style.display = 'none';
        }
    } catch (e) {
        console.error('check outdated failed:', e);
    }
}

async function refreshOutdated() {
    const btn = document.getElementById('outdated-refresh-btn');
    btn.disabled = true;
    btn.textContent = '处理中...';
    try {
        const resp = await fetch('/api/distill/refresh-outdated', { method: 'POST' });
        const data = await resp.json();
        if (resp.status === 409) {
            alert('正在处理中，请稍候');
        } else if (data.errors && data.errors.length > 0) {
            alert('部分文件处理失败：\n' + data.errors.join('\n'));
        } else {
            checkOutdated();
        }
    } finally {
        btn.disabled = false;
        btn.textContent = '重跑过时笔记';
    }
}
```

- [ ] **Step 3: Call `checkOutdated()` on page load**

In the existing `DOMContentLoaded` event listener (or add one), call `checkOutdated()` alongside `checkSchemaVersion()`.

- [ ] **Step 4: Commit**

```bash
rtk git add internal/webui/static/index.html
rtk git commit -m "feat(webui): add outdated notes banner"
```

---

### Task 6: Add `outdated_distill` lint check

**Files:**
- Modify: `internal/kb/service.go:283-323` (`KBLint` function)
- Test: `internal/kb/service_test.go`

**Interfaces:**
- Consumes: `FindOutdatedNotes` from Task 3
- Produces: `LintWarning{Kind: "outdated_distill"}` in lint output

- [ ] **Step 1: Add outdated check to `KBLint`**

In `internal/kb/service.go`, in `KBLint` after the `cleanBrokenLinks` block (after line 306), add:

```go
// Outdated distill version check.
outdated, outdatedErr := FindOutdatedNotes(db, version.Version)
if outdatedErr == nil {
    for _, p := range outdated {
        warnings = append(warnings, LintWarning{
            Path:   p,
            Kind:   "outdated_distill",
            Detail: "distill version missing or outdated",
        })
    }
}
```

- [ ] **Step 2: Write test**

In `internal/kb/service_test.go`, add:

```go
func TestKBLint_OutdatedDistill(t *testing.T) {
    dir := setupTestKB(t)
    t.Cleanup(CloseGlobalDB)
    db, err := OpenDB(dir)
    if err != nil {
        t.Fatal(err)
    }

    // Insert a source-note with old distill_version.
    db.Exec(`INSERT INTO documents (id, path, layer, kind, title, description, content, content_hash, updated_at, authority, doc_timestamp, distill_version)
        VALUES ('wiki/old.md', 'wiki/old.md', 'wiki', 'source-note', 'Old', '', 'body', 'h1', 1, 3, 0, '0.1.0')`)
    db.Close()

    result, err := KBLint(dir)
    if err != nil {
        t.Fatal(err)
    }

    var found bool
    for _, w := range result.Warnings {
        if w.Kind == "outdated_distill" && w.Path == "wiki/old.md" {
            found = true
        }
    }
    if !found {
        t.Errorf("expected outdated_distill warning for wiki/old.md, got %v", result.Warnings)
    }
}
```

- [ ] **Step 3: Run tests**

```bash
rtk go test ./internal/kb/ -run TestKBLint_OutdatedDistill -v -tags fts5
```

- [ ] **Step 4: Commit**

```bash
rtk git add internal/kb/service.go internal/kb/service_test.go
rtk git commit -m "feat(kb): add outdated_distill lint check"
```

---

### Task 7: Run full test suite and verify

**Files:**
- None (verification only)

- [ ] **Step 1: Run all tests**

```bash
rtk go test ./... -tags fts5
```

- [ ] **Step 2: Run lint CLI**

```bash
rtk go run ./cmd/piekbs lint
```

Verify `outdated_distill` warnings appear for notes with mismatched versions.

- [ ] **Step 3: Manual WebUI verification**

Start the server and verify:
1. Banner appears when outdated notes exist
2. Click "重跑过时笔记" → button shows "处理中..."
3. Second click while processing → alert "正在处理中"
4. After refresh completes, banner disappears

```bash
rtk go run ./cmd/piekbs serve
```

- [ ] **Step 4: Final commit (if any fixes needed)**

```bash
rtk git add -A && rtk git commit -m "fix: address test/integration issues"
```
