package gosip

import (
	"encoding/json"
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
	c.String("hidup joko")

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
	c.Status(StatusCreated).String("hidup joko")
	if rec.Code != StatusCreated {
		t.Fatalf("status = %d, harusnya: %d", rec.Code, StatusCreated)
	}
}

func TestContextJSON(t *testing.T) {
	c, rec := newTestContext(http.MethodGet, "/")

	c.Status(StatusOK).JSON(Map{"kasih": "pahambos"})

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

	c.Status(StatusOK).String("kesatu")
	c.Status(StatusInternalServerError).String("kedua")

	if rec.Code != StatusOK {
		t.Fatalf("status = %d, harusnya: %d (status kedua harusnya dicuekin)", rec.Code, StatusOK)
	}
	if rec.Body.String() != "kesatukedua" {
		t.Fatalf("body = %q, harusnya: %q", rec.Body.String(), "kesatukedua")
	}
}

func TestContextQuery(t *testing.T) {
	req := httptest.NewRequest("GET", "/search?keyword=gosip&page=1", nil)
	rec := httptest.NewRecorder()
	c := &Context{
		ResponseWriter: rec,
		Request:        req,
	}

	// Tes ambil query keyword
	got := c.Query("keyword")
	want := "gosip"
	if got != want {
		t.Errorf("harusnya: %q, hasilnya: %q", want, got)
	}

	// Tes ambil query page
	got = c.Query("page")
	want = "1"
	if got != want {
		t.Errorf("harusnya: %q, hasilnya: %q", want, got)
	}

	// Tes ambil query yang tidak ada
	got = c.Query("ghost")
	want = ""
	if got != want {
		t.Errorf("harusnya: %q, hasilnya: %q", want, got)
	}
}
