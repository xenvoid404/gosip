package gosip

import (
	"net/http/httptest"
	"testing"
)

func TestContextStatusDefault(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	c := &Context{
		ResponseWriter: rec,
		Request:        req,
		index:          -1,
	}

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
