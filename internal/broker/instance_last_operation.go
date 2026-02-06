package broker

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/pivotal-cf/brokerapi/v12/domain/apiresponses"
)

type LastOperationEndpoint struct {
	operationStorage  storage.Operations
	instancesArchived storage.InstancesArchived

	log *slog.Logger
}

func NewLastOperation(os storage.Operations, ia storage.InstancesArchived, log *slog.Logger) *LastOperationEndpoint {
	return &LastOperationEndpoint{
		operationStorage:  os,
		instancesArchived: ia,
		log:               log.With("service", "LastOperationEndpoint"),
	}
}

// LastOperation fetches last operation state for a service instance
//
//	GET /v2/service_instances/{instance_id}/last_operation
func (b *LastOperationEndpoint) LastOperation(ctx context.Context, instanceID string, details domain.PollDetails) (domain.LastOperation, error) {
	logger := b.log.With("instanceID", instanceID).With("operationID", details.OperationData)

	if details.OperationData == "" {
		lastOp, err := b.operationStorage.GetLastOperationByTypes(
			instanceID,
			[]internal.OperationType{
				internal.OperationTypeProvision,
				internal.OperationTypeDeprovision,
				internal.OperationTypeUpdate,
			},
		)
		if err != nil {
			statusCode := http.StatusInternalServerError
			if dberr.IsNotFound(err) {
				return b.responseFromInstanceArchived(instanceID, logger)
			}
			logger.Error(fmt.Sprintf("cannot get operation from storage: %s", err))
			return domain.LastOperation{}, apiresponses.NewFailureResponse(err, statusCode,
				"while getting last operation from storage")
		}
		return domain.LastOperation{
			State:       mapStateToOSBCompliantState(lastOp.State),
			Description: lastOp.Description,
		}, nil
	}

	operation, err := b.operationStorage.GetOperationByID(details.OperationData)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if dberr.IsNotFound(err) {
			return b.responseFromInstanceArchived(instanceID, logger)
		}
		logger.Error(fmt.Sprintf("cannot get operation from storage: %s", err))
		return domain.LastOperation{}, apiresponses.NewFailureResponse(err, statusCode,
			"while getting operation from storage")
	}

	if operation.InstanceID != instanceID {
		err := fmt.Errorf("operation exists, but instanceID is invalid")
		logger.Error(err.Error())
		return domain.LastOperation{}, apiresponses.NewFailureResponse(err, http.StatusBadRequest, err.Error())
	}

	return domain.LastOperation{
		State:       mapStateToOSBCompliantState(operation.State),
		Description: operation.Description,
	}, nil
}

func (b *LastOperationEndpoint) responseFromInstanceArchived(instanceID string, logger *slog.Logger) (domain.LastOperation, error) {
	_, err := b.instancesArchived.GetByInstanceID(instanceID)

	switch {
	case err == nil:
		return domain.LastOperation{
			State:       domain.Succeeded,
			Description: "Operation succeeded. The instance was deprovisioned.",
		}, nil
	case dberr.IsNotFound(err):
		return domain.LastOperation{}, apiresponses.NewFailureResponse(fmt.Errorf("Operation not found"), http.StatusNotFound, "Instance not found")
	default:
		logger.Error(fmt.Sprintf("unable to get instance from archived storage: %s", err.Error()))
		return domain.LastOperation{}, apiresponses.NewFailureResponse(err, http.StatusInternalServerError, "")
	}
}

func mapStateToOSBCompliantState(opState domain.LastOperationState) domain.LastOperationState {
	switch opState {
	case internal.OperationStatePending, internal.OperationStateRetrying:
		return domain.InProgress
	case internal.OperationStateCanceled, internal.OperationStateCanceling:
		return domain.Succeeded
	default:
		return opState
	}
}
