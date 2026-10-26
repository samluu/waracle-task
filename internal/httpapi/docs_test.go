package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/samluu/waracle-task/api"
)

func TestDocs(t *testing.T) {
	tests := []struct {
		path            string
		wantContentType string
		wantBody        string
	}{
		{"/openapi.yaml", "application/yaml", string(api.Spec)},
		{"/docs", "text/html; charset=utf-8", `url: "/openapi.yaml"`},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			NewRouter(fakeStore{}).ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, tt.path, nil))

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

func TestSpecEmbedded(t *testing.T) {
	if !bytes.HasPrefix(api.Spec, []byte("openapi:")) {
		t.Fatal("api.Spec is not the OpenAPI document")
	}
}
