package gosip

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

const (
	defaultReadHeaderTimeout = 5 * time.Second
	defaultReadTimeout       = 15 * time.Second
	defaultWriteTimeout      = 15 * time.Second
	defaultIdleTimeout       = 60 * time.Second
	defaultShutdownTimeout   = 10 * time.Second
)

// HandlerFunc mendefinisikan signature fungsi untuk menangani request HTTP.
type HandlerFunc func(*Context) error

// ErrorHandlerFunc mendefinisikan signature fungsi untuk menangani error dari HandlerFunc.
type ErrorHandlerFunc func(error, *Context) error

// Router adalah HTTP request multiplexer yang mendukung registrasi rute, grup, dan middleware.
type Router struct {
	mux              *http.ServeMux
	middlewares      []HandlerFunc
	prefix           string
	root             *Router
	notFound         HandlerFunc
	methodNotAllowed HandlerFunc
	errorHandler     ErrorHandlerFunc
	routeCount       int
}

// New membuat dan mengembalikan instance Router baru dengan konfigurasi bawaan.
func New() *Router {
	r := &Router{
		mux: http.NewServeMux(),
	}
	r.root = r
	r.notFound = defaultNotFound
	r.methodNotAllowed = defaultMethodNotAllowed
	r.errorHandler = defaultErrorHandler
	r.mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		fallback := r.root.notFound
		if r.isMethodNotAllowed(req) {
			fallback = r.root.methodNotAllowed
		}

		c := &Context{
			ResponseWriter: w,
			Request:        req,
			handlers:       []HandlerFunc{fallback},
			index:          -1,
		}
		r.dispatch(c)
	})

	return r
}

// Use menambahkan middleware ke dalam router saat ini dan sub-grup di bawahnya.
func (r *Router) Use(middleware ...HandlerFunc) {
	r.middlewares = append(r.middlewares, middleware...)
}

// Group membuat dan mengembalikan instance Router baru yang berbagi mux sama,
// dengan tambahan prefix pada path.
func (r *Router) Group(prefix string) *Router {
	return &Router{
		mux:         r.mux,
		middlewares: append([]HandlerFunc{}, r.middlewares...),
		prefix:      joinPaths(r.prefix, prefix),
		root:        r.root,
	}
}

// Get mendaftarkan handler untuk metode HTTP GET.
func (r *Router) Get(pattern string, handlers ...HandlerFunc) {
	r.handle(http.MethodGet, pattern, handlers...)
}

// Post mendaftarkan handler untuk metode HTTP POST.
func (r *Router) Post(pattern string, handlers ...HandlerFunc) {
	r.handle(http.MethodPost, pattern, handlers...)
}

// Put mendaftarkan handler untuk metode HTTP PUT.
func (r *Router) Put(pattern string, handlers ...HandlerFunc) {
	r.handle(http.MethodPut, pattern, handlers...)
}

// Patch mendaftarkan handler untuk metode HTTP PATCH.
func (r *Router) Patch(pattern string, handlers ...HandlerFunc) {
	r.handle(http.MethodPatch, pattern, handlers...)
}

// Delete mendaftarkan handler untuk metode HTTP DELETE.
func (r *Router) Delete(pattern string, handlers ...HandlerFunc) {
	r.handle(http.MethodDelete, pattern, handlers...)
}

// Options mendaftarkan handler untuk metode HTTP OPTIONS.
func (r *Router) Options(pattern string, handlers ...HandlerFunc) {
	r.handle(http.MethodOptions, pattern, handlers...)
}

// Head mendaftarkan handler untuk metode HTTP HEAD.
func (r *Router) Head(pattern string, handlers ...HandlerFunc) {
	r.handle(http.MethodHead, pattern, handlers...)
}

// handle merupakan fungsi internal untuk menyatukan middleware dan handler,
// lalu mendaftarkannya ke ServeMux internal.
func (r *Router) handle(method, pattern string, handlers ...HandlerFunc) {
	fullPath := joinPaths(r.prefix, pattern)
	fullPattern := method + " " + fullPath

	all := make([]HandlerFunc, 0, len(r.middlewares)+len(handlers))
	all = append(all, r.middlewares...)
	all = append(all, handlers...)

	r.root.routeCount++

	r.mux.HandleFunc(fullPattern, func(w http.ResponseWriter, req *http.Request) {
		c := &Context{
			ResponseWriter: w,
			Request:        req,
			handlers:       all,
			index:          -1,
		}
		r.dispatch(c)
	})
}

