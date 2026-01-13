package subaccountsync

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/cis"
	"golang.org/x/time/rate"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	queues "github.com/kyma-project/kyma-environment-broker/internal/syncqueues"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	subaccountId1 = "subaccount-1"
	subaccountId2 = "subaccount-2"
	subaccountId3 = "subaccount-3"
	subaccountId4 = "subaccount-4"
	runtimeId11   = "runtime-1-1"
	runtimeId12   = "runtime-1-2"
	runtimeId21   = "runtime-2-1"
	runtimeId31   = "runtime-3-1"
	veryOldTime   = 1
	oldTime       = 10
	notSoOldTime  = 20
	recent        = 40
)

var logger = slog.New(slog.NewTextHandler(os.Stderr, nil))

var useInMemoryStorage, _ = strconv.ParseBool(os.Getenv("DB_IN_MEMORY_FOR_E2E_TESTS"))

var tearDownFunc func()

func setupSuite(t testing.TB) func(t testing.TB) {
	logger.Info("setup suite")
	if useInMemoryStorage {
		logger.Info("using in-memory storage")
	} else {
		logger.Info("using real storage")
		tearDownFunc = setupStorageContainer()
	}

	return func(t testing.TB) {
		r := recover()
		logger.Info("teardown suite")
		if tearDownFunc != nil {
			tearDownFunc()
		}
		if r != nil {
			panic(r)
		}
	}
}

func setupTestNilStorage(t testing.TB) (func(t testing.TB), storage.BrokerStorage) {
	logger.Info("setup test - no storage needed")

	return func(t testing.TB) {
		if r := recover(); r != nil {
			if tearDownFunc != nil {
				tearDownFunc()
			}
			panic(r)
		}
		logger.Info("teardown test")
	}, nil
}

func setupTestWithStorage(t testing.TB) (func(t testing.TB), storage.BrokerStorage) {
	logger.Info("setup test - create storage")
	storageCleanup, brokerStorage, err := getStorageForTests()
	require.NoError(t, err)
	require.NotNil(t, brokerStorage)

	return func(t testing.TB) {
		if r := recover(); r != nil {
			if tearDownFunc != nil {
				tearDownFunc()
			}
			panic(r)
		}
		if storageCleanup != nil {
			logger.Info("teardown test - cleanup storage")
			err := storageCleanup()
			assert.NoError(t, err)
		}
	}, brokerStorage
}

