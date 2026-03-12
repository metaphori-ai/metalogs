package metalogs

import "database/sql"

const schema = `
CREATE TABLE IF NOT EXISTS logs (
	id        INTEGER PRIMARY KEY AUTOINCREMENT,
	site      TEXT    NOT NULL,
	layer     TEXT    NOT NULL,
	level     TEXT    NOT NULL,
	message   TEXT    NOT NULL,
	details   TEXT,
	source    TEXT,
	timestamp DATETIME NOT NULL,
	metadata  TEXT
);

CREATE INDEX IF NOT EXISTS idx_logs_site       ON logs(site);
CREATE INDEX IF NOT EXISTS idx_logs_layer      ON logs(layer);
CREATE INDEX IF NOT EXISTS idx_logs_site_layer ON logs(site, layer);
CREATE INDEX IF NOT EXISTS idx_logs_level      ON logs(level);
CREATE INDEX IF NOT EXISTS idx_logs_timestamp  ON logs(timestamp);

CREATE TABLE IF NOT EXISTS collections (
	name  TEXT NOT NULL,
	site  TEXT NOT NULL,
	layer TEXT NOT NULL,
	PRIMARY KEY (name, site, layer)
);
`

func migrate(db *sql.DB) error {
	_, err := db.Exec(schema)
	return err
}
