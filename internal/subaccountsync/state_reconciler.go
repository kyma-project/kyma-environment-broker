package subaccountsync

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/go-co-op/gocron"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/syncqueues"
)

const (
	usedForProductionLabel    = "USED_FOR_PRODUCTION"
	notUsedForProductionLabel = "NOT_USED_FOR_PRODUCTION"
)

func isBetaEnabledTrue(val string) bool {
	return val == "true" //nolint:goconst
}

func (reconciler *stateReconcilerType) recreateStateFromDB() {
	logs := reconciler.logger
	dbStates, err := reconciler.db.SubaccountStates().ListStates()
	if err != nil {
		logs.Error(fmt.Sprintf("while getting states from db: %s", err))
		return
	}

	for _, subaccount := range dbStates {
		//create subaccount state in inMemoryState
		reconciler.inMemoryState[subaccountIDType(subaccount.ID)] = subaccountStateType{
			cisState: CisStateType{isBetaEnabledTrue(subaccount.BetaEnabled), subaccount.UsedForProduction, subaccount.ModifiedAt},
		}
	}

	subaccountsMap, err := reconciler.getDistinctSubaccountsFromInstances()
	if err != nil {
		logs.Warn(fmt.Sprintf("while getting subaccounts from db: %s", err))
		return
	}

	for subaccount := range reconciler.inMemoryState {
		_, ok := subaccountsMap[subaccount]
		if !ok {
			logs.Warn(fmt.Sprintf("subaccount %s found in previous state but not found in current instances, will be deleted", subaccount))
			reconciler.setPendingDelete(subaccount)
		}
	}

	for subaccount := range subaccountsMap {
		_, ok := reconciler.inMemoryState[subaccount]
		if !ok {
			logs.Warn(fmt.Sprintf("subaccount %s not found in previous state but found in current instances", subaccount))
			reconciler.inMemoryState[subaccount] = subaccountStateType{}
		}
	}
	reconciler.setMetrics()
}

func (reconciler *stateReconcilerType) setPendingDelete(subaccount subaccountIDType) {
	reconciler.mutex.Lock()
	defer reconciler.mutex.Unlock()

	state := reconciler.inMemoryState[subaccount]
	state.pendingDelete = true
	reconciler.inMemoryState[subaccount] = state
}

func (reconciler *stateReconcilerType) setMetrics() {
	if reconciler.metrics == nil {
		return
	}
	total := len(reconciler.inMemoryState)
	reconciler.metrics.states.With(prometheus.Labels{"type": "total", "value": "total"}).Set(float64(total))

	betaEnabledCount := 0
	betaDisabledCount := 0
	resourcesStatesCount := 0
	runtimesCount := 0
	pendingDeleteCount := 0

	usedForProduction := make(map[string]int)
	for _, state := range reconciler.inMemoryState {
		if state.cisState != (CisStateType{}) {
			if state.cisState.BetaEnabled {
				betaEnabledCount++
			} else {
				betaDisabledCount++
			}
			usedForProduction[state.cisState.UsedForProduction]++
		}
		if state.resourcesState != nil {
			resourcesStatesCount++
			runtimesCount += len(state.resourcesState)
		}
		if state.pendingDelete {
			pendingDeleteCount++
		}
	}

	totalResourcesStatesCount := betaEnabledCount + betaDisabledCount
	reconciler.metrics.states.With(prometheus.Labels{"type": "betaEnabled", "value": "true"}).Set(float64(betaEnabledCount))
	reconciler.metrics.states.With(prometheus.Labels{"type": "betaEnabled", "value": "false"}).Set(float64(betaDisabledCount))
	reconciler.metrics.states.With(prometheus.Labels{"type": "total", "value": "cis-states"}).Set(float64(totalResourcesStatesCount))
	reconciler.metrics.states.With(prometheus.Labels{"type": "total", "value": "resources-states"}).Set(float64(resourcesStatesCount))
	reconciler.metrics.states.With(prometheus.Labels{"type": "total", "value": "pending-delete"}).Set(float64(pendingDeleteCount))
	reconciler.metrics.states.With(prometheus.Labels{"type": "total", "value": "runtimes"}).Set(float64(runtimesCount))

	others := 0
	for key, value := range usedForProduction {
		if key != usedForProductionLabel && key != notUsedForProductionLabel {
			others += value
		}
	}
	reconciler.metrics.states.With(prometheus.Labels{"type": usedForProductionLabel, "value": usedForProductionLabel}).Set(float64(usedForProduction[usedForProductionLabel]))
	reconciler.metrics.states.With(prometheus.Labels{"type": usedForProductionLabel, "value": notUsedForProductionLabel}).Set(float64(usedForProduction[notUsedForProductionLabel]))
	reconciler.metrics.states.With(prometheus.Labels{"type": usedForProductionLabel, "value": "others"}).Set(float64(others))
}

