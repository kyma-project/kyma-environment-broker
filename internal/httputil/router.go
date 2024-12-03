package httputil

import (
	"fmt"
	"net/http"

	"github.com/kyma-project/kyma-environment-broker/internal/middleware"
)

type Router struct {
	*http.ServeMux
	subrouters  []*http.ServeMux
	middlewares []middleware.MiddlewareFunc
}

func NewRouter() *Router {
	return &Router{
		ServeMux:    http.NewServeMux(),
		subrouters:  make([]*http.ServeMux, 0),
		middlewares: make([]middleware.MiddlewareFunc, 0),
	}
}

func (r *Router) Use(middlewares ...middleware.MiddlewareFunc) {
	for _, m := range middlewares {
		r.middlewares = append(r.middlewares, m)
	}
}

func (r *Router) PathPrefix(prefix string) {
	subrouter := http.NewServeMux()
	pattern := fmt.Sprintf("/%s/", prefix)
	subrouter.Handle(pattern, r)
	r.subrouters = append(r.subrouters, subrouter)
}

func (r *Router) Handle(pattern string, handler http.Handler) {
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		handler = r.middlewares[i](handler)
	}
	r.ServeMux.Handle(pattern, handler)
}

func (r *Router) HandleFunc(pattern string, handleFunc func(http.ResponseWriter, *http.Request)) {
	var handler http.Handler = http.HandlerFunc(handleFunc)
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		handler = r.middlewares[i](handler)
	}
	r.ServeMux.Handle(pattern, handler)
}
