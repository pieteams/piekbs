//go:build fts5

package kb

import (
	"strings"
	"testing"
)

// TestOpenDB_AppliesWALAndBusyTimeout guards the DSN pragma syntax.
// modernc.org/sqlite silently ignores the mattn-style _journal_mode=/_busy_timeout=
// keys, so the connection must be opened with the _pragma= form. Without the fix
// journal_mode reports "delete" and busy_timeout reports 0.
func TestOpenDB_AppliesWALAndBusyTimeout(t *testing.T) {
	db, err := OpenDB(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var journal string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&journal); err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(journal, "wal") {
		t.Fatalf("journal_mode = %q, want wal (DSN pragma not applied)", journal)
	}

	var busy int
	if err := db.QueryRow("PRAGMA busy_timeout").Scan(&busy); err != nil {
		t.Fatal(err)
	}
	if busy != 5000 {
		t.Fatalf("busy_timeout = %d, want 5000 (DSN pragma not applied)", busy)
	}
}
