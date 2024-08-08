package provisioning

import (
	"context"
	"fmt"
	"time"

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

	"github.com/kyma-project/kyma-environment-broker/internal/kim"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

type CreateRuntimeResourceStep struct {
	operationManager           *process.OperationManager
	instanceStorage            storage.Instances
	runtimeStateStorage        storage.RuntimeStates
	k8sClient                  client.Client
	kimConfig                  kim.Config
	config                     input.Config
	trialPlatformRegionMapping map[string]string
	useSmallerMachineTypes     bool
}

func NewCreateRuntimeResourceStep(os storage.Operations, is storage.Instances, k8sClient client.Client, kimConfig kim.Config, cfg input.Config,
	trialPlatformRegionMapping map[string]string, useSmallerMachines bool) *CreateRuntimeResourceStep {
	return &CreateRuntimeResourceStep{
		operationManager:           process.NewOperationManager(os),
		instanceStorage:            is,
		kimConfig:                  kimConfig,
		k8sClient:                  k8sClient,
		config:                     cfg,
		trialPlatformRegionMapping: trialPlatformRegionMapping,
		useSmallerMachineTypes:     useSmallerMachines,
	}
}

func (s *CreateRuntimeResourceStep) Name() string {
	return "Create_Runtime_Resource"
}

func (s *CreateRuntimeResourceStep) Run(operation internal.Operation, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	if time.Since(operation.UpdatedAt) > CreateRuntimeTimeout {
		log.Infof("operation has reached the time limit: updated operation time: %s", operation.UpdatedAt)
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("operation has reached the time limit: %s", CreateRuntimeTimeout), nil, log)
	}

	if !s.kimConfig.IsEnabledForPlan(broker.PlanNamesMapping[operation.ProvisioningParameters.PlanID]) {
		if !s.kimConfig.Enabled {
			log.Infof("KIM is not enabled, skipping")
			return operation, 0, nil
		}
		log.Infof("KIM is not enabled for plan %s, skipping", broker.PlanNamesMapping[operation.ProvisioningParameters.PlanID])
		return operation, 0, nil
	}

	kymaResourceName := operation.KymaResourceName
	kymaResourceNamespace := operation.KymaResourceNamespace
	runtimeResourceName := steps.KymaRuntimeResourceName(operation)
	log.Infof("KymaResourceName: %s, KymaResourceNamespace: %s, RuntimeResourceName: %s", kymaResourceName, kymaResourceNamespace, runtimeResourceName)

	if s.kimConfig.DryRun {
		runtimeCR := &imv1.Runtime{}
		err := s.updateRuntimeResourceObject(runtimeCR, operation, runtimeResourceName, kymaResourceName, kymaResourceNamespace)
		if err != nil {
			return s.operationManager.OperationFailed(operation, fmt.Sprintf("while updating Runtime resource object: %s", err), err, log)
		}
		yaml, err := RuntimeToYaml(runtimeCR)
		if err != nil {
			log.Errorf("failed to encode Runtime resource as yaml: %s", err)
		} else {
			fmt.Println(yaml)
		}
	} else {
		runtimeCR, err := s.getEmptyOrExistingRuntimeResource(runtimeResourceName, kymaResourceNamespace)
		if err != nil {
			log.Errorf("unable to get Runtime resource %s/%s", operation.KymaResourceNamespace, runtimeResourceName)
			return s.operationManager.RetryOperation(operation, "unable to get Runtime resource", err, 3*time.Second, 20*time.Second, log)
		}
		if runtimeCR.GetResourceVersion() != "" {
			log.Infof("Runtime resource already created %s/%s: ", operation.KymaResourceNamespace, runtimeResourceName)
			return operation, 0, nil
		} else {
			err := s.updateRuntimeResourceObject(runtimeCR, operation, runtimeResourceName, kymaResourceName, kymaResourceNamespace)
			if err != nil {
				return s.operationManager.OperationFailed(operation, fmt.Sprintf("while creating Runtime CR object: %s", err), err, log)
			}
			err = s.k8sClient.Create(context.Background(), runtimeCR)
			if err != nil {
				log.Errorf("unable to create Runtime resource: %s/%s: %s", operation.KymaResourceNamespace, runtimeResourceName, err.Error())
				return s.operationManager.RetryOperation(operation, "unable to create Runtime resource", err, 3*time.Second, 20*time.Second, log)
			}
		}
		log.Infof("Runtime resource %s/%s creation process finished successfully", operation.KymaResourceNamespace, runtimeResourceName)
	}
	return operation, 0, nil
}

