package cis

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"
	"golang.org/x/oauth2/clientcredentials"
)

const (
	eventServicePath = "%s/events/v1/events/central"
	eventType        = "Subaccount_Deletion"
	defaultPageSize  = "150"
)

type Config struct {
	ClientID             string
	ClientSecret         string
	AuthURL              string
	EventServiceURL      string
	PageSize             string        `envconfig:"optional"`
	RequestInterval      time.Duration `envconfig:"default=200ms,optional"`
	RateLimitingInterval time.Duration `envconfig:"default=2s,optional"`
	MaxRequestRetries    int           `envconfig:"default=3,optional"`
}

type Client struct {
	httpClient *http.Client
	config     Config
	log        *slog.Logger
}

func NewClient(ctx context.Context, config Config, log *slog.Logger) *Client {
	cfg := clientcredentials.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		TokenURL:     config.AuthURL,
	}
	httpClientOAuth := cfg.Client(ctx)

	if config.PageSize == "" {
		config.PageSize = defaultPageSize
	}

	return &Client{
		httpClient: httpClientOAuth,
		config:     config,
		log:        log,
	}
}

// SetHttpClient auxiliary method of testing to get rid of oAuth client wrapper
func (c *Client) SetHttpClient(httpClient *http.Client) {
	c.httpClient = httpClient
}

type subaccounts struct {
	total int
	ids   []string
	from  time.Time
	to    time.Time
}

func (c *Client) FetchSubaccountsToDelete() ([]string, error) {
	subaccounts := subaccounts{}

	err := c.fetchSubaccountsFromDeleteEvents(&subaccounts)
	if err != nil {
		return []string{}, fmt.Errorf("while fetching subaccounts from delete events: %w", err)
	}

	c.log.Info(fmt.Sprintf("CIS returned total amount of delete events: %d, client fetched %d subaccounts to delete. "+
		"The events include a range of time from %s to %s", subaccounts.total, len(subaccounts.ids), subaccounts.from, subaccounts.to))

	return subaccounts.ids, nil
}

func (c *Client) fetchSubaccountsFromDeleteEvents(subaccs *subaccounts) error {
	var currentPage, totalPages, retries int
	for currentPage <= totalPages {
		cisResponse, err := c.fetchSubaccountDeleteEventsForGivenPageNum(currentPage)
		if err != nil {
			if kebError.IsTemporaryError(err) && retries < c.config.MaxRequestRetries {
				time.Sleep(c.config.RateLimitingInterval)
				retries++
				continue
			}
			return fmt.Errorf("while fetching subaccount delete events for %d page: %w", currentPage, err)
		}
		totalPages = cisResponse.TotalPages
		subaccs.total = cisResponse.Total
		c.appendSubaccountsFromDeleteEvents(&cisResponse, subaccs)
		retries = 0
		currentPage++
		time.Sleep(c.config.RequestInterval)
	}

	return nil
}

func (c *Client) fetchSubaccountDeleteEventsForGivenPageNum(page int) (CisResponse, error) {
	request, err := c.buildRequest(page)
	if err != nil {
		return CisResponse{}, fmt.Errorf("while building request for event service: %w", err)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return CisResponse{}, fmt.Errorf("while executing request to event service: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	switch {
	case response.StatusCode == http.StatusTooManyRequests:
		return CisResponse{}, kebError.NewTemporaryError("rate limiting: %s", c.handleWrongStatusCode(response))
	case response.StatusCode != http.StatusOK:
		return CisResponse{}, fmt.Errorf("while processing response: %s", c.handleWrongStatusCode(response))
	}

	var cisResponse CisResponse
	err = json.NewDecoder(response.Body).Decode(&cisResponse)
	if err != nil {
		return CisResponse{}, fmt.Errorf("while decoding CIS response: %w", err)
	}

	return cisResponse, nil
}

func (c *Client) buildRequest(page int) (*http.Request, error) {
	request, err := http.NewRequest(http.MethodGet, fmt.Sprintf(eventServicePath, c.config.EventServiceURL), nil)
	if err != nil {
		return nil, fmt.Errorf("while creating request: %w", err)
	}

	q := request.URL.Query()
	q.Add("eventType", eventType)
	q.Add("pageSize", c.config.PageSize)
	q.Add("pageNum", strconv.Itoa(page))
	q.Add("sortField", "creationTime")
	q.Add("sortOrder", "ASC")

	request.URL.RawQuery = q.Encode()

	return request, nil
}

func (c *Client) handleWrongStatusCode(response *http.Response) string {
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Sprintf("server returned %d status code, response body is unreadable", response.StatusCode)
	}

	return fmt.Sprintf("server returned %d status code, body: %s", response.StatusCode, string(body))
}

func (c *Client) appendSubaccountsFromDeleteEvents(cisResp *CisResponse, subaccs *subaccounts) {
	for _, event := range cisResp.Events {
		if event.Type != eventType {
			c.log.Warn(fmt.Sprintf("event type %s is not equal to %s, skip event", event.Type, eventType))
			continue
		}
		subaccs.ids = append(subaccs.ids, event.SubAccount)

		if subaccs.from.IsZero() {
			subaccs.from = time.Unix(0, event.CreationTime*int64(1000000))
		}
		if subaccs.total == len(subaccs.ids) {
			subaccs.to = time.Unix(0, event.CreationTime*int64(1000000))
		}
	}
}
