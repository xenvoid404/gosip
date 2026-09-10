package gosip

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestContext(method, target string) (*Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, target, nil)
	return &Context{ResponseWriter: rec, Request: req, index: -1}, rec
}

func TestContextStatusDefault(t *testing.T) {
	c, rec := newTestContext(http.MethodGet, "/")
	_ = c.String("hidup joko")

	if rec.Code != StatusOK {
		t.Fatalf("status = %d, harusnya: %d", rec.Code, StatusOK)
	}
	if rec.Body.String() != "hidup joko" {
		t.Fatalf("body = %q, harusnya: %q", rec.Body.String(), "hidup joko")
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/plain" {
		t.Fatalf("Content-Type = %q, harusnya: %q", ct, "text/plain")
	}
}

func TestContextStatus(t *testing.T) {
	c, rec := newTestContext(http.MethodGet, "/")
	_ = c.Status(StatusCreated).String("hidup joko")
	if rec.Code != StatusCreated {
		t.Fatalf("status = %d, harusnya: %d", rec.Code, StatusCreated)
	}
}

func TestContextJSON(t *testing.T) {
	c, rec := newTestContext(http.MethodGet, "/")

	_ = c.Status(StatusOK).JSON(map[string]string{"kasih": "pahambos"})

	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, harusnya: %q", ct, "application/json")
	}

	var got map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("gagal decode body: %v", err)
	}
	if got["kasih"] != "pahambos" {
		t.Fatalf("body = %v, harusnya kasih=pahambos", got)
	}
}

func TestContextWriteHeader(t *testing.T) {
	c, rec := newTestContext(http.MethodGet, "/")

	_ = c.Status(StatusOK).String("kesatu")
	_ = c.Status(StatusInternalServerError).String("kedua")

	if rec.Code != StatusOK {
		t.Fatalf("status = %d, harusnya: %d (status kedua harusnya dicuekin)", rec.Code, StatusOK)
	}
	if rec.Body.String() != "kesatukedua" {
		t.Fatalf("body = %q, harusnya: %q", rec.Body.String(), "kesatukedua")
	}
}

func TestContextQuery(t *testing.T) {
	c, _ := newTestContext(http.MethodGet, "/search?keyword=gosip")

	if got := c.Query("keyword"); got != "gosip" {
		t.Fatalf("keyword: %q, harusnya: %q", got, "gosip")
	}
	if got := c.Query("ghost"); got != "" {
		t.Fatalf("ghost: %q, harusnya: %q", got, "")
	}
}

func TestContextParams(t *testing.T) {
	c, _ := newTestContext(http.MethodGet, "/capres/3")
	c.Request.SetPathValue("id", "3")
	if got := c.Params("id"); got != "3" {
		t.Fatalf("id = %q, harusnya %q", got, "3")
	}
}

func TestContextNextRun(t *testing.T) {
	var order []int
	handlers := []HandlerFunc{
		func(c *Context) error { order = append(order, 1); return c.Next() },
		func(c *Context) error { order = append(order, 2); return c.Next() },
		func(c *Context) error { order = append(order, 3); return nil },
	}
	c := &Context{handlers: handlers, index: -1}
	_ = c.Next()

	want := []int{1, 2, 3}
	if len(order) != len(want) {
		t.Fatalf("order = %v, harusnya %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order = %v, harusnya %v", order, want)
		}
	}
}

func TestContextNextNothing(t *testing.T) {
	called := 0
	handlers := []HandlerFunc{
		func(c *Context) error { called++; return nil },
	}
	c := &Context{handlers: handlers, index: -1}

	_ = c.Next()
	_ = c.Next() // pemanggilan ekstra ngga boleh jalanin handler lagi

	if called != 1 {
		t.Fatalf("handler dipanggil %d kali, harusnya 1", called)
	}
}

func TestContextNextPropagatesError(t *testing.T) {
	wantErr := errors.New("boom")
	handlers := []HandlerFunc{
		func(c *Context) error { return c.Next() },
		func(c *Context) error { return wantErr },
	}
	c := &Context{handlers: handlers, index: -1}

	if err := c.Next(); !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, harusnya %v", err, wantErr)
	}
}
