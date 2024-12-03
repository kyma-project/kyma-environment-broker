package broker

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/httputil"
	"github.com/pivotal-cf/brokerapi/v11/domain"
	"github.com/pivotal-cf/brokerapi/v11/handlers"
	"github.com/pivotal-cf/brokerapi/v11/middlewares"
)

type CreateBindingHandler struct {
	handler func(w http.ResponseWriter, req *http.Request)
}

func (h CreateBindingHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	h.handler(rw, r)
}

// copied from github.com/pivotal-cf/brokerapi/api.go
func AttachRoutes(router *httputil.Router, serviceBroker domain.ServiceBroker, logger *slog.Logger, createBindingTimeout time.Duration, prefixes []string) *httputil.Router {
	apiHandler := handlers.NewApiHandler(serviceBroker, logger)

	deprovision := func(w http.ResponseWriter, req *http.Request) {
		req2 := req.WithContext(context.WithValue(req.Context(), "User-Agent", req.Header.Get("User-Agent")))
		apiHandler.Deprovision(w, req2)
	}

	prefixes = append(prefixes, "")
	for _, prefix := range prefixes {
		registerRoutesAndHandlers(router, &apiHandler, deprovision, createBindingTimeout, prefix)
	}

	router.Use(middlewares.AddCorrelationIDToContext)
	apiVersionMiddleware := middlewares.APIVersionMiddleware{Logger: logger}

	router.Use(middlewares.AddOriginatingIdentityToContext)
	router.Use(apiVersionMiddleware.ValidateAPIVersionHdr)
	router.Use(middlewares.AddInfoLocationToContext)

	return router
}

func registerRoutesAndHandlers(router *httputil.Router, apiHandler *handlers.APIHandler, deprovisionFunc func(w http.ResponseWriter, req *http.Request), createBindingTimeout time.Duration, pathPrefix string) {
	router.HandleFunc(buildPathPattern(http.MethodGet, pathPrefix, "/v2/catalog"), apiHandler.Catalog)

	router.HandleFunc(buildPathPattern(http.MethodGet, pathPrefix, "/v2/service_instances/{instance_id}"), apiHandler.GetInstance)
	router.HandleFunc(buildPathPattern(http.MethodPut, pathPrefix, "/v2/service_instances/{instance_id}"), apiHandler.Provision)
	router.HandleFunc(buildPathPattern(http.MethodDelete, pathPrefix, "/v2/service_instances/{instance_id}"), deprovisionFunc)
	router.HandleFunc(buildPathPattern(http.MethodGet, pathPrefix, "/v2/service_instances/{instance_id}/last_operation"), apiHandler.LastOperation)
	router.HandleFunc(buildPathPattern(http.MethodPatch, pathPrefix, "/v2/service_instances/{instance_id}"), apiHandler.Update)

	router.HandleFunc(buildPathPattern(http.MethodGet, pathPrefix, "/v2/service_instances/{instance_id}/service_bindings/{binding_id}"), apiHandler.GetBinding)
	router.Handle(buildPathPattern(http.MethodPut, pathPrefix, "/v2/service_instances/{instance_id}/service_bindings/{binding_id}"), http.TimeoutHandler(CreateBindingHandler{apiHandler.Bind}, createBindingTimeout, fmt.Sprintf("request timeout: time exceeded %s", createBindingTimeout)))
	router.HandleFunc(buildPathPattern(http.MethodDelete, pathPrefix, "/v2/service_instances/{instance_id}/service_bindings/{binding_id}"), apiHandler.Unbind)

	router.HandleFunc(buildPathPattern(http.MethodGet, pathPrefix, "/v2/service_instances/{instance_id}/service_bindings/{binding_id}/last_operation"), apiHandler.LastBindingOperation)
}

func buildPathPattern(httpMethod, prefix, path string) string {
	return fmt.Sprintf("%s %s%s", httpMethod, prefix, path)
}
