package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// FindHotelByName returns the hotel with exactly this name, or ErrNotFound.
func (s *Store) FindHotelByName(ctx context.Context, name string) (Hotel, error) {
	var h Hotel
	err := s.db.GetContext(ctx, &h, `SELECT id, name FROM hotels WHERE name = $1`, name)
	if errors.Is(err, sql.ErrNoRows) {
		return Hotel{}, ErrNotFound
	}
	if err != nil {
		return Hotel{}, fmt.Errorf("find hotel: %w", err)
	}
	return h, nil
}
