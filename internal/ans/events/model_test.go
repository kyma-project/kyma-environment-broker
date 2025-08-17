package events

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_XsuaaRecipientWithMandatoryFields(t *testing.T) {
	recipient := NewXsuaaRecipient(LevelSubaccount, "recipient1", []RoleName{})
	require.NoError(t, recipient.Validate())
	recipientJSON, err := json.Marshal(recipient)
	require.NoError(t, err)
	assert.JSONEq(t, `{
	"level": "SUBACCOUNT",
	"tenantId": "recipient1"
}`, string(recipientJSON))
}

func Test_Validations(t *testing.T) {
	t.Run("XsuaaRecipient with invalid level",
		func(t *testing.T) {
			require.ErrorContains(t, NewXsuaaRecipient("invalidLevel", "recipient1", []RoleName{}).Validate(), "invalid XSUAA level: invalid level: invalidLevel")
		})
	t.Run("XsuaaRecipient with empty role names", func(t *testing.T) {
		require.ErrorContains(t, NewXsuaaRecipient(LevelSubaccount, "recipient1", []RoleName{""}).Validate(), "role names must not be empty")
	})
	t.Run("XsuaaRecipient with empty tenant ID", func(t *testing.T) {
		require.ErrorContains(t, NewXsuaaRecipient(LevelSubaccount, "", []RoleName{"role1"}).Validate(), "tenant ID must not be empty")
	})
	t.Run("XsuaaRecipient with empty level", func(t *testing.T) {
		require.ErrorContains(t, NewXsuaaRecipient("", "recipient1", []RoleName{"role1"}).Validate(), "invalid XSUAA level: invalid level: ")
	})
	t.Run("UserRecipient with empty email", func(t *testing.T) {
		require.ErrorContains(t, NewUserRecipient("", "test.sap.com").Validate(), "email must not be empty")
	})
	t.Run("UserRecipient with empty IAS host", func(t *testing.T) {
		require.ErrorContains(t, NewUserRecipient("user1", "").Validate(), "IAS host must not be empty")
	})
}

func Test_ResourceWithMandatoryFieldsOnly(t *testing.T) {
	resource := NewResource("resource1", "resource", "subaccount", "resourceGroup")
	require.NoError(t, resource.Validate())
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
	navigation := NewNavigation("navigation1", "navigation", map[string]string{})
	require.NoError(t, navigation.Validate())
	navigationJSON, err := json.Marshal(navigation)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"action": "navigation1",
		"object": "navigation"
	}`, string(navigationJSON))
}

func Test_NavigationWithParameters(t *testing.T) {
	parameters := map[string]string{"param1": "value1", "param2": "value2"}
	navigation := NewNavigation("navigation1", "navigation", parameters)
	require.NoError(t, navigation.Validate())
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
	userRecipient := NewUserRecipient("user1", "test.sap.com")
	require.NoError(t, userRecipient.Validate())
	secondUserRecipient := NewUserRecipient("user2", "test.sap.com")
	require.NoError(t, secondUserRecipient.Validate())
	xsuaaRecipient := NewXsuaaRecipient(LevelSubaccount, "recipient1", []RoleName{"role1", "role2"})
	require.NoError(t, xsuaaRecipient.Validate())
	secondXsuaaRecipient := NewXsuaaRecipient(LevelSubaccount, "recipient2", []RoleName{"role3"})
	require.NoError(t, secondXsuaaRecipient.Validate())
	users := []UserRecipient{*userRecipient, *secondUserRecipient}
	xsuaa := []XsuaaRecipient{*xsuaaRecipient, *secondXsuaaRecipient}
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
		{"No recipients", []UserRecipient{}, []XsuaaRecipient{}, "", "recipients must have at least one XSUAA user or user"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			recipients := NewRecipients(testCase.xsuaa, testCase.users)
			err := recipients.Validate()
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
	userRecipient := NewUserRecipient("user1", "test.sap.com")
	xsuaaRecipient := NewXsuaaRecipient(LevelSubaccount, "recipient1", []RoleName{"role1"})
	recipients := NewRecipients([]XsuaaRecipient{*xsuaaRecipient}, []UserRecipient{*userRecipient})

	// Test minimal NotificationMapping
	nm := NewNotificationMapping("typeKey", *recipients)
	require.NoError(t, nm.Validate())
	nmJSON, err := json.Marshal(nm)
	require.NoError(t, err)
	assert.Contains(t, string(nmJSON), "typeKey")
	assert.Contains(t, string(nmJSON), "recipient1")
	assert.Contains(t, string(nmJSON), "user1")

	// Test with options
	nm2 := NewNotificationMapping("typeKey2", *recipients, WithDeduplicationID("dedupID"), WithNotificationTemplateKey("templateKey"))
	require.NoError(t, nm.Validate())
	assert.Equal(t, "dedupID", nm2.DeduplicationID)
	assert.Equal(t, "templateKey", nm2.NotificationTemplateKey)

	// Test error: empty notificationTypeKey
	nm = NewNotificationMapping("", *recipients)
	require.ErrorContains(t, nm.Validate(), "notification type key must not be empty")

	// Test error: invalid recipients
	emptyRecipients := Recipients{}
	nm = NewNotificationMapping("typeKey", emptyRecipients)
	require.ErrorContains(t, nm.Validate(), "invalid recipients")
}

func Test_NewResourceEvent(t *testing.T) {
	userRecipient := NewUserRecipient("user1", "test.sap.com")
	require.NoError(t, userRecipient.Validate())
	xsuaaRecipient := NewXsuaaRecipient(LevelSubaccount, "recipient1", []RoleName{"role1"})
	require.NoError(t, xsuaaRecipient.Validate())
	recipients := NewRecipients([]XsuaaRecipient{*xsuaaRecipient}, []UserRecipient{*userRecipient})
	require.NoError(t, recipients.Validate())

	navigation := NewNavigation("action", "object", map[string]string{"param1": "value1"})
	require.NoError(t, navigation.Validate())

	notificationMapping := NewNotificationMapping("typeKey", *recipients, WithNotificationTemplateKey("templateKey"))
	require.NoError(t, notificationMapping.Validate())
	notificationMapping.Navigation = navigation

	resource := NewResource("resourceType", "resourceName", "subAccount", "resourceGroup")
	require.NoError(t, resource.Validate())

	resourceEvent, err := NewResourceEvent(
		"eventType",
		"body",
		"subject",
		resource,
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
