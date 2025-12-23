package broker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pivotal-cf/brokerapi/v7/domain"
	"github.com/thanhpk/randstr"
	"golang.org/x/oauth2/clientcredentials"
	"k8s.io/apimachinery/pkg/util/wait"
)

// s
type Config struct {
	ClientName     string
	ClientID       string
	ClientSecret   string
	Scope          string `envconfig:"default=broker:write"`
	TokenURL       string
	URL            string
	PlanID         string
	Region         string
	PlatformRegion string `envconfig:"optional"`
}

type BrokerOAuthConfig struct {
	ClientID     string
	ClientSecret string
	Scope        string
}

type Client struct {
	brokerConfig    Config
	clusterName     string
	instanceID      string
	globalAccountID string
	subAccountID    string
	userID          string

	client *http.Client
	log    *slog.Logger
}

func NewClient(ctx context.Context, config Config, globalAccountID, instanceID, subAccountID, userID string, oAuthCfg BrokerOAuthConfig, log *slog.Logger) *Client {
	cfg := clientcredentials.Config{
		ClientID:     oAuthCfg.ClientID,
		ClientSecret: oAuthCfg.ClientSecret,
		TokenURL:     config.TokenURL,
		Scopes:       []string{oAuthCfg.Scope},
	}
	httpClientOAuth := cfg.Client(ctx)
	httpClientOAuth.Timeout = 30 * time.Second

	return &Client{
		brokerConfig:    config,
		instanceID:      instanceID,
		clusterName:     fmt.Sprintf("%s-%s", "e2e-provisioning", strings.ToLower(randstr.String(10))),
		globalAccountID: globalAccountID,
		client:          httpClientOAuth,
		log:             log,
		subAccountID:    subAccountID,
		userID:          userID,
	}
}

const (
	kymaClassID = "47c9dcbf-ff30-448e-ab36-d3bad66ba281"
	trialPlanID = "7d55d31d-35ae-4438-bf13-6ffdfa107d9f"
)

type inputContext struct {
	TenantID        string `json:"tenant_id"`
	SubAccountID    string `json:"subaccount_id"`
	UserID          string `json:"user_id"`
	GlobalAccountID string `json:"globalaccount_id"`
	Active          *bool  `json:"active,omitempty"`
}

type provisionResponse struct {
	Operation string `json:"operation"`
}

type lastOperationResponse struct {
	State string `json:"state"`
}

type instanceDetailsResponse struct {
	DashboardURL string `json:"dashboard_url"`
}

type provisionParameters struct {
	Name        string `json:"name"`
	Region      string `json:"region,omitempty"`
	KymaVersion string `json:"kymaVersion,omitempty"`
}

