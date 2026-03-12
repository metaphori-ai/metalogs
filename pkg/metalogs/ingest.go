package metalogs

import (
	"encoding/json"
	"strings"
	"time"
)

const insertSQL = `INSERT INTO logs (site, layer, level, message, details, source, timestamp, metadata) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

// Ingest inserts a single log entry.
// Site, layer, and level are normalized to lowercase.
func (s *Store) Ingest(entry LogEntry) error {
	entry.Site = strings.ToLower(entry.Site)
	entry.Layer = strings.ToLower(entry.Layer)
	entry.Level = LogLevel(strings.ToLower(string(entry.Level)))
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}
	var meta *string
	if entry.Metadata != nil {
		m := string(*entry.Metadata)
		meta = &m
	}
	_, err := s.db.Exec(insertSQL,
		entry.Site, entry.Layer, string(entry.Level), entry.Message,
		entry.Details, entry.Source,
		entry.Timestamp.Format(time.RFC3339Nano),
		meta,
	)
	return err
}

// IngestBatch inserts multiple log entries in a single transaction.
func (s *Store) IngestBatch(entries []LogEntry) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(insertSQL)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now().UTC()
	for i := range entries {
		e := &entries[i]
		e.Site = strings.ToLower(e.Site)
		e.Layer = strings.ToLower(e.Layer)
		e.Level = LogLevel(strings.ToLower(string(e.Level)))
		if e.Timestamp.IsZero() {
			e.Timestamp = now
		}
		var meta *string
		if e.Metadata != nil {
			m := string(*e.Metadata)
			meta = &m
		}
		if _, err := stmt.Exec(
			e.Site, e.Layer, string(e.Level), e.Message,
			e.Details, e.Source,
			e.Timestamp.Format(time.RFC3339Nano),
			meta,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// IngestJSON is a convenience for ingesting a raw JSON byte slice as a LogEntry.
func (s *Store) IngestJSON(data []byte) error {
	var entry LogEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return err
	}
	return s.Ingest(entry)
}