func (reconciler *stateReconcilerType) periodicAccountsSync() (found int, notfound int, failures int) {
	logs := reconciler.logger

	// get distinct subaccounts from inMemoryState
	subaccountsSet := reconciler.getAllSubaccountIDsFromState()
	logs.Info(fmt.Sprintf("Running CIS accounts synchronization for %d subaccounts", len(subaccountsSet)))

	for subaccountID := range subaccountsSet {
		subaccountDataFromCis, err := reconciler.accountsClient.GetSubaccountData(string(subaccountID))
		if subaccountDataFromCis == (CisStateType{}) && err == nil {
			logs.Warn(fmt.Sprintf("subaccount %s not found in CIS", subaccountID))
			notfound++
			continue
		}
		if err != nil {
			failures++
			logs.Error(fmt.Sprintf("while getting data for subaccount:%s", err))
		} else {
			found++
			reconciler.reconcileCisAccount(subaccountID, subaccountDataFromCis)
		}
	}
	logs.Debug(fmt.Sprintf("Accounts synchronization finished: found: %d, notfound %d, failures: %d", found, notfound, failures))
	return found, notfound, failures
}

func (reconciler *stateReconcilerType) periodicEventsSync(fromActionTime int64) (success bool) {

	logs := reconciler.logger
	eventsClient := reconciler.eventsClient
	subaccountsSet := reconciler.getAllSubaccountIDsFromState()
	success = true

	logs.Info(fmt.Sprintf("Running CIS events synchronization from epoch: %d for %d subaccounts", fromActionTime, len(subaccountsSet)))

	eventsOfInterest, err := eventsClient.getEventsForSubaccounts(fromActionTime, *logs, subaccountsSet)
	if err != nil {
		success = false
		logs.Error(fmt.Sprintf("while getting subaccount events: %s", err))
		// we will retry in the next run
	}

	for _, event := range eventsOfInterest {
		reconciler.reconcileCisEvent(event)
		reconciler.eventWindow.UpdateToTime(event.ActionTime)
	}
	logs.Debug(fmt.Sprintf("Events synchronization finished with succcess==%t, the most recent reconciled event time: %d", success, reconciler.eventWindow.lastToTime))
	return success
}

func (reconciler *stateReconcilerType) getAllSubaccountIDsFromState() subaccountsSetType {
	subaccountsMap := make(subaccountsSetType)
	for subaccount := range reconciler.inMemoryState {
		subaccountsMap[subaccount] = struct{}{}
	}
	return subaccountsMap
}

func (reconciler *stateReconcilerType) runCronJobs(cfg Config, ctx context.Context) {
	s := gocron.NewScheduler(time.UTC)

	logs := reconciler.logger

	_, err := s.Every(cfg.EventsWindowInterval).Do(func() {
		// establish actual time window
		eventsFrom := reconciler.eventWindow.GetNextFromTime()

		ok := reconciler.periodicEventsSync(eventsFrom)
		if ok {
			reconciler.metrics.cisRequests.With(prometheus.Labels{"endpoint": "events", "status": "success"}).Inc()
		} else {
			reconciler.metrics.cisRequests.With(prometheus.Labels{"endpoint": "events", "status": "failure"}).Inc()
		}

		reconciler.eventWindow.UpdateFromTime(eventsFrom)
		logs.Debug(fmt.Sprintf("Running events synchronization from epoch: %d, lastFromTime: %d, lastToTime: %d", eventsFrom, reconciler.eventWindow.lastFromTime, reconciler.eventWindow.lastToTime))
	})
	if err != nil {
		logs.Error(fmt.Sprintf("while scheduling events sync job: %s", err))
	}

	_, err = s.Every(cfg.AccountsSyncInterval).Do(func() {
		found, notfound, failures := reconciler.periodicAccountsSync()

		reconciler.metrics.cisRequests.With(prometheus.Labels{"endpoint": "accounts", "status": "failure"}).Add(float64(failures))
		reconciler.metrics.cisRequests.With(prometheus.Labels{"endpoint": "accounts", "status": "success"}).Add(float64(found + notfound))
		reconciler.metrics.cisRequests.With(prometheus.Labels{"endpoint": "accounts", "status": "notfound"}).Add(float64(notfound))

	})
	if err != nil {
		logs.Error(fmt.Sprintf("while scheduling accounts sync job: %s", err))
	}

	_, err = s.Every(cfg.StorageSyncInterval).Do(func() {
		logs.Info(fmt.Sprintf("Running state storage synchronization"))
		reconciler.storeStateInDb()
	})
	if err != nil {
		logs.Error(fmt.Sprintf("while scheduling storage sync job: %s", err))
	}

	s.StartBlocking() // blocks the current goroutine - we do not reach the end of the runCronJobs function
}

