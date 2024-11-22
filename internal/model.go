package internal

import (
	"database/sql"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/euaccess"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/kyma-environment-broker/common/orchestration"
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"
	"github.com/kyma-project/kyma-environment-broker/internal/events"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	log "github.com/sirupsen/logrus"
)

type ProvisionerInputCreator interface {
	SetProvisioningParameters(params ProvisioningParameters) ProvisionerInputCreator
	SetShootName(string) ProvisionerInputCreator
	SetLabel(key, value string) ProvisionerInputCreator
	CreateProvisionRuntimeInput() (gqlschema.ProvisionRuntimeInput, error)
	CreateUpgradeRuntimeInput() (gqlschema.UpgradeRuntimeInput, error)
	CreateUpgradeShootInput() (gqlschema.UpgradeShootInput, error)
	Provider() pkg.CloudProvider
	Configuration() *ConfigForPlan

	CreateProvisionClusterInput() (gqlschema.ProvisionRuntimeInput, error)
	SetKubeconfig(kcfg string) ProvisionerInputCreator
	SetRuntimeID(runtimeID string) ProvisionerInputCreator
	SetInstanceID(instanceID string) ProvisionerInputCreator
	SetShootDomain(shootDomain string) ProvisionerInputCreator
	SetShootDNSProviders(dnsProviders gardener.DNSProvidersData) ProvisionerInputCreator
	SetClusterName(name string) ProvisionerInputCreator
	SetOIDCLastValues(oidcConfig gqlschema.OIDCConfigInput) ProvisionerInputCreator
}

type EventHub struct {
	Deleted bool `json:"event_hub_deleted"`
}

type Instance struct {
	InstanceID                  string
	RuntimeID                   string
	GlobalAccountID             string
	SubscriptionGlobalAccountID string
	SubAccountID                string
	ServiceID                   string
	ServiceName                 string
	ServicePlanID               string
	ServicePlanName             string

	DashboardURL   string
	Parameters     ProvisioningParameters
	ProviderRegion string

	InstanceDetails InstanceDetails

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt time.Time
	ExpiredAt *time.Time

	Version      int
	Provider     pkg.CloudProvider
	Reconcilable bool
}

func (i *Instance) IsExpired() bool {
	return i.ExpiredAt != nil
}

func (i *Instance) GetSubscriptionGlobalAccoundID() string {
	if i.SubscriptionGlobalAccountID != "" {
		return i.SubscriptionGlobalAccountID
	} else {
		return i.GlobalAccountID
	}
}

func (i *Instance) GetInstanceDetails() (InstanceDetails, error) {
	result := i.InstanceDetails
	//overwrite RuntimeID in InstanceDetails with Instance.RuntimeID
	//needed for runtimes suspended without clearing RuntimeID in deprovisioning operation
	result.RuntimeID = i.RuntimeID
	return result, nil
}

// OperationType defines the possible types of an asynchronous operation to a broker.
type OperationType string

const (
	// OperationTypeProvision means provisioning OperationType
	OperationTypeProvision OperationType = "provision"
	// OperationTypeDeprovision means deprovision OperationType
	OperationTypeDeprovision OperationType = "deprovision"
	// OperationTypeUndefined means undefined OperationType
	OperationTypeUndefined OperationType = ""
	// OperationTypeUpgradeKyma means upgrade Kyma OperationType
	OperationTypeUpgradeKyma OperationType = "upgradeKyma"
	// OperationTypeUpdate means update
	OperationTypeUpdate OperationType = "update"
	// OperationTypeUpgradeCluster means upgrade cluster (shoot) OperationType
	OperationTypeUpgradeCluster OperationType = "upgradeCluster"
)

