package cis

type Response struct {
	NextCursor string  `json:"nextCursor"`
	Events     []Event `json:"events"`
}

type Event struct {
	CreationTime int64  `json:"creationTime"`
	SubAccount   string `json:"entityId"`
	Type         string `json:"eventType"`
}
