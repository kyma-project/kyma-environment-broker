package hyperscalers

import (
	"context"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type ProviderClient interface {
	AvailableZones(ctx context.Context, machineType string) ([]string, error)
	AvailableZonesCount(ctx context.Context, machineType string) (int, error)
}

type Factory interface {
	NewFromSecret(ctx context.Context, provider pkg.CloudProvider, secret *unstructured.Unstructured, region string) (ProviderClient, error)
}
