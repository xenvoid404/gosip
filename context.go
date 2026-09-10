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
}

func (c *Context) Next() error {
	c.index++
	if c.index < len(c.handlers) {
		return c.handlers[c.index](c)
	}
	return nil
}

func (c *Context) Status(code int) *Context {
	c.statusCode = code
	return c
}

func (c *Context) JSON(data any) error {
	c.ResponseWriter.Header().Set("Content-Type", "application/json")
	c.writeHeader()
	return json.NewEncoder(c.ResponseWriter).Encode(data)
}

func (c *Context) String(data string) error {
	c.ResponseWriter.Header().Set("Content-Type", "text/plain")
	c.writeHeader()
	_, err := c.ResponseWriter.Write([]byte(data))
	return err
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
