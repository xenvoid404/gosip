package gosip

import (
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestRouterUnregisterPath(t *testing.T) {
	r := New()
	r.Get("/mbg", func(c *Context) { c.String("mantap") })

	rec := doRequest(r, http.MethodGet, "/carmen")

	if rec.Code != StatusNotFound {
		t.Fatalf("status = %d, harusnya %d", rec.Code, StatusNotFound)
	}
}

func TestRouterWrongMethod(t *testing.T) {
	r := New()
	r.Get("/mbg", func(c *Context) { c.String("mantap") })

	rec := doRequest(r, http.MethodPost, "/mbg")

	if rec.Code != StatusMethodNotAllowed {
		t.Fatalf("status = %d, harusnya %d", rec.Code, StatusMethodNotAllowed)
	}
	if allow := rec.Header().Get("Allow"); !strings.Contains(allow, http.MethodGet) {
		t.Fatalf("header = %q, harusnya %q", allow, http.MethodGet)
	}
}
