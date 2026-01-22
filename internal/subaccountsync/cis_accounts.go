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

func (c *RateLimitedCisClient) fetchDataForSetOfSubaccounts(subaccounts subaccountsSetType) (statesFromCisType, error) {

	subaccountsDataFromAccounts := make(statesFromCisType)
	for subaccount := range subaccounts {
		accountData, err := c.GetSubaccountData(string(subaccount))
		if err != nil {
			c.log.Error(fmt.Sprintf("while getting subaccount data: %s", err))
		} else {
			c.log.Debug(fmt.Sprintf("getting for subaccount %s", subaccount))
			subaccountsDataFromAccounts[subaccount] = accountData
		}
	}
	return subaccountsDataFromAccounts, nil
}

func (c *RateLimitedCisClient) GetSubaccountData(subaccountID string) (CisStateType, error) {
	request, err := c.buildSubaccountRequest(subaccountID)
	if err != nil {
		return CisStateType{}, fmt.Errorf("while building request for accounts technical service: %w", err)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return CisStateType{}, fmt.Errorf("while executing request to accounts technical service: %w", err)
	}
	defer func() {
		err := response.Body.Close()
		if err != nil {
			c.log.Warn(fmt.Sprintf("failed to close response body: %s", err.Error()))
		}
	}()

	if response.StatusCode == http.StatusNotFound {
		return CisStateType{}, nil
	}

	if response.StatusCode != http.StatusOK {
		return CisStateType{}, fmt.Errorf("while processing response: %s", c.handleErrorStatusCode(response))
	}

	var cisResponse CisStateType
	err = json.NewDecoder(response.Body).Decode(&cisResponse)
	if err != nil {
		return CisStateType{}, fmt.Errorf("while decoding CIS account response: %w", err)
	}

	return cisResponse, nil
}
