package runtime

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log/slog"
	"net/http"
	"os"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// tenantHeaderName is a header key name for request send by graphQL client
const tenantHeaderName = "tenant"

// Client allows to fetch runtime's config and execute the logic against it
type Client struct {
	httpClient http.Client
	log        *slog.Logger

	instanceID   string
	tenantID     string
	kcpK8sClient client.Client
}

func NewClient(tenantID, instanceID string, clientHttp http.Client, kcpK8sClient client.Client, log *slog.Logger) *Client {
	return &Client{
		tenantID:     tenantID,
		instanceID:   instanceID,
		httpClient:   clientHttp,
		log:          log,
		kcpK8sClient: kcpK8sClient,
	}
}

func (c *Client) kubeconfigForInstanceID() ([]byte, error) {
	secrets := &v1.SecretList{}
	listOpts := secretListOptions(c.instanceID)

	err := c.kcpK8sClient.List(context.Background(), secrets, listOpts...)

	if err != nil {
		return nil, fmt.Errorf("while getting secret from kcp for instanceID=%s : %w", c.instanceID, err)
	}
	if len(secrets.Items) != 1 {
		return nil, fmt.Errorf("found %d secrets for instanceID %s but expected 1", len(secrets.Items), c.instanceID)
	}
	config, ok := secrets.Items[0].Data["config"]
	if !ok {
		return nil, fmt.Errorf("while getting 'config' from secret from instance %s", c.instanceID)
	}
	if len(config) == 0 {
		return nil, fmt.Errorf("empty kubeconfig")
	}
	return config, nil
}

func (c *Client) FetchRuntimeConfig() (*string, error) {
	kubeconfig, err := c.kubeconfigForInstanceID()
	if err != nil {
		return nil, fmt.Errorf("while getting kubeconfig %s: %w", c.instanceID, err)
	}

	if err != nil {
		return nil, fmt.Errorf("while getting runtime config: %w", err)
	}
	if len(kubeconfig) > 0 {
		kcfg := string(kubeconfig)
		return &kcfg, nil
	}
	return nil, errors.New("kubeconfig shouldn't be nil")
}

func (c *Client) writeConfigToFile(config string) (string, error) {
	content := []byte(config)
	runtimeConfigTmpFile, err := ioutil.TempFile("", "runtime.*.yaml")
	if err != nil {
		return "", fmt.Errorf("while creating runtime config temp file: %w", err)
	}

	if _, err := runtimeConfigTmpFile.Write(content); err != nil {
		return "", fmt.Errorf("while writing runtime config temp file: %w", err)
	}
	if err := runtimeConfigTmpFile.Close(); err != nil {
		return "", fmt.Errorf("while closing runtime config temp file: %w", err)
	}

	return runtimeConfigTmpFile.Name(), nil
}

func (c *Client) removeFile(fileName string) {
	err := os.Remove(fileName)
	if err != nil {
		c.log.Error(err.Error())
		os.Exit(1)
	}
}

func (c *Client) warnOnError(err error) {
	if err != nil {
		c.log.Warn(err.Error())
	}
}

func (c *Client) objectKey(runtimeId string) client.ObjectKey {
	return client.ObjectKey{
		Namespace: "kcp-system",
		Name:      fmt.Sprintf("kubeconfig-%s", runtimeId),
	}
}

func secretListOptions(instanceID string) []client.ListOption {
	label := map[string]string{
		"kyma-project.io/instance-id": instanceID,
	}

	return []client.ListOption{
		client.InNamespace("kcp-system"),
		client.MatchingLabels(label),
	}
}
