package handlers

import (
	"testing"

	"github.com/samluu/waracle-task/internal/httpapi/handlers/mocks"
	"github.com/samluu/waracle-task/internal/store"
)

func TestFindHotels(t *testing.T) {
	tests := []struct {
		name     string
		params   FindHotelsParams
		newStore func() *mocks.StoreMock
		want     FindHotelsResponseObject
		wantErr  error
	}{
		{"found", FindHotelsParams{Name: "The Grand Hotel"}, mocks.FindHotelReturns(store.Hotel{ID: 1, Name: "The Grand Hotel"}, nil),
			FindHotels200JSONResponse{{Id: 1, Name: "The Grand Hotel"}}, nil},
		{"not found is an empty list", FindHotelsParams{Name: "Nope"}, mocks.FindHotelReturns(store.Hotel{}, store.ErrNotFound),
			FindHotels200JSONResponse{}, nil},
		{"blank name", FindHotelsParams{Name: " "}, mocks.NoStore,
			FindHotels400JSONResponse{badRequest("name is required")}, nil},
		{"store failure is returned", FindHotelsParams{Name: "X"}, mocks.FindHotelReturns(store.Hotel{}, errStore),
			nil, errStore},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newTestServer(tt.newStore()).FindHotels(t.Context(), FindHotelsRequestObject{Params: tt.params})
			assertResponse(t, got, err, tt.want, tt.wantErr)
		})
	}
}
