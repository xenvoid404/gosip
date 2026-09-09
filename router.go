package gosip

import (
	"net/http"
)

type HandlerFunc func(*Context)

type Router struct {
	mux         *http.ServeMux
	middlewares []HandlerFunc
}

func New() *Router {
	return &Router{mux: http.NewServeMux()}
}
