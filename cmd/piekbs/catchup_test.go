//go:build fts5

package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pieteams/piekbs/internal/kb"
)

const catchUpTestKB = "PIEKBS_TEST_CATCHUP_KB"

func TestCatchUpProcess(t *testing.T) {
	if root := os.Getenv(catchUpTestKB); root != "" {
		if err := runStdio(root); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
}

func startCatchUpProcess(t *testing.T, root string, stderr io.Writer) func() {
	t.Helper()
	t.Setenv(catchUpTestKB, root)
	t.Setenv("PIEKBS_DISTILL_BASE_URL", "")
	t.Setenv("PIEKBS_DISTILL_TOKEN", "")
	t.Setenv("PIEKBS_DISTILL_MODEL", "")
	t.Setenv("PIEKBS_DISTILL_API_TYPE", "")
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	t.Cleanup(cancel)
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCatchUpProcess$")
	cmd.Stdout = io.Discard
	cmd.Stderr = stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	return func() {
		_ = stdin.Close()
		if err := cmd.Wait(); err != nil {
			t.Error(err)
		}
	}
}

func waitForCatchUp(t *testing.T, ready func() bool, failure string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if ready() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal(failure)
}

func newOfflineCatchUpFixture(t *testing.T) (string, *sql.DB, time.Time) {
	t.Helper()
	root := t.TempDir()
	if err := ensureKBDirs(root); err != nil {
		t.Fatal(err)
	}
	db, err := kb.OpenDB(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	stopped := time.Now().Add(-time.Minute)
	if err := os.WriteFile(filepath.Join(root, lastShutdownFile), []byte(fmt.Sprint(stopped.Unix())), 0644); err != nil {
		t.Fatal(err)
	}
	return root, db, stopped.Add(30 * time.Second)
}

func writeOfflineFile(t *testing.T, path, contents string, modified time.Time) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, modified, modified); err != nil {
		t.Fatal(err)
	}
}

func hasPendingCatchUp(t *testing.T, root string) bool {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, "index"))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), catchUpPrefix) {
			return true
		}
	}
	return false
}

func runCatchUpWithFailure(t *testing.T, root, failure string) {
	t.Helper()
	logFile, err := os.CreateTemp(t.TempDir(), "catch-up-*.log")
	if err != nil {
		t.Fatal(err)
	}
	defer logFile.Close()
	stop := startCatchUpProcess(t, root, logFile)
	defer stop()
	waitForCatchUp(t, func() bool {
		contents, err := os.ReadFile(logFile.Name())
		if err != nil {
			t.Fatal(err)
		}
		log := string(contents)
		return strings.Contains(log, failure) ||
			strings.Contains(log, "catch-up:") && !hasPendingCatchUp(t, root)
	}, "the first catch-up attempt did not finish")
}

func catchUpResultExists(t *testing.T, db *sql.DB, query, id string) bool {
	t.Helper()
	hits, err := kb.FTSSearch(db, query, nil, 10)
	if err != nil {
		t.Fatal(err)
	}
	for _, hit := range hits {
		if hit.ID == id {
			return true
		}
	}
	return false
}

func TestStdioCatchesUpOfflineFiles(t *testing.T) {
	tests := []struct {
		name   string
		marker string
	}{
		{"clean shutdown", lastShutdownFile},
		{"interrupted shutdown", runningFile},
		{"first startup", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			if err := ensureKBDirs(root); err != nil {
				t.Fatal(err)
			}
			db, err := kb.OpenDB(root)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()

			existing := filepath.Join(root, "raw", "existing.md")
			previousRun := time.Now().Add(-time.Minute)
			writeOfflineFile(t, existing, "# OldOrchardMarker", previousRun.Add(-time.Second))
			if _, err := kb.IndexFiles(db, root); err != nil {
				t.Fatal(err)
			}
			if tt.marker != "" {
				timestamp := fmt.Sprint(previousRun.Unix())
				if err := os.WriteFile(filepath.Join(root, tt.marker), []byte(timestamp), 0644); err != nil {
					t.Fatal(err)
				}
			}
			offline := previousRun.Add(30 * time.Second)
			writeOfflineFile(t, existing, "# UpdatedOrchardMarker", offline)
			writeOfflineFile(t, filepath.Join(root, "raw", "offline.md"), "# NewAubergineMarker", offline)

			stop := startCatchUpProcess(t, root, os.Stderr)
			defer stop()

			search := func(query string) []kb.SearchResult {
				t.Helper()
				hits, err := kb.FTSSearch(db, query, nil, 5)
				if err != nil {
					t.Fatal(err)
				}
				return hits
			}
			waitForCatchUp(t, func() bool {
				updated := search("UpdatedOrchardMarker")
				added := search("NewAubergineMarker")
				if len(updated) != 1 || len(added) != 1 {
					return false
				}
				if updated[0].ID != "raw/existing.md" || added[0].ID != "raw/offline.md" {
					t.Fatalf("unexpected result IDs: %q, %q", updated[0].ID, added[0].ID)
				}
				if old := search("OldOrchardMarker"); len(old) != 0 {
					t.Fatalf("old content is still indexed: %v", old)
				}
				return true
			}, "the edited and added files were not both indexed during startup")
		})
	}
}

