package provisioning

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"

	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"

	"github.com/kyma-project/kyma-environment-broker/internal/customresources"

	"github.com/kyma-project/kyma-environment-broker/internal/ptr"

	"github.com/kyma-project/kyma-environment-broker/internal/process/steps"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/kyma-project/kyma-environment-broker/internal/networking"

	"sigs.k8s.io/controller-runtime/pkg/client"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-project/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/kyma-environment-broker/internal/provider"
	"k8s.io/apimachinery/pkg/util/intstr"

	"sigs.k8s.io/yaml"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

type CreateRuntimeResourceStep struct {
	operationManager           *process.OperationManager
	instanceStorage            storage.Instances
	runtimeStateStorage        storage.RuntimeStates
	k8sClient                  client.Client
	kimConfig                  broker.KimConfig
	config                     input.Config
	trialPlatformRegionMapping map[string]string
	useSmallerMachineTypes     bool
	oidcDefaultValues          pkg.OIDCConfigDTO
}

func NewCreateRuntimeResourceStep(os storage.Operations, is storage.Instances, k8sClient client.Client, kimConfig broker.KimConfig, cfg input.Config,
	trialPlatformRegionMapping map[string]string, useSmallerMachines bool, oidcDefaultValues pkg.OIDCConfigDTO) *CreateRuntimeResourceStep {
	step := &CreateRuntimeResourceStep{
		instanceStorage:            is,
		kimConfig:                  kimConfig,
		k8sClient:                  k8sClient,
		config:                     cfg,
		trialPlatformRegionMapping: trialPlatformRegionMapping,
		useSmallerMachineTypes:     useSmallerMachines,
		oidcDefaultValues:          oidcDefaultValues,
	}
	step.operationManager = process.NewOperationManager(os, step.Name(), kebError.InfrastructureManagerDependency)
	return step
}

func (s *CreateRuntimeResourceStep) Name() string {
	return "Create_Runtime_Resource"
}

func (s *CreateRuntimeResourceStep) Dependency() kebError.Component {
	return s.operationManager.Component()
}