type Operation struct {
	// following fields are stored in the storage
	ID        string        `json:"-"`
	Version   int           `json:"-"`
	CreatedAt time.Time     `json:"-"`
	UpdatedAt time.Time     `json:"-"`
	Type      OperationType `json:"-"`

	InstanceID             string                    `json:"-"`
	ProvisionerOperationID string                    `json:"-"`
	State                  domain.LastOperationState `json:"-"`
	Description            string                    `json:"-"`
	ProvisioningParameters ProvisioningParameters    `json:"-"`

	// OrchestrationID specifies the origin orchestration which triggers the operation, empty for OSB operations (provisioning/deprovisioning)
	OrchestrationID string   `json:"-"`
	FinishedStages  []string `json:"-"`

	// following fields are serialized to JSON and stored in the storage
	InstanceDetails

	// PROVISIONING
	DashboardURL string `json:"dashboardURL"`

	// DEPROVISIONING
	// Temporary indicates that this deprovisioning operation must not remove the instance
	Temporary                   bool     `json:"temporary"`
	ClusterConfigurationDeleted bool     `json:"clusterConfigurationDeleted"`
	ExcutedButNotCompleted      []string `json:"excutedButNotCompleted"`
	UserAgent                   string   `json:"userAgent,omitempty"`

	// UPDATING
	UpdatingParameters UpdatingParametersDTO `json:"updating_parameters"`

	// UPGRADE KYMA
	orchestration.RuntimeOperation `json:"runtime_operation"`
	ClusterConfigurationApplied    bool `json:"cluster_configuration_applied"`

	// KymaTemplate is read from the configuration then used in the apply_kyma step
	KymaTemplate string `json:"KymaTemplate"`

	LastError kebError.LastError `json:"last_error"`

	// Used during KIM integration while deprovisioning - to be removed later on when provisioner not used anymore
	KimDeprovisionsOnly bool `json:"kim_deprovisions_only"`

	// following fields are not stored in the storage and should be added to the Merge function
	InputCreator ProvisionerInputCreator `json:"-"`
}

type GroupedOperations struct {
	ProvisionOperations      []ProvisioningOperation
	DeprovisionOperations    []DeprovisioningOperation
	UpgradeClusterOperations []UpgradeClusterOperation
	UpdateOperations         []UpdatingOperation
}

func (o *Operation) IsFinished() bool {
	return o.State != orchestration.InProgress && o.State != orchestration.Pending && o.State != orchestration.Canceling && o.State != orchestration.Retrying
}

func (o *Operation) EventInfof(fmt string, args ...any) {
	events.Infof(o.InstanceID, o.ID, fmt, args...)
}

func (o *Operation) EventErrorf(err error, fmt string, args ...any) {
	events.Errorf(o.InstanceID, o.ID, err, fmt, args...)
}

func (o *Operation) Merge(operation *Operation) {
	o.InputCreator = operation.InputCreator
}

// Orchestration holds all information about an orchestration.
// Orchestration performs operations of a specific type UpgradeClusterOperation
// on specific targets of SKRs.
type Orchestration struct {
	OrchestrationID string
	Type            orchestration.Type
	State           string
	Description     string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Parameters      orchestration.Parameters
}

func (o *Orchestration) IsFinished() bool {
	return o.State == orchestration.Succeeded || o.State == orchestration.Failed || o.State == orchestration.Canceled
}

// IsCanceled returns true if orchestration's cancellation endpoint was ever triggered
func (o *Orchestration) IsCanceled() bool {
	return o.State == orchestration.Canceling || o.State == orchestration.Canceled
}

type InstanceWithOperation struct {
	Instance

	Type           sql.NullString
	State          sql.NullString
	Description    sql.NullString
	OpCreatedAt    time.Time
	IsSuspensionOp bool
}

