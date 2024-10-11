package broker

import (
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/pivotal-cf/brokerapi/v11/middlewares"
	"log/slog"
	"net/http"

	"github.com/pivotal-cf/brokerapi/v11/domain"
	"github.com/pivotal-cf/brokerapi/v11/handlers"
)

// copied from github.com/pivotal-cf/brokerapi/api.go
func AttachRoutes(router chi.Router, serviceBroker domain.ServiceBroker, logger *slog.Logger) chi.Router {
	router.Use(middlewares.AddCorrelationIDToContext)
	apiVersionMiddleware := middlewares.APIVersionMiddleware{Logger: logger}

	router.Use(middlewares.AddOriginatingIdentityToContext)
	router.Use(apiVersionMiddleware.ValidateAPIVersionHdr)
	router.Use(middlewares.AddInfoLocationToContext)

	apiHandler := handlers.NewApiHandler(serviceBroker, logger)
	deprovision := func(w http.ResponseWriter, req *http.Request) {
		req2 := req.WithContext(context.WithValue(req.Context(), "User-Agent", req.Header.Get("User-Agent")))
		apiHandler.Deprovision(w, req2)
	}
	router.Get("/v2/catalog", apiHandler.Catalog)

	router.Get("/v2/service_instances/{instance_id}", apiHandler.GetInstance)
	router.Put("/v2/service_instances/{instance_id}", apiHandler.Provision)
	router.Delete("/v2/service_instances/{instance_id}", deprovision)
	router.Get("/v2/service_instances/{instance_id}/last_operation", apiHandler.LastOperation)
	router.Patch("/v2/service_instances/{instance_id}", apiHandler.Update)

	router.Get("/v2/service_instances/{instance_id}/service_bindings/{binding_id}", apiHandler.GetBinding)
	router.Put("/v2/service_instances/{instance_id}/service_bindings/{binding_id}", apiHandler.Bind)
	router.Delete("/v2/service_instances/{instance_id}/service_bindings/{binding_id}", apiHandler.Unbind)

	router.Get("/v2/service_instances/{instance_id}/service_bindings/{binding_id}/last_operation", apiHandler.LastBindingOperation)

	return router
}