func (s *CreateRuntimeResourceStep) Run(operation internal.Operation, log *slog.Logger) (internal.Operation, time.Duration, error) {
	if time.Since(operation.UpdatedAt) > CreateRuntimeTimeout {
		log.Info(fmt.Sprintf("operation has reached the time limit: updated operation time: %s", operation.UpdatedAt))
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("operation has reached the time limit: %s", CreateRuntimeTimeout), nil, log)
	}

	if !s.kimConfig.IsEnabledForPlan(broker.PlanNamesMapping[operation.ProvisioningParameters.PlanID]) {
		if !s.kimConfig.Enabled {
			log.Info("KIM is not enabled, skipping")
			return operation, 0, nil
		}
		log.Info(fmt.Sprintf("KIM is not enabled for plan %s, skipping", broker.PlanNamesMapping[operation.ProvisioningParameters.PlanID]))
		return operation, 0, nil
	}

	kymaResourceName := operation.KymaResourceName
	kymaResourceNamespace := operation.KymaResourceNamespace
	runtimeResourceName := steps.KymaRuntimeResourceName(operation)
	log.Info(fmt.Sprintf("KymaResourceName: %s, KymaResourceNamespace: %s, RuntimeResourceName: %s", kymaResourceName, kymaResourceNamespace, runtimeResourceName))

	values, err := provider.GetPlanSpecificValues(&operation, s.config.MultiZoneCluster, s.config.DefaultTrialProvider, s.useSmallerMachineTypes, s.trialPlatformRegionMapping, s.config.DefaultGardenerShootPurpose)
	if err != nil {
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("while updating calculating plan specific values : %s", err), err, log)
	}
	operation.CloudProvider = string(provider.ProviderToCloudProvider(values.ProviderType))

	if s.kimConfig.DryRun {
		runtimeCR := &imv1.Runtime{}
		err := s.updateRuntimeResourceObject(values, runtimeCR, operation, runtimeResourceName, operation.CloudProvider)

		if err != nil {
			return s.operationManager.OperationFailed(operation, fmt.Sprintf("while updating Runtime resource object: %s", err), err, log)
		}
		yaml, err := RuntimeToYaml(runtimeCR)
		if err != nil {
			log.Error(fmt.Sprintf("failed to encode Runtime resource as yaml: %s", err))
		} else {
			fmt.Println(yaml)
		}
	} else {
		runtimeCR, err := s.getEmptyOrExistingRuntimeResource(runtimeResourceName, kymaResourceNamespace)
		if err != nil {
			log.Error(fmt.Sprintf("unable to get Runtime resource %s/%s", operation.KymaResourceNamespace, runtimeResourceName))
			return s.operationManager.RetryOperation(operation, "unable to get Runtime resource", err, 3*time.Second, 20*time.Second, log)
		}
		if runtimeCR.GetResourceVersion() != "" {
			log.Info(fmt.Sprintf("Runtime resource already created %s/%s: ", operation.KymaResourceNamespace, runtimeResourceName))
			return operation, 0, nil
		} else {
			err := s.updateRuntimeResourceObject(values, runtimeCR, operation, runtimeResourceName, operation.CloudProvider)
			if err != nil {
				return s.operationManager.OperationFailed(operation, fmt.Sprintf("while creating Runtime CR object: %s", err), err, log)
			}
			err = s.k8sClient.Create(context.Background(), runtimeCR)
			if err != nil {
				log.Error(fmt.Sprintf("unable to create Runtime resource: %s/%s: %s", operation.KymaResourceNamespace, runtimeResourceName, err.Error()))
				return s.operationManager.RetryOperation(operation, "unable to create Runtime resource", err, 3*time.Second, 20*time.Second, log)
			}
		}
		log.Info(fmt.Sprintf("Runtime resource %s/%s creation process finished successfully", operation.KymaResourceNamespace, runtimeResourceName))

		newOp, backoff, _ := s.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
			op.Region = runtimeCR.Spec.Shoot.Region
			op.CloudProvider = operation.CloudProvider
		}, log)
		if backoff > 0 {
			return newOp, backoff, nil
		}
		operation = newOp

		err = s.updateInstance(operation.InstanceID, runtimeCR.Spec.Shoot.Region)

		switch {
		case err == nil:
		case dberr.IsConflict(err):
			err := s.updateInstance(operation.InstanceID, runtimeCR.Spec.Shoot.Region)
			if err != nil {
				log.Error(fmt.Sprintf("cannot update instance: %s", err))
				return operation, 1 * time.Minute, nil
			}
		}
	}
	return operation, 0, nil
}

func (s *CreateRuntimeResourceStep) updateRuntimeResourceObject(values provider.Values, runtime *imv1.Runtime, operation internal.Operation, runtimeName, cloudProvider string) error {

	runtime.ObjectMeta.Name = runtimeName
	runtime.ObjectMeta.Namespace = operation.KymaResourceNamespace

	runtime.ObjectMeta.Labels = s.createLabelsForRuntime(operation, values.Region, cloudProvider)

	providerObj, err := s.createShootProvider(&operation, values)
	if err != nil {
		return err
	}

	runtime.Spec.Shoot.Provider = providerObj
	runtime.Spec.Shoot.Region = values.Region
	runtime.Spec.Shoot.Name = operation.ShootName
	runtime.Spec.Shoot.Purpose = gardener.ShootPurpose(values.Purpose)
	runtime.Spec.Shoot.PlatformRegion = operation.ProvisioningParameters.PlatformRegion
	runtime.Spec.Shoot.SecretBindingName = *operation.ProvisioningParameters.Parameters.TargetSecret
	if runtime.Spec.Shoot.ControlPlane == nil {
		runtime.Spec.Shoot.ControlPlane = &gardener.ControlPlane{}
	}
	runtime.Spec.Shoot.ControlPlane.HighAvailability = s.createHighAvailabilityConfiguration()
	runtime.Spec.Shoot.EnforceSeedLocation = operation.ProvisioningParameters.Parameters.ShootAndSeedSameRegion
	runtime.Spec.Shoot.Networking = s.createNetworkingConfiguration(operation)
	runtime.Spec.Shoot.Kubernetes = s.createKubernetesConfiguration(operation)

	runtime.Spec.Security = s.createSecurityConfiguration(operation)

	return nil
}

