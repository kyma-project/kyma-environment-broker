package config

import "github.com/kyma-project/kyma-environment-broker/internal"

type FakeProviderConfigProvider struct{}

func (p FakeProviderConfigProvider) Provide(cfgKeyName string, cfgDestObj any) error {
	cfg, _ := cfgDestObj.(*internal.ProviderConfig)
	regions := make([]string, 0)
	switch cfgKeyName {
	case "aws":
		regions = append(regions, "eu-central-1", "eu-west-1", "us-east-1")
	case "azure":
		regions = append(regions, "westeurope", "northeurope")
	case "gcp":
		regions = append(regions, "europe-west1", "us-central1")
	case "sapconvergedcloud", "openstack":
		regions = append(regions, "eu-de-1")
	case "alicloud":
		regions = append(regions, "cn-beijing")
	}
	cfg.SeedRegions = regions

	return nil
}
