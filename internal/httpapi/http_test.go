package httpapi

import (
	"net/http"
	"strings"
	"testing"

	"github.com/samluu/waracle-task/internal/store"
)

// These go through the full router with raw requests and JSON responses. They cover only what the
// handler tests can't: the wire format clients depend on (a renamed field passes struct-based tests,
// because both sides use the same generated struct), and code that only runs at the HTTP layer:
// the body-size middleware and the param, request and response error handlers.
func TestHTTP(t *testing.T) {
	tests := []struct {
		name       string
		newStore   func() *StoreMock
		method     string
		target     string
		body       string
		wantStatus int
		want       string
	}{
		// Wire format. The booking response nests Room and Hotel, so this pins all three schemas.
		{"create booking", createBookingReturns(testBooking, nil), http.MethodPost, "/bookings",
			`{"room_id":2,"guest_name":"Jane Doe","guests":1,"check_in":"2027-01-11","check_out":"2027-01-13"}`,
			http.StatusCreated, testBookingJSON},
		{"get booking", getBookingReturns(testBooking, nil), http.MethodGet, "/bookings/" + testRef, "",
			http.StatusOK, testBookingJSON},
		{"health", pingReturns(nil), http.MethodGet, "/health", "",
			http.StatusOK, `{"status":"ok"}`},

		// HTTP-layer error handling. None of the first four reach the store.
		{"unparseable parameter", noStore, http.MethodGet, "/hotels", "", // paramError, in the generated code's wording
			http.StatusBadRequest, `{"error":"Query argument name is required, but not found"}`},
		{"malformed booking reference", noStore, http.MethodGet, "/bookings/BK-ABCDEFGH", "", // paramError: the spec says format uuid
			http.StatusBadRequest, `{"error":"Invalid format for parameter reference: error unmarshaling 'BK-ABCDEFGH' text as *uuid.UUID: invalid UUID length: 11"}`},
		{"undecodable body", noStore, http.MethodPost, "/bookings", `{"room_id":`, // requestError
			http.StatusBadRequest, `{"error":"invalid request body: unexpected EOF"}`},
		{"body too large", noStore, http.MethodPost, "/bookings", // limitBody middleware + requestError
			`{"guest_name":"` + strings.Repeat("A", maxBodyBytes) + `"}`,
			http.StatusRequestEntityTooLarge, `{"error":"request body must be at most 65536 bytes"}`},
		{"unexpected error is hidden", getBookingReturns(store.BookingDetails{}, errStore), http.MethodGet, "/bookings/" + testRef, "", // responseError
			http.StatusInternalServerError, `{"error":"internal error"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := serve(t, tt.newStore(), tt.method, tt.target, tt.body)
			assertJSON(t, rec, tt.wantStatus, tt.want)
			if tt.wantStatus == http.StatusCreated {
				if loc := rec.Header().Get("Location"); loc != "/bookings/"+testRef {
					t.Fatalf("Location = %q", loc)
				}
			}
		})
	}
}
