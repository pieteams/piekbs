# Schema 版本独立 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Decouple schema version from application version, using a simple integer stored in `schema/VERSION`.

**Architecture:** Schema version is an integer (1, 2, 3...) stored in an embedded `VERSION` file. Notes record the schema version used during distillation. Comparison uses `<` to detect outdated notes.

**Tech Stack:** Go, SQLite (modernc.org/sqlite), embed

## Global Constraints

- Schema version is a simple integer, not semver
- `Config.SchemaVersion` stays `string` type, with `SchemaVersionInt()` for parsing
- Old `"0.4.8"` values parse to `0` (unversioned)
- DB column: `schema_version INTEGER DEFAULT 0`
- Migration: DROP old `distill_version` column, ADD new `schema_version` column
- `FindOutdatedNotes(db, 0)` returns nil (unversioned mode skips detection)

---

### Task 1: schema/VERSION file + Config adapter

**Files:**
- Create: `internal/kbinit/schema/VERSION`
- Modify: `internal/kbinit/init.go` (add `readSchemaVersion()`)
- Modify: `internal/config/config.go` (add `SchemaVersionInt()`)
- Test: `internal/config/config_test.go`

**Interfaces:**
- Produces: `readSchemaVersion() (int, error)` — reads embedded VERSION file
- Produces: `Config.SchemaVersionInt() int` — parses string to int, returns 0 on error

- [ ] **Step 1: Create VERSION file**

Create `internal/kbinit/schema/VERSION` with content:
```
1
```

- [ ] **Step 2: Add `readSchemaVersion()` to init.go**

In `internal/kbinit/init.go`, add:

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

Add `"strings"` to imports if not present.

- [ ] **Step 3: Add `SchemaVersionInt()` to config.go**

In `internal/config/config.go`, add method to `Config`:

```go
// SchemaVersionInt parses SchemaVersion as int. Returns 0 for non-numeric values.
func (c Config) SchemaVersionInt() int {
	v, err := strconv.Atoi(c.SchemaVersion)
	if err != nil {
		return 0
	}
	return v
}
```

Note: `strconv` is already imported.

- [ ] **Step 4: Write tests**

In `internal/config/config_test.go`, add:

```go
func TestSchemaVersionInt(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"0.4.8", 0},
		{"2", 2},
		{"", 0},
		{"abc", 0},
	}
	for _, tt := range tests {
		c := Config{SchemaVersion: tt.input}
		if got := c.SchemaVersionInt(); got != tt.want {
			t.Errorf("SchemaVersionInt(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
```

- [ ] **Step 5: Run tests**

```bash
rtk go test ./internal/config/ -v -tags fts5 -run TestSchemaVersionInt
```

- [ ] **Step 6: Commit**

```bash
rtk git add internal/kbinit/schema/VERSION internal/kbinit/init.go internal/config/config.go internal/config/config_test.go
rtk git commit -m "feat: add schema/VERSION file and SchemaVersionInt adapter"
```

---

### Task 2: Init/UpgradeSchema + distill injection

**Files:**
- Modify: `internal/kbinit/init.go` (Init + UpgradeSchema)
- Modify: `internal/distill/distill.go` (injectSchemaVersion)
- Test: `internal/kbinit/init_test.go`, `internal/distill/distill_test.go`

**Interfaces:**
- Consumes: `readSchemaVersion()` from Task 1
- Consumes: `Config.SchemaVersionInt()` from Task 1
- Produces: `injectSchemaVersion(text string, ver int) string`

- [ ] **Step 1: Update Init function**

In `internal/kbinit/init.go`, modify `Init` to use `readSchemaVersion()`:

```go
func Init(kbRoot string, force bool) error {
	// ... (directory creation unchanged) ...

	if err := writeSchemaFiles(kbRoot, force); err != nil {
		return err
	}

	schemaVer, err := readSchemaVersion()
	if err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}

	cfg, err := config.Load(kbRoot)
	if err != nil {
		return fmt.Errorf("load config for schema_version: %w", err)
	}
	if cfg.SchemaVersion == "0" || cfg.SchemaVersion == "" || force {
		cfg.SchemaVersion = strconv.Itoa(schemaVer)
		if err := config.Save(kbRoot, cfg); err != nil {
			return fmt.Errorf("save schema_version: %w", err)
		}
	}

	return nil
}
```

Add `"strconv"` to imports.

- [ ] **Step 2: Update UpgradeSchema function**

