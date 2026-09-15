package gosip

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

const Version = "0.1.0"

const (
	ansiReset  = "\x1b[0m"
	ansiBold   = "\x1b[1m"
	ansiRed    = "\x1b[38;5;9m"
	ansiPurple = "\x1b[38;5;141m"
	ansiCyan   = "\x1b[38;5;80m"
	ansiGreen  = "\x1b[38;5;114m"
)

var logo = [...]string{
	"  ██████      ██████      ████████  ██████████  ████████  ",
	"██      ██  ██      ██  ██              ██      ██      ██",
	"██          ██      ██  ██              ██      ██      ██",
	"██  ██████  ██      ██    ██████        ██      ████████  ",
	"██      ██  ██      ██          ██      ██      ██        ",
	"██      ██  ██      ██          ██      ██      ██        ",
	"  ██████      ██████    ████████    ██████████  ██        ",
}

type HandlerFunc func(*Ctx) error
type ErrorHandlerFunc func(error, *Ctx)

type Gosip struct {
	mux         *http.ServeMux
	srv         atomic.Pointer[http.Server]
	middlewares []HandlerFunc
	prefix      string
	cfg         *config
}

type config struct {
	notFound         HandlerFunc
	methodNotAllowed HandlerFunc
	errorHandler     ErrorHandlerFunc
	routeCount       atomic.Int32
	trustProxy       bool
	proxyHeader      string
}

func New() *Gosip {
	cfg := &config{
		notFound:         defaultNotFound,
		methodNotAllowed: defaultMethodNotAllowed,
		errorHandler:     defaultErrorHandler,
	}
	g := &Gosip{mux: http.NewServeMux(), cfg: cfg}
	g.mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		fallback := cfg.notFound
		if g.isMethodNotAllowed(req) {
			fallback = cfg.methodNotAllowed
		}
		all := make([]HandlerFunc, 0, len(g.middlewares)+1)
		all = append(all, g.middlewares...)
		all = append(all, fallback)
		c := &Ctx{ResponseWriter: w, Request: req, handlers: all, index: -1, cfg: g.cfg}
		g.dispatch(c)
	})

	return g
}

func (g *Gosip) Use(middleware ...HandlerFunc) {
	g.middlewares = append(g.middlewares, middleware...)
}

func (g *Gosip) Group(prefix string) *Gosip {
	return &Gosip{
		mux:         g.mux,
		middlewares: append([]HandlerFunc{}, g.middlewares...),
		prefix:      joinPaths(g.prefix, prefix),
		cfg:         g.cfg,
	}
}

func (g *Gosip) Get(pattern string, handlers ...HandlerFunc) {
	g.handle(MethodGet, pattern, handlers...)
}

func (g *Gosip) Post(pattern string, handlers ...HandlerFunc) {
	g.handle(MethodPost, pattern, handlers...)
}

func (g *Gosip) Put(pattern string, handlers ...HandlerFunc) {
	g.handle(MethodPut, pattern, handlers...)
}

func (g *Gosip) Patch(pattern string, handlers ...HandlerFunc) {
	g.handle(MethodPatch, pattern, handlers...)
}

func (g *Gosip) Delete(pattern string, handlers ...HandlerFunc) {
	g.handle(MethodDelete, pattern, handlers...)
}

func (g *Gosip) Options(pattern string, handlers ...HandlerFunc) {
	g.handle(MethodOptions, pattern, handlers...)
}

func (g *Gosip) Head(pattern string, handlers ...HandlerFunc) {
	g.handle(MethodHead, pattern, handlers...)
}

func (g *Gosip) handle(method, pattern string, handlers ...HandlerFunc) {
	fullPath := joinPaths(g.prefix, pattern)
	fullPattern := method + " " + fullPath

	all := make([]HandlerFunc, 0, len(g.middlewares)+len(handlers))
	all = append(all, g.middlewares...)
	all = append(all, handlers...)

	g.cfg.routeCount.Add(1)

	g.mux.HandleFunc(fullPattern, func(w http.ResponseWriter, req *http.Request) {
		c := &Ctx{ResponseWriter: w, Request: req, handlers: all, index: -1, cfg: g.cfg}
		g.dispatch(c)
	})
}

var allMethods = []string{
	MethodGet, MethodPost, MethodPut, MethodPatch,
	MethodDelete, MethodOptions, MethodHead,
}

func (g *Gosip) isMethodNotAllowed(req *http.Request) bool {
	probe := &http.Request{URL: req.URL, Host: req.Host}
	for _, method := range allMethods {
		if method == req.Method {
			continue
		}
		probe.Method = method
		_, pattern := g.mux.Handler(probe)
		if pattern != "/" && pattern != "" {
			return true
		}
	}
	return false
}

