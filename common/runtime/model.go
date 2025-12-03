package runtime

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"
)

type State string

const (
	// StateSucceeded means that the last operation of the runtime has succeeded.
	StateSucceeded State = "succeeded"
	// StateFailed means that the last operation is one of provision, deprovivion, suspension, unsuspension, which has failed.
	StateFailed State = "failed"
	// StateError means the runtime is in a recoverable error state, due to the last upgrade/update operation has failed.
	StateError State = "error"
	// StateProvisioning means that the runtime provisioning (or unsuspension) is in progress (by the last runtime operation).
	StateProvisioning State = "provisioning"
	// StateDeprovisioning means that the runtime deprovisioning (or suspension) is in progress (by the last runtime operation).
	StateDeprovisioning State = "deprovisioning"
	// StateDeprovisioned means that the runtime deprovisioning has finished removing the instance.
	// In case the instance has already been deleted, KEB will try best effort to reconstruct at least partial information regarding deprovisioned instances from residual operations.
	StateDeprovisioned State = "deprovisioned"
	// StateDeprovisionIncomplete means that the runtime deprovisioning has finished removing the instance but certain steps have not finished and the instance should be requeued for repeated deprovisioning.
	StateDeprovisionIncomplete State = "deprovisionincomplete"
	// StateUpgrading means that kyma upgrade or cluster upgrade operation is in progress.
	StateUpgrading State = "upgrading"
	// StateUpdating means the runtime configuration is being updated (i.e. OIDC is reconfigured).
	StateUpdating State = "updating"
	// StateSuspended means that the trial runtime is suspended (i.e. deprovisioned).
	StateSuspended State = "suspended"
	// AllState is a virtual state only used as query parameter in ListParameters to indicate "include all runtimes, which are excluded by default without state filters".
	AllState State = "all"
)

type RuntimeDTO struct {
	InstanceID                  string                    `json:"instanceID"`
	RuntimeID                   string                    `json:"runtimeID"`
	GlobalAccountID             string                    `json:"globalAccountID"`
	SubscriptionGlobalAccountID string                    `json:"subscriptionGlobalAccountID"`
	SubAccountID                string                    `json:"subAccountID"`
	ProviderRegion              string                    `json:"region"`
	SubAccountRegion            string                    `json:"subAccountRegion"`
	ShootName                   string                    `json:"shootName"`
	ServiceClassID              string                    `json:"serviceClassID"`
	ServiceClassName            string                    `json:"serviceClassName"`
	ServicePlanID               string                    `json:"servicePlanID"`
	ServicePlanName             string                    `json:"servicePlanName"`
	Provider                    string                    `json:"provider"`
	Parameters                  ProvisioningParametersDTO `json:"parameters,omitempty"`
	Status                      RuntimeStatus             `json:"status"`
	UserID                      string                    `json:"userID"`
	RuntimeConfig               *map[string]interface{}   `json:"runtimeConfig,omitempty"`
	Bindings                    []BindingDTO              `json:"bindings,omitempty"`
	BetaEnabled                 string                    `json:"betaEnabled,omitempty"`
	UsedForProduction           string                    `json:"usedForProduction,omitempty"`
	SubscriptionSecretName      *string                   `json:"subscriptionSecretName,omitempty"`
	LicenseType                 *string                   `json:"licenseType,omitempty"`
	CommercialModel             *string                   `json:"commercialModel,omitempty"`
	Actions                     []Action                  `json:"actions,omitempty"`
}

type CloudProvider string

const (
	Azure             CloudProvider = "Azure"
	AWS               CloudProvider = "AWS"
	GCP               CloudProvider = "GCP"
	UnknownProvider   CloudProvider = "unknown"
	SapConvergedCloud CloudProvider = "SapConvergedCloud"
	Alicloud          CloudProvider = "Alicloud"
)

type ProvisioningParametersDTO struct {
	AutoScalerParameters `json:",inline"`

	Name                  string   `json:"name"`
	TargetSecret          *string  `json:"targetSecret,omitempty"`
	MachineType           *string  `json:"machineType,omitempty"`
	Region                *string  `json:"region,omitempty"`
	Purpose               *string  `json:"purpose,omitempty"`
	Zones                 []string `json:"zones,omitempty"`
	RuntimeAdministrators []string `json:"administrators,omitempty"`
	// Provider - used in Trial plan to determine which cloud provider to use during provisioning
	Provider *CloudProvider `json:"provider,omitempty"`

	Kubeconfig  string `json:"kubeconfig,omitempty"`
	ShootName   string `json:"shootName,omitempty"`
	ShootDomain string `json:"shootDomain,omitempty"`

	OIDC                      *OIDCConnectDTO            `json:"oidc,omitempty"`
	Networking                *NetworkingDTO             `json:"networking,omitempty"`
	Modules                   *ModulesDTO                `json:"modules,omitempty"`
	ColocateControlPlane      *bool                      `json:"colocateControlPlane,omitempty"`
	AdditionalWorkerNodePools []AdditionalWorkerNodePool `json:"additionalWorkerNodePools,omitempty"`
	IngressFiltering          *bool                      `json:"ingressFiltering,omitempty"`
}

