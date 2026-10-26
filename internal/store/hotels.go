package store

import (
	"context"
	"fmt"
)

// FindHotels returns the hotels whose name is a fuzzy match for search, sorted by name.
//
// Matching is pg_trgm word similarity ($1 <% name): search matches a name when it's similar enough to
// some run of words in it, ignoring case, so whole or partial words and small misspellings match
// ("gran", "seasid", "grande"). Searches of one or two letters are too short to match anything. The
// search is plain text, not a pattern, so it needs no escaping. Served by hotels_name_trgm_idx.
func (s *Store) FindHotels(ctx context.Context, search string) ([]Hotel, error) {
	hotels := []Hotel{}
	if err := s.db.SelectContext(ctx, &hotels,
		`SELECT id, name FROM hotels WHERE $1 <% name ORDER BY name`, search); err != nil {
		return nil, fmt.Errorf("find hotels: %w", err)
	}
	return hotels, nil
}
