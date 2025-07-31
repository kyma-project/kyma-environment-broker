package ans

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_CreateMinimalNotification(t *testing.T) {
	recipient := NewRecipient("recipient1")
	notification := NewNotification("testType", []Recipient{*recipient})

	notificationJSON, err := json.Marshal(notification)
	require.NoError(t, err)
	assert.Equal(t, "{\"NotificationTypeKey\":\"testType\",\"Recipients\":[{\"RecipientId\":\"recipient1\"}]}", string(notificationJSON))
}

func Test_CreateNotificationWithAllFields(t *testing.T) {
	recipient := NewRecipient("recipient1").
		WithGlobalUserID("globalUser1").
		WithIasHost("iasHost1").
		WithProviderRecipientID("providerRecipient1").
		WithIasGroupID("iasGroup1").
		WithXsuaaLevel("xsuaaLevel1").
		WithTenantID("tenant1").
		WithRoleName("role1").
		WithLanguage("en")

	property := Property{
		Key:   "key1",
		Value: "value1",
	}

	targetParameter := TargetParameter{
		Key:   "paramKey",
		Value: "paramValue",
	}

	notification := NewNotification("testType", []Recipient{*recipient}).
		WithProperties([]Property{property}).
		WithTargetParameters([]TargetParameter{targetParameter})

	notificationJSON, err := json.Marshal(notification)
	require.NoError(t, err)
	assert.Contains(t, string(notificationJSON), "\"NotificationTypeKey\":\"testType\"")
	assert.Contains(t, string(notificationJSON), "\"Recipients\":[{\"RecipientId\":\"recipient1\",\"GlobalUserId\":\"globalUser1\",\"IasHost\":\"iasHost1\",\"ProviderRecipientId\":\"providerRecipient1\",\"IasGroupId\":\"iasGroup1\",\"XsuaaLevel\":\"xsuaaLevel1\",\"TenantId\":\"tenant1\",\"RoleName\":\"role1\",\"Language\":\"en\"}]")
	assert.Contains(t, string(notificationJSON), "\"Properties\":[{\"Key\":\"key1\",\"Value\":\"value1\"}]")
	assert.Contains(t, string(notificationJSON), "\"TargetParameters\":[{\"Key\":\"paramKey\",\"Value\":\"paramValue\"}]")
}
