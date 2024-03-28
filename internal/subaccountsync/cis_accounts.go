package subaccountsync

import (
	"encoding/json"
	"fmt"
	"io"
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
			c.log.Error(fmt.Sprintf("error while fetching subaccount data: %s", err))
		} else {
			c.log.Info(fmt.Sprintf("fetching for subaccount %s", subaccount))
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

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			c.log.Warn(fmt.Sprintf("failed to close response body: %s", err.Error()))
		}
	}(response.Body)

	if response.StatusCode == http.StatusNotFound {
		var cisAccountError CisAccountErrorResponseType
		err = json.NewDecoder(response.Body).Decode(&cisAccountError)
		if err != nil {
			return CisStateType{}, fmt.Errorf("while decoding CIS account error response: %w", err)
		}
		return CisStateType{}, fmt.Errorf("subaccount %s not found, message %s", subaccountID, cisAccountError.Error.Message)
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
