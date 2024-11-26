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
func AttachRoutes(router *httputil.Router, serviceBroker domain.ServiceBroker, logger *slog.Logger, createBindingTimeout time.Duration) *httputil.Router {
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
