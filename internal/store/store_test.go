package store

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// openTestStore connects to TEST_DATABASE_URL, or skips the test if it isn't set.
// The database must already be migrated; `make test-integration` does that first.
func openTestStore(t *testing.T) *Store {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}
	s, err := Open(t.Context(), url)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	// Tests reset and seed data, so refuse to run against anything that isn't a test database.
	var name string
	if err := s.db.GetContext(t.Context(), &name, `SELECT current_database()`); err != nil {
		t.Fatalf("current_database: %v", err)
	}
	if !strings.HasSuffix(name, "_test") {
		t.Fatalf("TEST_DATABASE_URL points at %q; integration tests only run against a *_test database", name)
	}
	return s
}

func date(s string) time.Time {
	d, err := time.Parse(time.DateOnly, s)
	if err != nil {
		panic(err)
	}
	return d
}

// testBookings are the bookings the query tests search around, added by seededTestStore. The store
// doesn't check dates against today, so fixed dates don't go stale here. Over 11-13 Jan at The Grand
// Hotel, one room of each type is booked and the other is free.
var testBookings = []struct {
	reference         string
	hotel, room       string
	guestName         string
	guests            int
	checkIn, checkOut string
}{
	{"5eed0000-0000-4000-8000-000000000001", "The Grand Hotel", "101", "Alice Smith", 1, "2027-01-11", "2027-01-13"},
	{"5eed0000-0000-4000-8000-000000000002", "The Grand Hotel", "201", "Bob Jones", 2, "2027-01-10", "2027-01-14"},
	{"5eed0000-0000-4000-8000-000000000003", "The Grand Hotel", "301", "Carol White", 4, "2027-01-12", "2027-01-15"},
	{"5eed0000-0000-4000-8000-000000000004", "The Grand Hotel", "201", "Dan Brown", 2, "2027-01-14", "2027-01-16"}, // checks in the day booking 2 checks out
	{"5eed0000-0000-4000-8000-000000000005", "Seaside Inn", "202", "Erin Green", 1, "2027-01-11", "2027-01-12"},
}

// seededTestStore is openTestStore with the database reset, freshly seeded, and testBookings added.
func seededTestStore(t *testing.T) *Store {
	t.Helper()
	s := openTestStore(t)
	if err := s.Reset(t.Context()); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if err := s.Seed(t.Context()); err != nil {
		t.Fatalf("seed: %v", err)
	}
	for _, b := range testBookings {
		if _, err := s.db.ExecContext(t.Context(),
			`INSERT INTO bookings (reference, room_id, guest_name, guests, check_in, check_out) VALUES ($1, $2, $3, $4, $5, $6)`,
			b.reference, roomID(t, s, b.hotel, b.room), b.guestName, b.guests, date(b.checkIn), date(b.checkOut)); err != nil {
			t.Fatalf("insert booking %s: %v", b.reference, err)
		}
	}
	return s
}

func TestSchemaConstraints(t *testing.T) {
	s := openTestStore(t)
	ctx := t.Context()

	var hotelID, roomID int64
	if err := s.db.GetContext(ctx, &hotelID, `INSERT INTO hotels (name) VALUES ('Test Hotel') RETURNING id`); err != nil {
		t.Fatalf("insert hotel: %v", err)
	}
	// Deleting the hotel cascades to its rooms and bookings, so the test leaves no rows behind.
	// t.Context() is already cancelled when cleanups run, so detach from its cancellation.
	t.Cleanup(func() {
		_, _ = s.db.ExecContext(context.WithoutCancel(t.Context()), `DELETE FROM hotels WHERE id = $1`, hotelID)
	})
	if err := s.db.GetContext(ctx, &roomID,
		`INSERT INTO rooms (hotel_id, number, type) VALUES ($1, '101', 'double') RETURNING id`, hotelID); err != nil {
		t.Fatalf("insert room: %v", err)
	}

	// Fixed references are safe across runs: the cleanup above deletes these bookings with the hotel.
	book := func(ref, checkIn, checkOut string) error {
		_, err := s.db.ExecContext(ctx,
			`INSERT INTO bookings (reference, room_id, guest_name, guests, check_in, check_out)
			 VALUES ($1, $2, 'Guest', 2, $3, $4)`,
			ref, roomID, checkIn, checkOut)
		return err
	}

	const firstRef = "00000000-0000-4000-8000-00000000000a"
	if err := book(firstRef, "2030-01-10", "2030-01-13"); err != nil {
		t.Fatalf("first booking: %v", err)
	}

	tests := []struct {
		name              string
		ref               string
		checkIn, checkOut string
		wantCode          string // "" means the insert should succeed
	}{
		{"overlapping nights", "00000000-0000-4000-8000-00000000000b", "2030-01-12", "2030-01-15", codeExclusionViolation},
		{"inside existing stay", "00000000-0000-4000-8000-00000000000c", "2030-01-11", "2030-01-12", codeExclusionViolation},
		{"check-in on previous check-out day", "00000000-0000-4000-8000-00000000000d", "2030-01-13", "2030-01-14", ""},
		{"check-out on existing check-in day", "00000000-0000-4000-8000-00000000000e", "2030-01-08", "2030-01-10", ""},
		{"duplicate reference", firstRef, "2030-02-01", "2030-02-02", codeUniqueViolation},
		{"check-out before check-in", "00000000-0000-4000-8000-00000000000f", "2030-03-05", "2030-03-04", codeCheckViolation},
		{"zero nights", "00000000-0000-4000-8000-000000000010", "2030-03-05", "2030-03-05", codeCheckViolation},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := book(tt.ref, tt.checkIn, tt.checkOut)
			if tt.wantCode == "" {
				if err != nil {
					t.Fatalf("want success, got %v", err)
				}
				return
			}
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) || pgErr.Code != tt.wantCode {
				t.Fatalf("want SQLSTATE %s, got %v", tt.wantCode, err)
			}
		})
	}
}

func TestRoomTypesSeeded(t *testing.T) {
	s := openTestStore(t)

	var got []struct {
		Type     RoomType `db:"type"`
		Capacity int      `db:"capacity"`
	}
	if err := s.db.SelectContext(t.Context(), &got, `SELECT type, capacity FROM room_types ORDER BY capacity`); err != nil {
		t.Fatalf("select room_types: %v", err)
	}
	want := []struct {
		Type     RoomType
		Capacity int
	}{{RoomTypeSingle, 1}, {RoomTypeDouble, 2}, {RoomTypeDeluxe, 4}}
	if len(got) != len(want) {
		t.Fatalf("got %d room types, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i].Type != want[i].Type || got[i].Capacity != want[i].Capacity {
			t.Fatalf("room_types[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}
