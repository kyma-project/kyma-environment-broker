package subaccountsync

import (
	"fmt"
	"log/slog"
	"reflect"

	"github.com/kyma-project/kyma-environment-broker/internal/customresources"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/tools/cache"
)

func configureInformer(informer *cache.SharedIndexInformer, handlers cache.ResourceEventHandlerFuncs, counter *prometheus.CounterVec) {
	_, err := (*informer).AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			counter.With(prometheus.Labels{"event": "add"}).Inc()
			if handlers.AddFunc != nil {
				handlers.AddFunc(obj)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			counter.With(prometheus.Labels{"event": "update"}).Inc()
			if handlers.UpdateFunc != nil {
				handlers.UpdateFunc(oldObj, newObj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			counter.With(prometheus.Labels{"event": "delete"}).Inc()
			if handlers.DeleteFunc != nil {
				handlers.DeleteFunc(obj)
			}
		},
	})
	fatalOnError(err)
}

func kymaCREventHandlers(stateReconciler *stateReconcilerType, logger *slog.Logger, alwaysGetSubaccountFromDB bool) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			u, ok := obj.(*unstructured.Unstructured)
			if !ok {
				logger.Error(fmt.Sprintf("added Kyma CR is not an Unstructured: %s", obj))
				return
			}
			subaccountID, runtimeID, betaEnabled, usedForProduction, err := getKymaCRRequiredData(u, logger, stateReconciler, alwaysGetSubaccountFromDB)
			if err != nil {
				return
			}
			stateReconciler.reconcileResourceUpdate(subaccountIDType(subaccountID), runtimeIDType(runtimeID), resourceStateType{kymaState: kymaStateType{betaEnabled: betaEnabled, usedForProduction: usedForProduction}})
			data, err := stateReconciler.accountsClient.GetSubaccountData(subaccountID)
			if err != nil {
				logger.Warn(fmt.Sprintf("while getting data for subaccount:%s", err))
			} else {
				stateReconciler.reconcileCisAccount(subaccountIDType(subaccountID), data)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			u, ok := newObj.(*unstructured.Unstructured)
			if !ok {
				logger.Error(fmt.Sprintf("updated Kyma CR is not an Unstructured: %s", newObj))
				return
			}
			subaccountID, runtimeID, betaEnabled, usedForProduction, err := getKymaCRRequiredData(u, logger, stateReconciler, alwaysGetSubaccountFromDB)
			if err != nil {
				return
			}
			if !reflect.DeepEqual(oldObj.(*unstructured.Unstructured).GetLabels(), u.GetLabels()) {
				stateReconciler.reconcileResourceUpdate(subaccountIDType(subaccountID), runtimeIDType(runtimeID), resourceStateType{kymaState: kymaStateType{betaEnabled: betaEnabled, usedForProduction: usedForProduction}})
			}
		},
		DeleteFunc: func(obj interface{}) {
			u, ok := obj.(*unstructured.Unstructured)
			if !ok {
				logger.Error(fmt.Sprintf("deleted Kyma CR is not an Unstructured: %s", obj))
				return
			}
			logger.Info(fmt.Sprintf("Kyma CR deleted: %s", u.GetName()))
			subaccountID, runtimeID, _ := getDataFromLabels(u)
			if subaccountID == "" || runtimeID == "" {
				return
			}
			stateReconciler.deleteRuntimeFromState(subaccountIDType(subaccountID), runtimeIDType(runtimeID))
		},
	}
}

func runtimeCREventHandlers(stateReconciler *stateReconcilerType, logger *slog.Logger) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			u, ok := obj.(*unstructured.Unstructured)
			if !ok {
				logger.Error(fmt.Sprintf("added Runtime CR is not an Unstructured: %s", obj))
				return
			}
			labels := u.GetLabels()
			subaccountID := labels[subaccountIDLabel]
			runtimeID := labels[runtimeIDLabel]
			if subaccountID == "" || runtimeID == "" {
				logger.Warn(fmt.Sprintf("Runtime CR %s has no subaccount or runtime label, skipping", u.GetName()))
				return
			}
			stateReconciler.reconcileRuntimeResourceUpdate(subaccountIDType(subaccountID), runtimeIDType(runtimeID), labels[customresources.UsedForProductionLabelKey])
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			u, ok := newObj.(*unstructured.Unstructured)
			if !ok {
				logger.Error(fmt.Sprintf("updated Runtime CR is not an Unstructured: %s", newObj))
				return
			}
			if !reflect.DeepEqual(oldObj.(*unstructured.Unstructured).GetLabels(), u.GetLabels()) {
				labels := u.GetLabels()
				subaccountID := labels[subaccountIDLabel]
				runtimeID := labels[runtimeIDLabel]
				if subaccountID == "" || runtimeID == "" {
					logger.Warn(fmt.Sprintf("Runtime CR %s has no subaccount or runtime label, skipping", u.GetName()))
					return
				}
				stateReconciler.reconcileRuntimeResourceUpdate(subaccountIDType(subaccountID), runtimeIDType(runtimeID), labels[customresources.UsedForProductionLabelKey])
			}
		},
	}
}
