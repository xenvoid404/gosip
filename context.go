// Package gosip adalah wrapper net/http router yang ringan dan minimalis.
package gosip

import (
	"encoding/json"
	"net/http"
)

// Context membawa objek request, response, dan state dari siklus eksekusi HTTP.
type Context struct {
	ResponseWriter http.ResponseWriter
	Request        *http.Request
	handlers       []HandlerFunc
	index          int
	statusCode     int
	wroteHeader    bool
}

// Next mengeksekusi handler atau middleware selanjutnya dalam rantai antrean.
func (c *Context) Next() error {
	c.index++
	if c.index < len(c.handlers) {
		return c.handlers[c.index](c)
	}
	return nil
}

// Status menetapkan kode status HTTP untuk respons.
func (c *Context) Status(code int) *Context {
	c.statusCode = code
	return c
}

// JSON mengirimkan respons berformat JSON dan secara otomatis mengatur header Content-Type.
func (c *Context) JSON(data any) error {
	c.ResponseWriter.Header().Set("Content-Type", "application/json")
	c.writeHeader()
	return json.NewEncoder(c.ResponseWriter).Encode(data)
}

// String mengirimkan respons berupa teks biasa (plain text).
func (c *Context) String(data string) error {
	c.ResponseWriter.Header().Set("Content-Type", "text/plain")
	c.writeHeader()
	_, err := c.ResponseWriter.Write([]byte(data))
	return err
}

// writeHeader menuliskan status header ke response jika belum ditulis sebelumnya.
func (c *Context) writeHeader() {
	if c.wroteHeader {
		return
	}
	if c.statusCode == 0 {
		c.statusCode = StatusOK
	}
	c.ResponseWriter.WriteHeader(c.statusCode)
	c.wroteHeader = true
}

// Query mengambil nilai dari parameter query URL berdasarkan kunci yang diberikan.
func (c *Context) Query(key string) string {
	return c.Request.URL.Query().Get(key)
}

// Params mengambil nilai dari parameter path dinamis URL berdasarkan kunci yang diberikan.
func (c *Context) Params(key string) string {
	return c.Request.PathValue(key)
}
