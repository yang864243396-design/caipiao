package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStripTrailingSlash(t *testing.T) {
	t.Parallel()
	var saw string
	h := StripTrailingSlash(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		saw = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/client/schemes/", nil)
	h.ServeHTTP(rec, req)
	if saw != "/api/v1/client/schemes" {
		t.Fatalf("path=%q want /api/v1/client/schemes", saw)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/client/schemes/def-1", nil)
	h.ServeHTTP(rec, req)
	if saw != "/api/v1/client/schemes/def-1" {
		t.Fatalf("path=%q want unchanged", saw)
	}
}
