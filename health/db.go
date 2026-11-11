package health

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const schemaDDL = `
CREATE TABLE IF NOT EXISTS weigh_ins (
	user_id TEXT NOT NULL,
	id TEXT NOT NULL,
	date TEXT NOT NULL,
	kg REAL NOT NULL,
	rhr_bpm INTEGER,
	note TEXT,
	protocol INTEGER NOT NULL DEFAULT 0,
	kg_raw TEXT,
	PRIMARY KEY (user_id, id)
);
CREATE INDEX IF NOT EXISTS idx_weigh_ins_user_date ON weigh_ins(user_id, date);

CREATE TABLE IF NOT EXISTS foods (
	user_id TEXT NOT NULL,
	id TEXT NOT NULL,
	name TEXT NOT NULL,
	kcal_per_100g REAL,
	sodium_mg_per_100g REAL,
	notes TEXT,
	PRIMARY KEY (user_id, id)
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_foods_user_name ON foods(user_id, name COLLATE NOCASE);

CREATE TABLE IF NOT EXISTS days (
	user_id TEXT NOT NULL,
	date TEXT NOT NULL,
	bp TEXT,
	walking TEXT,
	total_sodium_mg REAL NOT NULL DEFAULT 0,
	total_kcal REAL NOT NULL DEFAULT 0,
	PRIMARY KEY (user_id, date)
);

CREATE TABLE IF NOT EXISTS meal_slots (
	user_id TEXT NOT NULL,
	date TEXT NOT NULL,
	slot TEXT NOT NULL,
	raw_text TEXT,
	sodium_mg REAL NOT NULL DEFAULT 0,
	kcal REAL NOT NULL DEFAULT 0,
	kcal_margin REAL NOT NULL DEFAULT 0,
	items_json TEXT NOT NULL DEFAULT '[]',
	PRIMARY KEY (user_id, date, slot),
	FOREIGN KEY (user_id, date) REFERENCES days(user_id, date) ON DELETE CASCADE
);
`

// OpenSharedDB opens healthdata/health.db for all users.
func OpenSharedDB(dataRoot string) (*sql.DB, error) {
	if err := os.MkdirAll(dataRoot, 0755); err != nil {
		return nil, err
	}
	dbPath := filepath.Join(dataRoot, "health.db")
	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schemaDDL); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}
