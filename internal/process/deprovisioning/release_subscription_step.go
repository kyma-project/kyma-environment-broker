package deprovisioning

import (
	"fmt"
	"log/slog"
	"time"

	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"

	"github.com/kyma-project/kyma-environment-broker/internal/euaccess"

	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"

	"github.com/kyma-project/kyma-environment-broker/internal"
)

type ReleaseSubscriptionStep struct {
	operationManager *process.OperationManager
	instanceStorage  storage.Instances
	accountProvider  hyperscaler.AccountProvider
}

var _ process.Step = &ReleaseSubscriptionStep{}

func NewReleaseSubscriptionStep(os storage.Operations, instanceStorage storage.Instances, accountProvider hyperscaler.AccountProvider) ReleaseSubscriptionStep {
	step := ReleaseSubscriptionStep{
		instanceStorage: instanceStorage,
		accountProvider: accountProvider,
	}
	step.operationManager = process.NewOperationManager(os, step.Name(), kebError.AccountPoolDependency)
	return step
}

func (s ReleaseSubscriptionStep) Name() string {
	return "Release_Subscription"
}

func (s ReleaseSubscriptionStep) Dependency() kebError.Component {
	return s.operationManager.Component()
}

func (s ReleaseSubscriptionStep) Run(operation internal.Operation, log *slog.Logger) (internal.Operation, time.Duration, error) {

	planID := operation.ProvisioningParameters.PlanID
	if needsRelease(planID) {
		instance, err := s.instanceStorage.GetByID(operation.InstanceID)
		if err != nil {
			msg := fmt.Sprintf("after successful deprovisioning failing to release hyperscaler subscription - get the instance data for instanceID [%s]: %s", operation.InstanceID, err.Error())
			operation, repeat, err := s.operationManager.MarkStepAsExcutedButNotCompleted(operation, s.Name(), msg, log)
			if repeat != 0 {
				return operation, repeat, err
			}
			return operation, 0, nil
		}

		if string(instance.Provider) == "" {
			log.Info("Instance does not contain cloud provider info due to failed provisioning, skipping")
			return operation, 0, nil
		}

		hypType, err := hyperscaler.HypTypeFromCloudProviderWithRegion(instance.Provider, &instance.ProviderRegion, &operation.ProvisioningParameters.PlatformRegion)
		if err != nil {
			msg := fmt.Sprintf("after successful deprovisioning failing to release hyperscaler subscription - determine the type of hyperscaler to use for planID [%s]: %s", planID, err.Error())
			operation, repeat, err := s.operationManager.MarkStepAsExcutedButNotCompleted(operation, s.Name(), msg, log)
			if repeat != 0 {
				return operation, repeat, err
			}
			return operation, 0, nil
		}

		euAccess := euaccess.IsEURestrictedAccess(operation.ProvisioningParameters.PlatformRegion)
		err = s.accountProvider.MarkUnusedGardenerSecretBindingAsDirty(hypType, instance.GetSubscriptionGlobalAccoundID(), euAccess)
		if err != nil {
			log.Error(fmt.Sprintf("after successful deprovisioning failed to release hyperscaler subscription: %v", err))
			return operation, 10 * time.Second, nil
		}
	}
	return operation, 0, nil
}

func needsRelease(planID string) bool {
	return !broker.IsTrialPlan(planID) && !broker.IsOwnClusterPlan(planID) && !broker.IsSapConvergedCloudPlan(planID)
}
