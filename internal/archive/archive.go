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
	result.PlanName = broker.PlanNamesMapping[result.PlanID]
	result.InstanceID = provisioningOperation.InstanceID

	if len(operations) > 1 {
		lastDeprovisioning := operations[len(operations)-1]
		result.SubaccountID = lastDeprovisioning.SubAccountID
		result.GlobalAccountID = lastDeprovisioning.GlobalAccountID
		result.ShootName = lastDeprovisioning.ShootName
		result.Region = lastDeprovisioning.Region
		result.LastRuntimeID = lastDeprovisioning.RuntimeID
		result.LastDeprovisioningFinishedAt = lastDeprovisioning.UpdatedAt
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
