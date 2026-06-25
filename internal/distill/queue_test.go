//go:build fts5

package distill

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/jasen215/wikiloop/internal/kb"
)

func setupQueueDB(t *testing.T) (*sql.DB, string) {
	t.Helper()
	dir := t.TempDir()
	rawDir := filepath.Join(dir, "raw")
	if err := os.MkdirAll(rawDir, 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := kb.OpenDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db, dir
}

func TestEnqueueAndStats(t *testing.T) {
	db, kbRoot := setupQueueDB(t)

	// 写两个 raw 文件
	rawDir := filepath.Join(kbRoot, "raw")
	for _, name := range []string{"a.md", "b.md"} {
		if err := os.WriteFile(filepath.Join(rawDir, name), []byte("# "+name), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	n, err := Enqueue(db, kbRoot)
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	if n != 2 {
		t.Errorf("expected 2 enqueued, got %d", n)
	}

	stats, err := Stats(db)
	if err != nil {
		t.Fatal(err)
	}
	if stats["pending"] != 2 {
		t.Errorf("expected 2 pending, got %d", stats["pending"])
	}
}

func TestEnqueueIdempotent(t *testing.T) {
	db, kbRoot := setupQueueDB(t)
	rawDir := filepath.Join(kbRoot, "raw")
	if err := os.WriteFile(filepath.Join(rawDir, "a.md"), []byte("# a"), 0o644); err != nil {
		t.Fatal(err)
	}

	Enqueue(db, kbRoot)
	n, err := Enqueue(db, kbRoot) // 第二次入队
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("expected 0 new (idempotent), got %d", n)
	}
}

func TestNextPendingAndMarkDone(t *testing.T) {
	db, kbRoot := setupQueueDB(t)
	rawDir := filepath.Join(kbRoot, "raw")
	os.WriteFile(filepath.Join(rawDir, "a.md"), []byte("# a"), 0o644)
	Enqueue(db, kbRoot)

	path, err := NextPending(db)
	if err != nil || path == "" {
		t.Fatalf("NextPending: path=%q err=%v", path, err)
	}

	// status 应为 processing
	var status string
	db.QueryRow("SELECT status FROM distill_queue WHERE path=?", path).Scan(&status)
	if status != "processing" {
		t.Errorf("expected processing, got %q", status)
	}

	if err := MarkDone(db, path); err != nil {
		t.Fatal(err)
	}
	db.QueryRow("SELECT status FROM distill_queue WHERE path=?", path).Scan(&status)
	if status != "done" {
		t.Errorf("expected done, got %q", status)
	}
}

func TestMarkFailedRetry(t *testing.T) {
	db, kbRoot := setupQueueDB(t)
	rawDir := filepath.Join(kbRoot, "raw")
	os.WriteFile(filepath.Join(rawDir, "a.md"), []byte("# a"), 0o644)
	Enqueue(db, kbRoot)

	path, _ := NextPending(db)

	// 失败 4 次 → 每次重置为 pending
	for i := 0; i < 4; i++ {
		if err := MarkFailed(db, path, "LLM error"); err != nil {
			t.Fatal(err)
		}
		var status string
		var retryCount int
		db.QueryRow("SELECT status, retry_count FROM distill_queue WHERE path=?", path).Scan(&status, &retryCount)
		if status != "pending" {
			t.Errorf("retry %d: expected pending, got %q", i+1, status)
		}
		NextPending(db) // 重新取出为 processing
	}

	// 第 5 次失败 → 永久 failed
	MarkFailed(db, path, "final error")
	var status string
	db.QueryRow("SELECT status FROM distill_queue WHERE path=?", path).Scan(&status)
	if status != "failed" {
		t.Errorf("expected permanent failed, got %q", status)
	}
}

func TestRecoverStale(t *testing.T) {
	db, kbRoot := setupQueueDB(t)
	rawDir := filepath.Join(kbRoot, "raw")
	os.WriteFile(filepath.Join(rawDir, "a.md"), []byte("# a"), 0o644)
	Enqueue(db, kbRoot)
	path, _ := NextPending(db) // 变为 processing（模拟崩溃残留）

	if err := RecoverStale(db); err != nil {
		t.Fatal(err)
	}
	var status string
	db.QueryRow("SELECT status FROM distill_queue WHERE path=?", path).Scan(&status)
	if status != "pending" {
		t.Errorf("expected status=pending after RecoverStale, got %q", status)
	}
}
