package archive

import (
	"fmt"
	"strings"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"golang.org/x/exp/slices"
)

func NewInstanceArchivedFromOperationsAndInstance(instance internal.Instance, operations []internal.Operation) (internal.InstanceArchived, error) {
	result, err := NewInstanceArchivedFromOperations(operations)
	if err != nil {
		return result, err
	}
	result.Provider = string(instance.Provider)
	result.SubscriptionGlobalAccountID = instance.SubscriptionGlobalAccountID
	result.Region = instance.ProviderRegion

	return result, nil
}

func NewInstanceArchivedFromOperations(operations []internal.Operation) (internal.InstanceArchived, error) {
	result := internal.InstanceArchived{}
	cmp := func(a, b internal.Operation) int {
		return a.CreatedAt.Compare(b.CreatedAt)
	}
	if len(operations) == 0 {
		return result, fmt.Errorf("operations cannot be empty")
	}

	// sort operations - the older one must be the first one
	slices.SortFunc(operations, cmp)

	if operations[0].Type != internal.OperationTypeProvision {
		return result, fmt.Errorf("first operation must be Provision, but was %s", operations[0].Type)
	}

	// the first operation is Provisioning
	provisioningOperation := operations[0]
	result.ProvisioningStartedAt = provisioningOperation.CreatedAt
	result.ProvisioningFinishedAt = provisioningOperation.UpdatedAt
	result.ProvisioningState = provisioningOperation.State
	result.PlanID = provisioningOperation.ProvisioningParameters.PlanID
	result.PlanName = broker.AvailablePlans.GetPlanNameOrEmpty(broker.PlanIDType(result.PlanID))
	result.InstanceID = provisioningOperation.InstanceID
	result.GlobalAccountID = provisioningOperation.ProvisioningParameters.ErsContext.GlobalAccountID
	result.SubaccountID = provisioningOperation.ProvisioningParameters.ErsContext.SubAccountID
	if provisioningOperation.ProvisioningParameters.Parameters.Region != nil {
		result.Region = *provisioningOperation.ProvisioningParameters.Parameters.Region
	}

	if len(operations) > 1 {
		lastDeprovisioning := operations[len(operations)-1]
		result.ShootName = lastDeprovisioning.ShootName
		result.Region = lastDeprovisioning.Region
		result.LastRuntimeID = lastDeprovisioning.RuntimeID
		result.LastDeprovisioningFinishedAt = lastDeprovisioning.UpdatedAt

		// if GA or SA has been changed, let's take it from the last deprovisioning operation
		result.GlobalAccountID = lastDeprovisioning.ProvisioningParameters.ErsContext.GlobalAccountID
		result.SubaccountID = lastDeprovisioning.ProvisioningParameters.ErsContext.SubAccountID
	}
	result.InternalUser = strings.Contains(provisioningOperation.ProvisioningParameters.ErsContext.UserID, "@sap.com")
	result.SubaccountRegion = provisioningOperation.ProvisioningParameters.PlatformRegion

	// find first deprovisioning
	for _, op := range operations {
		if op.Type == internal.OperationTypeDeprovision && !op.Temporary {
			result.FirstDeprovisioningStartedAt = op.CreatedAt
			result.FirstDeprovisioningFinishedAt = op.UpdatedAt
			break
		}
	}

	return result, nil
}
