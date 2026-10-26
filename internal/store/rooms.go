package store

import (
	"context"
	"fmt"
	"time"
)

// RoomFilter describes a stay to find rooms for.
type RoomFilter struct {
	HotelName *string // nil searches every hotel
	CheckIn   time.Time
	CheckOut  time.Time
	Guests    int
}

// FindRooms returns rooms that can hold f.Guests and are free for every night of [f.CheckIn, f.CheckOut),
// sorted by hotel name then room number.
func (s *Store) FindRooms(ctx context.Context, f RoomFilter) ([]Room, error) {
	// The NOT EXISTS uses the same daterange && form as the bookings_no_overlap constraint,
	// so its GiST index serves the lookup.
	rooms := []Room{}
	err := s.db.SelectContext(ctx, &rooms, `
		SELECT r.id, r.hotel_id, r.number, r.type, rt.capacity, h.name AS hotel_name
		FROM rooms r
		JOIN hotels h ON h.id = r.hotel_id
		JOIN room_types rt ON rt.type = r.type
		WHERE ($1::text IS NULL OR h.name = $1)
		  AND rt.capacity >= $2
		  AND NOT EXISTS (
		    SELECT 1 FROM bookings b
		    WHERE b.room_id = r.id
		      AND daterange(b.check_in, b.check_out, '[)') && daterange($3::date, $4::date, '[)')
		  )
		ORDER BY h.name, r.number`,
		f.HotelName, f.Guests, f.CheckIn, f.CheckOut)
	if err != nil {
		return nil, fmt.Errorf("find rooms: %w", err)
	}
	return rooms, nil
}
