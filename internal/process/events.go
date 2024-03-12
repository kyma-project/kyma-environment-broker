package process

import (
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/pivotal-cf/brokerapi/v8/domain"
)

type StepProcessed struct {
	StepName string
	Duration time.Duration
	When     time.Duration
	Error    error
}

type ProvisioningStepProcessed struct {
	StepProcessed
	Operation internal.ProvisioningOperation
}

type UpdatingStepProcessed struct {
	StepProcessed
	OldOperation internal.UpdatingOperation
	Operation    internal.UpdatingOperation
}

type DeprovisioningStepProcessed struct {
	StepProcessed
	OldOperation internal.DeprovisioningOperation
	Operation    internal.DeprovisioningOperation
}

type UpgradeKymaStepProcessed struct {
	StepProcessed
	OldOperation internal.UpgradeKymaOperation
	Operation    internal.UpgradeKymaOperation
}

type UpgradeClusterStepProcessed struct {
	StepProcessed
	OldOperation internal.UpgradeClusterOperation
	Operation    internal.UpgradeClusterOperation
}

type ProvisioningSucceeded struct {
	Operation internal.ProvisioningOperation
}

type OperationStepProcessed struct {
	StepProcessed
	OldOperation internal.Operation
	Operation    internal.Operation
}

type OperationSucceeded struct {
	Operation internal.Operation
}

type OperationCounting struct {
	OpId    string
	PlanID  broker.PlanID
	OpState domain.LastOperationState
	OpType  internal.OperationType
}

type DeprovisioningSucceeded struct {
	Operation internal.DeprovisioningOperation
}