package cis

type Event struct {
	CreationTime int64  `json:"creationTime"`
	SubAccount   string `json:"entityId"`
	Type         string `json:"eventType"`
}

type CisResponse struct {
	Total      int     `json:"total"`
	TotalPages int     `json:"totalPages"`
	PageNum    int     `json:"pageNum"`
	Events     []Event `json:"events"`
}

type CisResponseV2 struct {
	NextCursor string  `json:"nextCursor"`
	Events     []Event `json:"events"`
}
