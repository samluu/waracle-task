package handlers

import (
	"context"
	"testing"

	"github.com/samluu/waracle-task/internal/httpapi/handlers/mocks"
	"github.com/samluu/waracle-task/internal/store"
)

func TestSeedData(t *testing.T) {
	tests := []struct {
		name    string
		st      *mocks.StoreMock
		want    SeedDataResponseObject
		wantErr error
	}{
		{"seeded", &mocks.StoreMock{SeedFunc: func(context.Context) error { return nil }},
			SeedData204Response{}, nil},
		{"already seeded", &mocks.StoreMock{SeedFunc: func(context.Context) error { return store.ErrAlreadySeeded }},
			SeedData409JSONResponse{Error: "database already seeded; reset it first"}, nil},
		{"store failure is returned", &mocks.StoreMock{SeedFunc: func(context.Context) error { return errStore }},
			nil, errStore},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newTestServer(tt.st).SeedData(t.Context(), SeedDataRequestObject{})
			assertResponse(t, got, err, tt.want, tt.wantErr)
		})
	}
}

func TestResetData(t *testing.T) {
	tests := []struct {
		name    string
		st      *mocks.StoreMock
		want    ResetDataResponseObject
		wantErr error
	}{
		{"reset", &mocks.StoreMock{ResetFunc: func(context.Context) error { return nil }},
			ResetData204Response{}, nil},
		{"store failure is returned", &mocks.StoreMock{ResetFunc: func(context.Context) error { return errStore }},
			nil, errStore},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newTestServer(tt.st).ResetData(t.Context(), ResetDataRequestObject{})
			assertResponse(t, got, err, tt.want, tt.wantErr)
		})
	}
}
