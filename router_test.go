package gosip

import (
	"bytes"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
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

func TestRouterSkipNext(t *testing.T) {
	r := New()
	handlerCalled := false
	r.Use(func(c *Context) { c.Status(StatusForbidden).String("blocked") }) // ekspektasi ngga memanggil Next()
	r.Get("/carmen", func(c *Context) { handlerCalled = true; c.String("ok") })

	rec := doRequest(r, http.MethodGet, "/carmen")

	if handlerCalled {
		t.Fatal("handler seharusnya tidak dipanggil karena middleware tidak memanggil Next()")
	}
	if rec.Code != StatusForbidden || rec.Body.String() != "blocked" {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestRouterGroupInherit(t *testing.T) {
	r := New()

	var order []string
	r.Use(func(c *Context) { order = append(order, "root-mw"); c.Next() })

	api := r.Group("/api")
	api.Use(func(c *Context) { order = append(order, "api-mw"); c.Next() })
	api.Get("/ping", func(c *Context) { order = append(order, "handler"); c.String("pong") })

	rec := doRequest(r, http.MethodGet, "/api/ping")
	if rec.Body.String() != "pong" {
		t.Fatalf("body = %q, harusnya %q", rec.Body.String(), "pong")
	}

	want := []string{"root-mw", "api-mw", "handler"}
	if len(order) != len(want) {
		t.Fatalf("order = %v, seharusnya %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order = %v, harusnya %v", order, want)
		}
	}
}

// Tes middleware yang didaftarkan di router utama setelah sebuah grup dibuat
// tidak boleh terpakai oleh routing grup yang sudah ada
func TestRouterGroupNotInherit(t *testing.T) {
	r := New()
	g1 := r.Group("/g1")
	g1.Get("/carmen", func(c *Context) { c.String("g1 handler") })

	// Middleware ini didaftarkan setelah g1 dibuat.
	r.Use(func(c *Context) { c.Status(StatusTeapot).String("from root mw") })

	rec := doRequest(r, http.MethodGet, "/g1/carmen")
	if rec.Code != StatusOK || rec.Body.String() != "g1 handler" {
		t.Fatalf("status=%d body=%q, harusnya status=200 body=%q (middleware root yang didaftarkan belakangan tidak boleh memengaruhi g1)",
			rec.Code, rec.Body.String(), "g1 handler")
	}
}

// Middleware yang didaftarkan ke dalam sebuah grup, tidak
// boleh bocor ke group lain ataupun ke router utama
func TestRouterGroupNotLeak(t *testing.T) {
	r := New()
	g1 := r.Group("/g1")
	g2 := r.Group("/g2")

	g1.Use(func(c *Context) { c.Status(StatusTeapot).String("from g1 mw") })
	g1.Get("/carmen", func(c *Context) { c.String("g1 handler") })
	g2.Get("/carmen", func(c *Context) { c.String("g2 handler") })

	recG2 := doRequest(r, http.MethodGet, "/g2/carmen")

	if recG2.Code != StatusOK || recG2.Body.String() != "g2 handler" {
		t.Fatalf("g2 status=%d body=%q, harusnya status=200 body=%q (middleware g1 tidak boleh memengaruhi g2)",
			recG2.Code, recG2.Body.String(), "g2 handler")
	}
}

func TestRouterNotFound(t *testing.T) {
	r := New()
	r.notFound = func(c *Context) { c.Status(StatusTeapot).String("kustom 404") }

	rec := doRequest(r, http.MethodGet, "/ghost")

	if rec.Code != StatusTeapot || rec.Body.String() != "kustom 404" {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestRouterErrorHandler(t *testing.T) {
	r := New()
	r.Get("/boom", func(c *Context) { c.Error(errors.New("boom")) })

	rec := doRequest(r, http.MethodGet, "/boom")

	if rec.Code != StatusInternalServerError {
		t.Fatalf("status = %d, harusnya %d", rec.Code, StatusInternalServerError)
	}
	if rec.Body.String() != "boom" {
		t.Fatalf("body = %q, harusnya %q", rec.Body.String(), "boom")
	}
}

func TestRouterErrorAfterHeaderSent(t *testing.T) {
	var logBuf bytes.Buffer
	log.SetOutput(&logBuf)
	defer log.SetOutput(os.Stderr)

	r := New()
	r.Get("/late", func(c *Context) {
		c.String("ok") // header + body sudah terkirim di sini
		c.Error(errors.New("terlambat"))
	})

	rec := doRequest(r, http.MethodGet, "/late")

	if rec.Code != StatusOK || rec.Body.String() != "ok" {
		t.Fatalf("response ke klien seharusnya tidak berubah, status=%d body=%q", rec.Code, rec.Body.String())
	}
	if !strings.Contains(logBuf.String(), "terlambat") {
		t.Fatalf("log = %q, harusnya memuat error yang muncul belakangan", logBuf.String())
	}
}

func TestJoinPaths(t *testing.T) {
	cases := []struct {
		prefix, path, want string
	}{
		{"", "/carmen", "/carmen"},
		{"/api", "", "/api"},
		{"/api", "/carmen", "/api/carmen"},
		{"/api/", "/carmen", "/api/carmen"},
		{"/api", "carmen", "/api/carmen"},
		{"/api/", "/carmen/", "/api/carmen/"},
	}
	for _, tc := range cases {
		if got := joinPaths(tc.prefix, tc.path); got != tc.want {
			t.Errorf("joinPaths(%q, %q) = %q, harusnya %q", tc.prefix, tc.path, got, tc.want)
		}
	}
}
