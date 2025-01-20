package broker

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	scope         = "broker:write"
	KymaServiceID = "47c9dcbf-ff30-448e-ab36-d3bad66ba281"
	trialPlanID   = "7d55d31d-35ae-4438-bf13-6ffdfa107d9f"
)

type OAuthCredentials struct {
	ClientID     string
	ClientSecret string
}
type BTPOperatorCreds struct {
	ClientID     string
	ClientSecret string
	SMURL        string
	TokenURL     string
}
type OAuthToken struct {
	TokenURL    string
	Credentials OAuthCredentials
	Token       string
	Expiry      time.Time
}

type BrokerClient struct {
	Token           *OAuthToken
	Host            string
	GlobalAccountID string
	SubaccountID    string
	UserID          string
	PlatformRegion  string
}

type BrokerConfig struct {
	Host            string
	Credentials     OAuthCredentials
	GlobalAccountID string
	SubaccountID    string
	UserID          string
	PlatformRegion  string
	TokenURL        string
}

func NewBrokerConfig() *BrokerConfig {
	return &BrokerConfig{
		Host:            getEnvOrThrow("KEB_HOST"),
		Credentials:     OAuthCredentials{ClientID: getEnvOrThrow("KEB_CLIENT_ID"), ClientSecret: getEnvOrThrow("KEB_CLIENT_SECRET")},
		GlobalAccountID: getEnvOrThrow("KEB_GLOBALACCOUNT_ID"),
		SubaccountID:    getEnvOrThrow("KEB_SUBACCOUNT_ID"),
		UserID:          getEnvOrThrow("KEB_USER_ID"),
		PlatformRegion:  os.Getenv("KEB_PLATFORM_REGION"),
		TokenURL:        getEnvOrThrow("KEB_TOKEN_URL"),
	}
}

func NewBrokerClient(config *BrokerConfig) *BrokerClient {
	tokenURL := fmt.Sprintf("https://oauth2.%s/oauth2/token", config.Host)
	if config.TokenURL != "" {
		tokenURL = config.TokenURL
	}
	return &BrokerClient{
		Token:           &OAuthToken{TokenURL: tokenURL, Credentials: config.Credentials},
		Host:            config.Host,
		GlobalAccountID: config.GlobalAccountID,
		SubaccountID:    config.SubaccountID,
		UserID:          config.UserID,
		PlatformRegion:  config.PlatformRegion,
	}
}

