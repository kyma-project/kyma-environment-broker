package provider

import "github.com/kyma-project/kyma-environment-broker/internal"

type OwnClusterinputProvider struct {
}

func (o *OwnClusterinputProvider) Provide() internal.ProviderValues {
	return internal.ProviderValues{
		ProviderType: OwnProviderType,
	}
}
