package hyperscaler

import (
	"fmt"
	"strings"
	"sync"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type AccountPool interface {
	CredentialsSecretBinding(hyperscalerType Type, euAccess bool, shared bool, tenantName string) (*gardener.SecretBinding, error)
	MarkSecretBindingAsDirty(hyperscalerType Type, tenantName string, euAccess bool) error
	IsSecretBindingUsed(hyperscalerType Type, tenantName string, euAccess bool) (bool, error)
	IsSecretBindingDirty(hyperscalerType Type, tenantName string, euAccess bool) (bool, error)
	IsSecretBindingInternal(hyperscalerType Type, tenantName string, euAccess bool) (bool, error)
	SharedCredentialsSecretBinding(hyperscalerType Type, euAccess bool) (*gardener.SecretBinding, error)
}

type BindingsClient interface {
	GetSecretBinding(labelSelector string) (*gardener.SecretBinding, error)
	GetSecretBindings(labelSelector string) ([]unstructured.Unstructured, error)
	GetLeastUsed(secretBindings []unstructured.Unstructured) (*gardener.SecretBinding, error)
	UpdateSecretBinding(secretBinding *gardener.SecretBinding) (*unstructured.Unstructured, error)
	IsUsedByShoot(secretBinding *gardener.SecretBinding) (bool, error)
}

func NewAccountPool(bindingsClient BindingsClient) AccountPool {
	return &secretBindingsAccountPool{
		bindingsClient: bindingsClient,
	}
}

type secretBindingsAccountPool struct {
	namespace      string
	mux            sync.Mutex
	bindingsClient BindingsClient
}

func getLabelsSelector(hyperscalerType Type, shared bool, euAccess bool) string {
	selector := fmt.Sprintf("hyperscalerType=%s", hyperscalerType.GetKey())
	if !shared {
		selector = fmt.Sprintf("%s, shared!=true", selector)
	} else {
		selector = fmt.Sprintf("%s, shared=true", selector)
	}
	selector = addEuAccessSelector(selector, euAccess)

	// old SharedCredentialsSecretBinding method ignored euAccess param
	if shared {
		selector = strings.ReplaceAll(selector, ", euAccess=true", "")
		selector = strings.ReplaceAll(selector, ", !euAccess", "")
	}
	return selector
}

func (p *secretBindingsAccountPool) IsSecretBindingInternal(hyperscalerType Type, tenantName string, euAccess bool) (bool, error) {

	hyperscalerLabels := getLabelsSelector(hyperscalerType, false, euAccess)

	labelSelector := fmt.Sprintf("%s, internal=true, tenantName=%s", hyperscalerLabels, tenantName)
	labelSelector = strings.ReplaceAll(labelSelector, ", shared!=true", "")
	secretBinding, err := p.bindingsClient.GetSecretBinding(labelSelector)
	if err != nil {
		return false, fmt.Errorf("looking for a secret binding used by the tenant %s and hyperscaler %s: %w", tenantName, hyperscalerType.GetKey(), err)
	}

	if secretBinding != nil {
		return true, nil
	}
	return false, nil
}

func (p *secretBindingsAccountPool) IsSecretBindingDirty(hyperscalerType Type, tenantName string, euAccess bool) (bool, error) {
	hyperscalerLabels := getLabelsSelector(hyperscalerType, false, euAccess)

	labelSelector := fmt.Sprintf("%s, dirty=true, tenantName=%s", hyperscalerLabels, tenantName)
	secretBinding, err := p.bindingsClient.GetSecretBinding(labelSelector)
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

	hyperscalerLabels := getLabelsSelector(hyperscalerType, false, euAccess)

	labelSelector := fmt.Sprintf("%s, tenantName=%s", hyperscalerLabels, tenantName)

	secretBinding, err := p.bindingsClient.GetSecretBinding(labelSelector)
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

	_, err = p.bindingsClient.UpdateSecretBinding(secretBinding)
	if err != nil {
		return fmt.Errorf("marking secret binding as dirty: failed to update secret binding for tenant: %s and hyperscaler: %s: %w", tenantName, hyperscalerType.GetKey(), err)

	}
	return nil
}

func (p *secretBindingsAccountPool) IsSecretBindingUsed(hyperscalerType Type, tenantName string, euAccess bool) (bool, error) {

	hyperscalerLabels := getLabelsSelector(hyperscalerType, false, euAccess)

	labelSelector := fmt.Sprintf("%s, tenantName=%s", hyperscalerLabels, tenantName)
	labelSelector = strings.ReplaceAll(labelSelector, ", shared!=true", "")
	secretBinding, err := p.bindingsClient.GetSecretBinding(labelSelector)
	if err != nil {
		return false, fmt.Errorf("counting subscription usage: could not find secret binding used by the tenant %s and hyperscaler %s: %w", tenantName, hyperscalerType.GetKey(), err)
	}
	// if there is no matching secret, that's ok (maybe it was not used, for example the step was not run)
	if secretBinding == nil {
		return false, nil
	}

	return p.bindingsClient.IsUsedByShoot(secretBinding)
}

func (sp *secretBindingsAccountPool) SharedCredentialsSecretBinding(hyperscalerType Type, euAccess bool) (*gardener.SecretBinding, error) {
	// selector
	shared := true
	selector := getLabelsSelector(hyperscalerType, shared, euAccess)

	// get binding
	secretBindings, err := sp.bindingsClient.GetSecretBindings(selector)
	if err != nil {
		return nil, fmt.Errorf("getting secret binding: %w", err)
	}

	return sp.bindingsClient.GetLeastUsed(secretBindings)
}

func (p *secretBindingsAccountPool) CredentialsSecretBinding(hyperscalerType Type, euAccess bool, shared bool, tenantName string) (*gardener.SecretBinding, error) {
	// selector
	selector := getLabelsSelector(hyperscalerType, shared, euAccess)

	// label selector modifications
	labelSelector := fmt.Sprintf("%s, tenantName=%s, !dirty", selector, tenantName)
	labelSelector = strings.ReplaceAll(labelSelector, ", shared!=true", "")

	// get binding
	secretBinding, err := p.bindingsClient.GetSecretBinding(labelSelector)
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

	unassignedSelector := fmt.Sprintf("%s, !tenantName, !dirty", selector)

	secretBinding, err = p.bindingsClient.GetSecretBinding(unassignedSelector)
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

	p.bindingsClient.UpdateSecretBinding(secretBinding)
	if err != nil {
		return nil, fmt.Errorf("updating secret binding with tenantName: %s: %w", tenantName, err)
	}

	return secretBinding, nil
}

func addEuAccessSelector(selector string, euAccess bool) string {
	if euAccess {
		return selector + ", euAccess=true"
	} else {
		return selector + ", !euAccess"
	}
}
