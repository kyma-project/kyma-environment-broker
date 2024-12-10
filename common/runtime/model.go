package runtime

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"

	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
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
	InstanceID                  string                         `json:"instanceID"`
	RuntimeID                   string                         `json:"runtimeID"`
	GlobalAccountID             string                         `json:"globalAccountID"`
	SubscriptionGlobalAccountID string                         `json:"subscriptionGlobalAccountID"`
	SubAccountID                string                         `json:"subAccountID"`
	ProviderRegion              string                         `json:"region"`
	SubAccountRegion            string                         `json:"subAccountRegion"`
	ShootName                   string                         `json:"shootName"`
	ServiceClassID              string                         `json:"serviceClassID"`
	ServiceClassName            string                         `json:"serviceClassName"`
	ServicePlanID               string                         `json:"servicePlanID"`
	ServicePlanName             string                         `json:"servicePlanName"`
	Provider                    string                         `json:"provider"`
	Parameters                  ProvisioningParametersDTO      `json:"parameters,omitempty"`
	Status                      RuntimeStatus                  `json:"status"`
	UserID                      string                         `json:"userID"`
	KymaConfig                  *gqlschema.KymaConfigInput     `json:"kymaConfig,omitempty"`
	ClusterConfig               *gqlschema.GardenerConfigInput `json:"clusterConfig,omitempty"`
	RuntimeConfig               *map[string]interface{}        `json:"runtimeConfig,omitempty"`
	Bindings                    []BindingDTO                   `json:"bindings,omitempty"`
	BetaEnabled                 string                         `json:"betaEnabled,omitempty"`
	UsedForProduction           string                         `json:"usedForProduction,omitempty"`
	SubscriptionSecretName      *string                        `json:"subscriptionSecretName,omitempty"`
}

type CloudProvider string

const (
	Azure             CloudProvider = "Azure"
	AWS               CloudProvider = "AWS"
	GCP               CloudProvider = "GCP"
	UnknownProvider   CloudProvider = "unknown"
	SapConvergedCloud CloudProvider = "SapConvergedCloud"
)

type ProvisioningParametersDTO struct {
	AutoScalerParameters `json:",inline"`

	Name         string  `json:"name"`
	TargetSecret *string `json:"targetSecret,omitempty"`
	VolumeSizeGb *int    `json:"volumeSizeGb,omitempty"`
	MachineType  *string `json:"machineType,omitempty"`
	Region       *string `json:"region,omitempty"`
	Purpose      *string `json:"purpose,omitempty"`
	// LicenceType - based on this parameter, some options can be enabled/disabled when preparing the input
	// for the provisioner e.g. use default overrides for SKR instead overrides from resource
	// with "provisioning-runtime-override" label when LicenceType is "TestDevelopmentAndDemo"
	LicenceType           *string  `json:"licence_type,omitempty"`
	Zones                 []string `json:"zones,omitempty"`
	RuntimeAdministrators []string `json:"administrators,omitempty"`
	// Provider - used in Trial plan to determine which cloud provider to use during provisioning
	Provider *CloudProvider `json:"provider,omitempty"`

	Kubeconfig  string `json:"kubeconfig,omitempty"`
	ShootName   string `json:"shootName,omitempty"`
	ShootDomain string `json:"shootDomain,omitempty"`

	OIDC                   *OIDCConfigDTO `json:"oidc,omitempty"`
	Networking             *NetworkingDTO `json:"networking,omitempty"`
	Modules                *ModulesDTO    `json:"modules,omitempty"`
	ShootAndSeedSameRegion *bool          `json:"shootAndSeedSameRegion,omitempty"`
}

type AutoScalerParameters struct {
	AutoScalerMin  *int `json:"autoScalerMin,omitempty"`
	AutoScalerMax  *int `json:"autoScalerMax,omitempty"`
	MaxSurge       *int `json:"maxSurge,omitempty"`
	MaxUnavailable *int `json:"maxUnavailable,omitempty"`
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
	ClientID       string   `json:"clientID" yaml:"clientID"`
	GroupsClaim    string   `json:"groupsClaim" yaml:"groupsClaim"`
	IssuerURL      string   `json:"issuerURL" yaml:"issuerURL"`
	SigningAlgs    []string `json:"signingAlgs" yaml:"signingAlgs"`
	UsernameClaim  string   `json:"usernameClaim" yaml:"usernameClaim"`
	UsernamePrefix string   `json:"usernamePrefix" yaml:"usernamePrefix"`
}