In `internal/kbinit/init.go`, modify `UpgradeSchema`:

```go
func UpgradeSchema(kbRoot string) ([]string, int, error) {
	oldCfg, err := config.Load(kbRoot)
	if err != nil {
		return nil, 0, fmt.Errorf("load config: %w", err)
	}
	oldVersion := oldCfg.SchemaVersionInt()

	if err := writeSchemaFiles(kbRoot, true); err != nil {
		return nil, oldVersion, err
	}

	newVersion, err := readSchemaVersion()
	if err != nil {
		return nil, oldVersion, fmt.Errorf("read schema version: %w", err)
	}

	// Collect updated file list.
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

	// Update config.
	oldCfg.SchemaVersion = strconv.Itoa(newVersion)
	if err := config.Save(kbRoot, oldCfg); err != nil {
		return nil, oldVersion, fmt.Errorf("save config: %w", err)
	}

	// Reindex.
	if _, err := kb.KBReindex(kbRoot, false); err != nil {
		return updated, oldVersion, fmt.Errorf("reindex: %w", err)
	}

	return updated, oldVersion, nil
}
```

Note: Return type changed from `([]string, string, error)` to `([]string, int, error)`.

- [ ] **Step 3: Update callers of UpgradeSchema**

In `internal/webui/api.go`, update `handleSchemaUpgrade`:

```go
func (s *Server) handleSchemaUpgrade(w http.ResponseWriter, r *http.Request) {
	// ... (method check unchanged) ...
	updated, oldVersion, err := kbinit.UpgradeSchema(s.kbRoot)
	if err != nil {
		kbErrToHTTP(w, err)
		return
	}
	writeJSON(w, map[string]interface{}{
		"updated":         updated,
		"old_version":     oldVersion,
		"current_version": version.Version,
	})
}
```

In `cmd/piekbs/main.go`, update `runSchemaUpgrade` if it calls `UpgradeSchema`.

- [ ] **Step 4: Rename injectDistillVersion to injectSchemaVersion**

In `internal/distill/distill.go`:

```go
// injectSchemaVersion adds a schema_version field to the YAML frontmatter.
// If frontmatter is absent, text is returned unchanged.
func injectSchemaVersion(text string, ver int) string {
	if !strings.HasPrefix(text, "---\n") {
		return text
	}
	end := strings.Index(text[4:], "\n---")
	if end < 0 {
		return text
	}
	fmEnd := 4 + end
	fm := text[4:fmEnd]
	rest := text[fmEnd:]

	// Don't duplicate if already present.
	if strings.Contains(fm, "schema_version:") {
		return text
	}

	newFM := strings.TrimRight(fm, "\n") + "\nschema_version: " + strconv.Itoa(ver)
	return "---\n" + newFM + rest
}
```

- [ ] **Step 5: Update DistillFile to use new function**

In `internal/distill/distill.go`, find the call to `injectDistillVersion` and replace:

```go
// Old:
// generated = injectDistillVersion(generated, version.Version)

// New:
cfg, _ := config.Load(kbRoot)
generated = injectSchemaVersion(generated, cfg.SchemaVersionInt())
```

Note: `config` package needs to be imported if not already.

- [ ] **Step 6: Run tests**

```bash
rtk go test ./internal/kbinit/ ./internal/distill/ -v -tags fts5
```

- [ ] **Step 7: Commit**

```bash
rtk git add internal/kbinit/init.go internal/distill/distill.go internal/webui/api.go
rtk git commit -m "feat: use schema version in Init/UpgradeSchema and distill injection"
```

---

### Task 3: Database migration + query interface

**Files:**
- Modify: `internal/kb/schema.go` (DDL)
- Modify: `internal/kb/db.go` (migration)
- Modify: `internal/kb/index.go` (upsertDocument)
- Modify: `internal/kb/outdated.go` (FindOutdatedNotes signature)
- Test: `internal/kb/db_test.go`, `internal/kb/outdated_test.go`

**Interfaces:**
- Consumes: `schema_version INTEGER DEFAULT 0` column
- Produces: `FindOutdatedNotes(db *sql.DB, currentSchemaVersion int) ([]string, error)`

- [ ] **Step 1: Update schema DDL**