func TestStateReconcilerWithFakeCisServer(t *testing.T) {
	teardownSuite := setupSuite(t)
	defer teardownSuite(t)

	srv, err := cis.NewFakeServer()
	defer srv.Close()
	require.NoError(t, err)

	cisClient := srv.Client()
	cisConfig := CisEndpointConfig{
		ServiceURL:             srv.URL,
		RateLimitingInterval:   time.Minute * 10,
		MaxRequestsPerInterval: 1000,
	}

	t.Run("should schedule update of one resource after getting account data from faked CIS, then new subaccount comes in", func(t *testing.T) {
		teardownTest, brokerStorage := setupTestNilStorage(t)
		defer teardownTest(t)

		reconciler := createNewReconcilerWithFakeCisServer(brokerStorage, cisClient, cisConfig)

		// given
		// initial event from a kyma resource, first runtime, no label
		reconciler.reconcileResourceUpdate(cis.FakeSubaccountID1, runtimeId11, runtimeStateType{betaEnabled: "", usedForProduction: ""})
		assert.Equal(t, 1, len(reconciler.inMemoryState))

		// queue should be empty since we have not got state from CIS
		assert.True(t, reconciler.syncQueue.IsEmpty())

		// then the same subaccount, second runtime, with false label
		reconciler.reconcileResourceUpdate(cis.FakeSubaccountID1, runtimeId12, runtimeStateType{betaEnabled: "false", usedForProduction: "NOT_USED_FOR_PRODUCTION"})
		assert.Equal(t, 1, len(reconciler.inMemoryState))
		assert.True(t, reconciler.syncQueue.IsEmpty())

		// then we get state from CIS (accounts)
		reconciler.periodicAccountsSync()

		// then queue should contain one element
		assert.False(t, reconciler.syncQueue.IsEmpty())
		element, ok := reconciler.syncQueue.Extract()
		assert.True(t, ok)
		assert.Equal(t, cis.FakeSubaccountID1, element.SubaccountID)
		assert.Equal(t, "false", element.BetaEnabled)
		assert.Equal(t, "NOT_USED_FOR_PRODUCTION", reconciler.inMemoryState[cis.FakeSubaccountID1].cisState.UsedForProduction)
		assert.True(t, reconciler.syncQueue.IsEmpty())

		//then we got update from the plane (updater updated resources)
		reconciler.reconcileResourceUpdate(cis.FakeSubaccountID1, runtimeId11, runtimeStateType{betaEnabled: "false", usedForProduction: "NOT_USED_FOR_PRODUCTION"})
		reconciler.reconcileResourceUpdate(cis.FakeSubaccountID1, runtimeId12, runtimeStateType{betaEnabled: "false", usedForProduction: "NOT_USED_FOR_PRODUCTION"})

		// then we add kyma resource, so we got update from informer
		reconciler.reconcileResourceUpdate(cis.FakeSubaccountID2, runtimeId21, runtimeStateType{betaEnabled: "false", usedForProduction: "USED_FOR_PRODUCTION"})
		assert.Equal(t, 2, len(reconciler.inMemoryState))
		assert.True(t, reconciler.syncQueue.IsEmpty())

		// then we get state from CIS (accounts)
		reconciler.periodicAccountsSync()

		// then queue should contain one element
		assert.False(t, reconciler.syncQueue.IsEmpty())
		element, ok = reconciler.syncQueue.Extract()
		assert.True(t, ok)
		assert.Equal(t, cis.FakeSubaccountID2, element.SubaccountID)
		assert.Equal(t, "true", element.BetaEnabled)
		assert.Equal(t, "USED_FOR_PRODUCTION", reconciler.inMemoryState[cis.FakeSubaccountID2].cisState.UsedForProduction)
		assert.True(t, reconciler.syncQueue.IsEmpty())
	})

	t.Run("should not schedule any change when event comes about one subaccount and betaEnabled has the same value", func(t *testing.T) {
		teardownTest, brokerStorage := setupTestNilStorage(t)
		defer teardownTest(t)

		reconciler := createNewReconcilerWithFakeCisServer(brokerStorage, cisClient, cisConfig)

		// given
		// initial event from a kyma resources, all true
		reconciler.reconcileResourceUpdate(cis.FakeSubaccountID1, runtimeId11, runtimeStateType{betaEnabled: "true"})
		reconciler.reconcileResourceUpdate(cis.FakeSubaccountID2, runtimeId21, runtimeStateType{betaEnabled: "true", usedForProduction: "USED_FOR_PRODUCTION"})
		assert.Equal(t, 2, len(reconciler.inMemoryState))

		// queue should be empty since we have not got state from CIS
		assert.True(t, reconciler.syncQueue.IsEmpty())

		// then we get state from CIS (accounts)
		reconciler.periodicAccountsSync()

		// then queue should contain one element
		assert.False(t, reconciler.syncQueue.IsEmpty())
		element, ok := reconciler.syncQueue.Extract()
		assert.True(t, ok)
		assert.Equal(t, cis.FakeSubaccountID1, element.SubaccountID)
		assert.Equal(t, "false", element.BetaEnabled)
		assert.Equal(t, "NOT_USED_FOR_PRODUCTION", reconciler.inMemoryState[cis.FakeSubaccountID1].cisState.UsedForProduction)
		assert.True(t, reconciler.syncQueue.IsEmpty())

		//then we got update from the plane (updater updated resources)
		reconciler.reconcileResourceUpdate(cis.FakeSubaccountID1, runtimeId11, runtimeStateType{betaEnabled: "false", usedForProduction: "NOT_USED_FOR_PRODUCTION"})

		assert.True(t, reconciler.syncQueue.IsEmpty())

		// then we get events from CIS but betaEnabled is not changed
		reconciler.periodicEventsSync(1710770000000)

		// then queue should contain be empty
		assert.True(t, reconciler.syncQueue.IsEmpty())
	})

	t.Run("should schedule update after event comes about change of one subaccount and betaEnabled has different value", func(t *testing.T) {
		teardownTest, brokerStorage := setupTestNilStorage(t)
		defer teardownTest(t)

		reconciler := createNewReconcilerWithFakeCisServer(brokerStorage, cisClient, cisConfig)

		// given
		// initial event from a kyma resources, all true
		reconciler.reconcileResourceUpdate(cis.FakeSubaccountID1, runtimeId11, runtimeStateType{betaEnabled: "true"})
		reconciler.reconcileResourceUpdate(cis.FakeSubaccountID2, runtimeId21, runtimeStateType{betaEnabled: "true", usedForProduction: "USED_FOR_PRODUCTION"})
		assert.Equal(t, 2, len(reconciler.inMemoryState))

		// queue should be empty since we have not got state from CIS
		assert.True(t, reconciler.syncQueue.IsEmpty())

		// then we get state from CIS (accounts)
		reconciler.periodicAccountsSync()

		// then queue should contain one element
		assert.False(t, reconciler.syncQueue.IsEmpty())
		element, ok := reconciler.syncQueue.Extract()
		assert.True(t, ok)
		assert.Equal(t, cis.FakeSubaccountID1, element.SubaccountID)
		assert.Equal(t, "false", element.BetaEnabled)
		assert.Equal(t, "NOT_USED_FOR_PRODUCTION", reconciler.inMemoryState[cis.FakeSubaccountID1].cisState.UsedForProduction)
		assert.True(t, reconciler.syncQueue.IsEmpty())

		//then we got update from the plane (updater updated resources)
		reconciler.reconcileResourceUpdate(cis.FakeSubaccountID1, runtimeId11, runtimeStateType{betaEnabled: "false", usedForProduction: "NOT_USED_FOR_PRODUCTION"})

		assert.True(t, reconciler.syncQueue.IsEmpty())

		// then we get events from CIS but betaEnabled is changed
		reconciler.periodicEventsSync(1710761400000)

		// then queue should contain one element
		assert.False(t, reconciler.syncQueue.IsEmpty())
		element, ok = reconciler.syncQueue.Extract()
		assert.True(t, ok)
		assert.Equal(t, cis.FakeSubaccountID1, element.SubaccountID)
		assert.Equal(t, "true", element.BetaEnabled)
		assert.Equal(t, "UNSET", reconciler.inMemoryState[cis.FakeSubaccountID1].cisState.UsedForProduction)
		assert.True(t, reconciler.syncQueue.IsEmpty())

		//then we got update from the plane (updater updated resources)
		reconciler.reconcileResourceUpdate(cis.FakeSubaccountID1, runtimeId11, runtimeStateType{betaEnabled: "true", usedForProduction: "UNSET"})

		assert.True(t, reconciler.syncQueue.IsEmpty())
	})

	t.Run("should schedule update after event comes about change of one subaccount and usedForProduction has different value", func(t *testing.T) {
		teardownTest, brokerStorage := setupTestNilStorage(t)
		defer teardownTest(t)

		reconciler := createNewReconcilerWithFakeCisServer(brokerStorage, cisClient, cisConfig)

		// given
		// initial event from a kyma resources
		reconciler.reconcileResourceUpdate(cis.FakeSubaccountID1, runtimeId11, runtimeStateType{betaEnabled: "false", usedForProduction: "USED_FOR_PRODUCTION"})
		reconciler.reconcileResourceUpdate(cis.FakeSubaccountID2, runtimeId21, runtimeStateType{betaEnabled: "true", usedForProduction: "USED_FOR_PRODUCTION"})
		assert.Equal(t, 2, len(reconciler.inMemoryState))

		// queue should be empty since we have not got state from CIS
		assert.True(t, reconciler.syncQueue.IsEmpty())

		// then we get state from CIS (accounts)
		reconciler.periodicAccountsSync()

		// then queue should contain one element
		assert.False(t, reconciler.syncQueue.IsEmpty())
		element, ok := reconciler.syncQueue.Extract()
		assert.True(t, ok)
		assert.Equal(t, cis.FakeSubaccountID1, element.SubaccountID)
		assert.Equal(t, "false", element.BetaEnabled)
		assert.Equal(t, "NOT_USED_FOR_PRODUCTION", element.UsedForProduction)
		assert.True(t, reconciler.syncQueue.IsEmpty())

		//then we got update from the plane (updater updated resources)
		reconciler.reconcileResourceUpdate(cis.FakeSubaccountID1, runtimeId11, runtimeStateType{betaEnabled: "false", usedForProduction: "NOT_USED_FOR_PRODUCTION"})

		assert.True(t, reconciler.syncQueue.IsEmpty())

		// then we get events from CIS but betaEnabled is changed
		reconciler.periodicEventsSync(1710761400000)

		// then queue should contain one element
		assert.False(t, reconciler.syncQueue.IsEmpty())
		element, ok = reconciler.syncQueue.Extract()
		assert.True(t, ok)
		assert.Equal(t, cis.FakeSubaccountID1, element.SubaccountID)
		assert.Equal(t, "true", element.BetaEnabled)
		assert.Equal(t, "UNSET", element.UsedForProduction)
		assert.True(t, reconciler.syncQueue.IsEmpty())

		//then we got update from the plane (updater updated resources)
		reconciler.reconcileResourceUpdate(cis.FakeSubaccountID1, runtimeId11, runtimeStateType{betaEnabled: "true", usedForProduction: "UNSET"})

		assert.True(t, reconciler.syncQueue.IsEmpty())
	})

	t.Run("should handle properly change for many runtimes repeating requests", func(t *testing.T) {
		teardownTest, brokerStorage := setupTestNilStorage(t)
		defer teardownTest(t)

		reconciler := createNewReconcilerWithFakeCisServer(brokerStorage, cisClient, cisConfig)

		// given
		// initial event from a kyma resources, all true
		reconciler.reconcileResourceUpdate(cis.FakeSubaccountID1, runtimeId11, runtimeStateType{betaEnabled: "true"})
		reconciler.reconcileResourceUpdate(cis.FakeSubaccountID1, runtimeId12, runtimeStateType{betaEnabled: "true"})
		assert.Equal(t, 1, len(reconciler.inMemoryState))

		// queue should be empty since we have not got state from CIS
		assert.True(t, reconciler.syncQueue.IsEmpty())

		// then we get state from CIS (accounts)
		reconciler.periodicAccountsSync()

		// then queue should contain one element
		assert.False(t, reconciler.syncQueue.IsEmpty())
		element, ok := reconciler.syncQueue.Extract()
		assert.True(t, ok)
		assert.Equal(t, cis.FakeSubaccountID1, element.SubaccountID)
		assert.Equal(t, "false", element.BetaEnabled)
		assert.Equal(t, "NOT_USED_FOR_PRODUCTION", reconciler.inMemoryState[cis.FakeSubaccountID1].cisState.UsedForProduction)
		assert.True(t, reconciler.syncQueue.IsEmpty())

		//then we got update from the plane (updater updated first resource)
		reconciler.reconcileResourceUpdate(cis.FakeSubaccountID1, runtimeId11, runtimeStateType{betaEnabled: "false", usedForProduction: "NOT_USED_FOR_PRODUCTION"})

		// but we still have one resource with true label so we enqueue the update request again
		assert.False(t, reconciler.syncQueue.IsEmpty())
		element, ok = reconciler.syncQueue.Extract()
		assert.True(t, ok)
		assert.Equal(t, cis.FakeSubaccountID1, element.SubaccountID)
		assert.Equal(t, "false", element.BetaEnabled)
		assert.Equal(t, "NOT_USED_FOR_PRODUCTION", reconciler.inMemoryState[cis.FakeSubaccountID1].cisState.UsedForProduction)
		assert.True(t, reconciler.syncQueue.IsEmpty())

		//then we got update from the plane (updater updated second resource)
		reconciler.reconcileResourceUpdate(cis.FakeSubaccountID1, runtimeId12, runtimeStateType{betaEnabled: "false", usedForProduction: "NOT_USED_FOR_PRODUCTION"})

		assert.True(t, reconciler.syncQueue.IsEmpty())
	})

	t.Run("should handle properly change for many runtimes with event coming", func(t *testing.T) {
		teardownTest, brokerStorage := setupTestNilStorage(t)
		defer teardownTest(t)

		reconciler := createNewReconcilerWithFakeCisServer(brokerStorage, cisClient, cisConfig)

		// given
		// initial event from a kyma resources, all true
		reconciler.reconcileResourceUpdate(cis.FakeSubaccountID1, runtimeId11, runtimeStateType{betaEnabled: "true"})
		reconciler.reconcileResourceUpdate(cis.FakeSubaccountID1, runtimeId12, runtimeStateType{betaEnabled: "true"})
		assert.Equal(t, 1, len(reconciler.inMemoryState))

		// queue should be empty since we have not got state from CIS
		assert.True(t, reconciler.syncQueue.IsEmpty())

		// then we get state from CIS (accounts)
		reconciler.periodicAccountsSync()

		// then queue should contain one element
		assert.False(t, reconciler.syncQueue.IsEmpty())
		element, ok := reconciler.syncQueue.Extract()
		assert.True(t, ok)
		assert.Equal(t, cis.FakeSubaccountID1, element.SubaccountID)
		assert.Equal(t, "false", element.BetaEnabled)
		assert.Equal(t, "NOT_USED_FOR_PRODUCTION", reconciler.inMemoryState[cis.FakeSubaccountID1].cisState.UsedForProduction)
		assert.True(t, reconciler.syncQueue.IsEmpty())

		//then we got update from the plane (updater updated first resource)
		reconciler.reconcileResourceUpdate(cis.FakeSubaccountID1, runtimeId11, runtimeStateType{betaEnabled: "false"})

		// but we still have one resource with true label so we enqueue the update request again
		assert.False(t, reconciler.syncQueue.IsEmpty())

		// then we get events from CIS and betaEnabled is changed
		reconciler.periodicEventsSync(1710761400000)

		// then queue should contain one element but with beteEnabled true
		assert.False(t, reconciler.syncQueue.IsEmpty())

		element, ok = reconciler.syncQueue.Extract()
		assert.True(t, ok)
		assert.Equal(t, cis.FakeSubaccountID1, element.SubaccountID)
		assert.Equal(t, "true", element.BetaEnabled)
		assert.Equal(t, "UNSET", reconciler.inMemoryState[cis.FakeSubaccountID1].cisState.UsedForProduction)
		assert.True(t, reconciler.syncQueue.IsEmpty())

		//then we got outstanding update from the plane (updater updated second resource with false label)
		reconciler.reconcileResourceUpdate(cis.FakeSubaccountID1, runtimeId12, runtimeStateType{betaEnabled: "false"})

		// so we have not reached the stable state so we enqueue the update request again
		assert.False(t, reconciler.syncQueue.IsEmpty())
		element, ok = reconciler.syncQueue.Extract()
		assert.True(t, ok)
		assert.Equal(t, cis.FakeSubaccountID1, element.SubaccountID)
		assert.Equal(t, "true", element.BetaEnabled)
		assert.True(t, reconciler.syncQueue.IsEmpty())

		//then we got update from the plane (updater updated second resource with true label)
		reconciler.reconcileResourceUpdate(cis.FakeSubaccountID1, runtimeId12, runtimeStateType{betaEnabled: "true", usedForProduction: "UNSET"})

		assert.False(t, reconciler.syncQueue.IsEmpty())
		element, ok = reconciler.syncQueue.Extract()
		assert.True(t, ok)
		assert.Equal(t, cis.FakeSubaccountID1, element.SubaccountID)
		assert.Equal(t, "true", element.BetaEnabled)
		assert.True(t, reconciler.syncQueue.IsEmpty())

		//then we got update from the plane (updater updated second resource with true label)
		reconciler.reconcileResourceUpdate(cis.FakeSubaccountID1, runtimeId11, runtimeStateType{betaEnabled: "true", usedForProduction: "UNSET"})
		assert.True(t, reconciler.syncQueue.IsEmpty())
	})
}