func (o *OAuthToken) GetToken(scopes string) (string, error) {
	if o.Token != "" && time.Now().Before(o.Expiry) {
		return o.Token, nil
	}

	data := fmt.Sprintf("grant_type=client_credentials&client_id=%s&client_secret=%s&scope=%s",
		o.Credentials.ClientID, o.Credentials.ClientSecret, scopes)
	req, err := http.NewRequest("POST", o.TokenURL, strings.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.New("failed to get token")
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	o.Token = result["access_token"].(string)
	o.Expiry = time.Now().Add(time.Duration(result["expires_in"].(float64)) * time.Second)

	return o.Token, nil
}

func (c *BrokerClient) BuildRequest(payload interface{}, endpoint, verb string) (*http.Request, error) {
	token, err := c.Token.GetToken(scope)
	if err != nil {
		return nil, err
	}
	platformRegion := c.GetPlatformRegion()
	url := fmt.Sprintf("https://kyma-env-broker.%s/oauth/%sv2/%s", c.Host, platformRegion, endpoint)
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(verb, url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Broker-API-Version", "2.14")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func (c *BrokerClient) BuildRequestWithoutToken(payload interface{}, endpoint, verb string) (*http.Request, error) {
	url := fmt.Sprintf("https://kyma-env-broker.%s/oauth/v2/%s", c.Host, endpoint)
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(verb, url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Broker-API-Version", "2.14")
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func (c *BrokerClient) CallBroker(payload interface{}, endpoint, verb string) (response map[string]interface{}, statusCode *int, err error) {
	req, err := c.BuildRequest(payload, endpoint, verb)
	if err != nil {
		return nil, nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusCreated {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return result, &resp.StatusCode, fmt.Errorf("error reading response body: %v", err)
		}
		return result, &resp.StatusCode, fmt.Errorf("error calling Broker: %s %s", resp.Status, string(body))
	}
	return result, &resp.StatusCode, nil
}

func (c *BrokerClient) CallBrokerWithoutToken(payload interface{}, endpoint, verb string) error {
	req, err := c.BuildRequestWithoutToken(payload, endpoint, verb)
	fmt.Printf("Request: %s %s\n", req.Method, req.URL)
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	fmt.Println("Response:", string(body))
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusUnauthorized {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
	return nil
}

func (c *BrokerClient) GetInstance(instanceID string) (response map[string]interface{}, statusCode *int, err error) {
	endpoint := fmt.Sprintf("service_instances/%s", instanceID)
	return c.CallBroker(nil, endpoint, "GET")
}

func (c *BrokerClient) GetCatalog() (response map[string]interface{}, statusCode *int, err error) {
	endpoint := "catalog"
	return c.CallBroker(nil, endpoint, "GET")
}

func (c *BrokerClient) BuildPayload(name, instanceID, planID, region string, btpOperatorCreds, customParams map[string]interface{}) map[string]interface{} {
	payload := map[string]interface{}{
		"service_id": KymaServiceID,
		"plan_id":    planID,
		"context": map[string]interface{}{
			"globalaccount_id": c.GlobalAccountID,
			"subaccount_id":    c.SubaccountID,
			"user_id":          c.UserID,
		},
		"parameters": map[string]interface{}{
			"name": name,
		},
	}

	for key, value := range customParams {
		payload["parameters"].(map[string]interface{})[key] = value
	}

	if planID != trialPlanID {
		payload["parameters"].(map[string]interface{})["region"] = region
	}

	if btpOperatorCreds != nil {
		payload["context"].(map[string]interface{})["sm_operator_credentials"] = map[string]interface{}{
			"clientid":     btpOperatorCreds["clientid"],
			"clientsecret": btpOperatorCreds["clientsecret"],
			"sm_url":       btpOperatorCreds["smURL"],
			"url":          btpOperatorCreds["url"],
		}
	}

	return payload
}

func (c *BrokerClient) ProvisionInstance(instanceID, planID, region string, btpOperatorCreds, customParams map[string]interface{}) (response map[string]interface{}, statusCode *int, err error) {
	payload := c.BuildPayload(instanceID, instanceID, planID, region, btpOperatorCreds, customParams)
	endpoint := fmt.Sprintf("service_instances/%s", instanceID)
	return c.CallBroker(payload, endpoint, "PUT")
}

func (c *BrokerClient) UpdateInstance(instanceID string, customParams map[string]interface{}) (response map[string]interface{}, statusCode *int, err error) {
	payload := map[string]interface{}{
		"service_id": KymaServiceID,
		"context": map[string]interface{}{
			"globalaccount_id": c.GlobalAccountID,
		},
		"parameters": customParams,
	}

	endpoint := fmt.Sprintf("service_instances/%s?accepts_incomplete=true", instanceID)
	return c.CallBroker(payload, endpoint, "PATCH")
}

func (c *BrokerClient) GetOperation(instanceID, operationID string) (response map[string]interface{}, statusCode *int, err error) {
	endpoint := fmt.Sprintf("service_instances/%s/last_operation?operation=%s", instanceID, operationID)
	return c.CallBroker(nil, endpoint, "GET")
}

func (c *BrokerClient) DeprovisionInstance(instanceID string) (response map[string]interface{}, statusCode *int, err error) {
	endpoint := fmt.Sprintf("service_instances/%s?service_id=%s&plan_id=not-empty", instanceID, KymaServiceID)
	return c.CallBroker(nil, endpoint, "DELETE")
}

func (c *BrokerClient) DownloadKubeconfig(instanceID string) (string, error) {
	downloadUrl := fmt.Sprintf("https://kyma-env-broker.%s/kubeconfig/%s", c.Host, instanceID)
	resp, err := http.Get(downloadUrl)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download kubeconfig: %s", resp.Status)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (c *BrokerClient) CreateBinding(instanceID, bindingID string, expirationSeconds int) (response map[string]interface{}, statusCode *int, err error) {
	payload := map[string]interface{}{
		"service_id": KymaServiceID,
		"plan_id":    "not-empty",
		"parameters": map[string]interface{}{
			"expiration_seconds": expirationSeconds,
		},
	}
	endpoint := fmt.Sprintf("service_instances/%s/service_bindings/%s?accepts_incomplete=false", instanceID, bindingID)
	return c.CallBroker(payload, endpoint, "PUT")
}

func (c *BrokerClient) DeleteBinding(instanceID, bindingID string) (response map[string]interface{}, statusCode *int, err error) {
	params := fmt.Sprintf("service_id=%s&plan_id=not-empty", KymaServiceID)
	endpoint := fmt.Sprintf("service_instances/%s/service_bindings/%s?accepts_incomplete=false&%s", instanceID, bindingID, params)
	return c.CallBroker(nil, endpoint, "DELETE")
}

func (c *BrokerClient) GetBinding(instanceID, bindingID string) (response map[string]interface{}, statusCode *int, err error) {
	endpoint := fmt.Sprintf("service_instances/%s/service_bindings/%s?accepts_incomplete=false", instanceID, bindingID)
	return c.CallBroker(nil, endpoint, "GET")
}

func (c *BrokerClient) GetPlatformRegion() string {
	if c.PlatformRegion != "" {
		return fmt.Sprintf("%s/", c.PlatformRegion)
	}
	return ""
}

func getEnvOrThrow(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(fmt.Sprintf("Environment variable %s not set", key))
	}
	return value
}
