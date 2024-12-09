package handlers

import (
	"fmt"
	"log/slog"
	"time"

	commonOrchestration "github.com/kyma-project/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

type Retryer struct {
	orchestrations storage.Orchestrations
	operations     storage.Operations
	queue          *process.Queue
	log            *slog.Logger
}

type clusterRetryer Retryer

func NewClusterRetryer(orchestrations storage.Orchestrations, operations storage.Operations, q *process.Queue, logger *slog.Logger) *clusterRetryer {
	return &clusterRetryer{
		orchestrations: orchestrations,
		operations:     operations,
		queue:          q,
		log:            logger,
	}
}

func (r *clusterRetryer) orchestrationRetry(o *internal.Orchestration, opsByOrch []internal.UpgradeClusterOperation, operationIDs []string) (commonOrchestration.RetryResponse, error) {
	var err error
	resp := commonOrchestration.RetryResponse{OrchestrationID: o.OrchestrationID}

	ops, invalidIDs := r.orchestrationOperationsFilter(opsByOrch, operationIDs)
	resp.InvalidOperations = invalidIDs
	if len(ops) == 0 {
		zeroValidOperationInfo(&resp, r.log)
		return resp, nil
	}

	// as failed orchestration has finished before
	// only retry the latest failed cluster upgrade operation for the same instance
	if o.State == commonOrchestration.Failed {
		var oldIDs []string
		var err error

		ops, oldIDs, err = r.latestOperationValidate(o.OrchestrationID, ops)
		if err != nil {
			return resp, err
		}
		resp.OldOperations = oldIDs

		if len(ops) == 0 {
			zeroValidOperationInfo(&resp, r.log)
			return resp, nil
		}
	}

	for _, op := range ops {
		resp.RetryShoots = append(resp.RetryShoots, op.Operation.InstanceDetails.ShootName)
	}
	resp.Msg = "retry operations are queued for processing"

	for _, op := range ops {
		o.Parameters.RetryOperation.RetryOperations = append(o.Parameters.RetryOperation.RetryOperations, op.Operation.ID)
	}

	// get orchestration state again in case in progress changed to failed, need to put in queue
	lastState, err := orchestrationStateUpdate(o, r.orchestrations, o.OrchestrationID, r.log)
	if err != nil {
		return resp, err
	}

	if lastState == commonOrchestration.Failed {
		r.queue.Add(o.OrchestrationID)
	}

	return resp, nil
}

func (r *clusterRetryer) orchestrationOperationsFilter(opsByOrch []internal.UpgradeClusterOperation, opsIDs []string) ([]internal.UpgradeClusterOperation, []string) {
	if len(opsIDs) <= 0 {
		return opsByOrch, nil
	}

	var retOps []internal.UpgradeClusterOperation
	var invalidIDs []string
	var found bool

	for _, opID := range opsIDs {
		for _, op := range opsByOrch {
			if opID == op.Operation.ID {
				retOps = append(retOps, op)
				found = true
				break
			}
		}

		if found {
			found = false
		} else {
			invalidIDs = append(invalidIDs, opID)
		}
	}

	return retOps, invalidIDs
}

func (r *clusterRetryer) latestOperationValidate(orchestrationID string, ops []internal.UpgradeClusterOperation) ([]internal.UpgradeClusterOperation, []string, error) {
	var retryOps []internal.UpgradeClusterOperation
	var oldIDs []string

	for _, op := range ops {
		instanceID := op.InstanceID

		clusterOps, err := r.operations.ListUpgradeClusterOperationsByInstanceID(instanceID)
		if err != nil {
			// fail for listing operations of one instance, then http return and report fail
			err = fmt.Errorf("while getting operations by instanceID %s: %w", instanceID, err)
			r.log.Error(err.Error())
			return nil, nil, err
		}

		var errFound, newerExist bool
		num := len(clusterOps)

		for i := 0; i < num; i++ {
			if op.CreatedAt.Before(clusterOps[i].CreatedAt) {
				if num == 1 {
					errFound = true
					break
				}

				// 'canceled' or 'canceling' newer op is not a newer op
				if clusterOps[i].State == commonOrchestration.Canceled || clusterOps[i].State == commonOrchestration.Canceling {
					continue
				}

				oldIDs = append(oldIDs, op.Operation.ID)
				newerExist = true
			}

			break
		}

		if num == 0 || errFound {
			err = fmt.Errorf("while getting operations by instanceID %s: %w", instanceID, err)
			r.log.Error(err.Error())
			return nil, nil, err
		}

		if newerExist {
			continue
		}

		retryOps = append(retryOps, op)
	}

	return retryOps, oldIDs, nil
}

func orchestrationStateUpdate(orch *internal.Orchestration, orchestrations storage.Orchestrations, orchestrationID string, log *slog.Logger) (string, error) {
	o, err := orchestrations.GetByID(orchestrationID)
	if err != nil {
		log.Error(fmt.Sprintf("while getting orchestration %s: %v", orchestrationID, err))
		return "", fmt.Errorf("while getting orchestration %s: %w", orchestrationID, err)
	}
	// last minute check in case in progress one got canceled.
	state := o.State
	if state == commonOrchestration.Canceling || state == commonOrchestration.Canceled {
		log.Info(fmt.Sprintf("orchestration %s was canceled right before retrying", orchestrationID))
		return state, fmt.Errorf("orchestration %s was canceled right before retrying", orchestrationID)
	}

	o.UpdatedAt = time.Now()
	o.Parameters.RetryOperation.RetryOperations = orch.Parameters.RetryOperation.RetryOperations
	o.Parameters.RetryOperation.Immediate = orch.Parameters.RetryOperation.Immediate
	if state == commonOrchestration.Failed {
		o.Description += ", retrying"
		o.State = commonOrchestration.Retrying
	}
	err = orchestrations.Update(*o)
	if err != nil {
		log.Error(fmt.Sprintf("while updating orchestration %s: %v", orchestrationID, err))
		return state, fmt.Errorf("while updating orchestration %s: %w", orchestrationID, err)
	}
	return state, nil
}

func zeroValidOperationInfo(resp *commonOrchestration.RetryResponse, log *slog.Logger) {
	log.Info(fmt.Sprintf("no valid operations to retry for orchestration %s", resp.OrchestrationID))
	resp.Msg = fmt.Sprintf("No valid operations to retry for orchestration %s", resp.OrchestrationID)
}
