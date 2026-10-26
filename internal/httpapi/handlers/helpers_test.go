package handlers

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/samluu/waracle-task/internal/store"
)

// The store is mocked with mocks.StoreMock; see the mocks package.

// testToday is the fixed "today" for handler tests.
var testToday = time.Date(2026, 10, 26, 15, 0, 0, 0, time.UTC)

// newTestServer returns the handlers backed by st, with "today" fixed to testToday. Handler tests call
// its methods directly with a request struct and compare the response struct.
func newTestServer(st Store) *Server {
	return NewServer(st, func() time.Time { return testToday })
}

// assertResponse fails unless a handler returned wantErr (matched with errors.Is), or no error and a
// response equal to want.
func assertResponse(t *testing.T, got any, gotErr error, want any, wantErr error) {
	t.Helper()
	if wantErr != nil {
		if !errors.Is(gotErr, wantErr) {
			t.Fatalf("want error %v, got %v (response %+v)", wantErr, gotErr, got)
		}
		return
	}
	if gotErr != nil {
		t.Fatalf("unexpected error: %v", gotErr)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("response\n got %#v\nwant %#v", got, want)
	}
}

func apiDate(s string) *openapi_types.Date {
	return &openapi_types.Date{Time: date(s)}
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

var testReference = uuid.MustParse(testRef)

var (
	testRoom    = store.Room{ID: 2, HotelID: 1, Number: "102", Type: store.RoomTypeSingle, Capacity: 1, HotelName: "The Grand Hotel"}
	testBooking = store.BookingDetails{
		Booking: store.Booking{
			Reference: testReference, RoomID: 2, GuestName: "Jane Doe", Guests: 1,
			CheckIn: date("2027-01-11"), CheckOut: date("2027-01-13"),
			CreatedAt: time.Date(2026, 10, 26, 15, 0, 0, 0, time.UTC),
		},
		Room: testRoom,
	}

	// The API values the handlers should return for testRoom and testBooking.
	apiTestRoom = Room{
		Id: 2, Number: "102", Type: Single, Capacity: 1,
		Hotel: Hotel{Id: 1, Name: "The Grand Hotel"},
	}
	apiTestBooking = Booking{
		Reference: testReference, GuestName: "Jane Doe", Guests: 1,
		CheckIn: *apiDate("2027-01-11"), CheckOut: *apiDate("2027-01-13"),
		CreatedAt: time.Date(2026, 10, 26, 15, 0, 0, 0, time.UTC),
		Room:      apiTestRoom,
	}

	errStore = errors.New("connection refused")
)

// newValidBooking returns a new valid booking request. Tests that need an invalid one change a field of it.
func newValidBooking() BookingRequest {
	return BookingRequest{
		RoomId: 2, GuestName: "Jane Doe", Guests: 1,
		CheckIn: *apiDate("2027-01-11"), CheckOut: *apiDate("2027-01-13"),
	}
}
