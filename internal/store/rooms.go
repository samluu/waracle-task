package store

import (
	"context"
	"fmt"
	"time"
)

// RoomFilter describes a stay to find rooms for, and which page of the results to return.
type RoomFilter struct {
	HotelName *string // nil searches every hotel
	CheckIn   time.Time
	CheckOut  time.Time
	Guests    int

	After *RoomKey // nil starts from the first room; otherwise only rooms sorted after this one
	Limit int      // the most rooms to return
}

// RoomKey is a room's position in FindRooms' order. Hotel names are unique, so it identifies one room.
type RoomKey struct {
	HotelName string
	Number    string
}

// FindRooms returns up to f.Limit rooms that can hold f.Guests and are free for every night of
// [f.CheckIn, f.CheckOut), sorted by hotel name then room number, starting after f.After.
func (s *Store) FindRooms(ctx context.Context, f RoomFilter) ([]Room, error) {
	var afterHotel, afterNumber *string
	if f.After != nil {
		afterHotel, afterNumber = &f.After.HotelName, &f.After.Number
	}

	// Paging is keyset-based: the next page is the rooms whose (hotel name, number) sorts after the
	// previous page's last room, so rooms becoming booked or free between pages can't make it skip or
	// repeat rooms the way an offset would. The NOT EXISTS uses the same daterange && form as the
	// bookings_no_overlap constraint, so its GiST index serves the lookup.
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
		  AND ($5::text IS NULL OR (h.name, r.number) > ($5::text, $6::text))
		ORDER BY h.name, r.number
		LIMIT $7`,
		f.HotelName, f.Guests, f.CheckIn, f.CheckOut, afterHotel, afterNumber, f.Limit)
	if err != nil {
		return nil, fmt.Errorf("find rooms: %w", err)
	}
	return rooms, nil
}
