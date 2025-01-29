package hap

import (
	"github.com/kyma-project/kyma-environment-broker/internal/utils"
)

type Config struct {
	SharedRule utils.Whitelist `envconfig`
	euAccessRule utils.Whitelist `envconfig`
	clusterRegionRule utils.Whitelist `envconfig`
	platformRegionRule utils.Whitelist `envconfig`
}
