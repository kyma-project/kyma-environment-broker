package hyperscaler

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

type AccountPool interface {
	CredentialsSecretBinding(hyperscalerType Type, tenantName string, euAccess bool, shared bool) (*gardener.SecretBinding, error)
	MarkSecretBindingAsDirty(hyperscalerType Type, tenantName string, euAccess bool) error
	IsSecretBindingUsed(hyperscalerType Type, tenantName string, euAccess bool) (bool, error)
	IsSecretBindingDirty(hyperscalerType Type, tenantName string, euAccess bool) (bool, error)
	IsSecretBindingInternal(hyperscalerType Type, tenantName string, euAccess bool) (bool, error)
	SharedCredentialsSecretBinding(hyperscalerType Type, euAccess bool) (*gardener.SecretBinding, error)
}

func NewAccountPool(gardenerClient dynamic.Interface, gardenerNamespace string) AccountPool {
	return &secretBindingsAccountPool{
		gardenerClient: gardenerClient,
		namespace:      gardenerNamespace,
	}
}

type secretBindingsAccountPool struct {
	gardenerClient dynamic.Interface
	namespace      string
	mux            sync.Mutex
}

func (p *secretBindingsAccountPool) IsSecretBindingInternal(hyperscalerType Type, tenantName string, euAccess bool) (bool, error) {
	hypLabels := fmt.Sprintf("hyperscalerType=%s", hyperscalerType.GetKey())
	hypLabels = addEuAccessSelector(hypLabels, euAccess)

	labelSelector := fmt.Sprintf("%s, internal=true, tenantName=%s", hypLabels, tenantName)
	secretBinding, err := p.getSecretBinding(labelSelector)
	if err != nil {
		return false, fmt.Errorf("looking for a secret binding used by the tenant %s and hyperscaler %s: %w", tenantName, hyperscalerType.GetKey(), err)
	}

	if secretBinding != nil {
		return true, nil
	}
	return false, nil
}

func (p *secretBindingsAccountPool) IsSecretBindingDirty(hyperscalerType Type, tenantName string, euAccess bool) (bool, error) {
	hypLabels := fmt.Sprintf("hyperscalerType=%s, shared!=true", hyperscalerType.GetKey())
	hypLabels = addEuAccessSelector(hypLabels, euAccess)

	labelSelector := fmt.Sprintf("%s, dirty=true, tenantName=%s", hypLabels, tenantName)
	secretBinding, err := p.getSecretBinding(labelSelector)
	if err != nil {
		return false, fmt.Errorf("looking for a secret binding used by the tenant %s and hyperscaler %s: %w", tenantName, hyperscalerType.GetKey(), err)
	}

	if secretBinding != nil {
		return true, nil
	}
	return false, nil
}

func (p *secretBindingsAccountPool) MarkSecretBindingAsDirty(hyperscalerType Type, tenantName string, euAccess bool) error {
	p.mux.Lock()
	defer p.mux.Unlock()

	hypLabels := fmt.Sprintf("hyperscalerType=%s, shared!=true", hyperscalerType.GetKey())
	hypLabels = addEuAccessSelector(hypLabels, euAccess)

	labelSelector := fmt.Sprintf("%s, tenantName=%s", hypLabels, tenantName)
	secretBinding, err := p.getSecretBinding(labelSelector)
	if err != nil {
		return fmt.Errorf("marking secret binding as dirty: failed to find secret binding used by the tenant %s and"+" hyperscaler %s: %w", tenantName, hyperscalerType.GetKey(), err)
	}
	// if there is no matching secret - do nothing
	if secretBinding == nil {
		return nil
	}

	labels := secretBinding.GetLabels()
	labels["dirty"] = "true"
	secretBinding.SetLabels(labels)

	_, err = p.gardenerClient.Resource(gardener.SecretBindingResource).Namespace(p.namespace).Update(context.Background(), &secretBinding.Unstructured, v1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("marking secret binding as dirty: failed to update secret binding for tenant: %s and hyperscaler: %s: %w", tenantName, hyperscalerType.GetKey(), err)

	}
	return nil
}

func (p *secretBindingsAccountPool) IsSecretBindingUsed(hyperscalerType Type, tenantName string, euAccess bool) (bool, error) {

	hypLabels := fmt.Sprintf("hyperscalerType=%s", hyperscalerType.GetKey())
	hypLabels = addEuAccessSelector(hypLabels, euAccess)

	labelSelector := fmt.Sprintf("%s, tenantName=%s", hypLabels, tenantName)
	secretBinding, err := p.getSecretBinding(labelSelector)
	if err != nil {
		return false, fmt.Errorf("counting subscription usage: could not find secret binding used by the tenant %s and hyperscaler %s: %w", tenantName, hyperscalerType.GetKey(), err)
	}
	// if there is no matching secret, that's ok (maybe it was not used, for example the step was not run)
	if secretBinding == nil {
		return false, nil
	}

	shootlist, err := p.gardenerClient.Resource(gardener.ShootResource).Namespace(p.namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return false, fmt.Errorf("listing Gardener shoots: %w", err)
	}

	for _, shoot := range shootlist.Items {
		sh := gardener.Shoot{Unstructured: shoot}
		if sh.GetSpecSecretBindingName() == secretBinding.GetName() {
			return true, nil
		}
	}

	return false, nil
}

