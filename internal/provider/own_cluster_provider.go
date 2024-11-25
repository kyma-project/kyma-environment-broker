package provider

import (
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
)

type NoHyperscalerInput struct {
}

func (p *NoHyperscalerInput) Defaults() *gqlschema.ClusterConfigInput {
	return &gqlschema.ClusterConfigInput{
		GardenerConfig: &gqlschema.GardenerConfigInput{},
		Administrators: nil,
	}
}

func (p *NoHyperscalerInput) ApplyParameters(input *gqlschema.ClusterConfigInput, pp internal.ProvisioningParameters) {
}

func (p *NoHyperscalerInput) Profile() gqlschema.KymaProfile {
	return gqlschema.KymaProfileEvaluation
}

func (p *NoHyperscalerInput) Provider() pkg.CloudProvider {
	return pkg.UnknownProvider
}
