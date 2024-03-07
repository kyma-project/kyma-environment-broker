package subaccountsync

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"golang.org/x/oauth2/clientcredentials"
	"golang.org/x/time/rate"
)

type RateLimitedCisClient struct {
	ctx         context.Context
	httpClient  *http.Client
	config      CisEndpointConfig
	log         *slog.Logger
	RateLimiter *rate.Limiter
}

type RateLimiter interface {
	Do(req *http.Request) (*http.Response, error)
}

func NewRateLimitedCisClient(ctx context.Context, config CisEndpointConfig, log *slog.Logger) *RateLimitedCisClient {
	cfg := clientcredentials.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		TokenURL:     config.AuthURL,
	}
	httpClientOAuth := cfg.Client(ctx)

	rl := rate.NewLimiter(rate.Every(config.RateLimitingInterval), config.MaxRequestsPerInterval)

	return &RateLimitedCisClient{
		ctx:         ctx,
		httpClient:  httpClientOAuth,
		config:      config,
		log:         log,
		RateLimiter: rl,
	}
}

func (c *RateLimitedCisClient) Do(req *http.Request) (*http.Response, error) {
	err := c.RateLimiter.Wait(c.ctx)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *RateLimitedCisClient) handleErrorStatusCode(response *http.Response) string {
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Sprintf("server returned %d status code, response body is unreadable", response.StatusCode)
	}

	return fmt.Sprintf("server returned %d status code, body: %s", response.StatusCode, string(body))
}