const oidcValidSigningAlgs = "RS256,RS384,RS512,ES256,ES384,ES512,PS256,PS384,PS512"

func (o *OIDCConfigDTO) IsProvided() bool {
	if o == nil {
		return false
	}
	if o.ClientID == "" && o.IssuerURL == "" && o.GroupsClaim == "" && o.UsernamePrefix == "" && o.UsernameClaim == "" && len(o.SigningAlgs) == 0 {
		return false
	}
	return true
}

func (o *OIDCConfigDTO) Validate() error {
	errs := make([]string, 0)
	if len(o.ClientID) == 0 {
		errs = append(errs, "clientID must not be empty")
	}
	if len(o.IssuerURL) == 0 {
		errs = append(errs, "issuerURL must not be empty")
	} else {
		issuer, err := url.Parse(o.IssuerURL)
		if err != nil || (issuer != nil && len(issuer.Host) == 0) {
			errs = append(errs, "issuerURL must be a valid URL")
		}
		if issuer != nil && issuer.Scheme != "https" {
			errs = append(errs, "issuerURL must have https scheme")
		}
	}
	if len(o.SigningAlgs) != 0 {
		validSigningAlgs := o.validSigningAlgsSet()
		for _, providedAlg := range o.SigningAlgs {
			if !validSigningAlgs[providedAlg] {
				errs = append(errs, "signingAlgs must contain valid signing algorithm(s)")
				break
			}
		}
	}

	if len(errs) > 0 {
		err := fmt.Errorf(strings.Join(errs, ", "))
		return err
	}
	return nil
}

func (o *OIDCConfigDTO) validSigningAlgsSet() map[string]bool {
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
}

type BindingDTO struct {
	ID                string    `json:"id"`
	CreatedAt         time.Time `json:"createdAt"`
	ExpirationSeconds int64     `json:"expiresInSeconds"`
	ExpiresAt         time.Time `json:"expiresAt"`
	KubeconfigExists  bool      `json:"kubeconfigExists"`
	CreatedBy         string    `json:"createdBy"`
}

type RuntimeStatus struct {
	CreatedAt        time.Time                 `json:"createdAt"`
	ModifiedAt       time.Time                 `json:"modifiedAt"`
	ExpiredAt        *time.Time                `json:"expiredAt,omitempty"`
	DeletedAt        *time.Time                `json:"deletedAt,omitempty"`
	State            State                     `json:"state"`
	Provisioning     *Operation                `json:"provisioning,omitempty"`
	Deprovisioning   *Operation                `json:"deprovisioning,omitempty"`
	UpgradingCluster *OperationsData           `json:"upgradingCluster,omitempty"`
	Update           *OperationsData           `json:"update,omitempty"`
	Suspension       *OperationsData           `json:"suspension,omitempty"`
	Unsuspension     *OperationsData           `json:"unsuspension,omitempty"`
	GardenerConfig   *gqlschema.GardenerConfig `json:"gardenerConfig,omitempty"`
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
	OrchestrationID              string                    `json:"orchestrationID,omitempty"`
	FinishedStages               []string                  `json:"finishedStages"`
	ExecutedButNotCompletedSteps []string                  `json:"executedButNotCompletedSteps,omitempty"`
	Parameters                   ProvisioningParametersDTO `json:"parameters,omitempty"`
	Error                        *kebError.LastError       `json:"error,omitempty"`
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

type ModulesDTO struct {
	Default *bool       `json:"default,omitempty" yaml:"default,omitempty"`
	List    []ModuleDTO `json:"list" yaml:"list"`
}

type Channel *string

type CustomResourcePolicy *string

type ModuleDTO struct {
	Name                 string               `json:"name,omitempty" yaml:"name,omitempty"`
	Channel              Channel              `json:"channel,omitempty" yaml:"channel,omitempty"`
	CustomResourcePolicy CustomResourcePolicy `json:"customResourcePolicy,omitempty" yaml:"customResourcePolicy,omitempty"`
}
