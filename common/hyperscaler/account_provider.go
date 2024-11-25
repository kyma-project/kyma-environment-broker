package hyperscaler

import (
	"fmt"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
)

//go:generate mockery --name=AccountProvider --output=automock --outpkg=automock --case=underscore
type AccountProvider interface {
	GardenerSecretName(hyperscalerType Type, tenantName string, euAccess bool) (string, error)
	GardenerSharedSecretName(hyperscalerType Type, euAccess bool) (string, error)
	MarkUnusedGardenerSecretBindingAsDirty(hyperscalerType Type, tenantName string, euAccess bool) error
}

type Credentials struct {
	Name            string
	HyperscalerType Type
	CredentialData  map[string][]byte
}

type accountProvider struct {
	gardenerPool       AccountPool
	sharedGardenerPool SharedPool
}

func NewAccountProvider(gardenerPool AccountPool, sharedGardenerPool SharedPool) AccountProvider {
	return &accountProvider{
		gardenerPool:       gardenerPool,
		sharedGardenerPool: sharedGardenerPool,
	}
}

func HypTypeFromCloudProviderWithRegion(cloudProvider pkg.CloudProvider, regionForSapConvergedCloud *string, platformRegion *string) (Type, error) {
	switch cloudProvider {
	case pkg.Azure:
		return Azure(), nil
	case pkg.AWS:
		return AWS(), nil
	case pkg.GCP:
		return GCP(*platformRegion), nil
	case pkg.SapConvergedCloud:
		return SapConvergedCloud(*regionForSapConvergedCloud), nil
	default:
		return Type{}, fmt.Errorf("cannot determine the type of Hyperscaler to use for cloud provider %s", cloudProvider)
	}
}

func (p *accountProvider) GardenerSecretName(hyperscalerType Type, tenantName string, euAccess bool) (string, error) {
	if p.gardenerPool == nil {
		return "", fmt.Errorf("failed to get Gardener Credentials. Gardener Account pool is not configured for tenant %s", tenantName)
	}

	secretBinding, err := p.gardenerPool.CredentialsSecretBinding(hyperscalerType, tenantName, euAccess)
	if err != nil {
		return "", fmt.Errorf("failed to get Gardener Credentials for tenant %s: %w", tenantName, err)
	}

	return secretBinding.GetSecretRefName(), nil
}

func (p *accountProvider) GardenerSharedSecretName(hyperscalerType Type, euAccess bool) (string, error) {
	if p.sharedGardenerPool == nil {
		return "", fmt.Errorf("failed to get shared Secret Binding name. Gardener Shared Account pool is not configured for hyperscaler type %s", hyperscalerType.GetKey())
	}

	secretBinding, err := p.sharedGardenerPool.SharedCredentialsSecretBinding(hyperscalerType, euAccess)
	if err != nil {
		return "", fmt.Errorf("getting shared secret binding: %w", err)
	}

	return secretBinding.GetSecretRefName(), nil
}

func (p *accountProvider) MarkUnusedGardenerSecretBindingAsDirty(hyperscalerType Type, tenantName string, euAccess bool) error {
	if p.gardenerPool == nil {
		return fmt.Errorf("failed to release subscription for tenant %s. Gardener Account pool is not configured", tenantName)
	}

	isInternal, err := p.gardenerPool.IsSecretBindingInternal(hyperscalerType, tenantName, euAccess)
	if err != nil {
		return fmt.Errorf("checking if secret binding is internal: %w", err)
	}
	if isInternal {
		return nil
	}

	isDirty, err := p.gardenerPool.IsSecretBindingDirty(hyperscalerType, tenantName, euAccess)
	if err != nil {
		return fmt.Errorf("checking if secret binding is dirty: %w", err)
	}
	if isDirty {
		return nil
	}

	isUsed, err := p.gardenerPool.IsSecretBindingUsed(hyperscalerType, tenantName, euAccess)
	if err != nil {
		return fmt.Errorf("cannot determine whether %s secret binding is used for tenant: %s: %w", hyperscalerType, tenantName, err)
	}
	if !isUsed {
		return p.gardenerPool.MarkSecretBindingAsDirty(hyperscalerType, tenantName, euAccess)
	}

	return nil
}
