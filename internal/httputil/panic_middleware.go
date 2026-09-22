package httputil

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
)

func PanicRecoveryMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					stack := string(debug.Stack())
					logger.Error(fmt.Sprintf("panic recovered in HTTP handler: %v", rec),
						"path", r.URL.Path,
						"method", r.Method,
						"stack", stack)

					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
