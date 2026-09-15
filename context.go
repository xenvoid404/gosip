package gosip

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strings"
)

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

func (c *Ctx) Context() context.Context {
	return c.Request.Context()
}

func (c *Ctx) Method() string {
	return c.Request.Method
}

func (c *Ctx) Path() string {
	return c.Request.URL.Path
}

func (c *Ctx) RemoteAddr() string {
	return c.Request.RemoteAddr
}

func (c *Ctx) Status(code int) *Ctx {
	c.statusCode = code
	return c
}

func (c *Ctx) StatusCode() int {
	if c.statusCode == 0 {
		return StatusOK
	}
	return c.statusCode
}

func (c *Ctx) WroteHeader() bool {
	return c.wroteHeader
}

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

func (c *Ctx) String(data string) error {
	c.ResponseWriter.Header().Set("Content-Type", "text/plain")
	c.writeHeader()
	_, err := c.ResponseWriter.Write([]byte(data))
	return err
}

func (c *Ctx) writeHeader() {
	if c.wroteHeader {
		return
	}
	c.ResponseWriter.WriteHeader(c.StatusCode())
	c.wroteHeader = true
}

func (c *Ctx) Query(key string) string {
	return c.Request.URL.Query().Get(key)
}

func (c *Ctx) Params(key string) string {
	return c.Request.PathValue(key)
}

func (c *Ctx) SetHeader(key, value string) {
	c.ResponseWriter.Header().Set(key, value)
}

func (c *Ctx) GetHeader(key string) string {
	return c.Request.Header.Get(key)
}

func (c *Ctx) SetCookie(cookie *http.Cookie) {
	http.SetCookie(c.ResponseWriter, cookie)
}

func (c *Ctx) GetCookie(name string) (*http.Cookie, error) {
	return c.Request.Cookie(name)
}

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

func (c *Ctx) Next() error {
	c.index++
	if c.index < len(c.handlers) {
		return c.handlers[c.index](c)
	}
	return nil
}

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
