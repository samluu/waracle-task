package httpapi

import (
	"errors"
	"testing"
)

func TestGetHealth(t *testing.T) {
	tests := []struct {
		name     string
		newStore func() *StoreMock
		want     GetHealthResponseObject
	}{
		{"db up", pingReturns(nil), GetHealth200JSONResponse{Status: "ok"}},
		{"db down", pingReturns(errors.New("boom")), GetHealth503JSONResponse{Error: "database unavailable"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newTestServer(tt.newStore()).GetHealth(t.Context(), GetHealthRequestObject{})
			assertResponse(t, got, err, tt.want, nil)
		})
	}
}
