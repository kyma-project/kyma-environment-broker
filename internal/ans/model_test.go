package ans

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_CreateMinimalNotification(t *testing.T) {
	recipient := NewRecipient("recipient1")
	notification := NewNotification("testType", []Recipient{*recipient})

	//marshal the notification to JSON
	notificationJSON, err := json.Marshal(notification)
	if err != nil {
		t.Fatalf("Failed to marshal notification: %v", err)
	}
	assert.Equal(t, "{\"NotificationTypeKey\":\"testType\",\"Recipients\":[{\"RecipientId\":\"recipient1\"}]}", string(notificationJSON))
}
