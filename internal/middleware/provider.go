package middleware

import (
	"context"
	"net/http"
	"regexp"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
)

// The providerKey type is no exported to prevent collisions with context keys
// defined in other packages.
type providerKey int

const (
	// requestRegionKey is the context key for the region from the request path.
	requestProviderKey providerKey = iota + 1
)

func AddProviderToContext() MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			region := req.PathValue("region")
			provider := pkg.UnknownProvider
			if len(region) > 0 {
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
	case "0":
		return pkg.SapConvergedCloud
	case "1":
		return pkg.AWS
	case "2":
		return pkg.Azure
	case "3":
		return pkg.GCP
	case "4":
		return pkg.Alicloud
	default:
		return pkg.UnknownProvider
	}
}