// ProvisionRuntime requests Runtime provisioning in KEB
// kymaVersion is optional, if it is empty, the default KEB version will be used
func (c *Client) ProvisionRuntime(kymaVersion string) (string, error) {
	c.log.Info(fmt.Sprintf("Provisioning Runtime [instanceID: %s, NAME: %s]", c.InstanceID(), c.ClusterName()))
	requestByte, err := c.prepareProvisionDetails(kymaVersion)
	if err != nil {
		return "", fmt.Errorf("while preparing provision details: %w", err)
	}
	c.log.Info(fmt.Sprintf("Provisioning parameters: %v", string(requestByte)))

	provisionURL := fmt.Sprintf("%s/service_instances/%s", c.baseURL(), c.InstanceID())
	response := provisionResponse{}
	err = wait.PollUntilContextTimeout(context.Background(), time.Second, time.Second*5, false, func(ctx context.Context) (bool, error) {
		err := c.executeRequest(http.MethodPut, provisionURL, http.StatusAccepted, bytes.NewReader(requestByte), &response)
		if err != nil {
			c.log.Warn(fmt.Errorf("while executing request: %w", err).Error())
			return false, nil
		}
		if response.Operation == "" {
			c.log.Warn("Got empty operation ID")
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return "", fmt.Errorf("while waiting for successful provision call: %w", err)
	}
	c.log.Info(fmt.Sprintf("Successfully send provision request, got operation ID %s", response.Operation))

	return response.Operation, nil
}

func (c *Client) DeprovisionRuntime() (string, error) {
	format := "%s/service_instances/%s?service_id=%s&plan_id=%s"
	deprovisionURL := fmt.Sprintf(format, c.baseURL(), c.InstanceID(), kymaClassID, c.brokerConfig.PlanID)

	response := provisionResponse{}
	c.log.Info(fmt.Sprintf("Deprovisioning Runtime [instanceID: %s, NAME: %s]", c.instanceID, c.clusterName))
	err := wait.PollUntilContextTimeout(context.Background(), time.Second, time.Second*5, false, func(ctx context.Context) (bool, error) {
		err := c.executeRequest(http.MethodDelete, deprovisionURL, http.StatusAccepted, nil, &response)
		if err != nil {
			c.log.Warn(fmt.Errorf("while executing request: %w", err).Error())
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return "", fmt.Errorf("while waiting for successful deprovision call: %w", err)
	}
	c.log.Info(fmt.Sprintf("Successfully send deprovision request, got operation ID %s", response.Operation))
	return response.Operation, nil
}

func (c *Client) SuspendRuntime() error {
	c.log.Info(fmt.Sprintf("Suspending Runtime [instanceID: %s, NAME: %s]", c.instanceID, c.clusterName))
	requestByte, err := c.prepareUpdateDetails(BoolPtr(false))
	if err != nil {
		return fmt.Errorf("while preparing update details: %w", err)
	}
	c.log.Info(fmt.Sprintf("Suspension parameters: %v", string(requestByte)))

	format := "%s/service_instances/%s"
	suspensionURL := fmt.Sprintf(format, c.baseURL(), c.instanceID)

	suspensionResponse := instanceDetailsResponse{}
	err = wait.PollUntilContextTimeout(context.Background(), time.Second, time.Second*5, false, func(ctx context.Context) (bool, error) {
		err := c.executeRequest(http.MethodPatch, suspensionURL, http.StatusOK, bytes.NewReader(requestByte), &suspensionResponse)
		if err != nil {
			c.log.Warn(fmt.Errorf("while executing request: %w", err).Error())
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("while waiting for successful suspension call: %w", err)
	}
	c.log.Info(fmt.Sprintf("Successfully send suspension request for %s", suspensionResponse))
	return nil
}

func (c *Client) UnsuspendRuntime() error {
	c.log.Info(fmt.Sprintf("Unsuspending Runtime [instanceID: %s, NAME: %s]", c.InstanceID(), c.ClusterName()))
	requestByte, err := c.prepareUpdateDetails(BoolPtr(true))
	if err != nil {
		return fmt.Errorf("while preparing update details: %w", err)
	}
	c.log.Info(fmt.Sprintf("Unsuspension parameters: %v", string(requestByte)))

	format := "%s/service_instances/%s"
	suspensionURL := fmt.Sprintf(format, c.baseURL(), c.InstanceID())

	unsuspensionResponse := instanceDetailsResponse{}
	err = wait.PollUntilContextTimeout(context.Background(), time.Second, time.Second*5, false, func(ctx context.Context) (bool, error) {
		err := c.executeRequest(http.MethodPatch, suspensionURL, http.StatusOK, bytes.NewReader(requestByte), &unsuspensionResponse)
		if err != nil {
			c.log.Warn(fmt.Errorf("while executing request: %w", err).Error())
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("while waiting for successful unsuspension call: %w", err)
	}
	c.log.Info(fmt.Sprintf("Successfully send unsuspension request for %s", unsuspensionResponse))
	return nil
}

func (c *Client) GlobalAccountID() string {
	return c.globalAccountID
}

func (c *Client) InstanceID() string {
	return c.instanceID
}

func (c *Client) SetInstanceID(id string) {
	c.instanceID = id
}

func (c *Client) SubAccountID() string {
	return c.subAccountID
}

func (c *Client) UserID() string {
	return c.userID
}

func (c *Client) ClusterName() string {
	return c.clusterName
}

func (c *Client) AwaitOperationSucceeded(operationID string, timeout time.Duration) error {
	lastOperationURL := fmt.Sprintf("%s/service_instances/%s/last_operation?operation=%s", c.baseURL(), c.InstanceID(), operationID)
	if operationID == "" {
		lastOperationURL = fmt.Sprintf("%s/service_instances/%s/last_operation", c.baseURL(), c.InstanceID())
	}

	c.log.Info(fmt.Sprintf("Waiting for operation at most %s", timeout.String()))

	response := lastOperationResponse{}
	err := wait.PollUntilContextTimeout(context.Background(), 5*time.Minute, timeout, false, func(ctx context.Context) (bool, error) {
		err := c.executeRequest(http.MethodGet, lastOperationURL, http.StatusOK, nil, &response)
		if err != nil {
			c.log.Warn(fmt.Errorf("while executing request: %w", err).Error())
			return false, nil
		}
		c.log.Info(fmt.Sprintf("Last operation status: %s", response.State))
		switch domain.LastOperationState(response.State) {
		case domain.Succeeded:
			c.log.Info("Operation succeeded!")
			return true, nil
		case domain.InProgress:
			return false, nil
		case domain.Failed:
			c.log.Info("Operation failed!")
			return true, errors.New("provisioning failed")
		default:
			if response.State == "" {
				c.log.Info("Got empty last operation response")
				return false, nil
			}
			return false, nil
		}
	})
	if err != nil {
		return fmt.Errorf("while waiting for succeeded last operation: %w", err)
	}
	return nil
}

func (c *Client) FetchDashboardURL() (string, error) {
	instanceDetailsURL := fmt.Sprintf("%s/service_instances/%s", c.baseURL(), c.InstanceID())

	c.log.Info("Fetching the Runtime's dashboard URL")
	response := instanceDetailsResponse{}
	err := wait.PollUntilContextTimeout(context.Background(), time.Second, time.Second*5, false, func(ctx context.Context) (bool, error) {
		err := c.executeRequest(http.MethodGet, instanceDetailsURL, http.StatusOK, nil, &response)
		if err != nil {
			c.log.Warn(fmt.Errorf("while executing request: %w", err).Error())
			return false, nil
		}
		if response.DashboardURL == "" {
			c.log.Warn("got empty dashboardURL")
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return "", fmt.Errorf("while waiting for dashboardURL: %w", err)
	}
	c.log.Info(fmt.Sprintf("Successfully fetched dashboard URL: %s", response.DashboardURL))

	return response.DashboardURL, nil
}

func (c *Client) prepareProvisionDetails(customVersion string) ([]byte, error) {
	parameters := provisionParameters{
		Name:        c.clusterName,
		KymaVersion: customVersion, // If empty filed will be omitted
	}
	if c.brokerConfig.PlanID != trialPlanID {
		parameters.Region = c.brokerConfig.Region
	}
	ctx := inputContext{
		TenantID:        "1eba80dd-8ff6-54ee-be4d-77944d17b10b",
		SubAccountID:    c.subAccountID,
		UserID:          c.userID,
		GlobalAccountID: c.globalAccountID,
	}
	rawParameters, err := json.Marshal(parameters)
	if err != nil {
		return nil, fmt.Errorf("while marshalling parameters body: %w", err)
	}
	rawContext, err := json.Marshal(ctx)
	if err != nil {
		return nil, fmt.Errorf("while marshalling context body: %w", err)
	}
	requestBody := domain.ProvisionDetails{
		ServiceID:        kymaClassID,
		PlanID:           c.brokerConfig.PlanID,
		OrganizationGUID: uuid.New().String(),
		SpaceGUID:        uuid.New().String(),
		RawContext:       rawContext,
		RawParameters:    rawParameters,
		MaintenanceInfo: &domain.MaintenanceInfo{
			Version:     "0.1.0",
			Description: "Kyma environment broker e2e-provisioning test",
		},
	}

	requestByte, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("while marshalling request body: %w", err)
	}
	return requestByte, nil
}

func (c *Client) prepareUpdateDetails(active *bool) ([]byte, error) {
	ctx := inputContext{
		TenantID:        "1eba80dd-8ff6-54ee-be4d-77944d17b10b",
		SubAccountID:    c.subAccountID,
		GlobalAccountID: c.globalAccountID,
		Active:          active,
	}
	rawContext, err := json.Marshal(ctx)
	if err != nil {
		return nil, fmt.Errorf("while marshalling context body: %w", err)
	}
	requestBody := domain.UpdateDetails{
		ServiceID:  kymaClassID,
		PlanID:     c.brokerConfig.PlanID,
		RawContext: rawContext,
		MaintenanceInfo: &domain.MaintenanceInfo{
			Version:     "0.1.0",
			Description: "Kyma environment broker e2e-provisioning test",
		},
	}
	requestByte, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("while marshalling request body: %w", err)
	}
	return requestByte, nil
}

func (c *Client) executeRequest(method, url string, expectedStatus int, body io.Reader, responseBody interface{}) error {
	request, err := http.NewRequest(method, url, body)
	if err != nil {
		return fmt.Errorf("while creating request for provisioning: %w", err)
	}
	request.Header.Set("X-Broker-API-Version", "2.14")

	resp, err := c.client.Do(request)
	if err != nil {
		return fmt.Errorf("while executing request URL: %s: %w", url, err)
	}
	defer c.warnOnError(resp.Body.Close)
	if resp.StatusCode != expectedStatus {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			c.log.Error(err.Error())
			os.Exit(1)
		}
		bodyString := string(bodyBytes)
		c.log.Warn(fmt.Sprintf("%s", bodyString))
		return fmt.Errorf("got unexpected status code while calling Kyma Environment Broker: want: %d, got: %d (url=%s)", expectedStatus, resp.StatusCode, url)
	}

	err = json.NewDecoder(resp.Body).Decode(responseBody)
	if err != nil {
		return fmt.Errorf("while decoding body: %w", err)
	}

	return nil
}

func (c *Client) warnOnError(do func() error) {
	if err := do(); err != nil {
		c.log.Warn(err.Error())
	}
}

func (c *Client) baseURL() string {
	base := fmt.Sprintf("%s/oauth", c.brokerConfig.URL)
	if c.brokerConfig.PlatformRegion == "" {
		return fmt.Sprintf("%s/v2", base)
	}
	return fmt.Sprintf("%s/%s/v2", base, c.brokerConfig.PlatformRegion)
}

func BoolPtr(b bool) *bool {
	return &b
}
