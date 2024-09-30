package update

import (
	"context"
	"fmt"
	"time"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const DryRunPrefix = "dry_run-"
const retryDuration = 10 * time.Second

type UpgradeShootStep struct {
	operationManager    *process.OperationManager
	provisionerClient   provisioner.Client
	runtimeStateStorage storage.RuntimeStates
	k8sClient           client.Client
}

// TODO: this step is not necessary when the Provisioner is switched to KIM

func NewUpgradeShootStep(
	os storage.Operations,
	runtimeStorage storage.RuntimeStates,
	cli provisioner.Client, k8sClient client.Client) *UpgradeShootStep {

	return &UpgradeShootStep{
		operationManager:    process.NewOperationManager(os),
		provisionerClient:   cli,
		runtimeStateStorage: runtimeStorage,
		k8sClient:           k8sClient,
	}
}

func (s *UpgradeShootStep) Name() string {
	return "Upgrade_Shoot"
}

func (s *UpgradeShootStep) Run(operation internal.Operation, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	if operation.RuntimeID == "" {
		log.Infof("Runtime does not exists, skipping a call to Provisioner")
		return operation, 0, nil
	}
	log = log.WithField("runtimeID", operation.RuntimeID)

	// decide if the step should be skipped because the runtime is not controlled by the provisioner
	var runtime imv1.Runtime
	err := s.k8sClient.Get(context.Background(), client.ObjectKey{Name: operation.GetRuntimeResourceName(),
		Namespace: operation.GetRuntimeResourceNamespace()}, &runtime)
	if err != nil && !errors.IsNotFound(err) {
		log.Warnf("Unable to read runtime: %s", err)
		return s.operationManager.RetryOperation(operation, err.Error(), err, 5*time.Second, 1*time.Minute, log)
	}
	if !runtime.IsControlledByProvisioner() {
		log.Infof("Skipping because the runtime is not controlled by the provisioner")
		return operation, 0, nil
	}

	latestRuntimeStateWithOIDC, err := s.runtimeStateStorage.GetLatestWithOIDCConfigByRuntimeID(operation.RuntimeID)
	if err != nil {
		return s.operationManager.RetryOperation(operation, err.Error(), err, 5*time.Second, 1*time.Minute, log)
	}

	input, err := s.createUpgradeShootInput(operation, &latestRuntimeStateWithOIDC.ClusterConfig)
	if err != nil {
		return s.operationManager.OperationFailed(operation, "invalid operation data - cannot create upgradeShoot input", err, log)
	}

	var provisionerResponse gqlschema.OperationStatus
	if operation.ProvisionerOperationID == "" {
		// trigger upgradeRuntime mutation
		provisionerResponse, err = s.provisionerClient.UpgradeShoot(operation.ProvisioningParameters.ErsContext.GlobalAccountID, operation.RuntimeID, input)
		if err != nil {
			log.Errorf("call to provisioner failed: %s", err)
			return s.operationManager.RetryOperation(operation, "call to provisioner failed", err, retryDuration, time.Minute, log)
		}

		repeat := time.Duration(0)
		operation, repeat, _ = s.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
			op.ProvisionerOperationID = *provisionerResponse.ID
			op.Description = "update in progress"
		}, log)
		if repeat != 0 {
			log.Errorf("cannot save operation ID from provisioner")
			return operation, retryDuration, nil
		}
	}

	log.Infof("call to provisioner succeeded for update, got operation ID %q", *provisionerResponse.ID)

	rs := internal.NewRuntimeState(*provisionerResponse.RuntimeID, operation.ID, nil, gardenerUpgradeInputToConfigInput(input))
	err = s.runtimeStateStorage.Insert(rs)
	if err != nil {
		log.Errorf("cannot insert runtimeState: %s", err)
		return operation, 10 * time.Second, nil
	}
	log.Infof("cluster upgrade process initiated successfully")

	// return repeat mode to start the initialization step which will now check the runtime status
	return operation, 0, nil

}

func (s *UpgradeShootStep) createUpgradeShootInput(operation internal.Operation, lastClusterConfig *gqlschema.GardenerConfigInput) (gqlschema.UpgradeShootInput, error) {
	operation.InputCreator.SetProvisioningParameters(operation.ProvisioningParameters)
	if lastClusterConfig.OidcConfig != nil {
		operation.InputCreator.SetOIDCLastValues(*lastClusterConfig.OidcConfig)
	}
	fullInput, err := operation.InputCreator.CreateUpgradeShootInput()
	if err != nil {
		return fullInput, fmt.Errorf("while building upgradeShootInput for provisioner: %w", err)
	}

	// modify configuration
	result := gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			OidcConfig:     fullInput.GardenerConfig.OidcConfig,
			AutoScalerMax:  operation.UpdatingParameters.AutoScalerMax,
			AutoScalerMin:  operation.UpdatingParameters.AutoScalerMin,
			MaxSurge:       operation.UpdatingParameters.MaxSurge,
			MaxUnavailable: operation.UpdatingParameters.MaxUnavailable,
			MachineType:    operation.UpdatingParameters.MachineType,
		},
		Administrators: fullInput.Administrators,
	}
	result.GardenerConfig.ShootNetworkingFilterDisabled = operation.ProvisioningParameters.ErsContext.DisableEnterprisePolicyFilter()

	return result, nil
}

func gardenerUpgradeInputToConfigInput(input gqlschema.UpgradeShootInput) *gqlschema.GardenerConfigInput {
	result := &gqlschema.GardenerConfigInput{
		MachineImage:        input.GardenerConfig.MachineImage,
		MachineImageVersion: input.GardenerConfig.MachineImageVersion,
		DiskType:            input.GardenerConfig.DiskType,
		VolumeSizeGb:        input.GardenerConfig.VolumeSizeGb,
		Purpose:             input.GardenerConfig.Purpose,
		OidcConfig:          input.GardenerConfig.OidcConfig,
	}
	if input.GardenerConfig.KubernetesVersion != nil {
		result.KubernetesVersion = *input.GardenerConfig.KubernetesVersion
	}
	if input.GardenerConfig.MachineType != nil {
		result.MachineType = *input.GardenerConfig.MachineType
	}
	if input.GardenerConfig.AutoScalerMin != nil {
		result.AutoScalerMin = *input.GardenerConfig.AutoScalerMin
	}
	if input.GardenerConfig.AutoScalerMax != nil {
		result.AutoScalerMax = *input.GardenerConfig.AutoScalerMax
	}
	if input.GardenerConfig.MaxSurge != nil {
		result.MaxSurge = *input.GardenerConfig.MaxSurge
	}
	if input.GardenerConfig.MaxUnavailable != nil {
		result.MaxUnavailable = *input.GardenerConfig.MaxUnavailable
	}
	if input.GardenerConfig.ShootNetworkingFilterDisabled != nil {
		result.ShootNetworkingFilterDisabled = input.GardenerConfig.ShootNetworkingFilterDisabled
	}

	return result
}
