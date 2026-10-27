package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type NewBooking struct {
	RoomID    int64
	GuestName string
	Guests    int
	CheckIn   time.Time
	CheckOut  time.Time
}

// bookingDetailsColumns selects a BookingDetails from bookings b joined to rooms r, room_types rt and hotels h.
const bookingDetailsColumns = `
	b.reference, b.room_id, b.guest_name, b.guests, b.check_in, b.check_out, b.created_at,
	r.id AS "room.id", r.hotel_id AS "room.hotel_id", r.number AS "room.number", r.type AS "room.type",
	rt.capacity AS "room.capacity", h.name AS "room.hotel_name"`

const bookingDetailsJoins = `
	JOIN rooms r ON r.id = b.room_id
	JOIN room_types rt ON rt.type = r.type
	JOIN hotels h ON h.id = r.hotel_id`

// CreateBooking books a room; the database generates its reference. The schema enforces the booking
// rules, so concurrent requests can't get around them: the room_id foreign key rejects unknown rooms, the
// bookings_capacity trigger too many guests, and the bookings_no_overlap constraint double bookings. It
// returns ErrRoomNotFound, ErrOverCapacity or ErrRoomUnavailable when the booking isn't possible.
func (s *Store) CreateBooking(ctx context.Context, nb NewBooking) (BookingDetails, error) {
	var b BookingDetails
	err := s.db.GetContext(ctx, &b, `
		WITH b AS (
		  INSERT INTO bookings (room_id, guest_name, guests, check_in, check_out)
		  VALUES ($1, $2, $3, $4, $5)
		  RETURNING *
		)
		SELECT `+bookingDetailsColumns+` FROM b`+bookingDetailsJoins,
		nb.RoomID, nb.GuestName, nb.Guests, nb.CheckIn, nb.CheckOut)

	if constraint, ok := pgConstraintError(err, codeForeignKeyViolation); ok && constraint == "bookings_room_id_fkey" {
		return BookingDetails{}, ErrRoomNotFound
	}
	if constraint, ok := pgConstraintError(err, codeCheckViolation); ok && constraint == "bookings_capacity" {
		return BookingDetails{}, ErrOverCapacity
	}
	if _, ok := pgConstraintError(err, codeExclusionViolation); ok {
		return BookingDetails{}, ErrRoomUnavailable
	}
	if err != nil {
		return BookingDetails{}, fmt.Errorf("create booking: %w", err)
	}
	return b, nil
}

// GetBooking returns the booking with this reference, or ErrBookingNotFound.
func (s *Store) GetBooking(ctx context.Context, reference uuid.UUID) (BookingDetails, error) {
	var b BookingDetails
	err := s.db.GetContext(ctx, &b,
		`SELECT `+bookingDetailsColumns+` FROM bookings b`+bookingDetailsJoins+` WHERE b.reference = $1`, reference)
	if errors.Is(err, sql.ErrNoRows) {
		return BookingDetails{}, ErrBookingNotFound
	}
	if err != nil {
		return BookingDetails{}, fmt.Errorf("get booking: %w", err)
	}
	return b, nil
}
