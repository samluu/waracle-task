package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
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

type seedBooking struct {
	reference         string
	hotel, room       string
	guestName         string
	guests            int
	checkIn, checkOut string // YYYY-MM-DD
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

// Fixed references and dates so testers know what to search for. Over 11-13 Jan 2027 at The Grand Hotel,
// one room of each type is booked and the other is free.
var seedBookings = []seedBooking{
	{"SEED-0001", "The Grand Hotel", "101", "Alice Smith", 1, "2027-01-11", "2027-01-13"},
	{"SEED-0002", "The Grand Hotel", "201", "Bob Jones", 2, "2027-01-10", "2027-01-14"},
	{"SEED-0003", "The Grand Hotel", "301", "Carol White", 4, "2027-01-12", "2027-01-15"},
	{"SEED-0004", "The Grand Hotel", "201", "Dan Brown", 2, "2027-01-14", "2027-01-16"}, // checks in the day 0002 checks out
	{"SEED-0005", "Seaside Inn", "202", "Erin Green", 1, "2027-01-11", "2027-01-12"},
}

// Seed inserts the test hotels, rooms and bookings in one transaction.
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
	roomIDs := map[[2]string]int64{} // {hotel name, room number} -> room id
	for _, h := range seedHotels {
		var hotelID int64
		err := tx.GetContext(ctx, &hotelID, `INSERT INTO hotels (name) VALUES ($1) RETURNING id`, h.name)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.ConstraintName == "hotels_name_key" {
			return ErrAlreadySeeded
		}
		if err != nil {
			return fmt.Errorf("insert hotel %q: %w", h.name, err)
		}
		for _, r := range h.rooms {
			var roomID int64
			if err := tx.GetContext(ctx, &roomID,
				`INSERT INTO rooms (hotel_id, number, type) VALUES ($1, $2, $3) RETURNING id`,
				hotelID, r.number, r.roomType); err != nil {
				return fmt.Errorf("insert room %s/%s: %w", h.name, r.number, err)
			}
			roomIDs[[2]string{h.name, r.number}] = roomID
		}
	}

	for _, b := range seedBookings {
		roomID, ok := roomIDs[[2]string{b.hotel, b.room}]
		if !ok {
			return fmt.Errorf("booking %s: unknown room %s/%s", b.reference, b.hotel, b.room)
		}
		checkIn, err := time.Parse(time.DateOnly, b.checkIn)
		if err != nil {
			return fmt.Errorf("booking %s: %w", b.reference, err)
		}
		checkOut, err := time.Parse(time.DateOnly, b.checkOut)
		if err != nil {
			return fmt.Errorf("booking %s: %w", b.reference, err)
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO bookings (reference, room_id, guest_name, guests, check_in, check_out)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			b.reference, roomID, b.guestName, b.guests, checkIn, checkOut); err != nil {
			return fmt.Errorf("insert booking %s: %w", b.reference, err)
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
