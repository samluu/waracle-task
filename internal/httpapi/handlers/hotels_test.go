package handlers

import (
	"context"
	"testing"

	"github.com/samluu/waracle-task/internal/httpapi/handlers/mocks"
	"github.com/samluu/waracle-task/internal/store"
)

func TestFindHotels(t *testing.T) {
	tests := []struct {
		name    string
		params  FindHotelsParams
		st      *mocks.StoreMock // only the methods this request should reach are set; any other call panics
		want    FindHotelsResponseObject
		wantErr error
	}{
		{"found", FindHotelsParams{Name: "The Grand Hotel"},
			&mocks.StoreMock{FindHotelByNameFunc: func(context.Context, string) (store.Hotel, error) {
				return store.Hotel{ID: 1, Name: "The Grand Hotel"}, nil
			}},
			FindHotels200JSONResponse{{Id: 1, Name: "The Grand Hotel"}}, nil},
		{"not found is an empty list", FindHotelsParams{Name: "Nope"},
			&mocks.StoreMock{FindHotelByNameFunc: func(context.Context, string) (store.Hotel, error) {
				return store.Hotel{}, store.ErrNotFound
			}},
			FindHotels200JSONResponse{}, nil},
		{"blank name", FindHotelsParams{Name: " "}, &mocks.StoreMock{},
			FindHotels400JSONResponse{badRequest("name is required")}, nil},
		{"store failure is returned", FindHotelsParams{Name: "X"},
			&mocks.StoreMock{FindHotelByNameFunc: func(context.Context, string) (store.Hotel, error) {
				return store.Hotel{}, errStore
			}},
			nil, errStore},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newTestServer(tt.st).FindHotels(t.Context(), FindHotelsRequestObject{Params: tt.params})
			assertResponse(t, got, err, tt.want, tt.wantErr)
		})
	}
}
