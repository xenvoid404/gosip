package gosip

import (
	"errors"
	"io"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestGosip_Routing(t *testing.T) {
	app := New()

	app.Get("/get", func(c *Ctx) error { return c.String("get") })
	app.Post("/post", func(c *Ctx) error { return c.String("post") })
	app.Put("/put", func(c *Ctx) error { return c.String("put") })
	app.Patch("/patch", func(c *Ctx) error { return c.String("patch") })
	app.Delete("/delete", func(c *Ctx) error { return c.String("delete") })
	app.Options("/options", func(c *Ctx) error { return c.String("options") })
	app.Head("/head", func(c *Ctx) error { return c.String("head") })

	methods := []struct {
		method string
		path   string
	}{
		{MethodGet, "/get"},
		{MethodPost, "/post"},
		{MethodPut, "/put"},
		{MethodPatch, "/patch"},
		{MethodDelete, "/delete"},
		{MethodOptions, "/options"},
		{MethodHead, "/head"},
	}

	for _, m := range methods {
		req := httptest.NewRequest(m.method, m.path, nil)
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)

		if w.Code != StatusOK {
			t.Errorf("Method %s Code: want 200, got %d", m.method, w.Code)
		}
		if m.method != MethodHead && w.Body.String() != strings.ToLower(m.method) {
			t.Errorf("Method %s Body: want %s, got %s", m.method, strings.ToLower(m.method), w.Body.String())
		}
	}
}

