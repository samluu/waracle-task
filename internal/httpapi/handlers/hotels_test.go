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
		{"found", FindHotelsParams{Name: "e"},
			&mocks.StoreMock{FindHotelsFunc: func(context.Context, string) ([]store.Hotel, error) {
				return []store.Hotel{{ID: 2, Name: "Seaside Inn"}, {ID: 1, Name: "The Grand Hotel"}}, nil
			}},
			FindHotels200JSONResponse{{Id: 2, Name: "Seaside Inn"}, {Id: 1, Name: "The Grand Hotel"}}, nil},
		{"none is an empty list", FindHotelsParams{Name: "Nope"},
			&mocks.StoreMock{FindHotelsFunc: func(context.Context, string) ([]store.Hotel, error) {
				return []store.Hotel{}, nil
			}},
			FindHotels200JSONResponse{}, nil},
		{"blank name", FindHotelsParams{Name: " "}, &mocks.StoreMock{},
			FindHotels400JSONResponse{badRequest("name is required")}, nil},
		{"store failure is returned", FindHotelsParams{Name: "X"},
			&mocks.StoreMock{FindHotelsFunc: func(context.Context, string) ([]store.Hotel, error) {
				return nil, errStore
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
