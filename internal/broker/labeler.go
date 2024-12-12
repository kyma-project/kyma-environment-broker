package broker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/kyma-project/kyma-environment-broker/internal/customresources"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaerrors "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Labeler struct {
	kcpClient client.Client
	log       *slog.Logger
}

func NewLabeler(kcpClient client.Client) *Labeler {
	return &Labeler{
		kcpClient: kcpClient,
		log: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})),
	}
}

func (l *Labeler) UpdateLabels(id, newGlobalAccountId string) error {
	kymaErr := l.updateCrLabel(id, customresources.KymaCr, newGlobalAccountId)
	gardenerClusterErr := l.updateCrLabel(id, customresources.GardenerClusterCr, newGlobalAccountId)
	runtimeErr := l.updateCrLabel(id, customresources.RuntimeCr, newGlobalAccountId)
	err := errors.Join(kymaErr, gardenerClusterErr, runtimeErr)
	return err
}

func (l *Labeler) updateCrLabel(id, crName, newGlobalAccountId string) error {
	l.log.Info(fmt.Sprintf("update label starting for runtime %s for %s cr with new value %s", id, crName, newGlobalAccountId))
	gvk, err := customresources.GvkByName(crName)
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
		l.log.Info(fmt.Sprintf("CRD for %s does not exist, skipping", crName))
		return nil
	}

	err = l.kcpClient.Get(context.Background(), types.NamespacedName{Namespace: KcpNamespace, Name: id}, &k8sObject)
	if err != nil {
		return fmt.Errorf("while getting k8s object of type %s from kcp cluster for instance %s, due to: %s", crName, id, err.Error())
	}

	err = addOrOverrideLabel(&k8sObject, customresources.GlobalAccountIdLabel, newGlobalAccountId)
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

func (l *Labeler) checkCRDExistence(gvk schema.GroupVersionKind) (bool, error) {
	crdName := fmt.Sprintf("%ss.%s", strings.ToLower(gvk.Kind), gvk.Group)
	crd := &apiextensionsv1.CustomResourceDefinition{}
	if err := l.kcpClient.Get(context.Background(), client.ObjectKey{Name: crdName}, crd); err != nil {
		if k8serrors.IsNotFound(err) || metaerrors.IsNoMatchError(err) {
			l.log.Error(fmt.Sprintf("CustomResourceDefinition does not exist %s", err.Error()))
			return false, nil
		} else {
			l.log.Error(fmt.Sprintf("while getting CRD %s: %s", crdName, err.Error()))
			return false, err
		}
	}
	return true, nil
}
