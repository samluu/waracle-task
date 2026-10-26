package handlers

import (
	"context"
	"fmt"
	"testing"

	"github.com/samluu/waracle-task/internal/httpapi/handlers/mocks"
	"github.com/samluu/waracle-task/internal/store"
)

// storeHotels returns n hotels named "Hotel 1".."Hotel n" as the store returns them, and apiHotels the
// same hotels as the handler returns them.
func storeHotels(n int) []store.Hotel {
	hotels := make([]store.Hotel, n)
	for i := range hotels {
		hotels[i] = store.Hotel{ID: int64(i + 1), Name: fmt.Sprint("Hotel ", i+1)}
	}
	return hotels
}

func apiHotels(n int) []Hotel {
	hotels := make([]Hotel, n)
	for i := range hotels {
		hotels[i] = Hotel{Id: int64(i + 1), Name: fmt.Sprint("Hotel ", i+1)}
	}
	return hotels
}

func TestFindHotels(t *testing.T) {
	tests := []struct {
		name    string
		params  FindHotelsParams
		st      *mocks.StoreMock // only the methods this request should reach are set; any other call panics
		want    FindHotelsResponseObject
		wantErr error
	}{
		{"found", FindHotelsParams{Name: "e"},
			&mocks.StoreMock{FindHotelsFunc: func(context.Context, store.HotelFilter) ([]store.Hotel, error) {
				return []store.Hotel{{ID: 2, Name: "Seaside Inn"}, {ID: 1, Name: "The Grand Hotel"}}, nil
			}},
			FindHotels200JSONResponse{Hotels: []Hotel{{Id: 2, Name: "Seaside Inn"}, {Id: 1, Name: "The Grand Hotel"}}}, nil},
		{"none is an empty page", FindHotelsParams{Name: "Nope"},
			&mocks.StoreMock{FindHotelsFunc: func(context.Context, store.HotelFilter) ([]store.Hotel, error) {
				return []store.Hotel{}, nil
			}},
			FindHotels200JSONResponse{Hotels: []Hotel{}}, nil},
		{"store failure is returned", FindHotelsParams{Name: "X"},
			&mocks.StoreMock{FindHotelsFunc: func(context.Context, store.HotelFilter) ([]store.Hotel, error) {
				return nil, errStore
			}},
			nil, errStore},

		// Paging works as for rooms: one hotel more than the page means there's a next page.
		{"exactly a default page: no next page", FindHotelsParams{Name: "hotel"},
			&mocks.StoreMock{FindHotelsFunc: func(context.Context, store.HotelFilter) ([]store.Hotel, error) {
				return storeHotels(20), nil
			}},
			FindHotels200JSONResponse{Hotels: apiHotels(20)}, nil},
		{"more than a default page", FindHotelsParams{Name: "hotel"},
			&mocks.StoreMock{FindHotelsFunc: func(context.Context, store.HotelFilter) ([]store.Hotel, error) {
				return storeHotels(21), nil
			}},
			FindHotels200JSONResponse{Hotels: apiHotels(20), NextCursor: new(encodeCursor(hotelCursor{HotelName: "Hotel 20"}))}, nil},
		{"more than a given limit", FindHotelsParams{Name: "hotel", Limit: new(2)},
			&mocks.StoreMock{FindHotelsFunc: func(context.Context, store.HotelFilter) ([]store.Hotel, error) {
				return storeHotels(3), nil
			}},
			FindHotels200JSONResponse{Hotels: apiHotels(2), NextCursor: new(encodeCursor(hotelCursor{HotelName: "Hotel 2"}))}, nil},
		{"blank name", FindHotelsParams{Name: " "}, &mocks.StoreMock{},
			FindHotels400JSONResponse{badRequest("name is required")}, nil},
		// The rules themselves are in paging_test.go.
		{"invalid limit", FindHotelsParams{Name: "hotel", Limit: new(0)}, &mocks.StoreMock{},
			FindHotels400JSONResponse{badRequest("limit must be between 1 and 100")}, nil},
		{"invalid cursor", FindHotelsParams{Name: "hotel", Cursor: new("not a cursor")}, &mocks.StoreMock{},
			FindHotels400JSONResponse{badRequest("invalid cursor")}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newTestServer(tt.st).FindHotels(t.Context(), FindHotelsRequestObject{Params: tt.params})
			assertResponse(t, got, err, tt.want, tt.wantErr)
		})
	}
}
