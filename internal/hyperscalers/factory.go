package hyperscalers

import (
	"context"
	"fmt"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal/hyperscalers/aws"
	"github.com/kyma-project/kyma-environment-broker/internal/provider/configuration"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type hyperscalerFactory struct {
	providerSpec *configuration.ProviderSpec
}

func NewFactory(providerSpec *configuration.ProviderSpec) Factory {
	return &hyperscalerFactory{providerSpec: providerSpec}
}

func (f *hyperscalerFactory) NewFromSecret(ctx context.Context, provider pkg.CloudProvider, secret *unstructured.Unstructured, region string) (ProviderClient, error) {
	switch provider {
	case pkg.AWS:
		return aws.NewClientFromSecret(ctx, f.providerSpec, secret, region)
	default:
		return nil, fmt.Errorf("zone discovery not supported for provider %s", provider)
	}
}
