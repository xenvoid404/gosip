package gosip

import (
	"log/slog"
	"time"
)

func Logger() HandlerFunc {
	return func(c *Ctx) error {
		start := time.Now()
		err := c.Next()
		duration := time.Since(start)

		if err != nil {
			slog.Error("request gagal",
				slog.String("method", c.Method()),
				slog.String("path", c.Path()),
				slog.String("ip", c.IP()),
				slog.Int("status", c.StatusCode()),
				slog.Duration("durasi", duration),
				slog.Any("error", err),
			)
			return err
		}

		slog.Info("request masuk",
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
			slog.String("ip", c.IP()),
			slog.Duration("durasi", duration),
			slog.Int("status", c.StatusCode()),
		)
		return nil
	}
}
