package ans

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
)

const notificationServicePath = "%s/odatav2/Notification.svc/Notifications"

func (c *RateLimitedAnsClient) buildNotificationRequest(notification Notification) (*http.Request, error) {
	requestBody, err := json.Marshal(notification)
	if err != nil {
		return nil, errors.Wrap(err, "while marshaling payload request")
	}

	request, err := http.NewRequest(http.MethodPost, fmt.Sprintf(notificationServicePath, c.config.ServiceURL), bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("while creating request: %w", err)
	}
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Accept", "application/json")
	request.Header.Add("DataServiceVersion", "2.0")
	return request, nil
}

func (c *RateLimitedAnsClient) postNotification(notification Notification) error {

	request, err := c.buildNotificationRequest(notification)
	if err != nil {
		return errors.Wrap(err, "while building notification request")
	}
	response, err := c.httpClient.Do(request)

	if err != nil {
		return errors.Wrapf(err, "while executing request to ANS on: %s", request.URL.String())
	}

	if response.StatusCode != http.StatusCreated {
		return errors.Wrap(err, c.handleErrorStatusCode(response))
	}

	return nil
}
