package broker

import (
	"context"
	"errors"
	"fmt"

	"github.com/kyma-project/kyma-environment-broker/internal/k8s"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Labeler struct {
	kcpClient client.Client
	log       logrus.FieldLogger
}

func NewLabeler(kcpClient client.Client) *Labeler {
	return &Labeler{
		kcpClient: kcpClient,
		log:       logrus.New(),
	}
}

func (l *Labeler) UpdateLabels(id, newGlobalAccountId string) error {
	kymaErr := l.updateCrLabel(id, k8s.KymaCr, newGlobalAccountId)
	gardenerClusterErr := l.updateCrLabel(id, k8s.GardenerClusterCr, newGlobalAccountId)
	runtimeErr := l.updateCrLabel(id, k8s.RuntimeCr, newGlobalAccountId)
	err := errors.Join(kymaErr, gardenerClusterErr, runtimeErr)
	return err
}

func (l *Labeler) updateCrLabel(id, crName, newGlobalAccountId string) error {
	l.log.Infof("update label starting for runtime %s for %s cr with new value %s", id, crName, newGlobalAccountId)
	gvk, err := k8s.GvkByName(crName)
	if err != nil {
		return fmt.Errorf("while getting gvk for name: %s: %s", crName, err.Error())
	}

	var k8sObject unstructured.Unstructured
	k8sObject.SetGroupVersionKind(gvk)
	err = l.kcpClient.Get(context.Background(), types.NamespacedName{Namespace: KcpNamespace, Name: id}, &k8sObject)
	if err != nil {
		return fmt.Errorf("while getting k8s object of type %s from kcp cluster for instance %s, due to: %s", crName, id, err.Error())
	}

	err = addOrOverrideLabel(&k8sObject, k8s.GlobalAccountIdLabel, newGlobalAccountId)
	if err != nil {
		return fmt.Errorf("while adding or overriding label (new=%s) for k8s object %s %s, because: %s", newGlobalAccountId, id, crName, err.Error())
	}

	err = l.kcpClient.Update(context.Background(), &k8sObject)
	if err != nil {
		return fmt.Errorf("while updating k8s object %s %s, because: %s", id, crName, err.Error())
	}

	return nil
}

func addOrOverrideLabel(k8sObject *unstructured.Unstructured, key, value string) error {
	if k8sObject == nil {
		return fmt.Errorf("object is nil")
	}

	labels := (*k8sObject).GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[key] = value
	(*k8sObject).SetLabels(labels)

	return nil
}
