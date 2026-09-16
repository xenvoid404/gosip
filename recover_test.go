package gosip

import (
	"net/http"
	"net/http/httptest"
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
