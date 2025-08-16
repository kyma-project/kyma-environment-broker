package notifications

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO verify timestamp type and default value - see Java client

func Test_CreateMinimalNotification(t *testing.T) {
	recipient := NewRecipient("recipient1", "test.iashost.com")
	require.NoError(t, recipient.Validate())
	notification := NewNotification("testType", []Recipient{*recipient})
	require.NoError(t, notification.Validate())
	notificationJSON, err := json.Marshal(notification)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"NotificationTypeKey": "testType",
		"Recipients": [
			{"RecipientId": "recipient1", "IasHost": "test.iashost.com"}
		]
	}`, string(notificationJSON))
}

func Test_CreateNotificationWithEmptyRecipients(t *testing.T) {
	notification := NewNotification("testType", []Recipient{})
	require.ErrorContains(t, notification.Validate(), "recipients must not be empty")
}

func Test_CreateNotificationWithInvalidType(t *testing.T) {
	recipient := NewRecipient("recipient1", "test.iashost.com")
	require.NoError(t, recipient.Validate())
	notification := NewNotification("", []Recipient{*recipient})
	require.ErrorContains(t, notification.Validate(), "notification type key must not be empty")
}

func Test_CreateNotificationsWithTwoRecipients(t *testing.T) {
	recipient1 := NewRecipient("recipient1", "test.iashost.com")
	require.NoError(t, recipient1.Validate())
	recipient2 := NewRecipient("recipient2", "test.iashost.com")
	require.NoError(t, recipient2.Validate())
	notification := NewNotification("testType", []Recipient{*recipient1, *recipient2})
	require.NoError(t, notification.Validate())
	notificationJSON, err := json.Marshal(notification)
	require.NoError(t, err)
	assert.JSONEq(t, `{
  "NotificationTypeKey": "testType",
  "Recipients": [
    {
      "RecipientId": "recipient1",
      "IasHost": "test.iashost.com"
    },
    {
      "RecipientId": "recipient2",
      "IasHost": "test.iashost.com"
    }
  ]
}`, string(notificationJSON))
}

func Test_CreateRecipientWithAllOptions(t *testing.T) {
	recipient := NewRecipient("recipient1", "test.sap.com",
		WithGlobalUserId("globalUser1"),
		WithIasGroupId("group1"),
		WithRoleName("admin"),
		WithLanguage("EN"),
		WithProviderRecipientID("recipient1"),
		WithTenantId("tenant1"),
		WithXsuaaLevel(XsuaaLevelSubaccount),
	)
	require.NoError(t, recipient.Validate())
	recipientJSON, err := json.Marshal(recipient)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"GlobalUserId": "globalUser1",
		"RecipientId": "recipient1",
		"IasHost": "test.sap.com",
		"ProviderRecipientId": "recipient1",
		"IasGroupId": "group1",
		"XsuaaLevel": "SUBACCOUNT",
		"TenantId": "tenant1",
		"RoleName": "admin",
		"Language": "EN"
	}`, string(recipientJSON))
}

func Test_CreateRecipientWithInvalidXsuaaLevel(t *testing.T) {
	recipient := NewRecipient("recipient1", "", WithXsuaaLevel("invalid"))
	err := recipient.Validate()
	require.ErrorContains(t, err, "invalid XSUAA level: invalid")
}

func Test_CreatePropertyWithAllOptions(t *testing.T) {
	property := NewProperty("key1", "value1",
		WithType(PropertyTypeString),
		WithIsSensitive(true),
		WithPropertyLanguage("EN"),
	)
	require.NoError(t, property.Validate())
	propertyJSON, err := json.Marshal(property)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"Language": "EN",
		"Key": "key1",
		"Value": "value1",
		"Type": "String",
		"IsSensitive": true
	}`, string(propertyJSON))
}

func Test_CreatePropertyWithInvalidType(t *testing.T) {
	property := NewProperty("key1", "value1", WithType("invalid"))
	require.ErrorContains(t, property.Validate(), "invalid property type: invalid")
}

func Test_CreateNotificationWithProperties(t *testing.T) {
	recipient := NewRecipient("recipient1", "test.iashost.com")
	require.NoError(t, recipient.Validate())
	property := NewProperty("key1", "value1",
		WithType(PropertyTypeString),
		WithIsSensitive(true),
		WithPropertyLanguage("EN"),
	)
	require.NoError(t, property.Validate())
	attachment := NewAttachment("application/json", "inline;filename=somefile.ext", "123", "path/to/file.json")
	targetParameter := NewTargetParameter("targetKey", "targetValue")
	notification := NewNotification("testType", []Recipient{*recipient},
		WithProperties([]Property{*property}),
		WithAttachments([]Attachment{attachment}),
		WithTargetParameters([]TargetParameter{targetParameter}),
		WithID("notificationID"),
		WithNotificationTypeID("notificationTypeID"),
		WithNotificationTypeVersion("1"),
		WithNotificationTypeTimestamp("2023-10-01T00:00:00Z"),
		WithNotificationTemplateKey("templateKey"),
		WithPriority(PriorityHigh),
		WithProviderID("providerID"),
	)
	require.NoError(t, notification.Validate())
	notificationJSON, err := json.Marshal(notification)
	require.NoError(t, err)
	assert.JSONEq(t, `{
  "id": "notificationID",
  "NotificationTypeKey": "testType",
  "NotificationTypeId": "notificationTypeID",
  "NotificationTypeVersion": "1",
  "NotificationTypeTimestamp": "2023-10-01T00:00:00Z",
  "NotificationTemplateKey": "templateKey",
  "Priority": "HIGH",
  "ProviderId": "providerID",
  "Recipients": [
    {"RecipientId": "recipient1", "IasHost": "test.iashost.com"}
  ],
  "Properties": [
    {"Language": "EN", "Key": "key1", "Value": "value1", "Type": "String", "IsSensitive": true}
  ],
  "TargetParameters": [
    {"Key": "targetKey", "Value": "targetValue"}
  ],
  "Attachments": [
    {
      "Headers": {
        "ContentType": "application/json",
        "ContentDisposition": "inline;filename=somefile.ext",
        "ContentId": "123"
      },
      "Content": {
        "External": {"Path": "path/to/file.json"}
      }
    }
  ]
}`, string(notificationJSON))
}
