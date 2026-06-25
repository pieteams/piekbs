//go:build fts5

package distill

import (
	"database/sql"
	"path/filepath"
	"time"
)

// Enqueue scans FindNewFiles and inserts not-yet-queued files into distill_queue.
// Idempotent: existing paths are skipped via INSERT OR IGNORE.
// Returns the number of newly inserted rows.
func Enqueue(db *sql.DB, kbRoot string) (int, error) {
	files := FindNewFiles(kbRoot)
	now := time.Now().Unix()
	inserted := 0
	for _, absPath := range files {
		rawDir := filepath.Join(kbRoot, "raw")
		rel, err := filepath.Rel(rawDir, absPath)
		if err != nil {
			continue
		}
		res, err := db.Exec(
			`INSERT OR IGNORE INTO distill_queue (path, status, retry_count, queued_at, updated_at)
             VALUES (?, 'pending', 0, ?, ?)`,
			filepath.ToSlash(rel), now, now,
		)
		if err != nil {
			continue
		}
		if n, _ := res.RowsAffected(); n > 0 {
			inserted++
		}
	}
	return inserted, nil
}

// NextPending atomically fetches one pending task and sets it to processing.
// Returns ("", nil) when no pending tasks exist.
func NextPending(db *sql.DB) (string, error) {
	tx, err := db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback() //nolint:errcheck

	var path string
	err = tx.QueryRow(
		`SELECT path FROM distill_queue WHERE status='pending' ORDER BY queued_at ASC LIMIT 1`,
	).Scan(&path)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	now := time.Now().Unix()
	if _, err := tx.Exec(
		`UPDATE distill_queue SET status='processing', updated_at=? WHERE path=?`,
		now, path,
	); err != nil {
		return "", err
	}
	return path, tx.Commit()
}

// MarkDone marks the task as done.
func MarkDone(db *sql.DB, path string) error {
	_, err := db.Exec(
		`UPDATE distill_queue SET status='done', updated_at=? WHERE path=?`,
		time.Now().Unix(), path,
	)
	return err
}

// MarkFailed increments retry_count atomically. If retry_count < 5, resets to pending for retry.
// At retry_count >= 5, sets permanent failed status.
func MarkFailed(db *sql.DB, path, errMsg string) error {
	now := time.Now().Unix()
	_, err := db.Exec(`
		UPDATE distill_queue
		SET retry_count = retry_count + 1,
		    status      = CASE WHEN retry_count + 1 >= 5 THEN 'failed' ELSE 'pending' END,
		    last_error  = ?,
		    updated_at  = ?
		WHERE path = ?`, errMsg, now, path)
	return err
}

// Stats returns task counts by status.
func Stats(db *sql.DB) (map[string]int, error) {
	rows, err := db.Query(`SELECT status, COUNT(*) FROM distill_queue GROUP BY status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := map[string]int{"pending": 0, "processing": 0, "done": 0, "failed": 0}
	for rows.Next() {
		var status string
		var n int
		if err := rows.Scan(&status, &n); err != nil {
			return nil, err
		}
		counts[status] = n
	}
	return counts, rows.Err()
}

// RecoverStale resets processing tasks (crash residue) back to pending on startup.
func RecoverStale(db *sql.DB) error {
	_, err := db.Exec(
		`UPDATE distill_queue SET status='pending', updated_at=? WHERE status='processing'`,
		time.Now().Unix(),
	)
	return err
}
