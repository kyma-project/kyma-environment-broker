package broker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/pivotal-cf/brokerapi/v12/domain/apiresponses"
	"golang.org/x/oauth2/clientcredentials"
)

const (
	kymaClassID                  = "47c9dcbf-ff30-448e-ab36-d3bad66ba281"
	AccountCleanupJob            = "accountcleanup-job"
	ServiceBindingCleanupJobName = "service-binding-cleanup-job"

	instancesURL       = "/oauth/v2/service_instances"
	expireInstanceURL  = "/expire/service_instance"
	deprovisionTmpl    = "%s%s/%s?service_id=%s&plan_id=%s"
	updateInstanceTmpl = "%s%s/%s"
	getInstanceTmpl    = "%s%s/%s"
	unbindTmpl         = "%s%s/%s/service_bindings/%s?service_id=%s&plan_id=%s"
)

type UnexpectedStatusCodeError struct {
	ExpectedStatusCode, UnexpectedStatusCode int
}

func NewUnexpectedStatusCodeError(expectedStatusCode, unexpectedStatusCode int) UnexpectedStatusCodeError {
	return UnexpectedStatusCodeError{
		ExpectedStatusCode:   expectedStatusCode,
		UnexpectedStatusCode: unexpectedStatusCode,
	}
}

func (e UnexpectedStatusCodeError) Error() string {
	return fmt.Sprintf("unexpected status code: want %d, got: %d", e.ExpectedStatusCode, e.UnexpectedStatusCode)
}

type (
	contextDTO struct {
		GlobalAccountID string `json:"globalaccount_id"`
		SubAccountID    string `json:"subaccount_id"`
		Active          *bool  `json:"active"`
	}

	parametersDTO struct {
		Expired *bool `json:"expired"`
	}

	serviceUpdatePatchDTO struct {
		ServiceID  string        `json:"service_id"`
		PlanID     string        `json:"plan_id"`
		Context    contextDTO    `json:"context"`
		Parameters parametersDTO `json:"parameters"`
	}

	serviceInstancesResponseDTO struct {
		Operation string `json:"operation"`
	}

	errorResponse struct {
		Error       string `json:"error"`
		Description string `json:"description"`
	}
)

type ClientConfig struct {
	URL          string
	TokenURL     string `envconfig:"optional"`
	ClientID     string `envconfig:"optional"`
	ClientSecret string `envconfig:"optional"`
	Scope        string `envconfig:"optional"`
}

type Client struct {
	brokerConfig   ClientConfig
	httpClient     *http.Client
	poller         Poller
	UserAgent      string
	RequestRetries int
}

func NewClientConfig(URL string) *ClientConfig {
	return &ClientConfig{
		URL: URL,
	}
}

func NewClient(ctx context.Context, config ClientConfig) *Client {
	return NewClientWithPoller(ctx, config, NewDefaultPoller())
}

func NewClientWithPoller(ctx context.Context, config ClientConfig, poller Poller) *Client {
	client := newClient(ctx, config)
	client.httpClient.Timeout = 30 * time.Second
	client.poller = poller
	return client
}

func NewClientWithRequestTimeoutAndRetries(ctx context.Context, config ClientConfig, requestTimeout time.Duration, requestRetries int) *Client {
	client := newClient(ctx, config)
	client.httpClient.Timeout = requestTimeout
	client.RequestRetries = requestRetries
	return client
}

func newClient(ctx context.Context, config ClientConfig) *Client {
	if config.TokenURL == "" {
		return &Client{
			brokerConfig: config,
			httpClient:   http.DefaultClient,
		}
	}

	cfg := clientcredentials.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		TokenURL:     config.TokenURL,
		Scopes:       []string{config.Scope},
	}

	httpClientOAuth := cfg.Client(ctx)

	return &Client{
		brokerConfig: config,
		httpClient:   httpClientOAuth,
	}
}

// Deprovision requests Runtime deprovisioning in KEB with given details
func (c *Client) Deprovision(instance internal.Instance) (string, error) {
	deprovisionURL, err := c.formatDeprovisionUrl(instance)
	if err != nil {
		return "", err
	}

	response := serviceInstancesResponseDTO{}
	slog.Info(fmt.Sprintf("Requesting deprovisioning of the environment with instance id: %q", instance.InstanceID))
	err = c.poller.Invoke(func() (bool, error) {
		err := c.executeRequest(http.MethodDelete, deprovisionURL, http.StatusAccepted, nil, &response)
		if err != nil {
			slog.Warn(fmt.Sprintf("while executing request: %s", err.Error()))
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		return "", fmt.Errorf("while waiting for successful deprovision call: %w", err)
	}

	return response.Operation, nil
}

// SendExpirationRequest requests Runtime suspension due to expiration
func (c *Client) SendExpirationRequest(instance internal.Instance) (suspensionUnderWay bool, err error) {
	request, err := prepareExpirationRequest(instance, c.brokerConfig.URL)
	if err != nil {
		return false, err
	}

	resp, err := c.httpClient.Do(request)
	if err != nil {
		return false, fmt.Errorf("while executing request URL: %s for instanceID: %s: %w", request.URL,
			instance.InstanceID, err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Warn(err.Error())
		}
	}()

	return processResponse(instance.InstanceID, resp.StatusCode, resp)
}

func (c *Client) GetInstanceRequest(instanceID string) (response *http.Response, err error) {
	request, err := prepareGetRequest(instanceID, c.brokerConfig.URL)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("while executing request URL: %s for instanceID: %s: %w", request.URL,
			instanceID, err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Warn(err.Error())
		}
	}()

	return resp, nil
}

// Unbind requests Service Binding unbinding in KEB with given details
func (c *Client) Unbind(binding internal.Binding) error {
	unbindURL, err := c.formatUnbindUrl(binding)
	if err != nil {
		return err
	}

	slog.Info(fmt.Sprintf("sending unbind request for service binding with ID %q and instance ID: %q", binding.ID, binding.InstanceID))

	emptyResponse := &apiresponses.EmptyResponse{}
	for requestAttemptNum := 1; requestAttemptNum <= c.RequestRetries; requestAttemptNum++ {
		if err = c.executeRequest(http.MethodDelete, unbindURL, http.StatusOK, nil, emptyResponse); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				slog.Warn(fmt.Sprintf("request failed - timeout (attempt %d/%d)", requestAttemptNum, c.RequestRetries))
				continue
			}
			break
		}
		break
	}
	if err != nil {
		return err
	}

	slog.Info(fmt.Sprintf("successfully unbound service binding with ID %q and instance ID %q", binding.ID, binding.InstanceID))

	return nil
}

