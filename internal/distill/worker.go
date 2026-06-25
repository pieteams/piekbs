//go:build fts5

package distill

import (
	"context"
	"log"
	"math"
	"path/filepath"
	"time"

	"github.com/jasen215/wikiloop/internal/kb"
)

type distillFunc func(Config, string, string, kb.Embedder) error

// RunWorkers starts n worker goroutines that consume distill_queue.
// Blocks until ctx is cancelled.
func RunWorkers(ctx context.Context, cfg Config, kbRoot string, n int) {
	runWorkersWithFn(ctx, cfg, kbRoot, n, DistillFile)
}

func runWorkersWithFn(ctx context.Context, cfg Config, kbRoot string, n int, fn distillFunc) {
	for i := 0; i < n; i++ {
		go workerLoop(ctx, cfg, kbRoot, fn)
	}
	<-ctx.Done()
}

func workerLoop(ctx context.Context, cfg Config, kbRoot string, fn distillFunc) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		db, err := kb.OpenDB(kbRoot)
		if err != nil {
			log.Printf("distill worker: open db: %v", err)
			sleep(ctx, 5*time.Second)
			continue
		}

		path, err := NextPending(db)
		if err != nil {
			log.Printf("distill worker: next pending: %v", err)
			db.Close()
			sleep(ctx, 5*time.Second)
			continue
		}

		if path == "" {
			db.Close()
			sleep(ctx, 5*time.Second)
			continue
		}

		rawPath := filepath.Join(kbRoot, "raw", filepath.FromSlash(path))
		log.Printf("distill worker: processing %s", path)

		distillErr := fn(cfg, rawPath, kbRoot, nil)
		if distillErr != nil {
			log.Printf("distill worker: failed %s: %v", path, distillErr)
			var retryCount int
			db.QueryRow("SELECT retry_count FROM distill_queue WHERE path=?", path).Scan(&retryCount) //nolint:errcheck
			MarkFailed(db, path, distillErr.Error())                                                  //nolint:errcheck
			db.Close()
			// 指数退避：10s, 20s, 40s, 80s, 160s
			backoff := time.Duration(math.Pow(2, float64(retryCount))) * 10 * time.Second
			sleep(ctx, backoff)
			continue
		}

		MarkDone(db, path) //nolint:errcheck
		log.Printf("distill worker: done %s", path)

		// post-distill reindex
		if _, err := kb.IndexFiles(db, kbRoot); err != nil {
			log.Printf("distill worker: post-distill reindex: %v", err)
		}
		db.Close()
	}
}

func sleep(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}
