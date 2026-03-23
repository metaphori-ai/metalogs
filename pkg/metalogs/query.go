package metalogs

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Query returns log entries matching the given options.
// Site, layer, and level filters are normalized to lowercase.
func (s *Store) Query(opts QueryOpts) ([]LogEntry, error) {
	opts.Site = strings.ToLower(opts.Site)
	opts.Layer = strings.ToLower(opts.Layer)
	opts.Collection = strings.ToLower(opts.Collection)
	for i, s := range opts.Sites {
		opts.Sites[i] = strings.ToLower(s)
	}
	for i, l := range opts.Layers {
		opts.Layers[i] = strings.ToLower(l)
	}
	for i, l := range opts.Levels {
		opts.Levels[i] = LogLevel(strings.ToLower(string(l)))
	}

	var where []string
	var args []any

	// Collection expands to a set of (site, layer) pairs
	if opts.Collection != "" {
		coll, err := s.GetCollection(opts.Collection)
		if err != nil {
			return nil, fmt.Errorf("collection %q: %w", opts.Collection, err)
		}
		if len(coll.Members) > 0 {
			var pairs []string
			for _, m := range coll.Members {
				pairs = append(pairs, "(site = ? AND layer = ?)")
				args = append(args, m.Site, m.Layer)
			}
			where = append(where, "("+strings.Join(pairs, " OR ")+")")
		}
	} else {
		if opts.Site != "" {
			where = append(where, "site = ?")
			args = append(args, opts.Site)
		}
		if len(opts.Sites) > 0 {
			placeholders := make([]string, len(opts.Sites))
			for i, site := range opts.Sites {
				placeholders[i] = "?"
				args = append(args, site)
			}
			where = append(where, fmt.Sprintf("site IN (%s)", strings.Join(placeholders, ",")))
		}
		if opts.Layer != "" {
			where = append(where, "layer = ?")
			args = append(args, opts.Layer)
		}
		if len(opts.Layers) > 0 {
			placeholders := make([]string, len(opts.Layers))
			for i, l := range opts.Layers {
				placeholders[i] = "?"
				args = append(args, l)
			}
			where = append(where, fmt.Sprintf("layer IN (%s)", strings.Join(placeholders, ",")))
		}
	}

	if len(opts.Levels) > 0 {
		placeholders := make([]string, len(opts.Levels))
		for i, l := range opts.Levels {
			placeholders[i] = "?"
			args = append(args, string(l))
		}
		where = append(where, fmt.Sprintf("level IN (%s)", strings.Join(placeholders, ",")))
	}

	if opts.Since != nil {
		where = append(where, "timestamp >= ?")
		args = append(args, opts.Since.Format(time.RFC3339Nano))
	}

	if opts.Until != nil {
		where = append(where, "timestamp <= ?")
		args = append(args, opts.Until.Format(time.RFC3339Nano))
	}

	if opts.Contains != "" {
		where = append(where, "message LIKE ?")
		args = append(args, "%"+opts.Contains+"%")
	}

	q := "SELECT id, site, layer, short_name, level, message, details, source, timestamp, metadata FROM logs"
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	q += " ORDER BY timestamp ASC"

	limit := opts.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	q += fmt.Sprintf(" LIMIT %d", limit)

	if opts.Offset > 0 {
		q += fmt.Sprintf(" OFFSET %d", opts.Offset)
	}

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []LogEntry
	for rows.Next() {
		var e LogEntry
		var ts string
		var meta *string
		if err := rows.Scan(&e.ID, &e.Site, &e.Layer, &e.ShortName, &e.Level, &e.Message, &e.Details, &e.Source, &ts, &meta); err != nil {
			return nil, err
		}
		e.Timestamp, _ = time.Parse(time.RFC3339Nano, ts)
		if meta != nil {
			raw := json.RawMessage(*meta)
			e.Metadata = &raw
		}
		results = append(results, e)
	}
	return results, rows.Err()
}
