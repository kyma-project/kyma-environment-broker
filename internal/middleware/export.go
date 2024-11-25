package middleware

import (
	"context"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
)

func AddRegionToCtx(ctx context.Context, region string) context.Context {
	return context.WithValue(ctx, requestRegionKey, region)
}

func AddProviderToCtx(ctx context.Context, provider pkg.CloudProvider) context.Context {
	return context.WithValue(ctx, requestProviderKey, provider)
}
