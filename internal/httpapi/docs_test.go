package httpapi

import (
	"net/http"
	"strings"
	"testing"
)

func TestDocs(t *testing.T) {
	tests := []struct {
		path            string
		wantContentType string
		wantBody        string
	}{
		// A line from api/openapi.yaml, rather than api.Spec itself, so an empty embed can't pass.
		{"/openapi.yaml", "application/yaml", "title: Hotel Booking API"},
		{"/docs", "text/html; charset=utf-8", `url: "/openapi.yaml"`},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			rec := serve(t, noStore(), http.MethodGet, tt.path, "")
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}
			if got := rec.Header().Get("Content-Type"); got != tt.wantContentType {
				t.Fatalf("Content-Type = %q, want %q", got, tt.wantContentType)
			}
			if !strings.Contains(rec.Body.String(), tt.wantBody) {
				t.Fatalf("body does not contain %q", tt.wantBody)
			}
		})
	}
}
