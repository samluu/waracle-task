package handlers

import (
	"encoding/base64"
	"errors"
	"testing"

	"github.com/samluu/waracle-task/internal/store"
)

func TestCursorRoundTrip(t *testing.T) {
	for _, key := range []store.RoomKey{
		{HotelName: "The Grand Hotel", Number: "102"},
		{HotelName: `Odd "name" / with ? & symbols`, Number: "A1"},
	} {
		got, err := decodeCursor(encodeCursor(key))
		assertResponse(t, got, err, key, nil)
	}
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
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := decodeCursor(c); !errors.Is(err, errInvalidCursor) {
				t.Fatalf("decodeCursor(%q) = %v, want errInvalidCursor", c, err)
			}
		})
	}
}
