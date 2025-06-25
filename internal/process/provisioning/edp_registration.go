package provisioning

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/edp"
	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

//go:generate mockery --name=EDPClient --output=automock --outpkg=automock --case=underscore
type EDPClient interface {
	CreateDataTenant(data edp.DataTenantPayload, log *slog.Logger) error
	CreateMetadataTenant(name, env string, data edp.MetadataTenantPayload, log *slog.Logger) error

	DeleteDataTenant(name, env string, log *slog.Logger) error
	DeleteMetadataTenant(name, env, key string, log *slog.Logger) error
}

type EDPRegistrationStep struct {
	operationManager *process.OperationManager
	client           EDPClient
	config           edp.Config
}

const (
	edpRetryInterval = 10 * time.Second
	edpRetryTimeout  = 30 * time.Minute
)

func NewEDPRegistrationStep(os storage.Operations, client EDPClient, config edp.Config) *EDPRegistrationStep {
	step := &EDPRegistrationStep{
		client: client,
		config: config,
	}
	step.operationManager = process.NewOperationManager(os, step.Name(), kebError.EDPDependency)
	return step
}

func (s *EDPRegistrationStep) Name() string {
	return "EDP_Registration"
}

func (s *EDPRegistrationStep) Run(operation internal.Operation, log *slog.Logger) (internal.Operation, time.Duration, error) {
	if operation.EDPCreated {
		return operation, 0, nil
	}
	subAccountID := strings.ToLower(operation.ProvisioningParameters.ErsContext.SubAccountID)

	log.Info(fmt.Sprintf("Create DataTenant for %s subaccount (env=%s)", subAccountID, s.config.Environment))
	err := s.client.CreateDataTenant(edp.DataTenantPayload{
		Name:        subAccountID,
		Environment: s.config.Environment,
		Secret:      s.generateSecret(subAccountID, s.config.Environment),
	}, log.With("service", "edpClient"))
	if err != nil {
		if edp.IsConflictError(err) {
			log.Warn("Data Tenant already exists, deleting")
			return s.handleConflict(operation, log)
		}
		return s.handleError(operation, err, log, "cannot create DataTenant")
	}

	log.Info(fmt.Sprintf("Create DataTenant metadata for %s subaccount", subAccountID))
	for key, value := range map[string]string{
		edp.MaasConsumerEnvironmentKey: s.selectEnvironmentKey(operation.ProvisioningParameters.PlatformRegion, log),
		edp.MaasConsumerRegionKey:      operation.ProvisioningParameters.PlatformRegion,
		edp.MaasConsumerSubAccountKey:  subAccountID,
		edp.MaasConsumerServicePlan:    SelectServicePlan(operation.ProvisioningParameters.PlanID),
	} {
		payload := edp.MetadataTenantPayload{
			Key:   key,
			Value: value,
		}
		log.Info(fmt.Sprintf("Sending metadata %s: %s", payload.Key, payload.Value))
		err = s.client.CreateMetadataTenant(subAccountID, s.config.Environment, payload, log.With("service", "edpClient"))
		if err != nil {
			if edp.IsConflictError(err) {
				log.Warn("Metadata already exists, deleting")
				return s.handleConflict(operation, log)
			}
			return s.handleError(operation, err, log, fmt.Sprintf("cannot create DataTenant metadata %s", key))
		}
	}

	newOp, repeat, _ := s.operationManager.UpdateOperation(operation, func(op *internal.Operation) {
		op.EDPCreated = true
	}, log)
	if repeat != 0 {
		log.Error("cannot update operation")
		return s.operationManager.RetryOperation(newOp, "cannot update operation", err, dbRetryInterval, dbRetryTimeout, log)
	}

	return newOp, 0, nil
}

func (s *EDPRegistrationStep) handleError(operation internal.Operation, err error, log *slog.Logger, msg string) (internal.Operation, time.Duration, error) {
	log.Warn(fmt.Sprintf("%s: %s", msg, err))

	if kebError.IsTemporaryError(err) {
		log.Warn(fmt.Sprintf("request to EDP failed: %s. Retry...", err))
		if s.config.Required {
			return s.operationManager.RetryOperation(operation, "request to EDP failed", err, edpRetryInterval, edpRetryTimeout, log)
		} else {
			return s.operationManager.RetryOperationWithoutFail(operation, s.Name(), "request to EDP failed", edpRetryInterval, edpRetryTimeout, log, err)
		}
	}

	if s.config.Required {
		return s.operationManager.OperationFailed(operation, msg, err, log)
	} else {
		log.Warn(fmt.Sprintf("Step %s failed. Step is not required. Quit step.", s.Name()))
		return operation, 0, nil
	}
}

func (s *EDPRegistrationStep) selectEnvironmentKey(region string, log *slog.Logger) string {
	parts := strings.Split(region, "-")
	switch parts[0] {
	case "cf":
		return "CF"
	case "k8s":
		return "KUBERNETES"
	case "neo":
		return "NEO"
	default:
		log.Warn(fmt.Sprintf("region %s does not fit any of the options, default CF is used", region))
		return "CF"
	}
}

func SelectServicePlan(planID string) string {
	switch planID {
	case broker.FreemiumPlanID:
		return "free"
	case broker.AzureLitePlanID:
		return "tdd"
	case broker.BuildRuntimeAWSPlanID, broker.BuildRuntimeGCPPlanID, broker.BuildRuntimeAzurePlanID, broker.PreviewPlanID:
		return "build-runtime"
	default:
		return "standard"
	}
}

// generateSecret generates secret during dataTenant creation, at this moment the secret is not needed
// except required parameter
func (s *EDPRegistrationStep) generateSecret(name, env string) string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s%s", name, env)))
}

func (s *EDPRegistrationStep) handleConflict(operation internal.Operation, log *slog.Logger) (internal.Operation, time.Duration, error) {
	for _, key := range []string{
		edp.MaasConsumerEnvironmentKey,
		edp.MaasConsumerRegionKey,
		edp.MaasConsumerSubAccountKey,
		edp.MaasConsumerServicePlan,
	} {
		log.Info(fmt.Sprintf("Deleting DataTenant metadata for subaccount %s (env=%s): %s", operation.SubAccountID, s.config.Environment, key))
		err := s.client.DeleteMetadataTenant(operation.SubAccountID, s.config.Environment, key, log.With("service", "edpClient"))
		if err != nil {
			return s.handleError(operation, err, log, fmt.Sprintf("cannot remove DataTenant metadata with key: %s", key))
		}
	}

	log.Info(fmt.Sprintf("Deleting DataTenant for subaccount %s (env=%s)", operation.SubAccountID, s.config.Environment))
	err := s.client.DeleteDataTenant(operation.SubAccountID, s.config.Environment, log.With("service", "edpClient"))
	if err != nil {
		return s.handleError(operation, err, log, "cannot remove DataTenant")
	}

	log.Info("Retrying...")
	// CAVEAT this retry operation is guarded by operation timeout at staged manager level, and could fail the operation eventually
	return operation, time.Second, nil
}
