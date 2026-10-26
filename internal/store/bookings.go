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

// CreateBooking books a room; the database generates its reference. The capacity check and the insert
// are one statement, and the bookings_no_overlap constraint rejects double bookings, so concurrent
// requests can't both succeed. It returns ErrRoomNotFound, ErrOverCapacity or ErrRoomUnavailable when
// the booking isn't possible.
func (s *Store) CreateBooking(ctx context.Context, nb NewBooking) (BookingDetails, error) {
	var b BookingDetails
	err := s.db.GetContext(ctx, &b, `
		WITH b AS (
		  INSERT INTO bookings (room_id, guest_name, guests, check_in, check_out)
		  SELECT r.id, $2, $3, $4, $5
		  FROM rooms r JOIN room_types rt ON rt.type = r.type
		  WHERE r.id = $1 AND rt.capacity >= $3
		  RETURNING *
		)
		SELECT `+bookingDetailsColumns+` FROM b`+bookingDetailsJoins,
		nb.RoomID, nb.GuestName, nb.Guests, nb.CheckIn, nb.CheckOut)

	if _, ok := pgConstraintError(err, codeExclusionViolation); ok {
		return BookingDetails{}, ErrRoomUnavailable
	}
	if errors.Is(err, sql.ErrNoRows) {
		// Nothing inserted: the room doesn't exist or is too small.
		return BookingDetails{}, s.whyNotBookable(ctx, nb)
	}
	if err != nil {
		return BookingDetails{}, fmt.Errorf("create booking: %w", err)
	}
	return b, nil
}

func (s *Store) whyNotBookable(ctx context.Context, nb NewBooking) error {
	var capacity int
	err := s.db.GetContext(ctx, &capacity,
		`SELECT rt.capacity FROM rooms r JOIN room_types rt ON rt.type = r.type WHERE r.id = $1`, nb.RoomID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrRoomNotFound
	}
	if err != nil {
		return fmt.Errorf("look up room: %w", err)
	}
	return ErrOverCapacity
}

// GetBooking returns the booking with this reference, or ErrNotFound.
func (s *Store) GetBooking(ctx context.Context, reference uuid.UUID) (BookingDetails, error) {
	var b BookingDetails
	err := s.db.GetContext(ctx, &b,
		`SELECT `+bookingDetailsColumns+` FROM bookings b`+bookingDetailsJoins+` WHERE b.reference = $1`, reference)
	if errors.Is(err, sql.ErrNoRows) {
		return BookingDetails{}, ErrNotFound
	}
	if err != nil {
		return BookingDetails{}, fmt.Errorf("get booking: %w", err)
	}
	return b, nil
}