func (reconciler *stateReconcilerType) reconcileCisAccount(subaccountID subaccountIDType, newCisState CisStateType) {
	reconciler.mutex.Lock()
	defer reconciler.mutex.Unlock()

	logs := reconciler.logger

	state, ok := reconciler.inMemoryState[subaccountID]
	if !ok {
		logs.Warn(fmt.Sprintf("subaccount %s not found in state when syncing accounts service - ignoring", subaccountID))
		return
	}
	if newCisState.ModifiedDate >= state.cisState.ModifiedDate {
		state.cisState = newCisState
		reconciler.inMemoryState[subaccountID] = state
		reconciler.enqueueSubaccountIfOutdated(subaccountID, state)
	}
	reconciler.setMetrics()
}

func (reconciler *stateReconcilerType) reconcileCisEvent(event Event) {
	reconciler.mutex.Lock()
	defer reconciler.mutex.Unlock()

	logs := reconciler.logger

	subaccount := subaccountIDType(event.SubaccountID)
	state, ok := reconciler.inMemoryState[subaccount]
	if !ok {
		logs.Warn(fmt.Sprintf("subaccount %s not found in state when syncing events service - ignoring", subaccount))
		return
	}
	if event.ActionTime >= state.cisState.ModifiedDate {
		cisState := CisStateType{
			BetaEnabled:       event.Details.BetaEnabled,
			UsedForProduction: event.Details.UsedForProduction,
			ModifiedDate:      event.ActionTime,
		}
		state.cisState = cisState
		reconciler.inMemoryState[subaccount] = state
		reconciler.enqueueSubaccountIfOutdated(subaccount, state)
	}
	reconciler.setMetrics()
}

func (reconciler *stateReconcilerType) reconcileResourceUpdate(subaccountID subaccountIDType, runtimeID runtimeIDType, runtimeState runtimeStateType) {
	reconciler.mutex.Lock()
	defer reconciler.mutex.Unlock()

	state, ok := reconciler.inMemoryState[subaccountID]
	if !ok {
		// we create new state, there is no state for this subaccount yet (no data from CIS to compare)
		reconciler.logger.Debug(fmt.Sprintf("subaccount %s not found in state - creating state", subaccountID))
		reconciler.inMemoryState[subaccountID] = subaccountStateType{
			resourcesState: subaccountRuntimesType{runtimeID: runtimeState},
		}
	} else {
		if state.resourcesState == nil {
			state.resourcesState = make(subaccountRuntimesType)
		}
		state.resourcesState[runtimeID] = runtimeState
		reconciler.inMemoryState[subaccountID] = state
		reconciler.logger.Debug(fmt.Sprintf("subaccount %s found in state, check if outdated", subaccountID))
		reconciler.enqueueSubaccountIfOutdated(subaccountID, state)
	}
	reconciler.setMetrics()
}

// mark state pending delete and remove runtime from subaccount state
func (reconciler *stateReconcilerType) deleteRuntimeFromState(subaccountID subaccountIDType, runtimeID runtimeIDType) {
	reconciler.mutex.Lock()
	defer reconciler.mutex.Unlock()

	logs := reconciler.logger
	state, ok := reconciler.inMemoryState[subaccountID]
	if !ok {
		logs.Warn(fmt.Sprintf("subaccount %s not found in state", subaccountID))
		return
	}
	_, ok = state.resourcesState[runtimeID]
	if !ok {
		logs.Warn(fmt.Sprintf("runtime %s not found in subaccount %s", runtimeID, subaccountID))
		return
	}
	delete(state.resourcesState, runtimeID)
	state.pendingDelete = len(state.resourcesState) == 0
	reconciler.inMemoryState[subaccountID] = state
	reconciler.setMetrics()
}

// Requests for change are queued in priority queue, the queue is consumed by the updater.
// Since there are multiple sources of changes (events, accounts, resources) and the changes can appear not chronologically, we use priority queue ordered by action time.
// By definition updater (single instance) processes the queue in order of action time and assures that the state is updated in the correct order.

