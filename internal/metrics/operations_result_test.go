package metrics

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/event"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

const (
	operationsCount = 1000
)

func TestOperationsResult(t *testing.T) {
	testName := fmt.Sprintf("%d metrics should be published with 1 or 0", operationsCount)
	t.Run(testName, func(t *testing.T) {

		log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))

		operations := storage.NewMemoryStorage().Operations()

		insertRandomOperations(t, operations, operationsCount)

		resultsCollector := NewOperationsResults(
			operations, Config{
				Enabled:                        true,
				OperationResultPollingInterval: 5 * time.Minute,
				OperationStatsPollingInterval:  5 * time.Minute,
				OperationResultRetentionPeriod: 24 * time.Hour,
			}, log,
		)

		eventBroker := event.NewPubSub(log)
		eventBroker.Subscribe(process.OperationFinished{}, resultsCollector.Handler)

		ops, err := operations.GetAllOperations()
		assert.NoError(t, err)
		assert.Equal(t, operationsCount, len(ops))

		err = resultsCollector.UpdateOperationResultsMetrics()
		assert.NoError(t, err)

		for _, op := range ops {
			assert.Equal(
				t, float64(1), testutil.ToFloat64(
					resultsCollector.metrics.With(GetLabels(op)),
				),
			)
		}

		newOp := fixRandomOp(time.Now().UTC(), domain.InProgress)
		err = operations.InsertOperation(newOp)
		assert.NoError(t, err)

		err = resultsCollector.UpdateOperationResultsMetrics()
		assert.NoError(t, err)

		assert.Equal(t, float64(1), testutil.ToFloat64(resultsCollector.metrics.With(GetLabels(newOp))))

		newOp.State = domain.InProgress
		newOp.UpdatedAt = time.Now().UTC().Add(1 * time.Second)
		_, err = operations.UpdateOperation(newOp)
		assert.NoError(t, err)
		assert.Equal(t, float64(1), testutil.ToFloat64(resultsCollector.metrics.With(GetLabels(newOp))))

		opEvent := fixRandomOp(randomCreatedAt(), domain.InProgress)
		err = resultsCollector.Handler(context.Background(), process.OperationFinished{Operation: opEvent})
		assert.NoError(t, err)

		assert.Equal(t, float64(1), testutil.ToFloat64(resultsCollector.metrics.With(GetLabels(opEvent))))

		nonExistingOp1 := fixRandomOp(randomCreatedAt(), domain.InProgress)
		nonExistingOp2 := fixRandomOp(randomCreatedAt(), domain.Failed)

		assert.Equal(t, float64(0), testutil.ToFloat64(resultsCollector.metrics.With(GetLabels(nonExistingOp1))))
		assert.Equal(t, float64(0), testutil.ToFloat64(resultsCollector.metrics.With(GetLabels(nonExistingOp2))))

		existingOp1 := fixRandomOp(time.Now().UTC(), domain.InProgress)
		err = operations.InsertOperation(existingOp1)
		assert.NoError(t, err)

		existingOp2 := fixRandomOp(time.Now().UTC(), domain.Succeeded)
		err = operations.InsertOperation(existingOp2)
		assert.NoError(t, err)

		existingOp3 := fixRandomOp(time.Now().UTC(), domain.InProgress)
		err = operations.InsertOperation(existingOp3)
		assert.NoError(t, err)

		existingOp4 := fixRandomOp(time.Now().UTC(), domain.Failed)
		err = operations.InsertOperation(existingOp4)
		assert.NoError(t, err)

		err = resultsCollector.UpdateOperationResultsMetrics()
		assert.NoError(t, err)

		assert.Equal(t, float64(1), testutil.ToFloat64(resultsCollector.metrics.With(GetLabels(existingOp1))))
		assert.Equal(t, float64(1), testutil.ToFloat64(resultsCollector.metrics.With(GetLabels(existingOp2))))
		assert.Equal(t, float64(1), testutil.ToFloat64(resultsCollector.metrics.With(GetLabels(existingOp4))))
		assert.Equal(t, float64(1), testutil.ToFloat64(resultsCollector.metrics.With(GetLabels(existingOp3))))
	})
}

func insertRandomOperations(t *testing.T, operations storage.Operations, count int) {
	for i := 0; i < count; i++ {
		op := internal.Operation{
			ID:         uuid.New().String(),
			InstanceID: uuid.New().String(),
			ProvisioningParameters: internal.ProvisioningParameters{
				PlanID: randomPlanId(),
			},
			CreatedAt: randomCreatedAt(),
			UpdatedAt: randomUpdatedAtAfterCreatedAt(),
			Type:      randomType(),
			State:     randomState(),
		}
		err := operations.InsertOperation(op)
		assert.NoError(t, err)
	}
}

func fixRandomOp(createdAt time.Time, state domain.LastOperationState) internal.Operation {
	return internal.Operation{
		ID:         uuid.New().String(),
		InstanceID: uuid.New().String(),
		ProvisioningParameters: internal.ProvisioningParameters{
			PlanID: randomPlanId(),
		},
		CreatedAt: createdAt,
		UpdatedAt: randomUpdatedAtAfterCreatedAt(),
		Type:      randomType(),
		State:     state,
	}
}

func randomState() domain.LastOperationState {
	return opStates[rand.Intn(len(opStates))]
}

func randomType() internal.OperationType {
	return opTypes[rand.Intn(len(opTypes))]
}

func randomPlanId() string {
	return string(plans[rand.Intn(len(plans))])
}

func randomCreatedAt() time.Time {
	return time.Now().UTC().Add(-time.Duration(rand.Intn(60)) * time.Minute)
}

func randomUpdatedAtAfterCreatedAt() time.Time {
	return randomCreatedAt().Add(time.Duration(rand.Intn(10)) * time.Minute)
}
