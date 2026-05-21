package cis

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"

	"golang.org/x/oauth2/clientcredentials"
)

const (
	eventServicePathV2 = "%s/events/v2/events/central"
	eventTypeV2        = "Subaccount_Deletion"
	entityTypeV2       = "Subaccount"
	defaultPageSizeV2  = "150"
	defaultSinceV2     = "30D"
)

type ClientV2 struct {
	httpClient *http.Client
	config     Config
	log        *slog.Logger
}

func NewClientV2(ctx context.Context, config Config, log *slog.Logger) *ClientV2 {
	cfg := clientcredentials.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		TokenURL:     config.AuthURL,
	}
	httpClientOAuth := cfg.Client(ctx)

	if config.PageSize == "" {
		config.PageSize = defaultPageSizeV2
	}

	return &ClientV2{
		httpClient: httpClientOAuth,
		config:     config,
		log:        log,
	}
}

// SetHttpClient auxiliary method of testing to get rid of oAuth client wrapper
func (c *ClientV2) SetHttpClient(httpClient *http.Client) {
	c.httpClient = httpClient
}

type subaccountsV2 struct {
	ids  []string
	from time.Time
	to   time.Time
}

func (c *ClientV2) FetchSubaccountsToDelete() ([]string, error) {
	subaccounts := subaccountsV2{}

	err := c.fetchSubaccountsFromDeleteEvents(&subaccounts)
	if err != nil {
		return []string{}, fmt.Errorf("while fetching subaccounts from delete events: %w", err)
	}

	c.log.Info(fmt.Sprintf("CIS v2 returned %d subaccounts to delete. "+
		"The events include a range of time from %s to %s", len(subaccounts.ids), subaccounts.from, subaccounts.to))

	return subaccounts.ids, nil
}

func (c *ClientV2) fetchSubaccountsFromDeleteEvents(subaccs *subaccountsV2) error {
	var cursor string
	var retries int
	for {
		cisResponse, err := c.fetchSubaccountDeleteEventsForCursor(cursor)
		if err != nil {
			if kebError.IsTemporaryError(err) && retries < c.config.MaxRequestRetries {
				time.Sleep(c.config.RateLimitingInterval)
				retries++
				continue
			}
			return fmt.Errorf("while fetching subaccount delete events (cursor %q): %w", cursor, err)
		}
		c.appendSubaccountsFromDeleteEvents(&cisResponse, subaccs)
		retries = 0

		if cisResponse.NextCursor == "" {
			break
		}
		cursor = cisResponse.NextCursor
		time.Sleep(c.config.RequestInterval)
	}

	return nil
}

func (c *ClientV2) fetchSubaccountDeleteEventsForCursor(cursor string) (CisResponseV2, error) {
	request, err := c.buildRequest(cursor)
	if err != nil {
		return CisResponseV2{}, fmt.Errorf("while building request for event service: %w", err)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return CisResponseV2{}, fmt.Errorf("while executing request to event service: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	switch {
	case response.StatusCode == http.StatusTooManyRequests:
		return CisResponseV2{}, kebError.NewTemporaryError("rate limiting: %s", c.handleWrongStatusCode(response))
	case response.StatusCode >= 500:
		return CisResponseV2{}, kebError.NewTemporaryError("server error: %s", c.handleWrongStatusCode(response))
	case response.StatusCode != http.StatusOK:
		return CisResponseV2{}, fmt.Errorf("while processing response: %s", c.handleWrongStatusCode(response))
	}

	var cisResponse CisResponseV2
	err = json.NewDecoder(response.Body).Decode(&cisResponse)
	if err != nil {
		return CisResponseV2{}, fmt.Errorf("while decoding CIS response: %w", err)
	}

	return cisResponse, nil
}

func (c *ClientV2) buildRequest(cursor string) (*http.Request, error) {
	request, err := http.NewRequest(http.MethodGet, fmt.Sprintf(eventServicePathV2, c.config.EventServiceURL), nil)
	if err != nil {
		return nil, fmt.Errorf("while creating request: %w", err)
	}

	q := request.URL.Query()
	if cursor != "" {
		q.Add("cursor", cursor)
	} else {
		q.Add("eventType", eventTypeV2)
		q.Add("entityType", entityTypeV2)
		q.Add("since", defaultSinceV2)
		q.Add("pageSize", c.config.PageSize)
		q.Add("sortField", "creationTime")
		q.Add("sortOrder", "ASC")
	}

	request.URL.RawQuery = q.Encode()

	return request, nil
}

func (c *ClientV2) handleWrongStatusCode(response *http.Response) string {
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Sprintf("server returned %d status code, response body is unreadable", response.StatusCode)
	}

	return fmt.Sprintf("server returned %d status code, body: %s", response.StatusCode, string(body))
}

func (c *ClientV2) appendSubaccountsFromDeleteEvents(cisResp *CisResponseV2, subaccs *subaccountsV2) {
	for _, event := range cisResp.Events {
		if event.Type != eventTypeV2 {
			c.log.Warn(fmt.Sprintf("event type %s is not equal to %s, skip event", event.Type, eventTypeV2))
			continue
		}
		subaccs.ids = append(subaccs.ids, event.SubAccount)

		if subaccs.from.IsZero() {
			subaccs.from = time.Unix(0, event.CreationTime*int64(1000000))
		}
		subaccs.to = time.Unix(0, event.CreationTime*int64(1000000))
	}
}
