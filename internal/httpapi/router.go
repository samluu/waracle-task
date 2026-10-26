// Package httpapi exposes the HTTP routes and handlers.
package httpapi

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.8.1 -config oapi-codegen.yaml ../../api/openapi.yaml
//go:generate go tool moq -out store_mock_test.go . Store

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/samluu/waracle-task/internal/store"
)

// Store is the data access the handlers depend on; *store.Store implements it.
type Store interface {
	Ping(ctx context.Context) error
	FindHotelByName(ctx context.Context, name string) (store.Hotel, error)
	FindRooms(ctx context.Context, f store.RoomFilter) ([]store.Room, error)
	CreateBooking(ctx context.Context, nb store.NewBooking) (store.BookingDetails, error)
	GetBooking(ctx context.Context, reference uuid.UUID) (store.BookingDetails, error)
}

// Server implements the strict handlers generated from api/openapi.yaml: each takes a decoded request
// and returns a typed response, and the generated code does the HTTP decoding and encoding.
type Server struct {
	store Store
	now   func() time.Time // overridden in tests to fix "today"
}

// Fails to compile if Server is missing a handler for an operation in the spec.
var _ StrictServerInterface = (*Server)(nil)

func NewRouter(s Store) http.Handler {
	return newRouter(&Server{store: s, now: time.Now})
}

func newRouter(srv *Server) http.Handler {
	mux := http.NewServeMux()
	// Docs routes sit outside the spec: they describe the API rather than being part of it.
	mux.HandleFunc("GET /openapi.yaml", serveSpec)
	mux.HandleFunc("GET /docs", serveDocs)

	return limitBody(HandlerWithOptions(newStrictHandler(srv), StdHTTPServerOptions{
		BaseRouter:       mux,
		ErrorHandlerFunc: paramError,
	}))
}

// newStrictHandler wraps srv in the generated code that decodes requests and encodes responses.
func newStrictHandler(srv *Server) ServerInterface {
	return NewStrictHandlerWithOptions(srv, nil, StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  requestError,
		ResponseErrorHandlerFunc: responseError,
	})
}

func (s *Server) GetHealth(ctx context.Context, _ GetHealthRequestObject) (GetHealthResponseObject, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := s.store.Ping(ctx); err != nil {
		slog.Error("health check: db unreachable", "err", err)
		return GetHealth503JSONResponse{Error: "database unavailable"}, nil
	}
	return GetHealth200JSONResponse{Status: "ok"}, nil
}