// E.g. consider following scenario (t1<t2<t3) and the approach with goroutines spawning (as opposite to single updater):
// 1. Kyma resource for SA1 has no betaEnabled label set
// 2. At t1 we fetch the state from CIS and set betaEnabled to "false" for SA1, we start goroutine G1 to update the state to "false"
// 3. At t2 user changes the betaEnabled label to "true" for SA1
// 4. At t3 we fetch the events from the event window and we get the event from t2, we start goroutine G2 to update the state to "true"
// There is no guarantee that G1 will finish before G2 and the final state will be "true". With the updater we are sure that the state will be updated in the correct order.

func (reconciler *stateReconcilerType) enqueueSubaccountIfOutdated(subaccountID subaccountIDType, state subaccountStateType) {
	if reconciler.isResourceOutdated(subaccountID, state) {
		reconciler.logger.Debug(fmt.Sprintf("Subaccount %s is outdated, enqueuing, setting betaEnabled %t", subaccountID, state.cisState.BetaEnabled))
		state := reconciler.inMemoryState[subaccountID]
		element := syncqueues.QueueElement{SubaccountID: string(subaccountID), ModifiedAt: state.cisState.ModifiedDate, BetaEnabled: fmt.Sprintf("%t", state.cisState.BetaEnabled), UsedForProduction: state.cisState.UsedForProduction}
		reconciler.syncQueue.Insert(element)
	} else {
		reconciler.logger.Debug(fmt.Sprintf("Subaccount %s is not to be updated", subaccountID))
	}
}

func (reconciler *stateReconcilerType) isResourceOutdated(subaccountID subaccountIDType, state subaccountStateType) bool {
	var outdated bool

	if state.resourcesState != nil && state.cisState.ModifiedDate > 0 {
		runtimes := state.resourcesState
		cisState := state.cisState
		for _, runtimeState := range runtimes {
			outdated = outdated || runtimeState.betaEnabled == ""
			outdated = outdated || runtimeState.usedForProduction == ""
			outdated = outdated || (cisState.BetaEnabled && !isBetaEnabledTrue(runtimeState.betaEnabled))
			outdated = outdated || (!cisState.BetaEnabled && runtimeState.betaEnabled != "false")
			outdated = outdated || cisState.UsedForProduction != runtimeState.usedForProduction
		}
		reconciler.logger.Debug(fmt.Sprintf("Subaccount %s has %d runtimes, outdated: %t", subaccountID, len(runtimes), outdated))
	} else {
		reconciler.logger.Debug(fmt.Sprintf("Subaccount %s has %d resources, cis state modified: %t: not outdated", subaccountID, len(state.resourcesState), state.cisState.ModifiedDate > 0))
	}
	return outdated
}

func (reconciler *stateReconcilerType) storeStateInDb() {
	reconciler.mutex.Lock()
	defer reconciler.mutex.Unlock()

	var upsertCnt, deleteCnt, failureCnt int
	logs := reconciler.logger

	logs.Info(fmt.Sprintf("Syncing state to persistent storage"))

	for subaccount, state := range reconciler.inMemoryState {
		if state.pendingDelete { // no runtimes left, we can delete the state from the storage
			err := reconciler.db.SubaccountStates().DeleteState(string(subaccount))
			if err != nil {
				logs.Error(fmt.Sprintf("while deleting subaccount:%s state from persistent storage: %s", subaccount, err))
				failureCnt++
				continue
			}
			deleteCnt++
			delete(reconciler.inMemoryState, subaccount)
			logs.Debug(fmt.Sprintf("Subaccount %s state deleted from persistent storage", subaccount))
		} else {
			err := reconciler.db.SubaccountStates().UpsertState(internal.SubaccountState{
				ID:                string(subaccount),
				BetaEnabled:       fmt.Sprintf("%t", state.cisState.BetaEnabled),
				UsedForProduction: state.cisState.UsedForProduction,
				ModifiedAt:        state.cisState.ModifiedDate,
			})
			if err != nil {
				failureCnt++
				logs.Error(fmt.Sprintf("while updating subaccount:%s state from persistent storage: %s", subaccount, err))
				continue
			}
			upsertCnt++
		}
	}
	logs.Info(fmt.Sprintf("State synced to persistent storage: %d upserts, %d deletes, %d failures", upsertCnt, deleteCnt, failureCnt))
}

func (reconciler *stateReconcilerType) getDistinctSubaccountsFromInstances() (subaccountsSetType, error) {
	reconciler.mutex.Lock()
	defer reconciler.mutex.Unlock()

	subaccounts, err := reconciler.db.Instances().GetDistinctSubAccounts()

	subaccountsSet := make(subaccountsSetType)
	for _, subaccount := range subaccounts {
		subaccountsSet[subaccountIDType(subaccount)] = struct{}{}
	}
	return subaccountsSet, err
}

func epochInMillis() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}