type InstanceDetails struct {
	EventHub EventHub `json:"eh"`

	SubAccountID      string                    `json:"sub_account_id"`
	RuntimeID         string                    `json:"runtime_id"`
	ShootName         string                    `json:"shoot_name"`
	ShootDomain       string                    `json:"shoot_domain"`
	ClusterName       string                    `json:"clusterName"`
	ShootDNSProviders gardener.DNSProvidersData `json:"shoot_dns_providers"`
	Monitoring        MonitoringData            `json:"monitoring"`
	EDPCreated        bool                      `json:"edp_created"`

	ClusterConfigurationVersion int64  `json:"cluster_configuration_version"`
	Kubeconfig                  string `json:"-"`

	ServiceManagerClusterID string `json:"sm_cluster_id"`

	KymaResourceNamespace string `json:"kyma_resource_namespace"`
	KymaResourceName      string `json:"kyma_resource_name"`
	GardenerClusterName   string `json:"gardener_cluster_name"`
	RuntimeResourceName   string `json:"runtime_resource_name"`

	EuAccess bool `json:"eu_access"`

	CloudProvider string `json:"cloud_provider"`

	// CompassRuntimeId - a runtime ID created by the Compass. Existing instances has a nil value (because the field was not existing) - it means the compass runtime Id is equal to runtime ID.
	// If the value is an empty string - it means the runtime was not registered by Provisioner in the Compass.
	// Should be removed after the migration of compass registration is completed
	CompassRuntimeId *string
}

// IsRegisteredInCompassByProvisioner returns true, if the runtime was registered in Compass by Provisioner
func (i *InstanceDetails) IsRegisteredInCompassByProvisioner() bool {
	return i.CompassRuntimeId == nil || *i.CompassRuntimeId != ""
}

func (i *InstanceDetails) SetCompassRuntimeIdNotRegisteredByProvisioner() {
	i.CompassRuntimeId = ptr.String("")
}

// GetCompassRuntimeId provides a compass runtime Id registered by Provisioner or empty string if it was not provisioned by Provisioner.
func (i *InstanceDetails) GetCompassRuntimeId() string {
	// for backward compatibility, if CompassRuntimeID field was not set, use RuntimeId
	if i.CompassRuntimeId == nil {
		return i.RuntimeID
	}
	return *i.CompassRuntimeId
}

func (i *InstanceDetails) GetRuntimeResourceName() string {
	name := i.RuntimeResourceName
	if name == "" {
		// fallback to runtime ID
		name = i.RuntimeID
	}
	return name
}

func (i *InstanceDetails) GetRuntimeResourceNamespace() string {
	namespace := i.KymaResourceNamespace
	if namespace == "" {
		// fallback to default namespace
		namespace = "kcp-system"
	}
	return namespace
}

// ProvisioningOperation holds all information about provisioning operation
type ProvisioningOperation struct {
	Operation
}

type InstanceArchived struct {
	InstanceID                  string
	GlobalAccountID             string
	SubaccountID                string
	SubscriptionGlobalAccountID string
	PlanID                      string
	PlanName                    string
	SubaccountRegion            string
	Region                      string
	Provider                    string
	LastRuntimeID               string
	InternalUser                bool
	ShootName                   string

	ProvisioningStartedAt         time.Time
	ProvisioningFinishedAt        time.Time
	ProvisioningState             domain.LastOperationState
	FirstDeprovisioningStartedAt  time.Time
	FirstDeprovisioningFinishedAt time.Time
	LastDeprovisioningFinishedAt  time.Time
}

func (a InstanceArchived) UserID() string {
	if a.InternalUser {
		return "somebody (at) sap.com"
	}
	return "- deleted -"
}

type MonitoringData struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// DeprovisioningOperation holds all information about de-provisioning operation
type DeprovisioningOperation struct {
	Operation
}

type UpdatingOperation struct {
	Operation
}

// UpgradeClusterOperation holds all information about upgrade cluster (shoot) operation
type UpgradeClusterOperation struct {
	Operation
}

func NewRuntimeState(runtimeID, operationID string, kymaConfig *gqlschema.KymaConfigInput, clusterConfig *gqlschema.GardenerConfigInput) RuntimeState {
	var (
		kymaConfigInput    gqlschema.KymaConfigInput
		clusterConfigInput gqlschema.GardenerConfigInput
	)
	if kymaConfig != nil {
		kymaConfigInput = *kymaConfig
	}
	if clusterConfig != nil {
		clusterConfigInput = *clusterConfig
	}

	return RuntimeState{
		ID:            uuid.New().String(),
		CreatedAt:     time.Now(),
		RuntimeID:     runtimeID,
		OperationID:   operationID,
		KymaConfig:    kymaConfigInput,
		ClusterConfig: clusterConfigInput,
	}
}

