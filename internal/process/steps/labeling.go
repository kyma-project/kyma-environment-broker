package steps

import (
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/customresources"
)

func SetCommonLabels(labels map[string]string, operation internal.Operation) map[string]string {
	labels[customresources.InstanceIdLabel] = operation.InstanceID
	labels[customresources.RuntimeIdLabel] = operation.RuntimeID
	labels[customresources.PlanIdLabel] = operation.ProvisioningParameters.PlanID
	labels[customresources.PlanNameLabel] = broker.PlanNamesMapping[operation.ProvisioningParameters.PlanID]
	labels[customresources.GlobalAccountIdLabel] = operation.ProvisioningParameters.ErsContext.GlobalAccountID
	labels[customresources.SubaccountIdLabel] = operation.ProvisioningParameters.ErsContext.SubAccountID
	labels[customresources.ShootNameLabel] = operation.ShootName
	if operation.ProvisioningParameters.PlatformRegion != "" {
		labels[customresources.PlatformRegionLabel] = operation.ProvisioningParameters.PlatformRegion
	}
	labels[customresources.KymaNameLabel] = operation.KymaResourceName
	return labels
}

func setLabelsForLM(labels map[string]string, operation internal.Operation) map[string]string {
	labels = SetCommonLabels(labels, operation)
	labels[customresources.RegionLabel] = operation.Region
	labels[customresources.ManagedByLabel] = "lifecycle-manager"
	labels[customresources.CloudProviderLabel] = string(operation.InputCreator.Provider()) //TODO change internal.CloudProvider
	if isKymaResourceInternal(operation) {
		labels[customresources.InternalLabel] = "true"
	}
	return labels
}
