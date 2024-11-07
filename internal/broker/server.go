package broker

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/gorilla/mux"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/pivotal-cf/brokerapi/v8/handlers"
	"github.com/pivotal-cf/brokerapi/v8/middlewares"
)

type CreateBindingHandler struct {
	handler func(w http.ResponseWriter, req *http.Request)
}

func (h CreateBindingHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	h.handler(rw, r)
}

// copied from github.com/pivotal-cf/brokerapi/api.go
func AttachRoutes(router *mux.Router, serviceBroker domain.ServiceBroker, logger lager.Logger, createBindingTimeout time.Duration) *mux.Router {
	apiHandler := handlers.NewApiHandler(serviceBroker, logger)
	deprovision := func(w http.ResponseWriter, req *http.Request) {
		req2 := req.WithContext(context.WithValue(req.Context(), "User-Agent", req.Header.Get("User-Agent")))
		apiHandler.Deprovision(w, req2)
	}
	router.HandleFunc("/v2/catalog", apiHandler.Catalog).Methods("GET")

	router.HandleFunc("/v2/service_instances/{instance_id}", apiHandler.GetInstance).Methods("GET")
	router.HandleFunc("/v2/service_instances/{instance_id}", apiHandler.Provision).Methods("PUT")
	router.HandleFunc("/v2/service_instances/{instance_id}", deprovision).Methods("DELETE")
	router.HandleFunc("/v2/service_instances/{instance_id}/last_operation", apiHandler.LastOperation).Methods("GET")
	router.HandleFunc("/v2/service_instances/{instance_id}", apiHandler.Update).Methods("PATCH")

	router.HandleFunc("/v2/service_instances/{instance_id}/service_bindings/{binding_id}", apiHandler.GetBinding).Methods("GET")
	router.Handle("/v2/service_instances/{instance_id}/service_bindings/{binding_id}", http.TimeoutHandler(CreateBindingHandler{apiHandler.Bind}, createBindingTimeout, fmt.Sprintf("request timeout: time exceeded %s", createBindingTimeout))).Methods("PUT")
	router.HandleFunc("/v2/service_instances/{instance_id}/service_bindings/{binding_id}", apiHandler.Unbind).Methods("DELETE")

	router.HandleFunc("/v2/service_instances/{instance_id}/service_bindings/{binding_id}/last_operation", apiHandler.LastBindingOperation).Methods("GET")

	router.Use(middlewares.AddCorrelationIDToContext)
	apiVersionMiddleware := middlewares.APIVersionMiddleware{LoggerFactory: logger}

	router.Use(middlewares.AddOriginatingIdentityToContext)
	router.Use(apiVersionMiddleware.ValidateAPIVersionHdr)
	router.Use(middlewares.AddInfoLocationToContext)

	return router
}
