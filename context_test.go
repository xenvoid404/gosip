package gosip

import (
	"net/http/httptest"
	"testing"
)

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
