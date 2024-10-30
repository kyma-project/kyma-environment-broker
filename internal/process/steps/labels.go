package steps

import (
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
)

const (
	GlobalAccountIdLabel = "kyma-project.io/global-account-id"
	InstanceIdLabel      = "kyma-project.io/instance-id"
	RuntimeIdLabel       = "kyma-project.io/runtime-id"
	PlanIdLabel          = "kyma-project.io/broker-plan-id"
	PlanNameLabel        = "kyma-project.io/broker-plan-name"
	SubaccountIdLabel    = "kyma-project.io/subaccount-id"
	ShootNameLabel       = "kyma-project.io/shoot-name"
	RegionLabel          = "kyma-project.io/region"
	PlatformRegionLabel  = "kyma-project.io/platform-region"
	CloudProviderLabel   = "kyma-project.io/provider"
	KymaNameLabel        = "operator.kyma-project.io/kyma-name"
	ManagedByLabel       = "operator.kyma-project.io/managed-by"
	InternalLabel        = "operator.kyma-project.io/internal"
)

func setCommonLabels(labels map[string]string, operation internal.Operation) map[string]string {
	labels[InstanceIdLabel] = operation.InstanceID
	labels[RuntimeIdLabel] = operation.RuntimeID
	labels[PlanIdLabel] = operation.ProvisioningParameters.PlanID
	labels[PlanNameLabel] = broker.PlanNamesMapping[operation.ProvisioningParameters.PlanID]
	labels[GlobalAccountIdLabel] = operation.GlobalAccountID
	labels[SubaccountIdLabel] = operation.SubAccountID
	labels[ShootNameLabel] = operation.ShootName
	if operation.ProvisioningParameters.PlatformRegion != "" {
		labels[PlatformRegionLabel] = operation.ProvisioningParameters.PlatformRegion
	}
	labels[KymaNameLabel] = operation.KymaResourceName
	return labels
}

func setLabelsForLM(labels map[string]string, operation internal.Operation) map[string]string {
	labels = setCommonLabels(labels, operation)
	labels[RegionLabel] = operation.Region
	labels[ManagedByLabel] = "lifecycle-manager"
	labels[CloudProviderLabel] = string(operation.InputCreator.Provider()) //TODO change internal.CloudProvider
	if isKymaResourceInternal(operation) {
		labels[InternalLabel] = "true"
	}
	return labels
}

func setLabelsForRuntime(labels map[string]string, operation internal.Operation, region string, cloudProvider string) map[string]string {
	labels = setCommonLabels(labels, operation)
	labels[RegionLabel] = region
	labels[CloudProviderLabel] = cloudProvider
	return labels
}
