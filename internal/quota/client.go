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

type CisConfig struct {
	ClientID     string
	ClientSecret string
	AuthURL      string
	ServiceURL   string
}

type CisClient struct {
	ctx        context.Context
	httpClient *http.Client
	config     CisConfig
	log        *slog.Logger
}

type Response struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
	Plan  string `json:"plan"`
	Quota int    `json:"quota"`
}

func NewCisClient(ctx context.Context, config CisConfig, log *slog.Logger) *CisClient {
	cfg := clientcredentials.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		TokenURL:     config.AuthURL,
	}
	httpClientOAuth := cfg.Client(ctx)

	return &CisClient{
		ctx:        ctx,
		httpClient: httpClientOAuth,
		config:     config,
		log:        log,
	}
}

func (c *CisClient) GetQuota(subAccountID, planName string) (int, error) {
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

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("API error: %s", response.Error.Message)
	}

	if response.Plan != planName {
		return 0, nil
	}

	return response.Quota, nil
}