type RuntimeState struct {
	ID string `json:"id"`

	CreatedAt time.Time `json:"created_at"`

	RuntimeID   string `json:"runtimeId"`
	OperationID string `json:"operationId"`

	KymaConfig    gqlschema.KymaConfigInput     `json:"kymaConfig"`
	ClusterConfig gqlschema.GardenerConfigInput `json:"clusterConfig"`
}

// OperationStats provide number of operations per type and state
type OperationStats struct {
	Provisioning   map[domain.LastOperationState]int
	Deprovisioning map[domain.LastOperationState]int
}

type OperationStatsV2 struct {
	Count  int
	Type   OperationType
	State  domain.LastOperationState
	PlanID string
}

// InstanceStats provide number of instances per Global Account ID
type InstanceStats struct {
	TotalNumberOfInstances int
	PerGlobalAccountID     map[string]int
}

// ERSContextStats provides aggregated information regarding ERSContext
type ERSContextStats struct {
	LicenseType map[string]int
}

type BindingStats struct {
	MinutesSinceEarliestExpiration float64 `db:"minutes_since_earliest_expiration"`
}

// NewProvisioningOperation creates a fresh (just starting) instance of the ProvisioningOperation
func NewProvisioningOperation(instanceID string, parameters ProvisioningParameters) (ProvisioningOperation, error) {
	return NewProvisioningOperationWithID(uuid.New().String(), instanceID, parameters)
}

// NewProvisioningOperationWithID creates a fresh (just starting) instance of the ProvisioningOperation with provided ID
func NewProvisioningOperationWithID(operationID, instanceID string, parameters ProvisioningParameters) (ProvisioningOperation, error) {
	return ProvisioningOperation{
		Operation: Operation{
			ID:                     operationID,
			Version:                0,
			Description:            "Operation created",
			InstanceID:             instanceID,
			State:                  domain.InProgress,
			CreatedAt:              time.Now(),
			UpdatedAt:              time.Now(),
			Type:                   OperationTypeProvision,
			ProvisioningParameters: parameters,
			RuntimeOperation: orchestration.RuntimeOperation{
				Runtime: orchestration.Runtime{
					GlobalAccountID: parameters.ErsContext.GlobalAccountID,
				},
			},
			InstanceDetails: InstanceDetails{
				SubAccountID: parameters.ErsContext.SubAccountID,
				Kubeconfig:   parameters.Parameters.Kubeconfig,
				EuAccess:     euaccess.IsEURestrictedAccess(parameters.PlatformRegion),
			},
			FinishedStages: make([]string, 0),
			LastError:      kebError.LastError{},
		},
	}, nil
}

// NewDeprovisioningOperationWithID creates a fresh (just starting) instance of the DeprovisioningOperation with provided ID
func NewDeprovisioningOperationWithID(operationID string, instance *Instance) (DeprovisioningOperation, error) {
	details, err := instance.GetInstanceDetails()
	if err != nil {
		return DeprovisioningOperation{}, err
	}
	return DeprovisioningOperation{
		Operation: Operation{
			RuntimeOperation: orchestration.RuntimeOperation{
				Runtime: orchestration.Runtime{GlobalAccountID: instance.GlobalAccountID, RuntimeID: instance.RuntimeID, Region: instance.ProviderRegion},
			},
			ID:                     operationID,
			Version:                0,
			Description:            "Operation created",
			InstanceID:             instance.InstanceID,
			State:                  orchestration.Pending,
			CreatedAt:              time.Now(),
			UpdatedAt:              time.Now(),
			Type:                   OperationTypeDeprovision,
			InstanceDetails:        details,
			FinishedStages:         make([]string, 0),
			ProvisioningParameters: instance.Parameters,
		},
	}, nil
}