func (g *Gosip) dispatch(c *Ctx) {
	if err := c.Next(); err != nil {
		g.handleError(err, c)
	}
}

func (g *Gosip) handleError(err error, c *Ctx) {
	if c.wroteHeader {
		log.Printf("gosip: error setelah response dikirim: %v", err)
		return
	}
	g.cfg.errorHandler(err, c)
}

func (g *Gosip) SetNotFoundHandler(h HandlerFunc)         { g.cfg.notFound = h }
func (g *Gosip) SetMethodNotAllowedHandler(h HandlerFunc) { g.cfg.methodNotAllowed = h }
func (g *Gosip) SetErrorHandler(h ErrorHandlerFunc)       { g.cfg.errorHandler = h }

func (g *Gosip) SetTrustProxy(trust bool, header ...string) {
	g.cfg.trustProxy = trust
	g.cfg.proxyHeader = "X-Forwarded-For"
	if len(header) > 0 && header[0] != "" {
		g.cfg.proxyHeader = header[0]
	}
}

func (g *Gosip) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	g.mux.ServeHTTP(w, req)
}

func (g *Gosip) Listen(addr string) error {
	return g.ListenWithServer(&http.Server{
		Addr:              addr,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MB
	})
}

func (g *Gosip) ListenWithServer(srv *http.Server) error {
	if srv.Handler == nil {
		srv.Handler = g
	}
	g.printBanner(srv.Addr)
	g.srv.Store(srv)

	err := srv.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (g *Gosip) Shutdown() error {
	return g.ShutdownWithTimeout(10 * time.Second)
}

func (g *Gosip) ShutdownWithTimeout(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return g.ShutdownWithContext(ctx)
}

func (g *Gosip) ShutdownWithContext(ctx context.Context) error {
	srv := g.srv.Load()
	if srv == nil {
		return nil
	}
	return srv.Shutdown(ctx)
}

func (g *Gosip) printBanner(addr string) {
	color := isTerminal() && os.Getenv("NO_COLOR") == ""

	host, port := splitAddr(addr)
	localURL := fmt.Sprintf("http://%s:%s/", prettyHost(host), port)

	purple := func(s string) string { return paint(s, ansiPurple, color) }
	red := func(s string) string { return paint(s, ansiRed, color) }
	boldCyan := func(s string) string { return paint(s, ansiBold+ansiCyan, color) }
	green := func(s string) string { return paint(s, ansiGreen, color) }
	arrow := purple("➜")

	var b strings.Builder
	b.WriteString("\n")
	for _, line := range logo {
		b.WriteString("  " + purple(line) + "\n")
	}
	b.WriteString("\n")
	b.WriteString("  " + red(" v"+Version) + "\n\n")
	fmt.Fprintf(&b, "  %s  %s %s\n", arrow, "Local:  ", boldCyan(localURL))
	fmt.Fprintf(&b, "  %s  %s %s\n", arrow, "Network:", "bound on "+host+":"+port)
	fmt.Fprintf(&b, "  %s  %s %s\n", arrow, "Routes: ", green(strconv.Itoa(int(g.cfg.routeCount.Load()))))
	fmt.Fprintf(&b, "  %s  %s %s\n", arrow, "PID:    ", red(strconv.Itoa(os.Getpid())))
	b.WriteString("\n")

	fmt.Print(b.String())
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

func defaultNotFound(c *Ctx) error {
	return c.Status(StatusNotFound).String("404 Not Found")
}

func defaultMethodNotAllowed(c *Ctx) error {
	return c.Status(StatusMethodNotAllowed).String("405 Method Not Allowed")
}

func defaultErrorHandler(err error, c *Ctx) {
	log.Printf("gosip: error yang tidak tertangani: %v", err)
	_ = c.Status(StatusInternalServerError).String("500 Internal Server Error")
}

func isTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func splitAddr(addr string) (host, port string) {
	if addr == "" {
		return "0.0.0.0", "8000"
	}
	i := strings.LastIndex(addr, ":")
	if i < 0 {
		return "0.0.0.0", addr
	}
	host = addr[:i]
	port = addr[i+1:]
	if host == "" {
		host = "0.0.0.0"
	}
	return
}

func prettyHost(host string) string {
	switch host {
	case "", "0.0.0.0", "::", "[::]":
		return "127.0.0.1"
	}
	return host
}

func paint(s, colorCode string, enabled bool) string {
	if !enabled {
		return s
	}
	return colorCode + s + ansiReset
}
