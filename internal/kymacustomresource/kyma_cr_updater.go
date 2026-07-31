package kymacustomresource

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/syncqueues"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

const (
	namespace                 = "kcp-system"
	subaccountIdLabelKey      = "kyma-project.io/subaccount-id"
	subaccountIdLabelFormat   = "kyma-project.io/subaccount-id=%s"
	k8sRequestInterval        = 5 * time.Second
	BetaEnabledLabelKey       = "operator.kyma-project.io/beta"
	UsedForProductionLabelKey = "operator.kyma-project.io/used-for-production"
)

type Updater struct {
	k8sClient     dynamic.Interface
	queue         syncqueues.MultiConsumerPriorityQueue
	kymaGVR       schema.GroupVersionResource
	runtimeGVR    schema.GroupVersionResource
	sleepDuration time.Duration
	ctx           context.Context
	logger        *slog.Logger
}

func NewUpdater(k8sClient dynamic.Interface,
	queue syncqueues.MultiConsumerPriorityQueue,
	kymaGVR schema.GroupVersionResource,
	runtimeGVR schema.GroupVersionResource,
	sleepDuration time.Duration,
	ctx context.Context,
	logger *slog.Logger) (*Updater, error) {

	logger.Info(fmt.Sprintf("Creating CR updater for Kyma (%s, %s) and Runtime (%s) CRs", BetaEnabledLabelKey, UsedForProductionLabelKey, UsedForProductionLabelKey))

	return &Updater{
		k8sClient:     k8sClient,
		queue:         queue,
		kymaGVR:       kymaGVR,
		runtimeGVR:    runtimeGVR,
		logger:        logger,
		sleepDuration: sleepDuration,
		ctx:           ctx,
	}, nil
}

func (u *Updater) Run() error {

	for {
		item, ok := u.queue.Extract()
		if !ok {
			time.Sleep(u.sleepDuration)
			continue
		}
		u.logger.Debug(fmt.Sprintf("Item dequeued - subaccountID: %s, betaEnabled %s", item.SubaccountID, item.BetaEnabled))

		ctxWithTimeout, cancel := context.WithTimeout(u.ctx, k8sRequestInterval)

		kymaRetry := u.patchKymaCRs(item.SubaccountID, item.BetaEnabled, item.UsedForProduction, ctxWithTimeout)
		runtimeRetry := u.patchRuntimeCRs(item.SubaccountID, item.UsedForProduction, ctxWithTimeout)
		retryRequired := kymaRetry || runtimeRetry

		cancel()
		if retryRequired {
			u.logger.Debug(fmt.Sprintf("Requeue item for subaccount: %s", item.SubaccountID))
			u.queue.Insert(item)
		}
	}
}

func (u *Updater) patchKymaCRs(subaccountID, betaEnabled, usedForProduction string, ctx context.Context) bool {
	list, err := u.k8sClient.Resource(u.kymaGVR).Namespace(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf(subaccountIdLabelFormat, subaccountID),
	})
	if err != nil {
		u.logger.Warn("while listing Kyma CRs: " + err.Error() + " requeue item")
		return true
	}
	if len(list.Items) == 0 {
		u.logger.Info("no Kyma CRs found for subaccount " + subaccountID)
		return false
	}
	u.logger.Debug(fmt.Sprintf("found %d Kyma CRs for subaccount", len(list.Items)))
	retryRequired := false
	for _, un := range list.Items {
		labels := un.GetLabels()
		if labels == nil {
			labels = make(map[string]string)
		}
		labels[BetaEnabledLabelKey] = betaEnabled
		labels[UsedForProductionLabelKey] = usedForProduction
		un.SetLabels(labels)
		if _, err := u.k8sClient.Resource(u.kymaGVR).Namespace(namespace).Update(ctx, &un, metav1.UpdateOptions{}); err != nil {
			u.logger.Warn("while updating Kyma CR: " + err.Error() + " item will be added back to the queue")
			retryRequired = true
		}
	}
	return retryRequired
}

func (u *Updater) patchRuntimeCRs(subaccountID, usedForProduction string, ctx context.Context) bool {
	list, err := u.k8sClient.Resource(u.runtimeGVR).Namespace(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf(subaccountIdLabelFormat, subaccountID),
	})
	if err != nil {
		u.logger.Warn("while listing Runtime CRs: " + err.Error() + " requeue item")
		return true
	}
	if len(list.Items) == 0 {
		u.logger.Info("no Runtime CRs found for subaccount " + subaccountID)
		return false
	}
	u.logger.Debug(fmt.Sprintf("found %d Runtime CRs for subaccount", len(list.Items)))
	retryRequired := false
	for _, un := range list.Items {
		labels := un.GetLabels()
		if labels == nil {
			labels = make(map[string]string)
		}
		labels[UsedForProductionLabelKey] = usedForProduction
		un.SetLabels(labels)
		if _, err := u.k8sClient.Resource(u.runtimeGVR).Namespace(namespace).Update(ctx, &un, metav1.UpdateOptions{}); err != nil {
			u.logger.Warn("while updating Runtime CR: " + err.Error() + " item will be added back to the queue")
			retryRequired = true
		}
	}
	return retryRequired
}
