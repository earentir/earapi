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

CREATE TABLE IF NOT EXISTS stock_lots (
	user_id TEXT NOT NULL,
	id TEXT NOT NULL,
	name TEXT NOT NULL,
	prepared_date TEXT,
	batch_grams REAL NOT NULL,
	remaining_grams REAL NOT NULL,
	active INTEGER NOT NULL DEFAULT 1,
	items_json TEXT NOT NULL DEFAULT '[]',
	notes TEXT,
	kcal_per_100g REAL,
	sodium_mg_per_100g REAL,
	PRIMARY KEY (user_id, id)
);
CREATE INDEX IF NOT EXISTS idx_stock_lots_user ON stock_lots(user_id);

CREATE TABLE IF NOT EXISTS stock_movements (
	user_id TEXT NOT NULL,
	id TEXT NOT NULL,
	lot_id TEXT NOT NULL,
	created_at TEXT NOT NULL,
	meal_date TEXT,
	meal_slot TEXT,
	grams REAL NOT NULL,
	kind TEXT NOT NULL,
	note TEXT,
	PRIMARY KEY (user_id, id),
	FOREIGN KEY (user_id, lot_id) REFERENCES stock_lots(user_id, id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_stock_movements_lot ON stock_movements(user_id, lot_id, created_at);

CREATE TABLE IF NOT EXISTS user_settings (
	user_id TEXT NOT NULL PRIMARY KEY,
	soft_low_kcal REAL NOT NULL DEFAULT 1500,
	soft_high_kcal REAL NOT NULL DEFAULT 1700,
	gain_min_kcal REAL NOT NULL DEFAULT 2200,
	meal_margin_percent REAL NOT NULL DEFAULT 10
);
`

// OpenSharedDB opens healthdata/health.db for all users.
func OpenSharedDB(dataRoot string) (*sql.DB, error) {
	if err := os.MkdirAll(dataRoot, 0755); err != nil {
		return nil, err
	}
	dbPath := filepath.Join(dataRoot, "health.db")
	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schemaDDL); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := ensureFoodAliasTable(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := migrateHealthDB(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func migrateHealthDB(db *sql.DB) error {
	if !tableHasColumn(db, "stock_lots", "active") {
		if _, err := db.Exec(`ALTER TABLE stock_lots ADD COLUMN active INTEGER NOT NULL DEFAULT 1`); err != nil {
			return err
		}
		_, _ = db.Exec(`UPDATE stock_lots SET active = 0 WHERE remaining_grams <= 0.01`)
	}
	if !tableHasColumn(db, "days", "legacy_total_sodium_mg") {
		if _, err := db.Exec(`ALTER TABLE days ADD COLUMN legacy_total_sodium_mg REAL NOT NULL DEFAULT 0`); err != nil {
			return err
		}
	}
	if !tableHasColumn(db, "days", "legacy_total_kcal") {
		if _, err := db.Exec(`ALTER TABLE days ADD COLUMN legacy_total_kcal REAL NOT NULL DEFAULT 0`); err != nil {
			return err
		}
	}
	if !tableHasColumn(db, "user_settings", "meal_margin_percent") {
		if _, err := db.Exec(`ALTER TABLE user_settings ADD COLUMN meal_margin_percent REAL NOT NULL DEFAULT 10`); err != nil {
			return err
		}
	}
	for _, col := range []string{
		"energy_kj_per_100g",
		"fat_g_per_100g",
		"saturates_g_per_100g",
		"carbs_g_per_100g",
		"sugars_g_per_100g",
		"protein_g_per_100g",
		"salt_g_per_100g",
	} {
		if !tableHasColumn(db, "foods", col) {
			if _, err := db.Exec(`ALTER TABLE foods ADD COLUMN ` + col + ` REAL`); err != nil {
				return err
			}
		}
	}
	return nil
}

func tableHasColumn(db *sql.DB, table, col string) bool {
	rows, err := db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return false
	}
	defer rows.Close()
	found := false
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk) == nil && name == col {
			found = true
			break
		}
	}
	return found && rows.Err() == nil
}
