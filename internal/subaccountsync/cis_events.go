package subaccountsync

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
)

func (c *RateLimitedCisClient) buildEventRequest(page int, fromActionTime int64) (*http.Request, error) {
	request, err := http.NewRequest(http.MethodGet, fmt.Sprintf(eventServicePath, c.config.ServiceURL), nil)
	if err != nil {
		return nil, fmt.Errorf("while creating request: %v", err)
	}
	q := request.URL.Query()
	q.Add("eventType", eventType)
	q.Add("pageSize", c.config.PageSize)
	q.Add("pageNum", strconv.Itoa(page))
	q.Add("fromActionTime", strconv.FormatInt(fromActionTime, 10))
	q.Add("sortField", "actionTime")
	q.Add("sortOrder", "ASC")

	request.URL.RawQuery = q.Encode()

	return request, nil
}

func (c *RateLimitedCisClient) FetchEventsWindow(fromActionTime int64) ([]Event, error) {
	var events []Event
	var currentPage int
	var totalPages int
	for {
		cisResponse, err := c.fetchEventsPage(currentPage, fromActionTime)
		if cisResponse.TotalPages > 0 {
			totalPages = cisResponse.TotalPages
		}
		if err != nil {
			c.log.Error(fmt.Sprintf("while getting subaccount events for %d page: %v", currentPage, err))
			c.log.Debug(fmt.Sprintf("event window fetched partially - pages: %d out of %d, fetched events: %d, from epoch: %d", currentPage, totalPages, len(events), fromActionTime))
			return events, err
		}
		events = append(events, cisResponse.Events...)
		currentPage++
		if cisResponse.TotalPages == currentPage {
			break
		}
	}
	c.log.Debug(fmt.Sprintf("Event window fetched - pages: %d, events: %d, from epoch: %d", currentPage, len(events), fromActionTime))
	return events, nil
}

func (c *RateLimitedCisClient) fetchEventsPage(page int, fromActionTime int64) (CisEventsResponse, error) {
	request, err := c.buildEventRequest(page, fromActionTime)
	if err != nil {
		return CisEventsResponse{}, fmt.Errorf("while building request for event service: %v", err)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return CisEventsResponse{}, fmt.Errorf("while executing request to event service: %v", err)
	}
	defer func() {
		err := response.Body.Close()
		if err != nil {
			c.log.Warn(fmt.Sprintf("failed to close response body: %s", err.Error()))
		}
	}()

	if response.StatusCode != http.StatusOK {
		return CisEventsResponse{}, fmt.Errorf("while processing response: %s", c.handleErrorStatusCode(response))
	}

	var cisResponse CisEventsResponse
	err = json.NewDecoder(response.Body).Decode(&cisResponse)
	if err != nil {
		return CisEventsResponse{}, fmt.Errorf("while decoding CIS events response: %v", err)
	}

	return cisResponse, nil
}

func (c *RateLimitedCisClient) getEventsForSubaccounts(fromActionTime int64, logs slog.Logger, subaccountsMap subaccountsSetType) ([]Event, error) {
	rawEvents, err := c.FetchEventsWindow(fromActionTime)
	if err != nil && len(rawEvents) == 0 {
		logs.Error(fmt.Sprintf("while getting events: %s", err))
		return nil, err
	}

	// filter events to get only the ones in subaccounts map
	filteredEventsFromCis := filterEvents(rawEvents, subaccountsMap)
	logs.Info(fmt.Sprintf("Raw events: %d, filtered: %d", len(rawEvents), len(filteredEventsFromCis)))

	return filteredEventsFromCis, err
}

func filterEvents(rawEvents []Event, subaccounts subaccountsSetType) []Event {
	var filteredEvents []Event
	for _, event := range rawEvents {
		if _, ok := subaccounts[subaccountIDType(event.SubaccountID)]; ok {
			filteredEvents = append(filteredEvents, event)
		}
	}
	return filteredEvents
}