func TestOutdatedPredicate(t *testing.T) {

	reconciler := createNewReconciler(nil)

	t.Run("should detect difference between false and true", func(t *testing.T) {
		state := subaccountStateType{
			cisState: CisStateType{BetaEnabled: false, UsedForProduction: "USED_FOR_PRODUCTION", ModifiedDate: veryOldTime},
			resourcesState: subaccountRuntimesType{
				runtimeId11: runtimeStateType{betaEnabled: "true", usedForProduction: "USED_FOR_PRODUCTION"},
			},
		}
		assert.True(t, reconciler.isResourceOutdated(subaccountId1, state))
	})
	t.Run("should detect difference between true and false", func(t *testing.T) {
		state := subaccountStateType{
			cisState: CisStateType{BetaEnabled: true, UsedForProduction: "USED_FOR_PRODUCTION", ModifiedDate: veryOldTime},
			resourcesState: subaccountRuntimesType{
				runtimeId11: runtimeStateType{betaEnabled: "false", usedForProduction: "USED_FOR_PRODUCTION"},
			},
		}
		assert.True(t, reconciler.isResourceOutdated(subaccountId1, state)) // outdated
	})
	t.Run("should detect difference between USED_FOR_PRODUCTION and NOT_USED_FOR_PRODUCTION", func(t *testing.T) {
		state := subaccountStateType{
			cisState: CisStateType{BetaEnabled: true, UsedForProduction: "USED_FOR_PRODUCTION", ModifiedDate: veryOldTime},
			resourcesState: subaccountRuntimesType{
				runtimeId11: runtimeStateType{betaEnabled: "true", usedForProduction: "NOT_USED_FOR_PRODUCTION"},
			},
		}
		assert.True(t, reconciler.isResourceOutdated(subaccountId1, state))
	})
	t.Run("should detect difference between NOT_USED_FOR_PRODUCTION and USED_FOR_PRODUCTION", func(t *testing.T) {
		state := subaccountStateType{
			cisState: CisStateType{BetaEnabled: true, UsedForProduction: "NOT_USED_FOR_PRODUCTION", ModifiedDate: veryOldTime},
			resourcesState: subaccountRuntimesType{
				runtimeId11: runtimeStateType{betaEnabled: "true", usedForProduction: "USED_FOR_PRODUCTION"},
			},
		}
		assert.True(t, reconciler.isResourceOutdated(subaccountId1, state))
	})
	t.Run("should detect difference between true and any other than boolean", func(t *testing.T) {
		state := subaccountStateType{
			cisState: CisStateType{BetaEnabled: true, UsedForProduction: "USED_FOR_PRODUCTION", ModifiedDate: veryOldTime},
			resourcesState: subaccountRuntimesType{
				runtimeId11: runtimeStateType{betaEnabled: "any", usedForProduction: "USED_FOR_PRODUCTION"},
			},
		}
		assert.True(t, reconciler.isResourceOutdated(subaccountId1, state)) // outdated
	})
	t.Run("should detect difference between false and any other than boolean", func(t *testing.T) {
		state := subaccountStateType{
			cisState: CisStateType{BetaEnabled: false, UsedForProduction: "USED_FOR_PRODUCTION", ModifiedDate: veryOldTime},
			resourcesState: subaccountRuntimesType{
				runtimeId11: runtimeStateType{betaEnabled: "any", usedForProduction: "USED_FOR_PRODUCTION"},
			},
		}
		assert.True(t, reconciler.isResourceOutdated(subaccountId1, state)) // outdated
	})
	t.Run("should detect difference between USED_FOR_PRODUCTION and any other", func(t *testing.T) {
		state := subaccountStateType{
			cisState: CisStateType{BetaEnabled: true, UsedForProduction: "USED_FOR_PRODUCTION", ModifiedDate: veryOldTime},
			resourcesState: subaccountRuntimesType{
				runtimeId11: runtimeStateType{betaEnabled: "true", usedForProduction: "any"},
			},
		}
		assert.True(t, reconciler.isResourceOutdated(subaccountId1, state)) // outdated
	})
	t.Run("should detect difference between NOT_USED_FOR_PRODUCTION and any other than boolean", func(t *testing.T) {
		state := subaccountStateType{
			cisState: CisStateType{BetaEnabled: false, UsedForProduction: "NOT_USED_FOR_PRODUCTION", ModifiedDate: veryOldTime},
			resourcesState: subaccountRuntimesType{
				runtimeId11: runtimeStateType{betaEnabled: "false", usedForProduction: "any"},
			},
		}
		assert.True(t, reconciler.isResourceOutdated(subaccountId1, state)) // outdated
	})
	t.Run("should detect difference between true and empty", func(t *testing.T) {
		state := subaccountStateType{
			cisState: CisStateType{BetaEnabled: true, UsedForProduction: "USED_FOR_PRODUCTION", ModifiedDate: veryOldTime},
			resourcesState: subaccountRuntimesType{
				runtimeId11: runtimeStateType{betaEnabled: "", usedForProduction: "USED_FOR_PRODUCTION"},
			},
		}
		assert.True(t, reconciler.isResourceOutdated(subaccountId1, state)) // outdated
	})
	t.Run("should detect difference between false and empty", func(t *testing.T) {
		state := subaccountStateType{
			cisState: CisStateType{BetaEnabled: false, UsedForProduction: "USED_FOR_PRODUCTION", ModifiedDate: veryOldTime},
			resourcesState: subaccountRuntimesType{
				runtimeId11: runtimeStateType{betaEnabled: "", usedForProduction: "USED_FOR_PRODUCTION"},
			},
		}
		assert.True(t, reconciler.isResourceOutdated(subaccountId1, state)) // outdated
	})
	t.Run("should detect difference between USED_FOR_PRODUCTION and empty", func(t *testing.T) {
		state := subaccountStateType{
			cisState: CisStateType{BetaEnabled: true, UsedForProduction: "USED_FOR_PRODUCTION", ModifiedDate: veryOldTime},
			resourcesState: subaccountRuntimesType{
				runtimeId11: runtimeStateType{betaEnabled: "true", usedForProduction: ""},
			},
		}
		assert.True(t, reconciler.isResourceOutdated(subaccountId1, state)) // outdated
	})
	t.Run("should detect difference between NOT_USED_FOR_PRODUCTION and empty", func(t *testing.T) {
		state := subaccountStateType{
			cisState: CisStateType{BetaEnabled: false, UsedForProduction: "NOT_USED_FOR_PRODUCTION", ModifiedDate: veryOldTime},
			resourcesState: subaccountRuntimesType{
				runtimeId11: runtimeStateType{betaEnabled: "false", usedForProduction: ""},
			},
		}
		assert.True(t, reconciler.isResourceOutdated(subaccountId1, state)) // outdated
	})
	t.Run("should treat as up-to-date true and true", func(t *testing.T) {
		state := subaccountStateType{
			cisState: CisStateType{BetaEnabled: true, UsedForProduction: "USED_FOR_PRODUCTION", ModifiedDate: veryOldTime},
			resourcesState: subaccountRuntimesType{
				runtimeId11: runtimeStateType{betaEnabled: "true", usedForProduction: "USED_FOR_PRODUCTION"},
			},
		}
		assert.False(t, reconciler.isResourceOutdated(subaccountId1, state))
	})
	t.Run("should treat as up-to-date false and false", func(t *testing.T) {
		state := subaccountStateType{
			cisState: CisStateType{BetaEnabled: false, UsedForProduction: "NOT_USED_FOR_PRODUCTION", ModifiedDate: veryOldTime},
			resourcesState: subaccountRuntimesType{
				runtimeId11: runtimeStateType{betaEnabled: "false", usedForProduction: "NOT_USED_FOR_PRODUCTION"},
			},
		}
		assert.False(t, reconciler.isResourceOutdated(subaccountId1, state))
	})
	t.Run("should detect difference between false and true if one is true", func(t *testing.T) {
		state := subaccountStateType{
			cisState: CisStateType{BetaEnabled: false, UsedForProduction: "USED_FOR_PRODUCTION", ModifiedDate: veryOldTime},
			resourcesState: subaccountRuntimesType{
				runtimeId11: runtimeStateType{betaEnabled: "true", usedForProduction: "USED_FOR_PRODUCTION"},
				runtimeId12: runtimeStateType{betaEnabled: "false", usedForProduction: "USED_FOR_PRODUCTION"},
			},
		}
		assert.True(t, reconciler.isResourceOutdated(subaccountId1, state))
	})
	t.Run("should detect difference between true and false if one is false", func(t *testing.T) {
		state := subaccountStateType{
			cisState: CisStateType{BetaEnabled: true, UsedForProduction: "USED_FOR_PRODUCTION", ModifiedDate: veryOldTime},
			resourcesState: subaccountRuntimesType{
				runtimeId11: runtimeStateType{betaEnabled: "false", usedForProduction: "USED_FOR_PRODUCTION"},
				runtimeId12: runtimeStateType{betaEnabled: "true", usedForProduction: "USED_FOR_PRODUCTION"},
			},
		}
		assert.True(t, reconciler.isResourceOutdated(subaccountId1, state)) // outdated
	})
	t.Run("should detect difference between NOT_USED_FOR_PRODUCTION and USED_FOR_PRODUCTION if one is USED_FOR_PRODUCTION", func(t *testing.T) {
		state := subaccountStateType{
			cisState: CisStateType{BetaEnabled: false, UsedForProduction: "NOT_USED_FOR_PRODUCTION", ModifiedDate: veryOldTime},
			resourcesState: subaccountRuntimesType{
				runtimeId11: runtimeStateType{betaEnabled: "false", usedForProduction: "USED_FOR_PRODUCTION"},
				runtimeId12: runtimeStateType{betaEnabled: "false", usedForProduction: "NOT_USED_FOR_PRODUCTION"},
			},
		}
		assert.True(t, reconciler.isResourceOutdated(subaccountId1, state))
	})
	t.Run("should detect difference between USED_FOR_PRODUCTION and NOT_USED_FOR_PRODUCTION if one is NOT_USED_FOR_PRODUCTION", func(t *testing.T) {
		state := subaccountStateType{
			cisState: CisStateType{BetaEnabled: true, UsedForProduction: "USED_FOR_PRODUCTION", ModifiedDate: veryOldTime},
			resourcesState: subaccountRuntimesType{
				runtimeId11: runtimeStateType{betaEnabled: "true", usedForProduction: "NOT_USED_FOR_PRODUCTION"},
				runtimeId12: runtimeStateType{betaEnabled: "true", usedForProduction: "USED_FOR_PRODUCTION"},
			},
		}
		assert.True(t, reconciler.isResourceOutdated(subaccountId1, state)) // outdated
	})
	t.Run("should detect difference between true and any other than boolean", func(t *testing.T) {
		state := subaccountStateType{
			cisState: CisStateType{BetaEnabled: true, UsedForProduction: "USED_FOR_PRODUCTION", ModifiedDate: veryOldTime},
			resourcesState: subaccountRuntimesType{
				runtimeId11: runtimeStateType{betaEnabled: "any", usedForProduction: "USED_FOR_PRODUCTION"},
				runtimeId12: runtimeStateType{betaEnabled: "true", usedForProduction: "USED_FOR_PRODUCTION"},
			},
		}
		assert.True(t, reconciler.isResourceOutdated(subaccountId1, state)) // outdated
	})
	t.Run("should detect difference between false and any other than boolean if one is not boolean", func(t *testing.T) {
		state := subaccountStateType{
			cisState: CisStateType{BetaEnabled: false, UsedForProduction: "USED_FOR_PRODUCTION", ModifiedDate: veryOldTime},
			resourcesState: subaccountRuntimesType{
				runtimeId11: runtimeStateType{betaEnabled: "any", usedForProduction: "USED_FOR_PRODUCTION"},
				runtimeId12: runtimeStateType{betaEnabled: "false", usedForProduction: "USED_FOR_PRODUCTION"},
			},
		}
		assert.True(t, reconciler.isResourceOutdated(subaccountId1, state)) // outdated
	})
	t.Run("should detect difference between true and empty if one is empty", func(t *testing.T) {
		state := subaccountStateType{
			cisState: CisStateType{BetaEnabled: true, UsedForProduction: "USED_FOR_PRODUCTION", ModifiedDate: veryOldTime},
			resourcesState: subaccountRuntimesType{
				runtimeId11: runtimeStateType{betaEnabled: "", usedForProduction: "USED_FOR_PRODUCTION"},
				runtimeId12: runtimeStateType{betaEnabled: "true", usedForProduction: "USED_FOR_PRODUCTION"},
			},
		}
		assert.True(t, reconciler.isResourceOutdated(subaccountId1, state)) // outdated
	})
	t.Run("should detect difference between false and empty if one is empty", func(t *testing.T) {
		state := subaccountStateType{
			cisState: CisStateType{BetaEnabled: false, UsedForProduction: "USED_FOR_PRODUCTION", ModifiedDate: veryOldTime},
			resourcesState: subaccountRuntimesType{
				runtimeId11: runtimeStateType{betaEnabled: "", usedForProduction: "USED_FOR_PRODUCTION"},
				runtimeId12: runtimeStateType{betaEnabled: "false", usedForProduction: "USED_FOR_PRODUCTION"},
			},
		}
		assert.True(t, reconciler.isResourceOutdated(subaccountId1, state)) // outdated
	})
	t.Run("should treat as up-to-date true and all true", func(t *testing.T) {
		state := subaccountStateType{
			cisState: CisStateType{BetaEnabled: true, UsedForProduction: "USED_FOR_PRODUCTION", ModifiedDate: veryOldTime},
			resourcesState: subaccountRuntimesType{
				runtimeId11: runtimeStateType{betaEnabled: "true", usedForProduction: "USED_FOR_PRODUCTION"},
				runtimeId12: runtimeStateType{betaEnabled: "true", usedForProduction: "USED_FOR_PRODUCTION"},
			},
		}
		assert.False(t, reconciler.isResourceOutdated(subaccountId1, state))
	})
	t.Run("should treat as up-to-date false and all false", func(t *testing.T) {
		state := subaccountStateType{
			cisState: CisStateType{BetaEnabled: false, UsedForProduction: "NOT_USED_FOR_PRODUCTION", ModifiedDate: veryOldTime},
			resourcesState: subaccountRuntimesType{
				runtimeId11: runtimeStateType{betaEnabled: "false", usedForProduction: "NOT_USED_FOR_PRODUCTION"},
				runtimeId12: runtimeStateType{betaEnabled: "false", usedForProduction: "NOT_USED_FOR_PRODUCTION"},
			},
		}
		assert.False(t, reconciler.isResourceOutdated(subaccountId1, state))
	})
}

