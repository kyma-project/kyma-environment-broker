package broker

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/httputil"
	"github.com/kyma-project/kyma-environment-broker/internal/middleware"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/pivotal-cf/brokerapi/v12/handlers"
	"github.com/pivotal-cf/brokerapi/v12/middlewares"
)

type contextKey string

const userAgentKey contextKey = "User-Agent"

type CreateBindingHandler struct {
	handler func(w http.ResponseWriter, req *http.Request)
}

func (h CreateBindingHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	h.handler(rw, r)
}

// AttachRoutes attaches all API routes for the service broker to the given router.
// It sets up middlewares, API version validation, and route handlers for each prefix.
//
// copied from github.com/pivotal-cf/brokerapi/api.go
func AttachRoutes(router *httputil.Router, serviceBroker domain.ServiceBroker, logger *slog.Logger, createBindingTimeout time.Duration, defaultRequestRegion string, prefixes []string) *httputil.Router {
	apiHandler := handlers.NewApiHandler(serviceBroker, logger)
	deprovision := func(w http.ResponseWriter, req *http.Request) {
		req2 := req.WithContext(context.WithValue(req.Context(), userAgentKey, req.Header.Get("User-Agent")))
		apiHandler.Deprovision(w, req2)
	}
	router.Use(middlewares.AddCorrelationIDToContext)
	apiVersionMiddleware := middlewares.APIVersionMiddleware{Logger: logger}

	router.Use(middlewares.AddOriginatingIdentityToContext)
	router.Use(apiVersionMiddleware.ValidateAPIVersionHdr)
	router.Use(middlewares.AddInfoLocationToContext)
	router.Use(middleware.AddRegionToContext(defaultRequestRegion))
	router.Use(middleware.AddProviderToContext())

	for _, prefix := range prefixes {
		registerRoutesAndHandlers(router, &apiHandler, deprovision, createBindingTimeout, prefix)
	}

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
