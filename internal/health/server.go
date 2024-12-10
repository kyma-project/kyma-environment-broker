package health

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gorilla/mux"
)

type Server struct {
	Address string
	Log     *slog.Logger
}

func NewServer(host, port string, log *slog.Logger) *Server {
	return &Server{
		Address: fmt.Sprintf("%s:%s", host, port),
		Log:     log.With("server", "health"),
	}
}

func (srv *Server) ServeAsync() {
	healthRouter := mux.NewRouter()
	healthRouter.HandleFunc("/healthz", livenessHandler())
	go func() {
		err := http.ListenAndServe(srv.Address, healthRouter)
		if err != nil {
			srv.Log.Error(fmt.Sprintf("HTTP Health server ListenAndServe: %v", err))
		}
	}()
}

func livenessHandler() func(w http.ResponseWriter, _ *http.Request) {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		return
	}
}
