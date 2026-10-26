package store

import (
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// Postgres error codes the schema's constraints raise.
const (
	codeUniqueViolation    = "23505"
	codeCheckViolation     = "23514"
	codeExclusionViolation = "23P01"
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
	t.Cleanup(func() { _, _ = s.db.Exec(`DELETE FROM hotels WHERE id = $1`, hotelID) })
	if err := s.db.GetContext(ctx, &roomID,
		`INSERT INTO rooms (hotel_id, number, type) VALUES ($1, '101', 'double') RETURNING id`, hotelID); err != nil {
		t.Fatalf("insert room: %v", err)
	}

	// Unique per run, so the test doesn't collide with existing data.
	prefix := fmt.Sprintf("T%d-", time.Now().UnixNano())
	book := func(ref, checkIn, checkOut string) error {
		_, err := s.db.ExecContext(ctx,
			`INSERT INTO bookings (reference, room_id, guest_name, guests, check_in, check_out)
			 VALUES ($1, $2, 'Guest', 2, $3, $4)`,
			prefix+ref, roomID, checkIn, checkOut)
		return err
	}

	if err := book("A", "2030-01-10", "2030-01-13"); err != nil {
		t.Fatalf("first booking: %v", err)
	}

	tests := []struct {
		name              string
		ref               string
		checkIn, checkOut string
		wantCode          string // "" means the insert should succeed
	}{
		{"overlapping nights", "B", "2030-01-12", "2030-01-15", codeExclusionViolation},
		{"inside existing stay", "C", "2030-01-11", "2030-01-12", codeExclusionViolation},
		{"check-in on previous check-out day", "D", "2030-01-13", "2030-01-14", ""},
		{"check-out on existing check-in day", "E", "2030-01-08", "2030-01-10", ""},
		{"duplicate reference", "A", "2030-02-01", "2030-02-02", codeUniqueViolation},
		{"check-out before check-in", "F", "2030-03-05", "2030-03-04", codeCheckViolation},
		{"zero nights", "G", "2030-03-05", "2030-03-05", codeCheckViolation},
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
