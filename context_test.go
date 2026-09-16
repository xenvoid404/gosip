package gosip

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newCtx(req *http.Request) *Ctx {
	w := httptest.NewRecorder()
	return &Ctx{ResponseWriter: w, Request: req, index: -1}
}

func TestCtx_ContextMethodPathRemoteAddr(t *testing.T) {
	req := httptest.NewRequest(MethodPost, "/castella", nil)
	req.RemoteAddr = "192.168.1.1:1234"

	// Buat context dengan custom value untuk tes Context()
	type key string
	req = req.WithContext(context.WithValue(req.Context(), key("kim"), "dahyun"))

	c := newCtx(req)

	if c.Method() != MethodPost {
		t.Errorf("Method() = %s, harusnya: POST", c.Method())
	}
	if c.Path() != "/castella" {
		t.Errorf("Path() = %s, harusnya: /hello", c.Path())
	}
	if c.RemoteAddr() != "192.168.1.1:1234" {
		t.Errorf("RemoteAddr() = %s, harusnya: 192.168.1.1:1234", c.RemoteAddr())
	}

	ctxVal := c.Context().Value(key("kim"))
	if ctxVal != "dahyun" {
		t.Errorf("Context() = %v, harusnya: bar", ctxVal)
	}
}

func TestCtx_StatusAndWroteHeader(t *testing.T) {
	req := httptest.NewRequest(MethodGet, "/", nil)
	c := newCtx(req)

	if c.StatusCode() != StatusOK {
		t.Errorf("StatusCode() = %d, harusnya: 200 (default)", c.StatusCode())
	}
	if c.WroteHeader() {
		t.Error("WroteHeader() = true, harusnya: false (default)")
	}

	c.Status(StatusCreated)
	if c.StatusCode() != StatusCreated {
		t.Errorf("StatusCode() setelah set = %d, harusnya: 201", c.StatusCode())
	}

	c.writeHeader() // Tulis header sekali aja
	if !c.WroteHeader() {
		t.Error("WroteHeader() setelah writeHeader = true, harusnya: false")
	}

	c.writeHeader() // Panggilan kerua harusnya dicuekin dan ngga bikin error/panic
}

func TestCtx_JSON(t *testing.T) {
	req := httptest.NewRequest(MethodGet, "/", nil)
	w := httptest.NewRecorder()
	c := &Ctx{ResponseWriter: w, Request: req}

	err := c.JSON(Map{"kim": "dahyun"})
	if err != nil {
		t.Errorf("JSON() = %v, harusnya ngga error", err)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("JSON() Content-Type = %s, harusnya: application/json", w.Header().Get("Content-Type"))
	}
	if w.Body.String() != `{"kim":"dahyun"}` {
		t.Errorf("JSON() Body = %s, harusnya: {\"msg\":\"ok\"}", w.Body.String())
	}

	// Tes marshal dengan tipe yang tidak didukung seperti channel
	ch := make(chan int)
	err = c.JSON(ch)
	if err == nil {
		t.Error("JSON() dengan channel harusnya error")
	}
}

func TestCtx_String(t *testing.T) {
	req := httptest.NewRequest(MethodGet, "/", nil)
	w := httptest.NewRecorder()
	c := &Ctx{ResponseWriter: w, Request: req}

	err := c.String("castella")
	if err != nil {
		t.Errorf("String() = %v, harusnya ngga error", err)
	}

	if w.Header().Get("Content-Type") != "text/plain" {
		t.Errorf("String() Content-Type = %s, harusnya: text/plain", w.Header().Get("Content-Type"))
	}
	if w.Body.String() != "castella" {
		t.Errorf("String() Body = %s, harusnya: castella", w.Body.String())
	}
}

func TestCtx_QueryAndParams(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/search?q=castella", nil)
	req.SetPathValue("id", "42")

	c := newCtx(req)

	if c.Query("q") != "castella" {
		t.Errorf("Query() = %s, harusnya: castella", c.Query("q"))
	}
	if c.Query("hantu") != "" {
		t.Errorf("Query() hantu = %s, harusnya kosong", c.Query("hantu"))
	}

	if c.Params("id") != "42" {
		t.Errorf("Params() = %s, harusnya: 42", c.Params("id"))
	}
}

