package health

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"net/http"

	log "github.com/sirupsen/logrus"
)

type Server struct {
	Address string
	Log     log.FieldLogger
}

func NewServer(host, port string, log *log.Logger) *Server {
	return &Server{
		Address: fmt.Sprintf("%s:%s", host, port),
		Log:     log.WithField("server", "health"),
	}
}

func (srv *Server) ServeAsync() {
	healthRouter := chi.NewRouter()
	healthRouter.HandleFunc("/healthz", livenessHandler())
	go func() {
		err := http.ListenAndServe(srv.Address, healthRouter)
		if err != nil {
			srv.Log.Errorf("HTTP Health server ListenAndServe: %v", err)
		}
	}()
}

func livenessHandler() func(w http.ResponseWriter, _ *http.Request) {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		return
	}
}
