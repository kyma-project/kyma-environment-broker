package update

import (
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
)

func ForBTPOperatorCredentialsProvided(op internal.Operation) bool {
	return op.ProvisioningParameters.ErsContext.SMOperatorCredentials != nil
}

func SkipForOwnClusterPlan(op internal.Operation) bool {
	return !broker.IsOwnClusterPlan(op.ProvisioningParameters.PlanID)
}
