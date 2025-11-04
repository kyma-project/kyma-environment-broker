package gardener

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const requestTimeout = 10 * time.Second

const (
	TenantNameLabelKey      = "tenantName"
	HyperscalerTypeLabelKey = "hyperscalerType"
	DirtyLabelKey           = "dirty"
	InternalLabelKey        = "internal"
	SharedLabelKey          = "shared"
	EUAccessLabelKey        = "euAccess"
)

type Client struct {
	dynamic.Interface
	namespace string
}

func NewClient(k8sClient dynamic.Interface, namespace string) *Client {
	return &Client{
		Interface: k8sClient,
		namespace: namespace,
	}
}

func (c *Client) Namespace() string {
	return c.namespace
}

func (c *Client) GetSecret(namespace, name string) (*unstructured.Unstructured, error) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	return c.Resource(SecretResource).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (c *Client) GetSecretBinding(name string) (*SecretBinding, error) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	secretBinding, err := c.Resource(SecretBindingResource).Namespace(c.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return NewSecretBinding(*secretBinding), err
}

func (c *Client) GetCredentialsBinding(name string) (*CredentialsBinding, error) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	binding, err := c.Resource(CredentialsBindingResource).Namespace(c.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return NewCredentialsBinding(*binding), err
}

func (c *Client) GetSecretBindings(labelSelector string) (*unstructured.UnstructuredList, error) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	return c.Resource(SecretBindingResource).Namespace(c.namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
}

func (c *Client) GetCredentialsBindings(labelSelector string) (*unstructured.UnstructuredList, error) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	return c.Resource(CredentialsBindingResource).Namespace(c.namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
}

func (c *Client) GetShoots() (*unstructured.UnstructuredList, error) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	return c.Resource(ShootResource).Namespace(c.namespace).List(ctx, metav1.ListOptions{})
}

