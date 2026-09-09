package gosip

import (
	"encoding/json"
	"net/http"
)

type Context struct {
	ResponseWriter http.ResponseWriter
	Request        *http.Request
	handlers       []HandlerFunc
	index          int
	statusCode     int
	wroteHeader    bool
	errors         []error
}

func (c *Context) Next() {
	c.index++
	if c.index < len(c.handlers) {
		c.handlers[c.index](c)
	}
}

func (c *Context) Status(code int) *Context {
	c.statusCode = code
	return c
}

func (c *Context) JSON(data any) *Context {
	c.ResponseWriter.Header().Set("Content-Type", "application/json")
	c.writeHeader()
	if err := json.NewEncoder(c.ResponseWriter).Encode(data); err != nil {
		c.Error(err)
	}
	return c
}

func (c *Context) String(data string) *Context {
	c.ResponseWriter.Header().Set("Content-Type", "text/plain")
	c.writeHeader()
	if _, err := c.ResponseWriter.Write([]byte(data)); err != nil {
		c.Error(err)
	}
	return c
}

func (c *Context) Error(err error) {
	c.errors = append(c.errors, err)
}

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

func (c *Context) Query(key string) string {
	return c.Request.URL.Query().Get(key)
}

func (c *Context) Params(key string) string {
	return c.Request.PathValue(key)
}
