package metrics

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestOperationsStats(t *testing.T) {
	var statsCollector *operationsStats

	operations := storage.NewMemoryStorage().Operations()
	testData := []struct {
		opType      internal.OperationType
		opState     domain.LastOperationState
		opPlan      broker.PlanID
		eventsCount int
		key         metricKey
	}{
		{
			opType:      internal.OperationTypeProvision,
			opState:     domain.Succeeded,
			opPlan:      broker.AzurePlanID,
			eventsCount: 5,
		},
		{
			opType:      internal.OperationTypeUpdate,
			opState:     domain.Failed,
			opPlan:      broker.AWSPlanID,
			eventsCount: 1,
		},
		{
			opType:      internal.OperationTypeDeprovision,
			opState:     domain.Failed,
			opPlan:      broker.GCPPlanID,
			eventsCount: 3,
		},
		{
			opType:      internal.OperationTypeDeprovision,
			opState:     domain.InProgress,
			opPlan:      broker.GCPPlanID,
			eventsCount: 0,
		},
		{
			opType:      internal.OperationTypeProvision,
			opState:     domain.InProgress,
			opPlan:      broker.AzurePlanID,
			eventsCount: 1,
		},
		{
			opType:      internal.OperationTypeProvision,
			opState:     domain.InProgress,
			opPlan:      broker.AWSPlanID,
			eventsCount: 1,
		},
		{
			opType:      internal.OperationTypeDeprovision,
			opState:     domain.InProgress,
			opPlan:      broker.AWSPlanID,
			eventsCount: 0,
		},
	}

	for i, data := range testData {
		key, err := statsCollector.makeKey(data.opType, data.opState, data.opPlan)
		assert.NoError(t, err)
		testData[i].key = key
	}

	err := operations.InsertOperation(internal.Operation{
		ID:    "test-4",
		State: testData[4].opState,
		Type:  testData[4].opType,
		ProvisioningParameters: internal.ProvisioningParameters{
			PlanID: string(testData[4].opPlan),
		},
	})
	assert.NoError(t, err)
	err = operations.InsertOperation(internal.Operation{
		ID:    "test-5",
		State: testData[5].opState,
		Type:  testData[5].opType,
		ProvisioningParameters: internal.ProvisioningParameters{
			PlanID: string(testData[5].opPlan),
		},
	})
	assert.NoError(t, err)

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})).With("metrics", "test")

	cfg := Config{
		OperationStatsPollingInterval:  1 * time.Minute,
		OperationResultPollingInterval: 1 * time.Minute,
		OperationResultRetentionPeriod: 1 * time.Hour,
	}

	statsCollector = NewOperationsStats(operations, cfg, log)
	statsCollector.MustRegister()
	err = statsCollector.UpdateStatsMetrics()
	assert.NoError(t, err)

	for i := 0; i < 3; i++ {
		for j := 0; j < testData[i].eventsCount; j++ {
			err = statsCollector.Handler(context.TODO(), process.OperationFinished{
				PlanID:    testData[i].opPlan,
				Operation: internal.Operation{Type: testData[i].opType, State: testData[i].opState, ID: fmt.Sprintf("test-%d", i)},
			})
			assert.NoError(t, err)
		}
	}

	t.Run("should get correct counters", func(t *testing.T) {
		assert.Equal(t, float64(testData[0].eventsCount), testutil.ToFloat64(statsCollector.counters[testData[0].key]))
		assert.Equal(t, float64(testData[1].eventsCount), testutil.ToFloat64(statsCollector.counters[testData[1].key]))
		assert.Equal(t, float64(testData[2].eventsCount), testutil.ToFloat64(statsCollector.counters[testData[2].key]))
	})

	t.Run("should get correct gauges", func(t *testing.T) {
		assert.Equal(t, float64(testData[3].eventsCount), testutil.ToFloat64(statsCollector.gauges[testData[3].key]))
		assert.Equal(t, float64(testData[4].eventsCount), testutil.ToFloat64(statsCollector.gauges[testData[4].key]))
		assert.Equal(t, float64(testData[5].eventsCount), testutil.ToFloat64(statsCollector.gauges[testData[5].key]))
		assert.Equal(t, float64(testData[6].eventsCount), testutil.ToFloat64(statsCollector.gauges[testData[6].key]))
	})
}
