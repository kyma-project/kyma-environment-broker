package notifications

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO use JSONEq instead of string comparison
// TODO verify timestamp type and default value - see Java client

func Test_CreateMinimalNotification(t *testing.T) {
	recipient, err := NewRecipient("recipient1")
	require.NoError(t, err)
	notification, err := NewNotification("testType", []Recipient{*recipient})
	require.NoError(t, err)
	notificationJSON, err := json.Marshal(notification)
	require.NoError(t, err)
	assert.Equal(t, "{\"NotificationTypeKey\":\"testType\",\"Recipients\":[{\"RecipientId\":\"recipient1\"}]}", string(notificationJSON))
}

func Test_CreateNotificationWithEmptyRecipients(t *testing.T) {
	_, err := NewNotification("testType", []Recipient{})
	require.ErrorContains(t, err, "invalid notification:  recipients must not be empty")
}

func Test_CreateNotificationWithInvalidType(t *testing.T) {
	recipient, err := NewRecipient("recipient1")
	require.NoError(t, err)
	_, err = NewNotification("", []Recipient{*recipient})
	require.ErrorContains(t, err, "notification type key must not be empty")
}

func Test_CreateNotificationsWithTwoRecipients(t *testing.T) {
	recipient1, err := NewRecipient("recipient1")
	require.NoError(t, err)
	recipient2, err := NewRecipient("recipient2")
	require.NoError(t, err)
	notification, err := NewNotification("testType", []Recipient{*recipient1, *recipient2})
	require.NoError(t, err)
	notificationJSON, err := json.Marshal(notification)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"NotificationTypeKey": "testType",
		"Recipients": [
			{"RecipientId": "recipient1"},
			{"RecipientId": "recipient2"}
		]
	}`, string(notificationJSON))
}

func Test_CreateRecipientWithAllOptions(t *testing.T) {
	recipient, err := NewRecipient("recipient1",
		WithGlobalUserID("globalUser1"),
		WithIasGroupId("group1"),
		WithIasHost("test.sap.com"),
		WithRoleName("admin"),
		WithLanguage("EN"),
		WithProviderRecipientID("recipient1"),
		WithTenantID("tenant1"),
		WithXsuaaLevel(XsuaaLevelSubaccount),
	)
	require.NoError(t, err)
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
	_, err := NewRecipient("recipient1", WithXsuaaLevel("invalid"))
	require.ErrorContains(t, err, "invalid XSUAA level: invalid")
}

func Test_CreatePropertyWithAllOptions(t *testing.T) {
	property, err := NewProperty("key1", "value1",
		WithType(PropertyTypeString),
		WithIsSensitive(true),
		WithPropertyLanguage("EN"),
	)
	require.NoError(t, err)
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
	_, err := NewProperty("key1", "value1", WithType("invalid"))
	require.ErrorContains(t, err, "invalid property type: invalid")
}

func Test_CreateNotificationWithProperties(t *testing.T) {
	recipient, err := NewRecipient("recipient1")
	require.NoError(t, err)
	property, err := NewProperty("key1", "value1",
		WithType(PropertyTypeString),
		WithIsSensitive(true),
		WithPropertyLanguage("EN"),
	)
	require.NoError(t, err)
	attachment := NewAttachment(NewHeaders("application/json", "inline;filename=somefile.ext", "123"), NewContent(NewExternal("path/to/file.json")))
	targetParameter := NewTargetParameter("targetKey", "targetValue")
	notification, err := NewNotification("testType", []Recipient{*recipient},
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
	require.NoError(t, err)
	notificationJSON, err := json.Marshal(notification)
	require.NoError(t, err)
	assert.Equal(t, "{\"id\":\"notificationID\",\"NotificationTypeKey\":\"testType\",\"NotificationTypeId\":\"notificationTypeID\",\"NotificationTypeVersion\":\"1\",\"NotificationTypeTimestamp\":\"2023-10-01T00:00:00Z\",\"NotificationTemplateKey\":\"templateKey\",\"Priority\":\"HIGH\",\"ProviderId\":\"providerID\",\"Recipients\":[{\"RecipientId\":\"recipient1\"}],\"Properties\":[{\"Language\":\"EN\",\"Key\":\"key1\",\"Value\":\"value1\",\"Type\":\"String\",\"IsSensitive\":true}],\"TargetParameters\":[{\"Key\":\"targetKey\",\"Value\":\"targetValue\"}],\"Attachments\":[{\"Headers\":{\"ContentType\":\"application/json\",\"ContentDisposition\":\"inline;filename=somefile.ext\",\"ContentId\":\"123\"},\"Content\":{\"External\":{\"Path\":\"path/to/file.json\"}}}]}", string(notificationJSON))
}
