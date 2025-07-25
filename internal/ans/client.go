package ans

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/kyma-project/kyma-environment-broker/internal/ans/events"
	"github.com/kyma-project/kyma-environment-broker/internal/ans/notifications"
	"github.com/pkg/errors"
	"golang.org/x/oauth2/clientcredentials"
	"golang.org/x/time/rate"
)

type Client struct {
	ctx         context.Context
	httpClient  *http.Client
	config      EndpointConfig
	log         *slog.Logger
	RateLimiter *rate.Limiter
}

type EventsClient struct {
	Client
}

type NotificationsClient struct {
	Client
}

type RateLimiter interface {
	Do(req *http.Request) (*http.Response, error)
}

func NewClient(ctx context.Context, config EndpointConfig, log *slog.Logger) *Client {
	cfg := clientcredentials.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		TokenURL:     config.AuthURL,
	}
	httpClientOAuth := cfg.Client(ctx)

	rl := rate.NewLimiter(rate.Every(config.RateLimitingInterval), config.MaxRequestsPerInterval)

	return &Client{
		ctx:         ctx,
		httpClient:  httpClientOAuth,
		config:      config,
		log:         log,
		RateLimiter: rl,
	}
}

func NewEventsClient(ctx context.Context, config EndpointConfig, log *slog.Logger) *EventsClient {
	return &EventsClient{
		Client: *NewClient(ctx, config, log),
	}
}

func NewNotificationsClient(ctx context.Context, config EndpointConfig, log *slog.Logger) *NotificationsClient {
	return &NotificationsClient{
		Client: *NewClient(ctx, config, log),
	}
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
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

func (c *Client) handleErrorStatusCode(response *http.Response) string {
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Sprintf("server returned %d status code, response body is unreadable", response.StatusCode)
	}

	return fmt.Sprintf("server returned %d status code, body: %s", response.StatusCode, string(body))
}

func (c *NotificationsClient) postNotification(notification notifications.Notification) error {
	requestBody, err := json.Marshal(notification)
	if err != nil {
		return errors.Wrap(err, "while marshaling payload request")
	}
	request, err := http.NewRequest(http.MethodPost, fmt.Sprintf(notificationServicePath, c.config.ServiceURL), bytes.NewReader(requestBody))
	if err != nil {
		return fmt.Errorf("while creating request: %w", err)
	}
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Accept", "application/json")
	request.Header.Add("DataServiceVersion", "2.0")

	c.log.Info(fmt.Sprintf("Request: %s, Headers: %v, Body %s", request.URL.String(), request.Header, requestBody))

	response, err := c.httpClient.Do(request)

	if err != nil {
		return errors.Wrapf(err, "while executing POST request on: %s", request.URL.String())
	}

	if response.StatusCode != http.StatusCreated {
		return errors.Wrap(err, c.handleErrorStatusCode(response))
	}

	return nil
}

func (c *EventsClient) postEvent(notification events.ResourceEvent) error {
	requestBody, err := json.Marshal(notification)
	if err != nil {
		return errors.Wrap(err, "while marshaling payload request")
	}
	request, err := http.NewRequest(http.MethodPost, fmt.Sprintf(eventsServicePath, c.config.ServiceURL), bytes.NewReader(requestBody))
	if err != nil {
		return fmt.Errorf("while creating request: %w", err)
	}
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Accept", "application/json")
	request.Header.Add("DataServiceVersion", "2.0")

	c.log.Info(fmt.Sprintf("Request: %s, Headers: %v, Body %s", request.URL.String(), request.Header, requestBody))

	response, err := c.httpClient.Do(request)

	if err != nil {
		return errors.Wrapf(err, "while executing POST request on: %s", request.URL.String())
	}

	if response.StatusCode != http.StatusCreated {
		return errors.Wrap(err, c.handleErrorStatusCode(response))
	}

	return nil
}
