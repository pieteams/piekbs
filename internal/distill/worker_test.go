//go:build fts5

package distill

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jasen215/wikiloop/internal/kb"
)

// fakeDistillFile 替换 DistillFile，写一个假 source-note 表示"蒸馏成功"
// 这个测试用真实队列，但用假蒸馏函数避免真实 LLM 调用。
// worker.go 通过 distillFn 参数注入，测试时传入 fakeDistillFile。
func fakeDistillFile(config Config, rawPath, kbRoot string, _ kb.Embedder) error {
	rawDir := filepath.Join(kbRoot, "raw")
	rel, _ := filepath.Rel(rawDir, rawPath)
	notePath := filepath.Join(kbRoot, "wiki", "source-notes", rel)
	if err := os.MkdirAll(filepath.Dir(notePath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(notePath, []byte("# fake note\n"), 0o644)
}

func TestRunWorkersProcessesQueue(t *testing.T) {
	dir := t.TempDir()
	rawDir := filepath.Join(dir, "raw")
	os.MkdirAll(rawDir, 0o755)

	// 写 2 个 raw 文件
	for _, name := range []string{"a.md", "b.md"} {
		os.WriteFile(filepath.Join(rawDir, name), []byte("# "+name), 0o644)
	}

	db, err := kb.OpenDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	Enqueue(db, dir)

	cfg := Config{BaseURL: "fake", Token: "fake", Model: "fake"}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 用 fakeDistillFile 注入，1 个 worker
	runWorkersWithFn(ctx, cfg, dir, 1, fakeDistillFile)

	// 等队列处理完
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		stats, _ := Stats(db)
		if stats["done"] == 2 {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	stats, _ := Stats(db)
	if stats["done"] != 2 {
		t.Errorf("expected 2 done, got %+v", stats)
	}
}
