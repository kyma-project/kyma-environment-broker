package subaccountsync

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func TestDurationToSince(t *testing.T) {
	tests := []struct {
		input    time.Duration
		expected string
	}{
		{20 * time.Minute, "1H"},
		{1 * time.Hour, "1H"},
		{90 * time.Minute, "2H"},
		{48 * time.Hour, "48H"},
		{1 * time.Second, "1H"},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.expected, durationToSince(tc.input), "input: %v", tc.input)
	}
}

func TestBuildEventRequest_V2_FirstCall(t *testing.T) {
	c := &RateLimitedCisClient{
		config:               CisEndpointConfig{ServiceURL: "http://example.com", PageSize: "10"},
		eventsServiceVersion: "v2",
		eventsWindowSize:     20 * time.Minute,
		ctx:                  context.Background(),
		RateLimiter:          rate.NewLimiter(rate.Every(time.Millisecond), 1000),
	}
	req, err := c.buildEventRequest(0, 0, "")
	require.NoError(t, err)
	q := req.URL.Query()
	assert.Equal(t, "1H", q.Get("since"))
	assert.Equal(t, "Subaccount", q.Get("entityType"))
	assert.ElementsMatch(t, eventTypes, q["eventType"])
	assert.Equal(t, "10", q.Get("pageSize"))
	assert.Equal(t, "", q.Get("cursor"))
	assert.Equal(t, "", q.Get("pageNum"))
	assert.Equal(t, "", q.Get("fromActionTime"))
	assert.Contains(t, req.URL.Path, "events/v2/events/central")
}

func TestBuildEventRequest_V2_SubsequentCall(t *testing.T) {
	c := &RateLimitedCisClient{
		config:               CisEndpointConfig{ServiceURL: "http://example.com", PageSize: "10"},
		eventsServiceVersion: "v2",
		eventsWindowSize:     20 * time.Minute,
		ctx:                  context.Background(),
		RateLimiter:          rate.NewLimiter(rate.Every(time.Millisecond), 1000),
	}
	req, err := c.buildEventRequest(0, 0, "cursor-abc")
	require.NoError(t, err)
	q := req.URL.Query()
	assert.Equal(t, "cursor-abc", q.Get("cursor"))
	assert.Equal(t, "", q.Get("since"))
	assert.Equal(t, "", q.Get("entityType"))
}

func TestBuildEventRequest_V1(t *testing.T) {
	c := &RateLimitedCisClient{
		config:               CisEndpointConfig{ServiceURL: "http://example.com", PageSize: "10"},
		eventsServiceVersion: "v1",
		ctx:                  context.Background(),
		RateLimiter:          rate.NewLimiter(rate.Every(time.Millisecond), 1000),
	}
	req, err := c.buildEventRequest(2, 1710748500000, "")
	require.NoError(t, err)
	q := req.URL.Query()
	assert.Equal(t, "2", q.Get("pageNum"))
	assert.Equal(t, "1710748500000", q.Get("fromActionTime"))
	assert.Contains(t, req.URL.Path, "events/v1/events/central")
	assert.Equal(t, "", q.Get("since"))
	assert.Equal(t, "", q.Get("cursor"))
}

func TestFetchEventsWindow_V2_CursorPagination(t *testing.T) {
	page1Events := []Event{
		{ActionTime: 1000, SubaccountID: "sa1", Type: "Subaccount_Update"},
	}
	page2Events := []Event{
		{ActionTime: 2000, SubaccountID: "sa2", Type: "Subaccount_Creation"},
	}

	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		var resp CisEventsResponse
		if callCount == 1 {
			assert.Equal(t, "", r.URL.Query().Get("cursor"), "first call must not have cursor")
			assert.NotEmpty(t, r.URL.Query().Get("since"))
			resp = CisEventsResponse{Events: page1Events, NextCursor: "cursor-page2"}
		} else {
			assert.Equal(t, "cursor-page2", r.URL.Query().Get("cursor"))
			resp = CisEventsResponse{Events: page2Events, NextCursor: ""}
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := &RateLimitedCisClient{
		httpClient:           srv.Client(),
		config:               CisEndpointConfig{ServiceURL: srv.URL, PageSize: "150"},
		eventsServiceVersion: "v2",
		eventsWindowSize:     20 * time.Minute,
		log:                  slog.Default(),
		RateLimiter:          rate.NewLimiter(rate.Every(time.Millisecond), 1000),
		ctx:                  context.Background(),
	}

	events, err := c.FetchEventsWindow(0)
	require.NoError(t, err)
	require.Len(t, events, 2)
	assert.Equal(t, "sa1", events[0].SubaccountID)
	assert.Equal(t, "sa2", events[1].SubaccountID)
	assert.Equal(t, 2, callCount)
}