In `internal/kb/schema.go`, change the documents table:

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
    schema_version INTEGER DEFAULT 0
);
```

- [ ] **Step 2: Update migration logic**

In `internal/kb/db.go`, in `migrateDescription`:

Replace `hasDistillVersion` with `hasSchemaVersion`:

```go
hasSchemaVersion := false
// ... in the PRAGMA loop:
if name == "schema_version" {
    hasSchemaVersion = true
}
```

Replace the migration block:

```go
if !hasSchemaVersion {
    // Drop old distill_version column if it exists.
    db.Exec("ALTER TABLE documents DROP COLUMN distill_version")
    // Add new integer column.
    if _, err := db.Exec("ALTER TABLE documents ADD COLUMN schema_version INTEGER DEFAULT 0"); err != nil {
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
            if v, ok := parsed.RawFM["schema_version"]; ok {
                if n, ok := v.(int); ok && n > 0 {
                    rel, _ := filepath.Rel(kbRoot, path)
                    db.Exec("UPDATE documents SET schema_version = ? WHERE path = ?",
                        n, filepath.ToSlash(rel))
                }
            }
            return nil
        })
    }
}
```

- [ ] **Step 3: Update upsertDocument**

In `internal/kb/index.go`, in `upsertDocument`:

```go
var schemaVersion int
if v, ok := parsed.RawFM["schema_version"]; ok {
    if n, ok := v.(int); ok {
        schemaVersion = n
    }
}
```

Update the INSERT/UPDATE SQL to use `schema_version` instead of `distill_version`.

- [ ] **Step 4: Update FindOutdatedNotes**

In `internal/kb/outdated.go`:

```go
func FindOutdatedNotes(db *sql.DB, currentSchemaVersion int) ([]string, error) {
    if currentSchemaVersion == 0 {
        return nil, nil
    }
    rows, err := db.Query(
        "SELECT path, schema_version FROM documents WHERE layer='wiki' AND kind='source-note'")
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var paths []string
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

- [ ] **Step 5: Update tests**

In `internal/kb/outdated_test.go`, update all tests to use `int` parameter:

```go
func TestFindOutdatedNotes_ZeroVersion(t *testing.T) {
    dir := setupTestKB(t)
    db, _ := OpenDB(dir)
    defer db.Close()
    paths, err := FindOutdatedNotes(db, 0)
    if err != nil {
        t.Fatal(err)
    }
    if paths != nil {
        t.Errorf("expected nil for version 0, got %v", paths)
    }
}

func TestFindOutdatedNotes_OlderVersion(t *testing.T) {
    dir := setupTestKB(t)
    db, _ := OpenDB(dir)
    defer db.Close()
    db.Exec(`INSERT INTO documents (id, path, layer, kind, title, description, content, content_hash, updated_at, authority, doc_timestamp, schema_version)
        VALUES ('wiki/old.md', 'wiki/old.md', 'wiki', 'source-note', 'Old', '', 'body', 'h1', 1, 3, 0, 1)`)
    paths, err := FindOutdatedNotes(db, 2)
    if err != nil {
        t.Fatal(err)
    }
    if len(paths) != 1 || paths[0] != "wiki/old.md" {
        t.Errorf("expected [wiki/old.md], got %v", paths)
    }
}

func TestFindOutdatedNotes_SameVersion(t *testing.T) {
    dir := setupTestKB(t)
    db, _ := OpenDB(dir)
    defer db.Close()
    db.Exec(`INSERT INTO documents (id, path, layer, kind, title, description, content, content_hash, updated_at, authority, doc_timestamp, schema_version)
        VALUES ('wiki/new.md', 'wiki/new.md', 'wiki', 'source-note', 'New', '', 'body', 'h2', 1, 3, 0, 2)`)
    paths, err := FindOutdatedNotes(db, 2)
    if err != nil {
        t.Fatal(err)
    }
    if len(paths) != 0 {
        t.Errorf("expected empty, got %v", paths)
    }
}
```

- [ ] **Step 6: Run tests**

```bash
rtk go test ./internal/kb/ -v -tags fts5 -run "TestFindOutdated|TestUpsertDocument"
```

- [ ] **Step 7: Commit**

```bash
rtk git add internal/kb/schema.go internal/kb/db.go internal/kb/index.go internal/kb/outdated.go internal/kb/outdated_test.go internal/kb/db_test.go
rtk git commit -m "feat: migrate to integer schema_version column"
```

---

### Task 4: WebUI + Lint adaptation

**Files:**
- Modify: `internal/webui/api.go` (handleSchemaStatus, handleDistillOutdated)
- Modify: `internal/kb/service.go` (KBLint)

**Interfaces:**
- Consumes: `Config.SchemaVersionInt()` from Task 1
- Consumes: `FindOutdatedNotes(db, int)` from Task 3

- [ ] **Step 1: Add embeddedSchemaVersion to Server**

In `internal/webui/server.go`, add field:

```go
type Server struct {
    kbRoot               string
    importLark           func(context.Context, string, string, string) (*larkimport.Result, error)
    refreshMu            sync.Mutex
    embeddedSchemaVersion int
}
```

In `NewServer`, read the embedded version:

```go
func NewServer(kbRoot string) *Server {
    // Read embedded schema version (best-effort).
    embeddedVer := 0
    // This will be set by the caller or read from kbinit.
    return &Server{
        kbRoot: kbRoot,
        // ...
        embeddedSchemaVersion: embeddedVer,
    }
}
```

Note: The embedded version needs to be passed from `main.go` where `kbinit` is available.

- [ ] **Step 2: Update handleSchemaStatus**

In `internal/webui/api.go`:

```go
func (s *Server) handleSchemaStatus(w http.ResponseWriter, r *http.Request) {
    cfg, err := config.Load(s.kbRoot)
    if err != nil {
        kbErrToHTTP(w, err)
        return
    }
    writeJSON(w, map[string]interface{}{
        "current_version": version.Version,
        "schema_version":  cfg.SchemaVersionInt(),
        "outdated":        cfg.SchemaVersionInt() < s.embeddedSchemaVersion,
    })
}
```

- [ ] **Step 3: Update handleDistillOutdated**

In `internal/webui/api.go`:

```go
func (s *Server) handleDistillOutdated(w http.ResponseWriter, r *http.Request) {
    db, err := kb.GlobalDB(s.kbRoot)
    if err != nil {
        kbErrToHTTP(w, err)
        return
    }
    cfg, err := config.Load(s.kbRoot)
    if err != nil {
        kbErrToHTTP(w, err)
        return
    }
    paths, err := kb.FindOutdatedNotes(db, cfg.SchemaVersionInt())
    if err != nil {
        kbErrToHTTP(w, err)
        return
    }
    writeJSON(w, map[string]interface{}{
        "count":           len(paths),
        "current_version": cfg.SchemaVersionInt(),
    })
}
```

- [ ] **Step 4: Update handleDistillRefreshOutdated**

In `internal/webui/api.go`, update to use `cfg.SchemaVersionInt()`:

```go
func (s *Server) handleDistillRefreshOutdated(w http.ResponseWriter, r *http.Request) {
    // ... (method check, mutex lock unchanged) ...
    cfg, err := config.Load(s.kbRoot)
    if err != nil {
        kbErrToHTTP(w, err)
        return
    }
    outdated, err := kb.FindOutdatedNotes(db, cfg.SchemaVersionInt())
    // ... (rest unchanged) ...
}
```

- [ ] **Step 5: Update KBLint**

In `internal/kb/service.go`:

```go
// In KBLint, after cleanBrokenLinks:
cfg, cfgErr := config.Load(kbRoot)
if cfgErr == nil {
    outdated, outdatedErr := FindOutdatedNotes(db, cfg.SchemaVersionInt())
    if outdatedErr == nil {
        for _, p := range outdated {
            warnings = append(warnings, LintWarning{
                Path:   p,
                Kind:   "outdated_distill",
                Detail: "schema version outdated",
            })
        }
    }
}
```

- [ ] **Step 6: Run full tests**

```bash
rtk go test ./... -tags fts5
```

- [ ] **Step 7: Commit**

```bash
rtk git add internal/webui/api.go internal/webui/server.go internal/kb/service.go
rtk git commit -m "feat: adapt WebUI and Lint to integer schema version"
```

---

### Task 5: Full verification

- [ ] **Step 1: Run all tests**

```bash
rtk go test ./... -tags fts5
```

- [ ] **Step 2: Build and test locally**

```bash
rtk go build -tags fts5 -o piekbs ./cmd/piekbs/
rtk pkill piekbs 2>/dev/null; rtk cp piekbs /Applications/PieKBS.app/Contents/MacOS/piekbs && rtk open /Applications/PieKBS.app
sleep 3 && rtk curl -s http://localhost:8766/api/schema/status
rtk curl -s http://localhost:8766/api/distill/outdated
```
