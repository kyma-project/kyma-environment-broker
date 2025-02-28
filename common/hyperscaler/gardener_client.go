package hyperscaler

import (
	"context"
	"fmt"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

func (sp *GardenerClient) IsUsedByShoot(secretBinding *gardener.SecretBinding) (bool, error) {
	shootlist, err := sp.gardenerClient.Resource(gardener.ShootResource).Namespace(sp.namespace).List(context.Background(), metav1.ListOptions{})
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

func NewGardenerClient(gardenerClient dynamic.Interface, gardenerNamespace string) BindingsClient {
	return &GardenerClient{
		gardenerClient: gardenerClient,
		namespace:      gardenerNamespace,
	}
}

type GardenerClient struct {
	gardenerClient dynamic.Interface
	namespace      string
}

func (p *GardenerClient) GetSecretBinding(labelSelector string) (*gardener.SecretBinding, error) {
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

func (sp *GardenerClient) GetSecretBindings(labelSelector string) ([]unstructured.Unstructured, error) {
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

func (sp *GardenerClient) GetLeastUsed(secretBindings []unstructured.Unstructured) (*gardener.SecretBinding, error) {
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

func (sp *GardenerClient) UpdateSecretBinding(secretBinding *gardener.SecretBinding) (*unstructured.Unstructured, error) {
	return sp.gardenerClient.Resource(gardener.SecretBindingResource).Namespace(sp.namespace).Update(context.Background(), &secretBinding.Unstructured, v1.UpdateOptions{})
}
