package subaccountsync

import (
	"fmt"
	"log/slog"
	"reflect"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/tools/cache"
)

func configureInformer(informer *cache.SharedIndexInformer, stateReconciler *stateReconcilerType, logger *slog.Logger, metrics *Metrics, alwaysUseDB bool) {
	_, err := (*informer).AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			metrics.informer.With(prometheus.Labels{"event": "add"}).Inc()
			u, ok := obj.(*unstructured.Unstructured)
			if !ok {
				logger.Error(fmt.Sprintf("added Kyma CR is not an Unstructured: %s", obj))
				return
			}
			subaccountID, runtimeID, betaEnabled, usedForProduction, err := getRequiredData(u, logger, stateReconciler, alwaysUseDB)
			if err != nil {
				return
			}

			stateReconciler.reconcileResourceUpdate(subaccountIDType(subaccountID), runtimeIDType(runtimeID), runtimeStateType{betaEnabled: betaEnabled, usedForProduction: usedForProduction})
			data, err := stateReconciler.accountsClient.GetSubaccountData(subaccountID)
			if err != nil {
				logger.Warn(fmt.Sprintf("while getting data for subaccount:%s", err))
			} else {
				stateReconciler.reconcileCisAccount(subaccountIDType(subaccountID), data)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			metrics.informer.With(prometheus.Labels{"event": "update"}).Inc()
			u, ok := newObj.(*unstructured.Unstructured)
			if !ok {
				logger.Error(fmt.Sprintf("updated Kyma CR is not an Unstructured: %s", newObj))
				return
			}
			subaccountID, runtimeID, betaEnabled, usedForProduction, err := getRequiredData(u, logger, stateReconciler, alwaysUseDB)
			if err != nil {
				return
			}
			if !reflect.DeepEqual(oldObj.(*unstructured.Unstructured).GetLabels(), u.GetLabels()) {
				stateReconciler.reconcileResourceUpdate(subaccountIDType(subaccountID), runtimeIDType(runtimeID), runtimeStateType{betaEnabled: betaEnabled, usedForProduction: usedForProduction})
			}
		},
		DeleteFunc: func(obj interface{}) {
			metrics.informer.With(prometheus.Labels{"event": "delete"}).Inc()
			u, ok := obj.(*unstructured.Unstructured)
			if !ok {
				logger.Error(fmt.Sprintf("deleted Kyma CR is not an Unstructured: %s", obj))
				return
			}
			logger.Info(fmt.Sprintf("Kyma CR deleted: %s", u.GetName()))
			subaccountID, runtimeID, _ := getDataFromLabels(u)
			if subaccountID == "" || runtimeID == "" {
				// deleted kyma resource without subaccount label or runtime label - no need to make fuss, silently ignore
				return
			}
			stateReconciler.deleteRuntimeFromState(subaccountIDType(subaccountID), runtimeIDType(runtimeID))
		},
	})
	fatalOnError(err)
}
