package handlers

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/samluu/waracle-task/internal/email"
	"github.com/samluu/waracle-task/internal/httpapi/handlers/mocks"
	"github.com/samluu/waracle-task/internal/store"
)

func TestCreateBooking(t *testing.T) {
	location := "/bookings/" + testRef
	tests := []struct {
		name    string
		request BookingRequest
		st      *mocks.StoreMock      // only the methods this request should reach are set; any other call panics
		emails  *mocks.EmailQueueMock // nil: no email may be queued, so an empty mock that panics if it is
		want    CreateBookingResponseObject
		wantErr error
	}{
		{"booked", newValidBooking(),
			&mocks.StoreMock{CreateBookingFunc: func(context.Context, store.NewBooking) (store.BookingDetails, error) {
				return testBooking, nil
			}},
			&mocks.EmailQueueMock{EnqueueFunc: func(email.Confirmation) bool { return true }},
			CreateBooking201JSONResponse{Body: apiTestBooking, Headers: CreateBooking201ResponseHeaders{Location: &location}}, nil},
		// A confirmation that can't be queued is logged, but the booking stands.
		{"booked, email queue full", newValidBooking(),
			&mocks.StoreMock{CreateBookingFunc: func(context.Context, store.NewBooking) (store.BookingDetails, error) {
				return testBooking, nil
			}},
			&mocks.EmailQueueMock{EnqueueFunc: func(email.Confirmation) bool { return false }},
			CreateBooking201JSONResponse{Body: apiTestBooking, Headers: CreateBooking201ResponseHeaders{Location: &location}}, nil},
		{"room not found", newValidBooking(),
			&mocks.StoreMock{CreateBookingFunc: func(context.Context, store.NewBooking) (store.BookingDetails, error) {
				return store.BookingDetails{}, store.ErrRoomNotFound
			}}, nil,
			CreateBooking404JSONResponse{Error: "room not found"}, nil},
		{"room unavailable", newValidBooking(),
			&mocks.StoreMock{CreateBookingFunc: func(context.Context, store.NewBooking) (store.BookingDetails, error) {
				return store.BookingDetails{}, store.ErrRoomUnavailable
			}}, nil,
			CreateBooking409JSONResponse{Error: "room is already booked for some of those nights"}, nil},
		{"over capacity", newValidBooking(),
			&mocks.StoreMock{CreateBookingFunc: func(context.Context, store.NewBooking) (store.BookingDetails, error) {
				return store.BookingDetails{}, store.ErrOverCapacity
			}}, nil,
			CreateBooking422JSONResponse{Error: "more guests than the room's capacity"}, nil},
		{"store failure is returned", newValidBooking(),
			&mocks.StoreMock{CreateBookingFunc: func(context.Context, store.NewBooking) (store.BookingDetails, error) {
				return store.BookingDetails{}, errStore
			}}, nil,
			nil, errStore},
		// The rules themselves are in validate_test.go.
		{"invalid request", func() BookingRequest {
			b := newValidBooking()
			b.GuestName = ""
			return b
		}(), &mocks.StoreMock{}, nil, CreateBooking400JSONResponse{badRequest("guest_name is required")}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			emails := tt.emails
			if emails == nil {
				emails = &mocks.EmailQueueMock{}
			}
			got, err := newBookingServer(tt.st, emails).CreateBooking(t.Context(), CreateBookingRequestObject{Body: &tt.request})
			assertResponse(t, got, err, tt.want, tt.wantErr)
		})
	}
}

func TestGetBooking(t *testing.T) {
	tests := []struct {
		name    string
		st      *mocks.StoreMock
		want    GetBookingResponseObject
		wantErr error
	}{
		{"found",
			&mocks.StoreMock{GetBookingFunc: func(context.Context, uuid.UUID) (store.BookingDetails, error) {
				return testBooking, nil
			}},
			GetBooking200JSONResponse(apiTestBooking), nil},
		{"not found",
			&mocks.StoreMock{GetBookingFunc: func(context.Context, uuid.UUID) (store.BookingDetails, error) {
				return store.BookingDetails{}, store.ErrNotFound
			}},
			GetBooking404JSONResponse{Error: "booking not found"}, nil},
		{"store failure is returned",
			&mocks.StoreMock{GetBookingFunc: func(context.Context, uuid.UUID) (store.BookingDetails, error) {
				return store.BookingDetails{}, errStore
			}},
			nil, errStore},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newTestServer(tt.st).GetBooking(t.Context(), GetBookingRequestObject{Reference: testReference})
			assertResponse(t, got, err, tt.want, tt.wantErr)
		})
	}
}
