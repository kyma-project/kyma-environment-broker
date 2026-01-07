package dashboard

import (
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/pivotal-cf/brokerapi/v12/domain"
)

func ProvideURL(instance *internal.Instance, provisioningOperation *internal.ProvisioningOperation) string {
	if provisioningOperation.State == domain.Succeeded {
		return instance.DashboardURL
	}
	return ""
}
