package metalogs

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

const (
	ingestBufferSize = 4096
	flushInterval    = 100 * time.Millisecond
	flushBatchSize   = 256
)

// Store wraps a SQLite database for log storage.
// In HTTP mode, writes are forwarded to a metalogs server
// while reads go directly to SQLite.
type Store struct {
	db         *sql.DB
	buf        chan LogEntry
	stop       chan struct{}
	wg         sync.WaitGroup
	httpURL    string // non-empty = HTTP ingest mode
	httpClient *http.Client

	// collCache tracks which (collection, site, layer) tuples have already
	// been registered, so we skip redundant INSERT OR IGNORE on every flush.
	collMu    sync.Mutex
	collCache map[string]struct{}
}

// DefaultDBPath returns ~/.metalogs/metalogs.db.
func DefaultDBPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home dir: %w", err)
	}
	return filepath.Join(home, ".metalogs", "metalogs.db"), nil
}

// NewStore opens or creates the SQLite database at dbPath,
// enables WAL mode, runs migrations, starts the write buffer,
// and returns a ready Store. Writes go directly to SQLite.
// Use this for the metalogs server itself.
func NewStore(dbPath string) (*Store, error) {
	db, err := openDB(dbPath)
	if err != nil {
		return nil, err
	}

	s := &Store{
		db:        db,
		buf:       make(chan LogEntry, ingestBufferSize),
		stop:      make(chan struct{}),
		collCache: make(map[string]struct{}),
	}

	s.wg.Add(1)
	go s.flushLoop()

	return s, nil
}

// NewHTTPClient creates a Store that forwards writes to a metalogs server
// via HTTP, while reads go directly to SQLite for zero-latency queries.
// Use this in Go backends (BFFs) to avoid SQLite write contention.
//
// serverURL is the metalogs server base URL, e.g. "http://localhost:9999".
// dbPath is the SQLite path for reads (defaults to ~/.metalogs/metalogs.db).
func NewHTTPClient(serverURL string, dbPath string) (*Store, error) {
	db, err := openDB(dbPath)
	if err != nil {
		return nil, err
	}

	s := &Store{
		db:      db,
		buf:     make(chan LogEntry, ingestBufferSize),
		stop:    make(chan struct{}),
		httpURL: serverURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		collCache: make(map[string]struct{}),
	}

	s.wg.Add(1)
	go s.flushLoop()

	return s, nil
}

func openDB(dbPath string) (*sql.DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	dsn := fmt.Sprintf("file:%s?_busy_timeout=5000", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	db.SetMaxOpenConns(1)

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable WAL: %w", err)
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return db, nil
}

// Close drains the write buffer and closes the database connection.
func (s *Store) Close() error {
	close(s.stop)
	s.wg.Wait()
	return s.db.Close()
}

func (s *Store) flushLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	batch := make([]LogEntry, 0, flushBatchSize)

	for {
		select {
		case entry := <-s.buf:
			batch = append(batch, entry)
			if len(batch) >= flushBatchSize {
				s.flush(batch)
				batch = batch[:0]
			}
		case <-ticker.C:
			for {
				select {
				case entry := <-s.buf:
					batch = append(batch, entry)
				default:
					goto done
				}
			}
		done:
			if len(batch) > 0 {
				s.flush(batch)
				batch = batch[:0]
			}
		case <-s.stop:
			for {
				select {
				case entry := <-s.buf:
					batch = append(batch, entry)
				default:
					goto shutdown
				}
			}
		shutdown:
			if len(batch) > 0 {
				s.flush(batch)
			}
			return
		}
	}
}

func (s *Store) flush(batch []LogEntry) {
	var err error
	if s.httpURL != "" {
		err = s.httpFlush(batch)
	} else {
		err = s.writeBatch(batch)
	}
	if err != nil {
		log.Printf("metalogs flush error: %v (%d entries dropped)", err, len(batch))
	}
}

func (s *Store) httpFlush(batch []LogEntry) error {
	body, err := json.Marshal(batch)
	if err != nil {
		return err
	}

	resp, err := s.httpClient.Post(s.httpURL+"/ingest/batch", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("metalogs server returned %d", resp.StatusCode)
	}
	return nil
}

func (s *Store) writeBatch(entries []LogEntry) error {
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

	// Collect collection registrations needed from this batch.
	type collKey struct{ coll, site, layer string }
	var newColls []collKey

	for i := range entries {
		e := &entries[i]
		var meta *string
		if e.Metadata != nil {
			m := string(*e.Metadata)
			meta = &m
		}
		if _, err := stmt.Exec(
			e.Site, e.Layer, e.ShortName, string(e.Level), e.Message,
			e.Details, e.Source,
			e.Timestamp.Format(time.RFC3339Nano),
			meta,
		); err != nil {
			return err
		}

		// Track collection memberships to register.
		for _, coll := range parseCollections(e.Collections) {
			key := coll + "\x00" + e.Site + "\x00" + e.Layer
			s.collMu.Lock()
			_, seen := s.collCache[key]
			if !seen {
				s.collCache[key] = struct{}{}
			}
			s.collMu.Unlock()
			if !seen {
				newColls = append(newColls, collKey{coll, e.Site, e.Layer})
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// Register new collection memberships outside the log-insert transaction.
	if len(newColls) > 0 {
		for _, ck := range newColls {
			s.db.Exec("INSERT OR IGNORE INTO collections (name, site, layer) VALUES (?, ?, ?)",
				ck.coll, ck.site, ck.layer)
		}
	}

	return nil
}
