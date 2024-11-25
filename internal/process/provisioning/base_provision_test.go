package provisioning

import (
	"testing"

	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newInputCreator() *simpleInputCreator {
	return &simpleInputCreator{
		labels: make(map[string]string),
	}
}

type simpleInputCreator struct {
	labels            map[string]string
	shootName         *string
	provider          pkg.CloudProvider
	shootDomain       string
	shootDnsProviders gardener.DNSProvidersData
	config            *internal.ConfigForPlan
}

func (c *simpleInputCreator) Configuration() *internal.ConfigForPlan {
	return c.config
}

func (c *simpleInputCreator) Provider() pkg.CloudProvider {
	if c.provider != "" {
		return c.provider
	}
	return pkg.Azure
}

func (c *simpleInputCreator) SetLabel(key, val string) internal.ProvisionerInputCreator {
	c.labels[key] = val
	return c
}

func (c *simpleInputCreator) SetShootName(name string) internal.ProvisionerInputCreator {
	c.shootName = &name
	return c
}

func (c *simpleInputCreator) SetShootDomain(name string) internal.ProvisionerInputCreator {
	c.shootDomain = name
	return c
}

func (c *simpleInputCreator) SetShootDNSProviders(providers gardener.DNSProvidersData) internal.ProvisionerInputCreator {
	c.shootDnsProviders = providers
	return c
}

func (c *simpleInputCreator) CreateProvisionRuntimeInput() (gqlschema.ProvisionRuntimeInput, error) {
	return gqlschema.ProvisionRuntimeInput{}, nil
}

func (c *simpleInputCreator) CreateUpgradeRuntimeInput() (gqlschema.UpgradeRuntimeInput, error) {
	return gqlschema.UpgradeRuntimeInput{}, nil
}

func (c *simpleInputCreator) CreateUpgradeShootInput() (gqlschema.UpgradeShootInput, error) {
	return gqlschema.UpgradeShootInput{}, nil
}

func (c *simpleInputCreator) SetProvisioningParameters(params internal.ProvisioningParameters) internal.ProvisionerInputCreator {
	return c
}

func (c *simpleInputCreator) SetKubeconfig(_ string) internal.ProvisionerInputCreator {
	return c
}

func (c *simpleInputCreator) SetRuntimeID(_ string) internal.ProvisionerInputCreator {
	return c
}

func (c *simpleInputCreator) SetInstanceID(_ string) internal.ProvisionerInputCreator {
	return c
}

func (c *simpleInputCreator) SetClusterName(_ string) internal.ProvisionerInputCreator {
	return c
}

func (c *simpleInputCreator) SetOIDCLastValues(oidcConfig gqlschema.OIDCConfigInput) internal.ProvisionerInputCreator {
	return c
}

func (c *simpleInputCreator) AssertLabel(t *testing.T, key, expectedValue string) {
	value, found := c.labels[key]
	require.True(t, found)
	assert.Equal(t, expectedValue, value)
}

func (c *simpleInputCreator) CreateProvisionClusterInput() (gqlschema.ProvisionRuntimeInput, error) {
	return gqlschema.ProvisionRuntimeInput{}, nil
}
