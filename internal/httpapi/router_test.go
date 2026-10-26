package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeStore struct{ pingErr error }

func (f fakeStore) Ping(context.Context) error { return f.pingErr }

func TestHealth(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		pingErr    error
		wantStatus int
		wantBody   string
	}{
		{"db up", http.MethodGet, nil, http.StatusOK, `{"status":"ok"}`},
		{"db down", http.MethodGet, errors.New("boom"), http.StatusServiceUnavailable, `{"error":"database unavailable"}`},
		{"wrong method", http.MethodPost, nil, http.StatusMethodNotAllowed, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			NewRouter(fakeStore{tt.pingErr}).ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), tt.method, "/health", nil))

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantBody != "" && strings.TrimSpace(rec.Body.String()) != tt.wantBody {
				t.Fatalf("body = %q, want %q", rec.Body.String(), tt.wantBody)
			}
		})
	}
}
