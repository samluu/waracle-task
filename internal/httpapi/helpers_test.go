package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/samluu/waracle-task/internal/httpapi/handlers"
	"github.com/samluu/waracle-task/internal/store"
)

// testToday is the fixed "today" for the router tests.
var testToday = time.Date(2026, 10, 26, 15, 0, 0, 0, time.UTC)

// serve sends one request through a router backed by st, as a client would: tests using it give
// the request as strings and check the response body as a JSON string.
func serve(t *testing.T, st handlers.Store, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	rec := httptest.NewRecorder()
	srv := handlers.NewServer(st, func() time.Time { return testToday })
	newRouter(srv).ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, target, r))
	return rec
}

func date(s string) time.Time {
	d, err := time.Parse(time.DateOnly, s)
	if err != nil {
		panic(err)
	}
	return d
}

// testRef is the reference of testBooking.
const testRef = "3f2b8c1e-9a4d-4c6e-b1f0-7d2e5a9c4b13"

var (
	testBooking = store.BookingDetails{
		Booking: store.Booking{
			Reference: uuid.MustParse(testRef), RoomID: 2, GuestName: "Jane Doe", Guests: 1,
			CheckIn: date("2027-01-11"), CheckOut: date("2027-01-13"),
			CreatedAt: time.Date(2026, 10, 26, 15, 0, 0, 0, time.UTC),
		},
		Room: store.Room{ID: 2, HotelID: 1, Number: "102", Type: store.RoomTypeSingle, Capacity: 1, HotelName: "The Grand Hotel"},
	}

	errStore = errors.New("connection refused")
)

// The JSON a client should receive for testBooking.
const testBookingJSON = `{"reference":"` + testRef + `","guest_name":"Jane Doe","guests":1,` +
	`"check_in":"2027-01-11","check_out":"2027-01-13","created_at":"2026-10-26T15:00:00Z",` +
	`"room":{"id":2,"number":"102","type":"single","capacity":1,"hotel":{"id":1,"name":"The Grand Hotel"}}}`

// assertJSON fails unless rec has the wanted status and a JSON body equal to want. Both are parsed
// before comparing, so field order and whitespace don't matter but every name, value and format does.
func assertJSON(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int, want string) {
	t.Helper()
	if rec.Code != wantStatus {
		t.Fatalf("status = %d, want %d; body %s", rec.Code, wantStatus, rec.Body)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}
	var got, wantV any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("body isn't JSON: %v; body %s", err, rec.Body)
	}
	if err := json.Unmarshal([]byte(want), &wantV); err != nil {
		t.Fatalf("bad want JSON in test: %v", err)
	}
	if !reflect.DeepEqual(got, wantV) {
		t.Fatalf("body\n got %s\nwant %s", strings.TrimSpace(rec.Body.String()), want)
	}
}
