package provisioning

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"strings"
	"time"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/customresources"
	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"
	"github.com/kyma-project/kyma-environment-broker/internal/networking"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/process/steps"
	"github.com/kyma-project/kyma-environment-broker/internal/provider"
	"github.com/kyma-project/kyma-environment-broker/internal/provider/configuration"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/kyma-environment-broker/internal/workers"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	config            broker.InfrastructureManager
	oidcDefaultValues pkg.OIDCConfigDTO
	workersProvider   *workers.Provider
	providerSpec      *configuration.ProviderSpec
}

func NewCreateRuntimeResourceStep(db storage.BrokerStorage, k8sClient client.Client, infrastructureManagerConfig broker.InfrastructureManager,
	oidcDefaultValues pkg.OIDCConfigDTO, workersProvider *workers.Provider, providerSpec *configuration.ProviderSpec) *CreateRuntimeResourceStep {
	step := &CreateRuntimeResourceStep{
		instanceStorage:   db.Instances(),
		k8sClient:         k8sClient,
		config:            infrastructureManagerConfig,
		oidcDefaultValues: oidcDefaultValues,
		workersProvider:   workersProvider,
		providerSpec:      providerSpec,
	}
	step.operationManager = process.NewOperationManager(db.Operations(), step.Name(), kebError.InfrastructureManagerDependency)
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
		var backoff time.Duration
		err = s.updateRuntimeResourceObject(log, *operation.ProviderValues, runtimeCR, operation, runtimeResourceName, operation.CloudProvider)
		if err != nil {
			return s.operationManager.OperationFailed(operation, fmt.Sprintf("while creating Runtime CR object: %s", err), err, log)
		}
		err = s.k8sClient.Create(context.Background(), runtimeCR)
		if err != nil {
			log.Error(fmt.Sprintf("unable to create Runtime resource: %s/%s: %s", operation.KymaResourceNamespace, runtimeResourceName, err.Error()))
			return s.operationManager.RetryOperation(operation, "unable to create Runtime resource", err, kcpRetryInterval, kcpRetryTimeout, log)
		}
		log.Info(fmt.Sprintf("Runtime resource %s/%s creation process finished successfully", operation.KymaResourceNamespace, runtimeResourceName))

		operation, backoff, _ = s.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
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
		return operation, 0, nil
	}
}

func (s *CreateRuntimeResourceStep) updateRuntimeResourceObject(log *slog.Logger, values internal.ProviderValues, runtime *imv1.Runtime, operation internal.Operation, runtimeName, cloudProvider string) error {

	runtime.ObjectMeta.Name = runtimeName
	runtime.ObjectMeta.Namespace = operation.KymaResourceNamespace

	runtime.ObjectMeta.Labels = s.createLabelsForRuntime(operation, values.Region, cloudProvider)

	providerObj, err := s.createShootProvider(log, &operation, values)
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
	runtime.Spec.Shoot.EnforceSeedLocation = operation.ProvisioningParameters.Parameters.ColocateControlPlane
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

	external := broker.IsExternalLicenseType(operation.ProvisioningParameters.ErsContext)

	// In Runtime CR logic is positive, so we need to negate the value
	security.Networking.Filter.Egress.Enabled = !external

	var ingressFiltering bool
	if steps.IsIngressFilteringEnabled(operation.ProvisioningParameters.PlanID, s.config, external) {
		ingressFiltering = operation.ProvisioningParameters.Parameters.IngressFiltering != nil && *operation.ProvisioningParameters.Parameters.IngressFiltering
	}
	security.Networking.Filter.Ingress = &imv1.Ingress{Enabled: ingressFiltering}

	return security
}

