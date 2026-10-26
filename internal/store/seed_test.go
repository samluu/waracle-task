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
	if got := count(`SELECT count(*) FROM bookings`); got != 0 {
		t.Errorf("bookings = %d, want 0: seeding is only hotels and rooms", got)
	}
	// Business rules the schema doesn't enforce, so the seed data has to.
	if got := count(`SELECT count(*) FROM (SELECT hotel_id FROM rooms GROUP BY hotel_id HAVING count(*) <> 6) x`); got != 0 {
		t.Errorf("%d hotels don't have exactly 6 rooms", got)
	}
	if got := count(`SELECT count(*) FROM (SELECT hotel_id, type FROM rooms GROUP BY hotel_id, type HAVING count(*) <> 2) x`); got != 0 {
		t.Errorf("%d hotel/type pairs don't have exactly 2 rooms", got)
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
	// Seeding adds no bookings, so make one as the API would, for reset to remove.
	if _, err := tx.ExecContext(ctx, `INSERT INTO bookings (room_id, guest_name, guests, check_in, check_out)
		SELECT id, 'Guest', 1, '2027-01-11', '2027-01-12' FROM rooms LIMIT 1`); err != nil {
		t.Fatalf("insert booking: %v", err)
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