func NewUpdateOperation(operationID string, instance *Instance, updatingParams UpdatingParametersDTO) Operation {

	op := Operation{
		ID:                     operationID,
		Version:                0,
		Description:            "Operation created",
		InstanceID:             instance.InstanceID,
		State:                  orchestration.Pending,
		CreatedAt:              time.Now(),
		UpdatedAt:              time.Now(),
		Type:                   OperationTypeUpdate,
		InstanceDetails:        instance.InstanceDetails,
		FinishedStages:         make([]string, 0),
		ProvisioningParameters: instance.Parameters,
		UpdatingParameters:     updatingParams,
		RuntimeOperation: orchestration.RuntimeOperation{
			Runtime: orchestration.Runtime{
				Region: instance.ProviderRegion},
		},
	}
	if updatingParams.OIDC != nil {
		op.ProvisioningParameters.Parameters.OIDC = updatingParams.OIDC
	}

	if len(updatingParams.RuntimeAdministrators) != 0 {
		op.ProvisioningParameters.Parameters.RuntimeAdministrators = updatingParams.RuntimeAdministrators
	}

	updatingParams.UpdateAutoScaler(&op.ProvisioningParameters.Parameters)
	if updatingParams.MachineType != nil && *updatingParams.MachineType != "" {
		op.ProvisioningParameters.Parameters.MachineType = updatingParams.MachineType
	}

	return op
}

// NewSuspensionOperationWithID creates a fresh (just starting) instance of the DeprovisioningOperation which does not remove the instance.
func NewSuspensionOperationWithID(operationID string, instance *Instance) DeprovisioningOperation {
	return DeprovisioningOperation{
		Operation: Operation{
			ID:                     operationID,
			Version:                0,
			Description:            "Operation created",
			InstanceID:             instance.InstanceID,
			State:                  orchestration.Pending,
			CreatedAt:              time.Now(),
			UpdatedAt:              time.Now(),
			Type:                   OperationTypeDeprovision,
			InstanceDetails:        instance.InstanceDetails,
			ProvisioningParameters: instance.Parameters,
			FinishedStages:         make([]string, 0),
			Temporary:              true,
			RuntimeOperation: orchestration.RuntimeOperation{
				Runtime: orchestration.Runtime{
					Region: instance.ProviderRegion},
			},
		},
	}
}

func (o *Operation) FinishStage(stageName string) {
	if stageName == "" {
		log.Warnf("Attempt to add empty stage.")
		return
	}

	if exists := o.IsStageFinished(stageName); exists {
		log.Warnf("Attempt to add stage (%s) which is already saved.", stageName)
		return
	}

	o.FinishedStages = append(o.FinishedStages, stageName)
}

func (o *Operation) IsStageFinished(stage string) bool {
	for _, value := range o.FinishedStages {
		if value == stage {
			return true
		}
	}
	return false
}

func (o *Operation) SuccessMustBeSaved() bool {

	// if the operation is temporary, it must be saved
	if o.Temporary {
		return true
	}

	// if the operation is not temporary and the last stage is success, it must not be saved
	// because all operations for that instance are gone
	if o.Type == OperationTypeDeprovision {
		return false
	}
	return true
}

type ConfigForPlan struct {
	KymaTemplate string `json:"kyma-template" yaml:"kyma-template"`
}

type SubaccountState struct {
	ID string `json:"id"`

	BetaEnabled       string `json:"betaEnabled"`
	UsedForProduction string `json:"usedForProduction"`
	ModifiedAt        int64  `json:"modifiedAt"`
}

type DeletedStats struct {
	NumberOfDeletedInstances              int
	NumberOfOperationsForDeletedInstances int
}

type Binding struct {
	ID         string
	InstanceID string

	CreatedAt time.Time
	UpdatedAt time.Time
	ExpiresAt time.Time

	Kubeconfig        string
	ExpirationSeconds int64
	CreatedBy         string
}
