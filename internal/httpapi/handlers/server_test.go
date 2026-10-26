package handlers

import (
	"context"
	"errors"
	"testing"

	"github.com/samluu/waracle-task/internal/httpapi/handlers/mocks"
)

func TestGetHealth(t *testing.T) {
	tests := []struct {
		name string
		st   *mocks.StoreMock
		want GetHealthResponseObject
	}{
		{"db up", &mocks.StoreMock{PingFunc: func(context.Context) error { return nil }},
			GetHealth200JSONResponse{Status: "ok"}},
		{"db down", &mocks.StoreMock{PingFunc: func(context.Context) error { return errors.New("boom") }},
			GetHealth503JSONResponse{Error: "database unavailable"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newTestServer(tt.st).GetHealth(t.Context(), GetHealthRequestObject{})
			assertResponse(t, got, err, tt.want, nil)
		})
	}
}
