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

func TestContextError(t *testing.T) {
	c, rec := newTestContext(http.MethodGet, "/")

	err1 := errors.New("e1")
	err2 := errors.New("e2")
	c.Error(err1)
	c.Error(err2)

	if len(c.errors) != 2 || c.errors[0] != err1 || c.errors[1] != err2 {
		t.Fatalf("error = %v, harusnya [%v, %v]", c.errors, err1, err2)
	}
	if c.wroteHeader {
		t.Fatal("harusnya ngga nulis header response")
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("harusnya ngga nulis body, body = %q", rec.Body.String())
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
