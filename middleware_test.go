package gosip

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRecoverMiddleware(t *testing.T) {
	app := New()
	app.Use(Recover())

	app.Get("/panic", func(c *Ctx) error {
		panic("test panic")
	})

	app.Get("/abort", func(c *Ctx) error {
		panic(http.ErrAbortHandler)
	})

	// Test normal panic
	req1 := httptest.NewRequest(MethodGet, "/panic", nil)
	w1 := httptest.NewRecorder()
	app.ServeHTTP(w1, req1)
	
	if w1.Code != StatusInternalServerError {
		t.Errorf("Recover panic Code: want 500, got %d", w1.Code)
	}

	// Test http.ErrAbortHandler panic
	defer func() {
		r := recover()
		if r != http.ErrAbortHandler {
			t.Errorf("Recover abort: want http.ErrAbortHandler, got %v", r)
		}
	}()
	
	req2 := httptest.NewRequest(MethodGet, "/abort", nil)
	w2 := httptest.NewRecorder()
	app.ServeHTTP(w2, req2)
}

func TestLoggerMiddleware(t *testing.T) {
	// Hijack slog
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	logger := slog.New(handler)
	slog.SetDefault(logger)

	app := New()
	app.Use(Logger())

	app.Get("/ok", func(c *Ctx) error {
		return c.Status(StatusOK).String("ok")
	})

	app.Get("/error", func(c *Ctx) error {
		c.Status(StatusBadRequest)
		return errors.New("test error")
	})

	// Test success log
	req1 := httptest.NewRequest(MethodGet, "/ok", nil)
	w1 := httptest.NewRecorder()
	app.ServeHTTP(w1, req1)

	if !strings.Contains(buf.String(), "request masuk") {
		t.Errorf("Logger success: want 'request masuk' in log, got: %s", buf.String())
	}
	if !strings.Contains(buf.String(), "status=200") {
		t.Errorf("Logger success: want status=200, got: %s", buf.String())
	}

	buf.Reset()

	// Test error log
	req2 := httptest.NewRequest(MethodGet, "/error", nil)
	w2 := httptest.NewRecorder()
	app.ServeHTTP(w2, req2)

	if !strings.Contains(buf.String(), "request gagal") {
		t.Errorf("Logger error: want 'request gagal' in log, got: %s", buf.String())
	}
	if !strings.Contains(buf.String(), "test error") {
		t.Errorf("Logger error: want 'test error', got: %s", buf.String())
	}
	if !strings.Contains(buf.String(), "status=400") {
		t.Errorf("Logger error: want status=400, got: %s", buf.String())
	}
}
