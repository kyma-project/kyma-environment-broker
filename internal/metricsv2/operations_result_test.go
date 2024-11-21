package metricsv2

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/event"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const (
	tries = 1000
)

func TestOperationsResult(t *testing.T) {
	t.Run("1000 metrics should be published with 1 or 0", func(t *testing.T) {
		operations := storage.NewMemoryStorage().Operations()
		for i := 0; i < tries; i++ {
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

		operationResult := NewOperationResult(
			context.Background(), operations, Config{
				Enabled: true, OperationResultPollingInterval: 10 * time.Millisecond,
				OperationStatsPollingInterval: 10 * time.Millisecond, OperationResultRetentionPeriod: 24 * time.Hour,
			}, logrus.New(),
		)

		eventBroker := event.NewPubSub(logrus.New())
		eventBroker.Subscribe(process.OperationFinished{}, operationResult.Handler)

		time.Sleep(20 * time.Millisecond)

		ops, err := operations.GetAllOperations()
		assert.NoError(t, err)
		assert.Equal(t, tries, len(ops))

		for _, op := range ops {
			assert.Equal(
				t, float64(1), testutil.ToFloat64(
					operationResult.metrics.With(GetLabels(op)),
				),
			)
		}

		newOp := getRandomOp(time.Now().UTC(), domain.InProgress)
		err = operations.InsertOperation(newOp)
		time.Sleep(20 * time.Millisecond)

		assert.NoError(t, err)
		assert.Equal(t, float64(1), testutil.ToFloat64(operationResult.metrics.With(GetLabels(newOp))))

		newOp.State = domain.InProgress
		newOp.UpdatedAt = time.Now().UTC().Add(1 * time.Second)
		_, err = operations.UpdateOperation(newOp)
		assert.NoError(t, err)
		assert.Equal(t, float64(1), testutil.ToFloat64(operationResult.metrics.With(GetLabels(newOp))))

		opEvent := getRandomOp(randomCreatedAt(), domain.InProgress)
		eventBroker.Publish(context.Background(), process.OperationFinished{Operation: opEvent})
		time.Sleep(20 * time.Millisecond)
		assert.Equal(t, float64(1), testutil.ToFloat64(operationResult.metrics.With(GetLabels(opEvent))))

		nonExistingOp1 := getRandomOp(randomCreatedAt(), domain.InProgress)
		nonExistingOp2 := getRandomOp(randomCreatedAt(), domain.Failed)
		time.Sleep(20 * time.Millisecond)

		assert.Equal(t, float64(0), testutil.ToFloat64(operationResult.metrics.With(GetLabels(nonExistingOp1))))
		assert.Equal(t, float64(0), testutil.ToFloat64(operationResult.metrics.With(GetLabels(nonExistingOp2))))

		existingOp1 := getRandomOp(time.Now().UTC(), domain.InProgress)
		err = operations.InsertOperation(existingOp1)
		assert.NoError(t, err)

		existingOp2 := getRandomOp(time.Now().UTC(), domain.Succeeded)
		err = operations.InsertOperation(existingOp2)
		assert.NoError(t, err)

		existingOp3 := getRandomOp(time.Now().UTC(), domain.InProgress)
		err = operations.InsertOperation(existingOp3)
		assert.NoError(t, err)

		existingOp4 := getRandomOp(time.Now().UTC(), domain.Failed)
		err = operations.InsertOperation(existingOp4)
		assert.NoError(t, err)

		time.Sleep(20 * time.Millisecond)

		assert.Equal(t, float64(1), testutil.ToFloat64(operationResult.metrics.With(GetLabels(existingOp1))))
		assert.Equal(t, float64(1), testutil.ToFloat64(operationResult.metrics.With(GetLabels(existingOp2))))
		assert.Equal(t, float64(1), testutil.ToFloat64(operationResult.metrics.With(GetLabels(existingOp4))))
		assert.Equal(t, float64(1), testutil.ToFloat64(operationResult.metrics.With(GetLabels(existingOp3))))
	})
}

func getRandomOp(createdAt time.Time, state domain.LastOperationState) internal.Operation {
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
