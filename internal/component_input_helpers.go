package internal

import (
	"fmt"
	"strings"

	reconcilerApi "github.com/kyma-incubator/reconciler/pkg/keb"
)

const (
	BTPOperatorComponentName = "btp-operator"

	// BTP Operator overrides keys
	BTPOperatorClientID      = "manager.secret.clientid"
	BTPOperatorClientSecret  = "manager.secret.clientsecret"
	BTPOperatorURL           = "manager.secret.url"    // deprecated, for btp-operator v0.2.0
	BTPOperatorSMURL         = "manager.secret.sm_url" // for btp-operator v0.2.3
	BTPOperatorTokenURL      = "manager.secret.tokenurl"
	BTPOperatorClusterID     = "cluster.id"
	BTPOperatorPriorityClass = "manager.priorityClassName"
)

var btpOperatorRequiredKeys = []string{BTPOperatorClientID, BTPOperatorClientSecret, BTPOperatorURL, BTPOperatorSMURL, BTPOperatorTokenURL, BTPOperatorClusterID, BTPOperatorPriorityClass}

type ClusterIDGetter func(string) (string, error)

func CheckBTPCredsValid(clusterConfiguration reconcilerApi.Cluster) error {
	vals := make(map[string]bool)
	hasBTPOperator := false
	var errs []string
	for _, c := range clusterConfiguration.KymaConfig.Components {
		if c.Component == BTPOperatorComponentName {
			hasBTPOperator = true
			for _, cfg := range c.Configuration {
				for _, key := range btpOperatorRequiredKeys {
					if cfg.Key == key {
						vals[key] = true
						if cfg.Value == nil {
							errs = append(errs, fmt.Sprintf("missing required value for %v", key))
						}
						if val, ok := cfg.Value.(string); !ok || val == "" {
							errs = append(errs, fmt.Sprintf("missing required value for %v", key))
						}
					}
				}
			}
		}
	}
	if hasBTPOperator {
		for _, key := range btpOperatorRequiredKeys {
			if !vals[key] {
				errs = append(errs, fmt.Sprintf("missing required key %v", key))
			}
		}
		if len(errs) != 0 {
			return fmt.Errorf("BTP Operator is about to be installed but is missing required configuration: %v", strings.Join(errs, ", "))
		}
	}
	return nil
}

func IsEuAccess(platformRegion string) bool {
	switch platformRegion {
	case "cf-eu11":
		return true
	case "cf-ch20":
		return true
	}
	return false
}
