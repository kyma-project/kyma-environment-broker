package ans

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/ans/events"
	"github.com/kyma-project/kyma-environment-broker/internal/ans/notifications"
	"github.com/stretchr/testify/require"
)

var logger = slog.New(slog.NewTextHandler(os.Stderr, nil))

func Test_PostEvent(t *testing.T) {
	t.Skip()
	config := EndpointConfig{
		ClientID:               os.Getenv("E_CLIENT_ID"),
		ClientSecret:           os.Getenv("E_CLIENT_SECRET"),
		AuthURL:                "https://jp-notifications-lxe3vgwv.authentication.stagingaws.hanavlab.ondemand.com/oauth/token",
		ServiceURL:             "https://clm-sl-ans-canary-ans-service-api.cfapps.eu12.hana.ondemand.com",
		RateLimitingInterval:   2 * time.Second,
		MaxRequestsPerInterval: 5,
	}
	client := NewEventsClient(context.Background(), config, logger.With("component", "ANS-eventsClient"))
	require.NotNil(t, client)
	recipient, err := events.NewXsuaaRecipient(events.LevelSubaccount, "2fd47ed4-dd54-40b5-99d8-36c4dc3b8cad", []events.RoleName{"Subaccount admin"})
	require.NoError(t, err)
	recipients, err := events.NewRecipients([]events.XsuaaRecipient{*recipient}, nil)
	require.NoError(t, err)
	notificationMapping, err := events.NewNotificationMapping("POC_WebOnlyType2", *recipients)
	require.NoError(t, err)
	eventTime := int64(1)
	resource, err := events.NewResource("broker",
		"keb",
		"2fd47ed4-dd54-40b5-99d8-36c4dc3b8cad",
		"2fd47ed4-dd54-40b5-99d8-36c4dc3b8cad",
		events.WithResourceGlobalAccount("8cd57dc2-edb2-45e0-af8b-7d881006e516"))
	require.NoError(t, err)
	event, err := events.NewResourceEvent(
		"POC-test-KEB-event",
		"Test body",
		"PoC for KEB",
		*resource,
		&eventTime,
		events.SeverityInfo,
		events.CategoryNotification,
		events.VisibilityOwnerSubAccount,
		*notificationMapping,
	)
	require.NotNil(t, event)
	require.NoError(t, err)
	eventAsJSON, err := json.Marshal(event)
	require.JSONEq(t, "{\"body\":\"Test body\",\"subject\":\"PoC for KEB\",\"eventType\":\"POC-test-KEB-event\",\"resource\":{\"resourceType\":\"broker\",\"resourceName\":\"keb\",\"subAccount\":\"2fd47ed4-dd54-40b5-99d8-36c4dc3b8cad\",\"globalAccount\":\"8cd57dc2-edb2-45e0-af8b-7d881006e516\",\"resourceGroup\":\"2fd47ed4-dd54-40b5-99d8-36c4dc3b8cad\"},\"severity\":\"INFO\",\"category\":\"NOTIFICATION\",\"visibility\":\"OWNER_SUBACCOUNT\",\"notificationMapping\":{\"notificationTypeKey\":\"POC_WebOnlyType2\",\"recipients\":{\"xsuaa\":[{\"level\":\"SUBACCOUNT\",\"tenantId\":\"2fd47ed4-dd54-40b5-99d8-36c4dc3b8cad\",\"roleNames\":[\"Subaccount admin\"]}]}}}", string(eventAsJSON))
	require.JSONEq(t, "{\n  \"eventType\": \"POC-test-KEB-event\",\n  \"body\": \"Test body\",\n  \"subject\": \"PoC for KEB\",\n  \"severity\": \"INFO\",\n  \"visibility\": \"OWNER_SUBACCOUNT\",\n  \"category\": \"NOTIFICATION\",\n  \"resource\": {\n    \"globalAccount\": \"8cd57dc2-edb2-45e0-af8b-7d881006e516\",\n    \"subAccount\": \"2fd47ed4-dd54-40b5-99d8-36c4dc3b8cad\",\n    \"resourceGroup\": \"2fd47ed4-dd54-40b5-99d8-36c4dc3b8cad\",\n    \"resourceType\": \"broker\",\n    \"resourceName\": \"keb\"\n  },\n  \"notificationMapping\": {\n    \"notificationTypeKey\": \"POC_WebOnlyType2\",\n    \"recipients\": {\n      \"xsuaa\":[\n\t{\n          \"tenantId\":\"2fd47ed4-dd54-40b5-99d8-36c4dc3b8cad\",\n          \"level\":\"SUBACCOUNT\",\n          \"roleNames\": [\"Subaccount admin\"]}\n      ]\n    }\n  }\n}", string(eventAsJSON))
	err = client.postEvent(*event)
	require.NoError(t, err)
}

func Test_PostNotifications(t *testing.T) {
	t.Skip()
	config := EndpointConfig{
		ClientID:               os.Getenv("N_CLIENT_ID"),
		ClientSecret:           os.Getenv("N_CLIENT_SECRET"),
		AuthURL:                "https://jp-notifications-lxe3vgwv.authentication.stagingaws.hanavlab.ondemand.com/oauth/token",
		ServiceURL:             "https://clm-sl-ans-canary-ans-service-api.cfapps.eu12.hana.ondemand.com",
		RateLimitingInterval:   2 * time.Second,
		MaxRequestsPerInterval: 5,
	}
	client := NewNotificationsClient(context.Background(), config, logger.With("component", "ANS-notificationsClient"))
	require.NotNil(t, client)
	recipient, err := notifications.NewRecipient("jaroslaw.pieszka@sap.com", notifications.WithIasHost("accounts.sap.com"))
	require.NoError(t, err)
	require.NotNil(t, recipient)
	property, err := notifications.NewProperty("shoot", "c0123456")
	require.NoError(t, err)
	require.NotNil(t, property)
	notification, err := notifications.NewNotification("POC_WebOnlyType",
		[]notifications.Recipient{*recipient},
		notifications.WithProperties([]notifications.Property{*property}))
	require.NoError(t, err)
	require.NotNil(t, notification)
	notificationAsJSON, err := json.Marshal(notification)
	require.JSONEq(t, "{\"NotificationTypeKey\":\"POC_WebOnlyType\",\"Recipients\":[{\"RecipientId\":\"jaroslaw.pieszka@sap.com\",\"IasHost\":\"accounts.sap.com\"}],\"Properties\":[{\"Key\":\"shoot\",\"Value\":\"c0123456\"}]}", string(notificationAsJSON))
	require.JSONEq(t, "{\n  \"NotificationTypeKey\": \"POC_WebOnlyType\",\n  \"Recipients\": [\n    {\n      \"RecipientId\": \"jaroslaw.pieszka@sap.com\",\n      \"IasHost\": \"accounts.sap.com\"\n    }\n  ],\n  \"Properties\": [\n    {\n      \"Key\": \"shoot\",\n      \"Value\": \"c0123456\"\n    }\n  ]\n}", string(notificationAsJSON))
	require.NoError(t, err)
	err = client.postNotification(*notification)
	require.NoError(t, err)
}
