package ans

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/kyma-project/kyma-environment-broker/internal/ans/notifications"
	"github.com/pkg/errors"
)

const notificationServicePath = "%s/odatav2/Notification.svc/Notifications"

func (c *RateLimitedNotificationClient) postNotification(notification notifications.Notification) error {
	requestBody, err := json.Marshal(notification)
	if err != nil {
		return errors.Wrap(err, "while marshaling payload request")
	}
	request, err := http.NewRequest(http.MethodPost, fmt.Sprintf(notificationServicePath, c.config.ServiceURL), bytes.NewReader(requestBody))
	if err != nil {
		return fmt.Errorf("while creating request: %w", err)
	}
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Accept", "application/json")
	request.Header.Add("DataServiceVersion", "2.0")

	c.log.Info(fmt.Sprintf("Request: %s, Headers: %v, Body %s", request.URL.String(), request.Header, requestBody))

	response, err := c.httpClient.Do(request)

	if err != nil {
		return errors.Wrapf(err, "while executing POST request on: %s", request.URL.String())
	}

	if response.StatusCode != http.StatusCreated {
		return errors.Wrap(err, c.handleErrorStatusCode(response))
	}

	return nil
}
