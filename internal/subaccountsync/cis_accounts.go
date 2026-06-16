package subaccountsync

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func (c *RateLimitedCisClient) buildSubaccountRequest(subaccountID string) (*http.Request, error) {
	request, err := http.NewRequest(http.MethodGet, fmt.Sprintf(subaccountServicePath, c.config.ServiceURL, subaccountID), nil)
	if err != nil {
		return nil, fmt.Errorf("while creating request: %w", err)
	}
	q := request.URL.Query()
	request.URL.RawQuery = q.Encode()
	return request, nil
}

func (c *RateLimitedCisClient) GetSubaccountData(subaccountID string) (CisStateType, error) {
	request, err := c.buildSubaccountRequest(subaccountID)
	if err != nil {
		return CisStateType{}, fmt.Errorf("while building request for accounts technical service: %w", err)
	}

	response, err := c.Do(request)
	if err != nil {
		c.incRequest("failure")
		return CisStateType{}, fmt.Errorf("while executing request to accounts technical service: %w", err)
	}
	defer func() {
		err := response.Body.Close()
		if err != nil {
			c.log.Warn(fmt.Sprintf("failed to close response body: %s", err.Error()))
		}
	}()

	if response.StatusCode == http.StatusNotFound {
		c.incRequest("notfound")
		return CisStateType{}, nil
	}

	if response.StatusCode != http.StatusOK {
		c.incRequest("failure")
		return CisStateType{}, fmt.Errorf("while processing response: %s", c.handleErrorStatusCode(response))
	}

	var cisResponse CisStateType
	err = json.NewDecoder(response.Body).Decode(&cisResponse)
	if err != nil {
		c.incRequest("failure")
		return CisStateType{}, fmt.Errorf("while decoding CIS account response: %w", err)
	}

	c.incRequest("success")
	return cisResponse, nil
}