func (s *CreateRuntimeResourceStep) createShootProvider(log *slog.Logger, operation *internal.Operation, values internal.ProviderValues) (imv1.Provider, error) {

	maxSurge := intstr.FromInt32(int32(DefaultIfParamNotSet(values.ZonesCount, operation.ProvisioningParameters.Parameters.MaxSurge)))
	maxUnavailable := intstr.FromInt32(int32(DefaultIfParamNotSet(0, operation.ProvisioningParameters.Parameters.MaxUnavailable)))

	scalerMax := int32(DefaultIfParamNotSet(values.DefaultAutoScalerMax, operation.ProvisioningParameters.Parameters.AutoScalerMax))
	scalerMin := int32(DefaultIfParamNotSet(values.DefaultAutoScalerMin, operation.ProvisioningParameters.Parameters.AutoScalerMin))

	zones := values.Zones
	if s.providerSpec.ZonesDiscovery(pkg.CloudProviderFromString(operation.ProviderValues.ProviderType)) {
		zones = operation.DiscoveredZones[DefaultIfParamNotSet(values.DefaultMachineType, operation.ProvisioningParameters.Parameters.MachineType)]
		zones = zones[:values.ZonesCount]
	}
	log.Info(fmt.Sprintf("Zones for Kyma worker node pool: %v", zones))

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
				Zones:          zones,
			},
		},
	}

	if steps.IsNotSapConvergedCloud(operation.CloudProvider) {
		provider.Workers[0].Volume = &gardener.Volume{
			Type:       ptr.String(values.DiskType),
			VolumeSize: fmt.Sprintf("%dGi", values.VolumeSizeGb),
		}
	}

	additionalWorkers, err := s.workersProvider.CreateAdditionalWorkers(values, nil, operation.ProvisioningParameters.Parameters.AdditionalWorkerNodePools,
		values.Zones, operation.ProvisioningParameters.PlanID, operation.DiscoveredZones, log)
	if err != nil {
		return imv1.Provider{}, fmt.Errorf("while creating additional workers: %w", err)
	}
	provider.AdditionalWorkers = &additionalWorkers

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

	var dualStack *bool
	if networkingParams.DualStack != nil {
		cloudProvider := pkg.CloudProviderFromString(operation.ProviderValues.ProviderType)
		if !s.providerSpec.IsDualStackSupported(cloudProvider) {
			dualStack = nil
		} else {
			dualStack = networkingParams.DualStack
		}
	}

	return imv1.Networking{
		Pods:      DefaultIfParamNotSet(networking.DefaultPodsCIDR, networkingParams.PodsCidr),
		Services:  DefaultIfParamNotSet(networking.DefaultServicesCIDR, networkingParams.ServicesCidr),
		Nodes:     nodes,
		DualStack: dualStack,
		// TODO remove when KIM is ready with setting this value
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
	oidc := s.createDefaultOIDCConfig()
	oidcInput := operation.ProvisioningParameters.Parameters.OIDC

	kubernetesConfig := imv1.Kubernetes{
		Version:       ptr.String(s.config.KubernetesVersion),
		KubeAPIServer: imv1.APIServer{},
	}

	if oidcInput == nil {
		kubernetesConfig.KubeAPIServer.AdditionalOidcConfig = &[]imv1.OIDCConfig{oidc}
	} else {
		kubernetesConfig.KubeAPIServer.AdditionalOidcConfig = s.createOIDCConfigFromInput(oidcInput, oidc)
	}

	return kubernetesConfig
}

func (s *CreateRuntimeResourceStep) createDefaultOIDCConfig() imv1.OIDCConfig {
	return imv1.OIDCConfig{
		OIDCConfig: gardener.OIDCConfig{
			ClientID:       &s.oidcDefaultValues.ClientID,
			GroupsClaim:    &s.oidcDefaultValues.GroupsClaim,
			IssuerURL:      &s.oidcDefaultValues.IssuerURL,
			SigningAlgs:    s.oidcDefaultValues.SigningAlgs,
			UsernameClaim:  &s.oidcDefaultValues.UsernameClaim,
			UsernamePrefix: &s.oidcDefaultValues.UsernamePrefix,
			GroupsPrefix:   &s.oidcDefaultValues.GroupsPrefix,
		},
	}
}

