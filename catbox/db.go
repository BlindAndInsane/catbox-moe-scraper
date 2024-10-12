package catbox

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

func InitDB(dbFile string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return nil, err
	}
	query := `
	CREATE TABLE IF NOT EXISTS valid_ids (
		id TEXT PRIMARY KEY,
		url TEXT,
		ext TEXT
	);
	`
	_, err = db.Exec(query)
	if err != nil {
		return nil, err
	}
	return db, nil
}