func (sp *secretBindingsAccountPool) SharedCredentialsSecretBinding(hyperscalerType Type, euAccess bool) (*gardener.SecretBinding, error) {
	// selector
	hypLabels := fmt.Sprintf("hyperscalerType=%s, shared=true", hyperscalerType.GetKey())

	// label selector moditifactions
	// - not present in SharedCredentialsSecretBinding

	// get binding
	secretBindings, err := sp.getSecretBindings(hypLabels)
	if err != nil {
		return nil, fmt.Errorf("getting secret binding: %w", err)
	}

	return sp.getLeastUsed(secretBindings)
}

func (p *secretBindingsAccountPool) CredentialsSecretBinding(hyperscalerType Type, tenantName string, euAccess bool, shared bool) (*gardener.SecretBinding, error) {
	// selector
	hypSelector := fmt.Sprintf("hyperscalerType=%s, shared!=true", hyperscalerType.GetKey())
	hypSelector = addEuAccessSelector(hypSelector, euAccess)

	// label selector moditifactions
	labelSelector := fmt.Sprintf("%s, tenantName=%s, !dirty", hypSelector, tenantName)
	labelSelector = strings.ReplaceAll(labelSelector, ", shared!=true", "")
	
	// get binding
	secretBinding, err := p.getSecretBinding(labelSelector)
	if err != nil {
		return nil, fmt.Errorf("getting secret binding: %w", err)
	}
	if secretBinding != nil {
		return secretBinding, nil
	}

	// lock so that only one thread can fetch an unassigned secret binding and assign it
	// (update secret binding with tenantName)
	p.mux.Lock()
	defer p.mux.Unlock()

	unassignedSelector := fmt.Sprintf("%s, !tenantName, !dirty", hypSelector)

	secretBinding, err = p.getSecretBinding(unassignedSelector)
	if err != nil {
		return nil, fmt.Errorf("getting secret binding: %w", err)
	}
	if secretBinding == nil {
		return nil, fmt.Errorf("failed to find unassigned secret binding for hyperscalerType: %s", hyperscalerType.GetKey())
	}

	// assign
	labels := secretBinding.GetLabels()
	labels["tenantName"] = tenantName
	secretBinding.SetLabels(labels)
	updatedSecretBinding, err := p.gardenerClient.Resource(gardener.SecretBindingResource).Namespace(p.namespace).Update(context.Background(), &secretBinding.Unstructured, v1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating secret binding with tenantName: %s: %w", tenantName, err)
	}

	return &gardener.SecretBinding{Unstructured: *updatedSecretBinding}, nil
}

func (p *secretBindingsAccountPool) getSecretBinding(labelSelector string) (*gardener.SecretBinding, error) {
	secretBindings, err := p.gardenerClient.Resource(gardener.SecretBindingResource).Namespace(p.namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("listing secret bindings for LabelSelector: %s: %w", labelSelector, err)
	}

	if secretBindings != nil && len(secretBindings.Items) > 0 {
		return &gardener.SecretBinding{Unstructured: secretBindings.Items[0]}, nil
	}
	return nil, nil
}

func (sp *secretBindingsAccountPool) getSecretBindings(labelSelector string) ([]unstructured.Unstructured, error) {
	secretBindings, err := sp.gardenerClient.Resource(gardener.SecretBindingResource).Namespace(sp.namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing secret bindings for %s label selector: %w", labelSelector, err)
	}

	if secretBindings == nil || len(secretBindings.Items) == 0 {
		return nil, fmt.Errorf("secretBindingsAccountPool error: no shared secret binding found for %s label selector, "+
			"namespace %s", labelSelector, sp.namespace)
	}

	return secretBindings.Items, nil
}

func (sp *secretBindingsAccountPool) getLeastUsed(secretBindings []unstructured.Unstructured) (*gardener.SecretBinding, error) {
	usageCount := make(map[string]int, len(secretBindings))
	for _, s := range secretBindings {
		usageCount[s.GetName()] = 0
	}

	shoots, err := sp.gardenerClient.Resource(gardener.ShootResource).Namespace(sp.namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error while listing Shoots: %w", err)
	}

	if shoots == nil || len(shoots.Items) == 0 {
		return &gardener.SecretBinding{Unstructured: secretBindings[0]}, nil
	}

	for _, shoot := range shoots.Items {
		s := gardener.Shoot{Unstructured: shoot}
		count, found := usageCount[s.GetSpecSecretBindingName()]
		if !found {
			continue
		}

		usageCount[s.GetSpecSecretBindingName()] = count + 1
	}

	min := usageCount[secretBindings[0].GetName()]
	minIndex := 0

	for i, sb := range secretBindings {
		if usageCount[sb.GetName()] < min {
			min = usageCount[sb.GetName()]
			minIndex = i
		}
	}

	return &gardener.SecretBinding{Unstructured: secretBindings[minIndex]}, nil
}

func addEuAccessSelector(selector string, euAccess bool) string {
	if euAccess {
		return selector + ", euAccess=true"
	} else {
		return selector + ", !euAccess"
	}
}
