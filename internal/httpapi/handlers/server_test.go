package handlers

import (
	"errors"
	"testing"

	"github.com/samluu/waracle-task/internal/httpapi/handlers/mocks"
)

func TestGetHealth(t *testing.T) {
	tests := []struct {
		name     string
		newStore func() *mocks.StoreMock
		want     GetHealthResponseObject
	}{
		{"db up", mocks.PingReturns(nil), GetHealth200JSONResponse{Status: "ok"}},
		{"db down", mocks.PingReturns(errors.New("boom")), GetHealth503JSONResponse{Error: "database unavailable"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newTestServer(tt.newStore()).GetHealth(t.Context(), GetHealthRequestObject{})
			assertResponse(t, got, err, tt.want, nil)
		})
	}
}
