package provisioning

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
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
	"github.com/kyma-project/kyma-environment-broker/internal/process/infrastructure_manager"
	"github.com/kyma-project/kyma-environment-broker/internal/provider"
	"k8s.io/apimachinery/pkg/util/intstr"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

const (
	kcpRetryInterval = 3 * time.Second
	kcpRetryTimeout  = 20 * time.Second
	dbRetryInterval  = 10 * time.Second
	dbRetryTimeout   = 1 * time.Minute
)

type CreateRuntimeResourceStep struct {
	operationManager  *process.OperationManager
	instanceStorage   storage.Instances
	k8sClient         client.Client
	config            infrastructure_manager.InfrastructureManagerConfig
	oidcDefaultValues pkg.OIDCConfigDTO
}

func NewCreateRuntimeResourceStep(os storage.Operations, is storage.Instances, k8sClient client.Client, infrastructureManagerConfig infrastructure_manager.InfrastructureManagerConfig,
	oidcDefaultValues pkg.OIDCConfigDTO) *CreateRuntimeResourceStep {
	step := &CreateRuntimeResourceStep{
		instanceStorage:   is,
		k8sClient:         k8sClient,
		config:            infrastructureManagerConfig,
		oidcDefaultValues: oidcDefaultValues,
	}
	step.operationManager = process.NewOperationManager(os, step.Name(), kebError.InfrastructureManagerDependency)
	return step
}

func (s *CreateRuntimeResourceStep) Name() string {
	return "Create_Runtime_Resource"
}

func (s *CreateRuntimeResourceStep) Run(operation internal.Operation, log *slog.Logger) (internal.Operation, time.Duration, error) {

	kymaResourceName := operation.KymaResourceName
	kymaResourceNamespace := operation.KymaResourceNamespace
	runtimeResourceName := steps.KymaRuntimeResourceName(operation)
	log.Info(fmt.Sprintf("KymaResourceName: %s, KymaResourceNamespace: %s, RuntimeResourceName: %s", kymaResourceName, kymaResourceNamespace, runtimeResourceName))

	operation.CloudProvider = string(provider.ProviderToCloudProvider(operation.ProviderValues.ProviderType))

	runtimeCR, err := s.getEmptyOrExistingRuntimeResource(runtimeResourceName, kymaResourceNamespace)
	if err != nil {
		log.Error(fmt.Sprintf("unable to get Runtime resource %s/%s", operation.KymaResourceNamespace, runtimeResourceName))
		return s.operationManager.RetryOperation(operation, "unable to get Runtime resource", err, kcpRetryInterval, kcpRetryTimeout, log)
	}

	if runtimeCR.GetResourceVersion() != "" {
		log.Info(fmt.Sprintf("Runtime resource already created %s/%s: ", operation.KymaResourceNamespace, runtimeResourceName))
		return operation, 0, nil
	} else {
		err = s.updateRuntimeResourceObject(*operation.ProviderValues, runtimeCR, operation, runtimeResourceName, operation.CloudProvider)
		if err != nil {
			return s.operationManager.OperationFailed(operation, fmt.Sprintf("while creating Runtime CR object: %s", err), err, log)
		}
		err = s.k8sClient.Create(context.Background(), runtimeCR)
		if err != nil {
			log.Error(fmt.Sprintf("unable to create Runtime resource: %s/%s: %s", operation.KymaResourceNamespace, runtimeResourceName, err.Error()))
			return s.operationManager.RetryOperation(operation, "unable to create Runtime resource", err, kcpRetryInterval, kcpRetryTimeout, log)
		}
		log.Info(fmt.Sprintf("Runtime resource %s/%s creation process finished successfully", operation.KymaResourceNamespace, runtimeResourceName))

		operation, backoff, _ := s.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
			op.Region = runtimeCR.Spec.Shoot.Region
			op.CloudProvider = operation.CloudProvider
		}, log)
		if backoff > 0 {
			return s.operationManager.RetryOperation(operation, "cannot update operation", err, dbRetryInterval, dbRetryTimeout, log)
		}

		err = s.updateInstance(operation.InstanceID, runtimeCR.Spec.Shoot.Region)

		switch {
		case err == nil:
		case dberr.IsConflict(err):
			err = s.updateInstance(operation.InstanceID, runtimeCR.Spec.Shoot.Region)
			if err != nil {
				log.Error(fmt.Sprintf("cannot update instance: %s", err))
				return s.operationManager.RetryOperation(operation, "cannot update instance", err, dbRetryInterval, dbRetryTimeout, log)
			}
		default:
			log.Error(fmt.Sprintf("cannot update instance: %s", err))
			return s.operationManager.RetryOperation(operation, "cannot update instance", err, dbRetryInterval, dbRetryTimeout, log)
		}
	}
	return operation, 0, nil
}