func TestStateReconciler(t *testing.T) {
	teardownSuite := setupSuite(t)
	defer teardownSuite(t)

	t.Run("should schedule update of one resource after getting account data from CIS", func(t *testing.T) {
		teardownTest, brokerStorage := setupTestNilStorage(t)
		defer teardownTest(t)

		reconciler := createNewReconciler(brokerStorage)

		// given
		// initial event from a kyma resource, first runtime, no label
		reconciler.reconcileResourceUpdate(subaccountId1, runtimeId11, runtimeStateType{betaEnabled: ""})
		assert.Equal(t, 1, len(reconciler.inMemoryState))

		// queue should be empty since we have not got state from CIS
		assert.True(t, reconciler.syncQueue.IsEmpty())

		// then the same subaccount, second runtime, with false label
		reconciler.reconcileResourceUpdate(subaccountId1, runtimeId12, runtimeStateType{betaEnabled: "false"})
		assert.Equal(t, 1, len(reconciler.inMemoryState))
		assert.True(t, reconciler.syncQueue.IsEmpty())

		// then we get state from CIS (accounts)
		reconciler.reconcileCisAccount(subaccountId1, CisStateType{BetaEnabled: true, UsedForProduction: "NOT_USED_FOR_PRODUCTION", ModifiedDate: veryOldTime})

		// queue should contain one element
		assert.False(t, reconciler.syncQueue.IsEmpty())
		element, ok := reconciler.syncQueue.Extract()
		assert.True(t, ok)
		assert.Equal(t, subaccountId1, element.SubaccountID)
		assert.Equal(t, "true", element.BetaEnabled)
		assert.Equal(t, "NOT_USED_FOR_PRODUCTION", element.UsedForProduction)

		assert.True(t, reconciler.syncQueue.IsEmpty())
	})

	t.Run("should schedule update of one resource after getting account and event from CIS", func(t *testing.T) {
		teardownTest, brokerStorage := setupTestNilStorage(t)
		defer teardownTest(t)

		reconciler := createNewReconciler(brokerStorage)

		// given
		// initial event from a kyma resource, first runtime, no label
		reconciler.reconcileResourceUpdate(subaccountId1, runtimeId11, runtimeStateType{betaEnabled: "", usedForProduction: ""})
		assert.Equal(t, 1, len(reconciler.inMemoryState))

		// queue should be empty since we have not got state from CIS
		assert.True(t, reconciler.syncQueue.IsEmpty())

		// when the same subaccount, second runtime, with false label
		reconciler.reconcileResourceUpdate(subaccountId1, runtimeId12, runtimeStateType{betaEnabled: "false", usedForProduction: "NOT_USED_FOR_PRODUCTION"})
		// then
		assert.Equal(t, 1, len(reconciler.inMemoryState))
		assert.True(t, reconciler.syncQueue.IsEmpty())

		// when we get state from CIS (accounts)
		reconciler.reconcileCisAccount(subaccountId1, CisStateType{BetaEnabled: false, UsedForProduction: "NOT_USED_FOR_PRODUCTION", ModifiedDate: 1})
		// then queue should contain one element
		assert.False(t, reconciler.syncQueue.IsEmpty())

		// when we get recent event from CIS with true label

		reconciler.reconcileCisEvent(fixCisUpdateEvent(subaccountId1, "true", "USED_FOR_PRODUCTION", oldTime))

		// then queue should contain one element
		assert.False(t, reconciler.syncQueue.IsEmpty())

		element, ok := reconciler.syncQueue.Extract()
		assert.True(t, ok)
		assert.Equal(t, subaccountId1, element.SubaccountID)
		assert.Equal(t, "true", element.BetaEnabled)
		assert.Equal(t, "USED_FOR_PRODUCTION", element.UsedForProduction)

		assert.True(t, reconciler.syncQueue.IsEmpty())

	})

	t.Run("should schedule update of one resource after getting event and account data from CIS", func(t *testing.T) {
		teardownTest, brokerStorage := setupTestNilStorage(t)
		defer teardownTest(t)

		reconciler := createNewReconciler(brokerStorage)

		// given
		// initial event from a kyma resource, first runtime, no label
		reconciler.reconcileResourceUpdate(subaccountId1, runtimeId11, runtimeStateType{betaEnabled: "", usedForProduction: ""})
		assert.Equal(t, 1, len(reconciler.inMemoryState))

		// queue should be empty since we have not got state from CIS
		assert.True(t, reconciler.syncQueue.IsEmpty())

		// when the same subaccount, second runtime, with false label
		reconciler.reconcileResourceUpdate(subaccountId1, runtimeId12, runtimeStateType{betaEnabled: "false", usedForProduction: "NOT_USED_FOR_PRODUCTION"})
		// then
		assert.Equal(t, 1, len(reconciler.inMemoryState))
		assert.True(t, reconciler.syncQueue.IsEmpty())

		// when we get recent event from CIS with true label
		reconciler.reconcileCisEvent(fixCisUpdateEvent(subaccountId1, "true", "USED_FOR_PRODUCTION", oldTime))

		// then queue should contain one element
		assert.False(t, reconciler.syncQueue.IsEmpty())

		// when we get very old state from CIS (accounts)
		cisState := CisStateType{BetaEnabled: false, UsedForProduction: "false", ModifiedDate: veryOldTime}
		reconciler.reconcileCisAccount(subaccountId1, cisState)
		// then queue should still contain one element
		assert.False(t, reconciler.syncQueue.IsEmpty())

		element, ok := reconciler.syncQueue.Extract()
		assert.True(t, ok)
		assert.Equal(t, subaccountId1, element.SubaccountID)
		assert.Equal(t, "true", element.BetaEnabled)
		assert.Equal(t, "USED_FOR_PRODUCTION", element.UsedForProduction)

		assert.True(t, reconciler.syncQueue.IsEmpty())
	})
	t.Run("should schedule update after getting event and then gets account data", func(t *testing.T) {
		teardownTest, brokerStorage := setupTestNilStorage(t)
		defer teardownTest(t)

		reconciler := createNewReconciler(brokerStorage)

		// given
		// initial event from a kyma resource, first runtime, no label
		reconciler.reconcileResourceUpdate(subaccountId1, runtimeId11, runtimeStateType{betaEnabled: "", usedForProduction: ""})
		assert.Equal(t, 1, len(reconciler.inMemoryState))

		// queue should be empty since we have not got state from CIS
		assert.True(t, reconciler.syncQueue.IsEmpty())

		// when the same subaccount, second runtime, with false label
		reconciler.reconcileResourceUpdate(subaccountId1, runtimeId12, runtimeStateType{betaEnabled: "false", usedForProduction: "NOT_USED_FOR_PRODUCTION"})
		// then
		assert.Equal(t, 1, len(reconciler.inMemoryState))
		assert.True(t, reconciler.syncQueue.IsEmpty())

		// when we get recent event from CIS with true label
		reconciler.reconcileCisEvent(fixCisUpdateEvent(subaccountId1, "true", "USED_FOR_PRODUCTION", oldTime))

		// then queue should contain one element
		assert.False(t, reconciler.syncQueue.IsEmpty())

		element, ok := reconciler.syncQueue.Extract()
		assert.True(t, ok)
		assert.Equal(t, subaccountId1, element.SubaccountID)
		assert.Equal(t, "true", element.BetaEnabled)
		assert.Equal(t, "USED_FOR_PRODUCTION", element.UsedForProduction)
		assert.True(t, reconciler.syncQueue.IsEmpty())

		// when we get older state from CIS (accounts)
		reconciler.reconcileCisAccount(subaccountId1, CisStateType{BetaEnabled: false, UsedForProduction: "NOT_USED_FOR_PRODUCTION", ModifiedDate: 1})

		// then queue should still be empty since we used more recent event to update the resource
		assert.True(t, reconciler.syncQueue.IsEmpty())

	})
	t.Run("should schedule update after each event", func(t *testing.T) {
		teardownTest, brokerStorage := setupTestNilStorage(t)
		defer teardownTest(t)

		reconciler := createNewReconciler(brokerStorage)

		// given
		// initial event from a kyma resource, first runtime, no label
		reconciler.reconcileResourceUpdate(subaccountId1, runtimeId11, runtimeStateType{betaEnabled: "", usedForProduction: ""})
		assert.Equal(t, 1, len(reconciler.inMemoryState))

		// queue should be empty since we have not got state from CIS
		assert.True(t, reconciler.syncQueue.IsEmpty())

		// then the same subaccount, second runtime, with false label
		reconciler.reconcileResourceUpdate(subaccountId1, runtimeId12, runtimeStateType{betaEnabled: "false", usedForProduction: "NOT_USED_FOR_PRODUCTION"})
		assert.Equal(t, 1, len(reconciler.inMemoryState))
		assert.True(t, reconciler.syncQueue.IsEmpty())

		// then we get event from CIS with false label
		reconciler.reconcileCisEvent(fixCisUpdateEvent(subaccountId1, "false", "NOT_USED_FOR_PRODUCTION", oldTime))

		// queue should contain one element
		assert.False(t, reconciler.syncQueue.IsEmpty())
		element, ok := reconciler.syncQueue.Extract()
		assert.True(t, ok)
		assert.Equal(t, subaccountId1, element.SubaccountID)
		assert.Equal(t, "false", element.BetaEnabled)

		assert.True(t, reconciler.syncQueue.IsEmpty())

		// then we get event from CIS with false label

		reconciler.reconcileCisEvent(fixCisUpdateEvent(subaccountId1, "true", "USED_FOR_PRODUCTION", notSoOldTime))

		// queue should contain one element
		assert.False(t, reconciler.syncQueue.IsEmpty())
		element, ok = reconciler.syncQueue.Extract()
		assert.True(t, ok)
		assert.Equal(t, subaccountId1, element.SubaccountID)
		assert.Equal(t, "true", element.BetaEnabled)
		assert.Equal(t, "USED_FOR_PRODUCTION", element.UsedForProduction)

		assert.True(t, reconciler.syncQueue.IsEmpty())
	})
	t.Run("should schedule update after two consecutive events", func(t *testing.T) {
		teardownTest, brokerStorage := setupTestNilStorage(t)
		defer teardownTest(t)

		reconciler := createNewReconciler(brokerStorage)

		// given
		// initial event from a kyma resource, first runtime, no label
		reconciler.reconcileResourceUpdate(subaccountId1, runtimeId11, runtimeStateType{betaEnabled: "", usedForProduction: ""})
		assert.Equal(t, 1, len(reconciler.inMemoryState))

		// queue should be empty since we have not got state from CIS
		assert.True(t, reconciler.syncQueue.IsEmpty())

		// then the same subaccount, second runtime, with false label
		reconciler.reconcileResourceUpdate(subaccountId1, runtimeId12, runtimeStateType{betaEnabled: "false", usedForProduction: "NOT_USED_FOR_PRODUCTION"})
		assert.Equal(t, 1, len(reconciler.inMemoryState))
		assert.True(t, reconciler.syncQueue.IsEmpty())

		// then we get event from CIS with false label
		reconciler.reconcileCisEvent(fixCisUpdateEvent(subaccountId1, "false", "NOT_USED_FOR_PRODUCTION", oldTime))

		// queue should contain one element
		assert.False(t, reconciler.syncQueue.IsEmpty())

		// then we get event from CIS with false label
		reconciler.reconcileCisEvent(fixCisUpdateEvent(subaccountId1, "true", "USED_FOR_PRODUCTION", notSoOldTime))

		// queue should contain one element
		assert.False(t, reconciler.syncQueue.IsEmpty())
		element, ok := reconciler.syncQueue.Extract()
		assert.True(t, ok)
		assert.Equal(t, subaccountId1, element.SubaccountID)
		assert.Equal(t, "true", element.BetaEnabled)
		assert.Equal(t, "USED_FOR_PRODUCTION", element.UsedForProduction)

		assert.True(t, reconciler.syncQueue.IsEmpty())
	})
	t.Run("should schedule update after two consecutive events in reversed order", func(t *testing.T) {
		teardownTest, brokerStorage := setupTestNilStorage(t)
		defer teardownTest(t)

		reconciler := createNewReconciler(brokerStorage)

		// given
		// initial event from a kyma resource, first runtime, no label
		reconciler.reconcileResourceUpdate(subaccountId1, runtimeId11, runtimeStateType{betaEnabled: "", usedForProduction: ""})
		assert.Equal(t, 1, len(reconciler.inMemoryState))

		// queue should be empty since we have not got state from CIS
		assert.True(t, reconciler.syncQueue.IsEmpty())

		// then the same subaccount, second runtime, with false label
		reconciler.reconcileResourceUpdate(subaccountId1, runtimeId12, runtimeStateType{betaEnabled: "false", usedForProduction: "NOT_USED_FOR_PRODUCTION"})
		assert.Equal(t, 1, len(reconciler.inMemoryState))
		assert.True(t, reconciler.syncQueue.IsEmpty())

		// then we get recent event from CIS with true label
		reconciler.reconcileCisEvent(fixCisUpdateEvent(subaccountId1, "true", "USED_FOR_PRODUCTION", notSoOldTime))

		// then we get older event from CIS with false label
		reconciler.reconcileCisEvent(fixCisUpdateEvent(subaccountId1, "false", "NOT_USED_FOR_PRODUCTION", oldTime))

		// queue should contain one element
		assert.False(t, reconciler.syncQueue.IsEmpty())

		// queue should contain one element
		assert.False(t, reconciler.syncQueue.IsEmpty())
		element, ok := reconciler.syncQueue.Extract()
		assert.True(t, ok)
		assert.Equal(t, subaccountId1, element.SubaccountID)
		assert.Equal(t, "true", element.BetaEnabled)
		assert.Equal(t, "USED_FOR_PRODUCTION", element.UsedForProduction)

		assert.True(t, reconciler.syncQueue.IsEmpty())
	})
	t.Run("should ignore outdated event after more recent one - before update", func(t *testing.T) {
		teardownTest, brokerStorage := setupTestNilStorage(t)
		defer teardownTest(t)

		reconciler := createNewReconciler(brokerStorage)

		// given
		// initial event from a kyma resource, first runtime, no label
		reconciler.reconcileResourceUpdate(subaccountId1, runtimeId11, runtimeStateType{betaEnabled: "", usedForProduction: ""})
		assert.Equal(t, 1, len(reconciler.inMemoryState))

		// queue should be empty since we have not got state from CIS
		assert.True(t, reconciler.syncQueue.IsEmpty())

		// then we get recent event from CIS with true label
		reconciler.reconcileCisEvent(fixCisUpdateEvent(subaccountId1, "true", "USED_FOR_PRODUCTION", notSoOldTime))

		// queue should contain one element
		assert.False(t, reconciler.syncQueue.IsEmpty())
		element, ok := reconciler.syncQueue.Extract()
		assert.True(t, ok)
		assert.Equal(t, subaccountId1, element.SubaccountID)
		assert.Equal(t, "true", element.BetaEnabled)
		assert.Equal(t, "USED_FOR_PRODUCTION", element.UsedForProduction)

		// then we get older event from CIS with false label, we do not know if real update already happened
		reconciler.reconcileCisEvent(fixCisUpdateEvent(subaccountId1, "false", "NOT_USED_FOR_PRODUCTION", oldTime))

		// queue should be empty since we used more recent event to update the resource
		assert.True(t, reconciler.syncQueue.IsEmpty())
	})
	t.Run("should ignore outdated event after more recent was applied", func(t *testing.T) {
		teardownTest, brokerStorage := setupTestNilStorage(t)
		defer teardownTest(t)

		reconciler := createNewReconciler(brokerStorage)

		// given
		// initial event from a kyma resource, first runtime, no label
		reconciler.reconcileResourceUpdate(subaccountId1, runtimeId11, runtimeStateType{betaEnabled: "", usedForProduction: ""})
		assert.Equal(t, 1, len(reconciler.inMemoryState))

		// queue should be empty since we have not got state from CIS
		assert.True(t, reconciler.syncQueue.IsEmpty())

		// then we get recent event from CIS with true label
		reconciler.reconcileCisEvent(fixCisUpdateEvent(subaccountId1, "true", "USED_FOR_PRODUCTION", notSoOldTime))

		// queue should contain one element
		assert.False(t, reconciler.syncQueue.IsEmpty())
		element, ok := reconciler.syncQueue.Extract()
		assert.True(t, ok)
		assert.Equal(t, subaccountId1, element.SubaccountID)
		assert.Equal(t, "true", element.BetaEnabled)
		assert.Equal(t, "USED_FOR_PRODUCTION", element.UsedForProduction)

		// then we got confirmation that the update was applied
		reconciler.reconcileResourceUpdate(subaccountId1, runtimeId11, runtimeStateType{betaEnabled: "true", usedForProduction: "USED_FOR_PRODUCTION"})

		// then we get older event from CIS with false label
		reconciler.reconcileCisEvent(fixCisUpdateEvent(subaccountId1, "false", "NOT_USED_FOR_PRODUCTION", oldTime))

		// queue should be empty since we used more recent event to update the resource
		assert.True(t, reconciler.syncQueue.IsEmpty())
	})
	t.Run("should update resource after restart regardless of lost event", func(t *testing.T) {
		teardownTest, brokerStorage := setupTestWithStorage(t)
		defer teardownTest(t)

		reconciler := createNewReconciler(brokerStorage)

		// given
		reconciler.recreateStateFromDB()
		assert.True(t, reconciler.syncQueue.IsEmpty())
		assert.Equal(t, 0, len(reconciler.inMemoryState))

		// when
		// initial add event from a kyma resource, first runtime, no label
		reconciler.reconcileResourceUpdate(subaccountId1, runtimeId11, runtimeStateType{betaEnabled: "", usedForProduction: ""})
		assert.Equal(t, 1, len(reconciler.inMemoryState))

		// then we fetch state from accounts endpoint
		reconciler.reconcileCisAccount(subaccountId1, CisStateType{BetaEnabled: false, UsedForProduction: "NOT_USED_FOR_PRODUCTION", ModifiedDate: veryOldTime})

		// queue should contain one element
		assert.False(t, reconciler.syncQueue.IsEmpty())

		// then comes app restart
		newReconciler := createNewReconciler(brokerStorage)
		// we lost the event, lost the state, lost the queue
		newReconciler.recreateStateFromDB()
		// when
		// initial add event from a kyma resource, first runtime, no label
		reconciler.reconcileResourceUpdate(subaccountId1, runtimeId11, runtimeStateType{betaEnabled: "", usedForProduction: ""})
		reconciler.reconcileCisAccount(subaccountId1, CisStateType{BetaEnabled: false, UsedForProduction: "NOT_USED_FOR_PRODUCTION", ModifiedDate: veryOldTime})
		// queue should contain one element again

		assert.True(t, newReconciler.syncQueue.IsEmpty())
	})
	t.Run("should update resource after restart", func(t *testing.T) {
		teardownTest, brokerStorage := setupTestWithStorage(t)
		defer teardownTest(t)

		reconciler := createNewReconciler(brokerStorage)

		reconciler.recreateStateFromDB()
		assert.True(t, reconciler.syncQueue.IsEmpty())
		assert.Equal(t, 0, len(reconciler.inMemoryState))

		// when
		// initial add event from a kyma resource, first runtime, no label
		reconciler.reconcileResourceUpdate(subaccountId1, runtimeId11, runtimeStateType{betaEnabled: "", usedForProduction: ""})
		assert.Equal(t, 1, len(reconciler.inMemoryState))

		// then we fetch state from accounts endpoint
		reconciler.reconcileCisAccount(subaccountId1, CisStateType{BetaEnabled: false, UsedForProduction: "NOT_USED_FOR_PRODUCTION", ModifiedDate: veryOldTime})

		// queue should contain one element
		assert.False(t, reconciler.syncQueue.IsEmpty())

		// then comes app restart
		newReconciler := createNewReconciler(brokerStorage)
		// we lost the state, lost the queue
		newReconciler.recreateStateFromDB()
		// when
		// initial add event from the kyma resource, first runtime, no label
		reconciler.reconcileResourceUpdate(subaccountId1, runtimeId11, runtimeStateType{betaEnabled: ""})
		reconciler.reconcileCisAccount(subaccountId1, CisStateType{BetaEnabled: false, UsedForProduction: "NOT_USED_FOR_PRODUCTION", ModifiedDate: veryOldTime})
		// queue should contain one element again
		assert.False(t, reconciler.syncQueue.IsEmpty())

		element, ok := reconciler.syncQueue.Extract()
		assert.True(t, ok)
		assert.Equal(t, subaccountId1, element.SubaccountID)
		assert.True(t, reconciler.syncQueue.IsEmpty())

		// then we get update event from informer
		reconciler.reconcileResourceUpdate(subaccountId1, runtimeId11, runtimeStateType{betaEnabled: "false", usedForProduction: "NOT_USED_FOR_PRODUCTION"})

		// then there is nothing to do
		assert.True(t, newReconciler.syncQueue.IsEmpty())
	})
	t.Run("should update resource before restart", func(t *testing.T) {
		teardownTest, brokerStorage := setupTestWithStorage(t)
		defer teardownTest(t)

		reconciler := createNewReconciler(brokerStorage)

		// given
		reconciler.recreateStateFromDB()
		assert.True(t, reconciler.syncQueue.IsEmpty())
		assert.Equal(t, 0, len(reconciler.inMemoryState))

		// when
		// initial add event from a kyma resource, first runtime, no label
		reconciler.reconcileResourceUpdate(subaccountId1, runtimeId11, runtimeStateType{betaEnabled: "", usedForProduction: ""})
		assert.Equal(t, 1, len(reconciler.inMemoryState))

		// then we fetch state from accounts endpoint
		reconciler.reconcileCisAccount(subaccountId1, CisStateType{BetaEnabled: false, UsedForProduction: "NOT_USED_FOR_PRODUCTION", ModifiedDate: veryOldTime})

		// queue should contain one element
		assert.False(t, reconciler.syncQueue.IsEmpty())

		// then we get old event from CIS with true label
		reconciler.reconcileCisEvent(fixCisUpdateEvent(subaccountId1, "true", "USED_FOR_PRODUCTION", oldTime))

		// then queue should contain one element
		assert.False(t, reconciler.syncQueue.IsEmpty())

		// we extract the element
		element, ok := reconciler.syncQueue.Extract()
		assert.True(t, ok)
		assert.Equal(t, subaccountId1, element.SubaccountID)
		assert.Equal(t, "true", element.BetaEnabled)
		assert.Equal(t, "USED_FOR_PRODUCTION", element.UsedForProduction)

		// and update resource so update event comes from informer
		reconciler.reconcileResourceUpdate(subaccountId1, runtimeId11, runtimeStateType{betaEnabled: "true", usedForProduction: "USED_FOR_PRODUCTION"})

		// then comes app restart
		newReconciler := createNewReconciler(brokerStorage)
		// we lost the event, lost the state, lost the queue
		newReconciler.recreateStateFromDB()
		// when
		// initial add event from the kyma resource, first runtime, no label
		reconciler.reconcileResourceUpdate(subaccountId1, runtimeId11, runtimeStateType{betaEnabled: "true", usedForProduction: "USED_FOR_PRODUCTION"})
		reconciler.reconcileCisAccount(subaccountId1, CisStateType{BetaEnabled: true, UsedForProduction: "USED_FOR_PRODUCTION", ModifiedDate: oldTime})
		// queue should be empty - no difference, resource is up-to-date
		assert.True(t, reconciler.syncQueue.IsEmpty())
	})
	t.Run("should schedule resource update", func(t *testing.T) {
		teardownTest, brokerStorage := setupTestWithStorage(t)
		defer teardownTest(t)

		reconciler := createNewReconciler(brokerStorage)

		// given
		// initial event from a kyma resource, first runtime, no label
		reconciler.reconcileResourceUpdate(subaccountId1, runtimeId11, runtimeStateType{betaEnabled: "", usedForProduction: ""})
		assert.Equal(t, 1, len(reconciler.inMemoryState))

		// queue should be empty since we have not got state from CIS
		assert.True(t, reconciler.syncQueue.IsEmpty())

		// then the same subaccount, second runtime, with false label
		reconciler.reconcileResourceUpdate(subaccountId1, runtimeId12, runtimeStateType{betaEnabled: "false", usedForProduction: "NOT_USED_FOR_PRODUCTION"})
		assert.Equal(t, 1, len(reconciler.inMemoryState))
		assert.True(t, reconciler.syncQueue.IsEmpty())

		// then we get state from CIS (accounts)
		reconciler.reconcileCisAccount(subaccountId1, CisStateType{BetaEnabled: false, UsedForProduction: "NOT_USED_FOR_PRODUCTION", ModifiedDate: 1})
		// queue should contain one element
		assert.False(t, reconciler.syncQueue.IsEmpty())
		element, ok := reconciler.syncQueue.Extract()
		assert.True(t, ok)
		assert.Equal(t, subaccountId1, element.SubaccountID)
		assert.Equal(t, "false", element.BetaEnabled)
		assert.Equal(t, "NOT_USED_FOR_PRODUCTION", element.UsedForProduction)

		assert.True(t, reconciler.syncQueue.IsEmpty())
	})
	t.Run("should schedule two resource updates in proper order", func(t *testing.T) {
		// informer sends two events
		// then we get accounts
		// then we get event from CIS for the second subaccount

		teardownTest, brokerStorage := setupTestWithStorage(t)
		defer teardownTest(t)

		reconciler := createNewReconciler(brokerStorage)

		// given
		// initial event from a kyma resource, first runtime, no label
		reconciler.reconcileResourceUpdate(subaccountId1, runtimeId11, runtimeStateType{betaEnabled: "", usedForProduction: ""})
		assert.Equal(t, 1, len(reconciler.inMemoryState))

		// initial event from a second kyma resource, first runtime, no label
		reconciler.reconcileResourceUpdate(subaccountId2, runtimeId21, runtimeStateType{betaEnabled: "", usedForProduction: ""})
		assert.Equal(t, 2, len(reconciler.inMemoryState))

		// queue should be empty since we have not got state from CIS
		assert.True(t, reconciler.syncQueue.IsEmpty())

		// then the same subaccount, second runtime, with false label
		reconciler.reconcileResourceUpdate(subaccountId1, runtimeId12, runtimeStateType{betaEnabled: "false", usedForProduction: "NOT_USED_FOR_PRODUCTION"})
		assert.Equal(t, 2, len(reconciler.inMemoryState))
		assert.True(t, reconciler.syncQueue.IsEmpty())

		// then we get state from CIS (accounts)
		reconciler.reconcileCisAccount(subaccountId1, CisStateType{BetaEnabled: false, UsedForProduction: "NOT_USED_FOR_PRODUCTION", ModifiedDate: veryOldTime})

		// queue should contain only one element
		assert.False(t, reconciler.syncQueue.IsEmpty())

		// then we get event from CIS for the second subaccount
		reconciler.reconcileCisAccount(subaccountId2, CisStateType{BetaEnabled: true, UsedForProduction: "NOT_SET", ModifiedDate: oldTime})

		element1, ok := reconciler.syncQueue.Extract()
		assert.True(t, ok)

		assert.Equal(t, subaccountId1, element1.SubaccountID)
		assert.Equal(t, "false", element1.BetaEnabled)
		assert.Equal(t, "NOT_USED_FOR_PRODUCTION", element1.UsedForProduction)

		element2, ok := reconciler.syncQueue.Extract()
		assert.True(t, ok)
		assert.Equal(t, subaccountId2, element2.SubaccountID)
		assert.Equal(t, "true", element2.BetaEnabled)
		assert.Equal(t, "NOT_SET", element2.UsedForProduction)

		assert.True(t, reconciler.syncQueue.IsEmpty())
	})

	t.Run("should recreate state from DB", func(t *testing.T) {
		teardownTest, brokerStorage := setupTestWithStorage(t)
		defer teardownTest(t)

		reconciler := createNewReconciler(brokerStorage)

		// when there are two subaccounts with current CIS states and one without CIS state
		reconciler.reconcileResourceUpdate(subaccountId1, runtimeId11, runtimeStateType{betaEnabled: ""})
		reconciler.reconcileResourceUpdate(subaccountId2, runtimeId21, runtimeStateType{betaEnabled: ""})
		reconciler.reconcileResourceUpdate(subaccountId3, runtimeId31, runtimeStateType{betaEnabled: ""})
		reconciler.reconcileCisAccount(subaccountId1, CisStateType{BetaEnabled: false, UsedForProduction: "NOT_SET", ModifiedDate: 1})
		reconciler.reconcileCisAccount(subaccountId2, CisStateType{BetaEnabled: true, UsedForProduction: "NOT_SET", ModifiedDate: 2})
		reconciler.reconcileCisAccount(subaccountId3, CisStateType{BetaEnabled: true, UsedForProduction: "USED_FOR_PRODUCTION", ModifiedDate: 2})
		// then

		knownSubaccounts := reconciler.getAllSubaccountIDsFromState()
		assert.Equal(t, 3, len(knownSubaccounts))
		assert.Contains(t, knownSubaccounts, subaccountIDType(subaccountId1))
		assert.Contains(t, knownSubaccounts, subaccountIDType(subaccountId2))
		assert.Contains(t, knownSubaccounts, subaccountIDType(subaccountId3))

		// then
		reconciler.storeStateInDb()

		reconcilerAfterReset := createNewReconciler(brokerStorage)

		reconcilerAfterReset.recreateStateFromDB()
		knownSubaccounts = reconciler.getAllSubaccountIDsFromState()

		// then
		assert.Equal(t, 3, len(knownSubaccounts))
		assert.Contains(t, knownSubaccounts, subaccountIDType(subaccountId1))
		assert.Contains(t, knownSubaccounts, subaccountIDType(subaccountId2))
		assert.Contains(t, knownSubaccounts, subaccountIDType(subaccountId3))
		assert.Equal(t, 3, len(reconciler.inMemoryState))
	})

	t.Run("should recreate state from DB despite inconsistency between states and instances", func(t *testing.T) {
		// three subaccounts in the instances table
		// three subaccounts in the subaccount_states, but different set

		teardownTest, brokerStorage := setupTestWithStorage(t)
		defer teardownTest(t)

		reconciler := createNewReconciler(brokerStorage)

		// given
		fixInstancesTableWithFourInstancesAndThreeRuntimes(t, reconciler.db)
		fixSubaccountStatesTable(t, reconciler.db,
			internal.SubaccountState{ID: subaccountId1, BetaEnabled: "true", UsedForProduction: "NOT_SET", ModifiedAt: oldTime},
			internal.SubaccountState{ID: subaccountId2, BetaEnabled: "false", UsedForProduction: "NOT_SET", ModifiedAt: notSoOldTime},
			internal.SubaccountState{ID: subaccountId4, BetaEnabled: "true", UsedForProduction: "USED_FOR_PRODUCTION", ModifiedAt: recent})

		// when
		reconciler.recreateStateFromDB()

		// then
		knownSubaccounts := reconciler.getAllSubaccountIDsFromState()
		assert.Equal(t, 4, len(knownSubaccounts))
		assert.Contains(t, knownSubaccounts, subaccountIDType(subaccountId1))
		assert.Contains(t, knownSubaccounts, subaccountIDType(subaccountId2))
		assert.Contains(t, knownSubaccounts, subaccountIDType(subaccountId3))
		assert.Contains(t, knownSubaccounts, subaccountIDType(subaccountId4))
		assert.Equal(t, 4, len(reconciler.inMemoryState))

		// and when
		reconciler.storeStateInDb()

		// and app restarts

		reconcilerAfterReset := createNewReconciler(brokerStorage)

		reconcilerAfterReset.recreateStateFromDB()

		// then we remove state which is not in the instances table
		knownSubaccounts = reconciler.getAllSubaccountIDsFromState()
		assert.Equal(t, 3, len(knownSubaccounts))
		assert.Contains(t, knownSubaccounts, subaccountIDType(subaccountId1))
		assert.Contains(t, knownSubaccounts, subaccountIDType(subaccountId2))
		assert.Contains(t, knownSubaccounts, subaccountIDType(subaccountId3))
		assert.Equal(t, 3, len(reconciler.inMemoryState))
	})
	t.Run("should handle resource deletion and remove the state", func(t *testing.T) {
		teardownTest, brokerStorage := setupTestWithStorage(t)
		defer teardownTest(t)

		reconciler := createNewReconciler(brokerStorage)

		// given
		// initial event from a kyma resource, first runtime, no label
		reconciler.reconcileResourceUpdate(subaccountId1, runtimeId11, runtimeStateType{betaEnabled: ""})
		assert.Equal(t, 1, len(reconciler.inMemoryState))

		// queue should be empty since we have not got state from CIS
		assert.True(t, reconciler.syncQueue.IsEmpty())

		// when the same subaccount, second runtime, with false label
		reconciler.reconcileResourceUpdate(subaccountId1, runtimeId12, runtimeStateType{betaEnabled: "false"})
		// then
		assert.Equal(t, 1, len(reconciler.inMemoryState))
		assert.True(t, reconciler.syncQueue.IsEmpty())

		// then we get deleted event from informer
		reconciler.deleteRuntimeFromState(subaccountId1, runtimeId11)

		// then we should still keep the state
		assert.Equal(t, 1, len(reconciler.inMemoryState))

		// when we get the last runtime deleted
		reconciler.deleteRuntimeFromState(subaccountId1, runtimeId12)

		// then we should still keep the state
		assert.Equal(t, 1, len(reconciler.inMemoryState))

		// then we store the state in the db, as side effect we remove state from the memory
		reconciler.storeStateInDb()

		// then we should have no state in the memory
		assert.Equal(t, 0, len(reconciler.inMemoryState))
		assert.True(t, reconciler.syncQueue.IsEmpty())

		// then runtime is added again and we recreate the state
		reconciler.reconcileResourceUpdate(subaccountId1, runtimeId11, runtimeStateType{betaEnabled: ""})
		assert.Equal(t, 1, len(reconciler.inMemoryState))
	})
	t.Run("should handle resource update and update the state", func(t *testing.T) {
		teardownTest, brokerStorage := setupTestWithStorage(t)
		defer teardownTest(t)

		reconciler := createNewReconciler(brokerStorage)

		// given
		// initial event from a kyma resource, first runtime, no label
		reconciler.reconcileResourceUpdate(subaccountId1, runtimeId11, runtimeStateType{betaEnabled: ""})
		assert.Equal(t, 1, len(reconciler.inMemoryState))

		// queue should be empty since we have not got state from CIS
		assert.True(t, reconciler.syncQueue.IsEmpty())

		// when the same subaccount, first runtime, with false label
		reconciler.reconcileResourceUpdate(subaccountId1, runtimeId11, runtimeStateType{betaEnabled: "false"})
		// then
		assert.Equal(t, 1, len(reconciler.inMemoryState))
		assert.True(t, reconciler.syncQueue.IsEmpty())

		// and when the same subaccount, first runtime, with false label
		reconciler.reconcileResourceUpdate(subaccountId1, runtimeId11, runtimeStateType{betaEnabled: "true"})

		// then
		assert.Equal(t, 1, len(reconciler.inMemoryState))
		assert.True(t, reconciler.syncQueue.IsEmpty())

		// but then comes account from CIS
		reconciler.reconcileCisAccount(subaccountId1, CisStateType{BetaEnabled: false, UsedForProduction: "NOT_SET", ModifiedDate: veryOldTime})
		assert.False(t, reconciler.syncQueue.IsEmpty())

		// then someone changes label
		reconciler.reconcileResourceUpdate(subaccountId1, runtimeId11, runtimeStateType{betaEnabled: "false"})

		// but we have the update in the queue already even if it is futile (false to false) - this is current behavior
		assert.False(t, reconciler.syncQueue.IsEmpty())
	})
	t.Run("should handle subaccount creation - resource already with proper label value", func(t *testing.T) {
		// caveat - if the subaccount was neither listed in the instances nor in the previous state, we won't notice the event
		// the other approach would be to watch all subaccounts coming in the events, possible but not feasible
		teardownTest, brokerStorage := setupTestWithStorage(t)
		defer teardownTest(t)

		reconciler := createNewReconciler(brokerStorage)

		// given
		fixInstancesTableWithOneInstance(t, reconciler.db)

		// and given previous state empty
		// when
		reconciler.recreateStateFromDB()

		// then assuming someone added subaccountId3 to the instances table when app was down - hence we have not got it in the previous state
		assert.Equal(t, 1, len(reconciler.inMemoryState))

		// and the resource is created already but with desired label
		reconciler.reconcileResourceUpdate(subaccountId1, runtimeId11, runtimeStateType{betaEnabled: "true"})

		// then we query the accounts
		reconciler.reconcileCisAccount(subaccountId3, CisStateType{BetaEnabled: true, UsedForProduction: "NOT_SET", ModifiedDate: veryOldTime})
		assert.True(t, reconciler.syncQueue.IsEmpty())

		reconciler.reconcileCisEvent(fixCisCreateEvent(subaccountId3, "true", "NOT_SET", veryOldTime))

		// then
		// queue should be empty
		assert.True(t, reconciler.syncQueue.IsEmpty())
	})
	t.Run("should handle subaccount creation - resource without label", func(t *testing.T) {
		teardownTest, brokerStorage := setupTestWithStorage(t)
		defer teardownTest(t)

		reconciler := createNewReconciler(brokerStorage)

		// given
		fixInstancesTableWithOneInstance(t, reconciler.db)

		// and given previous state empty
		// when
		reconciler.recreateStateFromDB()

		// then assuming someone added subaccountId3 to the instances table when app was down - hence we have not got it in the previous state
		assert.Equal(t, 1, len(reconciler.inMemoryState))

		// and the resource is created already, but with not desired label
		reconciler.reconcileResourceUpdate(subaccountId3, runtimeId31, runtimeStateType{betaEnabled: ""})

		// then we query the accounts
		reconciler.reconcileCisAccount(subaccountId3, CisStateType{BetaEnabled: true, UsedForProduction: "NOT_SET", ModifiedDate: veryOldTime})
		assert.False(t, reconciler.syncQueue.IsEmpty())

		// then we get the creation event
		reconciler.reconcileCisEvent(fixCisCreateEvent(subaccountId3, "true", "NOT_SET", veryOldTime))

		// then
		// queue should contain one element
		assert.False(t, reconciler.syncQueue.IsEmpty())
		element, ok := reconciler.syncQueue.Extract()
		assert.True(t, ok)
		assert.Equal(t, subaccountId3, element.SubaccountID)
		assert.Equal(t, "true", element.BetaEnabled)
		assert.Equal(t, "NOT_SET", element.UsedForProduction)
	})
	t.Run("should handle subaccount creation - no resource yet", func(t *testing.T) {
		teardownTest, brokerStorage := setupTestWithStorage(t)
		defer teardownTest(t)

		reconciler := createNewReconciler(brokerStorage)

		// given
		fixInstancesTableWithOneInstance(t, reconciler.db)

		// and given previous state empty
		// when
		reconciler.recreateStateFromDB()

		// then assuming someone added subaccountId3 to the instances table when app was down - hence we have not got it in the previous state
		assert.Equal(t, 1, len(reconciler.inMemoryState))

		// and the resource is not there
		// then we query the accounts
		reconciler.reconcileCisAccount(subaccountId3, CisStateType{BetaEnabled: true, UsedForProduction: "NOT_SET", ModifiedDate: veryOldTime})
		assert.True(t, reconciler.syncQueue.IsEmpty())

		reconciler.reconcileCisEvent(fixCisCreateEvent(subaccountId3, "true", "NOT_SET", veryOldTime))

		// then
		// queue should be empty since there is nothing to be updated
		assert.True(t, reconciler.syncQueue.IsEmpty())

		// but then the resource is created, but without label
		reconciler.reconcileResourceUpdate(subaccountId3, runtimeId31, runtimeStateType{betaEnabled: ""})
		assert.False(t, reconciler.syncQueue.IsEmpty())

		element, ok := reconciler.syncQueue.Extract()
		assert.True(t, ok)
		assert.Equal(t, subaccountId3, element.SubaccountID)
		assert.Equal(t, "true", element.BetaEnabled)
		assert.Equal(t, "NOT_SET", element.UsedForProduction)
	})

}