func TestStdioRetriesFailedCatchUpOnNextStartup(t *testing.T) {
	root, db, modified := newOfflineCatchUpFixture(t)
	writeOfflineFile(t, filepath.Join(root, "raw", "offline.md"), "# RetryRecoveryMarker\n", modified)
	if _, err := db.Exec(`CREATE TRIGGER fail_offline_insert BEFORE INSERT ON documents
		WHEN NEW.id = 'raw/offline.md' BEGIN SELECT RAISE(ABORT, 'forced catch-up failure'); END`); err != nil {
		t.Fatal(err)
	}

	runCatchUpWithFailure(t, root, "forced catch-up failure")
	if _, err := db.Exec(`DROP TRIGGER fail_offline_insert`); err != nil {
		t.Fatal(err)
	}

	stop := startCatchUpProcess(t, root, os.Stderr)
	defer stop()
	waitForCatchUp(t, func() bool {
		return catchUpResultExists(t, db, "RetryRecoveryMarker", "raw/offline.md")
	}, "the next startup did not recover the previously failed file")
	waitForCatchUp(t, func() bool {
		return !hasPendingCatchUp(t, root)
	}, "successful catch-up did not clear the pending checkpoints")
}

func TestStdioRetriesPartialConversionOnNextStartup(t *testing.T) {
	root, db, modified := newOfflineCatchUpFixture(t)
	raw := filepath.Join(root, "raw")
	writeOfflineFile(t, filepath.Join(raw, "blocked", "retry.csv"), "name,value\nConversionRetryMarker,42\n", modified)
	writeOfflineFile(t, filepath.Join(raw, "ready.csv"), "name,value\nConversionReadyMarker,42\n", modified)
	blocker := filepath.Join(raw, "converted", "blocked")
	writeOfflineFile(t, blocker, "prevents creation of the output directory", modified)

	runCatchUpWithFailure(t, root, "catch-up: convert files:")
	if !catchUpResultExists(t, db, "ConversionReadyMarker", "raw/converted/ready.md") {
		t.Fatal("the successful conversion was not indexed while another file failed")
	}
	// The successful output must not trigger an unrelated scan on restart.
	if err := os.Chtimes(filepath.Join(raw, "converted", "ready.md"), modified, modified); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(blocker); err != nil {
		t.Fatal(err)
	}

	stop := startCatchUpProcess(t, root, os.Stderr)
	defer stop()
	waitForCatchUp(t, func() bool {
		return catchUpResultExists(t, db, "ConversionRetryMarker", "raw/converted/blocked/retry.md") && !hasPendingCatchUp(t, root)
	}, "the next startup did not finish and index the previously blocked conversion")
}

func TestStdioRetriesPartialEnqueueOnNextStartup(t *testing.T) {
	root, db, modified := newOfflineCatchUpFixture(t)
	raw := filepath.Join(root, "raw")
	writeOfflineFile(t, filepath.Join(raw, "a-retry.md"), "# EnqueueRetryMarker\n", modified)
	writeOfflineFile(t, filepath.Join(raw, "b-ready.md"), "# EnqueueReadyMarker\n", modified)
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "test provider unavailable", http.StatusServiceUnavailable)
	}))
	defer provider.Close()
	config := fmt.Sprintf("distill:\n  base_url: %q\n  token: test-only\n  model: test-only\n  workers: 1\n", provider.URL)
	if err := os.WriteFile(filepath.Join(root, "config.yaml"), []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TRIGGER fail_offline_enqueue BEFORE INSERT ON distill_queue
		WHEN NEW.path = 'a-retry.md' BEGIN SELECT RAISE(ABORT, 'forced enqueue failure'); END`); err != nil {
		t.Fatal(err)
	}
	queued := func(path string) bool {
		var count int
		if err := db.QueryRow(`SELECT count(*) FROM distill_queue WHERE path=?`, path).Scan(&count); err != nil {
			t.Fatal(err)
		}
		return count == 1
	}

	runCatchUpWithFailure(t, root, "forced enqueue failure")
	if !queued("b-ready.md") || !catchUpResultExists(t, db, "EnqueueRetryMarker", "raw/a-retry.md") {
		t.Fatal("an enqueue failure prevented indexing the document or queuing its sibling")
	}
	if _, err := db.Exec(`DROP TRIGGER fail_offline_enqueue`); err != nil {
		t.Fatal(err)
	}

	stop := startCatchUpProcess(t, root, os.Stderr)
	defer stop()
	waitForCatchUp(t, func() bool {
		return queued("a-retry.md") && queued("b-ready.md") && !hasPendingCatchUp(t, root)
	}, "the next startup did not queue the failed file and retire the checkpoint")
}
