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

type HandlerFunc func(*Context)
type ErrorHandlerFunc func([]error, *Context)

type Router struct {
	mux              *http.ServeMux
	middlewares      []HandlerFunc
	prefix           string
	root             *Router
	notFound         HandlerFunc
	methodNotAllowed HandlerFunc
	errorHandler     ErrorHandlerFunc
	methods          map[string]struct{}
}

func New() *Router {
	r := &Router{
		mux:     http.NewServeMux(),
		methods: make(map[string]struct{}),
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
		c.Next()
		r.handleErrors(c)
	})

	return r
}

func (r *Router) Use(middleware ...HandlerFunc) {
	r.middlewares = append(r.middlewares, middleware...)
}

func (r *Router) Group(prefix string) *Router {
	return &Router{
		mux:              r.mux,
		middlewares:      append([]HandlerFunc{}, r.middlewares...),
		prefix:           joinPaths(r.prefix, prefix),
		root:             r.root,
		notFound:         r.root.notFound,
		methodNotAllowed: r.root.methodNotAllowed,
		errorHandler:     r.root.errorHandler,
		methods:          r.root.methods,
	}
}

func (r *Router) Get(pattern string, handlers ...HandlerFunc) {
	r.handle(http.MethodGet, pattern, handlers...)
}

func (r *Router) Post(pattern string, handlers ...HandlerFunc) {
	r.handle(http.MethodPost, pattern, handlers...)
}

func (r *Router) Put(pattern string, handlers ...HandlerFunc) {
	r.handle(http.MethodPut, pattern, handlers...)
}

func (r *Router) Patch(pattern string, handlers ...HandlerFunc) {
	r.handle(http.MethodPatch, pattern, handlers...)
}

func (r *Router) Delete(pattern string, handlers ...HandlerFunc) {
	r.handle(http.MethodDelete, pattern, handlers...)
}

func (r *Router) Options(pattern string, handlers ...HandlerFunc) {
	r.handle(http.MethodOptions, pattern, handlers...)
}

func (r *Router) Head(pattern string, handlers ...HandlerFunc) {
	r.handle(http.MethodHead, pattern, handlers...)
}

func (r *Router) handle(method, pattern string, handlers ...HandlerFunc) {
	fullPath := joinPaths(r.prefix, pattern)
	fullPattern := method + " " + fullPath

	all := make([]HandlerFunc, 0, len(r.middlewares)+len(handlers))
	all = append(all, r.middlewares...)
	all = append(all, handlers...)

	r.root.methods[method] = struct{}{}

	r.mux.HandleFunc(fullPattern, func(w http.ResponseWriter, req *http.Request) {
		c := &Context{
			ResponseWriter: w,
			Request:        req,
			handlers:       all,
			index:          -1,
		}
		c.Next()
		r.handleErrors(c)
	})
}

func (r *Router) isMethodNotAllowed(req *http.Request) bool {
	if len(r.root.methods) == 0 {
		return false
	}

	originalMethod := req.Method
	defer func() {
		req.Method = originalMethod
	}()

	for method := range r.root.methods {
		if method == originalMethod {
			continue
		}
		req.Method = method
		_, pattern := r.root.mux.Handler(req)
		if pattern != "/" {
			return true
		}
	}

	return false
}

func (r *Router) handleErrors(c *Context) {
	if len(c.errors) == 0 {
		return
	}
	if !c.wroteHeader {
		r.root.errorHandler(c.errors, c)
		return
	}
	for _, err := range c.errors {
		log.Printf("gosip: error setelah response dikirim: %v", err)
	}
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

func (r *Router) Listen(addr string) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		ReadTimeout:       defaultReadTimeout,
		WriteTimeout:      defaultWriteTimeout,
		IdleTimeout:       defaultIdleTimeout,
	}
	return r.ListenWithServer(srv)
}

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

func joinPaths(prefix, path string) string {
	if prefix == "" {
		return path
	}
	if path == "" {
		return prefix
	}
	return strings.TrimRight(prefix, "/") + "/" + strings.TrimLeft(path, "/")
}

func defaultNotFound(c *Context) {
	c.Status(StatusNotFound).String("404 Not Found")
}

func defaultMethodNotAllowed(c *Context) {
	c.Status(StatusMethodNotAllowed).String("405 Method Not Allowed")
}

func defaultErrorHandler(errs []error, c *Context) {
	c.Status(StatusInternalServerError).String(errs[0].Error())
}
