package gosip

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCtx_ContextMethodPathRemoteAddr(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/hello", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	
	// Create context with custom value to test Context()
	type key string
	req = req.WithContext(context.WithValue(req.Context(), key("foo"), "bar"))

	c := newCtx(req)

	if c.Method() != http.MethodPost {
		t.Errorf("Method: want POST, got %s", c.Method())
	}
	if c.Path() != "/hello" {
		t.Errorf("Path: want /hello, got %s", c.Path())
	}
	if c.RemoteAddr() != "192.168.1.1:1234" {
		t.Errorf("RemoteAddr: want 192.168.1.1:1234, got %s", c.RemoteAddr())
	}
	
	ctxVal := c.Context().Value(key("foo"))
	if ctxVal != "bar" {
		t.Errorf("Context: want bar, got %v", ctxVal)
	}
}

func TestCtx_StatusAndWroteHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c := newCtx(req)

	if c.StatusCode() != StatusOK {
		t.Errorf("StatusCode default: want 200, got %d", c.StatusCode())
	}
	if c.WroteHeader() {
		t.Error("WroteHeader default: want false, got true")
	}

	c.Status(StatusCreated)
	if c.StatusCode() != StatusCreated {
		t.Errorf("StatusCode after Set: want 201, got %d", c.StatusCode())
	}
	
	c.writeHeader() // Write it once
	if !c.WroteHeader() {
		t.Error("WroteHeader after writeHeader: want true, got false")
	}
	
	c.writeHeader() // Second time should be ignored and not panic/cause error
}

func TestCtx_JSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	c := &Ctx{ResponseWriter: w, Request: req}

	err := c.JSON(Map{"msg": "ok"})
	if err != nil {
		t.Errorf("JSON expected no error, got %v", err)
	}
	
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("JSON Content-Type: want application/json, got %s", w.Header().Get("Content-Type"))
	}
	if w.Body.String() != `{"msg":"ok"}` {
		t.Errorf("JSON Body: want {\"msg\":\"ok\"}, got %s", w.Body.String())
	}

	// Test error marshaling (using unsupported type like channel)
	ch := make(chan int)
	err = c.JSON(ch)
	if err == nil {
		t.Error("JSON with channel should error")
	}
}

func TestCtx_String(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	c := &Ctx{ResponseWriter: w, Request: req}

	err := c.String("hello world")
	if err != nil {
		t.Errorf("String expected no error, got %v", err)
	}
	
	if w.Header().Get("Content-Type") != "text/plain" {
		t.Errorf("String Content-Type: want text/plain, got %s", w.Header().Get("Content-Type"))
	}
	if w.Body.String() != "hello world" {
		t.Errorf("String Body: want hello world, got %s", w.Body.String())
	}
}

func TestCtx_QueryAndParams(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?page=5", nil)
	
	// Mock PathValue (requires Go 1.22 mux routing simulation)
	// We'll just test that it calls Request.PathValue, which is part of the request object.
	req.SetPathValue("id", "42")
	
	c := newCtx(req)

	if c.Query("page") != "5" {
		t.Errorf("Query: want 5, got %s", c.Query("page"))
	}
	if c.Query("not_exist") != "" {
		t.Errorf("Query not_exist: want empty, got %s", c.Query("not_exist"))
	}
	
	if c.Params("id") != "42" {
		t.Errorf("Params: want 42, got %s", c.Params("id"))
	}
}

func TestCtx_HeadersAndCookies(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Custom", "req-val")
	req.AddCookie(&http.Cookie{Name: "session", Value: "123"})
	
	w := httptest.NewRecorder()
	c := &Ctx{ResponseWriter: w, Request: req}

	if c.GetHeader("X-Custom") != "req-val" {
		t.Errorf("GetHeader: want req-val, got %s", c.GetHeader("X-Custom"))
	}
	
	c.SetHeader("X-Res", "res-val")
	if w.Header().Get("X-Res") != "res-val" {
		t.Errorf("SetHeader: want res-val, got %s", w.Header().Get("X-Res"))
	}

	cookie, err := c.GetCookie("session")
	if err != nil || cookie.Value != "123" {
		t.Errorf("GetCookie: want 123, got %v (err: %v)", cookie, err)
	}

	_, err = c.GetCookie("not_exist")
	if err == nil {
		t.Error("GetCookie not_exist should error")
	}

	c.SetCookie(&http.Cookie{Name: "new_session", Value: "456"})
	resCookie := w.Header().Get("Set-Cookie")
	if resCookie != "new_session=456" {
		t.Errorf("SetCookie: want new_session=456, got %s", resCookie)
	}
}

func TestCtx_Locals(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c := newCtx(req)

	if c.Locals("user") != nil {
		t.Errorf("Locals default: want nil, got %v", c.Locals("user"))
	}

	c.Locals("user", "dika")
	if c.Locals("user") != "dika" {
		t.Errorf("Locals after set: want dika, got %v", c.Locals("user"))
	}
	
	// Test setting another value
	c.Locals("role", "admin")
	if c.Locals("role") != "admin" {
		t.Errorf("Locals multiple: want admin, got %v", c.Locals("role"))
	}
}

func TestCtx_Next(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c := newCtx(req)

	var calls int
	h1 := func(ctx *Ctx) error { calls++; return ctx.Next() }
	h2 := func(ctx *Ctx) error { calls++; return nil }
	
	c.handlers = []HandlerFunc{h1, h2}
	
	err := c.Next()
	if err != nil {
		t.Errorf("Next error: %v", err)
	}
	if calls != 2 {
		t.Errorf("Next calls: want 2, got %d", calls)
	}
	
	// Test calling next out of bounds
	if err := c.Next(); err != nil {
		t.Errorf("Next out of bounds should return nil, got %v", err)
	}
}

func TestCtx_IP(t *testing.T) {
	// 1. Tanpa proxy, IP normal dengan port
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.RemoteAddr = "192.168.1.5:8080"
	c1 := &Ctx{Request: req1}
	if c1.IP() != "192.168.1.5" {
		t.Errorf("IP normal: want 192.168.1.5, got %s", c1.IP())
	}

	// 2. Tanpa proxy, format aneh/tanpa port (akan gagal SplitHostPort)
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.RemoteAddr = "10.0.0.1"
	c2 := &Ctx{Request: req2}
	if c2.IP() != "10.0.0.1" {
		t.Errorf("IP no-port: want 10.0.0.1, got %s", c2.IP())
	}

	// 3. Dengan proxy diaktifkan, ada header valid
	req3 := httptest.NewRequest(http.MethodGet, "/", nil)
	req3.RemoteAddr = "127.0.0.1:9000"
	req3.Header.Set("X-Forwarded-For", "203.0.113.1, 198.51.100.1")
	c3 := &Ctx{
		Request: req3,
		cfg: &config{
			trustProxy:  true,
			proxyHeader: "X-Forwarded-For",
		},
	}
	if c3.IP() != "203.0.113.1" {
		t.Errorf("IP proxy: want 203.0.113.1, got %s", c3.IP())
	}

	// 4. Dengan proxy diaktifkan, tapi header kosong (fallback ke RemoteAddr)
	req4 := httptest.NewRequest(http.MethodGet, "/", nil)
	req4.RemoteAddr = "127.0.0.1:9000"
	req4.Header.Set("X-Forwarded-For", "   ")
	c4 := &Ctx{
		Request: req4,
		cfg: &config{
			trustProxy:  true,
			proxyHeader: "X-Forwarded-For",
		},
	}
	if c4.IP() != "127.0.0.1" {
		t.Errorf("IP proxy empty header: want 127.0.0.1, got %s", c4.IP())
	}
}
