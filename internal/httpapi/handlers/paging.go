package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
)

const (
	defaultPageSize = 20
	maxPageSize     = 100
)

var errInvalidCursor = errors.New("invalid cursor")

// A cursor points after the last item of a page, so the next page starts after it.
type cursor interface {
	hotelCursor | roomCursor
	valid() bool
}

type hotelCursor struct {
	HotelName string `json:"h"`
}

func (c hotelCursor) valid() bool { return c.HotelName != "" }

type roomCursor struct {
	HotelName string `json:"h"`
	Number    string `json:"n"`
}

func (c roomCursor) valid() bool { return c.HotelName != "" && c.Number != "" }

// encodeCursor returns an opaque cursor for c. Clients pass it back unchanged; the encoding isn't part
// of the API, so it can change without breaking them.
func encodeCursor[C cursor](c C) string {
	b, _ := json.Marshal(c) // can't fail for a struct of strings
	return base64.RawURLEncoding.EncodeToString(b)
}

// decodeCursor reverses encodeCursor, rejecting anything it didn't produce: invalid base64 or JSON,
// unknown fields, or a missing field.
func decodeCursor[C cursor](s string) (C, error) {
	var c C
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return c, errInvalidCursor
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil || !c.valid() {
		return c, errInvalidCursor
	}
	return c, nil
}

// pageSize returns the requested page size, or the default when none was given.
func pageSize(limit *int) (int, error) {
	if limit == nil {
		return defaultPageSize, nil
	}
	if *limit < 1 || *limit > maxPageSize {
		return 0, fmt.Errorf("limit must be between 1 and %d", maxPageSize)
	}
	return *limit, nil
}
