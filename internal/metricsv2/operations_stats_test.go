package metricsv2

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestOperationsStats(t *testing.T) {
	var ctr *OperationStats

	opType1 := internal.OperationTypeProvision
	opState1 := domain.Succeeded
	opPlan1 := broker.PlanID(broker.AzurePlanID)
	eventsCount1 := 5
	key1, err := ctr.makeKey(opType1, opState1, opPlan1)
	assert.NoError(t, err)

	opType2 := internal.OperationTypeUpdate
	opState2 := domain.Failed
	opPlan2 := broker.PlanID(broker.AWSPlanID)
	key2, err := ctr.makeKey(opType2, opState2, opPlan2)
	eventsCount2 := 1
	assert.NoError(t, err)

	opType3 := internal.OperationTypeDeprovision
	opState3 := domain.Failed
	opPlan3 := broker.PlanID(broker.GCPPlanID)
	eventsCount3 := 3
	key3, err := ctr.makeKey(opType3, opState3, opPlan3)
	assert.NoError(t, err)

	opType4 := internal.OperationTypeDeprovision
	opState4 := domain.InProgress
	opPlan4 := broker.PlanID(broker.GCPPlanID)
	eventsCount4 := 0
	key4, err := ctr.makeKey(opType4, opState4, opPlan4)
	assert.NoError(t, err)

	operations := storage.NewMemoryStorage().Operations()

	opType5 := internal.OperationTypeProvision
	opState5 := domain.InProgress
	opPlan5 := broker.AzurePlanID
	eventsCount5 := 1
	key5, err := ctr.makeKey(opType5, opState5, broker.PlanID(opPlan5))
	assert.NoError(t, err)

	opType6 := internal.OperationTypeProvision
	opState6 := domain.InProgress
	opPlan6 := broker.AWSPlanID
	eventsCount6 := 1
	key6, err := ctr.makeKey(opType6, opState6, broker.PlanID(opPlan6))
	assert.NoError(t, err)

	opType7 := internal.OperationTypeDeprovision
	opState7 := domain.InProgress
	opPlan7 := broker.AWSPlanID
	eventsCount7 := 0
	key7, err := ctr.makeKey(opType7, opState7, broker.PlanID(opPlan7))
	assert.NoError(t, err)

	cfg := Config{
		OperationStatsPollingInterval:  1 * time.Millisecond,
		OperationResultPollingInterval: 1 * time.Millisecond,
		OperationResultRetentionPeriod: 1 * time.Minute,
	}

	t.Run("create counter key", func(t *testing.T) {
		log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})).With("metrics", "test")
		ctr = NewOperationsStats(operations, cfg, log)
		ctr.MustRegister(context.Background())
	})

	t.Run("gauge in_progress operations test", func(t *testing.T) {
		err := operations.InsertOperation(internal.Operation{
			ID:    "opState6",
			State: opState5,
			Type:  opType5,
			ProvisioningParameters: internal.ProvisioningParameters{
				PlanID: opPlan5,
			},
		})
		assert.NoError(t, err)
		err = operations.InsertOperation(internal.Operation{
			ID:    "opState7",
			State: opState6,
			Type:  opType6,
			ProvisioningParameters: internal.ProvisioningParameters{
				PlanID: opPlan6,
			},
		})
		assert.NoError(t, err)
	})

	t.Run("should increase all counter", func(t *testing.T) {
		t.Run("should increase counter", func(t *testing.T) {
			t.Parallel()
			wg := sync.WaitGroup{}
			for i := 0; i < eventsCount1; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					err := ctr.Handler(context.TODO(), process.OperationFinished{
						PlanID:    opPlan1,
						Operation: internal.Operation{Type: opType1, State: opState1, ID: "test1"},
					})
					assert.NoError(t, err)
				}()
			}
			wg.Wait()
		})

		t.Run("should increase counter", func(t *testing.T) {
			t.Parallel()
			wg := sync.WaitGroup{}
			for i := 0; i < eventsCount2; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					err := ctr.Handler(context.TODO(), process.OperationFinished{
						PlanID:    opPlan2,
						Operation: internal.Operation{Type: opType2, State: opState2, ID: "test2"},
					})
					assert.NoError(t, err)
				}()
			}
			wg.Wait()
		})

		t.Run("should increase counter", func(t *testing.T) {
			t.Parallel()
			wg := sync.WaitGroup{}
			for i := 0; i < eventsCount3; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					err := ctr.Handler(context.TODO(), process.OperationFinished{
						PlanID:    opPlan3,
						Operation: internal.Operation{Type: opType3, State: opState3, ID: "test3"},
					})
					assert.NoError(t, err)
				}()
			}
			wg.Wait()
		})
	})

	t.Run("should get correct number of metrics", func(t *testing.T) {
		time.Sleep(1 * time.Second)
		assert.Equal(t, float64(eventsCount1), testutil.ToFloat64(ctr.counters[key1]))
		assert.Equal(t, float64(eventsCount2), testutil.ToFloat64(ctr.counters[key2]))
		assert.Equal(t, float64(eventsCount3), testutil.ToFloat64(ctr.counters[key3]))
		assert.Equal(t, float64(eventsCount4), testutil.ToFloat64(ctr.gauges[key4]))
		assert.Equal(t, float64(eventsCount5), testutil.ToFloat64(ctr.gauges[key5]))
		assert.Equal(t, float64(eventsCount6), testutil.ToFloat64(ctr.gauges[key6]))
		assert.Equal(t, float64(eventsCount7), testutil.ToFloat64(ctr.gauges[key7]))
	})

	t.Cleanup(func() {
		ctr = nil
	})
}
