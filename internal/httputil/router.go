package httputil

import (
	"fmt"
	"net/http"

	"github.com/kyma-project/kyma-environment-broker/internal/middleware"
)

type Router struct {
	*http.ServeMux
	subrouters  map[string]*http.ServeMux
	middlewares []middleware.MiddlewareFunc
}

func NewRouter() *Router {
	return &Router{
		ServeMux:    http.NewServeMux(),
		subrouters:  make(map[string]*http.ServeMux),
		middlewares: make([]middleware.MiddlewareFunc, 0),
	}
}

func (r *Router) Use(middlewares ...middleware.MiddlewareFunc) {
	r.middlewares = append(r.middlewares, middlewares...)
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

func (r *Router) NewSubRouter(name string) (*Router, error) {
	if _, exists := r.subrouters[name]; exists {
		return nil, fmt.Errorf("subrouter %s already exists", name)
	}
	subrouter := &Router{
		ServeMux:    http.NewServeMux(),
		subrouters:  make(map[string]*http.ServeMux),
		middlewares: append([]middleware.MiddlewareFunc{}, r.middlewares...),
	}
	r.subrouters[name] = subrouter.ServeMux
	return subrouter, nil
}

func (r *Router) GetSubRouter(name string) (*http.ServeMux, error) {
	subrouter, exists := r.subrouters[name]
	if !exists {
		return nil, fmt.Errorf("subrouter %s not found", name)
	}
	return subrouter, nil
}
