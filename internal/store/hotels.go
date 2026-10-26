package store

import (
	"context"
	"fmt"
)

// HotelFilter describes a hotel name search, and which page of the results to return.
type HotelFilter struct {
	Search string // matched fuzzily against hotel names

	After *string // nil starts from the first match; otherwise only hotels whose name sorts after this
	Limit int     // the most hotels to return
}

// FindHotels returns up to f.Limit hotels whose name is a fuzzy match for f.Search, sorted by name,
// starting after f.After.
//
// Matching is pg_trgm word similarity ($1 <% name): a search matches a name when it's similar enough to
// some run of words in it, ignoring case, so whole or partial words and small misspellings match
// ("gran", "seasid", "grande"). Searches of one or two letters are too short to match anything. The
// search is plain text, not a pattern, so it needs no escaping.
//
// Names are unique, so paging is keyset-based on the name alone. The MATERIALIZED CTE finds the matches
// through hotels_name_trgm_idx first; see roomsInMatchingHotels.
func (s *Store) FindHotels(ctx context.Context, f HotelFilter) ([]Hotel, error) {
	hotels := []Hotel{}
	err := s.db.SelectContext(ctx, &hotels, `
		WITH matching_hotels AS MATERIALIZED (SELECT id, name FROM hotels WHERE $1 <% name)
		SELECT id, name
		FROM matching_hotels
		WHERE $2::text IS NULL OR name > $2::text
		ORDER BY name
		LIMIT $3`,
		f.Search, f.After, f.Limit)
	if err != nil {
		return nil, fmt.Errorf("find hotels: %w", err)
	}
	return hotels, nil
}
