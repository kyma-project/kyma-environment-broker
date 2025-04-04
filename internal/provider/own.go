package provider

import "github.com/kyma-project/kyma-environment-broker/internal"

type OwnClusterinputProvider struct {
}

func (o *OwnClusterinputProvider) Provide() internal.ProviderValues {
	return internal.ProviderValues{
		DefaultAutoScalerMax: 0,
		DefaultAutoScalerMin: 0,
		ZonesCount:           0,
		Zones:                nil,
		ProviderType:         OwnProviderType,
		DefaultMachineType:   "",
		Region:               "",
		Purpose:              "",
		VolumeSizeGb:         0,
		DiskType:             "",
		FailureTolerance:     nil,
	}
}
