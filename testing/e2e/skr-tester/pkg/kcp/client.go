package kcp

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type KCPConfig struct {
	AuthType          string
	Host              string
	IssuerURL         string
	GardenerNamespace string
	Username          string
	Password          string
	ClientID          string
	ClientSecret      string
	KubeConfigApiUrl  string
}

type KCPClient struct {
	Config *KCPConfig
}

func NewKCPConfig() *KCPConfig {
	return &KCPConfig{
		AuthType:          getEnvOrThrow("KCP_AUTH_TYPE"),
		Host:              getEnvOrThrow("KCP_KEB_API_URL"),
		IssuerURL:         getEnvOrThrow("KCP_OIDC_ISSUER_URL"),
		ClientID:          getEnvOrThrow("KCP_OIDC_CLIENT_ID"),
		ClientSecret:      getEnvOrThrow("KCP_OIDC_CLIENT_SECRET"),
		GardenerNamespace: getEnvOrThrow("KCP_GARDENER_NAMESPACE"),
		Username:          getEnvOrThrow("KCP_TECH_USER_LOGIN"),
		Password:          getEnvOrThrow("KCP_TECH_USER_PASSWORD"),
		KubeConfigApiUrl:  getEnvOrThrow("KCP_KUBECONFIG_API_URL"),
	}
}

func NewKCPClient() (*KCPClient, error) {
	client := &KCPClient{}
	if clientSecret := os.Getenv("KCP_OIDC_CLIENT_SECRET"); clientSecret != "" {
		client.Config = NewKCPConfig()
		client.WriteConfigToFile()
	}
	args := []string{"login"}
	if clientSecret := os.Getenv("KCP_OIDC_CLIENT_SECRET"); clientSecret != "" {
		args = append(args, "--config", "config.yaml", "-u", client.Config.Username, "-p", client.Config.Password)
	}
	_, err := exec.Command("kcp", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to login: %w", err)
	}
	return client, nil
}

func (c *KCPClient) WriteConfigToFile() {
	file, err := os.Create("config.yaml")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	file.WriteString(fmt.Sprintf("auth-type: \"%s\"\n", c.Config.AuthType))
	file.WriteString(fmt.Sprintf("gardener-namespace: \"%s\"\n", c.Config.GardenerNamespace))
	file.WriteString(fmt.Sprintf("oidc-issuer-url: \"%s\"\n", c.Config.IssuerURL))
	file.WriteString(fmt.Sprintf("oidc-client-id: \"%s\"\n", c.Config.ClientID))
	file.WriteString(fmt.Sprintf("oidc-client-secret: %s\n", c.Config.ClientSecret))
	file.WriteString(fmt.Sprintf("username: %s\n", c.Config.Username))
	file.WriteString(fmt.Sprintf("keb-api-url: \"%s\"\n", c.Config.Host))
	file.WriteString(fmt.Sprintf("kubeconfig-api-url: \"%s\"\n", c.Config.KubeConfigApiUrl))
}

func (c *KCPClient) GetCurrentMachineType(instanceID string) (*string, error) {
	args := []string{"rt", "-i", instanceID, "--runtime-config", "-o", "custom=:{.runtimeConfig.spec.shoot.provider.workers[0].machine.type}"}
	if clientSecret := os.Getenv("KCP_OIDC_CLIENT_SECRET"); clientSecret != "" {
		args = append(args, "--config", "config.yaml")
	}
	output, err := exec.Command("kcp", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get current machine type: %w", err)
	}
	machineType := string(output)
	machineType = strings.TrimSpace(machineType)
	return &machineType, nil
}

func (c *KCPClient) GetCurrentOIDCConfig(instanceID string) (interface{}, error) {
	args := []string{"rt", "-i", instanceID, "--runtime-config", "-o", "custom=:{.runtimeConfig.spec.shoot.kubernetes.kubeAPIServer.oidcConfig}"}
	if clientSecret := os.Getenv("KCP_OIDC_CLIENT_SECRET"); clientSecret != "" {
		args = append(args, "--config", "config.yaml")
	}
	output, err := exec.Command("kcp", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get current OIDC config: %w", err)
	}
	var oidcConfig interface{}
	if err := json.Unmarshal(output, &oidcConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal OIDC config: %w", err)
	}
	return oidcConfig, nil
}

func getEnvOrThrow(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(fmt.Sprintf("Environment variable %s is required", key))
	}
	return value
}
