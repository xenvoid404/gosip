package gosip

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
)

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
