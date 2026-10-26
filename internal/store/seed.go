package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// ErrAlreadySeeded is returned by Seed when the seed hotels already exist.
var ErrAlreadySeeded = errors.New("database already seeded; reset it first")

type seedHotel struct {
	name  string
	rooms []seedRoom
}

type seedRoom struct {
	number   string
	roomType RoomType
}

// Every hotel gets the same six rooms: two of each type.
var seedRooms = []seedRoom{
	{"101", RoomTypeSingle}, {"102", RoomTypeSingle},
	{"201", RoomTypeDouble}, {"202", RoomTypeDouble},
	{"301", RoomTypeDeluxe}, {"302", RoomTypeDeluxe},
}

var seedHotels = []seedHotel{
	{name: "The Grand Hotel", rooms: seedRooms},
	{name: "Seaside Inn", rooms: seedRooms},
}

// Seed inserts the test hotels and rooms in one transaction. It adds no bookings: bookings can't start in
// the past, so seeded ones with fixed dates would eventually stop working. Testers make them with the API.
func (s *Store) Seed(ctx context.Context) error {
	return s.inTx(ctx, seed)
}

// Reset removes all hotels, rooms and bookings. room_types is reference data from the migration and stays.
func (s *Store) Reset(ctx context.Context) error {
	return s.inTx(ctx, reset)
}

func (s *Store) inTx(ctx context.Context, fn func(context.Context, *sqlx.Tx) error) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // no-op after a successful commit
	if err := fn(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}

func seed(ctx context.Context, tx *sqlx.Tx) error {
	for _, h := range seedHotels {
		var hotelID int64
		err := tx.GetContext(ctx, &hotelID, `INSERT INTO hotels (name) VALUES ($1) RETURNING id`, h.name)
		if constraint, ok := pgConstraintError(err, codeUniqueViolation); ok && constraint == "hotels_name_key" {
			return ErrAlreadySeeded
		}
		if err != nil {
			return fmt.Errorf("insert hotel %q: %w", h.name, err)
		}
		for _, r := range h.rooms {
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO rooms (hotel_id, number, type) VALUES ($1, $2, $3)`,
				hotelID, r.number, r.roomType); err != nil {
				return fmt.Errorf("insert room %s/%s: %w", h.name, r.number, err)
			}
		}
	}
	return nil
}

func reset(ctx context.Context, tx *sqlx.Tx) error {
	if _, err := tx.ExecContext(ctx, `TRUNCATE bookings, rooms, hotels RESTART IDENTITY`); err != nil {
		return fmt.Errorf("truncate: %w", err)
	}
	return nil
}