// test fixtures

func createNewReconciler(storage storage.BrokerStorage) stateReconcilerType {
	return stateReconcilerType{
		inMemoryState: make(inMemoryStateType),
		mutex:         sync.Mutex{},
		logger:        logger,
		syncQueue:     queues.NewPriorityQueueWithCallbacks(logger, &queues.EventHandler{}),
		db:            storage,
	}
}

func createFakeRateLimitedCisClient(ctx context.Context, httpClient *http.Client, config CisEndpointConfig, log *slog.Logger) *RateLimitedCisClient {

	rl := rate.NewLimiter(rate.Every(config.RateLimitingInterval), config.MaxRequestsPerInterval)

	return &RateLimitedCisClient{
		ctx:         ctx,
		httpClient:  httpClient,
		config:      config,
		RateLimiter: rl,
		log:         log,
	}
}

func createNewReconcilerWithFakeCisServer(brokerStorage storage.BrokerStorage, client *http.Client, cisEndpointConfig CisEndpointConfig) stateReconcilerType {
	rtlClient := createFakeRateLimitedCisClient(context.Background(), client, cisEndpointConfig, logger)
	var epochInStubs = int64(1710748500000)
	return stateReconcilerType{
		inMemoryState:  make(inMemoryStateType),
		mutex:          sync.Mutex{},
		logger:         logger,
		syncQueue:      queues.NewPriorityQueueWithCallbacks(logger, &queues.EventHandler{}),
		db:             brokerStorage,
		accountsClient: rtlClient,
		eventsClient:   rtlClient,
		eventWindow: NewEventWindow(60*1000, func() int64 {
			return epochInStubs
		}),
	}
}

