package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/metaphori-ai/metalogs/pkg/metalogs"
)

// Config holds server configuration.
type Config struct {
	Port       int
	CleanupTTL time.Duration
}

// Server is the metalogs HTTP server.
type Server struct {
	store  *metalogs.Store
	config Config
	http   *http.Server
	stop   chan struct{}
}

// New creates a new Server with routes registered.
func New(store *metalogs.Store, cfg Config) *Server {
	s := &Server{
		store:  store,
		config: cfg,
		stop:   make(chan struct{}),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /ingest", s.handleIngest)
	mux.HandleFunc("POST /ingest/batch", s.handleIngestBatch)
	mux.HandleFunc("GET /query", s.handleQuery)
	mux.HandleFunc("POST /cleanup", s.handleCleanup)
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /sites", s.handleSites)
	mux.HandleFunc("GET /collections", s.handleListCollections)
	mux.HandleFunc("POST /collections", s.handleCreateCollection)
	mux.HandleFunc("DELETE /collections/{name}", s.handleDeleteCollection)

	s.http = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      withLogging(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return s
}

// Start begins listening and starts the background cleanup goroutine.
func (s *Server) Start() error {
	go s.cleanupLoop()
	log.Printf("metalogs server listening on :%d (cleanup TTL: %s)", s.config.Port, s.config.CleanupTTL)
	return s.http.ListenAndServe()
}

// Shutdown gracefully stops the server and cleanup goroutine.
func (s *Server) Shutdown(ctx context.Context) error {
	close(s.stop)
	return s.http.Shutdown(ctx)
}

func (s *Server) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			deleted, err := s.store.Cleanup(s.config.CleanupTTL)
			if err != nil {
				log.Printf("cleanup error: %v", err)
			} else if deleted > 0 {
				log.Printf("cleanup: deleted %d old logs", deleted)
			}
		case <-s.stop:
			return
		}
	}
}
