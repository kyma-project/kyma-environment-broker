package middleware

import (
	"context"
	"net/http"
	"regexp"

	"github.com/gorilla/mux"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
)

// The providerKey type is no exported to prevent collisions with context keys
// defined in other packages.
type providerKey int

const (
	// requestRegionKey is the context key for the region from the request path.
	requestProviderKey providerKey = iota + 1
)

func AddProviderToContext() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			vars := mux.Vars(req)
			region, found := vars["region"]
			provider := pkg.UnknownProvider
			if found {
				provider = platformProvider(region)
			}

			newCtx := context.WithValue(req.Context(), requestProviderKey, provider)
			next.ServeHTTP(w, req.WithContext(newCtx))
		})
	}
}

// ProviderFromContext returns request provider associated with the context if possible.
func ProviderFromContext(ctx context.Context) (pkg.CloudProvider, bool) {
	provider, ok := ctx.Value(requestProviderKey).(pkg.CloudProvider)
	return provider, ok
}

var platformRegionProviderRE = regexp.MustCompile("[0-9]")

func platformProvider(region string) pkg.CloudProvider {
	if region == "" {
		return pkg.UnknownProvider
	}
	digit := platformRegionProviderRE.FindString(region)
	switch digit {
	case "1":
		return pkg.AWS
	case "2":
		return pkg.Azure
	default:
		return pkg.UnknownProvider
	}
}
