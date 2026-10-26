package handlers

import (
	"context"
	"fmt"
	"testing"

	"github.com/samluu/waracle-task/internal/httpapi/handlers/mocks"
	"github.com/samluu/waracle-task/internal/store"
)

// validRoomSearch returns a valid search for 1 guest over 11-13 Jan 2027 in any hotel.
func validRoomSearch() FindRoomsParams {
	return FindRoomsParams{CheckIn: *apiDate("2027-01-11"), CheckOut: *apiDate("2027-01-13"), Guests: 1}
}

// storeRooms returns n rooms at The Grand Hotel numbered 1..n, as the store returns them, and apiRooms
// the first n of the same rooms as the handler returns them.
func storeRooms(n int) []store.Room {
	rooms := make([]store.Room, n)
	for i := range rooms {
		rooms[i] = store.Room{ID: int64(i + 1), HotelID: 1, Number: fmt.Sprint(i + 1), Type: store.RoomTypeSingle, Capacity: 1, HotelName: "The Grand Hotel"}
	}
	return rooms
}

func apiRooms(n int) []Room {
	rooms := make([]Room, n)
	for i := range rooms {
		rooms[i] = Room{Id: int64(i + 1), Number: fmt.Sprint(i + 1), Type: Single, Capacity: 1, Hotel: Hotel{Id: 1, Name: "The Grand Hotel"}}
	}
	return rooms
}

func TestFindRooms(t *testing.T) {
	tests := []struct {
		name    string
		params  FindRoomsParams
		st      *mocks.StoreMock // only the methods this request should reach are set; any other call panics
		want    FindRoomsResponseObject
		wantErr error
	}{
		{"rooms", validRoomSearch(),
			&mocks.StoreMock{FindRoomsFunc: func(context.Context, store.RoomFilter) ([]store.Room, error) {
				return []store.Room{testRoom}, nil
			}},
			FindRooms200JSONResponse{Rooms: []Room{apiTestRoom}}, nil},
		{"none is an empty page", validRoomSearch(),
			&mocks.StoreMock{FindRoomsFunc: func(context.Context, store.RoomFilter) ([]store.Room, error) {
				return nil, nil
			}},
			FindRooms200JSONResponse{Rooms: []Room{}}, nil},

		// Paging: the handler asks the store for one room more than the page; getting it back means
		// there's a next page, whose cursor points after the page's last room.
		{"exactly a default page: no next page", validRoomSearch(),
			&mocks.StoreMock{FindRoomsFunc: func(context.Context, store.RoomFilter) ([]store.Room, error) {
				return storeRooms(20), nil
			}},
			FindRooms200JSONResponse{Rooms: apiRooms(20)}, nil},
		{"more than a default page", validRoomSearch(),
			&mocks.StoreMock{FindRoomsFunc: func(context.Context, store.RoomFilter) ([]store.Room, error) {
				return storeRooms(21), nil
			}},
			FindRooms200JSONResponse{Rooms: apiRooms(20),
				NextCursor: new(encodeCursor(roomCursor{HotelName: "The Grand Hotel", Number: "20"}))}, nil},
		{"more than a given limit", func() FindRoomsParams {
			p := validRoomSearch()
			p.Limit = new(2)
			return p
		}(),
			&mocks.StoreMock{FindRoomsFunc: func(context.Context, store.RoomFilter) ([]store.Room, error) {
				return storeRooms(3), nil
			}},
			FindRooms200JSONResponse{Rooms: apiRooms(2),
				NextCursor: new(encodeCursor(roomCursor{HotelName: "The Grand Hotel", Number: "2"}))}, nil},
		{"limit over the maximum", func() FindRoomsParams {
			p := validRoomSearch()
			p.Limit = new(101)
			return p
		}(), &mocks.StoreMock{}, FindRooms400JSONResponse{badRequest("limit must be between 1 and 100")}, nil},
		{"zero limit", func() FindRoomsParams {
			p := validRoomSearch()
			p.Limit = new(0)
			return p
		}(), &mocks.StoreMock{}, FindRooms400JSONResponse{badRequest("limit must be between 1 and 100")}, nil},
		{"blank hotel name", func() FindRoomsParams {
			p := validRoomSearch()
			p.HotelName = new("  ")
			return p
		}(), &mocks.StoreMock{}, FindRooms400JSONResponse{badRequest("hotel_name must not be blank; omit it to search every hotel")}, nil},
		{"invalid cursor", func() FindRoomsParams { // decoding itself is in cursor_test.go
			p := validRoomSearch()
			p.Cursor = new("not a cursor")
			return p
		}(), &mocks.StoreMock{}, FindRooms400JSONResponse{badRequest("invalid cursor")}, nil},

		{"invalid stay", func() FindRoomsParams { // the rules themselves are in validate_test.go
			p := validRoomSearch()
			p.CheckIn, p.CheckOut = p.CheckOut, p.CheckIn
			return p
		}(), &mocks.StoreMock{}, FindRooms400JSONResponse{badRequest("check_out must be after check_in")}, nil},
		{"zero guests", func() FindRoomsParams {
			p := validRoomSearch()
			p.Guests = 0
			return p
		}(), &mocks.StoreMock{}, FindRooms400JSONResponse{badRequest("guests must be at least 1")}, nil},
		{"store failure is returned", validRoomSearch(),
			&mocks.StoreMock{FindRoomsFunc: func(context.Context, store.RoomFilter) ([]store.Room, error) {
				return nil, errStore
			}},
			nil, errStore},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newTestServer(tt.st).FindRooms(t.Context(), FindRoomsRequestObject{Params: tt.params})
			assertResponse(t, got, err, tt.want, tt.wantErr)
		})
	}
}