func (s *CreateRuntimeResourceStep) updateRuntimeResourceObject(runtime *imv1.Runtime, operation internal.Operation, runtimeName, kymaName, kymaNamespace string) error {

	// get plan specific values (like zones, default machine type etc.
	values, err := s.providerValues(&operation)
	if err != nil {
		return err
	}
	runtime.ObjectMeta.Name = runtimeName
	runtime.ObjectMeta.Namespace = kymaNamespace
	runtime.ObjectMeta.Labels = s.createLabelsForRuntime(operation, kymaName, values.Region)

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
	runtime.Spec.Shoot.ControlPlane.HighAvailability = s.createHighAvailabilityConfiguration()
	runtime.Spec.Shoot.EnforceSeedLocation = operation.ProvisioningParameters.Parameters.ShootAndSeedSameRegion
	runtime.Spec.Security = s.createSecurityConfiguration(operation)
	runtime.Spec.Shoot.Networking = s.createNetworkingConfiguration(operation)
	return nil
}

func (s *CreateRuntimeResourceStep) createLabelsForRuntime(operation internal.Operation, kymaName string, region string) map[string]string {
	labels := map[string]string{
		"kyma-project.io/instance-id":        operation.InstanceID,
		"kyma-project.io/runtime-id":         operation.RuntimeID,
		"kyma-project.io/broker-plan-id":     operation.ProvisioningParameters.PlanID,
		"kyma-project.io/broker-plan-name":   broker.PlanNamesMapping[operation.ProvisioningParameters.PlanID],
		"kyma-project.io/global-account-id":  operation.ProvisioningParameters.ErsContext.GlobalAccountID,
		"kyma-project.io/subaccount-id":      operation.ProvisioningParameters.ErsContext.SubAccountID,
		"kyma-project.io/shoot-name":         operation.ShootName,
		"kyma-project.io/region":             region,
		"operator.kyma-project.io/kyma-name": kymaName,
	}
	if s.kimConfig.ViewOnly {
		labels["kyma-project.io/controlled-by-provisioner"] = "true"
	}
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

	security.Networking.Filter.Egress.Enabled = false
	// Ingress is not supported yet, nevertheless we set it for completeness
	security.Networking.Filter.Ingress = &imv1.Ingress{Enabled: false}
	return security
}

func (s *CreateRuntimeResourceStep) createShootProvider(operation *internal.Operation, values provider.Values) (imv1.Provider, error) {

	maxSurge := intstr.FromInt32(int32(DefaultIfParamNotSet(values.ZonesCount, operation.ProvisioningParameters.Parameters.MaxSurge)))
	maxUnavailable := intstr.FromInt32(int32(DefaultIfParamNotSet(0, operation.ProvisioningParameters.Parameters.MaxUnavailable)))

	max := int32(DefaultIfParamNotSet(values.DefaultAutoScalerMax, operation.ProvisioningParameters.Parameters.AutoScalerMax))
	min := int32(DefaultIfParamNotSet(values.DefaultAutoScalerMin, operation.ProvisioningParameters.Parameters.AutoScalerMin))

	providerObj := imv1.Provider{
		Type: values.ProviderType,
		Workers: []gardener.Worker{
			{
				Machine: gardener.Machine{
					Type: DefaultIfParamNotSet(values.DefaultMachineType, operation.ProvisioningParameters.Parameters.MachineType),
					Image: &gardener.ShootMachineImage{
						Name:    s.config.MachineImage,
						Version: &s.config.MachineImageVersion,
					},
				},
				Maximum:        max,
				Minimum:        min,
				MaxSurge:       &maxSurge,
				MaxUnavailable: &maxUnavailable,
				Zones:          values.Zones,
			},
		},
	}
	return providerObj, nil
}

type Provider interface {
	Provide() provider.Values
}

