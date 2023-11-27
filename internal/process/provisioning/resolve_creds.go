package provisioning

import (
	"fmt"
	"time"

	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"

	"github.com/sirupsen/logrus"
)

type ResolveCredentialsStep struct {
	operationManager *process.OperationManager
	accountProvider  hyperscaler.AccountProvider
	opStorage        storage.Operations
	tenant           string
}

func NewResolveCredentialsStep(os storage.Operations, accountProvider hyperscaler.AccountProvider) *ResolveCredentialsStep {
	return &ResolveCredentialsStep{
		operationManager: process.NewOperationManager(os),
		opStorage:        os,
		accountProvider:  accountProvider,
	}
}

func (s *ResolveCredentialsStep) Name() string {
	return "Resolve_Target_Secret"
}

func (s *ResolveCredentialsStep) Run(operation internal.Operation, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {

	hypType, err := HypTypeFromOperation(operation)
	if err != nil {
		msg := fmt.Sprintf("failing to determine the type of Hyperscaler to use for planID: %s", operation.ProvisioningParameters.PlanID)
		log.Errorf("Aborting after %s", msg)
		return s.operationManager.OperationFailed(operation, msg, err, log)
	}

	euAccess := internal.IsEuAccess(operation.ProvisioningParameters.PlatformRegion)

	log.Infof("HAP lookup for credentials secret binding to provision cluster for global account ID %s on Hyperscaler %s, euAccess %v", operation.ProvisioningParameters.ErsContext.GlobalAccountID, hypType.GetKey(), euAccess)

	targetSecret, err := s.getTargetSecretFromGardener(operation, log, hypType, euAccess)
	if err != nil {
		return s.retryOrFailOperation(operation, log, hypType, err)
	}

	s.overwriteProvisioningParameters(&operation, targetSecret, hypType)
	updatedOperation, err := s.opStorage.UpdateOperation(operation)
	if err != nil {
		return operation, 1 * time.Minute, nil
	}

	log.Infof("Resolved %s as target secret name to use for cluster provisioning for global account ID %s on Hyperscaler %s", *operation.ProvisioningParameters.Parameters.TargetSecret, operation.ProvisioningParameters.ErsContext.GlobalAccountID, hypType.GetKey())

	return *updatedOperation, 0, nil
}

func (s *ResolveCredentialsStep) retryOrFailOperation(operation internal.Operation, log logrus.FieldLogger, hypType hyperscaler.Type, err error) (internal.Operation, time.Duration, error) {
	msg := fmt.Sprintf("HAP lookup for secret binding to provision cluster for global account ID %s on Hyperscaler %s has failed", operation.ProvisioningParameters.ErsContext.GlobalAccountID, hypType.GetKey())
	errMsg := fmt.Sprintf("%s: %s", msg, err)
	log.Info(errMsg)

	// if failed retry step every 10s by next 10min
	dur := time.Since(operation.UpdatedAt).Round(time.Minute)

	if dur < 10*time.Minute {
		return operation, 10 * time.Second, nil
	}

	log.Errorf("Aborting after 10 minutes of failing to resolve provisioning secret binding for global account ID %s on Hyperscaler %s", operation.ProvisioningParameters.ErsContext.GlobalAccountID, hypType.GetKey())

	return s.operationManager.OperationFailed(operation, msg, err, log)
}

func (s *ResolveCredentialsStep) overwriteProvisioningParameters(operation *internal.Operation, targetSecret string, hypType hyperscaler.Type) {
	operation.ProvisioningParameters.Parameters.TargetSecret = &targetSecret

	if hypType.GetName() == "openstack" {
		// Overwrite the region parameter in case default region is used. This is necessary until region is mandatory (Jan 2024).
		// This is the simplest way to make the region available during deprovisioning when we release subscription
		effectiveRegion := getEffectiveRegion(*operation)
		operation.ProvisioningParameters.Parameters.Region = &effectiveRegion
	}
}

func (s *ResolveCredentialsStep) getTargetSecretFromGardener(operation internal.Operation, log logrus.FieldLogger, hypType hyperscaler.Type, euAccess bool) (string, error) {
	var secretName string
	var err error
	if !broker.IsTrialPlan(operation.ProvisioningParameters.PlanID) {
		log.Infof("HAP lookup for secret binding")
		secretName, err = s.accountProvider.GardenerSecretName(hypType, operation.ProvisioningParameters.ErsContext.GlobalAccountID, euAccess)
	} else {
		log.Infof("HAP lookup for shared secret binding")
		secretName, err = s.accountProvider.GardenerSharedSecretName(hypType, euAccess)
	}
	return secretName, err
}

func getEffectiveRegion(operation internal.Operation) string {
	clusterInput, _ := operation.InputCreator.CreateProvisionClusterInput()
	effectiveRegion := clusterInput.ClusterConfig.GardenerConfig.Region
	return effectiveRegion
}

func HypTypeFromOperation(operation internal.Operation) (hyperscaler.Type, error) {
	cloudProvider := operation.InputCreator.Provider()
	switch cloudProvider {
	case internal.Azure:
		return hyperscaler.Azure(), nil
	case internal.AWS:
		return hyperscaler.AWS(), nil
	case internal.GCP:
		return hyperscaler.GCP(), nil
	case internal.Openstack:
		return hyperscaler.Openstack(getEffectiveRegion(operation)), nil
	default:
		return hyperscaler.Type{}, fmt.Errorf("cannot determine the type of Hyperscaler to use for cloud provider %s", cloudProvider)
	}
}