func fixInstancesTableWithFourInstancesAndThreeRuntimes(t *testing.T, brokerStorage storage.BrokerStorage) {
	require.NoError(t, brokerStorage.Instances().Insert(fixInstance("1", subaccountId1, runtimeId11)))
	require.NoError(t, brokerStorage.Instances().Insert(fixInstance("2", subaccountId1, runtimeId12)))
	require.NoError(t, brokerStorage.Instances().Insert(fixInstance("3", subaccountId2, runtimeId21)))
	require.NoError(t, brokerStorage.Instances().Insert(fixInstance("4", subaccountId3, runtimeId31)))
}

func fixInstancesTableWithOneInstance(t *testing.T, brokerStorage storage.BrokerStorage) {
	require.NoError(t, brokerStorage.Instances().Insert(fixInstance("1", subaccountId3, runtimeId31)))
}

func fixSubaccountStatesTable(t *testing.T, brokerStorage storage.BrokerStorage, subaccountStates ...internal.SubaccountState) {

	for _, state := range subaccountStates {
		require.NoError(t, brokerStorage.SubaccountStates().UpsertState(state))
	}
}

func fixInstance(id string, subaccountID string, runtimeID string) internal.Instance {
	return internal.Instance{
		InstanceID:      id,
		RuntimeID:       runtimeID,
		GlobalAccountID: fmt.Sprintf("GlobalAccountID field for SubAccountID: %s", subaccountID),
		SubAccountID:    subaccountID,
		ServiceID:       fmt.Sprintf("ServiceID field. IDX: %s", id),
		ServiceName:     fmt.Sprintf("ServiceName field. IDX: %s", id),
		ServicePlanID:   fmt.Sprintf("ServicePlanID field. IDX: %s", id),
		ServicePlanName: fmt.Sprintf("ServicePlanName field. IDX: %s", id),
		Parameters: internal.ProvisioningParameters{
			PlatformRegion: fmt.Sprintf("region-value-%s", id),
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func fixCisUpdateEvent(subaccountID string, betaEnabled string, usedForProduction string, actionTime int64) Event {
	return fixCisEvent(subaccountID, "Subaccount_Update", betaEnabled, usedForProduction, actionTime)
}

func fixCisCreateEvent(subaccountID string, betaEnabled string, usedForProduction string, actionTime int64) Event {
	return fixCisEvent(subaccountID, "Subaccount_Create", betaEnabled, usedForProduction, actionTime)
}

func fixCisEvent(subaccountID string, eventType string, betaEnabled string, usedForProduction string, actionTime int64) Event {
	return Event{
		ActionTime:   actionTime,
		SubaccountID: subaccountID,
		Type:         eventType,
		Details: EventDetails{
			BetaEnabled:       betaEnabled == "true",
			UsedForProduction: usedForProduction,
		},
	}
}

func brokerStorageTestConfig() storage.Config {
	return storage.Config{
		Host:            "localhost",
		User:            "test",
		Password:        "FIPS-compl1antPwd!",
		Port:            "5432",
		Name:            "test-sync",
		SSLMode:         "disable",
		SecretKey:       "################################",
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxLifetime: time.Minute,
	}
}

func getStorageForTests() (func() error, storage.BrokerStorage, error) {
	if useInMemoryStorage {
		return nil, storage.NewMemoryStorage(), nil
	}
	return storage.GetStorageForTest(brokerStorageTestConfig())
}

func setupStorageContainer() func() {
	config := brokerStorageTestConfig()

	docker, err := internal.NewDockerHandler()
	if err != nil {
		logger.Error(fmt.Sprintf("Error creating docker handler: %s, exiting...", err))
		os.Exit(1)
	}
	defer func(docker *internal.DockerHelper) {
		err := docker.CloseDockerClient()
		if err != nil {
			logger.Error(fmt.Sprintf("Error creating docker client: %s", err))
			os.Exit(1)
		}
	}(docker)

	cleanupContainer, err := docker.CreateDBContainer(internal.ContainerCreateRequest{
		Port:          config.Port,
		User:          config.User,
		Password:      config.Password,
		Name:          config.Name,
		Host:          config.Host,
		ContainerName: "subaccount-sync-tests",
		Image:         internal.PostgresImage,
	})
	return func() {
		if cleanupContainer != nil {
			err := cleanupContainer()
			if err != nil {
				logger.Error(fmt.Sprintf("Error cleaning container: %s", err))
				os.Exit(1)
			}
		}
	}
}
