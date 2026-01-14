package provisioning

import (
	"github.com/kyma-project/kyma-environment-broker/internal"
)

func WhenBTPOperatorCredentialsProvided(op internal.Operation) bool {
	return op.ProvisioningParameters.ErsContext.SMOperatorCredentials != nil
}
