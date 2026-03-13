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
		Handler:      withCORS(withLogging(store)(mux)),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return s
}

// Start begins listening and starts the background cleanup goroutine.
func (s *Server) Start() error {
	go s.cleanupLoop()
	msg := fmt.Sprintf("server started on :%d (cleanup TTL: %s)", s.config.Port, s.config.CleanupTTL)
	log.Print(msg)
	s.selfLog(metalogs.LevelInfo, msg)
	return s.http.ListenAndServe()
}

// Shutdown gracefully stops the server and cleanup goroutine.
func (s *Server) Shutdown(ctx context.Context) error {
	s.selfLog(metalogs.LevelInfo, "server shutting down")
	close(s.stop)
	return s.http.Shutdown(ctx)
}

func (s *Server) selfLog(level metalogs.LogLevel, message string) {
	source := "metalogs/server"
	s.store.Ingest(metalogs.LogEntry{
		Site:    "metalogs",
		Layer:   "server",
		Level:   level,
		Message: message,
		Source:  &source,
	})
}

func (s *Server) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			deleted, err := s.store.Cleanup(s.config.CleanupTTL)
			if err != nil {
				msg := fmt.Sprintf("cleanup error: %v", err)
				log.Print(msg)
				s.selfLog(metalogs.LevelError, msg)
			} else if deleted > 0 {
				msg := fmt.Sprintf("cleanup: deleted %d old logs", deleted)
				log.Print(msg)
				s.selfLog(metalogs.LevelInfo, msg)
			}
		case <-s.stop:
			return
		}
	}
}
