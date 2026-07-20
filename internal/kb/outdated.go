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
