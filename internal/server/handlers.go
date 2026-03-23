package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/metaphori-ai/metalogs/pkg/metalogs"
)

func (s *Server) handleIngest(w http.ResponseWriter, r *http.Request) {
	var entry metalogs.LogEntry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := s.store.Ingest(entry); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if rw, ok := w.(*responseWriter); ok {
		rw.ingestSite = entry.Site
		rw.ingestLayer = entry.Layer
		rw.ingestLevel = string(entry.Level)
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "ok"})
}

func (s *Server) handleIngestBatch(w http.ResponseWriter, r *http.Request) {
	var entries []metalogs.LogEntry
	if err := json.NewDecoder(r.Body).Decode(&entries); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := s.store.IngestBatch(entries); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if rw, ok := w.(*responseWriter); ok {
		rw.ingestCount = len(entries)
		if len(entries) > 0 {
			rw.ingestSite = entries[0].Site
			rw.ingestLayer = entries[0].Layer
			rw.ingestLevel = highestLevel(entries)
		}
	}
	writeJSON(w, http.StatusCreated, map[string]int{"ingested": len(entries)})
}

func (s *Server) handleQuery(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	opts := metalogs.QueryOpts{
		Site:       q.Get("site"),
		Layer:      q.Get("layer"),
		Collection: q.Get("collection"),
		Contains:   q.Get("contains"),
	}

	if sites := q.Get("sites"); sites != "" {
		for _, site := range strings.Split(sites, ",") {
			opts.Sites = append(opts.Sites, strings.TrimSpace(site))
		}
	}

	if layers := q.Get("layers"); layers != "" {
		for _, layer := range strings.Split(layers, ",") {
			opts.Layers = append(opts.Layers, strings.TrimSpace(layer))
		}
	}

	if levels := q.Get("level"); levels != "" {
		for _, l := range strings.Split(levels, ",") {
			opts.Levels = append(opts.Levels, metalogs.LogLevel(strings.TrimSpace(l)))
		}
	}

	if since := q.Get("since"); since != "" {
		if d, err := parseDuration(since); err == nil {
			t := time.Now().UTC().Add(-d)
			opts.Since = &t
		} else if t, err := time.Parse(time.RFC3339, since); err == nil {
			opts.Since = &t
		}
	}

	if until := q.Get("until"); until != "" {
		if t, err := time.Parse(time.RFC3339, until); err == nil {
			opts.Until = &t
		}
	}

	if limit := q.Get("limit"); limit != "" {
		if n, err := strconv.Atoi(limit); err == nil {
			opts.Limit = n
		}
	}

	if offset := q.Get("offset"); offset != "" {
		if n, err := strconv.Atoi(offset); err == nil {
			opts.Offset = n
		}
	}

	results, err := s.store.Query(opts)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if results == nil {
		results = []metalogs.LogEntry{}
	}
	writeJSON(w, http.StatusOK, results)
}

func (s *Server) handleCleanup(w http.ResponseWriter, r *http.Request) {
	ttl := s.config.CleanupTTL
	if olderThan := r.URL.Query().Get("older_than"); olderThan != "" {
		if d, err := parseDuration(olderThan); err == nil {
			ttl = d
		}
	}
	deleted, err := s.store.Cleanup(ttl)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"deleted": deleted})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleSites(w http.ResponseWriter, r *http.Request) {
	pairs, err := s.store.ListSiteLayers()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if pairs == nil {
		pairs = []metalogs.SiteLayer{}
	}
	writeJSON(w, http.StatusOK, pairs)
}

func (s *Server) handleListCollections(w http.ResponseWriter, r *http.Request) {
	colls, err := s.store.ListCollections()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if colls == nil {
		colls = []metalogs.Collection{}
	}
	writeJSON(w, http.StatusOK, colls)
}

func (s *Server) handleCreateCollection(w http.ResponseWriter, r *http.Request) {
	var coll metalogs.Collection
	if err := json.NewDecoder(r.Body).Decode(&coll); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if coll.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}
	if err := s.store.CreateCollection(coll.Name, coll.Members); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "ok"})
}

func (s *Server) handleDeleteCollection(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}
	if err := s.store.DeleteCollection(name); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// highestLevel returns the most severe log level from a batch.
func highestLevel(entries []metalogs.LogEntry) string {
	order := map[metalogs.LogLevel]int{
		metalogs.LevelDebug: 0,
		metalogs.LevelInfo:  1,
		metalogs.LevelWarn:  2,
		metalogs.LevelError: 3,
		metalogs.LevelFatal: 4,
	}
	best := entries[0].Level
	for _, e := range entries[1:] {
		if order[e.Level] > order[best] {
			best = e.Level
		}
	}
	return string(best)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// parseDuration extends time.ParseDuration to support "d" (days) suffix.
func parseDuration(s string) (time.Duration, error) {
	if strings.HasSuffix(s, "d") {
		s = strings.TrimSuffix(s, "d")
		n, err := strconv.Atoi(s)
		if err != nil {
			return 0, err
		}
		return time.Duration(n) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}