const HAAutoscalerMinimumValue = 3

type AutoScalerParameters struct {
	AutoScalerMin  *int `json:"autoScalerMin,omitempty"`
	AutoScalerMax  *int `json:"autoScalerMax,omitempty"`
	MaxSurge       *int `json:"maxSurge,omitempty"`
	MaxUnavailable *int `json:"maxUnavailable,omitempty"`
}

func CloudProviderFromString(provider string) CloudProvider {
	p := strings.ToLower(provider)
	switch p {
	case "aws":
		return AWS
	case "azure":
		return Azure
	case "gcp":
		return GCP
	case "sapconvergedcloud", "openstack", "sap-converged-cloud":
		return SapConvergedCloud
	case "alicloud":
		return Alicloud
	default:
		return UnknownProvider
	}
}

// FIXME: this is a makeshift check until the provisioner is capable of returning error messages
// https://github.com/kyma-project/control-plane/issues/946
func (p AutoScalerParameters) Validate(planMin, planMax int) error {
	min, max := planMin, planMax
	if p.AutoScalerMin != nil {
		min = *p.AutoScalerMin
	}
	if p.AutoScalerMax != nil {
		max = *p.AutoScalerMax
	}
	if min > max {
		userMin := fmt.Sprintf("%v", p.AutoScalerMin)
		if p.AutoScalerMin != nil {
			userMin = fmt.Sprintf("%v", *p.AutoScalerMin)
		}
		userMax := fmt.Sprintf("%v", p.AutoScalerMax)
		if p.AutoScalerMax != nil {
			userMax = fmt.Sprintf("%v", *p.AutoScalerMax)
		}
		return fmt.Errorf("AutoScalerMax %v should be larger than AutoScalerMin %v. User provided values min:%v, max:%v; plan defaults min:%v, max:%v", max, min, userMin, userMax, planMin, planMax)
	}
	return nil
}

type OIDCConfigDTO struct {
	ClientID         string   `json:"clientID" yaml:"clientID"`
	GroupsClaim      string   `json:"groupsClaim" yaml:"groupsClaim"`
	GroupsPrefix     string   `json:"groupsPrefix,omitempty" yaml:"groupsPrefix,omitempty"`
	IssuerURL        string   `json:"issuerURL" yaml:"issuerURL"`
	SigningAlgs      []string `json:"signingAlgs" yaml:"signingAlgs"`
	UsernameClaim    string   `json:"usernameClaim" yaml:"usernameClaim"`
	UsernamePrefix   string   `json:"usernamePrefix" yaml:"usernamePrefix"`
	RequiredClaims   []string `json:"requiredClaims,omitempty" yaml:"requiredClaims,omitempty"`
	EncodedJwksArray string   `json:"encodedJwksArray,omitempty" yaml:"encodedJwksArray,omitempty"`
}

const oidcValidSigningAlgs = "RS256,RS384,RS512,ES256,ES384,ES512,PS256,PS384,PS512"

func (o *OIDCConnectDTO) IsProvided() bool {
	return o != nil && (o.OIDCConfigDTO != nil || o.List != nil)
}

func (o *OIDCConfigDTO) IsEmpty() bool {
	return o.ClientID == "" && o.IssuerURL == "" && o.GroupsClaim == "" &&
		o.UsernamePrefix == "" && o.UsernameClaim == "" && len(o.SigningAlgs) == 0 &&
		len(o.RequiredClaims) == 0 && o.GroupsPrefix == "" && o.EncodedJwksArray == ""
}

func (o *OIDCConnectDTO) Validate(instanceOidcConfig *OIDCConnectDTO) error {
	if o.List != nil && o.OIDCConfigDTO != nil {
		return fmt.Errorf("both list and object OIDC cannot be set")
	}

	var errs []string

	if o.OIDCConfigDTO != nil {
		if err := o.validateSingleOIDC(instanceOidcConfig, &errs); err != nil {
			return err
		}
	} else if o.List != nil && len(o.List) > 0 {
		o.validateOIDCList(&errs)
	}

	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, ", "))
	}
	return nil
}

