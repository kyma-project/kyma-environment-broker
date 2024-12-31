package keb

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	scope                    = "broker:write"
	kymaServiceID            = "47c9dcbf-ff30-448e-ab36-d3bad66ba281"
	trialPlanID              = "7d55d31d-35ae-4438-bf13-6ffdfa107d9f"
	defaultExpirationSeconds = 600
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

type KEBClient struct {
	Token           *OAuthToken
	Host            string
	GlobalAccountID string
	SubaccountID    string
	UserID          string
	PlatformRegion  string
}

type KEBConfig struct {
	Host            string
	Credentials     OAuthCredentials
	GlobalAccountID string
	SubaccountID    string
	UserID          string
	PlatformRegion  string
	TokenURL        string
}

func NewKEBConfig() *KEBConfig {
	return &KEBConfig{
		Host:            getEnvOrThrow("KEB_HOST"),
		Credentials:     OAuthCredentials{ClientID: getEnvOrThrow("KEB_CLIENT_ID"), ClientSecret: getEnvOrThrow("KEB_CLIENT_SECRET")},
		GlobalAccountID: getEnvOrThrow("KEB_GLOBALACCOUNT_ID"),
		SubaccountID:    getEnvOrThrow("KEB_SUBACCOUNT_ID"),
		UserID:          getEnvOrThrow("KEB_USER_ID"),
		PlatformRegion:  os.Getenv("KEB_PLATFORM_REGION"),
		TokenURL:        getEnvOrThrow("KEB_TOKEN_URL"),
	}
}

func NewKEBClient(config *KEBConfig) *KEBClient {
	tokenURL := fmt.Sprintf("https://oauth2.%s/oauth2/token", config.Host)
	if config.TokenURL != "" {
		tokenURL = config.TokenURL
	}
	return &KEBClient{
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

func (c *KEBClient) BuildRequest(payload interface{}, endpoint, verb string) (*http.Request, error) {
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

func (c *KEBClient) BuildRequestWithoutToken(payload interface{}, endpoint, verb string) (*http.Request, error) {
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

func (c *KEBClient) CallKEB(payload interface{}, endpoint, verb string) (map[string]interface{}, error) {
	req, err := c.BuildRequest(payload, endpoint, verb)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("error calling KEB: %s %s", resp.Status, resp.Status)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	return result, nil
}

func (c *KEBClient) CallKEBWithoutToken(payload interface{}, endpoint, verb string) error {
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

func (c *KEBClient) GetSKR(instanceID string) (map[string]interface{}, error) {
	endpoint := fmt.Sprintf("service_instances/%s", instanceID)
	return c.CallKEB(nil, endpoint, "GET")
}

func (c *KEBClient) GetCatalog() (map[string]interface{}, error) {
	endpoint := "catalog"
	return c.CallKEB(nil, endpoint, "GET")
}

func (c *KEBClient) BuildPayload(name, instanceID, planID, region string, btpOperatorCreds map[string]interface{}) map[string]interface{} {
	payload := map[string]interface{}{
		"service_id": kymaServiceID,
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

func (c *KEBClient) ProvisionSKR(instanceID, planID, region string, btpOperatorCreds map[string]interface{}) (map[string]interface{}, error) {
	payload := c.BuildPayload(instanceID, instanceID, planID, region, btpOperatorCreds)
	endpoint := fmt.Sprintf("service_instances/%s", instanceID)
	return c.CallKEB(payload, endpoint, "PUT")
}

func (c *KEBClient) UpdateSKR(instanceID string, customParams, btpOperatorCreds map[string]interface{}, isMigration bool) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"service_id": kymaServiceID,
		"context": map[string]interface{}{
			"globalaccount_id": c.GlobalAccountID,
			"isMigration":      isMigration,
		},
		"parameters": customParams,
	}

	if btpOperatorCreds != nil {
		payload["context"].(map[string]interface{})["sm_operator_credentials"] = map[string]interface{}{
			"clientid":     btpOperatorCreds["clientid"],
			"clientsecret": btpOperatorCreds["clientsecret"],
			"sm_url":       btpOperatorCreds["smURL"],
			"url":          btpOperatorCreds["url"],
		}
	}

	endpoint := fmt.Sprintf("service_instances/%s?accepts_incomplete=true", instanceID)
	return c.CallKEB(payload, endpoint, "PATCH")
}

func (c *KEBClient) GetOperation(instanceID, operationID string) (map[string]interface{}, error) {
	endpoint := fmt.Sprintf("service_instances/%s/last_operation?operation=%s", instanceID, operationID)
	return c.CallKEB(nil, endpoint, "GET")
}

func (c *KEBClient) DeprovisionSKR(instanceID string) (map[string]interface{}, error) {
	endpoint := fmt.Sprintf("service_instances/%s?service_id=%s&plan_id=not-empty", instanceID, kymaServiceID)
	return c.CallKEB(nil, endpoint, "DELETE")
}

func (c *KEBClient) DownloadKubeconfig(instanceID string) (string, error) {
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

func (c *KEBClient) CreateBinding(instanceID, bindingID string, expirationSeconds int) (map[string]interface{}, error) {
	if expirationSeconds == 0 {
		expirationSeconds = defaultExpirationSeconds
	}
	payload := map[string]interface{}{
		"service_id": kymaServiceID,
		"plan_id":    "not-empty",
		"parameters": map[string]interface{}{
			"expiration_seconds": expirationSeconds,
		},
	}
	endpoint := fmt.Sprintf("service_instances/%s/service_bindings/%s?accepts_incomplete=false", instanceID, bindingID)
	return c.CallKEB(payload, endpoint, "PUT")
}

func (c *KEBClient) DeleteBinding(instanceID, bindingID string) (map[string]interface{}, error) {
	params := fmt.Sprintf("service_id=%s&plan_id=not-empty", kymaServiceID)
	endpoint := fmt.Sprintf("service_instances/%s/service_bindings/%s?accepts_incomplete=false&%s", instanceID, bindingID, params)
	return c.CallKEB(nil, endpoint, "DELETE")
}

func (c *KEBClient) GetBinding(instanceID, bindingID string) (map[string]interface{}, error) {
	endpoint := fmt.Sprintf("service_instances/%s/service_bindings/%s?accepts_incomplete=false", instanceID, bindingID)
	return c.CallKEB(nil, endpoint, "GET")
}

func (c *KEBClient) GetPlatformRegion() string {
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
