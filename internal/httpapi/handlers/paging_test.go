package handlers

import (
	"encoding/base64"
	"errors"
	"testing"
)

func TestCursorRoundTrip(t *testing.T) {
	for _, c := range []roomCursor{
		{HotelName: "The Grand Hotel", Number: "102"},
		{HotelName: `Odd "name" / with ? & symbols`, Number: "A1"},
	} {
		got, err := decodeCursor[roomCursor](encodeCursor(c))
		assertResponse(t, got, err, c, nil)
	}
	h := hotelCursor{HotelName: "Seaside Inn"}
	got, err := decodeCursor[hotelCursor](encodeCursor(h))
	assertResponse(t, got, err, h, nil)
}

func TestDecodeCursorRejects(t *testing.T) {
	b64 := func(s string) string { return base64.RawURLEncoding.EncodeToString([]byte(s)) }
	for name, c := range map[string]string{
		"empty":            "",
		"not base64":       "not a cursor!",
		"not JSON":         b64("hello"),
		"missing number":   b64(`{"h":"The Grand Hotel"}`),
		"missing hotel":    b64(`{"n":"102"}`),
		"wrong field type": b64(`{"h":1,"n":2}`),
		"unknown field":    b64(`{"h":"The Grand Hotel","n":"102","x":1}`),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := decodeCursor[roomCursor](c); !errors.Is(err, errInvalidCursor) {
				t.Fatalf("decodeCursor(%q) = %v, want errInvalidCursor", c, err)
			}
		})
	}
	// A room cursor isn't a hotel cursor: the extra field is rejected.
	if _, err := decodeCursor[hotelCursor](encodeCursor(roomCursor{HotelName: "H", Number: "1"})); !errors.Is(err, errInvalidCursor) {
		t.Fatalf("room cursor decoded as a hotel cursor: %v", err)
	}
}

func TestPageSize(t *testing.T) {
	for _, tt := range []struct {
		name    string
		limit   *int
		want    int
		wantErr string
	}{
		{"default", nil, 20, ""},
		{"smallest", new(1), 1, ""},
		{"largest", new(100), 100, ""},
		{"zero", new(0), 0, "limit must be between 1 and 100"},
		{"over the maximum", new(101), 0, "limit must be between 1 and 100"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := pageSize(tt.limit)
			assertErr(t, err, tt.wantErr)
			if got != tt.want {
				t.Fatalf("got %d, want %d", got, tt.want)
			}
		})
	}
}
