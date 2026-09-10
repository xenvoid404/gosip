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
}

func TestRouterChaining(t *testing.T) {
	r := New()

	var order []string
	r.Use(func(c *Context) { order = append(order, "mw1"); c.Next() })
	r.Use(func(c *Context) { order = append(order, "mw2"); c.Next() })
	r.Get("/carmen", func(c *Context) { order = append(order, "handler"); c.String("ok") })

	rec := doRequest(r, http.MethodGet, "/carmen")
	if rec.Body.String() != "ok" {
		t.Fatalf("body = %q, harusnya %q", rec.Body.String(), "ok")
	}

	want := []string{"mw1", "mw2", "handler"}
	if len(order) != len(want) {
		t.Fatalf("order = %v, harusnya %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order = %v, harusnya %v", order, want)
		}
	}
}