func TestGosip_Group(t *testing.T) {
	app := New()
	api := app.Group("/api")
	v1 := api.Group("/v1")

	v1.Get("/users", func(c *Ctx) error {
		return c.String("users_v1")
	})

	req := httptest.NewRequest(MethodGet, "/api/v1/users", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != StatusOK {
		t.Errorf("Group Code: want 200, got %d", w.Code)
	}
	if w.Body.String() != "users_v1" {
		t.Errorf("Group Body: want users_v1, got %s", w.Body.String())
	}
}

func TestGosip_Middleware(t *testing.T) {
	app := New()

	order := []string{}

	app.Use(func(c *Ctx) error {
		order = append(order, "1")
		return c.Next()
	})

	api := app.Group("/api")
	api.Use(func(c *Ctx) error {
		order = append(order, "2")
		return c.Next()
	})

	api.Get("/test", func(c *Ctx) error {
		order = append(order, "3")
		return c.String("ok")
	})

	req := httptest.NewRequest(MethodGet, "/api/test", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if strings.Join(order, "") != "123" {
		t.Errorf("Middleware order: want 123, got %s", strings.Join(order, ""))
	}
}

func TestGosip_ErrorHandling(t *testing.T) {
	app := New()

	customErr := errors.New("something went wrong")

	app.Get("/error", func(c *Ctx) error {
		return customErr
	})

	app.Get("/error_after_write", func(c *Ctx) error {
		c.String("already wrote")
		return errors.New("late error")
	})

	// Test default error handler
	req := httptest.NewRequest(MethodGet, "/error", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Code != StatusInternalServerError {
		t.Errorf("Default error handler Code: want 500, got %d", w.Code)
	}

	// Test error after write header
	req2 := httptest.NewRequest(MethodGet, "/error_after_write", nil)
	w2 := httptest.NewRecorder()
	app.ServeHTTP(w2, req2)
	// Status code should remain 200 because it was written by c.String()
	if w2.Code != StatusOK {
		t.Errorf("Late error Code: want 200, got %d", w2.Code)
	}

	// Test custom error handler
	app.SetErrorHandler(func(err error, c *Ctx) {
		c.Status(StatusBadRequest).String("custom err: " + err.Error())
	})

	req3 := httptest.NewRequest(MethodGet, "/error", nil)
	w3 := httptest.NewRecorder()
	app.ServeHTTP(w3, req3)
	if w3.Code != StatusBadRequest {
		t.Errorf("Custom error handler Code: want 400, got %d", w3.Code)
	}
	if w3.Body.String() != "custom err: something went wrong" {
		t.Errorf("Custom error handler Body: got %s", w3.Body.String())
	}
}

func TestGosip_NotFound_MethodNotAllowed(t *testing.T) {
	app := New()
	app.Get("/hello", func(c *Ctx) error { return c.String("hello") })

	// Test 404 Not Found
	req1 := httptest.NewRequest(MethodGet, "/notexist", nil)
	w1 := httptest.NewRecorder()
	app.ServeHTTP(w1, req1)
	if w1.Code != StatusNotFound {
		t.Errorf("Not Found Code: want 404, got %d", w1.Code)
	}

	// Test 405 Method Not Allowed (Route exists, but wrong method)
	req2 := httptest.NewRequest(MethodPost, "/hello", nil)
	w2 := httptest.NewRecorder()
	app.ServeHTTP(w2, req2)
	if w2.Code != StatusMethodNotAllowed {
		t.Errorf("Method Not Allowed Code: want 405, got %d", w2.Code)
	}

	// Test Custom Handlers
	app.SetNotFoundHandler(func(c *Ctx) error {
		return c.Status(404).String("custom 404")
	})
	app.SetMethodNotAllowedHandler(func(c *Ctx) error {
		return c.Status(405).String("custom 405")
	})

	w3 := httptest.NewRecorder()
	app.ServeHTTP(w3, req1)
	if w3.Body.String() != "custom 404" {
		t.Errorf("Custom 404 Body: got %s", w3.Body.String())
	}

	w4 := httptest.NewRecorder()
	app.ServeHTTP(w4, req2)
	if w4.Body.String() != "custom 405" {
		t.Errorf("Custom 405 Body: got %s", w4.Body.String())
	}
}

func TestGosip_TrustProxy(t *testing.T) {
	app := New()
	app.SetTrustProxy(true, "X-Real-IP")

	if !app.cfg.trustProxy {
		t.Error("trustProxy should be true")
	}
	if app.cfg.proxyHeader != "X-Real-IP" {
		t.Errorf("proxyHeader should be X-Real-IP, got %s", app.cfg.proxyHeader)
	}

	// Test default header fallback
	app.SetTrustProxy(true)
	if app.cfg.proxyHeader != "X-Forwarded-For" {
		t.Errorf("default proxyHeader should be X-Forwarded-For, got %s", app.cfg.proxyHeader)
	}
}

func TestGosip_JoinPaths(t *testing.T) {
	tests := []struct {
		prefix string
		path   string
		want   string
	}{
		{"", "/hello", "/hello"},
		{"/api", "", "/api"},
		{"/api", "/hello", "/api/hello"},
		{"/api/", "/hello", "/api/hello"},
		{"/api", "hello", "/api/hello"},
	}

	for _, tt := range tests {
		got := joinPaths(tt.prefix, tt.path)
		if got != tt.want {
			t.Errorf("joinPaths(%q, %q) = %q; want %q", tt.prefix, tt.path, got, tt.want)
		}
	}
}

func TestGosip_SplitAddr_PrettyHost(t *testing.T) {
	tests := []struct {
		addr     string
		wantHost string
		wantPort string
	}{
		{"", "0.0.0.0", "8000"},
		{":8080", "0.0.0.0", "8080"},
		{"127.0.0.1:9090", "127.0.0.1", "9090"},
		{"localhost", "0.0.0.0", "localhost"}, // without colon
	}

	for _, tt := range tests {
		host, port := splitAddr(tt.addr)
		if host != tt.wantHost || port != tt.wantPort {
			t.Errorf("splitAddr(%q) = %q, %q; want %q, %q", tt.addr, host, port, tt.wantHost, tt.wantPort)
		}
	}

	if prettyHost("") != "127.0.0.1" {
		t.Errorf("prettyHost empty: want 127.0.0.1, got %s", prettyHost(""))
	}
	if prettyHost("0.0.0.0") != "127.0.0.1" {
		t.Errorf("prettyHost 0.0.0.0: want 127.0.0.1, got %s", prettyHost("0.0.0.0"))
	}
	if prettyHost("10.0.0.1") != "10.0.0.1" {
		t.Errorf("prettyHost 10.0.0.1: want 10.0.0.1, got %s", prettyHost("10.0.0.1"))
	}
}

func TestGosip_Paint_And_IsTerminal(t *testing.T) {
	s := paint("test", ansiRed, true)
	if !strings.Contains(s, ansiRed) || !strings.Contains(s, ansiReset) {
		t.Error("paint enabled should contain ansi codes")
	}

	s2 := paint("test", ansiRed, false)
	if s2 != "test" {
		t.Error("paint disabled should return raw string")
	}

	// Just call isTerminal to ensure it doesn't panic
	isTerminal()
}

func TestGosip_Listen_And_Shutdown(t *testing.T) {
	app := New()
	app.Get("/", func(c *Ctx) error { return c.String("ok") })

	// Hijack stdout to silence banner during test
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	go func() {
		// Use a random port
		err := app.Listen("127.0.0.1:0")
		if err != nil {
			t.Logf("Listen error: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond) // Give it time to start

	// Attempt graceful shutdown
	err := app.Shutdown()
	if err != nil {
		t.Errorf("Shutdown error: %v", err)
	}

	// Test ShutdownWithTimeout and ShutdownWithContext with nil server (calling after shutdown or before start)
	app2 := New()
	if err := app2.ShutdownWithTimeout(time.Second); err != nil {
		t.Errorf("ShutdownWithTimeout on nil server should not error, got: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout
	io.ReadAll(r) // Consume hijacked output
}
