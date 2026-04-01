package broker

import (
	"errors"

	"github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal/whitelist"
)

func validateGvisorWhitelist(gvisor bool, globalAccountID string, wl whitelist.Set) error {
	if gvisor && whitelist.IsNotWhitelisted(globalAccountID, wl) {
		return errors.New(GvisorNotAvailableForAccountMsg)
	}
	return nil
}

func gvisorToBool(gvisor *runtime.GvisorDTO) bool {
	return gvisor != nil && gvisor.Enabled
}