func (s *CreateRuntimeResourceStep) createLabelsForRuntime(operation internal.Operation, region string, cloudProvider string) map[string]string {
	labels := steps.SetCommonLabels(map[string]string{}, operation)
	labels[customresources.RegionLabel] = region
	labels[customresources.CloudProviderLabel] = cloudProvider

	controlledByProvisioner := s.kimConfig.ViewOnly && !s.kimConfig.IsDrivenByKimOnly(broker.PlanNamesMapping[operation.ProvisioningParameters.PlanID])
	labels[imv1.LabelControlledByProvisioner] = strconv.FormatBool(controlledByProvisioner)
	return labels
}

func (s *CreateRuntimeResourceStep) createSecurityConfiguration(operation internal.Operation) imv1.Security {
	security := imv1.Security{}
	if len(operation.ProvisioningParameters.Parameters.RuntimeAdministrators) == 0 {
		// default admin set from UserID in ERSContext
		security.Administrators = []string{operation.ProvisioningParameters.ErsContext.UserID}
	} else {
		security.Administrators = operation.ProvisioningParameters.Parameters.RuntimeAdministrators
	}

	// In Runtime CR logic is positive, so we need to negate the value
	disabled := *operation.ProvisioningParameters.ErsContext.DisableEnterprisePolicyFilter()
	security.Networking.Filter.Egress.Enabled = !disabled

	// Ingress is not supported yet, nevertheless we set it for completeness
	security.Networking.Filter.Ingress = &imv1.Ingress{Enabled: false}
	return security
}

func (s *CreateRuntimeResourceStep) createShootProvider(operation *internal.Operation, values provider.Values) (imv1.Provider, error) {

	maxSurge := intstr.FromInt32(int32(DefaultIfParamNotSet(values.ZonesCount, operation.ProvisioningParameters.Parameters.MaxSurge)))
	maxUnavailable := intstr.FromInt32(int32(DefaultIfParamNotSet(0, operation.ProvisioningParameters.Parameters.MaxUnavailable)))

	scalerMax := int32(DefaultIfParamNotSet(values.DefaultAutoScalerMax, operation.ProvisioningParameters.Parameters.AutoScalerMax))
	scalerMin := int32(DefaultIfParamNotSet(values.DefaultAutoScalerMin, operation.ProvisioningParameters.Parameters.AutoScalerMin))

	provider := imv1.Provider{
		Type: values.ProviderType,
		Workers: []gardener.Worker{
			{
				Name: "cpu-worker-0",
				Machine: gardener.Machine{
					Type: DefaultIfParamNotSet(values.DefaultMachineType, operation.ProvisioningParameters.Parameters.MachineType),
					Image: &gardener.ShootMachineImage{
						Name:    s.config.MachineImage,
						Version: &s.config.MachineImageVersion,
					},
				},
				Maximum:        scalerMax,
				Minimum:        scalerMin,
				MaxSurge:       &maxSurge,
				MaxUnavailable: &maxUnavailable,
				Zones:          values.Zones,
			},
		},
	}

	if values.ProviderType != "openstack" {
		volumeSize := strconv.Itoa(DefaultIfParamNotSet(values.VolumeSizeGb, operation.ProvisioningParameters.Parameters.VolumeSizeGb))
		provider.Workers[0].Volume = &gardener.Volume{
			Type:       ptr.String(values.DiskType),
			VolumeSize: fmt.Sprintf("%sGi", volumeSize),
		}
	}
	return provider, nil
}

func (s *CreateRuntimeResourceStep) createHighAvailabilityConfiguration() *gardener.HighAvailability {

	failureToleranceType := gardener.FailureToleranceTypeZone
	if s.config.ControlPlaneFailureTolerance != string(gardener.FailureToleranceTypeZone) {
		failureToleranceType = gardener.FailureToleranceTypeNode
	}

	return &gardener.HighAvailability{
		FailureTolerance: gardener.FailureTolerance{
			Type: failureToleranceType,
		},
	}
}

