package handlers

import (
	"testing"
	"time"

	"github.com/samluu/waracle-task/internal/store"
)

func TestValidateStay(t *testing.T) {
	today := date("2026-10-26")
	tests := []struct {
		name              string
		checkIn, checkOut time.Time
		wantErr           string // "" means valid
	}{
		{"valid", date("2027-01-11"), date("2027-01-13"), ""},
		{"one night", date("2027-01-11"), date("2027-01-12"), ""},
		{"check-in today", today, date("2026-10-27"), ""},
		{"check-in yesterday", date("2026-10-25"), date("2026-10-27"), "check_in must not be in the past"},
		{"check-in exactly a year ahead", date("2027-10-26"), date("2027-10-27"), ""},
		{"check-in a year and a day ahead", date("2027-10-27"), date("2027-10-28"), "check_in must be no more than a year from today"},
		{"30 nights", date("2027-01-01"), date("2027-01-31"), ""},
		{"31 nights", date("2027-01-01"), date("2027-02-01"), "a stay can be at most 30 nights"},
		{"30 nights across a year end", date("2026-12-15"), date("2027-01-14"), ""},
		{"check-out same day", date("2027-01-11"), date("2027-01-11"), "check_out must be after check_in"},
		{"check-out before check-in", date("2027-01-11"), date("2027-01-10"), "check_out must be after check_in"},
		{"missing check-in", time.Time{}, date("2027-01-13"), "check_in is required"},
		{"missing check-out", date("2027-01-11"), time.Time{}, "check_out is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertErr(t, validateStay(tt.checkIn, tt.checkOut, today), tt.wantErr)
		})
	}
}

func TestValidateBookingRequest(t *testing.T) {
	tests := []struct {
		name    string
		request BookingRequest
		wantErr string
	}{
		{"valid", newValidBooking(), ""},
		{"missing room_id", func() BookingRequest {
			b := newValidBooking()
			b.RoomId = 0
			return b
		}(), "room_id is required"},
		{"negative room_id", func() BookingRequest {
			b := newValidBooking()
			b.RoomId = -1
			return b
		}(), "room_id is required"},
		{"missing guest_name", func() BookingRequest {
			b := newValidBooking()
			b.GuestName = ""
			return b
		}(), "guest_name is required"},
		{"blank guest_name", func() BookingRequest {
			b := newValidBooking()
			b.GuestName = " \t "
			return b
		}(), "guest_name is required"},
		{"zero guests", func() BookingRequest {
			b := newValidBooking()
			b.Guests = 0
			return b
		}(), "guests must be at least 1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertErr(t, validateBookingRequest(tt.request), tt.wantErr)
		})
	}
}

func TestToNewBooking(t *testing.T) {
	today := date("2026-10-26")
	tests := []struct {
		name    string
		request BookingRequest
		want    store.NewBooking
		wantErr string
	}{
		{"valid", newValidBooking(),
			store.NewBooking{RoomID: 2, GuestName: "Jane Doe", Guests: 1, CheckIn: date("2027-01-11"), CheckOut: date("2027-01-13")}, ""},
		{"guest name trimmed", func() BookingRequest {
			b := newValidBooking()
			b.GuestName = "  Jane Doe \t"
			return b
		}(), store.NewBooking{RoomID: 2, GuestName: "Jane Doe", Guests: 1, CheckIn: date("2027-01-11"), CheckOut: date("2027-01-13")}, ""},
		// One failure from each validator, to show both run; their rules are tested above.
		{"invalid request", func() BookingRequest {
			b := newValidBooking()
			b.Guests = 0
			return b
		}(), store.NewBooking{}, "guests must be at least 1"},
		{"invalid stay", func() BookingRequest {
			b := newValidBooking()
			b.CheckIn, b.CheckOut = b.CheckOut, b.CheckIn
			return b
		}(), store.NewBooking{}, "check_out must be after check_in"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toNewBooking(tt.request, today)
			assertErr(t, err, tt.wantErr)
			if got != tt.want {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

// "Today" is the UTC date, whatever the server's local time zone.
func TestToday(t *testing.T) {
	nyEvening := time.Date(2026, 10, 26, 22, 0, 0, 0, time.FixedZone("EDT", -4*60*60)) // already the 27th in UTC
	s := &Server{now: func() time.Time { return nyEvening }}
	if got, want := s.today(), date("2026-10-27"); !got.Equal(want) {
		t.Fatalf("today = %v, want %v", got, want)
	}
}

func assertErr(t *testing.T, err error, want string) {
	t.Helper()
	switch {
	case want == "" && err != nil:
		t.Fatalf("want valid, got %v", err)
	case want != "" && (err == nil || err.Error() != want):
		t.Fatalf("want error %q, got %v", want, err)
	}
}
