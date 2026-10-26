package store

import (
	"context"
	"fmt"
	"time"
)

// RoomFilter describes a stay to find rooms for, and which page of the results to return.
type RoomFilter struct {
	HotelSearch *string // only hotels whose name is a fuzzy match for this, as FindHotels; nil searches every hotel
	CheckIn     time.Time
	CheckOut    time.Time
	Guests      int

	After *RoomKey // nil starts from the first room; otherwise only rooms sorted after this one
	Limit int      // the most rooms to return
}

// RoomKey is a room's position in FindRooms' order. Hotel names are unique, so it identifies one room.
type RoomKey struct {
	HotelName string
	Number    string
}

// roomsQuery finds a page of free rooms. The first %s is an optional WITH clause and the second the table
// of hotels to search: all of them, or only those matching a name search.
//
// Paging is keyset-based: the next page is the rooms whose (hotel name, number) sorts after the previous
// page's last room, so rooms becoming booked or free between pages can't make it skip or repeat rooms the
// way an offset would. The row comparison spans two tables, so no index can seek on it alone. The extra
// h.name >= coalesce($4, <empty string>) bound can: it lets hotels_name_key start the scan at the cursor's
// hotel instead of the first hotel, so every page costs the same however deep it is. It's written with
// coalesce rather than "$4 IS NULL OR" so it stays an index condition in Postgres's generic plans.
//
// The NOT EXISTS uses the same daterange && form as the bookings_no_overlap constraint, so its GiST index
// serves the lookup.
const roomsQuery = `%s
	SELECT r.id, r.hotel_id, r.number, r.type, rt.capacity, h.name AS hotel_name
	FROM rooms r
	JOIN %s h ON h.id = r.hotel_id
	JOIN room_types rt ON rt.type = r.type
	WHERE rt.capacity >= $1
	  AND NOT EXISTS (
	    SELECT 1 FROM bookings b
	    WHERE b.room_id = r.id
	      AND daterange(b.check_in, b.check_out, '[)') && daterange($2::date, $3::date, '[)')
	  )
	  AND h.name >= coalesce($4::text, '')
	  AND ($4::text IS NULL OR (h.name, r.number) > ($4::text, $5::text))
	ORDER BY h.name, r.number
	LIMIT $6`

// Two fixed queries rather than one with "$7 IS NULL OR $7 <% h.name": in a generic plan, that OR
// stops Postgres using any index for the hotel filter, so naming a hotel would scan every hotel.
//
// Searching every hotel walks hotels_name_key from the cursor. Searching by name finds the matching hotels
// first, through hotels_name_trgm_idx, then their rooms. The CTE is MATERIALIZED to keep that order:
// otherwise Postgres, not knowing how many hotels the search matches, walks hotels_name_key checking
// every hotel against it (measured at 180ms with 200,000 hotels, against a few ms this way).
var (
	roomsInAllHotels      = fmt.Sprintf(roomsQuery, "", "hotels")
	roomsInMatchingHotels = fmt.Sprintf(roomsQuery,
		"WITH matching_hotels AS MATERIALIZED (SELECT id, name FROM hotels WHERE $7 <% name)", "matching_hotels")
)

// FindRooms returns up to f.Limit rooms that can hold f.Guests and are free for every night of
// [f.CheckIn, f.CheckOut), sorted by hotel name then room number, starting after f.After.
func (s *Store) FindRooms(ctx context.Context, f RoomFilter) ([]Room, error) {
	var afterHotel, afterNumber *string
	if f.After != nil {
		afterHotel, afterNumber = &f.After.HotelName, &f.After.Number
	}
	query, args := roomsInAllHotels, []any{f.Guests, f.CheckIn, f.CheckOut, afterHotel, afterNumber, f.Limit}
	if f.HotelSearch != nil {
		query, args = roomsInMatchingHotels, append(args, *f.HotelSearch)
	}

	rooms := []Room{}
	if err := s.db.SelectContext(ctx, &rooms, query, args...); err != nil {
		return nil, fmt.Errorf("find rooms: %w", err)
	}
	return rooms, nil
}
