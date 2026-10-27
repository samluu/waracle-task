package handlers

import (
	"context"
	"errors"
	"log/slog"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/samluu/waracle-task/internal/email"
	"github.com/samluu/waracle-task/internal/store"
)

func (s *Server) CreateBooking(ctx context.Context, req CreateBookingRequestObject) (CreateBookingResponseObject, error) {
	// The body is decoded by the generated strict handler; decoding errors go to requestError.
	nb, err := toNewBooking(*req.Body, s.today())
	if err != nil {
		return CreateBooking400JSONResponse{badRequest(err.Error())}, nil
	}

	booking, err := s.store.CreateBooking(ctx, nb)
	switch {
	case errors.Is(err, store.ErrRoomNotFound):
		return CreateBooking404JSONResponse{Error: err.Error()}, nil
	case errors.Is(err, store.ErrRoomUnavailable):
		return CreateBooking409JSONResponse{Error: err.Error()}, nil
	case errors.Is(err, store.ErrOverCapacity):
		return CreateBooking422JSONResponse{Error: err.Error()}, nil
	case err != nil:
		return nil, err
	}

	// The booking is committed, so confirm it. Sending happens in the background: the response doesn't
	// wait for it, and a confirmation that can't be queued doesn't undo the booking.
	if !s.emails.Enqueue(email.Confirmation{
		Reference:  booking.Reference,
		GuestName:  booking.GuestName,
		HotelName:  booking.Room.HotelName,
		RoomNumber: booking.Room.Number,
		CheckIn:    booking.CheckIn,
		CheckOut:   booking.CheckOut,
	}) {
		slog.Error("confirmation email not queued", "reference", booking.Reference)
	}

	location := "/bookings/" + booking.Reference.String()
	return CreateBooking201JSONResponse{
		Body:    toAPIBooking(booking),
		Headers: CreateBooking201ResponseHeaders{Location: &location},
	}, nil
}

func (s *Server) GetBooking(ctx context.Context, req GetBookingRequestObject) (GetBookingResponseObject, error) {
	booking, err := s.store.GetBooking(ctx, req.Reference)
	if errors.Is(err, store.ErrBookingNotFound) {
		return GetBooking404JSONResponse{Error: "booking not found"}, nil
	}
	if err != nil {
		return nil, err
	}
	return GetBooking200JSONResponse(toAPIBooking(booking)), nil
}

// toAPIBooking is shared by create and get so both return exactly the same shape.
func toAPIBooking(b store.BookingDetails) Booking {
	return Booking{
		Reference: b.Reference,
		GuestName: b.GuestName,
		Guests:    b.Guests,
		CheckIn:   openapi_types.Date{Time: b.CheckIn},
		CheckOut:  openapi_types.Date{Time: b.CheckOut},
		CreatedAt: b.CreatedAt,
		Room: Room{
			Id:       b.Room.ID,
			Number:   b.Room.Number,
			Type:     RoomType(b.Room.Type),
			Capacity: b.Room.Capacity,
			Hotel:    Hotel{Id: b.Room.HotelID, Name: b.Room.HotelName},
		},
	}
}
