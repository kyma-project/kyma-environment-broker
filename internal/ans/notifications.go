package ans

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
)

const notificationServicePath = "%s/odatav2/Notification.svc/Notifications"

func (c *RateLimitedAnsClient) postNotification(notification Notification) error {
	requestBody, err := json.Marshal(notification)
	if err != nil {
		errors.Wrap(err, "while marshaling payload request")
	}
	request, err := http.NewRequest(http.MethodPost, fmt.Sprintf(notificationServicePath, c.config.ServiceURL), bytes.NewReader(requestBody))
	if err != nil {
		fmt.Errorf("while creating request: %w", err)
	}
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Accept", "application/json")
	request.Header.Add("DataServiceVersion", "2.0")

	c.log.Info(fmt.Sprintf("Request: %s, Headers: %v", request.URL.String(), request.Header))

	response, err := c.httpClient.Do(request)

	if err != nil {
		return errors.Wrapf(err, "while executing POST request on: %s", request.URL.String())
	}

	if response.StatusCode != http.StatusCreated {
		return errors.Wrap(err, c.handleErrorStatusCode(response))
	}

	return nil
}
