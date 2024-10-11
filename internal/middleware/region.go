package middleware

import (
	"context"
	"github.com/go-chi/chi/v5"
	"net/http"
)

// The key type is no exported to prevent collisions with context keys
// defined in other packages.
type key int

const (
	// requestRegionKey is the context key for the region from the request path.
	requestRegionKey key = iota + 1
)

func AddRegionToContext(defaultRegion string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			region := chi.URLParam(req, "region")
			if region == "" {
				region = defaultRegion
			}

			newCtx := context.WithValue(req.Context(), requestRegionKey, region)
			next.ServeHTTP(w, req.WithContext(newCtx))
		})
	}
}

// RegionFromContext returns request region associated with the context if possible.
func RegionFromContext(ctx context.Context) (string, bool) {
	region, ok := ctx.Value(requestRegionKey).(string)
	return region, ok
}