func (s *CreateRuntimeResourceStep) providerValues(operation *internal.Operation) (provider.Values, error) {
	var p Provider
	switch operation.ProvisioningParameters.PlanID {
	case broker.AWSPlanID:
		p = &provider.AWSInputProvider{
			MultiZone:              s.config.MultiZoneCluster,
			ProvisioningParameters: operation.ProvisioningParameters,
		}
	case broker.PreviewPlanID:
		p = &provider.AWSInputProvider{
			MultiZone:              s.config.MultiZoneCluster,
			ProvisioningParameters: operation.ProvisioningParameters,
		}
	case broker.AzurePlanID:
		p = &provider.AzureInputProvider{
			MultiZone:              s.config.MultiZoneCluster,
			ProvisioningParameters: operation.ProvisioningParameters,
		}
	case broker.AzureLitePlanID:
		p = &provider.AzureLiteInputProvider{
			UseSmallerMachineTypes: s.useSmallerMachineTypes,
			ProvisioningParameters: operation.ProvisioningParameters,
		}
	case broker.GCPPlanID:
		p = &provider.GCPInputProvider{
			MultiZone:              s.config.MultiZoneCluster,
			ProvisioningParameters: operation.ProvisioningParameters,
		}
	case broker.FreemiumPlanID:
		switch operation.ProvisioningParameters.PlatformProvider {
		case internal.AWS:
			p = &provider.AWSFreemiumInputProvider{
				UseSmallerMachineTypes: s.useSmallerMachineTypes,
				ProvisioningParameters: operation.ProvisioningParameters,
			}
		case internal.Azure:
			p = &provider.AzureFreemiumInputProvider{
				UseSmallerMachineTypes: s.useSmallerMachineTypes,
				ProvisioningParameters: operation.ProvisioningParameters,
			}
		default:
			return provider.Values{}, fmt.Errorf("freemium provider for '%s' is not supported", operation.ProvisioningParameters.PlatformProvider)
		}
	case broker.SapConvergedCloudPlanID:
		p = &provider.SapConvergedCloudInputProvider{
			MultiZone:              s.config.MultiZoneCluster,
			ProvisioningParameters: operation.ProvisioningParameters,
		}
	case broker.TrialPlanID:
		var trialProvider internal.CloudProvider
		if operation.ProvisioningParameters.Parameters.Provider == nil {
			trialProvider = s.config.DefaultTrialProvider
		} else {
			trialProvider = *operation.ProvisioningParameters.Parameters.Provider
		}
		switch trialProvider {
		case internal.AWS:
			p = &provider.AWSTrialInputProvider{
				PlatformRegionMapping:  s.trialPlatformRegionMapping,
				UseSmallerMachineTypes: s.useSmallerMachineTypes,
				ProvisioningParameters: operation.ProvisioningParameters,
			}
		case internal.GCP:
			p = &provider.GCPTrialInputProvider{
				PlatformRegionMapping:  s.trialPlatformRegionMapping,
				ProvisioningParameters: operation.ProvisioningParameters,
			}
		case internal.Azure:
			p = &provider.AzureTrialInputProvider{
				PlatformRegionMapping:  s.trialPlatformRegionMapping,
				UseSmallerMachineTypes: s.useSmallerMachineTypes,
				ProvisioningParameters: operation.ProvisioningParameters,
			}
		default:
			return provider.Values{}, fmt.Errorf("trial provider for %s not yet implemented", trialProvider)
		}

		// todo: implement for all plans
	default:
		return provider.Values{}, fmt.Errorf("plan %s not supported", operation.ProvisioningParameters.PlanID)
	}
	return p.Provide(), nil
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
		networkingParams = &internal.NetworkingDTO{}
	}

	nodes := networkingParams.NodesCidr
	if nodes == "" {
		nodes = networking.DefaultNodesCIDR
	}

	return imv1.Networking{
		Pods:     DefaultIfParamNotSet(networking.DefaultPodsCIDR, networkingParams.PodsCidr),
		Services: DefaultIfParamNotSet(networking.DefaultServicesCIDR, networkingParams.ServicesCidr),
		Nodes:    nodes,
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
