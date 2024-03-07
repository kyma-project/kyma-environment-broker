package kymacustomresource

import (
	"fmt"

	"github.com/kyma-project/kyma-environment-broker/internal/syncqueues"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

type Updater struct {
	k8sClient *dynamic.DynamicClient
	queue     syncqueues.PriorityQueue
}

func NewUpdater(restCfg *rest.Config, queue syncqueues.PriorityQueue) (*Updater, error) {
	k8sClient, err := dynamic.NewForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("while creating k8s client: %w", err)
	}

	return &Updater{
		k8sClient: k8sClient,
		queue:     queue,
	}, nil
}
