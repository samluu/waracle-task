// Package httpapi exposes the HTTP routes and handlers.
package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// Store is the data access the handlers depend on; *store.Store implements it.
type Store interface {
	Ping(ctx context.Context) error
}

func NewRouter(s Store) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", health(s))
	return mux
}

func health(s Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := s.Ping(ctx); err != nil {
			slog.Error("health check: db unreachable", "err", err)
			writeError(w, http.StatusServiceUnavailable, "database unavailable")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}
