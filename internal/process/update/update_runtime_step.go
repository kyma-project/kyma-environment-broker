package update

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/provider"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"

	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"
	"github.com/kyma-project/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/kyma-environment-broker/internal/process/provisioning"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type UpdateRuntimeStep struct {
	operationManager           *process.OperationManager
	k8sClient                  client.Client
	delay                      time.Duration
	config                     input.Config
	useSmallerMachineTypes     bool
	trialPlatformRegionMapping map[string]string
}

func NewUpdateRuntimeStep(os storage.Operations, k8sClient client.Client, delay time.Duration, cfg input.Config, useSmallerMachines bool, trialPlatformRegionMapping map[string]string) *UpdateRuntimeStep {
	step := &UpdateRuntimeStep{
		k8sClient:                  k8sClient,
		delay:                      delay,
		config:                     cfg,
		useSmallerMachineTypes:     useSmallerMachines,
		trialPlatformRegionMapping: trialPlatformRegionMapping,
	}
	step.operationManager = process.NewOperationManager(os, step.Name(), kebError.InfrastructureManagerDependency)
	return step
}

func (s *UpdateRuntimeStep) Name() string {
	return "Update_Runtime_Resource"
}

func (s *UpdateRuntimeStep) Run(operation internal.Operation, log *slog.Logger) (internal.Operation, time.Duration, error) {
	// Check if the runtime exists

	var runtime = imv1.Runtime{}
	err := s.k8sClient.Get(context.Background(), client.ObjectKey{Name: operation.GetRuntimeResourceName(), Namespace: operation.GetRuntimeResourceNamespace()}, &runtime)
	if errors.IsNotFound(err) {
		// todo: after the switch to KIM, this should throw an error
		log.Info("Runtime not found, skipping")
		return operation, 0, nil
	}

	// Update the runtime

	runtime.Spec.Shoot.Provider.Workers[0].Machine.Type = provisioning.DefaultIfParamNotSet(runtime.Spec.Shoot.Provider.Workers[0].Machine.Type, operation.UpdatingParameters.MachineType)
	runtime.Spec.Shoot.Provider.Workers[0].Minimum = int32(provisioning.DefaultIfParamNotSet(int(runtime.Spec.Shoot.Provider.Workers[0].Minimum), operation.UpdatingParameters.AutoScalerMin))
	runtime.Spec.Shoot.Provider.Workers[0].Maximum = int32(provisioning.DefaultIfParamNotSet(int(runtime.Spec.Shoot.Provider.Workers[0].Maximum), operation.UpdatingParameters.AutoScalerMax))

	maxSurge := intstr.FromInt32(int32(provisioning.DefaultIfParamNotSet(runtime.Spec.Shoot.Provider.Workers[0].MaxSurge.IntValue(), operation.UpdatingParameters.MaxSurge)))
	runtime.Spec.Shoot.Provider.Workers[0].MaxSurge = &maxSurge
	maxUnavailable := intstr.FromInt32(int32(provisioning.DefaultIfParamNotSet(runtime.Spec.Shoot.Provider.Workers[0].MaxUnavailable.IntValue(), operation.UpdatingParameters.MaxUnavailable)))
	runtime.Spec.Shoot.Provider.Workers[0].MaxUnavailable = &maxUnavailable

	if operation.UpdatingParameters.AdditionalWorkerNodePools != nil {
		values, err := provider.GetPlanSpecificValues(&operation, s.config.MultiZoneCluster, s.config.DefaultTrialProvider, s.useSmallerMachineTypes, s.trialPlatformRegionMapping,
			s.config.DefaultGardenerShootPurpose, s.config.ControlPlaneFailureTolerance)
		if err != nil {
			return s.operationManager.OperationFailed(operation, fmt.Sprintf("while calculating plan specific values: %s", err), err, log)
		}

		currentAdditionalWorkers := make(map[string]gardener.Worker)
		if runtime.Spec.Shoot.Provider.AdditionalWorkers != nil {
			for _, worker := range *runtime.Spec.Shoot.Provider.AdditionalWorkers {
				currentAdditionalWorkers[worker.Name] = worker
			}
		}

		additionalWorkers := provisioning.CreateAdditionalWorkers(s.config, values, currentAdditionalWorkers, operation.UpdatingParameters.AdditionalWorkerNodePools, runtime.Spec.Shoot.Provider.Workers[0].Zones)
		runtime.Spec.Shoot.Provider.AdditionalWorkers = &additionalWorkers
	}

	if operation.UpdatingParameters.OIDC != nil {
		input := operation.UpdatingParameters.OIDC
		if s.config.UseMainOIDC {
			if len(input.SigningAlgs) > 0 {
				runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.SigningAlgs = input.SigningAlgs
			}
			if input.ClientID != "" {
				runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.ClientID = &input.ClientID
			}
			if input.IssuerURL != "" {
				runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.IssuerURL = &input.IssuerURL
			}
			if input.GroupsClaim != "" {
				runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.GroupsClaim = &input.GroupsClaim
			}
			if input.UsernamePrefix != "" {
				runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.UsernamePrefix = &input.UsernamePrefix
			}
			if input.UsernameClaim != "" {
				runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig.UsernameClaim = &input.UsernameClaim
			}
		}

		if s.config.UseAdditionalOIDC {
			if runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig == nil {
				runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig = &[]gardener.OIDCConfig{{}}
			}
			if len(input.SigningAlgs) > 0 {
				(*runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)[0].SigningAlgs = input.SigningAlgs
			}
			if input.ClientID != "" {
				(*runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)[0].ClientID = &input.ClientID
			}
			if input.IssuerURL != "" {
				(*runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)[0].IssuerURL = &input.IssuerURL
			}
			if input.GroupsClaim != "" {
				(*runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)[0].GroupsClaim = &input.GroupsClaim
			}
			if input.UsernamePrefix != "" {
				(*runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)[0].UsernamePrefix = &input.UsernamePrefix
			}
			if input.UsernameClaim != "" {
				(*runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig)[0].UsernameClaim = &input.UsernameClaim
			}
		}
	}

	// operation.ProvisioningParameters were calculated and joined across provisioning and all update operations
	if len(operation.ProvisioningParameters.Parameters.RuntimeAdministrators) != 0 {
		// prepare new admins list for existing runtime
		newAdministrators := make([]string, 0, len(operation.ProvisioningParameters.Parameters.RuntimeAdministrators))
		newAdministrators = append(newAdministrators, operation.ProvisioningParameters.Parameters.RuntimeAdministrators...)

		runtime.Spec.Security.Administrators = newAdministrators
	} else {
		if operation.ProvisioningParameters.ErsContext.UserID != "" {
			// get default admin (user_id from provisioning operation)
			runtime.Spec.Security.Administrators = []string{operation.ProvisioningParameters.ErsContext.UserID}
		} else {
			// some old clusters does not have an user_id
			runtime.Spec.Security.Administrators = []string{}
		}
	}

	if operation.ProvisioningParameters.ErsContext.LicenseType != nil {
		disabled := *operation.ProvisioningParameters.ErsContext.DisableEnterprisePolicyFilter()
		runtime.Spec.Security.Networking.Filter.Egress.Enabled = !disabled
	}

	err = s.k8sClient.Update(context.Background(), &runtime)
	if err != nil {
		return s.operationManager.RetryOperation(operation, "unable to update runtime", err, 10*time.Second, 1*time.Minute, log)
	}

	// this sleep is needed to wait for the runtime to be updated by the infrastructure manager with state PENDING,
	// then we can wait for the state READY in the next step
	time.Sleep(s.delay)

	return operation, 0, nil
}