func (s *CreateRuntimeResourceStep) createNetworkingConfiguration(operation internal.Operation) imv1.Networking {

	networkingParams := operation.ProvisioningParameters.Parameters.Networking
	if networkingParams == nil {
		networkingParams = &pkg.NetworkingDTO{}
	}

	nodes := networking.DefaultNodesCIDR
	if networkingParams.NodesCidr != "" {
		nodes = networkingParams.NodesCidr
	}

	return imv1.Networking{
		Pods:     DefaultIfParamNotSet(networking.DefaultPodsCIDR, networkingParams.PodsCidr),
		Services: DefaultIfParamNotSet(networking.DefaultServicesCIDR, networkingParams.ServicesCidr),
		Nodes:    nodes,
		//TODO remove when KIM is ready with setting this value
		Type: ptr.String("calico"),
	}
}

func (s *CreateRuntimeResourceStep) getEmptyOrExistingRuntimeResource(name, namespace string) (*imv1.Runtime, error) {
	runtime := imv1.Runtime{}
	err := s.k8sClient.Get(context.Background(), client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}, &runtime)

	if err != nil && !errors.IsNotFound(err) {
		return nil, err
	}
	return &runtime, nil
}

func (s *CreateRuntimeResourceStep) createKubernetesConfiguration(operation internal.Operation) imv1.Kubernetes {
	oidc := gardener.OIDCConfig{
		ClientID:       &s.oidcDefaultValues.ClientID,
		GroupsClaim:    &s.oidcDefaultValues.GroupsClaim,
		IssuerURL:      &s.oidcDefaultValues.IssuerURL,
		SigningAlgs:    s.oidcDefaultValues.SigningAlgs,
		UsernameClaim:  &s.oidcDefaultValues.UsernameClaim,
		UsernamePrefix: &s.oidcDefaultValues.UsernamePrefix,
	}
	if operation.ProvisioningParameters.Parameters.OIDC != nil {
		if operation.ProvisioningParameters.Parameters.OIDC.ClientID != "" {
			oidc.ClientID = &operation.ProvisioningParameters.Parameters.OIDC.ClientID
		}
		if operation.ProvisioningParameters.Parameters.OIDC.GroupsClaim != "" {
			oidc.GroupsClaim = &operation.ProvisioningParameters.Parameters.OIDC.GroupsClaim
		}
		if operation.ProvisioningParameters.Parameters.OIDC.IssuerURL != "" {
			oidc.IssuerURL = &operation.ProvisioningParameters.Parameters.OIDC.IssuerURL
		}
		if len(operation.ProvisioningParameters.Parameters.OIDC.SigningAlgs) > 0 {
			oidc.SigningAlgs = operation.ProvisioningParameters.Parameters.OIDC.SigningAlgs
		}
		if operation.ProvisioningParameters.Parameters.OIDC.UsernameClaim != "" {
			oidc.UsernameClaim = &operation.ProvisioningParameters.Parameters.OIDC.UsernameClaim
		}
		if operation.ProvisioningParameters.Parameters.OIDC.UsernamePrefix != "" {
			oidc.UsernamePrefix = &operation.ProvisioningParameters.Parameters.OIDC.UsernamePrefix
		}
	}

	return imv1.Kubernetes{
		Version: ptr.String(s.config.KubernetesVersion),
		KubeAPIServer: imv1.APIServer{
			OidcConfig:           oidc,
			AdditionalOidcConfig: nil,
		},
	}
}

func (s *CreateRuntimeResourceStep) updateInstance(id string, region string) error {
	instance, err := s.instanceStorage.GetByID(id)
	if err != nil {
		return fmt.Errorf("while getting instance: %w", err)
	}
	instance.ProviderRegion = region
	_, err = s.instanceStorage.Update(*instance)
	if err != nil {
		return fmt.Errorf("while updating instance: %w", err)
	}

	return nil
}

func DefaultIfParamNotSet[T interface{}](d T, param *T) T {
	if param == nil {
		return d
	}
	return *param
}

func RuntimeToYaml(runtime *imv1.Runtime) (string, error) {
	result, err := yaml.Marshal(runtime)
	if err != nil {
		return "", err
	}
	return string(result), nil
}
