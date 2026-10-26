// Package handlers implements the API defined in api/openapi.yaml. The generated code in api.gen.go
// provides the request and response types and the strict interface Server implements; the httpapi
// package wires it into an HTTP router.
package handlers

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.8.1 -config oapi-codegen.yaml ../../../api/openapi.yaml
//go:generate go tool moq -pkg mocks -skip-ensure -out mocks/store.gen.go . Store
//go:generate go tool moq -pkg mocks -skip-ensure -out mocks/email_queue.gen.go . EmailQueue

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/samluu/waracle-task/internal/email"
	"github.com/samluu/waracle-task/internal/store"
)

// Store is the data access the handlers depend on; *store.Store implements it.
type Store interface {
	Ping(ctx context.Context) error
	FindHotels(ctx context.Context, f store.HotelFilter) ([]store.Hotel, error)
	FindRooms(ctx context.Context, f store.RoomFilter) ([]store.Room, error)
	CreateBooking(ctx context.Context, nb store.NewBooking) (store.BookingDetails, error)
	GetBooking(ctx context.Context, reference uuid.UUID) (store.BookingDetails, error)
}

// EmailQueue takes confirmation emails to send in the background; *email.Queue implements it. Enqueue
// must not block: it reports false if it can't take the email.
type EmailQueue interface {
	Enqueue(c email.Confirmation) bool
}

// Server implements the strict handlers generated from api/openapi.yaml: each takes a decoded request
// and returns a typed response, and the generated code does the HTTP decoding and encoding.
type Server struct {
	store  Store
	emails EmailQueue
	now    func() time.Time
}

// Fails to compile if Server is missing a handler for an operation in the spec.
var _ StrictServerInterface = (*Server)(nil)

// NewServer returns handlers backed by st, sending booking confirmations through emails. now gives the
// current time for date validation; it's a parameter so tests can fix "today" (production passes time.Now).
func NewServer(st Store, emails EmailQueue, now func() time.Time) *Server {
	return &Server{store: st, emails: emails, now: now}
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
