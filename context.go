package gosip

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strings"
)

// Ctx adalah nyawanya gosip! Ini wrapper buat http.ResponseWriter dan *http.Request.
// Semua interaksi request dan response bakal lewat struct ini.
type Ctx struct {
	ResponseWriter http.ResponseWriter
	Request        *http.Request
	handlers       []HandlerFunc
	index          int
	statusCode     int
	wroteHeader    bool
	locals         map[string]any
	cfg            *config
}

// Context ngambil context.Context bawaan dari request.
// Berguna banget kalau mau pass context ke database atau API luar.
func (c *Ctx) Context() context.Context {
	return c.Request.Context()
}

// Method balikin HTTP method dari request (misal: "GET", "POST").
func (c *Ctx) Method() string {
	return c.Request.Method
}

// Path balikin URL path dari request (misal: "/api/users").
func (c *Ctx) Path() string {
	return c.Request.URL.Path
}

// RemoteAddr balikin alamat IP asli klien dari request.
func (c *Ctx) RemoteAddr() string {
	return c.Request.RemoteAddr
}

// Status nge-set HTTP status code buat response nanti.
// Ingat, ini cuma nge-set valuenya, belum beneran dikirim ke klien sampai kamu nulis response.
// Enaknya, dia nge-return *Ctx lagi jadi bisa di-chain (contoh: c.Status(200).JSON(...)).
func (c *Ctx) Status(code int) *Ctx {
	c.statusCode = code
	return c
}

// StatusCode ngambil HTTP status code yang udah di-set.
// Kalau belum di-set sama sekali, dia otomatis balikin 200 (StatusOK).
func (c *Ctx) StatusCode() int {
	if c.statusCode == 0 {
		return StatusOK
	}
	return c.statusCode
}

// WroteHeader ngasih tau apakah HTTP header udah beneran dikirim ke klien atau belum.
func (c *Ctx) WroteHeader() bool {
	return c.wroteHeader
}

// JSON ngebantuin kirim response dalam format JSON.
// Udah otomatis nge-set header Content-Type ke application/json.
func (c *Ctx) JSON(data any) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	c.ResponseWriter.Header().Set("Content-Type", "application/json")
	c.writeHeader()
	_, err = c.ResponseWriter.Write(b)
	return err
}

// String ngebantuin kirim response berupa plain text biasa.
// Otomatis nge-set header Content-Type ke text/plain.
func (c *Ctx) String(data string) error {
	c.ResponseWriter.Header().Set("Content-Type", "text/plain")
	c.writeHeader()
	_, err := c.ResponseWriter.Write([]byte(data))
	return err
}

// writeHeader ini fungsi internal buat nulis HTTP status code ke response.
// Cuma dijalanin sekali biar nggak kena error double-write.
func (c *Ctx) writeHeader() {
	if c.wroteHeader {
		return
	}
	c.ResponseWriter.WriteHeader(c.StatusCode())
	c.wroteHeader = true
}

// Query ngambil nilai dari URL query parameter (contoh: ?key=value).
func (c *Ctx) Query(key string) string {
	return c.Request.URL.Query().Get(key)
}

// Params ngambil nilai dari URL path parameter (contoh: /users/{id}).
// Fitur ini manfaatin routing bawaan Go 1.22+.
func (c *Ctx) Params(key string) string {
	return c.Request.PathValue(key)
}

// SetHeader nambahin atau nimpa HTTP header di response.
func (c *Ctx) SetHeader(key, value string) {
	c.ResponseWriter.Header().Set(key, value)
}

// GetHeader ngambil nilai HTTP header dari request klien.
func (c *Ctx) GetHeader(key string) string {
	return c.Request.Header.Get(key)
}

// SetCookie nge-set HTTP cookie ke browser klien.
func (c *Ctx) SetCookie(cookie *http.Cookie) {
	http.SetCookie(c.ResponseWriter, cookie)
}

// GetCookie ngambil cookie tertentu dari request klien.
// Bakal error kalau cookienya nggak ketemu.
func (c *Ctx) GetCookie(name string) (*http.Cookie, error) {
	return c.Request.Cookie(name)
}

// Locals ini tempat nitip data antar middleware/handler (kayak context value).
// Kalau value-nya diisi, dia nge-set nilainya.
// Kalau nggak diisi, dia balikin nilai yang udah ada.
func (c *Ctx) Locals(key string, value ...any) any {
	if len(value) > 0 {
		if c.locals == nil {
			c.locals = make(map[string]any)
		}
		c.locals[key] = value[0]
		return value[0]
	}
	if c.locals == nil {
		return nil
	}
	return c.locals[key]
}

// Next manggil middleware/handler selanjutnya dalam chain.
// Wajib dipanggil kalau kamu bikin middleware, biar requestnya jalan terus.
func (c *Ctx) Next() error {
	c.index++
	if c.index < len(c.handlers) {
		return c.handlers[c.index](c)
	}
	return nil
}

// IP ngebantuin dapet IP address asli dari klien.
// Kalau trustProxy diaktifin di config, dia bakal ngecek header proxy (kayak X-Forwarded-For).
// Kalau nggak, dia langsung balikin dari RemoteAddr.
func (c *Ctx) IP() string {
	if c.cfg != nil && c.cfg.trustProxy {
		if v := c.GetHeader(c.cfg.proxyHeader); v != "" {
			if ip := strings.TrimSpace(strings.Split(v, ",")[0]); ip != "" {
				return ip
			}
		}
	}
	host, _, err := net.SplitHostPort(c.RemoteAddr())
	if err != nil {
		return c.RemoteAddr()
	}
	return host
}
