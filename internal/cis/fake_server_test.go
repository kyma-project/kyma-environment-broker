package cis

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCisFakeServer(t *testing.T) {
	srv, err := NewFakeServer()
	defer srv.Close()
	require.NoError(t, err)

	client := srv.Client()

	t.Run("should get a subaccount for the given ID", func(t *testing.T) {
		resp, err := client.Get(srv.URL + "/accounts/v1/technical/subaccounts/" + FakeSubaccountID1)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		data := make(map[string]interface{})
		err = json.NewDecoder(resp.Body).Decode(&data)
		require.NoError(t, err)

		assert.Equal(t, FakeSubaccountID1, data["guid"])
	})

	t.Run("should get all events", func(t *testing.T) {
		resp, err := client.Get(srv.URL + "/events/v1/events/central")
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		var eventsData eventsEndpointResponse
		err = json.NewDecoder(resp.Body).Decode(&eventsData)
		require.NoError(t, err)

		assert.Equal(t, 6, eventsData.Total)
		assert.Equal(t, 1, eventsData.TotalPages)
		assert.False(t, eventsData.MorePages)
		assert.Len(t, eventsData.Events, 6)
	})

	t.Run("should get Subaccount_Update events", func(t *testing.T) {
		resp, err := client.Get(srv.URL + "/events/v1/events/central?eventType=Subaccount_Update")
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		var eventsData eventsEndpointResponse
		err = json.NewDecoder(resp.Body).Decode(&eventsData)
		require.NoError(t, err)

		assert.Equal(t, 4, eventsData.Total)
		assert.Equal(t, 1, eventsData.TotalPages)
		assert.False(t, eventsData.MorePages)
		assert.Len(t, eventsData.Events, 4)

		for _, event := range eventsData.Events {
			assert.Equal(t, "Subaccount_Update", event["eventType"])
		}
	})

	t.Run("should get all events from the given action time", func(t *testing.T) {
		var actionTime int64 = 1710760200000
		resp, err := client.Get(srv.URL + "/events/v1/events/central?fromActionTime=" + fmt.Sprintf("%d", actionTime))
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		var eventsData eventsEndpointResponse
		err = json.NewDecoder(resp.Body).Decode(&eventsData)
		require.NoError(t, err)

		assert.Equal(t, 2, eventsData.Total)
		assert.Equal(t, 1, eventsData.TotalPages)
		assert.False(t, eventsData.MorePages)
		assert.Len(t, eventsData.Events, 2)

		for _, event := range eventsData.Events {
			ival, ok := event[actionTimeJSONKey]
			require.True(t, ok)
			actualActionTimeFloat, ok := ival.(float64)
			require.True(t, ok)
			actualActionTimeInUnixMilli := int64(actualActionTimeFloat)
			assert.GreaterOrEqual(t, actualActionTimeInUnixMilli, actionTime)
		}
	})

	t.Run("should get Subaccount_Update events from the given action time", func(t *testing.T) {
		var actionTime int64 = 1710761400000
		resp, err := client.Get(srv.URL + "/events/v1/events/central?eventType=Subaccount_Update&fromActionTime=" + fmt.Sprintf("%d", actionTime))
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		var eventsData eventsEndpointResponse
		err = json.NewDecoder(resp.Body).Decode(&eventsData)
		require.NoError(t, err)

		assert.Equal(t, 1, eventsData.Total)
		assert.Equal(t, 1, eventsData.TotalPages)
		assert.False(t, eventsData.MorePages)
		assert.Len(t, eventsData.Events, 1)

		for _, event := range eventsData.Events {
			assert.Equal(t, "Subaccount_Update", event["eventType"])
			ival, ok := event[actionTimeJSONKey]
			require.True(t, ok)
			actualActionTimeFloat, ok := ival.(float64)
			require.True(t, ok)
			actualActionTimeInUnixMilli := int64(actualActionTimeFloat)
			assert.GreaterOrEqual(t, actualActionTimeInUnixMilli, actionTime)
		}
	})
	t.Run("should get Subaccount_Update and Subaccount_Creation events from the given action time", func(t *testing.T) {
		var actionTime int64 = 1710759300000
		resp, err := client.Get(srv.URL + "/events/v1/events/central?eventType=Subaccount_Update,Subaccount_Creation&fromActionTime=" + fmt.Sprintf("%d", actionTime))
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		var eventsData eventsEndpointResponse
		err = json.NewDecoder(resp.Body).Decode(&eventsData)
		require.NoError(t, err)

		assert.Equal(t, 3, eventsData.Total)
		assert.Equal(t, 1, eventsData.TotalPages)
		assert.False(t, eventsData.MorePages)
		assert.Len(t, eventsData.Events, 3)

		for _, event := range eventsData.Events {
			assert.True(t, event["eventType"] == "Subaccount_Update" || event["eventType"] == "Subaccount_Creation")
			ival, ok := event[actionTimeJSONKey]
			require.True(t, ok)
			actualActionTimeFloat, ok := ival.(float64)
			require.True(t, ok)
			actualActionTimeInUnixMilli := int64(actualActionTimeFloat)
			assert.GreaterOrEqual(t, actualActionTimeInUnixMilli, actionTime)
		}
	})
}
