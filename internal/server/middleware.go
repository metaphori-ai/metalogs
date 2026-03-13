package server

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/metaphori-ai/metalogs/pkg/metalogs"
)

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func withLogging(store *metalogs.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rw, r)
			dur := time.Since(start).Round(time.Microsecond)
			log.Printf("%s %s %d %s", r.Method, r.URL.Path, rw.status, dur)

			// Self-log to metalogs — skip ingest paths to avoid infinite recursion
			if strings.HasPrefix(r.URL.Path, "/ingest") {
				return
			}
			selfLog(store, r, rw.status, dur)
		})
	}
}

func selfLog(store *metalogs.Store, r *http.Request, status int, dur time.Duration) {
	level := metalogs.LevelInfo
	if status >= 500 {
		level = metalogs.LevelError
	} else if status >= 400 {
		level = metalogs.LevelWarn
	}

	msg := fmt.Sprintf("%s %s %d %s", r.Method, r.URL.Path, status, dur)
	source := "metalogs/server"

	store.Ingest(metalogs.LogEntry{
		Site:    "metalogs",
		Layer:   "server",
		Level:   level,
		Message: msg,
		Source:  &source,
	})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
