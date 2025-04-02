package input

import (
	"github.com/kyma-project/kyma-environment-broker/internal"
)

type (
	ConfigurationProvider interface {
		ProvideForGivenPlan(planName string) (*internal.ConfigForPlan, error)
	}
)
