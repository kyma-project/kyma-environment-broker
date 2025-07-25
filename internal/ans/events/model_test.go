package events

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_XsuaaRecipientWithMandatoryFields(t *testing.T) {
	recipient, err := NewXsuaaRecipient(LevelSubaccount, "recipient1", []RoleName{})
	require.NoError(t, err)
	recipientJSON, err := json.Marshal(recipient)
	require.NoError(t, err)
	assert.JSONEq(t, `{
	"level": "SUBACCOUNT",
	"tenantId": "recipient1"
}`, string(recipientJSON))
}

func Test_XsuaaRecipientWithInvalidLevel(t *testing.T) {
	_, err := NewXsuaaRecipient("invalidLevel", "recipient1", []RoleName{})
	require.ErrorContains(t, err, "invalid XSUAA level: invalid level: invalidLevel")
}

func Test_XsuaaRecipientWithEmptyRoleName(t *testing.T) {
	_, err := NewXsuaaRecipient(LevelSubaccount, "recipient1", []RoleName{""})
	require.ErrorContains(t, err, "role name must not be empty")
}

func Test_XsuaaRecipientWithEmptyTenantID(t *testing.T) {
	_, err := NewXsuaaRecipient(LevelSubaccount, "", []RoleName{"role1"})
	require.ErrorContains(t, err, "tenant ID must not be empty")
}

func Test_UserRecipientWithMandatoryFields(t *testing.T) {
	recipient, err := NewUserRecipient("user1", "test.sap.com")
	require.NoError(t, err)
	recipientJSON, err := json.Marshal(recipient)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"email": "user1",
		"iasHost": "test.sap.com"
	}`, string(recipientJSON))
}

func Test_UserRecipientWithEmptyUserID(t *testing.T) {
	_, err := NewUserRecipient("", "test.sap.com")
	require.ErrorContains(t, err, "email must not be empty")
}

func Test_UserRecipientWithEmptyIasHost(t *testing.T) {
	_, err := NewUserRecipient("user1", "")
	require.ErrorContains(t, err, "IAS host must not be empty")
}

func Test_ResourceWithMandatoryFields(t *testing.T) {
	resource, err := NewResource("resource1", "resource", "subaccount", "resourceGroup")
	require.NoError(t, err)
	resourceJSON, err := json.Marshal(resource)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"resourceType": "resource1",
		"resourceName": "resource",
		"subAccount": "subaccount",
		"resourceGroup": "resourceGroup"
	}`, string(resourceJSON))
}

func Test_NavigationWithMandatoryFields(t *testing.T) {
	navigation, err := NewNavigation("navigation1", "navigation", map[string]string{})
	require.NoError(t, err)
	navigationJSON, err := json.Marshal(navigation)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"action": "navigation1",
		"object": "navigation"
	}`, string(navigationJSON))
}

func Test_NavigationWithParameters(t *testing.T) {
	parameters := map[string]string{"param1": "value1", "param2": "value2"}
	navigation, err := NewNavigation("navigation1", "navigation", parameters)
	require.NoError(t, err)
	navigationJSON, err := json.Marshal(navigation)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"action": "navigation1",
		"object": "navigation",
		"parameters": {
			"param1": "value1",
			"param2": "value2"
		}
	}`, string(navigationJSON))
}

func Test_Recipients(t *testing.T) {
	userRecipient, err := NewUserRecipient("user1", "test.sap.com")
	require.NoError(t, err)
	secondUserRecipient, err := NewUserRecipient("user2", "test.sap.com")
	require.NoError(t, err)
	xsuaaRecipient, err := NewXsuaaRecipient(LevelSubaccount, "recipient1", []RoleName{"role1", "role2"})
	require.NoError(t, err)
	secondXsuaaRecipient, err := NewXsuaaRecipient(LevelSubaccount, "recipient2", []RoleName{"role3"})
	require.NoError(t, err)
	users := []UserRecipient{*userRecipient, *secondUserRecipient}
	xsuaa := []XsuaaRecipient{*xsuaaRecipient, *secondXsuaaRecipient}
	// create loop over test cases
	tests := []struct {
		name          string
		users         []UserRecipient
		xsuaa         []XsuaaRecipient
		expected      string
		expectedError string
	}{
		{"Both users and XSUAA recipients", users, xsuaa, `{
			"xsuaa": [
				{"level": "SUBACCOUNT", "tenantId": "recipient1", "roleNames": ["role1", "role2"]},
				{"level": "SUBACCOUNT", "tenantId": "recipient2", "roleNames": ["role3"]}
			],
			"users": [
				{"email": "user1", "iasHost": "test.sap.com"},
				{"email": "user2", "iasHost": "test.sap.com"}
			]
		}`, ""},
		{"Only XSUAA recipients", []UserRecipient{}, xsuaa, `{
			"xsuaa": [
				{"level": "SUBACCOUNT", "tenantId": "recipient1", "roleNames": ["role1", "role2"]},
				{"level": "SUBACCOUNT", "tenantId": "recipient2", "roleNames": ["role3"]}
			]
		}`, ""},
		{"Only users", users, []XsuaaRecipient{}, `{
			"users": [
				{"email": "user1", "iasHost": "test.sap.com"},
				{"email": "user2", "iasHost": "test.sap.com"}
			]
		}`, ""},
		{"No recipients", []UserRecipient{}, []XsuaaRecipient{}, "", "invalid recipients: recipients must have at least one XSUAA user or user"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			recipients, err := NewRecipients(testCase.xsuaa, testCase.users)
			if testCase.expectedError != "" {
				require.ErrorContains(t, err, testCase.expectedError)
				return
			} else {
				testCaseJSON, err := json.Marshal(recipients)
				require.NoError(t, err)
				assert.JSONEq(t, testCase.expected, string(testCaseJSON))
			}
		})
	}

}

