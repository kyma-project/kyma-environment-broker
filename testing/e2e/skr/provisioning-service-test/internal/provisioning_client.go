package internal

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	interval              = time.Minute
	provisioningTimeout   = 30 * time.Minute
	deprovisioningTimeout = 90 * time.Minute
	environmentType       = "kyma"
	serviceName           = "kymaruntime"
	accessTokenPath       = "/oauth/token"
	environmentsPath      = "/provisioning/v1/environments"
	bindingsPath          = "/bindings"
)

type ProvisioningConfig struct {
	URL          string
	ClientID     string
	ClientSecret string
	UAA_URL      string
	Kyma         KymaConfig
}

type KymaConfig struct {
	PlanName     string
	PlanID       string
	User         string
	InstanceName string
	Region       string
}

type ProvisioningClient struct {
	cfg         ProvisioningConfig
	logger      *slog.Logger
	cli         *http.Client
	ctx         context.Context
	accessToken string
}

func NewProvisioningClient(cfg ProvisioningConfig, logger *slog.Logger, ctx context.Context, timeoutSeconds time.Duration) *ProvisioningClient {
	cli := &http.Client{Timeout: timeoutSeconds * time.Second}
	return &ProvisioningClient{
		cfg:    cfg,
		logger: logger,
		cli:    cli,
		ctx:    ctx,
	}
}

func (p *ProvisioningClient) CreateEnvironment() (CreatedEnvironmentResponse, error) {
	requestBody := CreateEnvironmentRequest{
		EnvironmentType: environmentType,
		PlanName:        p.cfg.Kyma.PlanName,
		ServiceName:     serviceName,
		User:            p.cfg.Kyma.User,
		Parameters: EnvironmentParameters{
			Name:   p.cfg.Kyma.InstanceName,
			Region: p.cfg.Kyma.Region,
		},
	}

	requestBodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return CreatedEnvironmentResponse{}, fmt.Errorf("failed to marshal request body: %v", err)
	}

	resp, err := p.sendRequest(http.MethodPost, p.environmentsPath(), http.StatusAccepted, bytes.NewBuffer(requestBodyBytes))
	if err != nil {
		return CreatedEnvironmentResponse{}, err
	}

	var environment CreatedEnvironmentResponse
	err = p.unmarshallResponse(resp, &environment)
	if err != nil {
		return CreatedEnvironmentResponse{}, err
	}

	return environment, nil
}

func (p *ProvisioningClient) GetEnvironment(environmentID string) (EnvironmentResponse, error) {
	resp, err := p.sendRequest(http.MethodGet, p.environmentsWithIDPath(environmentID), http.StatusOK, nil)
	if err != nil {
		return EnvironmentResponse{}, err
	}

	var environment EnvironmentResponse
	err = p.unmarshallResponse(resp, &environment)
	if err != nil {
		return EnvironmentResponse{}, err
	}

	return environment, nil
}

func (p *ProvisioningClient) GetEnvironments() (EnvironmentsResponse, error) {
	resp, err := p.sendRequest(http.MethodGet, p.environmentsPath(), http.StatusOK, nil)
	if err != nil {
		return EnvironmentsResponse{}, err
	}

	var environments EnvironmentsResponse
	err = p.unmarshallResponse(resp, &environments)
	if err != nil {
		return EnvironmentsResponse{}, err
	}

	return environments, nil
}

func (p *ProvisioningClient) DeleteEnvironment(environmentID string) (EnvironmentResponse, error) {
	resp, err := p.sendRequest(http.MethodDelete, p.environmentsWithIDPath(environmentID), http.StatusAccepted, nil)
	if err != nil {
		return EnvironmentResponse{}, err
	}

	var environment EnvironmentResponse
	err = p.unmarshallResponse(resp, &environment)
	if err != nil {
		return EnvironmentResponse{}, err
	}

	return environment, nil
}

func (p *ProvisioningClient) CreateBinding(environmentID string) (CreatedBindingResponse, error) {
	requestBody := CreateBindingRequest{
		ServiceInstanceID: environmentID,
		PlanID:            p.cfg.Kyma.PlanID,
	}

	requestBodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return CreatedBindingResponse{}, fmt.Errorf("failed to marshal request body: %v", err)
	}

	resp, err := p.sendRequest(http.MethodPut, p.bindingsPath(environmentID), http.StatusAccepted, bytes.NewBuffer(requestBodyBytes))
	if err != nil {
		return CreatedBindingResponse{}, err
	}

	var binding CreatedBindingResponse
	err = p.unmarshallResponse(resp, &binding)
	if err != nil {
		return CreatedBindingResponse{}, err
	}
	binding.ID = resp.Header.Get("location")

	return binding, nil
}