func (o *OIDCConnectDTO) validateSingleOIDC(instanceOidcConfig *OIDCConnectDTO, errs *[]string) error {
	if instanceOidcConfig != nil && instanceOidcConfig.List != nil {
		return fmt.Errorf("an object OIDC cannot be used because the instance OIDC configuration uses a list")
	}
	if o.OIDCConfigDTO.IsEmpty() {
		return nil
	}
	if len(o.OIDCConfigDTO.ClientID) == 0 {
		*errs = append(*errs, "clientID must not be empty")
	}
	if len(o.OIDCConfigDTO.IssuerURL) == 0 {
		*errs = append(*errs, "issuerURL must not be empty")
	} else {
		o.validateIssuerURL(o.OIDCConfigDTO.IssuerURL, nil, errs)
	}
	o.validateSigningAlgs(o.OIDCConfigDTO.SigningAlgs, nil, errs)
	o.validateRequiredClaims(o.OIDCConfigDTO.RequiredClaims, nil, errs)
	if o.OIDCConfigDTO.EncodedJwksArray != "" && o.OIDCConfigDTO.EncodedJwksArray != "-" {
		if _, err := base64.StdEncoding.DecodeString(o.OIDCConfigDTO.EncodedJwksArray); err != nil {
			*errs = append(*errs, "encodedJwksArray must be a valid base64-encoded value or set to '-' to disable it if it was used previously")
		}
	}
	return nil
}

func (o *OIDCConnectDTO) validateOIDCList(errs *[]string) {
	for i, oidc := range o.List {
		if len(oidc.ClientID) == 0 {
			*errs = append(*errs, fmt.Sprintf("clientID must not be empty for OIDC at index %d", i))
		}
		if len(oidc.IssuerURL) == 0 {
			*errs = append(*errs, fmt.Sprintf("issuerURL must not be empty for OIDC at index %d", i))
		} else {
			o.validateIssuerURL(oidc.IssuerURL, &i, errs)
		}
		if oidc.EncodedJwksArray != "" {
			if _, err := base64.StdEncoding.DecodeString(oidc.EncodedJwksArray); err != nil {
				*errs = append(*errs, fmt.Sprintf("encodedJwksArray must be a valid base64 encoded value at index %d", i))
			}
		}
		o.validateSigningAlgs(oidc.SigningAlgs, &i, errs)
		o.validateRequiredClaims(oidc.RequiredClaims, &i, errs)
	}
}

func (o *OIDCConnectDTO) validateIssuerURL(issuerURL string, index *int, errs *[]string) {
	issuer, err := url.Parse(issuerURL)
	if err != nil || (issuer != nil && len(issuer.Host) == 0) {
		if index != nil {
			*errs = append(*errs, fmt.Sprintf("issuerURL must be a valid URL, issuerURL must have https scheme for OIDC at index %d", *index))
		} else {
			*errs = append(*errs, "issuerURL must be a valid URL, issuerURL must have https scheme")
		}
		return
	}
	if issuer.Fragment != "" {
		if index != nil {
			*errs = append(*errs, fmt.Sprintf("issuerURL must not contain a fragment for OIDC at index %d", *index))
		} else {
			*errs = append(*errs, "issuerURL must not contain a fragment")
		}
	}
	if issuer.User != nil {
		if index != nil {
			*errs = append(*errs, fmt.Sprintf("issuerURL must not contain a username or password for OIDC at index %d", *index))
		} else {
			*errs = append(*errs, "issuerURL must not contain a username or password")
		}
	}
	if len(issuer.RawQuery) > 0 {
		if index != nil {
			*errs = append(*errs, fmt.Sprintf("issuerURL must not contain a query for OIDC at index %d", *index))
		} else {
			*errs = append(*errs, "issuerURL must not contain a query")
		}
	}
	if issuer.Scheme != "https" {
		if index != nil {
			*errs = append(*errs, fmt.Sprintf("issuerURL must have https scheme for OIDC at index %d", *index))
		} else {
			*errs = append(*errs, "issuerURL must have https scheme")
		}
	}
}