func (s *CreateRuntimeResourceStep) createOIDCConfigFromInput(oidcInput *pkg.OIDCConnectDTO, defaultOIDC imv1.OIDCConfig) *[]imv1.OIDCConfig {
	if oidcInput.List != nil {
		return s.createOIDCConfigList(oidcInput.List)
	}

	if oidcInput.OIDCConfigDTO != nil {
		return &[]imv1.OIDCConfig{s.mergeOIDCConfig(defaultOIDC, oidcInput.OIDCConfigDTO)}
	}

	return &[]imv1.OIDCConfig{defaultOIDC}
}

func (s *CreateRuntimeResourceStep) createOIDCConfigList(oidcList []pkg.OIDCConfigDTO) *[]imv1.OIDCConfig {
	configs := make([]imv1.OIDCConfig, 0, len(oidcList))

	for _, oidcConfig := range oidcList {
		requiredClaims := s.parseRequiredClaims(oidcConfig.RequiredClaims)
		groupsPrefix := s.oidcDefaultValues.GroupsPrefix
		if oidcConfig.GroupsPrefix != "" {
			groupsPrefix = oidcConfig.GroupsPrefix
		}
		oidc := imv1.OIDCConfig{
			OIDCConfig: gardener.OIDCConfig{
				ClientID:       &oidcConfig.ClientID,
				IssuerURL:      &oidcConfig.IssuerURL,
				SigningAlgs:    oidcConfig.SigningAlgs,
				GroupsClaim:    &oidcConfig.GroupsClaim,
				UsernamePrefix: &oidcConfig.UsernamePrefix,
				UsernameClaim:  &oidcConfig.UsernameClaim,
				GroupsPrefix:   &groupsPrefix,
				RequiredClaims: requiredClaims,
			},
		}
		oidc.JWKS, _ = base64.StdEncoding.DecodeString(oidcConfig.EncodedJwksArray)
		configs = append(configs, oidc)
	}

	return &configs
}

func (s *CreateRuntimeResourceStep) mergeOIDCConfig(defaultOIDC imv1.OIDCConfig, inputOIDC *pkg.OIDCConfigDTO) imv1.OIDCConfig {
	if inputOIDC.ClientID != "" {
		defaultOIDC.ClientID = &inputOIDC.ClientID
	}
	if inputOIDC.GroupsClaim != "" {
		defaultOIDC.GroupsClaim = &inputOIDC.GroupsClaim
	}
	if inputOIDC.GroupsPrefix != "" {
		defaultOIDC.GroupsPrefix = &inputOIDC.GroupsPrefix
	}
	if inputOIDC.IssuerURL != "" {
		defaultOIDC.IssuerURL = &inputOIDC.IssuerURL
	}
	if len(inputOIDC.SigningAlgs) > 0 {
		defaultOIDC.SigningAlgs = inputOIDC.SigningAlgs
	}
	if inputOIDC.UsernameClaim != "" {
		defaultOIDC.UsernameClaim = &inputOIDC.UsernameClaim
	}
	if inputOIDC.UsernamePrefix != "" {
		defaultOIDC.UsernamePrefix = &inputOIDC.UsernamePrefix
	}
	if len(inputOIDC.RequiredClaims) > 0 {
		defaultOIDC.RequiredClaims = s.parseRequiredClaims(inputOIDC.RequiredClaims)
	}
	if inputOIDC.EncodedJwksArray != "" && inputOIDC.EncodedJwksArray != "-" {
		defaultOIDC.JWKS, _ = base64.StdEncoding.DecodeString(inputOIDC.EncodedJwksArray)
	}
	return defaultOIDC
}

func (s *CreateRuntimeResourceStep) parseRequiredClaims(claims []string) map[string]string {
	requiredClaims := make(map[string]string)
	if len(claims) == 1 && claims[0] == "-" {
		return requiredClaims
	}
	for _, claim := range claims {
		parts := strings.SplitN(claim, "=", 2)
		requiredClaims[parts[0]] = parts[1]
	}
	return requiredClaims
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
