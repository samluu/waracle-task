package httpapi

import (
	"testing"

	"github.com/samluu/waracle-task/internal/store"
)

func TestCreateBooking(t *testing.T) {
	location := "/bookings/" + testRef
	tests := []struct {
		name     string
		request  BookingRequest
		newStore func() *StoreMock
		want     CreateBookingResponseObject
		wantErr  error
	}{
		{"booked", newValidBooking(), createBookingReturns(testBooking, nil),
			CreateBooking201JSONResponse{Body: apiTestBooking, Headers: CreateBooking201ResponseHeaders{Location: &location}}, nil},
		{"room not found", newValidBooking(), createBookingReturns(store.BookingDetails{}, store.ErrRoomNotFound),
			CreateBooking404JSONResponse{Error: "room not found"}, nil},
		{"room unavailable", newValidBooking(), createBookingReturns(store.BookingDetails{}, store.ErrRoomUnavailable),
			CreateBooking409JSONResponse{Error: "room is already booked for some of those nights"}, nil},
		{"over capacity", newValidBooking(), createBookingReturns(store.BookingDetails{}, store.ErrOverCapacity),
			CreateBooking422JSONResponse{Error: "more guests than the room's capacity"}, nil},
		{"store failure is returned", newValidBooking(), createBookingReturns(store.BookingDetails{}, errStore),
			nil, errStore},
		// The rules themselves are in validate_test.go.
		{"invalid request", func() BookingRequest {
			b := newValidBooking()
			b.GuestName = ""
			return b
		}(), noStore, CreateBooking400JSONResponse{badRequest("guest_name is required")}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newTestServer(tt.newStore()).CreateBooking(t.Context(), CreateBookingRequestObject{Body: &tt.request})
			assertResponse(t, got, err, tt.want, tt.wantErr)
		})
	}
}

func TestGetBooking(t *testing.T) {
	tests := []struct {
		name     string
		newStore func() *StoreMock
		want     GetBookingResponseObject
		wantErr  error
	}{
		{"found", getBookingReturns(testBooking, nil), GetBooking200JSONResponse(apiTestBooking), nil},
		{"not found", getBookingReturns(store.BookingDetails{}, store.ErrNotFound), GetBooking404JSONResponse{Error: "booking not found"}, nil},
		{"store failure is returned", getBookingReturns(store.BookingDetails{}, errStore), nil, errStore},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newTestServer(tt.newStore()).GetBooking(t.Context(), GetBookingRequestObject{Reference: testReference})
			assertResponse(t, got, err, tt.want, tt.wantErr)
		})
	}
}
