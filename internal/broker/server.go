package broker

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/mux"
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

func ServeMuxCompMiddleware(f func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		route, err := mux.CurrentRoute(r).GetPathTemplate()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			if _, err := w.Write([]byte(fmt.Sprintf("error getting route template: %s", err))); err != nil {
				slog.Error("error writing response", "error", err)
			}
			return
		}
		pattern := fmt.Sprintf(r.Method + " " + route)
		stdLibMux := http.NewServeMux()
		stdLibMux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) { return })
		stdLibMux.ServeHTTP(w, r)
		f(w, r)
	}
}

// copied from github.com/pivotal-cf/brokerapi/api.go
func AttachRoutes(router *mux.Router, serviceBroker domain.ServiceBroker, logger *slog.Logger, createBindingTimeout time.Duration) *mux.Router {
	apiHandler := handlers.NewApiHandler(serviceBroker, logger)
	deprovision := func(w http.ResponseWriter, req *http.Request) {
		req2 := req.WithContext(context.WithValue(req.Context(), "User-Agent", req.Header.Get("User-Agent")))
		ServeMuxCompMiddleware(apiHandler.Deprovision)(w, req2)
	}
	router.HandleFunc("/v2/catalog", apiHandler.Catalog).Methods("GET")

	router.HandleFunc("/v2/service_instances/{instance_id}", ServeMuxCompMiddleware(apiHandler.GetInstance)).Methods("GET")
	router.HandleFunc("/v2/service_instances/{instance_id}", ServeMuxCompMiddleware(apiHandler.Provision)).Methods("PUT")
	router.HandleFunc("/v2/service_instances/{instance_id}", deprovision).Methods("DELETE")
	router.HandleFunc("/v2/service_instances/{instance_id}/last_operation", ServeMuxCompMiddleware(apiHandler.LastOperation)).Methods("GET")
	router.HandleFunc("/v2/service_instances/{instance_id}", ServeMuxCompMiddleware(apiHandler.Update)).Methods("PATCH")

	router.HandleFunc("/v2/service_instances/{instance_id}/service_bindings/{binding_id}", ServeMuxCompMiddleware(apiHandler.GetBinding)).Methods("GET")
	router.Handle("/v2/service_instances/{instance_id}/service_bindings/{binding_id}", http.TimeoutHandler(CreateBindingHandler{ServeMuxCompMiddleware(apiHandler.Bind)}, createBindingTimeout, fmt.Sprintf("request timeout: time exceeded %s", createBindingTimeout))).Methods("PUT")
	router.HandleFunc("/v2/service_instances/{instance_id}/service_bindings/{binding_id}", ServeMuxCompMiddleware(apiHandler.Unbind)).Methods("DELETE")

	router.HandleFunc("/v2/service_instances/{instance_id}/service_bindings/{binding_id}/last_operation", ServeMuxCompMiddleware(apiHandler.LastBindingOperation)).Methods("GET")

	router.Use(middlewares.AddCorrelationIDToContext)
	apiVersionMiddleware := middlewares.APIVersionMiddleware{Logger: logger}

	router.Use(middlewares.AddOriginatingIdentityToContext)
	router.Use(apiVersionMiddleware.ValidateAPIVersionHdr)
	router.Use(middlewares.AddInfoLocationToContext)

	return router
}
