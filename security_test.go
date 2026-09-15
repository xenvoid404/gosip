package gosip

import (
	"net/http/httptest"
	"testing"
)

func TestCORS_Defaults(t *testing.T) {
	app := New()
	app.Use(CORS(CORSConfig{
		AllowOrigins: []string{"*"},
	}))

	app.Get("/", func(c *Ctx) error { return c.String("ok") })

	// Test normal request
	req1 := httptest.NewRequest(MethodGet, "/", nil)
	w1 := httptest.NewRecorder()
	app.ServeHTTP(w1, req1)
	if w1.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("CORS Allow-Origin: want *, got %s", w1.Header().Get("Access-Control-Allow-Origin"))
	}

	// Test preflight (OPTIONS)
	req2 := httptest.NewRequest(MethodOptions, "/", nil)
	w2 := httptest.NewRecorder()
	app.ServeHTTP(w2, req2)
	
	if w2.Code != StatusNoContent {
		t.Errorf("CORS OPTIONS Code: want 204, got %d", w2.Code)
	}
	if w2.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("CORS OPTIONS should have Access-Control-Allow-Methods")
	}
	if w2.Header().Get("Access-Control-Max-Age") != "86400" {
		t.Errorf("CORS Max-Age default: want 86400, got %s", w2.Header().Get("Access-Control-Max-Age"))
	}
}

func TestCORS_SpecificOrigin_And_Credentials(t *testing.T) {
	app := New()
	app.Use(CORS(CORSConfig{
		AllowOrigins:     []string{"https://example.com"},
		AllowCredentials: true,
		MaxAge:           3600,
	}))
	app.Get("/", func(c *Ctx) error { return c.String("ok") })

	// Test with matched origin
	req1 := httptest.NewRequest(MethodGet, "/", nil)
	req1.Header.Set("Origin", "https://example.com")
	w1 := httptest.NewRecorder()
	app.ServeHTTP(w1, req1)
	
	if w1.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
		t.Errorf("CORS Origin matched: want https://example.com, got %s", w1.Header().Get("Access-Control-Allow-Origin"))
	}
	if w1.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Errorf("CORS Credentials: want true, got %s", w1.Header().Get("Access-Control-Allow-Credentials"))
	}
	if w1.Header().Get("Vary") != "Origin" {
		t.Errorf("CORS Vary: want Origin, got %s", w1.Header().Get("Vary"))
	}

	// Test with unmatched origin
	req2 := httptest.NewRequest(MethodGet, "/", nil)
	req2.Header.Set("Origin", "https://hacker.com")
	w2 := httptest.NewRecorder()
	app.ServeHTTP(w2, req2)
	
	if w2.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("CORS Origin unmatched: want empty, got %s", w2.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORS_Panic_AllOrigin_WithCredentials(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("CORS should panic when AllowOrigins='*' and AllowCredentials=true")
		}
	}()
	
	CORS(CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowCredentials: true,
	})
}

func TestHelmet_Defaults(t *testing.T) {
	app := New()
	app.Use(Helmet())
	app.Get("/", func(c *Ctx) error { return c.String("ok") })

	req := httptest.NewRequest(MethodGet, "/", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	h := w.Header()
	if h.Get("X-Content-Type-Options") != "nosniff" {
		t.Error("Helmet missing X-Content-Type-Options")
	}
	if h.Get("X-Frame-Options") != "DENY" {
		t.Error("Helmet missing/wrong X-Frame-Options default")
	}
	if h.Get("Referrer-Policy") != "no-referrer" {
		t.Error("Helmet missing Referrer-Policy")
	}
	if h.Get("X-XSS-Protection") != "0" {
		t.Error("Helmet missing X-XSS-Protection")
	}
	if h.Get("Permissions-Policy") == "" {
		t.Error("Helmet missing Permissions-Policy")
	}
	if h.Get("Cross-Origin-Opener-Policy") != "same-origin" {
		t.Error("Helmet missing Cross-Origin-Opener-Policy")
	}
}

func TestHelmet_CustomConfig(t *testing.T) {
	app := New()
	app.Use(Helmet(HelmetConfig{
		XFrameOptions:         "SAMEORIGIN",
		ContentSecurityPolicy: "default-src 'self'",
		HSTS:                  true,
		// HSTSMaxAge deliberately left 0 to test default
	}))
	app.Get("/", func(c *Ctx) error { return c.String("ok") })

	req := httptest.NewRequest(MethodGet, "/", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	h := w.Header()
	if h.Get("X-Frame-Options") != "SAMEORIGIN" {
		t.Errorf("Helmet X-Frame-Options: want SAMEORIGIN, got %s", h.Get("X-Frame-Options"))
	}
	if h.Get("Content-Security-Policy") != "default-src 'self'" {
		t.Errorf("Helmet CSP: want default-src 'self', got %s", h.Get("Content-Security-Policy"))
	}
	if h.Get("Strict-Transport-Security") != "max-age=31536000; includeSubDomains" {
		t.Errorf("Helmet HSTS: want max-age=31536000..., got %s", h.Get("Strict-Transport-Security"))
	}
}
