package handlers

import (
	"encoding/base64"
	"encoding/json"
	"errors"

	"github.com/samluu/waracle-task/internal/store"
)

var errInvalidCursor = errors.New("invalid cursor")

// cursor is what a room-search cursor encodes: the last room of a page, so the next page starts after it.
type cursor struct {
	HotelName string `json:"h"`
	Number    string `json:"n"`
}

// encodeCursor returns an opaque cursor pointing after key. Clients pass it back unchanged; the encoding
// isn't part of the API, so it can change without breaking them.
func encodeCursor(key store.RoomKey) string {
	b, _ := json.Marshal(cursor{HotelName: key.HotelName, Number: key.Number}) // can't fail for two strings
	return base64.RawURLEncoding.EncodeToString(b)
}

// decodeCursor reverses encodeCursor, rejecting anything it didn't produce.
func decodeCursor(s string) (store.RoomKey, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return store.RoomKey{}, errInvalidCursor
	}
	var c cursor
	if err := json.Unmarshal(b, &c); err != nil || c.HotelName == "" || c.Number == "" {
		return store.RoomKey{}, errInvalidCursor
	}
	return store.RoomKey{HotelName: c.HotelName, Number: c.Number}, nil
}
