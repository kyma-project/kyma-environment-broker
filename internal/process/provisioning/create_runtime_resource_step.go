package provisioning

import (
	"context"
	"fmt"
	"time"

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
	operationManager            *process.OperationManager
	instanceStorage             storage.Instances
	runtimeStateStorage         storage.RuntimeStates
	k8sClient                   client.Client
	kimConfig                   kim.Config
	config                      input.Config
	trialPlatformRegionMapping  map[string]string
	useSmallerMachinesForTrials bool
}

func NewCreateRuntimeResourceStep(os storage.Operations, is storage.Instances, k8sClient client.Client, kimConfig kim.Config, cfg input.Config,
	trialPlatformRegionMapping map[string]string, useSmallerMachinesForTrials bool) *CreateRuntimeResourceStep {
	return &CreateRuntimeResourceStep{
		operationManager:            process.NewOperationManager(os),
		instanceStorage:             is,
		kimConfig:                   kimConfig,
		k8sClient:                   k8sClient,
		config:                      cfg,
		trialPlatformRegionMapping:  trialPlatformRegionMapping,
		useSmallerMachinesForTrials: useSmallerMachinesForTrials,
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
		log.Infof("KIM is not enabled for plan %s, skipping", broker.PlanNamesMapping[operation.ProvisioningParameters.PlanID])
		return operation, 0, nil
	}

	kymaResourceName := operation.KymaResourceName
	kymaResourceNamespace := operation.KymaResourceNamespace

	runtimeCR, err := s.createRuntimeResourceObject(operation, kymaResourceName, kymaResourceNamespace)
	if err != nil {
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("while creating Runtime CR object: %s", err), err, log)
	}

	if s.kimConfig.DryRun {
		yaml, err := RuntimeToYaml(runtimeCR)
		if err != nil {
			log.Errorf("failed to encode Runtime CR as yaml: %s", err)
		} else {
			fmt.Println(yaml)
		}
	} else {
		err := s.k8sClient.Create(context.Background(), runtimeCR)
		if err != nil {
			log.Error("unable to create Runtime resource: %s", err)
			return s.operationManager.OperationFailed(operation, fmt.Sprintf("unable to Runtime resource: %s", err), err, log)
		}
		log.Infof("Runtime CR %s creation process finished successfully", operation.RuntimeID)
	}
	return operation, 0, nil
}

func (s *CreateRuntimeResourceStep) CreateResource(cr *imv1.Runtime) error {
	logrus.Info("Creating Runtime CR - TO BE IMPLEMENTED")
	return nil
}

func (s *CreateRuntimeResourceStep) createRuntimeResourceObject(operation internal.Operation, kymaName, kymaNamespace string) (*imv1.Runtime, error) {

	runtime := imv1.Runtime{}
	runtime.ObjectMeta.Name = operation.RuntimeID
	runtime.ObjectMeta.Namespace = kymaNamespace
	runtime.ObjectMeta.Labels = s.createLabelsForRuntime(operation, kymaName)

	// get plan specific values (like zones, default machine type etc.
	values, err := s.providerValues(&operation)
	if err != nil {
		return nil, err
	}

	providerObj, err := s.createShootProvider(&operation, values)
	if err != nil {
		return nil, err
	}

	runtime.Spec.Shoot.Provider = providerObj
	runtime.Spec.Shoot.Region = values.Region
	runtime.Spec.Shoot.PlatformRegion = operation.ProvisioningParameters.PlatformRegion

	runtime.Spec.Security = s.createSecurityConfiguration(operation)
	return &runtime, nil
}

func (s *CreateRuntimeResourceStep) createLabelsForRuntime(operation internal.Operation, kymaName string) map[string]string {
	labels := map[string]string{
		"kyma-project.io/instance-id":        operation.InstanceID,
		"kyma-project.io/runtime-id":         operation.RuntimeID,
		"kyma-project.io/broker-plan-id":     operation.ProvisioningParameters.PlanID,
		"kyma-project.io/broker-plan-name":   broker.PlanNamesMapping[operation.ProvisioningParameters.PlanID],
		"kyma-project.io/global-account-id":  operation.ProvisioningParameters.ErsContext.GlobalAccountID,
		"kyma-project.io/subaccount-id":      operation.ProvisioningParameters.ErsContext.SubAccountID,
		"kyma-project.io/shoot-name":         operation.ShootName,
		"kyma-project.io/region":             *operation.ProvisioningParameters.Parameters.Region,
		"operator.kyma-project.io/kyma-name": kymaName,
	}
	return labels
}

func (s *CreateRuntimeResourceStep) createSecurityConfiguration(operation internal.Operation) imv1.Security {
	security := imv1.Security{}
	security.Administrators = operation.ProvisioningParameters.Parameters.RuntimeAdministrators
	security.Networking.Filter.Egress.Enabled = false
	// Ingress is not supported yet, nevertheless we set it for completeness
	security.Networking.Filter.Ingress = &imv1.Ingress{Enabled: false}
	return security
}

func RuntimeToYaml(runtime *imv1.Runtime) (string, error) {
	result, err := yaml.Marshal(runtime)
	if err != nil {
		return "", err
	}
	return string(result), nil
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
	case broker.AzurePlanID:
		p = &provider.AzureInputProvider{
			MultiZone:              s.config.MultiZoneCluster,
			ProvisioningParameters: operation.ProvisioningParameters,
		}
	case broker.GCPPlanID:
		p = &provider.GCPInputProvider{
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
				UseSmallerMachineTypes: s.useSmallerMachinesForTrials,
				ProvisioningParameters: operation.ProvisioningParameters,
			}
		case internal.GCP:
			p = &provider.GCPTrialInputProvider{
				PlatformRegionMapping:  s.trialPlatformRegionMapping,
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

func DefaultIfParamNotSet[T interface{}](d T, param *T) T {
	if param == nil {
		return d
	}
	return *param
}
