package avs

import "github.com/kyma-project/kyma-environment-broker/internal"

type ModelConfigurator interface {
	ProvideSuffix() string
	ProvideTesterAccessId(pp internal.ProvisioningParameters) int64
	ProvideGroupId(pp internal.ProvisioningParameters) int64
	ProvideParentId(pp internal.ProvisioningParameters) int64
	ProvideTags(o internal.Operation) []*Tag
	ProvideNewOrDefaultServiceName(defaultServiceName string) string
	ProvideCheckType() string
}