func (o *OIDCConnectDTO) validateSigningAlgs(signingAlgs []string, index *int, errs *[]string) {
	if len(signingAlgs) != 0 {
		validSigningAlgs := o.validSigningAlgsSet()
		for _, providedAlg := range signingAlgs {
			if !validSigningAlgs[providedAlg] {
				if index != nil {
					*errs = append(*errs, fmt.Sprintf("signingAlgs must contain valid signing algorithm(s) for OIDC at index %d", *index))
				} else {
					*errs = append(*errs, "signingAlgs must contain valid signing algorithm(s)")
				}
				break
			}
		}
	}
}

func (o *OIDCConnectDTO) validateRequiredClaims(requiredClaims []string, index *int, errs *[]string) {
	if len(requiredClaims) != 0 {
		if index == nil && len(requiredClaims) == 1 && requiredClaims[0] == "-" {
			return
		}
		for _, claim := range requiredClaims {
			if !strings.Contains(claim, "=") {
				if index != nil {
					*errs = append(*errs, fmt.Sprintf("requiredClaims must be in claim=value format, invalid claim: %s for OIDC at index %d", claim, *index))
				} else {
					*errs = append(*errs, fmt.Sprintf("requiredClaims must be in claim=value format, invalid claim: %s", claim))
				}
				continue
			}
			parts := strings.SplitN(claim, "=", 2)
			if len(parts[0]) == 0 || len(parts[1]) == 0 {
				if index != nil {
					*errs = append(*errs, fmt.Sprintf("requiredClaims must be in claim=value format, invalid claim: %s for OIDC at index %d", claim, *index))
				} else {
					*errs = append(*errs, fmt.Sprintf("requiredClaims must be in claim=value format, invalid claim: %s", claim))
				}
			}
		}
	}
}

func (o *OIDCConnectDTO) validSigningAlgsSet() map[string]bool {
	algs := strings.Split(oidcValidSigningAlgs, ",")
	signingAlgsSet := make(map[string]bool, len(algs))

	for _, v := range algs {
		signingAlgsSet[v] = true
	}

	return signingAlgsSet
}

type NetworkingDTO struct {
	NodesCidr    string  `json:"nodes,omitempty"`
	PodsCidr     *string `json:"pods,omitempty"`
	ServicesCidr *string `json:"services,omitempty"`
	DualStack    *bool   `json:"dualStack,omitempty"`
}

type BindingDTO struct {
	ID                string    `json:"id"`
	CreatedAt         time.Time `json:"createdAt"`
	ExpirationSeconds int64     `json:"expiresInSeconds"`
	ExpiresAt         time.Time `json:"expiresAt"`
	KubeconfigExists  bool      `json:"kubeconfigExists"`
	CreatedBy         string    `json:"createdBy"`
}

type ActionType string

const (
	PlanUpdateActionType         ActionType = "plan_update"
	SubaccountMovementActionType ActionType = "subaccount_movement"
)

type Action struct {
	ID         string     `json:"ID,omitempty"`
	Type       ActionType `json:"type,omitempty"`
	InstanceID string     `json:"-"`
	Message    string     `json:"message,omitempty"`
	OldValue   string     `json:"oldValue,omitempty"`
	NewValue   string     `json:"newValue,omitempty"`
	CreatedAt  time.Time  `json:"createdAt,omitempty"`
}

type RuntimeStatus struct {
	CreatedAt        time.Time       `json:"createdAt"`
	ModifiedAt       time.Time       `json:"modifiedAt"`
	ExpiredAt        *time.Time      `json:"expiredAt,omitempty"`
	DeletedAt        *time.Time      `json:"deletedAt,omitempty"`
	State            State           `json:"state"`
	Provisioning     *Operation      `json:"provisioning,omitempty"`
	Deprovisioning   *Operation      `json:"deprovisioning,omitempty"`
	UpgradingCluster *OperationsData `json:"upgradingCluster,omitempty"`
	Update           *OperationsData `json:"update,omitempty"`
	Suspension       *OperationsData `json:"suspension,omitempty"`
	Unsuspension     *OperationsData `json:"unsuspension,omitempty"`
}

type OperationType string

const (
	Provision      OperationType = "provision"
	Deprovision    OperationType = "deprovision"
	UpgradeCluster OperationType = "cluster upgrade"
	Update         OperationType = "update"
	Suspension     OperationType = "suspension"
	Unsuspension   OperationType = "unsuspension"
)

type OperationsData struct {
	Data       []Operation `json:"data"`
	TotalCount int         `json:"totalCount"`
	Count      int         `json:"count"`
}

