package quota

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"golang.org/x/oauth2/clientcredentials"
)

const (
	quotaServicePath = "%s/api/v2.0/subaccounts/%s/services/kymaruntime/plan/%s"
)

type Config struct {
	ClientID     string
	ClientSecret string
	AuthURL      string
	ServiceURL   string
}

type Client struct {
	ctx        context.Context
	httpClient *http.Client
	config     Config
	log        *slog.Logger
}

type Response struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
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
	req, err := http.NewRequestWithContext(c.ctx, http.MethodGet, fmt.Sprintf(quotaServicePath, c.config.ServiceURL, subAccountID, planName), nil)
	if err != nil {
		return 0, fmt.Errorf("while creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("while performing request: %w", err)
	}

	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			c.log.Warn(fmt.Sprintf("while closing response body: %s", err.Error()))
		}
	}(resp.Body)
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("while reading response body: %w", err)
	}

	var response Response
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return 0, fmt.Errorf("while unmarshaling response: %w", err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		if response.Plan != planName {
			return 0, nil
		}
		return response.Quota, nil
	case http.StatusNotFound:
		c.log.Error(fmt.Sprintf("Quota API returned %d: %s", resp.StatusCode, response.Error.Message))
		return 0, fmt.Errorf("Subaccount %s does not exist", subAccountID)
	default:
		c.log.Error(fmt.Sprintf("Quota API returned %d: %s", resp.StatusCode, response.Error.Message))
		return 0, fmt.Errorf("The provisioning service is currently unavailable. Please try again later")
	}
}
