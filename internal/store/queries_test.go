package store

import (
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
)

// The seed data (seed.go) has 2 hotels with rooms 101-102 single, 201-202 double, 301-302 deluxe, and seededTestStore adds testBookings. At The Grand Hotel: 101 booked
// 11-13 Jan, 201 booked 10-14 and 14-16 Jan, 301 booked 12-15 Jan; and at Seaside Inn: 202 booked 11-12 Jan (all 2027).

func roomID(t *testing.T, s *Store, hotel, number string) int64 {
	t.Helper()
	var id int64
	if err := s.db.GetContext(t.Context(), &id,
		`SELECT r.id FROM rooms r JOIN hotels h ON h.id = r.hotel_id WHERE h.name = $1 AND r.number = $2`,
		hotel, number); err != nil {
		t.Fatalf("room %s/%s: %v", hotel, number, err)
	}
	return id
}

func TestFindHotels(t *testing.T) {
	s := seededTestStore(t)
	// A third hotel sharing a word with each seed hotel, so a search can match more than one.
	if _, err := s.db.ExecContext(t.Context(), `INSERT INTO hotels (name) VALUES ('Grand Seaside Resort')`); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		search string
		want   []string
	}{
		{"The Grand Hotel", []string{"The Grand Hotel"}}, // the full name still matches
		// Any case. Fuzzy: "Grand Seaside Resort" shares most of it too (word similarity 0.67, over 0.6).
		{"SEASIDE inn", []string{"Grand Seaside Resort", "Seaside Inn"}},
		{"grand", []string{"Grand Seaside Resort", "The Grand Hotel"}}, // several matches, sorted by name
		{"gran", []string{"Grand Seaside Resort", "The Grand Hotel"}},  // part of a word
		{"seasid", []string{"Grand Seaside Resort", "Seaside Inn"}},
		{"grande", []string{"Grand Seaside Resort", "The Grand Hotel"}}, // a small misspelling
		{"resort", []string{"Grand Seaside Resort"}},
		{"lodge", nil},
		{"gr", []string{"Grand Seaside Resort", "The Grand Hotel"}}, // two letters can match the start of a word
		{"e", nil}, // a single letter is too short to match
		{"%", nil}, // plain text, not a pattern: no wildcards to escape
		{"_", nil},
	} {
		t.Run(tt.search, func(t *testing.T) {
			hotels, err := s.FindHotels(t.Context(), HotelFilter{Search: tt.search, Limit: 100})
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, h := range hotels {
				if h.ID == 0 {
					t.Errorf("hotel %+v has no id", h)
				}
				got = append(got, h.Name)
			}
			if !slices.Equal(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// Paging through the matches, 2 at a time, returns each matching hotel exactly once, in name order.
func TestFindHotelsPages(t *testing.T) {
	s := seededTestStore(t)
	for _, name := range []string{"Grand Seaside Resort", "Grand Lake Lodge", "Little Grand Inn"} {
		if _, err := s.db.ExecContext(t.Context(), `INSERT INTO hotels (name) VALUES ($1)`, name); err != nil {
			t.Fatal(err)
		}
	}
	page := func(after *string) []string {
		t.Helper()
		hotels, err := s.FindHotels(t.Context(), HotelFilter{Search: "grand", After: after, Limit: 2})
		if err != nil {
			t.Fatal(err)
		}
		var got []string
		for _, h := range hotels {
			got = append(got, h.Name)
		}
		return got
	}

	// "Seaside Inn" doesn't match "grand", so it never appears.
	pages := [][]string{{"Grand Lake Lodge", "Grand Seaside Resort"}, {"Little Grand Inn", "The Grand Hotel"}}
	var after *string
	for i, want := range pages {
		got := page(after)
		if !slices.Equal(got, want) {
			t.Fatalf("page %d: got %v, want %v", i+1, got, want)
		}
		after = &got[len(got)-1]
	}
	if got := page(after); got != nil {
		t.Fatalf("after the last match: got %v, want nothing", got)
	}
}

func TestFindRooms(t *testing.T) {
	s := seededTestStore(t)
	grand, seaside, nope := "The Grand Hotel", "Seaside Inn", "Nope"
	partial := "GRAND"
	search := func(hotel *string, checkIn, checkOut string, guests int) RoomFilter { // one page holds every result
		return RoomFilter{HotelSearch: hotel, CheckIn: date(checkIn), CheckOut: date(checkOut), Guests: guests, Limit: 100}
	}
	allRooms := func(hotel string) []string {
		return []string{hotel + "/101", hotel + "/102", hotel + "/201", hotel + "/202", hotel + "/301", hotel + "/302"}
	}

	tests := []struct {
		name   string
		filter RoomFilter
		want   []string // "hotel/number"
	}{
		{"no bookings that week: every room, sorted by hotel then number", search(nil, "2027-03-01", "2027-03-02", 1),
			append(allRooms(seaside), allRooms(grand)...)},
		{"every hotel, booked rooms left out", search(nil, "2027-01-11", "2027-01-13", 1),
			[]string{"Seaside Inn/101", "Seaside Inn/102", "Seaside Inn/201", "Seaside Inn/301", "Seaside Inn/302",
				"The Grand Hotel/102", "The Grand Hotel/202", "The Grand Hotel/302"}},
		{"every hotel, 2 guests", search(nil, "2027-01-11", "2027-01-13", 2),
			[]string{"Seaside Inn/201", "Seaside Inn/301", "Seaside Inn/302", "The Grand Hotel/202", "The Grand Hotel/302"}},
		{"one hotel: one of each type free", search(&grand, "2027-01-11", "2027-01-13", 1),
			[]string{"The Grand Hotel/102", "The Grand Hotel/202", "The Grand Hotel/302"}},
		{"hotel by part of its name, any case", search(&partial, "2027-01-11", "2027-01-13", 1),
			[]string{"The Grand Hotel/102", "The Grand Hotel/202", "The Grand Hotel/302"}},
		{"capacity filters out singles", search(&grand, "2027-01-11", "2027-01-13", 2),
			[]string{"The Grand Hotel/202", "The Grand Hotel/302"}},
		{"only deluxe fits 3", search(&grand, "2027-01-11", "2027-01-13", 3),
			[]string{"The Grand Hotel/302"}},
		{"nothing fits 5", search(&grand, "2027-01-11", "2027-01-13", 5), nil},
		{"free from the day a booking checks out", search(&grand, "2027-01-16", "2027-01-17", 2),
			[]string{"The Grand Hotel/201", "The Grand Hotel/202", "The Grand Hotel/301", "The Grand Hotel/302"}},
		{"unknown hotel", search(&nope, "2027-01-11", "2027-01-13", 1), nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rooms, err := s.FindRooms(t.Context(), tt.filter)
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, r := range rooms {
				got = append(got, r.HotelName+"/"+r.Number)
				if r.Capacity < tt.filter.Guests || r.HotelID == 0 || r.Type == "" {
					t.Errorf("room %+v not fully populated or too small", r)
				}
			}
			if !slices.Equal(got, tt.want) {
				t.Fatalf("got %v\nwant %v", got, tt.want)
			}
		})
	}
}

// Paging through every free room, 3 at a time, returns each room exactly once, in order.
func TestFindRoomsPages(t *testing.T) {
	s := seededTestStore(t)
	grand := "The Grand Hotel"
	page := func(hotel *string, after *RoomKey, limit int) []string {
		t.Helper()
		rooms, err := s.FindRooms(t.Context(), RoomFilter{
			HotelSearch: hotel, CheckIn: date("2027-01-11"), CheckOut: date("2027-01-13"), Guests: 1, After: after, Limit: limit,
		})
		if err != nil {
			t.Fatal(err)
		}
		var got []string
		for _, r := range rooms {
			got = append(got, r.HotelName+"/"+r.Number)
		}
		return got
	}

	// Over 11-13 Jan, 8 rooms are free across both hotels (see TestFindRooms).
	pages := [][]string{
		{"Seaside Inn/101", "Seaside Inn/102", "Seaside Inn/201"},
		{"Seaside Inn/301", "Seaside Inn/302", "The Grand Hotel/102"},
		{"The Grand Hotel/202", "The Grand Hotel/302"},
	}
	var after *RoomKey
	for i, want := range pages {
		got := page(nil, after, 3)
		if !slices.Equal(got, want) {
			t.Fatalf("page %d: got %v\nwant %v", i+1, got, want)
		}
		last := strings.SplitN(got[len(got)-1], "/", 2)
		after = &RoomKey{HotelName: last[0], Number: last[1]}
	}
	if got := page(nil, after, 3); got != nil {
		t.Fatalf("after the last room: got %v, want nothing", got)
	}

	// A cursor from one hotel's page carries on within that hotel when the search names it.
	if got, want := page(&grand, &RoomKey{HotelName: grand, Number: "102"}, 3), []string{"The Grand Hotel/202", "The Grand Hotel/302"}; !slices.Equal(got, want) {
		t.Fatalf("one hotel after 102: got %v\nwant %v", got, want)
	}
	// After a room that's now booked, paging still starts at the next room in order.
	if got, want := page(nil, &RoomKey{HotelName: grand, Number: "101"}, 3), []string{"The Grand Hotel/102", "The Grand Hotel/202", "The Grand Hotel/302"}; !slices.Equal(got, want) {
		t.Fatalf("after a booked room: got %v\nwant %v", got, want)
	}
}

func TestCreateBooking(t *testing.T) {
	s := seededTestStore(t)
	grand := func(number string) int64 { return roomID(t, s, "The Grand Hotel", number) }

	b, err := s.CreateBooking(t.Context(), NewBooking{
		RoomID: grand("102"), GuestName: "Jane Doe", Guests: 1, CheckIn: date("2027-01-11"), CheckOut: date("2027-01-13"),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if b.Reference.Version() != 4 { // gen_random_uuid() makes random (version 4) UUIDs; the zero UUID would mean it wasn't set
		t.Errorf("reference %s isn't a database-generated UUID", b.Reference)
	}
	if b.GuestName != "Jane Doe" || b.Guests != 1 || !b.CheckIn.Equal(date("2027-01-11")) || !b.CheckOut.Equal(date("2027-01-13")) ||
		b.CreatedAt.IsZero() || b.Room.Number != "102" || b.Room.HotelName != "The Grand Hotel" || b.Room.Capacity != 1 {
		t.Errorf("booking not fully populated: %+v", b)
	}

	tests := []struct {
		name              string
		room              int64
		guests            int
		checkIn, checkOut string
		wantErr           error
	}{
		{"overlaps an existing booking", grand("101"), 1, "2027-01-12", "2027-01-14", ErrRoomUnavailable},
		{"overlaps the booking just made", grand("102"), 1, "2027-01-12", "2027-01-13", ErrRoomUnavailable},
		{"more guests than capacity", grand("202"), 3, "2027-02-01", "2027-02-02", ErrOverCapacity},
		{"room doesn't exist", 999999, 1, "2027-02-01", "2027-02-02", ErrRoomNotFound},
		{"checks in the day another checks out", grand("101"), 1, "2027-01-13", "2027-01-15", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.CreateBooking(t.Context(), NewBooking{
				RoomID: tt.room, GuestName: "Guest", Guests: tt.guests, CheckIn: date(tt.checkIn), CheckOut: date(tt.checkOut),
			})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("want %v, got %v", tt.wantErr, err)
			}
		})
	}
}

// Many requests for the same room and nights at once: exactly one may win, the rest must be told it's taken.
func TestCreateBookingConcurrent(t *testing.T) {
	s := seededTestStore(t)
	room := roomID(t, s, "The Grand Hotel", "302")

	const attempts = 10
	errs := make([]error, attempts)
	var wg sync.WaitGroup
	for i := range attempts {
		wg.Go(func() {
			_, errs[i] = s.CreateBooking(t.Context(), NewBooking{
				RoomID: room, GuestName: "Racer", Guests: 2, CheckIn: date("2027-02-01"), CheckOut: date("2027-02-03"),
			})
		})
	}
	wg.Wait()

	var booked, unavailable int
	for _, err := range errs {
		switch {
		case err == nil:
			booked++
		case errors.Is(err, ErrRoomUnavailable):
			unavailable++
		default:
			t.Errorf("unexpected error: %v", err)
		}
	}
	if booked != 1 || unavailable != attempts-1 {
		t.Fatalf("booked %d, unavailable %d; want 1 and %d", booked, unavailable, attempts-1)
	}
}

func TestGetBooking(t *testing.T) {
	s := seededTestStore(t)

	b, err := s.GetBooking(t.Context(), uuid.MustParse("5eed0000-0000-4000-8000-000000000002"))
	if err != nil {
		t.Fatal(err)
	}
	if b.GuestName != "Bob Jones" || b.Guests != 2 || !b.CheckIn.Equal(date("2027-01-10")) || !b.CheckOut.Equal(date("2027-01-14")) ||
		b.Room.Number != "201" || b.Room.Type != RoomTypeDouble || b.Room.Capacity != 2 || b.Room.HotelName != "The Grand Hotel" ||
		b.RoomID != b.Room.ID {
		t.Errorf("got %+v", b)
	}

	if _, err := s.GetBooking(t.Context(), uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}