func (s *CreateRuntimeResourceStep) updateRuntimeResourceObject(values internal.ProviderValues, runtime *imv1.Runtime, operation internal.Operation, runtimeName, cloudProvider string) error {

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
	runtime.Spec.Shoot.ControlPlane = s.createHighAvailabilityConfiguration(values.FailureTolerance)
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

func (s *CreateRuntimeResourceStep) createShootProvider(operation *internal.Operation, values internal.ProviderValues) (imv1.Provider, error) {

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

	if len(operation.ProvisioningParameters.Parameters.AdditionalWorkerNodePools) > 0 {
		additionalWorkers := CreateAdditionalWorkers(s.config, values, nil, operation.ProvisioningParameters.Parameters.AdditionalWorkerNodePools, values.Zones)
		provider.AdditionalWorkers = &additionalWorkers
	}

	return provider, nil
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

	kubernetesConfig := imv1.Kubernetes{
		Version:       ptr.String(s.config.KubernetesVersion),
		KubeAPIServer: imv1.APIServer{},
	}

	if s.config.UseMainOIDC {
		kubernetesConfig.KubeAPIServer.OidcConfig = oidc
		kubernetesConfig.KubeAPIServer.AdditionalOidcConfig = nil
	}

	if s.config.UseAdditionalOIDC {
		oidc.GroupsPrefix = ptr.String("-")
		kubernetesConfig.KubeAPIServer.AdditionalOidcConfig = &[]gardener.OIDCConfig{oidc}
	}

	return kubernetesConfig
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

func (s *CreateRuntimeResourceStep) createHighAvailabilityConfiguration(tolerance *string) *gardener.ControlPlane {
	if tolerance == nil {
		return nil
	}
	if *tolerance == "" {
		return nil
	}
	return &gardener.ControlPlane{HighAvailability: &gardener.HighAvailability{
		FailureTolerance: gardener.FailureTolerance{
			Type: gardener.FailureToleranceType(*tolerance),
		},
	},
	}
}

func DefaultIfParamNotSet[T interface{}](d T, param *T) T {
	if param == nil {
		return d
	}
	return *param
}

func CreateAdditionalWorkers(imConfig infrastructure_manager.InfrastructureManagerConfig, values internal.ProviderValues, currentAdditionalWorkers map[string]gardener.Worker, additionalWorkerNodePools []pkg.AdditionalWorkerNodePool, zones []string) []gardener.Worker {
	additionalWorkerNodePoolsMaxUnavailable := intstr.FromInt32(int32(0))
	workers := make([]gardener.Worker, 0, len(additionalWorkerNodePools))

	for _, additionalWorkerNodePool := range additionalWorkerNodePools {
		currentAdditionalWorker, exists := currentAdditionalWorkers[additionalWorkerNodePool.Name]

		var workerZones []string
		if exists {
			workerZones = currentAdditionalWorker.Zones
		} else {
			workerZones = zones
			if !additionalWorkerNodePool.HAZones {
				rand.Shuffle(len(workerZones), func(i, j int) { workerZones[i], workerZones[j] = workerZones[j], workerZones[i] })
				workerZones = workerZones[:1]
			}
		}
		workerMaxSurge := intstr.FromInt32(int32(len(workerZones)))

		worker := gardener.Worker{
			Name: additionalWorkerNodePool.Name,
			Machine: gardener.Machine{
				Type: additionalWorkerNodePool.MachineType,
				Image: &gardener.ShootMachineImage{
					Name:    imConfig.MachineImage,
					Version: &imConfig.MachineImageVersion,
				},
			},
			Maximum:        int32(additionalWorkerNodePool.AutoScalerMax),
			Minimum:        int32(additionalWorkerNodePool.AutoScalerMin),
			MaxSurge:       &workerMaxSurge,
			MaxUnavailable: &additionalWorkerNodePoolsMaxUnavailable,
			Zones:          workerZones,
		}

		if values.ProviderType != "openstack" {
			volumeSize := strconv.Itoa(values.VolumeSizeGb)
			worker.Volume = &gardener.Volume{
				Type:       ptr.String(values.DiskType),
				VolumeSize: fmt.Sprintf("%sGi", volumeSize),
			}
		}

		workers = append(workers, worker)
	}

	return workers
}
