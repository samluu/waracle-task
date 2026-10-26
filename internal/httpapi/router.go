// Package httpapi exposes the HTTP routes and handlers.
package httpapi

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.8.1 -config oapi-codegen.yaml ../../api/openapi.yaml

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

// Server implements the handlers generated from api/openapi.yaml.
type Server struct {
	store Store
}

// Fails to compile if Server is missing a handler for an operation in the spec.
var _ ServerInterface = (*Server)(nil)

func NewRouter(s Store) http.Handler {
	mux := http.NewServeMux()
	// Docs routes sit outside the spec: they describe the API rather than being part of it.
	mux.HandleFunc("GET /openapi.yaml", serveSpec)
	mux.HandleFunc("GET /docs", serveDocs)
	return HandlerFromMux(&Server{store: s}, mux)
}

func (s *Server) GetHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := s.store.Ping(ctx); err != nil {
		slog.Error("health check: db unreachable", "err", err)
		writeError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	writeJSON(w, http.StatusOK, HealthResponse{Status: "ok"})
}
