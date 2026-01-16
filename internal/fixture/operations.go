package fixture

import (
	"fmt"
	"time"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/pivotal-cf/brokerapi/v12/domain"
)

type OperationOption func(operation *internal.Operation)

func WithProvisioningParameters(params internal.ProvisioningParameters) OperationOption {
	return func(o *internal.Operation) {
		o.ProvisioningParameters = params
	}
}

func WithProvider(provider string) OperationOption {
	return func(o *internal.Operation) {
		o.CloudProvider = provider
	}
}

func WithPlanID(planID string) OperationOption {
	return func(o *internal.Operation) {
		o.ProvisioningParameters.PlanID = planID
	}
}

func SetTemporary() OperationOption {
	return func(o *internal.Operation) {
		o.Temporary = true
	}
}

func FixProvisioningOperation(operationID, instanceID string, opts ...OperationOption) internal.Operation {

	o := FixOperation(operationID, instanceID, internal.OperationTypeProvision, opts...)

	for _, opt := range opts {
		opt(&o)
	}

	return o
}

func FixOperation(operationID, instanceID string, operationType internal.OperationType, opts ...OperationOption) internal.Operation {
	o := internal.Operation{
		InstanceDetails:        FixInstanceDetails(instanceID),
		ID:                     operationID,
		Type:                   operationType,
		Version:                0,
		CreatedAt:              time.Now(),
		UpdatedAt:              time.Now().Add(time.Hour * 48),
		InstanceID:             instanceID,
		ProvisionerOperationID: "",
		State:                  domain.Succeeded,
		Description:            fmt.Sprintf("Description for operation %s", operationID),
		ProvisioningParameters: FixProvisioningParameters(operationID),
		FinishedStages:         []string{"prepare", "check_provisioning"},
		DashboardURL:           "https://console.kyma.org",
		Temporary:              false,
	}

	for _, opt := range opts {
		opt(&o)
	}
	return o
}

func FixUpdatingOperation(operationId, instanceId string) internal.UpdatingOperation {
	o := FixOperation(operationId, instanceId, internal.OperationTypeUpdate)
	o.UpdatingParameters = internal.UpdatingParametersDTO{
		OIDC: &pkg.OIDCConnectDTO{
			List: []pkg.OIDCConfigDTO{
				{
					ClientID:       "client-id-oidc",
					GroupsClaim:    "groups",
					IssuerURL:      "issuer-url",
					SigningAlgs:    []string{"signingAlgs"},
					UsernameClaim:  "sub",
					UsernamePrefix: "",
				},
			},
		},
	}
	return internal.UpdatingOperation{
		Operation: o,
	}
}

func FixUpdatingOperationWithOIDCObject(operationId, instanceId string) internal.UpdatingOperation {
	o := FixOperation(operationId, instanceId, internal.OperationTypeUpdate)
	o.UpdatingParameters = internal.UpdatingParametersDTO{
		OIDC: &pkg.OIDCConnectDTO{
			OIDCConfigDTO: &pkg.OIDCConfigDTO{
				ClientID:       "client-id-oidc",
				GroupsClaim:    "groups",
				GroupsPrefix:   "-",
				IssuerURL:      "issuer-url",
				SigningAlgs:    []string{"signingAlgs"},
				UsernameClaim:  "sub",
				UsernamePrefix: "",
				RequiredClaims: []string{"claim1=value1", "claim2=value2"},
			},
		},
	}
	return internal.UpdatingOperation{
		Operation: o,
	}
}

func FixDeprovisioningOperation(operationId, instanceId string) internal.DeprovisioningOperation {
	return internal.DeprovisioningOperation{
		Operation: FixDeprovisioningOperationAsOperation(operationId, instanceId),
	}
}

func FixDeprovisioningOperationAsOperation(operationId, instanceId string) internal.Operation {
	return FixOperation(operationId, instanceId, internal.OperationTypeDeprovision)
}

func FixSuspensionOperationAsOperation(operationId, instanceId string) internal.Operation {
	return FixOperation(operationId, instanceId, internal.OperationTypeDeprovision, SetTemporary(), WithPlanID(TrialPlan))
}
