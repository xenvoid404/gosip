package gosip

import (
	"net/http/httptest"
	"testing"
)

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