func Test_NotificationMapping(t *testing.T) {
	userRecipient, err := NewUserRecipient("user1", "test.sap.com")
	require.NoError(t, err)
	xsuaaRecipient, err := NewXsuaaRecipient(LevelSubaccount, "recipient1", []RoleName{"role1"})
	require.NoError(t, err)
	recipients, err := NewRecipients([]XsuaaRecipient{*xsuaaRecipient}, []UserRecipient{*userRecipient})
	require.NoError(t, err)

	// Test minimal NotificationMapping
	nm, err := NewNotificationMapping("typeKey", *recipients)
	require.NoError(t, err)
	nmJSON, err := json.Marshal(nm)
	require.NoError(t, err)
	assert.Contains(t, string(nmJSON), "typeKey")
	assert.Contains(t, string(nmJSON), "recipient1")
	assert.Contains(t, string(nmJSON), "user1")

	// Test with options
	nm2, err := NewNotificationMapping("typeKey2", *recipients, WithDeduplicationID("dedupID"), WithNotificationTemplateKey("templateKey"))
	require.NoError(t, err)
	assert.Equal(t, "dedupID", nm2.DeduplicationID)
	assert.Equal(t, "templateKey", nm2.NotificationTemplateKey)

	// Test error: empty notificationTypeKey
	_, err = NewNotificationMapping("", *recipients)
	require.ErrorContains(t, err, "notification type key must not be empty")

	// Test error: invalid recipients
	emptyRecipients := Recipients{}
	_, err = NewNotificationMapping("typeKey", emptyRecipients)
	require.ErrorContains(t, err, "invalid recipients")
}

func Test_NewResourceEvent(t *testing.T) {
	userRecipient, err := NewUserRecipient("user1", "test.sap.com")
	require.NoError(t, err)
	xsuaaRecipient, err := NewXsuaaRecipient(LevelSubaccount, "recipient1", []RoleName{"role1"})
	require.NoError(t, err)
	recipients, err := NewRecipients([]XsuaaRecipient{*xsuaaRecipient}, []UserRecipient{*userRecipient})
	require.NoError(t, err)

	navigation, err := NewNavigation("action", "object", map[string]string{"param1": "value1"})
	require.NoError(t, err)

	notificationMapping, err := NewNotificationMapping("typeKey", *recipients, WithNotificationTemplateKey("templateKey"))
	require.NoError(t, err)
	notificationMapping.Navigation = *navigation

	resource, err := NewResource("resourceType", "resourceName", "subAccount", "resourceGroup")
	require.NoError(t, err)

	eventTime := int64(1234567890)
	resourceEvent, err := NewResourceEvent(
		"eventType",
		"body",
		"subject",
		*resource,
		&eventTime,
		SeverityInfo,
		CategoryNotification,
		VisibilityOwnerSubAccount,
		*notificationMapping,
	)
	require.NoError(t, err)
	resourceEventJSON, err := json.Marshal(resourceEvent)
	require.NoError(t, err)
	assert.JSONEq(t, `{
  "body" : "body",
  "subject" : "subject",
  "eventType" : "eventType",
  "resource" : {
    "resourceType" : "resourceType",
    "resourceName" : "resourceName",
    "subAccount" : "subAccount",
    "resourceGroup" : "resourceGroup"
  },
  "eventTimeStamp" : 1234567890,
  "severity" : "INFO",
  "category" : "NOTIFICATION",
  "visibility" : "OWNER_SUBACCOUNT",
  "notificationMapping" : {
    "notificationTypeKey" : "typeKey",
    "notificationTemplateKey" : "templateKey",
    "recipients" : {
      "xsuaa" : [ {
        "level" : "SUBACCOUNT",
        "tenantId" : "recipient1",
        "roleNames" : [ "role1" ]
      } ],
      "users" : [ {
        "email" : "user1",
        "iasHost" : "test.sap.com"
      } ]
    },
    "navigation" : {
      "action" : "action",
      "object" : "object",
      "parameters" : {
        "param1" : "value1"
      }
    }
  }
}`, string(resourceEventJSON))
}
