package dbmodel

import (
	"time"
)

type RuntimeStateDTO struct {
	ID string `json:"id"`

	CreatedAt time.Time `json:"created_at"`

	RuntimeID   string `json:"runtimeId"`
	OperationID string `json:"operationId"`

	KymaConfig    string `json:"kymaConfig"`
	ClusterConfig string `json:"clusterConfig"`

	// this field is also available in above configs
	// it is set separately to make fetching easier
	K8SVersion string `json:"k8s_version"`
}
