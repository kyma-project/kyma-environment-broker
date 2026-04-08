package config

import (
	"fmt"

	"github.com/kyma-project/kyma-environment-broker/internal/whitelist"
)

type GlobalAccountsConfig struct {
	MaxPodsWhitelistedGlobalAccountIds   whitelist.Set
	OpenShellWhitelistedGlobalAccountIds whitelist.Set
}

func (c *GlobalAccountsConfig) String() string {
	return fmt.Sprintf("MaxPodsWhitelistedGlobalAccountIds: %s, OpenShellWhitelistedGlobalAccountIds: %s", c.MaxPodsWhitelistedGlobalAccountIds, c.OpenShellWhitelistedGlobalAccountIds)
}
