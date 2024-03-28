package subaccountsync

type (
	EventDetails struct {
		BetaEnabled       bool   `json:"betaEnabled"`
		UsedForProduction string `json:"usedForProduction"`
	}

	Event struct {
		ActionTime   int64  `json:"actionTime"`
		SubaccountID string `json:"entityId"`
		Type         string `json:"eventType"`
		Details      EventDetails
	}

	CisEventsResponse struct {
		Total      int     `json:"total"`
		TotalPages int     `json:"totalPages"`
		PageNum    int     `json:"pageNum"`
		Events     []Event `json:"events"`
	}

	CisStateType struct {
		BetaEnabled       bool   `json:"betaEnabled"`
		UsedForProduction string `json:"usedForProduction"`
		ModifiedDate      int64  `json:"modifiedDate"`
	}

	CisErrorResponseType struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}

	CisAccountErrorResponseType struct {
		Error CisErrorResponseType `json:"error"`
	}
)
