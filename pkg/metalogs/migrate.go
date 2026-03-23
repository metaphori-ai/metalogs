package metalogs

import "database/sql"

const schema = `
CREATE TABLE IF NOT EXISTS logs (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	site       TEXT    NOT NULL,
	layer      TEXT    NOT NULL,
	short_name TEXT    NOT NULL DEFAULT '',
	level      TEXT    NOT NULL,
	message    TEXT    NOT NULL,
	details    TEXT,
	source     TEXT,
	timestamp  DATETIME NOT NULL,
	metadata   TEXT
);

CREATE INDEX IF NOT EXISTS idx_logs_site       ON logs(site);
CREATE INDEX IF NOT EXISTS idx_logs_layer      ON logs(layer);
CREATE INDEX IF NOT EXISTS idx_logs_site_layer ON logs(site, layer);
CREATE INDEX IF NOT EXISTS idx_logs_short_name ON logs(short_name);
CREATE INDEX IF NOT EXISTS idx_logs_level      ON logs(level);
CREATE INDEX IF NOT EXISTS idx_logs_timestamp  ON logs(timestamp);

CREATE TABLE IF NOT EXISTS collections (
	name  TEXT NOT NULL,
	site  TEXT NOT NULL,
	layer TEXT NOT NULL,
	PRIMARY KEY (name, site, layer)
);
`

// migrations that add columns to existing databases.
var migrations = []string{
	"ALTER TABLE logs ADD COLUMN short_name TEXT NOT NULL DEFAULT ''",
	"CREATE INDEX IF NOT EXISTS idx_logs_short_name ON logs(short_name)",
}

func migrate(db *sql.DB) error {
	if _, err := db.Exec(schema); err != nil {
		return err
	}
	// Apply additive migrations — ignore errors from already-applied ones.
	for _, m := range migrations {
		db.Exec(m)
	}
	return nil
}