func (p *ProvisioningClient) GetBinding(environmentID, bindingID string) (GetBindingResponse, error) {
	resp, err := p.sendRequest(http.MethodGet, p.bindingsWithIDPath(environmentID, bindingID), http.StatusOK, nil)
	if err != nil {
		return GetBindingResponse{}, err
	}

	var binding GetBindingResponse
	err = p.unmarshallResponse(resp, &binding)
	if err != nil {
		return GetBindingResponse{}, err
	}

	return binding, nil
}

func (p *ProvisioningClient) DeleteBinding(environmentID, bindingID string) error {
	if _, err := p.sendRequest(http.MethodDelete, p.bindingsWithIDPath(environmentID, bindingID), http.StatusOK, nil); err != nil {
		return err
	}

	return nil
}

func (p *ProvisioningClient) GetAccessToken() error {
	data := url.Values{}
	data.Set("grant_type", "client_credentials")

	clientCredentials := fmt.Sprintf("%s:%s", p.cfg.ClientID, p.cfg.ClientSecret)
	encodedCredentials := base64.StdEncoding.EncodeToString([]byte(clientCredentials))

	req, err := http.NewRequest(http.MethodPost, p.accessTokenPath(), bytes.NewBufferString(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %v", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+encodedCredentials)

	resp, err := p.cli.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var accessToken AccessToken
	err = p.unmarshallResponse(resp, &accessToken)
	if err != nil {
		return err
	}

	p.accessToken = accessToken.Token

	return nil
}

func (p *ProvisioningClient) AwaitEnvironmentCreated(environmentID string) error {
	err := wait.PollUntilContextTimeout(p.ctx, interval, provisioningTimeout, true, func(ctx context.Context) (bool, error) {
		environment, err := p.GetEnvironment(environmentID)
		if err != nil {
			p.logger.Warn(fmt.Sprintf("error getting environment: %v", err))
			return false, nil
		}
		p.logger.Info("Received environment state", "environmentID", environmentID, "state", environment.State)
		if environment.State == OK {
			return true, nil
		}
		return false, nil
	})

	if err != nil {
		return fmt.Errorf("failed to wait for environment creation: %v", err)
	}

	return nil
}

func (p *ProvisioningClient) AwaitEnvironmentDeleted(environmentID string) error {
	err := wait.PollUntilContextTimeout(p.ctx, interval, deprovisioningTimeout, true, func(ctx context.Context) (bool, error) {
		environment, err := p.GetEnvironment(environmentID)
		if err != nil {
			if err.Error() == "unexpected status code 404: Environment instance not found" {
				return true, nil
			}
			p.logger.Warn(fmt.Sprintf("error getting environment: %v", err))
			return false, nil
		}
		p.logger.Info("Received environment state", "environmentID", environmentID, "state", environment.State)
		return false, nil
	})

	if err != nil {
		return fmt.Errorf("failed to wait for environment deletion: %v", err)
	}

	return nil
}

func (p *ProvisioningClient) sendRequest(method string, url string, expectedStatus int, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.accessToken)
	if method == http.MethodPost || method == http.MethodPatch || method == http.MethodPut {
		req.Header.Add("Content-Type", "application/json")
	}

	resp, err := p.cli.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}

	if resp.StatusCode != expectedStatus {
		var errorResponse ErrorResponse
		err = p.unmarshallResponse(resp, &errorResponse)
		if err != nil {
			return nil, fmt.Errorf("unexpected status code %d: %v", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("unexpected status code %d: %v", resp.StatusCode, errorResponse.Error.Message)
	}

	return resp, nil
}

func (p *ProvisioningClient) unmarshallResponse(resp *http.Response, output any) error {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	if len(body) == 0 {
		return fmt.Errorf("body is empty")
	}

	if err := json.Unmarshal(body, output); err != nil {
		return fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return nil
}

func (p *ProvisioningClient) environmentsPath() string {
	return fmt.Sprintf("%s%s", p.cfg.URL, environmentsPath)
}

func (p *ProvisioningClient) environmentsWithIDPath(environmentID string) string {
	return fmt.Sprintf("%s/%s", p.environmentsPath(), environmentID)
}

func (p *ProvisioningClient) bindingsPath(environmentID string) string {
	return fmt.Sprintf("%s%s", p.environmentsWithIDPath(environmentID), bindingsPath)
}

func (p *ProvisioningClient) bindingsWithIDPath(environmentID string, bindingID string) string {
	return fmt.Sprintf("%s/%s", p.bindingsPath(environmentID), bindingID)
}

func (p *ProvisioningClient) accessTokenPath() string {
	return fmt.Sprintf("%s%s", p.cfg.UAA_URL, accessTokenPath)
}
