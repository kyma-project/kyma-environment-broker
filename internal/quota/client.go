package quota

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"golang.org/x/oauth2/clientcredentials"
)

const (
	quotaServicePath = "%s/api/v2.0/subaccounts/%s/services/kymaruntime/plan/%s"
)

type Config struct {
	ClientID     string `envconfig:"optional"`
	ClientSecret string `envconfig:"optional"`
	AuthURL      string
	ServiceURL   string
	Retries      int           `envconfig:"default=5"`
	Interval     time.Duration `envconfig:"default=1s"`
}

type Client struct {
	ctx        context.Context
	httpClient *http.Client
	config     Config
	log        *slog.Logger
}

type Response struct {
	Plan  string `json:"plan"`
	Quota int    `json:"quota"`
}

func NewClient(ctx context.Context, config Config, log *slog.Logger) *Client {
	cfg := clientcredentials.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		TokenURL:     config.AuthURL,
	}
	httpClientOAuth := cfg.Client(ctx)

	return &Client{
		ctx:        ctx,
		httpClient: httpClientOAuth,
		config:     config,
		log:        log,
	}
}

func (c *Client) GetQuota(subAccountID, planName string) (int, error) {
	var lastErr error

	for i := 0; i < c.config.Retries; i++ {
		quota, err, retry := c.do(subAccountID, planName)
		if err == nil {
			return quota, nil
		}

		lastErr = err
		if !retry {
			return 0, lastErr
		}

		c.log.Warn(fmt.Sprintf("Error fetching quota, retrying in %s: %v", c.config.Interval, err))
		time.Sleep(c.config.Interval)
	}

	return 0, lastErr
}

func (c *Client) do(subAccountID, planName string) (int, error, bool) {
	req, err := http.NewRequestWithContext(c.ctx, http.MethodGet, fmt.Sprintf(quotaServicePath, c.config.ServiceURL, subAccountID, planName), nil)
	if err != nil {
		return 0, fmt.Errorf("while creating request: %w", err), false
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.log.Error(fmt.Sprintf("Authentication API returned: %v", err))
		return 0, fmt.Errorf("The authentication service is currently unavailable. Please try again later"), true
	}

	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			c.log.Warn(fmt.Sprintf("while closing response body: %s", err.Error()))
		}
	}(resp.Body)
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("while reading response body: %w", err), true
	}

	switch resp.StatusCode {
	case http.StatusOK:
		var response Response
		if err := json.Unmarshal(bodyBytes, &response); err != nil {
			return 0, fmt.Errorf("while unmarshaling response: %w", err), true
		}
		if response.Plan != planName {
			return 0, nil, false
		}
		return response.Quota, nil, false
	case http.StatusNotFound:
		c.log.Error(fmt.Sprintf("Quota API returned %d: %s", resp.StatusCode, string(bodyBytes)))
		return 0, fmt.Errorf("Subaccount %s does not exist", subAccountID), false
	default:
		c.log.Error(fmt.Sprintf("Quota API returned %d: %s", resp.StatusCode, string(bodyBytes)))
		return 0, fmt.Errorf("The provisioning service is currently unavailable. Please try again later"), true
	}
}
