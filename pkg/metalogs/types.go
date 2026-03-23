package metalogs

import (
	"encoding/json"
	"time"
)

// LogLevel represents valid log severity levels.
type LogLevel string

const (
	LevelDebug LogLevel = "debug"
	LevelInfo  LogLevel = "info"
	LevelWarn  LogLevel = "warn"
	LevelError LogLevel = "error"
	LevelFatal LogLevel = "fatal"
)

// ValidLevel returns true if the level is recognized.
func ValidLevel(l LogLevel) bool {
	switch l {
	case LevelDebug, LevelInfo, LevelWarn, LevelError, LevelFatal:
		return true
	}
	return false
}

// LogEntry represents a single log record.
type LogEntry struct {
	ID          int64            `json:"id,omitempty"`
	Site        string           `json:"site"`
	Layer       string           `json:"layer"`
	ShortName   string           `json:"short_name,omitempty"`
	Collections string           `json:"collections,omitempty"` // comma-separated, ingest-only (not stored per log)
	Level       LogLevel         `json:"level"`
	Message     string           `json:"message"`
	Details     *string          `json:"details,omitempty"`
	Source      *string          `json:"source,omitempty"`
	Timestamp   time.Time        `json:"timestamp"`
	Metadata    *json.RawMessage `json:"metadata,omitempty"`
}

// QueryOpts defines filters for querying logs.
type QueryOpts struct {
	Site       string
	Layer      string
	Sites      []string
	Layers     []string
	Collection string
	Levels     []LogLevel
	Since      *time.Time
	Until      *time.Time
	Contains   string
	Limit      int
	Offset     int
}

// SiteLayer is a site + layer pair.
type SiteLayer struct {
	Site  string `json:"site"`
	Layer string `json:"layer"`
}

// Collection is a named group of site+layer pairs.
type Collection struct {
	Name    string      `json:"name"`
	Members []SiteLayer `json:"members"`
}
