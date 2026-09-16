package gosip

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ── BindJSON ─────────────────────────────────────────────────────────────────

func TestBindJSON_OK(t *testing.T) {
	type Body struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	payload, _ := json.Marshal(Body{Name: "Dika", Age: 22})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	c := newCtx(req)
	var got Body
	if err := c.BindJSON(&got); err != nil {
		t.Fatalf("tidak harusnya error: %v", err)
	}
	if got.Name != "Dika" || got.Age != 22 {
		t.Errorf("hasil tidak sesuai: %+v", got)
	}
}

func TestBindJSON_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(`{invalid`)))
	c := newCtx(req)
	var got any
	if err := c.BindJSON(&got); err == nil {
		t.Fatal("seharusnya return error untuk JSON tidak valid")
	}
}

func TestBindJSON_NotPointer(t *testing.T) {
	payload, _ := json.Marshal(map[string]any{"x": 1})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(payload))
	c := newCtx(req)
	// json.Decode ke non-pointer juga akan error
	var got map[string]any
	if err := c.BindJSON(got); err == nil {
		t.Fatal("seharusnya return error untuk non-pointer")
	}
}

// ── BindQuery ─────────────────────────────────────────────────────────────────

func TestBindQuery_Primitives(t *testing.T) {
	type Filter struct {
		Page    int     `query:"page"`
		Limit   int     `query:"limit"`
		Active  bool    `query:"active"`
		Score   float64 `query:"score"`
		Keyword string  `query:"q"`
	}

	req := httptest.NewRequest(http.MethodGet, "/?page=2&limit=10&active=true&score=9.5&q=golang", nil)
	c := newCtx(req)

	var f Filter
	if err := c.BindQuery(&f); err != nil {
		t.Fatalf("tidak harusnya error: %v", err)
	}
	if f.Page != 2 {
		t.Errorf("Page: want 2, got %d", f.Page)
	}
	if f.Limit != 10 {
		t.Errorf("Limit: want 10, got %d", f.Limit)
	}
	if !f.Active {
		t.Errorf("Active: want true, got %v", f.Active)
	}
	if f.Score != 9.5 {
		t.Errorf("Score: want 9.5, got %f", f.Score)
	}
	if f.Keyword != "golang" {
		t.Errorf("Keyword: want golang, got %s", f.Keyword)
	}
}

func TestBindQuery_Slice(t *testing.T) {
	type Filter struct {
		Tags []string `query:"tag"`
		IDs  []int    `query:"id"`
	}

	req := httptest.NewRequest(http.MethodGet, "/?tag=go&tag=web&id=1&id=2&id=3", nil)
	c := newCtx(req)

	var f Filter
	if err := c.BindQuery(&f); err != nil {
		t.Fatalf("tidak harusnya error: %v", err)
	}
	if len(f.Tags) != 2 || f.Tags[0] != "go" || f.Tags[1] != "web" {
		t.Errorf("Tags: want [go web], got %v", f.Tags)
	}
	if len(f.IDs) != 3 || f.IDs[0] != 1 || f.IDs[1] != 2 || f.IDs[2] != 3 {
		t.Errorf("IDs: want [1 2 3], got %v", f.IDs)
	}
}

func TestBindQuery_Pointer(t *testing.T) {
	type Filter struct {
		Limit *int `query:"limit"`
	}

	req := httptest.NewRequest(http.MethodGet, "/?limit=50", nil)
	c := newCtx(req)

	var f Filter
	if err := c.BindQuery(&f); err != nil {
		t.Fatalf("tidak harusnya error: %v", err)
	}
	if f.Limit == nil || *f.Limit != 50 {
		t.Errorf("Limit: want *50, got %v", f.Limit)
	}
}

func TestBindQuery_MissingField_LeftAsZero(t *testing.T) {
	type Filter struct {
		Page  int    `query:"page"`
		Extra string `query:"extra"`
	}

	req := httptest.NewRequest(http.MethodGet, "/?page=1", nil)
	c := newCtx(req)

	var f Filter
	if err := c.BindQuery(&f); err != nil {
		t.Fatalf("tidak harusnya error: %v", err)
	}
	if f.Extra != "" {
		t.Errorf("Extra harusnya zero value, got %q", f.Extra)
	}
}

func TestBindQuery_InvalidInt(t *testing.T) {
	type Filter struct {
		Page int `query:"page"`
	}
	req := httptest.NewRequest(http.MethodGet, "/?page=abc", nil)
	c := newCtx(req)
	var f Filter
	if err := c.BindQuery(&f); err == nil {
		t.Fatal("seharusnya return error untuk nilai int tidak valid")
	}
}

func TestBindQuery_NotPointer(t *testing.T) {
	type Filter struct {
		Page int `query:"page"`
	}
	req := httptest.NewRequest(http.MethodGet, "/?page=1", nil)
	c := newCtx(req)
	var f Filter
	if err := c.BindQuery(f); err == nil {
		t.Fatal("seharusnya return error untuk non-pointer")
	}
}

func TestBindQuery_DashTag_Ignored(t *testing.T) {
	type Filter struct {
		Internal string `query:"-"`
		Name     string `query:"name"`
	}
	req := httptest.NewRequest(http.MethodGet, "/?name=go&-=leaked", nil)
	c := newCtx(req)
	var f Filter
	if err := c.BindQuery(&f); err != nil {
		t.Fatalf("tidak harusnya error: %v", err)
	}
	if f.Internal != "" {
		t.Errorf("field dengan tag \"-\" harusnya diabaikan, got %q", f.Internal)
	}
	if f.Name != "go" {
		t.Errorf("Name: want go, got %s", f.Name)
	}
}