func (c *Client) GetLeastUsedSecretBindingFromSecretBindings(secretBindings []unstructured.Unstructured) (*SecretBinding, error) {
	usageCount := make(map[string]int, len(secretBindings))
	for _, s := range secretBindings {
		usageCount[s.GetName()] = 0
	}

	shoots, err := c.GetShoots()
	if err != nil {
		return nil, fmt.Errorf("while listing shoots: %w", err)
	}

	if shoots == nil || len(shoots.Items) == 0 {
		return &SecretBinding{Unstructured: secretBindings[0]}, nil
	}

	for _, shoot := range shoots.Items {
		s := Shoot{Unstructured: shoot}
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

	return &SecretBinding{Unstructured: secretBindings[minIndex]}, nil
}

func (c *Client) GetLeastUsedCredentialsBindingFromSecretBindings(credentialsBindings []unstructured.Unstructured) (*CredentialsBinding, error) {
	usageCount := make(map[string]int, len(credentialsBindings))
	for _, s := range credentialsBindings {
		usageCount[s.GetName()] = 0
	}

	shoots, err := c.GetShoots()
	if err != nil {
		return nil, fmt.Errorf("while listing shoots: %w", err)
	}

	if shoots == nil || len(shoots.Items) == 0 {
		return &CredentialsBinding{Unstructured: credentialsBindings[0]}, nil
	}

	for _, shoot := range shoots.Items {
		s := Shoot{Unstructured: shoot}
		count, found := usageCount[s.GetSpecCredentialsBindingName()]
		if !found {
			continue
		}

		usageCount[s.GetSpecCredentialsBindingName()] = count + 1
	}

	min := usageCount[credentialsBindings[0].GetName()]
	minIndex := 0

	for i, cb := range credentialsBindings {
		if usageCount[cb.GetName()] < min {
			min = usageCount[cb.GetName()]
			minIndex = i
		}
	}

	return &CredentialsBinding{Unstructured: credentialsBindings[minIndex]}, nil
}

func (c *Client) UpdateCredentialsBinding(credentialsBinding *CredentialsBinding) (*CredentialsBinding, error) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	u, err := c.Resource(CredentialsBindingResource).Namespace(c.namespace).Update(ctx, &credentialsBinding.Unstructured, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	return NewCredentialsBinding(*u), nil
}

func (c *Client) UpdateSecretBinding(secretBinding *SecretBinding) (*SecretBinding, error) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	u, err := c.Resource(SecretBindingResource).Namespace(c.namespace).Update(ctx, &secretBinding.Unstructured, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	return NewSecretBinding(*u), nil
}

type CredentialsBinding struct {
	unstructured.Unstructured
}

func NewCredentialsBinding(u unstructured.Unstructured) *CredentialsBinding {
	return &CredentialsBinding{u}
}

func (b *CredentialsBinding) GetSecretRefName() string {
	str, _, err := unstructured.NestedString(b.Unstructured.Object, "credentialsRef", "name")
	if err != nil {
		// NOTE this is a safety net, gardener v1beta1 API would need to break the contract for this to panic
		panic(fmt.Sprintf("CredentialsBinding missing field '.secretRef.name': %v", err))
	}
	return str
}

func (b *CredentialsBinding) GetSecretRefNamespace() string {
	str, _, err := unstructured.NestedString(b.Unstructured.Object, "credentialsRef", "namespace")
	if err != nil {
		// NOTE this is a safety net, gardener v1beta1 API would need to break the contract for this to panic
		panic(fmt.Sprintf("CredentialsBinding missing field '.secretRef.namespace': %v", err))
	}
	return str
}

func (b *CredentialsBinding) SetSecretRefName(val string) {
	_ = unstructured.SetNestedField(b.Unstructured.Object, val, "credentialsRef", "name")
}

func (b *CredentialsBinding) SetSecretRefNamespace(val string) {
	_ = unstructured.SetNestedField(b.Unstructured.Object, val, "credentialsRef", "namespace")
}

type SecretBinding struct {
	unstructured.Unstructured
}

func NewSecretBinding(u unstructured.Unstructured) *SecretBinding {
	return &SecretBinding{u}
}

func (b *SecretBinding) GetSecretRefName() string {
	str, _, err := unstructured.NestedString(b.Unstructured.Object, "secretRef", "name")
	if err != nil {
		// NOTE this is a safety net, gardener v1beta1 API would need to break the contract for this to panic
		panic(fmt.Sprintf("SecretBinding missing field '.secretRef.name': %v", err))
	}
	return str
}

func (b *SecretBinding) GetSecretRefNamespace() string {
	str, _, err := unstructured.NestedString(b.Unstructured.Object, "secretRef", "namespace")
	if err != nil {
		// NOTE this is a safety net, gardener v1beta1 API would need to break the contract for this to panic
		panic(fmt.Sprintf("SecretBinding missing field '.secretRef.namespace': %v", err))
	}
	return str
}

func (b *SecretBinding) SetSecretRefName(val string) {
	_ = unstructured.SetNestedField(b.Unstructured.Object, val, "secretRef", "name")
}

func (b *SecretBinding) SetSecretRefNamespace(val string) {
	_ = unstructured.SetNestedField(b.Unstructured.Object, val, "secretRef", "namespace")
}

type Shoot struct {
	unstructured.Unstructured
}

func (b Shoot) GetSpecSecretBindingName() string {
	str, _, err := unstructured.NestedString(b.Unstructured.Object, "spec", "secretBindingName")
	if err != nil {
		// NOTE this is a safety net, gardener v1beta1 API would need to break the contract for this to panic
		panic(fmt.Sprintf("Shoot missing field '.spec.secretBindingName': %v", err))
	}
	return str
}

func (b Shoot) GetSpecCredentialsBindingName() string {
	str, _, err := unstructured.NestedString(b.Unstructured.Object, "spec", "credentialsBindingName")
	if err != nil {
		// NOTE this is a safety net, gardener v1beta1 API would need to break the contract for this to panic
		panic(fmt.Sprintf("Shoot missing field '.spec.credentialsBindingName': %v", err))
	}
	return str
}

func (b Shoot) GetSpecMaintenanceTimeWindowBegin() string {
	str, _, err := unstructured.NestedString(b.Unstructured.Object, "spec", "maintenance", "timeWindow", "begin")
	if err != nil {
		// NOTE this is a safety net, gardener v1beta1 API would need to break the contract for this to panic
		panic(fmt.Sprintf("Shoot missing field '.spec.maintenance.timeWindow.begin': %v", err))
	}
	return str
}

func (b Shoot) GetSpecMaintenanceTimeWindowEnd() string {
	str, _, err := unstructured.NestedString(b.Unstructured.Object, "spec", "maintenance", "timeWindow", "end")
	if err != nil {
		// NOTE this is a safety net, gardener v1beta1 API would need to break the contract for this to panic
		panic(fmt.Sprintf("Shoot missing field '.spec.maintenance.timeWindow.end': %v", err))
	}
	return str
}

func (b Shoot) GetSpecRegion() string {
	str, _, err := unstructured.NestedString(b.Unstructured.Object, "spec", "region")
	if err != nil {
		// NOTE this is a safety net, gardener v1beta1 API would need to break the contract for this to panic
		panic(fmt.Sprintf("Shoot missing field '.spec.region': %v", err))
	}
	return str
}

var (
	SecretResource             = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"}
	SecretBindingResource      = schema.GroupVersionResource{Group: "core.gardener.cloud", Version: "v1beta1", Resource: "secretbindings"}
	ShootResource              = schema.GroupVersionResource{Group: "core.gardener.cloud", Version: "v1beta1", Resource: "shoots"}
	SecretGVK                  = schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Secret"}
	SecretBindingGVK           = schema.GroupVersionKind{Group: "core.gardener.cloud", Version: "v1beta1", Kind: "SecretBinding"}
	ShootGVK                   = schema.GroupVersionKind{Group: "core.gardener.cloud", Version: "v1beta1", Kind: "Shoot"}
	CredentialsBindingResource = schema.GroupVersionResource{Group: "security.gardener.cloud", Version: "v1alpha1", Resource: "credentialsbindings"}
	CredentialsBindingGVK      = schema.GroupVersionKind{Group: "security.gardener.cloud", Version: "v1alpha1", Kind: "CredentialsBinding"}
)

func NewGardenerClusterConfig(kubeconfigPath string) (*restclient.Config, error) {

	rawKubeconfig, err := ioutil.ReadFile(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Gardener Kubeconfig from path %s: %s", kubeconfigPath, err.Error())
	}

	gardenerClusterConfig, err := RESTConfig(rawKubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create RESTConfig for Gardener client: %v", err)
	}

	return gardenerClusterConfig, nil
}

func RESTConfig(kubeconfig []byte) (*restclient.Config, error) {
	return clientcmd.RESTConfigFromKubeConfig(kubeconfig)
}