var allMethods = []string{
	http.MethodGet, http.MethodPost, http.MethodPut,
	http.MethodPatch, http.MethodDelete, http.MethodOptions, http.MethodHead,
}

// isMethodNotAllowed mengecek apakah path valid namun method HTTP tidak diizinkan.
func (r *Router) isMethodNotAllowed(req *http.Request) bool {
	originalMethod := req.Method
	defer func() { req.Method = originalMethod }()

	for _, method := range allMethods {
		if method == originalMethod {
			continue
		}
		req.Method = method
		_, pattern := r.root.mux.Handler(req)
		if pattern != "/" && pattern != "" {
			return true
		}
	}
	return false
}

// dispatch mengeksekusi urutan handler dan menangani error jika terjadi kegagalan.
func (r *Router) dispatch(c *Context) {
	if err := c.Next(); err != nil {
		r.handleError(err, c)
	}
}

// handleError memastikan error ditangani oleh error handler yang didaftarkan.
func (r *Router) handleError(err error, c *Context) {
	if c.wroteHeader {
		log.Printf("gosip: error setelah response dikirim: %v", err)
		return
	}
	if herr := r.root.errorHandler(err, c); herr != nil {
		log.Printf("gosip: error handler gagal: %v", herr)
	}
}

// SetNotFoundHandler menetapkan handler yang dipanggil ketika tidak ada route yang
// cocok dengan path request. Menggantikan handler 404 default.
func (r *Router) SetNotFoundHandler(h HandlerFunc) {
	r.root.notFound = h
}

// SetMethodNotAllowedHandler menetapkan handler yang dipanggil ketika path cocok
// tetapi HTTP method tidak terdaftar. Menggantikan handler 405 default.
func (r *Router) SetMethodNotAllowedHandler(h HandlerFunc) {
	r.root.methodNotAllowed = h
}

// SetErrorHandler menetapkan handler yang dipanggil ketika sebuah handler
// mengembalikan error dan respons belum dikirim ke klien.
// Menggantikan error handler default.
func (r *Router) SetErrorHandler(h ErrorHandlerFunc) {
	r.root.errorHandler = h
}

// ServeHTTP memungkinkan Router untuk digunakan sebagai http.Handler yang standar.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

// Listen memulai server HTTP dengan konfigurasi timeout standar dan dukungan graceful shutdown.
func (r *Router) Listen(addr string) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		ReadTimeout:       defaultReadTimeout,
		WriteTimeout:      defaultWriteTimeout,
		IdleTimeout:       defaultIdleTimeout,
	}
	r.printBanner(addr)
	return r.ListenWithServer(srv)
}

// ListenWithServer menjalankan server menggunakan instansiasi *http.Server kustom,
// serta menangani graceful shutdown secara otomatis.
func (r *Router) ListenWithServer(srv *http.Server) error {
	if srv.Handler == nil {
		srv.Handler = r
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case sig := <-quit:
		log.Printf("menerima signal %s, mematikan server...", sig)
		ctx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
		defer cancel()
		return srv.Shutdown(ctx)
	}
}

// joinPaths menggabungkan prefix dan path untuk mencegah slash ganda secara konsisten.
func joinPaths(prefix, path string) string {
	if prefix == "" {
		return path
	}
	if path == "" {
		return prefix
	}
	return strings.TrimRight(prefix, "/") + "/" + strings.TrimLeft(path, "/")
}

// defaultNotFound adalah fallback ketika URL yang diminta tidak ditemukan.
func defaultNotFound(c *Context) error {
	return c.Status(StatusNotFound).String("404 Not Found")
}

// defaultMethodNotAllowed adalah fallback saat path tersedia tapi dengan method yang berbeda.
func defaultMethodNotAllowed(c *Context) error {
	return c.Status(StatusMethodNotAllowed).String("405 Method Not Allowed")
}

// defaultErrorHandler adalah fallback untuk menangani error yang dikembalikan handler atau middleware.
func defaultErrorHandler(err error, c *Context) error {
	log.Printf("gosip: error yang tidak tertangani %v", err)
	return c.Status(StatusInternalServerError).String("500 Internal Server Error")
}
