package broker

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/middleware"
	"github.com/pivotal-cf/brokerapi/v11/domain"
	"github.com/pivotal-cf/brokerapi/v11/handlers"
	"github.com/pivotal-cf/brokerapi/v11/middlewares"
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
	var handler http.Handler
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		handler = r.middlewares[i](http.HandlerFunc(handleFunc))
	}
	r.ServeMux.Handle(pattern, handler)
}

type CreateBindingHandler struct {
	handler func(w http.ResponseWriter, req *http.Request)
}

func (h CreateBindingHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	h.handler(rw, r)
}

// copied from github.com/pivotal-cf/brokerapi/api.go
func AttachRoutes(router *Router, serviceBroker domain.ServiceBroker, logger *slog.Logger, createBindingTimeout time.Duration) *Router {
	apiHandler := handlers.NewApiHandler(serviceBroker, logger)
	deprovision := func(w http.ResponseWriter, req *http.Request) {
		req2 := req.WithContext(context.WithValue(req.Context(), "User-Agent", req.Header.Get("User-Agent")))
		apiHandler.Deprovision(w, req2)
	}
	router.HandleFunc("GET /v2/catalog", apiHandler.Catalog)

	router.HandleFunc("GET /v2/service_instances/{instance_id}", apiHandler.GetInstance)
	router.HandleFunc("PUT /v2/service_instances/{instance_id}", apiHandler.Provision)
	router.HandleFunc("DELETE /v2/service_instances/{instance_id}", deprovision)
	router.HandleFunc("GET /v2/service_instances/{instance_id}/last_operation", apiHandler.LastOperation)
	router.HandleFunc("PATCH /v2/service_instances/{instance_id}", apiHandler.Update)

	router.HandleFunc("GET /v2/service_instances/{instance_id}/service_bindings/{binding_id}", apiHandler.GetBinding)
	router.Handle("PUT /v2/service_instances/{instance_id}/service_bindings/{binding_id}", http.TimeoutHandler(CreateBindingHandler{apiHandler.Bind}, createBindingTimeout, fmt.Sprintf("request timeout: time exceeded %s", createBindingTimeout)))
	router.HandleFunc("DELETE /v2/service_instances/{instance_id}/service_bindings/{binding_id}", apiHandler.Unbind)

	router.HandleFunc("GET /v2/service_instances/{instance_id}/service_bindings/{binding_id}/last_operation", apiHandler.LastBindingOperation)

	router.Use(middlewares.AddCorrelationIDToContext)
	apiVersionMiddleware := middlewares.APIVersionMiddleware{Logger: logger}

	router.Use(middlewares.AddOriginatingIdentityToContext)
	router.Use(apiVersionMiddleware.ValidateAPIVersionHdr)
	router.Use(middlewares.AddInfoLocationToContext)

	return router
}
