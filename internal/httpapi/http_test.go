package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/samluu/waracle-task/internal/httpapi/handlers/mocks"
	"github.com/samluu/waracle-task/internal/store"
)

// These go through the full router with raw requests and JSON responses. They cover only what the
// handler tests can't: the wire format clients depend on (a renamed field passes struct-based tests,
// because both sides use the same generated struct), and code that only runs at the HTTP layer:
// the body-size middleware and the param, request and response error handlers.
func TestHTTP(t *testing.T) {
	tests := []struct {
		name       string
		st         *mocks.StoreMock // only the methods this request should reach are set; any other call panics
		method     string
		target     string
		body       string
		wantStatus int
		want       string
	}{
		// Wire format. The booking response nests Room and Hotel, so this pins all three schemas.
		{"create booking",
			&mocks.StoreMock{CreateBookingFunc: func(context.Context, store.NewBooking) (store.BookingDetails, error) {
				return testBooking, nil
			}},
			http.MethodPost, "/bookings",
			`{"room_id":2,"guest_name":"Jane Doe","guests":1,"check_in":"2027-01-11","check_out":"2027-01-13"}`,
			http.StatusCreated, testBookingJSON},
		{"get booking",
			&mocks.StoreMock{GetBookingFunc: func(context.Context, uuid.UUID) (store.BookingDetails, error) {
				return testBooking, nil
			}},
			http.MethodGet, "/bookings/" + testRef, "",
			http.StatusOK, testBookingJSON},
		{"health", &mocks.StoreMock{PingFunc: func(context.Context) error { return nil }},
			http.MethodGet, "/health", "",
			http.StatusOK, `{"status":"ok"}`},

		// HTTP-layer error handling. None of the first four reach the store.
		{"unparseable parameter", &mocks.StoreMock{}, http.MethodGet, "/hotels", "", // paramError, in the generated code's wording
			http.StatusBadRequest, `{"error":"Query argument name is required, but not found"}`},
		{"malformed booking reference", &mocks.StoreMock{}, http.MethodGet, "/bookings/BK-ABCDEFGH", "", // paramError: the spec says format uuid
			http.StatusBadRequest, `{"error":"Invalid format for parameter reference: error unmarshaling 'BK-ABCDEFGH' text as *uuid.UUID: invalid UUID length: 11"}`},
		{"undecodable body", &mocks.StoreMock{}, http.MethodPost, "/bookings", `{"room_id":`, // requestError
			http.StatusBadRequest, `{"error":"invalid request body: unexpected EOF"}`},
		{"body too large", &mocks.StoreMock{}, http.MethodPost, "/bookings", // limitBody middleware + requestError
			`{"guest_name":"` + strings.Repeat("A", maxBodyBytes) + `"}`,
			http.StatusRequestEntityTooLarge, `{"error":"request body must be at most 65536 bytes"}`},
		{"unexpected error is hidden", // responseError
			&mocks.StoreMock{GetBookingFunc: func(context.Context, uuid.UUID) (store.BookingDetails, error) {
				return store.BookingDetails{}, errStore
			}},
			http.MethodGet, "/bookings/" + testRef, "",
			http.StatusInternalServerError, `{"error":"internal error"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := serve(t, tt.st, tt.method, tt.target, tt.body)
			assertJSON(t, rec, tt.wantStatus, tt.want)
			if tt.wantStatus == http.StatusCreated {
				if loc := rec.Header().Get("Location"); loc != "/bookings/"+testRef {
					t.Fatalf("Location = %q", loc)
				}
			}
		})
	}
}

// Pins the room search's page format: the rooms array, next_cursor only when there's another page, and
// an empty page as [] rather than null. The cursor itself is opaque, so only its presence is checked.
func TestHTTPRoomPage(t *testing.T) {
	const search = "/rooms?check_in=2027-01-11&check_out=2027-01-13&guests=1&limit=1"
	twoRooms := &mocks.StoreMock{FindRoomsFunc: func(context.Context, store.RoomFilter) ([]store.Room, error) {
		return []store.Room{testBooking.Room, testBooking.Room}, nil // one more than the page: there's a next page
	}}
	var page struct {
		Rooms      []json.RawMessage `json:"rooms"`
		NextCursor *string           `json:"next_cursor"`
	}
	rec := serve(t, twoRooms, http.MethodGet, search, "")
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil || rec.Code != http.StatusOK {
		t.Fatalf("status %d, body %s (%v)", rec.Code, rec.Body, err)
	}
	if len(page.Rooms) != 1 || page.NextCursor == nil || *page.NextCursor == "" {
		t.Fatalf("want 1 room and a next_cursor, got %s", rec.Body)
	}
	assertJSON(t, serve(t, &mocks.StoreMock{FindRoomsFunc: func(context.Context, store.RoomFilter) ([]store.Room, error) {
		return []store.Room{testBooking.Room}, nil
	}}, http.MethodGet, search, ""),
		http.StatusOK, `{"rooms":[{"id":2,"number":"102","type":"single","capacity":1,"hotel":{"id":1,"name":"The Grand Hotel"}}]}`)
	assertJSON(t, serve(t, &mocks.StoreMock{FindRoomsFunc: func(context.Context, store.RoomFilter) ([]store.Room, error) {
		return nil, nil
	}}, http.MethodGet, search, ""),
		http.StatusOK, `{"rooms":[]}`)
}

// Pins the hotel search's page format, as TestHTTPRoomPage does for rooms.
func TestHTTPHotelPage(t *testing.T) {
	const search = "/hotels?name=hotel&limit=1"
	hotels := func(hs ...store.Hotel) *mocks.StoreMock {
		return &mocks.StoreMock{FindHotelsFunc: func(context.Context, store.HotelFilter) ([]store.Hotel, error) { return hs, nil }}
	}
	grand, seaside := store.Hotel{ID: 1, Name: "The Grand Hotel"}, store.Hotel{ID: 2, Name: "Seaside Inn"}

	var page struct {
		Hotels     []json.RawMessage `json:"hotels"`
		NextCursor *string           `json:"next_cursor"`
	}
	rec := serve(t, hotels(seaside, grand), http.MethodGet, search, "") // one more than the page: there's a next page
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil || rec.Code != http.StatusOK {
		t.Fatalf("status %d, body %s (%v)", rec.Code, rec.Body, err)
	}
	if len(page.Hotels) != 1 || page.NextCursor == nil || *page.NextCursor == "" {
		t.Fatalf("want 1 hotel and a next_cursor, got %s", rec.Body)
	}
	assertJSON(t, serve(t, hotels(grand), http.MethodGet, search, ""),
		http.StatusOK, `{"hotels":[{"id":1,"name":"The Grand Hotel"}]}`)
	assertJSON(t, serve(t, hotels(), http.MethodGet, search, ""),
		http.StatusOK, `{"hotels":[]}`)
}

// Pins the admin endpoints' wire format: 204 with no body, and the JSON error when already seeded.
func TestHTTPAdmin(t *testing.T) {
	ok := &mocks.StoreMock{
		SeedFunc:  func(context.Context) error { return nil },
		ResetFunc: func(context.Context) error { return nil },
	}
	for _, path := range []string{"/admin/seed", "/admin/reset"} {
		rec := serve(t, ok, http.MethodPost, path, "")
		if rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
			t.Fatalf("POST %s: status %d, body %q; want 204 and no body", path, rec.Code, rec.Body)
		}
	}
	seeded := &mocks.StoreMock{SeedFunc: func(context.Context) error { return store.ErrAlreadySeeded }}
	assertJSON(t, serve(t, seeded, http.MethodPost, "/admin/seed", ""),
		http.StatusConflict, `{"error":"database already seeded; reset it first"}`)
}
