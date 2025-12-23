package broker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log/slog"
	"net/http"
	"time"

	"golang.org/x/oauth2/clientcredentials"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	infoRuntimePath      = "%s/info/runtimes"
	upgradeInstancePath  = "%s/upgrade/kyma"
	getOrchestrationPath = "%s/orchestrations/%s"
)

type UpgradeClient struct {
	log    *slog.Logger
	URL    string
	client *http.Client
}

func NewUpgradeClient(ctx context.Context, oAuthConfig BrokerOAuthConfig, config Config, log *slog.Logger) *UpgradeClient {
	cfg := clientcredentials.Config{
		ClientID:     oAuthConfig.ClientID,
		ClientSecret: oAuthConfig.ClientSecret,
		TokenURL:     config.TokenURL,
		Scopes:       []string{oAuthConfig.Scope},
	}
	httpClientOAuth := cfg.Client(ctx)
	httpClientOAuth.Timeout = 30 * time.Second

	return &UpgradeClient{
		log:    log.With("client", "upgrade_broker_client"),
		URL:    config.URL,
		client: httpClientOAuth,
	}
}

func (c *UpgradeClient) UpgradeRuntime(runtimeID string) (string, error) {
	payload := UpgradeRuntimeRequest{
		Targets: Target{
			Include: []RuntimeTarget{{RuntimeID: runtimeID}},
		},
	}
	requestBody, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("while marshaling payload request: %w", err)
	}

	response, err := c.executeRequest(http.MethodPost, fmt.Sprintf(upgradeInstancePath, c.URL), bytes.NewReader(requestBody))
	if err != nil {
		return "", fmt.Errorf("while executing upgrade runtime request: %w", err)
	}
	if response.StatusCode != http.StatusAccepted {
		return "", c.handleUnsupportedStatusCode(response)
	}

	upgradeResponse := &UpgradeRuntimeResponse{}
	err = json.NewDecoder(response.Body).Decode(upgradeResponse)
	if err != nil {
		return "", fmt.Errorf("while decoding upgrade response: %w", err)
	}

	return upgradeResponse.OrchestrationID, nil
}

func (c *UpgradeClient) FetchRuntimeID(instanceID string) (string, error) {
	var runtimeID string
	err := wait.PollUntilContextTimeout(context.Background(), 3*time.Second, 1*time.Minute, false, func(ctx context.Context) (bool, error) {
		id, permanentError, err := c.fetchRuntimeID(instanceID)
		if err != nil && permanentError {
			return true, fmt.Errorf("cannot fetch runtimeID: %w", err)
		}
		if err != nil {
			c.log.Warn(fmt.Sprintf("runtime is not ready: %s ...", err))
			return false, nil
		}
		runtimeID = id
		return true, nil
	})
	if err != nil {
		return runtimeID, fmt.Errorf("while waiting for runtimeID: %w", err)
	}

	return runtimeID, nil
}

func (c *UpgradeClient) fetchRuntimeID(instanceID string) (string, bool, error) {
	response, err := c.executeRequest(http.MethodGet, fmt.Sprintf(infoRuntimePath, c.URL), nil)
	if err != nil {
		return "", false, fmt.Errorf("while executing fetch runtime request: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return "", false, c.handleUnsupportedStatusCode(response)
	}

	var runtimes []Runtime
	err = json.NewDecoder(response.Body).Decode(&runtimes)
	if err != nil {
		return "", true, fmt.Errorf("while decoding upgrade response: %w", err)
	}

	for _, runtime := range runtimes {
		if runtime.ServiceInstanceID != instanceID {
			continue
		}
		if runtime.RuntimeID == "" {
			continue
		}
		return runtime.RuntimeID, false, nil
	}

	return "", false, fmt.Errorf("runtimeID for instanceID %s not exist", instanceID)
}

func (c *UpgradeClient) AwaitOperationFinished(orchestrationID string, timeout time.Duration) error {
	err := wait.PollUntilContextTimeout(context.Background(), 10*time.Second, timeout, false, func(ctx context.Context) (bool, error) {
		permanentError, err := c.awaitOperationFinished(orchestrationID)
		if err != nil && permanentError {
			return true, fmt.Errorf("cannot fetch operation status: %w", err)
		}
		if err != nil {
			c.log.Warn(fmt.Sprintf("upgrade is not ready: %s ...", err))
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("while waiting for upgrade operation finished: %w", err)
	}

	return nil
}

func (c *UpgradeClient) awaitOperationFinished(orchestrationID string) (bool, error) {
	response, err := c.executeRequest(http.MethodGet, fmt.Sprintf(getOrchestrationPath, c.URL, orchestrationID), nil)
	if err != nil {
		return false, fmt.Errorf("while executing get orchestration request: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		return false, c.handleUnsupportedStatusCode(response)
	}

	orchestrationResponse := &OrchestrationResponse{}
	err = json.NewDecoder(response.Body).Decode(orchestrationResponse)
	if err != nil {
		return true, fmt.Errorf("while decoding orchestration response: %w", err)
	}

	switch orchestrationResponse.State {
	case "succeeded":
		return false, nil
	case "failed":
		return true, fmt.Errorf("operation is in failed state")
	default:
		return false, fmt.Errorf("operation is in %s state", orchestrationResponse.State)
	}
}

func (c *UpgradeClient) executeRequest(method, url string, body io.Reader) (*http.Response, error) {
	request, err := http.NewRequest(method, url, body)
	if err != nil {
		return &http.Response{}, fmt.Errorf("while creating request for KEB: %w", err)
	}
	request.Header.Set("X-Broker-API-Version", "2.14")

	response, err := c.client.Do(request)
	if err != nil {
		return &http.Response{}, fmt.Errorf("while executing request to KEB on: %s: %w", url, err)
	}

	return response, nil
}

func (c *UpgradeClient) handleUnsupportedStatusCode(response *http.Response) error {
	var body string
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		body = "cannot read body response"
	} else {
		body = string(responseBody)
	}

	return fmt.Errorf("unsupported status code %d: (%s): %w", response.StatusCode, body, err)
}
