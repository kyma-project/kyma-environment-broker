package kymacustomresource

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/syncqueues"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

const (
	namespace               = "kcp-system"
	subaccountIdLabelFormat = "kyma-project.io/subaccount-id=%s"
	betaEnabledLabelKey     = "kyma-project.io/beta-enabled"
	emptyQueueSleepDuration = 30 * time.Second
)

type Updater struct {
	k8sClient *dynamic.DynamicClient
	queue     syncqueues.PriorityQueue
	kymaGVR   schema.GroupVersionResource
	logger    *slog.Logger
}

func NewUpdater(restCfg *rest.Config, queue syncqueues.PriorityQueue, gvr schema.GroupVersionResource) (*Updater, error) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	k8sClient, err := dynamic.NewForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("while creating k8s client: %w", err)
	}

	return &Updater{
		k8sClient: k8sClient,
		queue:     queue,
		kymaGVR:   gvr,
		logger:    logger,
	}, nil
}

func (u *Updater) Run() error {
	for {
		if u.queue.IsEmpty() {
			time.Sleep(emptyQueueSleepDuration)
			continue
		}
		item := u.queue.Extract()
		unstructuredList, err := u.k8sClient.Resource(u.kymaGVR).Namespace(namespace).List(context.Background(), metav1.ListOptions{
			LabelSelector: fmt.Sprintf(subaccountIdLabelFormat, item.SubaccountID),
		})
		if err != nil {
			u.logger.Warn("while listing Kyma CRs", err, "adding item back to the queue")
			u.queue.Insert(item)
			continue
		}
		if len(unstructuredList.Items) == 0 {
			u.logger.Info("no Kyma CRs found for subaccount", item.SubaccountID)
			continue
		}
		retryRequired := false
		for _, kymaCrUnstructured := range unstructuredList.Items {
			if err := u.updateBetaEnabledLabel(kymaCrUnstructured, item.BetaEnabled); err != nil {
				u.logger.Warn("while updating Kyma CR", err, "item will be added back to the queue")
				retryRequired = true
			}
		}
		if retryRequired {
			u.logger.Info("adding item back to the queue")
			u.queue.Insert(item)
		}
	}
}

func (u *Updater) updateBetaEnabledLabel(un unstructured.Unstructured, betaEnabled string) error {
	labels := un.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[betaEnabledLabelKey] = betaEnabled
	un.SetLabels(labels)

	_, err := u.k8sClient.Resource(u.kymaGVR).Namespace(namespace).Update(context.Background(), &un, metav1.UpdateOptions{})
	return err
}
