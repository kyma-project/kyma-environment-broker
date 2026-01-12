package broker

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/dashboard"

	"github.com/kyma-project/kyma-environment-broker/internal/kubeconfig"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"

	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/pivotal-cf/brokerapi/v12/domain/apiresponses"
)

type GetInstanceEndpoint struct {
	config            Config
	instancesStorage  storage.Instances
	operationsStorage storage.Provisioning
	brokerURL         string
	kcBuilder         kubeconfig.KcBuilder
	log               *slog.Logger
}

func NewGetInstance(cfg Config,
	instancesStorage storage.Instances,
	operationsStorage storage.Provisioning,
	kcBuilder kubeconfig.KcBuilder,
	log *slog.Logger,
) *GetInstanceEndpoint {
	return &GetInstanceEndpoint{
		config:            cfg,
		instancesStorage:  instancesStorage,
		operationsStorage: operationsStorage,
		kcBuilder:         kcBuilder,
		log:               log.With("service", "GetInstanceEndpoint"),
	}
}

// GetInstance fetches information about a service instance
// GET /v2/service_instances/{instance_id}
func (b *GetInstanceEndpoint) GetInstance(_ context.Context, instanceID string, _ domain.FetchInstanceDetails) (domain.GetInstanceDetailsSpec, error) {
	logger := b.log.With("instanceID", instanceID)
	logger.Info("GetInstance called")

	instance, err := b.instancesStorage.GetByID(instanceID)
	if err != nil {
		statusCode := http.StatusNotFound
		if !dberr.IsNotFound(err) {
			statusCode = http.StatusInternalServerError
			return domain.GetInstanceDetailsSpec{}, apiresponses.NewFailureResponse(fmt.Errorf("failed to get instanceID %s", instanceID), statusCode, fmt.Sprintf("failed to get instanceID %s", instanceID))
		}
		return domain.GetInstanceDetailsSpec{}, apiresponses.NewFailureResponse(fmt.Errorf("instance with instanceID %s does not exist", instanceID), statusCode, fmt.Sprintf("instance with instanceID %s does not exist", instanceID))
	}

	// check if provisioning still in progress
	op, err := b.operationsStorage.GetProvisioningOperationByInstanceID(instanceID)
	if err != nil {
		return domain.GetInstanceDetailsSpec{}, apiresponses.NewFailureResponse(fmt.Errorf("failed to get operation for instanceID %s", instanceID), http.StatusNotFound, fmt.Sprintf("failed to get operation for instanceID %s", instanceID))
	} else if op.State == domain.InProgress || op.State == domain.Failed {
		err = fmt.Errorf("provisioning of instanceID %s %s", instanceID, op.State)
		return domain.GetInstanceDetailsSpec{}, apiresponses.NewFailureResponse(err, http.StatusNotFound, err.Error())
	}

	if !instance.DeletedAt.IsZero() {
		return domain.GetInstanceDetailsSpec{}, apiresponses.NewFailureResponse(fmt.Errorf("instance with instanceID %s does not exist", instanceID), http.StatusNotFound, fmt.Sprintf("instance with instanceID %s does not exist", instanceID))
	}

	parameters := b.prepareParametersToReturn(instance.Parameters)

	spec := domain.GetInstanceDetailsSpec{
		ServiceID:    instance.ServiceID,
		PlanID:       instance.ServicePlanID,
		DashboardURL: dashboard.ProvideURL(instance, op),
		Parameters:   parameters,
		Metadata: domain.InstanceMetadata{
			Labels: ResponseLabels(*instance, b.config.URL, b.kcBuilder),
		},
	}

	if instance.ServicePlanID == TrialPlanID {
		spec.Metadata.Labels = ResponseLabelsWithExpirationInfo(*instance, b.config.URL, b.config.TrialDocsURL, trialDocsKey, trialExpireDuration, trialExpiryDetailsKey, trialExpiredInfoFormat, b.kcBuilder)
	}

	if instance.ServicePlanID == FreemiumPlanID {
		spec.Metadata.Labels = ResponseLabelsWithExpirationInfo(*instance, b.config.URL, b.config.FreeDocsURL, freeDocsKey, b.config.FreeExpirationPeriod, freeExpiryDetailsKey, freeExpiredInfoFormat, b.kcBuilder)
	}

	return spec, nil
}

func (b *GetInstanceEndpoint) prepareParametersToReturn(parameters internal.ProvisioningParameters) internal.ProvisioningParameters {
	parameters.Parameters.Kubeconfig = ""
	parameters.ErsContext.SMOperatorCredentials = nil
	return parameters
}
