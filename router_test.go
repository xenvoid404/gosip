package gosip

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func doRequest(r *Router, method, target string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestRouterBasic(t *testing.T) {
	r := New()
	r.Get("/mbg", func(c *Context) { c.String("mantap") })

	rec := doRequest(r, http.MethodGet, "/mbg")

	if rec.Code != StatusOK || rec.Body.String() != "mantap" {
		t.Fatalf("status = %d, body = %q", rec.Code, rec.Body.String())
	}
}
