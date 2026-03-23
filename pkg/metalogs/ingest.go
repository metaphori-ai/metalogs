package metalogs

import (
	"encoding/json"
	"strings"
	"time"
)

const insertSQL = `INSERT INTO logs (site, layer, short_name, level, message, details, source, timestamp, metadata) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

func normalize(entry *LogEntry) {
	entry.Site = strings.ToLower(entry.Site)
	entry.Layer = strings.ToLower(entry.Layer)
	entry.ShortName = strings.ToLower(entry.ShortName)
	entry.Level = LogLevel(strings.ToLower(string(entry.Level)))
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	} else {
		entry.Timestamp = entry.Timestamp.UTC()
	}
}

// parseCollections splits a comma-separated collections string into a slice.
func parseCollections(s string) []string {
	if s == "" {
		return nil
	}
	var result []string
	for _, c := range strings.Split(s, ",") {
		c = strings.TrimSpace(strings.ToLower(c))
		if c != "" {
			result = append(result, c)
		}
	}
	return result
}

// Ingest queues a single log entry for writing.
// This is non-blocking — entries are flushed in batches by a background goroutine.
// Site, layer, short_name, and level are normalized to lowercase.
// If Collections is set (comma-separated), the site+layer pair is auto-registered
// into each named collection.
func (s *Store) Ingest(entry LogEntry) error {
	normalize(&entry)
	select {
	case s.buf <- entry:
		return nil
	default:
		// Buffer full — drop silently rather than block the caller
		return nil
	}
}

// IngestBatch queues multiple log entries for writing.
func (s *Store) IngestBatch(entries []LogEntry) error {
	for i := range entries {
		normalize(&entries[i])
		select {
		case s.buf <- entries[i]:
		default:
			// Buffer full — drop remaining
			return nil
		}
	}
	return nil
}

// IngestJSON is a convenience for ingesting a raw JSON byte slice as a LogEntry.
func (s *Store) IngestJSON(data []byte) error {
	var entry LogEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return err
	}
	return s.Ingest(entry)
}
