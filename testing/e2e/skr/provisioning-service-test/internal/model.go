package internal

type AccessToken struct {
	Token string `json:"access_token"`
}

type ErrorResponse struct {
	Error Error `json:"error"`
}

type Error struct {
	Message string `json:"message"`
}

type CreateEnvironmentRequest struct {
	EnvironmentType string                `json:"environmentType"`
	ServiceName     string                `json:"serviceName"`
	PlanName        string                `json:"planName"`
	User            string                `json:"user"`
	Parameters      EnvironmentParameters `json:"parameters"`
}

type EnvironmentParameters struct {
	Name   string `json:"name"`
	Region string `json:"region"`
}

type CreatedEnvironmentResponse struct {
	ID string `json:"id"`
}

type State string

const (
	CREATING        State = "CREATING"
	UPDATING        State = "UPDATING"
	DELETING        State = "DELETING"
	OK              State = "OK"
	CREATION_FAILED State = "CREATION_FAILED"
	DELETION_FAILED State = "DELETION_FAILED"
	UPDATE_FAILED   State = "UPDATE_FAILED"
)

type EnvironmentType string

const (
	KYMA EnvironmentType = "kyma"
)

type EnvironmentsResponse struct {
	Environments []EnvironmentResponse `json:"environmentInstances"`
}

type EnvironmentResponse struct {
	ID              string          `json:"id"`
	State           State           `json:"state"`
	EnvironmentType EnvironmentType `json:"environmentType"`
}

type CreateBindingRequest struct {
	ServiceInstanceID string `json:"serviceInstanceId"`
	PlanID            string `json:"planId"`
}

type CreatedBindingResponse struct {
	ID          string
	Credentials Credentials `json:"credentials"`
}

type GetBindingResponse struct {
	Credentials Credentials `json:"credentials"`
}

type Credentials struct {
	Kubeconfig string `json:"kubeconfig"`
}
