package gosip

import (
	"strconv"
	"strings"
)

type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	AllowCredentials bool
	MaxAge           int
}

func CORS(cfg CORSConfig) HandlerFunc {
	if len(cfg.AllowMethods) == 0 {
		cfg.AllowMethods = []string{MethodGet, MethodPost, MethodPut, MethodPatch, MethodDelete, MethodOptions}
	}
	if len(cfg.AllowHeaders) == 0 {
		cfg.AllowHeaders = []string{"Content-Type", "Authorization"}
	}
	if cfg.MaxAge == 0 {
		cfg.MaxAge = 86400
	}

	allowAll := len(cfg.AllowOrigins) == 1 && cfg.AllowOrigins[0] == "*"
	if allowAll && cfg.AllowCredentials {
		panic("gosip: CORS AllowOrigins \"*\" tidak boleh dipakai bersama AllowCredentials=true")
	}

	origins := make(map[string]struct{}, len(cfg.AllowOrigins))
	for _, o := range cfg.AllowOrigins {
		origins[o] = struct{}{}
	}

	allowMethods := strings.Join(cfg.AllowMethods, ", ")
	allowHeaders := strings.Join(cfg.AllowHeaders, ", ")
	maxAge := strconv.Itoa(cfg.MaxAge)

	return func(c *Ctx) error {
		origin := c.GetHeader("Origin")

		switch {
		case allowAll:
			c.SetHeader("Access-Control-Allow-Origin", "*")
		case origin != "":
			if _, ok := origins[origin]; ok {
				c.SetHeader("Access-Control-Allow-Origin", origin)
				c.SetHeader("Vary", "Origin")
			}
		}

		if cfg.AllowCredentials {
			c.SetHeader("Access-Control-Allow-Credentials", "true")
		}

		if c.Request.Method == MethodOptions {
			c.SetHeader("Access-Control-Allow-Methods", allowMethods)
			c.SetHeader("Access-Control-Allow-Headers", allowHeaders)
			c.SetHeader("Access-Control-Max-Age", maxAge)
			c.Status(StatusNoContent)
			c.writeHeader() // tulis header tanpa body, request berhenti di sini
			return nil
		}

		return c.Next()
	}
}
