package middleware

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

func AddRequestLogging(logs logrus.FieldLogger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if strings.Contains(req.URL.Path, "service_bindings") {
				logs.WithFields(logrus.Fields{
					"method":      req.Method,
					"url":         req.URL.String(),
					"query":       req.URL.Query(),
					"headers":     req.Header,
					"remote_addr": req.RemoteAddr,
					"host":        req.Host,
					"user_agent":  req.UserAgent(),
				}).Info("Request details")

				if req.Body == nil {
					logs.Info("Request body is nil")
				} else {
					bodyBytes, err := io.ReadAll(req.Body)
					if err != nil {
						logs.Error("Failed to read request body:", err)
					} else {
						logs.WithField("body", string(bodyBytes)).Info("Request body")

						req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
					}
				}
			}

			next.ServeHTTP(w, req)
		})
	}
}
