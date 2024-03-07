package subaccountsync

import (
	"fmt"
	"log/slog"
	"reflect"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/tools/cache"
)

func configureInformer(informer *cache.SharedIndexInformer, logs *logrus.Logger, stateReconciler *stateReconcilerType, logger *slog.Logger, metrics *Metrics) {
	_, err := (*informer).AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			metrics.informer.With(prometheus.Labels{"event": "add"}).Inc()
			u := obj.(*unstructured.Unstructured)
			logs.Info(fmt.Sprintf("Kyma resource added: %s.%s", u.GetNamespace(), u.GetName()))
			subaccountID, runtimeID, betaEnabled := getDataFromLabels(obj)
			if subaccountID == "" {
				logs.Error(fmt.Sprintf("added Kyma resource has no subaccount label: %s", u.GetName()))
				return
			}
			stateReconciler.reconcileResourceUpdate(subaccountIDType(subaccountID), runtimeIDType(runtimeID), runtimeStateType{betaEnabled: betaEnabled})
			data, err := stateReconciler.accountsClient.GetSubaccountData(subaccountID)
			if err != nil {
				logs.Warnf("error while getting data for subaccount:%s", err)
			} else {
				stateReconciler.reconcileCisAccount(subaccountIDType(subaccountID), data)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			metrics.informer.With(prometheus.Labels{"event": "update"}).Inc()
			u := newObj.(*unstructured.Unstructured)
			subaccountID, runtimeID, betaEnabled := getDataFromLabels(newObj)
			if subaccountID == "" {
				logs.Error(fmt.Sprintf("updated Kyma resource has no subaccount label: %s", u.GetName()))
				return
			}
			if !reflect.DeepEqual(oldObj.(*unstructured.Unstructured).GetLabels(), u.GetLabels()) {
				stateReconciler.reconcileResourceUpdate(subaccountIDType(subaccountID), runtimeIDType(runtimeID), runtimeStateType{betaEnabled: betaEnabled})
				logs.Info(fmt.Sprintf("Kyma resource labels changed: %s subaccountID: %s", u.GetName(), u.GetLabels()[subaccountIDLabel]))
			}
		},
		DeleteFunc: func(obj interface{}) {
			metrics.informer.With(prometheus.Labels{"event": "delete"}).Inc()
			logs.Info(fmt.Sprintf("Kyma resource deleted: %s", obj.(*unstructured.Unstructured).GetName()))
			subaccountID, runtimeID, _ := getDataFromLabels(obj)
			if subaccountID == "" || runtimeID == "" {
				// deleted kyma resource without subaccount label or runtime label - no need to make fuss, silently ignore
				return
			}
			stateReconciler.deleteRuntimeFromState(subaccountIDType(subaccountID), runtimeIDType(runtimeID))
		},
	})
	fatalOnError(err)
}
