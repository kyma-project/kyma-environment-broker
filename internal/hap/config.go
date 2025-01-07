package hap

import (
	"github.com/kyma-project/kyma-environment-broker/internal/utils"
)

type Config struct {
	SharedSecretPlans utils.Whitelist `envconfig:"default=trial:*;sap-converged-cloud:*"`
}