type Operation struct {
	State                        string                    `json:"state"`
	Type                         OperationType             `json:"type,omitempty"`
	Description                  string                    `json:"description"`
	CreatedAt                    time.Time                 `json:"createdAt"`
	UpdatedAt                    time.Time                 `json:"updatedAt"`
	OperationID                  string                    `json:"operationID"`
	FinishedStages               []string                  `json:"finishedStages"`
	ExecutedButNotCompletedSteps []string                  `json:"executedButNotCompletedSteps,omitempty"`
	Parameters                   ProvisioningParametersDTO `json:"parameters,omitempty"`
	Error                        *kebError.LastError       `json:"error,omitempty"`
	UpdatedPlanName              string                    `json:"updatedPlanName,omitempty"`
}

type RuntimesPage struct {
	Data       []RuntimeDTO `json:"data"`
	Count      int          `json:"count"`
	TotalCount int          `json:"totalCount"`
}

const (
	GlobalAccountIDParam = "account"
	SubAccountIDParam    = "subaccount"
	InstanceIDParam      = "instance_id"
	RuntimeIDParam       = "runtime_id"
	RegionParam          = "region"
	ShootParam           = "shoot"
	PlanParam            = "plan"
	StateParam           = "state"
	OperationDetailParam = "op_detail"
	KymaConfigParam      = "kyma_config"
	ClusterConfigParam   = "cluster_config"
	ExpiredParam         = "expired"
	GardenerConfigParam  = "gardener_config"
	RuntimeConfigParam   = "runtime_config"
	BindingsParam        = "bindings"
	WithBindingsParam    = "with_bindings"
	ActionsParam         = "actions"
)

type OperationDetail string

const (
	LastOperation OperationDetail = "last"
	AllOperation  OperationDetail = "all"
)

type ListParameters struct {
	// Page specifies the offset for the runtime results in the total count of matching runtimes
	Page int
	// PageSize specifies the count of matching runtimes returned in a response
	PageSize int
	// OperationDetail specifies whether the server should respond with all operations, or only the last operation. If not set, the server by default sends all operations
	OperationDetail OperationDetail
	// KymaConfig specifies whether kyma configuration details should be included in the response for each runtime
	KymaConfig bool
	// ClusterConfig specifies whether Gardener cluster configuration details should be included in the response for each runtime
	ClusterConfig bool
	// RuntimeResourceConfig specifies whether current Runtime Custom Resource details should be included in the response for each runtime
	RuntimeResourceConfig bool
	// Bindings specifies whether runtime bindings should be included in the response for each runtime
	Bindings bool
	// WithBindings parameter filters runtimes to show only those with bindings
	WithBindings bool
	// GardenerConfig specifies whether current Gardener cluster configuration details from provisioner should be included in the response for each runtime
	GardenerConfig bool
	// GlobalAccountIDs parameter filters runtimes by specified global account IDs
	GlobalAccountIDs []string
	// SubAccountIDs parameter filters runtimes by specified subaccount IDs
	SubAccountIDs []string
	// InstanceIDs parameter filters runtimes by specified instance IDs
	InstanceIDs []string
	// RuntimeIDs parameter filters runtimes by specified instance IDs
	RuntimeIDs []string
	// Regions parameter filters runtimes by specified provider regions
	Regions []string
	// Shoots parameter filters runtimes by specified shoot cluster names
	Shoots []string
	// Plans parameter filters runtimes by specified service plans
	Plans []string
	// States parameter filters runtimes by specified runtime states. See type State for possible values
	States []State
	// Expired parameter filters runtimes to show only expired ones.
	Expired bool
	// Events parameter fetches tracing events per instance
	Events string
	// Actions specifies whether audit logs should be included in the response for each runtime
	Actions bool
}

