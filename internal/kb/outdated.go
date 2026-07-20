//go:build fts5

package kb

import (
	"database/sql"
)

// FindOutdatedNotes returns wiki source-notes whose schema_version is less
// than currentSchemaVersion. Returns nil when currentSchemaVersion is 0
// (development mode or unset).
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