func TestCtx_HeadersAndCookies(t *testing.T) {
	req := httptest.NewRequest(MethodGet, "/", nil)
	req.Header.Set("X-Idol", "castella")
	req.AddCookie(&http.Cookie{Name: "session", Value: "123"})

	w := httptest.NewRecorder()
	c := &Ctx{ResponseWriter: w, Request: req}

	if c.GetHeader("X-Idol") != "castella" {
		t.Errorf("GetHeader() = %s, harusnya: castella", c.GetHeader("X-Idol"))
	}

	c.SetHeader("X-Presiden", "mbg")
	if w.Header().Get("X-Presiden") != "mbg" {
		t.Errorf("SetHeader() = %s, harusnya: mbg", w.Header().Get("X-Presiden"))
	}

	cookie, err := c.GetCookie("session")
	if err != nil || cookie.Value != "123" {
		t.Errorf("GetCookie() = %v (err: %v), harusnya: 123", cookie, err)
	}

	_, err = c.GetCookie("hantu")
	if err == nil {
		t.Error("GetCookie() hantu harusnya error")
	}

	c.SetCookie(&http.Cookie{Name: "new_session", Value: "456"})
	resCookie := w.Header().Get("Set-Cookie")
	if resCookie != "new_session=456" {
		t.Errorf("SetCookie() = %s, harusnya: new_session=456", resCookie)
	}
}

func TestCtx_Locals(t *testing.T) {
	req := httptest.NewRequest(MethodGet, "/", nil)
	c := newCtx(req)

	if c.Locals("user") != nil {
		t.Errorf("Locals() = %s, harusnya: nil", c.Locals("user"))
	}

	c.Locals("user", "castella")
	if c.Locals("user") != "castella" {
		t.Errorf("Locals() setelah set = %v, harusnya: castella", c.Locals("user"))
	}

	// Test dengan nilai lain
	c.Locals("role", "vocal")
	if c.Locals("role") != "vocal" {
		t.Errorf("Locals() timpa = %v, harusnya: vocal", c.Locals("role"))
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
		t.Errorf("Next() error: %v", err)
	}
	if calls != 2 {
		t.Errorf("Next() calls = %d, harusnya 2", calls)
	}

	if err := c.Next(); err != nil {
		t.Errorf("Next() = %v, harusnya: nil", err)
	}
}

func TestCtx_IP(t *testing.T) {
	// 1. Tanpa proxy, IP normal dengan port
	req1 := httptest.NewRequest(MethodGet, "/", nil)
	req1.RemoteAddr = "192.168.1.5:8080"
	c1 := &Ctx{Request: req1}
	if c1.IP() != "192.168.1.5" {
		t.Errorf("IP() normal = %s, harusnya: 192.168.1.5", c1.IP())
	}

	// 2. Tanpa proxy, format aneh/tanpa port (akan gagal SplitHostPort)
	req2 := httptest.NewRequest(MethodGet, "/", nil)
	req2.RemoteAddr = "10.0.0.1"
	c2 := &Ctx{Request: req2}
	if c2.IP() != "10.0.0.1" {
		t.Errorf("IP() tanpa port = %s, harusnya: 10.0.0.1", c2.IP())
	}

	// 3. Dengan proxy diaktifkan, ada header valid
	req3 := httptest.NewRequest(MethodGet, "/", nil)
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
		t.Errorf("IP() proxy = %s, harusnya: 203.0.113.1", c3.IP())
	}

	// 4. Dengan proxy diaktifkan, tapi header kosong (fallback ke RemoteAddr)
	req4 := httptest.NewRequest(MethodGet, "/", nil)
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
		t.Errorf("IP() proxy dengan header kosong = %s, harusnya: 127.0.0.1", c4.IP())
	}
}
