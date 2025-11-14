package internal

import (
	"reflect"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
)

type ProvisioningParameters struct {
	PlanID     string                        `json:"plan_id"`
	ServiceID  string                        `json:"service_id"`
	ErsContext ERSContext                    `json:"ers_context"`
	Parameters pkg.ProvisioningParametersDTO `json:"parameters"`

	// PlatformRegion defines the Platform region sent in the request path.
	PlatformRegion string `json:"platform_region"`

	PlatformProvider pkg.CloudProvider `json:"platform_provider"`
}

func (p ProvisioningParameters) IsEqual(input ProvisioningParameters) bool {
	if p.PlanID != input.PlanID {
		return false
	}
	if p.ServiceID != input.ServiceID {
		return false
	}
	if p.PlatformRegion != input.PlatformRegion {
		return false
	}

	if !reflect.DeepEqual(p.ErsContext, input.ErsContext) {
		return false
	}

	p.Parameters.TargetSecret = nil

	if !reflect.DeepEqual(p.Parameters, input.Parameters) {
		return false
	}

	return true
}

type UpdatingParametersDTO struct {
	pkg.AutoScalerParameters `json:",inline"`

	Name                      *string                        `json:"name,omitempty"`
	OIDC                      *pkg.OIDCConnectDTO            `json:"oidc,omitempty"`
	RuntimeAdministrators     []string                       `json:"administrators,omitempty"`
	MachineType               *string                        `json:"machineType,omitempty"`
	AdditionalWorkerNodePools []pkg.AdditionalWorkerNodePool `json:"additionalWorkerNodePools"`
	IngressFiltering          *bool                          `json:"ingressFiltering,omitempty"`
}

func (u UpdatingParametersDTO) UpdateAutoScaler(p *pkg.ProvisioningParametersDTO) bool {
	updated := false
	if u.AutoScalerMin != nil {
		updated = true
		p.AutoScalerMin = u.AutoScalerMin
	}
	if u.AutoScalerMax != nil {
		updated = true
		p.AutoScalerMax = u.AutoScalerMax
	}
	if u.MaxSurge != nil {
		updated = true
		p.MaxSurge = u.MaxSurge
	}
	if u.MaxUnavailable != nil {
		updated = true
		p.MaxUnavailable = u.MaxUnavailable
	}
	return updated
}

type ERSContext struct {
	TenantID              string                             `json:"tenant_id,omitempty"`
	SubAccountID          string                             `json:"subaccount_id"`
	GlobalAccountID       string                             `json:"globalaccount_id"`
	SMOperatorCredentials *ServiceManagerOperatorCredentials `json:"sm_operator_credentials,omitempty"`
	Active                *bool                              `json:"active,omitempty"`
	UserID                string                             `json:"user_id"`
	CommercialModel       *string                            `json:"commercial_model,omitempty"`
	LicenseType           *string                            `json:"license_type,omitempty"`
	Origin                *string                            `json:"origin,omitempty"`
	Platform              *string                            `json:"platform,omitempty"`
	Region                *string                            `json:"region,omitempty"`
}

func InheritMissingERSContext(currentOperation, previousOperation ERSContext) ERSContext {
	if currentOperation.SMOperatorCredentials == nil {
		currentOperation.SMOperatorCredentials = previousOperation.SMOperatorCredentials
	}
	if currentOperation.CommercialModel == nil {
		currentOperation.CommercialModel = previousOperation.CommercialModel
	}
	if currentOperation.LicenseType == nil {
		currentOperation.LicenseType = previousOperation.LicenseType
	}
	if currentOperation.Origin == nil {
		currentOperation.Origin = previousOperation.Origin
	}
	if currentOperation.Platform == nil {
		currentOperation.Platform = previousOperation.Platform
	}
	if currentOperation.Region == nil {
		currentOperation.Region = previousOperation.Region
	}
	return currentOperation
}

func UpdateInstanceERSContext(instance, operation ERSContext) ERSContext {
	if operation.SMOperatorCredentials != nil {
		instance.SMOperatorCredentials = operation.SMOperatorCredentials
	}
	if operation.CommercialModel != nil {
		instance.CommercialModel = operation.CommercialModel
	}
	if operation.LicenseType != nil {
		instance.LicenseType = operation.LicenseType
	}
	if operation.Origin != nil {
		instance.Origin = operation.Origin
	}
	if operation.Platform != nil {
		instance.Platform = operation.Platform
	}
	if operation.Region != nil {
		instance.Region = operation.Region
	}
	return instance
}

func (e ERSContext) ExternalLicenseType() *bool {
	external := false
	if e.LicenseType == nil {
		return &external
	}
	switch *e.LicenseType {
	case "CUSTOMER", "PARTNER", "TRIAL":
		external = true
		return &external
	}
	return &external
}

func (e ERSContext) ERSUpdate() bool {
	if e.SMOperatorCredentials != nil {
		return true
	}
	if e.CommercialModel != nil {
		return true
	}
	if e.LicenseType != nil {
		return true
	}
	if e.Origin != nil {
		return true
	}
	if e.Platform != nil {
		return true
	}
	if e.Region != nil {
		return true
	}
	return false
}

type ServiceManagerEntryDTO struct {
	Credentials ServiceManagerCredentials `json:"credentials"`
	URL         string                    `json:"url"`
}

type ServiceManagerCredentials struct {
	BasicAuth ServiceManagerBasicAuth `json:"basic"`
}

type ServiceManagerBasicAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type ServiceManagerOperatorCredentials struct {
	ClientID          string `json:"clientid"`
	ClientSecret      string `json:"clientsecret"`
	ServiceManagerURL string `json:"sm_url"`
	URL               string `json:"url"`
	XSAppName         string `json:"xsappname"`
}

var (
	Fast    pkg.Channel = ptr.String("fast")
	Regular pkg.Channel = ptr.String("regular")
)

var (
	Ignore          pkg.CustomResourcePolicy = ptr.String("Ignore")
	CreateAndDelete pkg.CustomResourcePolicy = ptr.String("CreateAndDelete")
)