func processResponse(instanceID string, statusCode int, resp *http.Response) (suspensionUnderWay bool, err error) {
	switch statusCode {
	case http.StatusAccepted, http.StatusOK:
		{
			slog.Info(fmt.Sprintf("Request for instanceID: %s accepted with status: %+v", instanceID, statusCode))
			operation, err := decodeOperation(resp)
			if err != nil {
				return false, err
			}
			slog.Info(fmt.Sprintf("For instanceID: %s we received operation: %s", instanceID, operation))
			return true, nil
		}
	case http.StatusUnprocessableEntity:
		{
			slog.Warn(fmt.Sprintf("For instanceID: %s we received entity unprocessable - status: %+v", instanceID, statusCode))
			description, errorString, err := decodeErrorResponse(resp)
			if err != nil {
				return false, fmt.Errorf("for instanceID: %s: %w", instanceID, err)
			}
			slog.Warn(fmt.Sprintf("error: %+v description: %+v instanceID: %s", errorString, description, instanceID))
			return false, nil
		}
	default:
		{
			if statusCode >= 200 && statusCode <= 299 {
				return false, fmt.Errorf("for instanceID: %s we received unexpected status: %+v", instanceID, statusCode)
			}
			description, errorString, err := decodeErrorResponse(resp)
			if err != nil {
				return false, fmt.Errorf("for instanceID: %s: %w", instanceID, err)
			}
			return false, fmt.Errorf("error: %+v description: %+v instanceID: %s", errorString, description, instanceID)
		}
	}
}

func decodeOperation(resp *http.Response) (string, error) {
	response := serviceInstancesResponseDTO{}
	err := json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return "", fmt.Errorf("while decoding response body: %w", err)
	}
	return response.Operation, nil
}

func decodeErrorResponse(resp *http.Response) (string, string, error) {
	response := errorResponse{}
	err := json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return "", "", fmt.Errorf("while decoding error response body: %w", err)
	}
	return response.Description, response.Error, nil
}

func prepareExpirationRequest(instance internal.Instance, brokerConfigURL string) (*http.Request, error) {
	expireInstanceUrl := fmt.Sprintf(updateInstanceTmpl, brokerConfigURL, expireInstanceURL, instance.InstanceID)

	slog.Info(fmt.Sprintf("Requesting expiration of the environment with instanceID: %q", instance.InstanceID))

	request, err := http.NewRequest(http.MethodPut, expireInstanceUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("while creating request for instanceID: %s: %w", instance.InstanceID, err)
	}
	return request, nil
}

func prepareGetRequest(instanceID string, brokerConfigURL string) (*http.Request, error) {
	getInstanceUrl := fmt.Sprintf(getInstanceTmpl, brokerConfigURL, instancesURL, instanceID)

	request, err := http.NewRequest(http.MethodGet, getInstanceUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("while creating GET request for instanceID: %s: %w", instanceID, err)
	}
	request.Header.Set("X-Broker-API-Version", "2.14")
	return request, nil
}

func (c *Client) formatDeprovisionUrl(instance internal.Instance) (string, error) {
	if len(instance.ServicePlanID) == 0 {
		return "", fmt.Errorf("empty ServicePlanID")
	}

	return fmt.Sprintf(deprovisionTmpl, c.brokerConfig.URL, instancesURL, instance.InstanceID, kymaClassID, instance.ServicePlanID), nil
}

func (c *Client) formatUnbindUrl(binding internal.Binding) (string, error) {
	if len(binding.InstanceID) == 0 {
		return "", fmt.Errorf("empty InstanceID")
	}

	return fmt.Sprintf(unbindTmpl, c.brokerConfig.URL, instancesURL, binding.InstanceID, binding.ID, kymaClassID, AWSPlanID), nil
}

func (c *Client) executeRequest(method, url string, expectedStatus int, requestBody io.Reader, responseBody interface{}) error {
	request, err := http.NewRequest(method, url, requestBody)
	if err != nil {
		return fmt.Errorf("while creating request: %w", err)
	}
	request.Header.Set("X-Broker-API-Version", "2.14")
	if len(c.UserAgent) != 0 {
		request.Header.Set("User-Agent", c.UserAgent)
	}

	resp, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Warn(err.Error())
		}
	}()
	if resp.StatusCode != expectedStatus {
		return NewUnexpectedStatusCodeError(expectedStatus, resp.StatusCode)
	}

	err = json.NewDecoder(resp.Body).Decode(responseBody)
	if err != nil {
		return fmt.Errorf("while decoding response body: %w", err)
	}

	return nil
}

func (c *Client) warnOnError(do func() error) {
	if err := do(); err != nil {
		slog.Warn(err.Error())
	}
}

// setHttpClient auxiliary method of testing to get rid of oAuth client wrapper
func (c *Client) setHttpClient(httpClient *http.Client) {
	c.httpClient = httpClient
}
