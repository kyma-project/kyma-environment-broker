package provisioningservice

type AccessToken struct {
	Token string `json:"access_token"`
}

type CreateEnvironment struct {
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

type Environment struct {
	ID    string `json:"id"`
	State State  `json:"state"`
}
