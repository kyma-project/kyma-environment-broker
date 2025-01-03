package kcp

import (
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
	OAuthClientID     string
	OAuthSecret       string
	OAuthIssuer       string
	MotherShipApiUrl  string
	KubeConfigApiUrl  string
}

func getEnvOrThrow(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(fmt.Sprintf("Environment variable %s is required", key))
	}
	return value
}

func NewKCPConfig() *KCPConfig {
	config := &KCPConfig{
		AuthType:          getEnvOrThrow("KCP_AUTH_TYPE"),
		Host:              getEnvOrThrow("KCP_KEB_API_URL"),
		IssuerURL:         getEnvOrThrow("KCP_OIDC_ISSUER_URL"),
		GardenerNamespace: getEnvOrThrow("KCP_GARDENER_NAMESPACE"),
		Username:          getEnvOrThrow("KCP_TECH_USER_LOGIN"),
		Password:          getEnvOrThrow("KCP_TECH_USER_PASSWORD"),
		ClientID:          getEnvOrThrow("KCP_OIDC_CLIENT_ID"),
		MotherShipApiUrl:  getEnvOrThrow("KCP_MOTHERSHIP_API_URL"),
		KubeConfigApiUrl:  getEnvOrThrow("KCP_KUBECONFIG_API_URL"),
	}

	if clientSecret := os.Getenv("KCP_OIDC_CLIENT_SECRET"); clientSecret != "" {
		config.ClientSecret = getEnvOrThrow("KCP_OIDC_CLIENT_SECRET")
	} else {
		config.OAuthClientID = getEnvOrThrow("KCP_OAUTH2_CLIENT_ID")
		config.OAuthSecret = getEnvOrThrow("KCP_OAUTH2_CLIENT_SECRET")
		config.OAuthIssuer = getEnvOrThrow("KCP_OAUTH2_ISSUER_URL")
	}

	return config
}

type KCPClient struct {
	Config *KCPConfig
}

func NewKCPClient() *KCPClient {
	client := &KCPClient{}
	if clientSecret := os.Getenv("KCP_OIDC_CLIENT_SECRET"); clientSecret != "" {
		client.Config = NewKCPConfig()
		client.WriteConfigToFile()
	}
	return client
}

func (c *KCPClient) WriteConfigToFile() {
	file, err := os.Create("config.yaml")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	file.WriteString(fmt.Sprintf("auth-type: \"%s\"\n", c.Config.AuthType))
	file.WriteString(fmt.Sprintf("gardener-namespace: \"%s\"\n", c.Config.GardenerNamespace))
	if os.Getenv("KCP_OIDC_CLIENT_SECRET") != "" {
		file.WriteString(fmt.Sprintf("oidc-client-id: \"%s\"\n", c.Config.ClientID))
		file.WriteString(fmt.Sprintf("oidc-client-secret: %s\n", c.Config.ClientSecret))
		file.WriteString(fmt.Sprintf("username: %s\n", c.Config.Username))
	} else {
		file.WriteString(fmt.Sprintf("oauth2-client-id: \"%s\"\n", c.Config.OAuthClientID))
		file.WriteString(fmt.Sprintf("oauth2-client-secret: \"%s\"\n", c.Config.OAuthSecret))
		file.WriteString(fmt.Sprintf("oauth2-issuer-url: \"%s\"\n", c.Config.OAuthIssuer))
	}

	file.WriteString(fmt.Sprintf("keb-api-url: \"%s\"\n", c.Config.Host))
	file.WriteString(fmt.Sprintf("oidc-issuer-url: \"%s\"\n", c.Config.IssuerURL))
	file.WriteString(fmt.Sprintf("mothership-api-url: \"%s\"\n", c.Config.MotherShipApiUrl))
	file.WriteString(fmt.Sprintf("kubeconfig-api-url: \"%s\"\n", c.Config.KubeConfigApiUrl))
}

func (c *KCPClient) Login() error {
	args := []string{"login"}
	if clientSecret := os.Getenv("KCP_OIDC_CLIENT_SECRET"); clientSecret != "" {
		args = append(args, "-u", os.Getenv("KCP_TECH_USER_LOGIN"), "-p", os.Getenv("KCP_TECH_USER_PASSWORD"))
	}
	_, err := exec.Command("kcp", args...).Output()
	if err != nil {
		return fmt.Errorf("failed to login: %w", err)
	}
	return nil
}

func (c *KCPClient) GetCurrentMachineType(instanceID string) (*string, error) {
	if err := c.Login(); err != nil {
		return nil, fmt.Errorf("failed to login: %w", err)
	}
	output, err := exec.Command("kcp", "rt", "-i", instanceID, "--runtime-config", "-o", "custom=:{.runtimeConfig.spec.shoot.provider.workers[0].machine.type}").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get current machine type: %w", err)
	}
	machineType := string(output)
	machineType = strings.TrimSpace(machineType)
	return &machineType, nil
}
