// Package handlers implements the API defined in api/openapi.yaml. The generated code in api.gen.go
// provides the request and response types and the strict interface Server implements; the httpapi
// package wires it into an HTTP router.
package handlers

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.8.1 -config oapi-codegen.yaml ../../../api/openapi.yaml
//go:generate go tool moq -pkg mocks -skip-ensure -out mocks/store.gen.go . Store

import (
	"context"
	"log/slog"
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
	now   func() time.Time
}

// Fails to compile if Server is missing a handler for an operation in the spec.
var _ StrictServerInterface = (*Server)(nil)

// NewServer returns handlers backed by st. now gives the current time for date validation; it's a
// parameter so tests can fix "today" (production passes time.Now).
func NewServer(st Store, now func() time.Time) *Server {
	return &Server{store: st, now: now}
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

func badRequest(msg string) BadRequestJSONResponse {
	return BadRequestJSONResponse{Error: msg}
}
