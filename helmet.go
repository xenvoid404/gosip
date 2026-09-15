package gosip

import "fmt"

type HelmetConfig struct {
	XFrameOptions         string
	ContentSecurityPolicy string
	HSTS                  bool
	HSTSMaxAge            int
}

func Helmet(cfg ...HelmetConfig) HandlerFunc {
	c := HelmetConfig{}
	if len(cfg) > 0 {
		c = cfg[0]
	}
	if c.XFrameOptions == "" {
		c.XFrameOptions = "DENY"
	}
	if c.HSTS && c.HSTSMaxAge == 0 {
		c.HSTSMaxAge = 31536000
	}

	return func(ctx *Ctx) error {
		ctx.SetHeader("X-Content-Type-Options", "nosniff")
		ctx.SetHeader("X-Frame-Options", c.XFrameOptions)
		ctx.SetHeader("Referrer-Policy", "no-referrer")
		ctx.SetHeader("X-XSS-Protection", "0") // browser modern sudah deprecate; "0" sesuai rekomendasi OWASP terbaru
		ctx.SetHeader("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		ctx.SetHeader("Cross-Origin-Opener-Policy", "same-origin")
		ctx.SetHeader("Cross-Origin-Resource-Policy", "same-origin")

		if c.ContentSecurityPolicy != "" {
			ctx.SetHeader("Content-Security-Policy", c.ContentSecurityPolicy)
		}
		if c.HSTS {
			ctx.SetHeader("Strict-Transport-Security", fmt.Sprintf("max-age=%d; includeSubDomains", c.HSTSMaxAge))
		}

		return ctx.Next()
	}
}
