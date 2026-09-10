package gosip

import (
	"errors"
	"log"
	"runtime/debug"
)

// ErrPanic adalah error yang dikembalikan Recover ketika menangkap panic.
// Pesan detail panic tetap dicatat via log, tetapi error yang dilempar ke
// error handler bersifat generik agar tidak bocor ke klien.
var ErrPanic = errors.New("panic recovered")

// Recover mengembalikan middleware yang menangkap panic dari handler di
// hilirnya. Panic diubah menjadi error sehingga ditangani oleh error
// handler yang terdaftar via Router.OnError.
func Recover() HandlerFunc {
	return func(c *Context) (err error) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("gosip: panic: %v\n%s", rec, debug.Stack())
				err = ErrPanic
			}
		}()
		return c.Next()
	}
}
