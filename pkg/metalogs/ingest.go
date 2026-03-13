package metalogs

import (
	"encoding/json"
	"strings"
	"time"
)

const insertSQL = `INSERT INTO logs (site, layer, level, message, details, source, timestamp, metadata) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

func normalize(entry *LogEntry) {
	entry.Site = strings.ToLower(entry.Site)
	entry.Layer = strings.ToLower(entry.Layer)
	entry.Level = LogLevel(strings.ToLower(string(entry.Level)))
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}
}

// Ingest queues a single log entry for writing.
// This is non-blocking — entries are flushed in batches by a background goroutine.
// Site, layer, and level are normalized to lowercase.
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
