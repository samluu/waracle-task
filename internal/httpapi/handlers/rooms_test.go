package handlers

import (
	"testing"

	"github.com/samluu/waracle-task/internal/httpapi/handlers/mocks"
	"github.com/samluu/waracle-task/internal/store"
)

// validRoomSearch returns a valid search for 1 guest over 11-13 Jan 2027 in any hotel.
func validRoomSearch() FindRoomsParams {
	return FindRoomsParams{CheckIn: *apiDate("2027-01-11"), CheckOut: *apiDate("2027-01-13"), Guests: 1}
}

func TestFindRooms(t *testing.T) {
	tests := []struct {
		name     string
		params   FindRoomsParams
		newStore func() *mocks.StoreMock
		want     FindRoomsResponseObject
		wantErr  error
	}{
		{"rooms", validRoomSearch(), mocks.FindRoomsReturns([]store.Room{testRoom}, nil), FindRooms200JSONResponse{apiTestRoom}, nil},
		{"none is an empty list", validRoomSearch(), mocks.FindRoomsReturns(nil, nil), FindRooms200JSONResponse{}, nil},
		{"invalid stay", func() FindRoomsParams { // the rules themselves are in validate_test.go
			p := validRoomSearch()
			p.CheckIn, p.CheckOut = p.CheckOut, p.CheckIn
			return p
		}(), mocks.NoStore, FindRooms400JSONResponse{badRequest("check_out must be after check_in")}, nil},
		{"zero guests", func() FindRoomsParams {
			p := validRoomSearch()
			p.Guests = 0
			return p
		}(), mocks.NoStore, FindRooms400JSONResponse{badRequest("guests must be at least 1")}, nil},
		{"store failure is returned", validRoomSearch(), mocks.FindRoomsReturns(nil, errStore), nil, errStore},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newTestServer(tt.newStore()).FindRooms(t.Context(), FindRoomsRequestObject{Params: tt.params})
			assertResponse(t, got, err, tt.want, tt.wantErr)
		})
	}
}
