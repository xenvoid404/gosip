package gosip

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Recover ini middleware penyelamat!
// Kalau handler kamu tiba-tiba panic, middleware ini bakal nangkep panic-nya,
// nge-log pakai slog, dan ngubah panic itu jadi error biasa biar server nggak mati.
// Pengecualian buat http.ErrAbortHandler, dia tetep di-panic sesuai standar stdlib.
func Recover() HandlerFunc {
	return func(c *Ctx) (err error) {
		defer func() {
			if r := recover(); r != nil {
				if r == http.ErrAbortHandler {
					panic(r)
				}
				slog.Error("gosip: panic recovered",
					slog.Any("panic", r),
					slog.String("stack", string(debug.Stack())),
				)
				err = fmt.Errorf("panic: %v", r)
			}
		}()
		return c.Next()
	}
}