// ── BindParams ────────────────────────────────────────────────────────────────

func TestBindParams_OK(t *testing.T) {
	type Params struct {
		ID   int    `param:"id"`
		Slug string `param:"slug"`
	}

	mux := http.NewServeMux()
	var got Params
	var bindErr error

	mux.HandleFunc("GET /posts/{id}/{slug}", func(w http.ResponseWriter, r *http.Request) {
		c := &Ctx{ResponseWriter: w, Request: r, index: -1}
		bindErr = c.BindParams(&got)
	})

	req := httptest.NewRequest(http.MethodGet, "/posts/42/hello-world", nil)
	mux.ServeHTTP(httptest.NewRecorder(), req)

	if bindErr != nil {
		t.Fatalf("tidak harusnya error: %v", bindErr)
	}
	if got.ID != 42 {
		t.Errorf("ID: want 42, got %d", got.ID)
	}
	if got.Slug != "hello-world" {
		t.Errorf("Slug: want hello-world, got %s", got.Slug)
	}
}

func TestBindParams_InvalidInt(t *testing.T) {
	type Params struct {
		ID int `param:"id"`
	}

	mux := http.NewServeMux()
	var bindErr error

	mux.HandleFunc("GET /items/{id}", func(w http.ResponseWriter, r *http.Request) {
		c := &Ctx{ResponseWriter: w, Request: r, index: -1}
		var p Params
		bindErr = c.BindParams(&p)
	})

	req := httptest.NewRequest(http.MethodGet, "/items/abc", nil)
	mux.ServeHTTP(httptest.NewRecorder(), req)

	if bindErr == nil {
		t.Fatal("seharusnya return error untuk nilai int tidak valid")
	}
}

func TestBindParams_NotPointer(t *testing.T) {
	type Params struct {
		ID int `param:"id"`
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c := newCtx(req)
	var p Params
	if err := c.BindParams(p); err == nil {
		t.Fatal("seharusnya return error untuk non-pointer")
	}
}

// ── Coverage Tests ────────────────────────────────────────────────────────────

func TestBindQuery_AllTypes_AndErrors(t *testing.T) {
	type AllTypes struct {
		UintField    uint      `query:"u"`
		FloatField   float32   `query:"f"`
		BoolField    bool      `query:"b"`
		ComplexField complex64 `query:"c"`
		unexported   int
		PointerErr   *int  `query:"pe"`
		SliceErr     []int `query:"se"`
	}

	// Test valid all types
	req := httptest.NewRequest(http.MethodGet, "/?u=123&f=12.5&b=true", nil)
	c := newCtx(req)
	var valid AllTypes
	if err := c.BindQuery(&valid); err != nil {
		t.Fatalf("tidak harusnya error: %v", err)
	}
	if valid.UintField != 123 || valid.FloatField != 12.5 || !valid.BoolField {
		t.Errorf("hasil binding AllTypes valid tidak sesuai: %+v", valid)
	}

	// Test invalid uint
	reqUint := httptest.NewRequest(http.MethodGet, "/?u=abc", nil)
	if err := newCtx(reqUint).BindQuery(&AllTypes{}); err == nil {
		t.Error("seharusnya error invalid uint")
	}

	// Test invalid float
	reqFloat := httptest.NewRequest(http.MethodGet, "/?f=abc", nil)
	if err := newCtx(reqFloat).BindQuery(&AllTypes{}); err == nil {
		t.Error("seharusnya error invalid float")
	}

	// Test invalid bool
	reqBool := httptest.NewRequest(http.MethodGet, "/?b=notabool", nil)
	if err := newCtx(reqBool).BindQuery(&AllTypes{}); err == nil {
		t.Error("seharusnya error invalid bool")
	}

	// Test unsupported type
	reqComplex := httptest.NewRequest(http.MethodGet, "/?c=1+2i", nil)
	if err := newCtx(reqComplex).BindQuery(&AllTypes{}); err == nil {
		t.Error("seharusnya error unsupported type")
	}

	// Test pointer error
	reqPtrErr := httptest.NewRequest(http.MethodGet, "/?pe=abc", nil)
	if err := newCtx(reqPtrErr).BindQuery(&AllTypes{}); err == nil {
		t.Error("seharusnya error pointer ke invalid int")
	}

	// Test slice error
	reqSliceErr := httptest.NewRequest(http.MethodGet, "/?se=1&se=abc", nil)
	if err := newCtx(reqSliceErr).BindQuery(&AllTypes{}); err == nil {
		t.Error("seharusnya error elemen slice invalid")
	}
}

func TestBind_StructElem_NonStructPtr(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?q=1", nil)
	c := newCtx(req)
	var x int
	if err := c.BindQuery(&x); err == nil {
		t.Error("seharusnya error jika bukan pointer ke struct")
	}
}

func TestBindParams_EmptyValAndUnexported(t *testing.T) {
	type Params struct {
		unexported int
		ID         int `param:"id"`
	}

	mux := http.NewServeMux()
	var bindErr error

	// Pattern URL tidak mendefinisikan {id}, jadi PathValue("id") akan ""
	mux.HandleFunc("GET /posts", func(w http.ResponseWriter, r *http.Request) {
		c := &Ctx{ResponseWriter: w, Request: r, index: -1}
		var p Params
		bindErr = c.BindParams(&p)
	})

	req := httptest.NewRequest(http.MethodGet, "/posts", nil)
	mux.ServeHTTP(httptest.NewRecorder(), req)

	if bindErr != nil {
		t.Errorf("tidak harusnya error walau parameter kosong (hanya diabaikan), got: %v", bindErr)
	}
}
