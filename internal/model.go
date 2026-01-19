package internal

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/euaccess"

	"github.com/google/uuid"
	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	kebError "github.com/kyma-project/kyma-environment-broker/internal/error"
	"github.com/kyma-project/kyma-environment-broker/internal/events"
	"github.com/pivotal-cf/brokerapi/v12/domain"
)

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
	SubscriptionSecretName      string

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

type InstanceWithSubaccountState struct {
	Instance
	BetaEnabled       string
	UsedForProduction string
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
	// overwrite RuntimeID in InstanceDetails with Instance.RuntimeID
	// needed for runtimes suspended without clearing RuntimeID in deprovisioning operation
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

// replacement for orchestration constants
const (
	OperationStatePending    = "pending"
	OperationStateCanceled   = "canceled"
	OperationStateRetrying   = "retrying"
	OperationStateCanceling  = "canceling"
	OperationStateInProgress = "in progress"
)

// RuntimeOperation this structure is needed for backward compatibility with the old data persisted by orchestration code
type RuntimeOperation struct {
	GlobalAccountID string `json:"globalAccountId"`
	Region          string `json:"region"`
}

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

	FinishedStages []string `json:"-"`

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

	// UpdatedPlanID is used to store the plan ID if the plan has been changed, "" if not changed
	UpdatedPlanID string `json:"updated_plan_id,omitempty"`

	// UPGRADE KYMA
	RuntimeOperation            `json:"runtime_operation"`
	ClusterConfigurationApplied bool `json:"cluster_configuration_applied"`

	// KymaTemplate is read from the configuration then used in the apply_kyma step
	KymaTemplate string `json:"KymaTemplate"`

	LastError kebError.LastError `json:"last_error"`

	// DiscoveredZones stores availability zones per machine type, resolved at runtime
	DiscoveredZones map[string][]string `json:"discovered_zones"`
}

// ProviderValues contains values which are specific to particular plans (and provisioning parameters)
type ProviderValues struct {
	DefaultAutoScalerMax int
	DefaultAutoScalerMin int
	ZonesCount           int
	Zones                []string
	ProviderType         string
	DefaultMachineType   string
	Region               string
	Purpose              string
	VolumeSizeGb         int
	DiskType             string
	FailureTolerance     *string
}

type GroupedOperations struct {
	ProvisionOperations      []ProvisioningOperation
	DeprovisionOperations    []DeprovisioningOperation
	UpgradeClusterOperations []UpgradeClusterOperation
	UpdateOperations         []UpdatingOperation
}

func (o *Operation) IsFinished() bool {
	return o.State != OperationStateInProgress && o.State != OperationStatePending && o.State != OperationStateCanceling && o.State != OperationStateRetrying
}

func (o *Operation) EventInfof(fmt string, args ...any) {
	events.Infof(o.InstanceID, o.ID, fmt, args...)
}

func (o *Operation) EventErrorf(err error, fmt string, args ...any) {
	events.Errorf(o.InstanceID, o.ID, err, fmt, args...)
}

func (o *Operation) Merge(operation *Operation) {
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

	ProviderValues *ProviderValues `json:"providerValues"`
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

type RuntimeState struct {
	ID string `json:"id"`

	CreatedAt time.Time `json:"created_at"`

	RuntimeID   string `json:"runtimeId"`
	OperationID string `json:"operationId"`
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
	PerSubAcocuntID        map[string]int
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
			RuntimeOperation: RuntimeOperation{
				GlobalAccountID: parameters.ErsContext.GlobalAccountID,
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
			RuntimeOperation: RuntimeOperation{
				GlobalAccountID: instance.GlobalAccountID,
				Region:          instance.ProviderRegion,
			},
			ID:                     operationID,
			Version:                0,
			Description:            "Operation created",
			InstanceID:             instance.InstanceID,
			State:                  OperationStatePending,
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
		State:                  OperationStatePending,
		CreatedAt:              time.Now(),
		UpdatedAt:              time.Now(),
		Type:                   OperationTypeUpdate,
		InstanceDetails:        instance.InstanceDetails,
		FinishedStages:         make([]string, 0),
		ProvisioningParameters: instance.Parameters,
		UpdatingParameters:     updatingParams,
		RuntimeOperation: RuntimeOperation{
			GlobalAccountID: instance.GlobalAccountID,
			Region:          instance.ProviderRegion},
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

	if updatingParams.AdditionalWorkerNodePools != nil {
		op.ProvisioningParameters.Parameters.AdditionalWorkerNodePools = updatingParams.AdditionalWorkerNodePools
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
			State:                  OperationStatePending,
			CreatedAt:              time.Now(),
			UpdatedAt:              time.Now(),
			Type:                   OperationTypeDeprovision,
			InstanceDetails:        instance.InstanceDetails,
			ProvisioningParameters: instance.Parameters,
			FinishedStages:         make([]string, 0),
			Temporary:              true,
		},
	}
}

func (o *Operation) FinishStage(stageName string) {
	if stageName == "" {
		slog.Warn("Attempt to add empty stage.")
		return
	}

	if exists := o.IsStageFinished(stageName); exists {
		slog.Warn(fmt.Sprintf("Attempt to add stage (%s) which is already saved.", stageName))
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

type RetryTuple struct {
	Timeout  time.Duration
	Interval time.Duration
}

type ProviderConfig struct {
	SeedRegions []string `json:"seedRegions" yaml:"seedRegions"`
}

type RegionsSupporter interface {
	IsSupported(region string, machineType string) bool
	SupportedRegions(machineType string) []string
	AvailableZonesForAdditionalWorkers(machineType, region, planID string) ([]string, error)
}
