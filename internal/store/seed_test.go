package store

import (
	"errors"
	"testing"
)

// Seed and reset run inside a transaction that's rolled back, so the dev db's data is untouched.
func TestSeedAndReset(t *testing.T) {
	s := openTestStore(t)
	ctx := t.Context()

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Start from empty, whatever the dev db holds.
	if err := reset(ctx, tx); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if err := seed(ctx, tx); err != nil {
		t.Fatalf("seed: %v", err)
	}

	count := func(query string) int {
		t.Helper()
		var n int
		if err := tx.GetContext(ctx, &n, query); err != nil {
			t.Fatalf("%s: %v", query, err)
		}
		return n
	}

	if got := count(`SELECT count(*) FROM hotels`); got != len(seedHotels) {
		t.Errorf("hotels = %d, want %d", got, len(seedHotels))
	}
	if got := count(`SELECT count(*) FROM bookings`); got != len(seedBookings) {
		t.Errorf("bookings = %d, want %d", got, len(seedBookings))
	}
	// Business rules the schema doesn't enforce, so the seed data has to.
	if got := count(`SELECT count(*) FROM (SELECT hotel_id FROM rooms GROUP BY hotel_id HAVING count(*) <> 6) x`); got != 0 {
		t.Errorf("%d hotels don't have exactly 6 rooms", got)
	}
	if got := count(`SELECT count(*) FROM (SELECT hotel_id, type FROM rooms GROUP BY hotel_id, type HAVING count(*) <> 2) x`); got != 0 {
		t.Errorf("%d hotel/type pairs don't have exactly 2 rooms", got)
	}
	if got := count(`SELECT count(*) FROM bookings b JOIN rooms r ON r.id = b.room_id
		JOIN room_types rt ON rt.type = r.type WHERE b.guests > rt.capacity`); got != 0 {
		t.Errorf("%d bookings exceed their room's capacity", got)
	}

	// Seeding again must fail cleanly rather than duplicate data. The failed insert aborts the
	// transaction, so this is the last check that uses it.
	if err := seed(ctx, tx); !errors.Is(err, ErrAlreadySeeded) {
		t.Fatalf("second seed: want ErrAlreadySeeded, got %v", err)
	}
}

func TestReset(t *testing.T) {
	s := openTestStore(t)
	ctx := t.Context()

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Reset, seed, reset: the first reset means seed can't fail on data the dev db already has.
	if err := reset(ctx, tx); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if err := seed(ctx, tx); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := reset(ctx, tx); err != nil {
		t.Fatalf("reset: %v", err)
	}

	for _, table := range []string{"hotels", "rooms", "bookings"} {
		var n int
		if err := tx.GetContext(ctx, &n, `SELECT count(*) FROM `+table); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if n != 0 {
			t.Errorf("%s has %d rows after reset, want 0", table, n)
		}
	}
	var roomTypes int
	if err := tx.GetContext(ctx, &roomTypes, `SELECT count(*) FROM room_types`); err != nil {
		t.Fatalf("count room_types: %v", err)
	}
	if roomTypes != 3 {
		t.Errorf("room_types has %d rows after reset, want 3 (reference data must survive)", roomTypes)
	}
}