func (rt RuntimeDTO) LastOperation() Operation {
	op := Operation{}

	if rt.Status.Provisioning != nil {
		op = *rt.Status.Provisioning
		op.Type = Provision
	}
	// Take the first cluster upgrade operation, assuming that Data is sorted by CreatedAt DESC.
	if rt.Status.UpgradingCluster != nil && rt.Status.UpgradingCluster.Count > 0 {
		op = rt.Status.UpgradingCluster.Data[0]
		op.Type = UpgradeCluster
	}

	// Take the first unsuspension operation, assuming that Data is sorted by CreatedAt DESC.
	if rt.Status.Unsuspension != nil && rt.Status.Unsuspension.Count > 0 && rt.Status.Unsuspension.Data[0].CreatedAt.After(op.CreatedAt) {
		op = rt.Status.Unsuspension.Data[0]
		op.Type = Unsuspension
	}

	// Take the first suspension operation, assuming that Data is sorted by CreatedAt DESC.
	if rt.Status.Suspension != nil && rt.Status.Suspension.Count > 0 && rt.Status.Suspension.Data[0].CreatedAt.After(op.CreatedAt) {
		op = rt.Status.Suspension.Data[0]
		op.Type = Suspension
	}

	if rt.Status.Deprovisioning != nil && rt.Status.Deprovisioning.CreatedAt.After(op.CreatedAt) {
		op = *rt.Status.Deprovisioning
		op.Type = Deprovision
	}

	// Take the first update operation, assuming that Data is sorted by CreatedAt DESC.
	if rt.Status.Update != nil && rt.Status.Update.Count > 0 && rt.Status.Update.Data[0].CreatedAt.After(op.CreatedAt) {
		op = rt.Status.Update.Data[0]
		op.Type = Update
	}

	return op
}

type OIDCConnectDTO struct {
	*OIDCConfigDTO
	List []OIDCConfigDTO `json:"list,omitzero" yaml:"list,omitzero"`
}

type Channel *string

type ModulesDTO struct {
	Channel Channel     `json:"channel,omitempty" yaml:"channel,omitempty"`
	Default *bool       `json:"default,omitempty" yaml:"default,omitempty"`
	List    []ModuleDTO `json:"list" yaml:"list"`
}

type CustomResourcePolicy *string

type ModuleDTO struct {
	Name                 string               `json:"name,omitempty" yaml:"name,omitempty"`
	Channel              Channel              `json:"channel,omitempty" yaml:"channel,omitempty"`
	CustomResourcePolicy CustomResourcePolicy `json:"customResourcePolicy,omitempty" yaml:"customResourcePolicy,omitempty"`
}

type AdditionalWorkerNodePool struct {
	Name          string `json:"name"`
	MachineType   string `json:"machineType"`
	HAZones       bool   `json:"haZones"`
	AutoScalerMin int    `json:"autoScalerMin"`
	AutoScalerMax int    `json:"autoScalerMax"`
}

func (a AdditionalWorkerNodePool) Validate() error {
	if a.AutoScalerMin > a.AutoScalerMax {
		return fmt.Errorf("AutoScalerMax value %v should be larger than AutoScalerMin value %v for %s additional worker node pool", a.AutoScalerMax, a.AutoScalerMin, a.Name)
	}
	if a.HAZones && a.AutoScalerMin < HAAutoscalerMinimumValue {
		return fmt.Errorf("AutoScalerMin value %v should be at least %v when HA zones are enabled for %s additional worker node pool", a.AutoScalerMin, HAAutoscalerMinimumValue, a.Name)
	}
	if a.AutoScalerMin < 0 {
		return fmt.Errorf("AutoScalerMin value cannot be lower than 0 for %s additional worker node pool", a.Name)
	}
	return nil
}

func (a AdditionalWorkerNodePool) ValidateHAZonesUnchanged(currentAdditionalWorkerNodePools []AdditionalWorkerNodePool) bool {
	for _, currentAdditionalWorkerNodePool := range currentAdditionalWorkerNodePools {
		if a.Name == currentAdditionalWorkerNodePool.Name {
			if a.HAZones != currentAdditionalWorkerNodePool.HAZones {
				return false
			}
		}
	}
	return true
}

func (a AdditionalWorkerNodePool) ValidateMachineTypeChange(currentAdditionalWorkerNodePools []AdditionalWorkerNodePool, allowedMachines []string) error {
	for _, currentAdditionalWorkerNodePool := range currentAdditionalWorkerNodePools {
		if a.Name == currentAdditionalWorkerNodePool.Name {
			if a.MachineType == currentAdditionalWorkerNodePool.MachineType {
				continue
			}

			// machine type change validation
			// check if the initial machine type is a compute-instensive
			if !slices.Contains(allowedMachines, currentAdditionalWorkerNodePool.MachineType) {
				return fmt.Errorf("You cannot update the %s machine type in the %s additional worker node pool. "+
					"You cannot perform updates from compute-intensive machine types", currentAdditionalWorkerNodePool.MachineType, a.Name)
			}

			// check if the target machine type is a compute-instensive
			if !slices.Contains(allowedMachines, a.MachineType) {
				return fmt.Errorf("You cannot update the machine type in the %s additional worker node pool to %s. "+
					"You cannot perform updates to compute-intensive machine types", a.Name, a.MachineType)
			}

		}
	}
	return nil
}
