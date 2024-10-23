package broker

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/kyma-project/kyma-environment-broker/internal/k8s"
	"github.com/sirupsen/logrus"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaerrors "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Labels struct {
	kcpClient client.Client
	log       logrus.FieldLogger
}

func NewLabels(kcpClient client.Client) *Labels {
	return &Labels{
		kcpClient: kcpClient,
		log:       logrus.New(),
	}
}

func (l *Labels) UpdateLabels(id, newGlobalAccountId string) error {
	kymaErr := l.updateCrLabel(id, k8s.KymaCr, newGlobalAccountId)
	gardenerClusterErr := l.updateCrLabel(id, k8s.GardenerClusterCr, newGlobalAccountId)
	runtimeErr := l.updateCrLabel(id, k8s.RuntimeCr, newGlobalAccountId)
	err := errors.Join(kymaErr, gardenerClusterErr, runtimeErr)
	return err
}

func (l *Labels) updateCrLabel(id, crName, newGlobalAccountId string) error {
	l.log.Infof("update label starting for runtime %s for %s cr with new value %s", id, crName, newGlobalAccountId)
	gvk, err := k8s.GvkByName(crName)
	if err != nil {
		return fmt.Errorf("while getting gvk for name: %s: %s", crName, err.Error())
	}

	var k8sObject unstructured.Unstructured
	k8sObject.SetGroupVersionKind(gvk)
	crdExists, err := l.checkCRDExistence(gvk)
	if err != nil {
		return fmt.Errorf("while checking existence of CRD for %s: %s", crName, err.Error())
	}
	if !crdExists {
		l.log.Infof("CRD for %s does not exist, skipping", crName)
		return nil
	}

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

func (l *Labels) checkCRDExistence(gvk schema.GroupVersionKind) (bool, error) {
	crdName := fmt.Sprintf("%ss.%s", strings.ToLower(gvk.Kind), gvk.Group)
	crd := &apiextensionsv1.CustomResourceDefinition{}
	if err := l.kcpClient.Get(context.Background(), client.ObjectKey{Name: crdName}, crd); err != nil {
		if k8serrors.IsNotFound(err) || metaerrors.IsNoMatchError(err) {
			l.log.Error("CustomResourceDefinition does not exist")
			return false, nil
		} else {
			l.log.Errorf("while getting CRD %s: %s", crdName, err.Error())
			return false, err
		}
	}
	return true, nil
}
