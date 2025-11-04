package broker

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules"
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/additionalproperties"
	"github.com/kyma-project/kyma-environment-broker/internal/config"
	"github.com/kyma-project/kyma-environment-broker/internal/dashboard"
	"github.com/kyma-project/kyma-environment-broker/internal/euaccess"
	"github.com/kyma-project/kyma-environment-broker/internal/hyperscalers/aws"
	"github.com/kyma-project/kyma-environment-broker/internal/kubeconfig"
	"github.com/kyma-project/kyma-environment-broker/internal/middleware"
	"github.com/kyma-project/kyma-environment-broker/internal/networking"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/kyma-project/kyma-environment-broker/internal/subscriptions"
	"github.com/kyma-project/kyma-environment-broker/internal/validator"
	"github.com/kyma-project/kyma-environment-broker/internal/whitelist"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/pivotal-cf/brokerapi/v12/domain/apiresponses"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"k8s.io/client-go/tools/clientcmd"
)

//go:generate mockery --name=Queue --output=automock --outpkg=automock --case=underscore
//go:generate mockery --name=PlanValidator --output=automock --outpkg=automock --case=underscore
//go:generate mockery --name=QuotaClient --output=automock --outpkg=automock --case=underscore

type (
	Queue interface {
		Add(operationId string)
	}

	PlanValidator interface {
		IsPlanSupport(planID string) bool
		GetDefaultOIDC() *pkg.OIDCConfigDTO
	}

	ValuesProvider interface {
		ValuesForPlanAndParameters(provisioningParameters internal.ProvisioningParameters) (internal.ProviderValues, error)
	}

	ConfigurationProvider interface {
		RegionSupportingMachine(providerType string) (internal.RegionsSupporter, error)
		ZonesDiscovery(cp pkg.CloudProvider) bool
	}

	QuotaClient interface {
		GetQuota(subAccountID, planName string) (int, error)
	}
)

type ProvisionEndpoint struct {
	config                  Config
	infrastructureManager   InfrastructureManager
	operationsStorage       storage.Operations
	instanceStorage         storage.Instances
	instanceArchivedStorage storage.InstancesArchived
	queue                   Queue
	enabledPlanIDs          map[string]struct{}
	plansConfig             PlansConfig

	shootDomain       string
	shootProject      string
	shootDnsProviders gardener.DNSProvidersData

	dashboardConfig dashboard.Config
	kcBuilder       kubeconfig.KcBuilder

	freemiumWhiteList whitelist.Set

	log                    *slog.Logger
	valuesProvider         ValuesProvider
	useSmallerMachineTypes bool
	schemaService          *SchemaService
	providerConfigProvider config.ConfigMapConfigProvider
	providerSpec           ConfigurationProvider
	quotaClient            QuotaClient
	quotaWhitelist         whitelist.Set
	rulesService           *rules.RulesService
	gardenerClient         *gardener.Client
	awsClientFactory       aws.ClientFactory
	useCredentialsBindings bool
}

const (
	ConvergedCloudBlockedMsg                           = "This offer is currently not available."
	IngressFilteringNotSupportedForPlanMsg             = "ingress filtering is not available for %s plan"
	IngressFilteringNotSupportedForExternalCustomerMsg = "ingress filtering is not available for your type of license"
	IngressFilteringOptionIsNotSupported               = "ingress filtering option is not available"
	FailedToValidateZonesMsg                           = "Failed to validate the number of available zones. Please try again later."
)

func NewProvision(brokerConfig Config,
	gardenerConfig gardener.Config,
	imConfig InfrastructureManager,
	db storage.BrokerStorage,
	queue Queue,
	plansConfig PlansConfig,
	log *slog.Logger,
	dashboardConfig dashboard.Config,
	kcBuilder kubeconfig.KcBuilder,
	freemiumWhitelist whitelist.Set,
	schemaService *SchemaService,
	providerSpec ConfigurationProvider,
	valuesProvider ValuesProvider,
	useSmallerMachineTypes bool,
	providerConfigProvider config.ConfigMapConfigProvider,
	quotaClient QuotaClient,
	quotaWhitelist whitelist.Set,
	rulesService *rules.RulesService,
	gardenerClient *gardener.Client,
	awsClientFactory aws.ClientFactory,
) *ProvisionEndpoint {
	enabledPlanIDs := map[string]struct{}{}
	for _, planName := range brokerConfig.EnablePlans {
		id := PlanIDsMapping[planName]
		enabledPlanIDs[id] = struct{}{}
	}

	return &ProvisionEndpoint{
		config:                  brokerConfig,
		infrastructureManager:   imConfig,
		operationsStorage:       db.Operations(),
		instanceStorage:         db.Instances(),
		instanceArchivedStorage: db.InstancesArchived(),
		queue:                   queue,
		log:                     log.With("service", "ProvisionEndpoint"),
		enabledPlanIDs:          enabledPlanIDs,
		plansConfig:             plansConfig,
		shootDomain:             gardenerConfig.ShootDomain,
		shootProject:            gardenerConfig.Project,
		shootDnsProviders:       gardenerConfig.DNSProviders,
		dashboardConfig:         dashboardConfig,
		freemiumWhiteList:       freemiumWhitelist,
		kcBuilder:               kcBuilder,
		providerSpec:            providerSpec,
		valuesProvider:          valuesProvider,
		useSmallerMachineTypes:  useSmallerMachineTypes,
		schemaService:           schemaService,
		providerConfigProvider:  providerConfigProvider,
		quotaClient:             quotaClient,
		quotaWhitelist:          quotaWhitelist,
		rulesService:            rulesService,
		gardenerClient:          gardenerClient,
		awsClientFactory:        awsClientFactory,
	}
}

// Provision creates a new service instance
//
//	PUT /v2/service_instances/{instance_id}
func (b *ProvisionEndpoint) Provision(ctx context.Context, instanceID string, details domain.ProvisionDetails, asyncAllowed bool) (domain.ProvisionedServiceSpec, error) {
	operationID := uuid.New().String()
	logger := b.log.With("instanceID", instanceID, "operationID", operationID, "planID", details.PlanID)
	logger.Info(fmt.Sprintf("Provision called with context: %s", marshallRawContext(hideSensitiveDataFromRawContext(details.RawContext))))

	region, found := middleware.RegionFromContext(ctx)
	if !found {
		err := fmt.Errorf("%s", "No region specified in request.")
		return domain.ProvisionedServiceSpec{}, apiresponses.NewFailureResponse(err, http.StatusInternalServerError, "provisioning")
	}
	platformProvider, found := middleware.ProviderFromContext(ctx)
	if !found {
		err := fmt.Errorf("%s", "No provider specified in request.")
		return domain.ProvisionedServiceSpec{}, apiresponses.NewFailureResponse(err, http.StatusInternalServerError, "provisioning")
	}

	// EXTRACT INPUT PARAMETERS / PROVISIONING PARAMETERS
	parameters, err := b.extractInputParameters(details)
	if err != nil {
		return domain.ProvisionedServiceSpec{}, apiresponses.NewFailureResponse(err, http.StatusBadRequest, "while extracting input parameters")
	}
	ersContext, err := b.extractERSContext(details)
	logger = logger.With("globalAccountID", ersContext.GlobalAccountID)
	if err != nil {
		return domain.ProvisionedServiceSpec{}, apiresponses.NewFailureResponse(err, http.StatusBadRequest, "while extracting context")
	}
	if b.config.MonitorAdditionalProperties {
		b.monitorAdditionalProperties(instanceID, ersContext, details.RawParameters)
	}
	provisioningParameters := internal.ProvisioningParameters{
		PlanID:           details.PlanID,
		ServiceID:        details.ServiceID,
		ErsContext:       ersContext,
		Parameters:       parameters,
		PlatformRegion:   region,
		PlatformProvider: platformProvider,
	}
	// TODO: remove once we implemented proper filtering of parameters - removing parameters that are not supported by the plan
	if details.PlanID == TrialPlanID {
		provisioningParameters.Parameters.MachineType = nil
		provisioningParameters.Parameters.AutoScalerMin = nil
		provisioningParameters.Parameters.AutoScalerMax = nil
	}
	providerValues, err := b.valuesProvider.ValuesForPlanAndParameters(provisioningParameters)
	if err != nil {
		errMsg := fmt.Sprintf("unable to provide default values for instance %s: %s", instanceID, err)
		return domain.ProvisionedServiceSpec{}, apiresponses.NewFailureResponse(err, http.StatusBadRequest, errMsg)
	}

	// validation of incoming input
	err = b.validate(ctx, details, provisioningParameters, logger)
	if err != nil {
		errMsg := fmt.Sprintf("[instanceID: %s] %s", instanceID, err)
		return domain.ProvisionedServiceSpec{}, apiresponses.NewFailureResponse(err, http.StatusBadRequest, errMsg)
	}

	logger.Info(fmt.Sprintf("Starting provisioning runtime: Name=%s, GlobalAccountID=%s, SubAccountID=%s, PlatformRegion=%s, ProvisioningParameters.Region=%s, ProvisioningParameters.ColocateControlPlane=%t, ProvisioningParameters.MachineType=%s",
		parameters.Name, ersContext.GlobalAccountID, ersContext.SubAccountID, region, valueOfPtr(parameters.Region),
		valueOfBoolPtr(parameters.ColocateControlPlane), valueOfPtr(parameters.MachineType)))
	logParametersWithMaskedKubeconfig(parameters, logger)

	// check if operation with instance ID already created
	existingOperation, errStorage := b.operationsStorage.GetProvisioningOperationByInstanceID(instanceID)
	switch {
	case errStorage != nil && !dberr.IsNotFound(errStorage):
		logger.Error(fmt.Sprintf("cannot get existing operation from storage %s", errStorage))
		return domain.ProvisionedServiceSpec{}, fmt.Errorf("cannot get existing operation from storage")
	case existingOperation != nil && !dberr.IsNotFound(errStorage):
		return b.handleExistingOperation(existingOperation, provisioningParameters)
	}

	shootName := gardener.CreateShootName()
	shootDomainSuffix := strings.Trim(b.shootDomain, ".")

	dashboardURL := b.createDashboardURL(details.PlanID, instanceID)

	// create and save new operation
	operation, err := internal.NewProvisioningOperationWithID(operationID, instanceID, provisioningParameters)
	if err != nil {
		logger.Error(fmt.Sprintf("cannot create new operation: %s", err))
		return domain.ProvisionedServiceSpec{}, fmt.Errorf("cannot create new operation")
	}

	operation.ProviderValues = &providerValues
	operation.ShootName = shootName
	operation.ShootDomain = fmt.Sprintf("%s.%s", shootName, shootDomainSuffix)
	operation.ShootDNSProviders = b.shootDnsProviders
	operation.DashboardURL = dashboardURL
	// for own cluster plan - KEB uses provided shoot name and shoot domain
	if IsOwnClusterPlan(provisioningParameters.PlanID) {
		operation.ShootName = provisioningParameters.Parameters.ShootName
		operation.ShootDomain = provisioningParameters.Parameters.ShootDomain
	}
	logger.Info(fmt.Sprintf("Runtime ShootDomain: %s", operation.ShootDomain))

	err = b.operationsStorage.InsertOperation(operation.Operation)
	if err != nil {
		logger.Error(fmt.Sprintf("cannot save operation: %s", err))
		return domain.ProvisionedServiceSpec{}, fmt.Errorf("cannot save operation")
	}

	instance := internal.Instance{
		InstanceID:      instanceID,
		GlobalAccountID: ersContext.GlobalAccountID,
		SubAccountID:    ersContext.SubAccountID,
		ServiceID:       provisioningParameters.ServiceID,
		ServiceName:     KymaServiceName,
		ServicePlanID:   provisioningParameters.PlanID,
		ServicePlanName: PlanNamesMapping[provisioningParameters.PlanID],
		DashboardURL:    dashboardURL,
		Parameters:      operation.ProvisioningParameters,
		Provider:        pkg.CloudProviderFromString(providerValues.ProviderType),
	}
	err = b.instanceStorage.Insert(instance)
	if err != nil {
		logger.Error(fmt.Sprintf("cannot save instance in storage: %s", err))
		return domain.ProvisionedServiceSpec{}, fmt.Errorf("cannot save instance")
	}

	err = b.instanceStorage.UpdateInstanceLastOperation(instanceID, operationID)
	if err != nil {
		logger.Error(fmt.Sprintf("cannot save instance in storage: %s", err))
		return domain.ProvisionedServiceSpec{}, fmt.Errorf("cannot save instance")
	}

	logger.Info("Adding operation to provisioning queue")
	b.queue.Add(operation.ID)

	return domain.ProvisionedServiceSpec{
		IsAsync:       true,
		OperationData: operation.ID,
		DashboardURL:  dashboardURL,
		Metadata: domain.InstanceMetadata{
			Labels: ResponseLabels(operation, instance, b.config.URL, b.kcBuilder),
		},
	}, nil
}

// UseCredentialsBindings indicates whether to use credentials bindings when creating AWS clients, it is a deprecated func and will be removed in future releases
// when all KCP instances are migrated to use credentials bindings
func (b *ProvisionEndpoint) UseCredentialsBindings() {
	b.useCredentialsBindings = true
}

func logParametersWithMaskedKubeconfig(parameters pkg.ProvisioningParametersDTO, logger *slog.Logger) {
	parameters.Kubeconfig = "*****"
	logger.Info(fmt.Sprintf("Runtime parameters: %+v", parameters))
}

func valueOfPtr(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

func valueOfBoolPtr(ptr *bool) bool {
	if ptr == nil {
		return false
	}
	return *ptr
}

func (b *ProvisionEndpoint) validate(ctx context.Context, details domain.ProvisionDetails, provisioningParameters internal.ProvisioningParameters, l *slog.Logger) error {
	parameters := provisioningParameters.Parameters
	if details.ServiceID != KymaServiceID {
		return fmt.Errorf("service_id not recognized")
	}
	if _, exists := b.enabledPlanIDs[details.PlanID]; !exists {
		return fmt.Errorf("plan ID %q is not recognized", details.PlanID)
	}

	values, err := b.valuesProvider.ValuesForPlanAndParameters(provisioningParameters)
	if err != nil {
		return fmt.Errorf("while obtaining plan defaults: %w", err)
	}

	if b.config.CheckQuotaLimit && whitelist.IsNotWhitelisted(provisioningParameters.ErsContext.SubAccountID, b.quotaWhitelist) {
		if err := validateQuotaLimit(b.instanceStorage, b.quotaClient, provisioningParameters.ErsContext.SubAccountID, provisioningParameters.PlanID, false); err != nil {
			return err
		}
	}

	colocateControlPlane := valueOfBoolPtr(parameters.ColocateControlPlane)
	if colocateControlPlane {
		platformRegion, _ := middleware.RegionFromContext(ctx)
		supportedRegions := b.schemaService.PlanRegions(PlanNamesMapping[details.PlanID], platformRegion)
		if err := b.validateColocationRegion(strings.ToLower(values.ProviderType), valueOfPtr(parameters.Region), supportedRegions, l); err != nil {
			return fmt.Errorf("validation of the region for colocating the control plane: %w", err)
		}
	}

	regionsSupportingMachine, err := b.providerSpec.RegionSupportingMachine(values.ProviderType)
	if err != nil {
		return fmt.Errorf("while obtaining regions supporting machine: %w", err)
	}
	if !regionsSupportingMachine.IsSupported(valueOfPtr(parameters.Region), valueOfPtr(parameters.MachineType)) {
		return fmt.Errorf(
			"In the region %s, the machine type %s is not available, it is supported in the %v",
			valueOfPtr(parameters.Region),
			valueOfPtr(parameters.MachineType),
			strings.Join(regionsSupportingMachine.SupportedRegions(valueOfPtr(parameters.MachineType)), ", "),
		)
	}

	discoveredZones := make(map[string]int)
	if b.providerSpec.ZonesDiscovery(pkg.CloudProviderFromString(values.ProviderType)) {
		kymaMachineType := values.DefaultMachineType
		if parameters.MachineType != nil {
			kymaMachineType = *parameters.MachineType
		}

		discoveredZones[kymaMachineType] = 0
		for _, additionalWorkerNodePool := range parameters.AdditionalWorkerNodePools {
			discoveredZones[additionalWorkerNodePool.MachineType] = 0
		}

		// todo: simplify it, remove "if" when all KCP insdtances are migrated to use credentials bindings
		var awsClient aws.Client
		if b.useCredentialsBindings {
			awsClient, err = newAWSClientUsingCredentialsBinding(ctx, l, b.rulesService, b.gardenerClient, b.awsClientFactory, provisioningParameters, values)
		} else {
			awsClient, err = newAWSClient(ctx, l, b.rulesService, b.gardenerClient, b.awsClientFactory, provisioningParameters, values)
		}
		if err != nil {
			l.Error(fmt.Sprintf("unable to create AWS client: %s", err))
			return apiresponses.NewFailureResponse(fmt.Errorf(FailedToValidateZonesMsg), http.StatusUnprocessableEntity, FailedToValidateZonesMsg)
		}

		for machineType := range discoveredZones {
			zonesCount, err := awsClient.AvailableZonesCount(ctx, machineType)
			if err != nil {
				l.Error(fmt.Sprintf("unable to get available zones: %s", err))
				return apiresponses.NewFailureResponse(fmt.Errorf(FailedToValidateZonesMsg), http.StatusUnprocessableEntity, FailedToValidateZonesMsg)
			}
			discoveredZones[machineType] = zonesCount
		}

		if discoveredZones[kymaMachineType] < values.ZonesCount {
			message := fmt.Sprintf("In the %s, the %s machine type is not available in %v zones.", values.Region, kymaMachineType, values.ZonesCount)
			return apiresponses.NewFailureResponse(fmt.Errorf("%s", message), http.StatusUnprocessableEntity, message)
		}
	}

	if err := b.validateNetworking(parameters); err != nil {
		return err
	}

	if err := parameters.AutoScalerParameters.Validate(values.DefaultAutoScalerMin, values.DefaultAutoScalerMax); err != nil {
		return apiresponses.NewFailureResponse(err, http.StatusUnprocessableEntity, err.Error())
	}
	if parameters.OIDC.IsProvided() {
		if err := parameters.OIDC.Validate(nil); err != nil {
			return apiresponses.NewFailureResponse(err, http.StatusUnprocessableEntity, err.Error())
		}
	}

	if parameters.AdditionalWorkerNodePools != nil {
		if !supportsAdditionalWorkerNodePools(details.PlanID) {
			message := fmt.Sprintf("additional worker node pools are not supported for plan ID: %s", details.PlanID)
			return apiresponses.NewFailureResponse(fmt.Errorf("%s", message), http.StatusUnprocessableEntity, message)
		}

		if !AreNamesUnique(parameters.AdditionalWorkerNodePools) {
			message := "names of additional worker node pools must be unique"
			return apiresponses.NewFailureResponse(fmt.Errorf("%s", message), http.StatusUnprocessableEntity, message)
		}

		if IsExternalLicenseType(provisioningParameters.ErsContext) {
			if err := checkGPUMachinesUsage(parameters.AdditionalWorkerNodePools); err != nil {
				return apiresponses.NewFailureResponse(err, http.StatusUnprocessableEntity, err.Error())
			}
		}

		if err := checkUnsupportedMachines(regionsSupportingMachine, valueOfPtr(parameters.Region), parameters.AdditionalWorkerNodePools); err != nil {
			return apiresponses.NewFailureResponse(err, http.StatusUnprocessableEntity, err.Error())
		}

		if err := checkAutoScalerConfiguration(parameters.AdditionalWorkerNodePools); err != nil {
			return apiresponses.NewFailureResponse(err, http.StatusUnprocessableEntity, err.Error())
		}

		if err := checkAvailableZones(
			l,
			regionsSupportingMachine,
			parameters.AdditionalWorkerNodePools,
			valueOfPtr(parameters.Region),
			details.PlanID,
			b.providerSpec.ZonesDiscovery(pkg.CloudProviderFromString(values.ProviderType)),
			discoveredZones,
		); err != nil {
			return apiresponses.NewFailureResponse(err, http.StatusUnprocessableEntity, err.Error())
		}
	}

	err = validateIngressFiltering(provisioningParameters, parameters.IngressFiltering, b.infrastructureManager.IngressFilteringPlans, l)
	if err != nil {
		return apiresponses.NewFailureResponse(err, http.StatusBadRequest, err.Error())
	}

	planValidator, err := b.validator(&details, provisioningParameters.PlatformProvider, ctx)
	if err != nil {
		return fmt.Errorf("while creating plan validator: %w", err)
	}

	var rawParameters any
	if err = json.Unmarshal(details.RawParameters, &rawParameters); err != nil {
		return fmt.Errorf("while unmarshaling raw parameters: %w", err)
	}

	if err = planValidator.Validate(rawParameters); err != nil {
		return fmt.Errorf("while validating input parameters: %s", validator.FormatError(err))
	}

	// EU Access
	if isEuRestrictedAccess(ctx) {
		l.Info("EU Access restricted instance creation")
	}

	if IsOwnClusterPlan(details.PlanID) {
		decodedKubeconfig, err := base64.StdEncoding.DecodeString(parameters.Kubeconfig)
		if err != nil {
			return fmt.Errorf("while decoding kubeconfig: %w", err)
		}
		parameters.Kubeconfig = string(decodedKubeconfig)
		err = validateKubeconfig(parameters.Kubeconfig)
		if err != nil {
			return fmt.Errorf("while validating kubeconfig: %w", err)
		}
	}

	if IsTrialPlan(details.PlanID) && parameters.Region != nil && *parameters.Region != "" {
		_, valid := validRegionsForTrial[TrialCloudRegion(*parameters.Region)]
		if !valid {
			return fmt.Errorf("invalid region specified in request for trial")
		}
	}

	if IsTrialPlan(details.PlanID) && b.config.OnlySingleTrialPerGA {
		count, err := b.instanceStorage.GetNumberOfInstancesForGlobalAccountID(provisioningParameters.ErsContext.GlobalAccountID)
		if err != nil {
			return fmt.Errorf("while checking if a trial Kyma instance exists for given global account: %w", err)
		}

		if count > 0 {
			l.Info("Provisioning Trial SKR rejected, such instance was already created for this Global Account")
			return fmt.Errorf("trial Kyma was created for the global account, but there is only one allowed")
		}
	}

	if IsFreemiumPlan(details.PlanID) && b.config.OnlyOneFreePerGA && whitelist.IsNotWhitelisted(provisioningParameters.ErsContext.GlobalAccountID, b.freemiumWhiteList) {
		count, err := b.instanceArchivedStorage.TotalNumberOfInstancesArchivedForGlobalAccountID(provisioningParameters.ErsContext.GlobalAccountID, FreemiumPlanID)
		if err != nil {
			return fmt.Errorf("while checking if a free Kyma instance existed for given global account: %w", err)
		}
		if count > 0 {
			l.Info("Provisioning Free SKR rejected, such instance was already created for this Global Account")
			return fmt.Errorf("provisioning request rejected, you have already used the available free service plan quota in this global account")
		}

		instanceFilter := dbmodel.InstanceFilter{
			GlobalAccountIDs: []string{provisioningParameters.ErsContext.GlobalAccountID},
			PlanIDs:          []string{FreemiumPlanID},
			States:           []dbmodel.InstanceState{dbmodel.InstanceSucceeded},
		}
		_, _, count, err = b.instanceStorage.List(instanceFilter)
		if err != nil {
			return fmt.Errorf("while checking if a free Kyma instance existed for given global account: %w", err)
		}
		if count > 0 {
			l.Info("Provisioning Free SKR rejected, such instance was already created for this Global Account")
			return fmt.Errorf("provisioning request rejected, you have already used the available free service plan quota in this global account")
		}
	}

	return nil
}

func validateIngressFiltering(provisioningParameters internal.ProvisioningParameters, ingressFilteringParameter *bool, plans EnablePlans, log *slog.Logger) error {
	if ingressFilteringParameter != nil {
		if !plans.Contains(PlanNamesMapping[provisioningParameters.PlanID]) {
			log.Info(fmt.Sprintf(IngressFilteringNotSupportedForPlanMsg, PlanNamesMapping[provisioningParameters.PlanID]))
			return fmt.Errorf(IngressFilteringOptionIsNotSupported)
		}
		if IsExternalLicenseType(provisioningParameters.ErsContext) && *ingressFilteringParameter {
			log.Info(IngressFilteringNotSupportedForExternalCustomerMsg)
			return fmt.Errorf(IngressFilteringOptionIsNotSupported)
		}
	}
	return nil
}

func isEuRestrictedAccess(ctx context.Context) bool {
	platformRegion, _ := middleware.RegionFromContext(ctx)
	return euaccess.IsEURestrictedAccess(platformRegion)
}

func supportsAdditionalWorkerNodePools(planID string) bool {
	var unsupportedPlans = []string{
		FreemiumPlanID,
		TrialPlanID,
	}
	for _, unsupportedPlan := range unsupportedPlans {
		if planID == unsupportedPlan {
			return false
		}
	}
	return true
}

func AreNamesUnique(pools []pkg.AdditionalWorkerNodePool) bool {
	nameSet := make(map[string]struct{})
	for _, pool := range pools {
		if _, exists := nameSet[pool.Name]; exists {
			return false
		}
		nameSet[pool.Name] = struct{}{}
	}
	return true
}

func IsExternalLicenseType(ersContext internal.ERSContext) bool {
	return *ersContext.ExternalLicenseType()
}

func checkGPUMachinesUsage(additionalWorkerNodePools []pkg.AdditionalWorkerNodePool) error {
	var GPUMachines = []string{
		"g2-standard",
		"g6",
		"g4dn",
		"Standard_NC",
	}

	usedGPUMachines := make(map[string][]string)
	var orderedMachineTypes []string

	for _, pool := range additionalWorkerNodePools {
		for _, GPUMachine := range GPUMachines {
			if strings.HasPrefix(pool.MachineType, GPUMachine) {
				if _, exists := usedGPUMachines[pool.MachineType]; !exists {
					orderedMachineTypes = append(orderedMachineTypes, pool.MachineType)
				}
				usedGPUMachines[pool.MachineType] = append(usedGPUMachines[pool.MachineType], pool.Name)
			}
		}
	}

	if len(usedGPUMachines) == 0 {
		return nil
	}

	var errorMsg strings.Builder
	errorMsg.WriteString("The following GPU machine types: ")

	for i, machineType := range orderedMachineTypes {
		if i > 0 {
			errorMsg.WriteString(", ")
		}
		errorMsg.WriteString(fmt.Sprintf("%s (used in worker node pools: %s)", machineType, strings.Join(usedGPUMachines[machineType], ", ")))
	}

	errorMsg.WriteString(" are not available for your account. For details, please contact your sales representative.")

	return fmt.Errorf("%s", errorMsg.String())
}

func checkUnsupportedMachines(regionsSupportingMachine internal.RegionsSupporter, region string, additionalWorkerNodePools []pkg.AdditionalWorkerNodePool) error {
	unsupportedMachines := make(map[string][]string)
	var orderedMachineTypes []string

	for _, pool := range additionalWorkerNodePools {
		if !regionsSupportingMachine.IsSupported(region, pool.MachineType) {
			if _, exists := unsupportedMachines[pool.MachineType]; !exists {
				orderedMachineTypes = append(orderedMachineTypes, pool.MachineType)
			}
			unsupportedMachines[pool.MachineType] = append(unsupportedMachines[pool.MachineType], pool.Name)
		}
	}

	if len(unsupportedMachines) == 0 {
		return nil
	}

	var errorMsg strings.Builder
	errorMsg.WriteString(fmt.Sprintf("In the region %s, the following machine types are not available: ", region))

	for i, machineType := range orderedMachineTypes {
		if i > 0 {
			errorMsg.WriteString("; ")
		}
		availableRegions := strings.Join(regionsSupportingMachine.SupportedRegions(machineType), ", ")
		errorMsg.WriteString(fmt.Sprintf("%s (used in: %s), it is supported in the %s", machineType, strings.Join(unsupportedMachines[machineType], ", "), availableRegions))
	}

	return fmt.Errorf("%s", errorMsg.String())
}

func checkAutoScalerConfiguration(additionalWorkerNodePools []pkg.AdditionalWorkerNodePool) error {
	var errors []string
	for _, additionalWorkerNodePool := range additionalWorkerNodePools {
		if err := additionalWorkerNodePool.Validate(); err != nil {
			errors = append(errors, err.Error())
		}
	}

	if len(errors) == 0 {
		return nil
	}

	message := "The following additionalWorkerPools have validation issues: "
	message = message + strings.Join(errors, "; ")
	message = message + "."

	return fmt.Errorf("%s", message)
}

func checkAvailableZones(
	log *slog.Logger,
	regionsSupportingMachine internal.RegionsSupporter,
	additionalWorkerNodePools []pkg.AdditionalWorkerNodePool,
	region, planID string,
	zonesDiscovery bool,
	discoveredZones map[string]int,
) error {
	HAUnavailableMachines := make(map[string][]string)
	var orderedMachineTypes []string

	for _, additionalWorkerNodePool := range additionalWorkerNodePools {
		if zonesDiscovery {
			if discoveredZones[additionalWorkerNodePool.MachineType] < 1 {
				return fmt.Errorf("In the %s, the %s machine type is not available.", region, additionalWorkerNodePool.MachineType)
			}
			if additionalWorkerNodePool.HAZones && discoveredZones[additionalWorkerNodePool.MachineType] < 3 {
				if _, exists := HAUnavailableMachines[additionalWorkerNodePool.MachineType]; !exists {
					orderedMachineTypes = append(orderedMachineTypes, additionalWorkerNodePool.MachineType)
				}
				HAUnavailableMachines[additionalWorkerNodePool.MachineType] = append(HAUnavailableMachines[additionalWorkerNodePool.MachineType], additionalWorkerNodePool.Name)
			}
		} else {
			zones, err := regionsSupportingMachine.AvailableZonesForAdditionalWorkers(additionalWorkerNodePool.MachineType, region, planID)
			if err != nil {
				log.Error(fmt.Sprintf("while getting available zones: %v", err))
				return fmt.Errorf(FailedToValidateZonesMsg)
			}
			if len(zones) > 0 && len(zones) < 3 && additionalWorkerNodePool.HAZones {
				if _, exists := HAUnavailableMachines[additionalWorkerNodePool.MachineType]; !exists {
					orderedMachineTypes = append(orderedMachineTypes, additionalWorkerNodePool.MachineType)
				}
				HAUnavailableMachines[additionalWorkerNodePool.MachineType] = append(HAUnavailableMachines[additionalWorkerNodePool.MachineType], additionalWorkerNodePool.Name)
			}
		}
	}

	if len(HAUnavailableMachines) == 0 {
		return nil
	}

	message := fmt.Sprintf("In the %s, the machine types: ", region)
	var machineTypeMessages []string
	for _, machineType := range orderedMachineTypes {
		pools := HAUnavailableMachines[machineType]
		machineTypeMessages = append(machineTypeMessages,
			fmt.Sprintf("%s (used in worker node pools: %s)", machineType, strings.Join(pools, ", ")))
	}
	message += strings.Join(machineTypeMessages, ", ")
	message += " are not available in 3 zones. If you want to use this machine types, set HA to false."

	return fmt.Errorf("%s", message)
}

// Rudimentary kubeconfig validation
func validateKubeconfig(kubeconfig string) error {
	config, err := clientcmd.Load([]byte(kubeconfig))
	if err != nil {
		return err
	}
	err = clientcmd.Validate(*config)
	if err != nil {
		return err
	}
	return nil
}

func (b *ProvisionEndpoint) extractERSContext(details domain.ProvisionDetails) (internal.ERSContext, error) {
	var ersContext internal.ERSContext
	err := json.Unmarshal(details.RawContext, &ersContext)
	if err != nil {
		return ersContext, fmt.Errorf("while decoding context: %w", err)
	}

	if ersContext.GlobalAccountID == "" {
		return ersContext, fmt.Errorf("global accountID parameter cannot be empty")
	}
	if ersContext.SubAccountID == "" {
		return ersContext, fmt.Errorf("subAccountID parameter cannot be empty")
	}
	if ersContext.UserID == "" {
		return ersContext, fmt.Errorf("UserID parameter cannot be empty")
	}
	ersContext.UserID = strings.ToLower(ersContext.UserID)

	return ersContext, nil
}

func (b *ProvisionEndpoint) extractInputParameters(details domain.ProvisionDetails) (pkg.ProvisioningParametersDTO, error) {
	var parameters pkg.ProvisioningParametersDTO
	err := json.Unmarshal(details.RawParameters, &parameters)
	if err != nil {
		return parameters, fmt.Errorf("while unmarshaling raw parameters: %w", err)
	}
	return parameters, nil
}

func (b *ProvisionEndpoint) handleExistingOperation(operation *internal.ProvisioningOperation, input internal.ProvisioningParameters) (domain.ProvisionedServiceSpec, error) {

	if !operation.ProvisioningParameters.IsEqual(input) {
		err := fmt.Errorf("provisioning operation already exist")
		msg := fmt.Sprintf("provisioning operation with InstanceID %s already exist", operation.InstanceID)
		return domain.ProvisionedServiceSpec{}, apiresponses.NewFailureResponse(err, http.StatusConflict, msg)
	}

	instance, err := b.instanceStorage.GetByID(operation.InstanceID)
	if err != nil {
		err := fmt.Errorf("cannot fetch instance for operation")
		msg := fmt.Sprintf("cannot fetch instance with ID: %s for operation woth ID: %s", operation.InstanceID, operation.ID)
		return domain.ProvisionedServiceSpec{}, apiresponses.NewFailureResponse(err, http.StatusConflict, msg)
	}

	return domain.ProvisionedServiceSpec{
		IsAsync:       true,
		OperationData: operation.ID,
		DashboardURL:  operation.DashboardURL,
		Metadata: domain.InstanceMetadata{
			Labels: ResponseLabels(*operation, *instance, b.config.URL, b.kcBuilder),
		},
	}, nil
}

func (b *ProvisionEndpoint) validator(details *domain.ProvisionDetails, provider pkg.CloudProvider, ctx context.Context) (*jsonschema.Schema, error) {
	platformRegion, _ := middleware.RegionFromContext(ctx)
	plans := b.schemaService.Plans(b.plansConfig, platformRegion, provider)
	plan := plans[details.PlanID]

	return validator.NewFromSchema(plan.Schemas.Instance.Create.Parameters)
}

func (b *ProvisionEndpoint) createDashboardURL(planID, instanceID string) string {
	if IsOwnClusterPlan(planID) {
		return b.dashboardConfig.LandscapeURL
	} else {
		return fmt.Sprintf("%s/?kubeconfigID=%s", b.dashboardConfig.LandscapeURL, instanceID)
	}
}

func validateCidr(cidr string) (*net.IPNet, error) {
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}
	// find cases like: 10.250.0.1/19
	if ipNet != nil {
		if !ipNet.IP.Equal(ip) {
			return nil, fmt.Errorf("%s must be valid canonical CIDR", ip)
		}
	}
	return ipNet, nil
}

func (b *ProvisionEndpoint) validateNetworking(parameters pkg.ProvisioningParametersDTO) error {
	var err, e error
	if len(parameters.Zones) > 4 {
		// the algorithm of creating AWS zone CIDRs does not work for more than 4 zones
		err = multierror.Append(err, fmt.Errorf("number of zones must not be greater than 4"))
	}
	if parameters.Networking == nil {
		return nil
	}

	var nodes, services, pods *net.IPNet
	if nodes, e = validateCidr(parameters.Networking.NodesCidr); e != nil {
		err = multierror.Append(err, fmt.Errorf("while parsing nodes CIDR: %w", e))
	}
	// error is handled before, in the validate CIDR
	cidr, _ := netip.ParsePrefix(parameters.Networking.NodesCidr)
	const maxSuffix = 23
	if cidr.Bits() > maxSuffix {
		err = multierror.Append(err, fmt.Errorf("the suffix of the node CIDR must not be greater than %d", maxSuffix))
	}

	if parameters.Networking.PodsCidr != nil {
		if pods, e = validateCidr(*parameters.Networking.PodsCidr); e != nil {
			err = multierror.Append(err, fmt.Errorf("while parsing pods CIDR: %w", e))
		}
	} else {
		_, pods, _ = net.ParseCIDR(networking.DefaultPodsCIDR)
	}
	if parameters.Networking.ServicesCidr != nil {
		if services, e = validateCidr(*parameters.Networking.ServicesCidr); e != nil {
			err = multierror.Append(err, fmt.Errorf("while parsing services CIDR: %w", e))
		}
	} else {
		_, services, _ = net.ParseCIDR(networking.DefaultServicesCIDR)
	}
	if err != nil {
		return err
	}

	for _, seed := range networking.GardenerSeedCIDRs {
		_, seedCidr, _ := net.ParseCIDR(seed)
		if e := validateOverlapping(*nodes, *seedCidr); e != nil {
			err = multierror.Append(err, fmt.Errorf("nodes CIDR must not overlap %s", seed))
		}
		if e := validateOverlapping(*services, *seedCidr); e != nil {
			err = multierror.Append(err, fmt.Errorf("services CIDR must not overlap %s", seed))
		}
		if e := validateOverlapping(*pods, *seedCidr); e != nil {
			err = multierror.Append(err, fmt.Errorf("pods CIDR must not overlap %s", seed))
		}
	}

	if err != nil {
		return err
	}

	if e := validateOverlapping(*nodes, *pods); e != nil {
		err = multierror.Append(err, fmt.Errorf("nodes CIDR must not overlap pods CIDR"))
	}
	if e := validateOverlapping(*nodes, *services); e != nil {
		err = multierror.Append(err, fmt.Errorf("nodes CIDR must not overlap serivces CIDR"))
	}
	if e := validateOverlapping(*services, *pods); e != nil {
		err = multierror.Append(err, fmt.Errorf("services CIDR must not overlap pods CIDR"))
	}

	return err
}

func (b *ProvisionEndpoint) monitorAdditionalProperties(instanceID string, ersContext internal.ERSContext, rawParameters json.RawMessage) {
	var parameters pkg.ProvisioningParametersDTO
	decoder := json.NewDecoder(bytes.NewReader(rawParameters))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&parameters); err == nil {
		return
	}
	if err := insertRequest(instanceID, filepath.Join(b.config.AdditionalPropertiesPath, additionalproperties.ProvisioningRequestsFileName), ersContext, rawParameters); err != nil {
		b.log.Error(fmt.Sprintf("failed to save provisioning request with additonal properties: %v", err))
	}
}

func (b *ProvisionEndpoint) validateColocationRegion(providerType, region string, supportedRegions []string, logger *slog.Logger) error {
	providerConfig := &internal.ProviderConfig{}
	if err := b.providerConfigProvider.Provide(providerType, providerConfig); err != nil {
		logger.Error(fmt.Sprintf("while loading %s provider config", providerType), "error", err)
		return fmt.Errorf("unable to load %s provider config", providerType)
	}
	supportedSeedRegions := b.filterOutUnsupportedSeedRegions(supportedRegions, providerConfig.SeedRegions)
	if !slices.Contains(supportedSeedRegions, region) {
		logger.Warn(fmt.Sprintf("missing seed region %s for provider %s", region, providerType))
		msg := fmt.Sprintf("Provider %s can have control planes in the following regions: %s", providerType, supportedSeedRegions)
		return fmt.Errorf("cannot colocate the control plane in the %s region. %s", region, msg)
	}

	return nil
}

func (b *ProvisionEndpoint) filterOutUnsupportedSeedRegions(supportedRegions, seedRegions []string) []string {
	supportedRegionsSet := make(map[string]struct{}, len(supportedRegions))
	for _, region := range supportedRegions {
		supportedRegionsSet[region] = struct{}{}
	}
	supportedSeedRegions := make([]string, 0)
	for _, seedRegion := range seedRegions {
		if _, ok := supportedRegionsSet[seedRegion]; ok {
			supportedSeedRegions = append(supportedSeedRegions, seedRegion)
		}
	}
	return supportedSeedRegions
}

func insertRequest(instanceID, filePath string, ersContext internal.ERSContext, rawParameters json.RawMessage) error {
	payload := map[string]interface{}{
		"globalAccountID": ersContext.GlobalAccountID,
		"subAccountID":    ersContext.SubAccountID,
		"instanceID":      instanceID,
		"payload":         rawParameters,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write payload: %w", err)
	}

	return nil
}

func validateOverlapping(n1 net.IPNet, n2 net.IPNet) error {

	if n1.Contains(n2.IP) || n2.Contains(n1.IP) {
		return fmt.Errorf("%s overlaps %s", n1.String(), n2.String())
	}

	return nil
}

func validateQuotaLimit(instanceStorage storage.Instances, quotaClient QuotaClient, subAccountID, planID string, update bool) error {
	instanceFilter := dbmodel.InstanceFilter{
		SubAccountIDs: []string{subAccountID},
		PlanIDs:       []string{planID},
	}
	_, _, usedQuota, err := instanceStorage.List(instanceFilter)
	if err != nil {
		return fmt.Errorf(
			"while listing instances for subaccount %s and plan ID %s: %w",
			subAccountID,
			planID,
			err,
		)
	}

	if usedQuota > 0 || update {
		assignedQuota, err := quotaClient.GetQuota(subAccountID, PlanNamesMapping[planID])
		if err != nil {
			return fmt.Errorf("Failed to get assigned quota for plan %s: %w.", PlanNamesMapping[planID], err)
		}

		if usedQuota >= assignedQuota {
			return fmt.Errorf("Kyma instances quota exceeded for plan %s. assignedQuota: %d, remainingQuota: 0. Contact your administrator.", PlanNamesMapping[planID], assignedQuota)
		}
	}

	return nil
}

func newAWSClient(
	ctx context.Context,
	log *slog.Logger,
	rulesService *rules.RulesService,
	gardenerClient *gardener.Client,
	awsClientFactory aws.ClientFactory,
	provisioningParameters internal.ProvisioningParameters,
	values internal.ProviderValues,
) (aws.Client, error) {
	log.Info("Zones discovery enabled, validating zone count using subscription secret")
	attr := &rules.ProvisioningAttributes{
		Plan:              PlanNamesMapping[provisioningParameters.PlanID],
		PlatformRegion:    provisioningParameters.PlatformRegion,
		HyperscalerRegion: values.Region,
		Hyperscaler:       values.ProviderType,
	}
	log.Info(fmt.Sprintf("matching provisioning attributes %q to filtering rule", attr))

	parsedRule, found := rulesService.MatchProvisioningAttributesWithValidRuleset(attr)
	if !found {
		return nil, fmt.Errorf("no matching rule for provisioning attributes %q", attr)
	}
	log.Info(fmt.Sprintf("matched rule: %q", parsedRule.Rule()))

	labelSelectorBuilder := subscriptions.NewLabelSelectorFromRuleset(parsedRule)
	labelSelector := labelSelectorBuilder.BuildAnySubscription()

	log.Info(fmt.Sprintf("getting secret binding with selector %q", labelSelector))
	secretBindings, err := gardenerClient.GetSecretBindings(labelSelector)
	if err != nil {
		return nil, fmt.Errorf("while getting secret bindings with selector %q: %w", labelSelector, err)
	}
	if secretBindings == nil || len(secretBindings.Items) == 0 {
		return nil, fmt.Errorf("while getting secret bindings with selector %q: %w", labelSelector, err)
	}
	secretBinding := gardener.NewSecretBinding(secretBindings.Items[0])

	log.Info(fmt.Sprintf("getting subscription secret with name %s/%s", secretBinding.GetSecretRefNamespace(), secretBinding.GetSecretRefName()))
	secret, err := gardenerClient.GetSecret(secretBinding.GetSecretRefNamespace(), secretBinding.GetSecretRefName())
	if err != nil {
		return nil, fmt.Errorf("unable to get secret %s/%s", secretBinding.GetSecretRefNamespace(), secretBinding.GetSecretRefName())
	}

	accessKeyID, secretAccessKey, err := aws.ExtractCredentials(secret)
	if err != nil {
		return nil, fmt.Errorf("failed to extract AWS credentials")
	}
	client, err := awsClientFactory.New(ctx, accessKeyID, secretAccessKey, values.Region)
	if err != nil {
		return nil, fmt.Errorf("unable to create AWS client")
	}

	return client, nil
}

func newAWSClientUsingCredentialsBinding(
	ctx context.Context,
	log *slog.Logger,
	rulesService *rules.RulesService,
	gardenerClient *gardener.Client,
	awsClientFactory aws.ClientFactory,
	provisioningParameters internal.ProvisioningParameters,
	values internal.ProviderValues,
) (aws.Client, error) {
	log.Info("Zones discovery enabled, validating zone count using subscription secret")
	attr := &rules.ProvisioningAttributes{
		Plan:              PlanNamesMapping[provisioningParameters.PlanID],
		PlatformRegion:    provisioningParameters.PlatformRegion,
		HyperscalerRegion: values.Region,
		Hyperscaler:       values.ProviderType,
	}
	log.Info(fmt.Sprintf("matching provisioning attributes %q to filtering rule", attr))

	parsedRule, found := rulesService.MatchProvisioningAttributesWithValidRuleset(attr)
	if !found {
		return nil, fmt.Errorf("no matching rule for provisioning attributes %q", attr)
	}
	log.Info(fmt.Sprintf("matched rule: %q", parsedRule.Rule()))

	labelSelectorBuilder := subscriptions.NewLabelSelectorFromRuleset(parsedRule)
	labelSelector := labelSelectorBuilder.BuildAnySubscription()

	log.Info(fmt.Sprintf("getting secret binding with selector %q", labelSelector))
	credentialsBindings, err := gardenerClient.GetCredentialsBindings(labelSelector)
	if err != nil {
		return nil, fmt.Errorf("while getting credentials bindings with selector %q: %w", labelSelector, err)
	}
	if credentialsBindings == nil || len(credentialsBindings.Items) == 0 {
		return nil, fmt.Errorf("while getting credentials bindings with selector %q: %w", labelSelector, err)
	}
	credentialsBinding := gardener.NewCredentialsBinding(credentialsBindings.Items[0])

	log.Info(fmt.Sprintf("getting subscription credentials with name %s/%s", credentialsBinding.GetSecretRefNamespace(), credentialsBinding.GetSecretRefName()))
	secret, err := gardenerClient.GetSecret(credentialsBinding.GetSecretRefNamespace(), credentialsBinding.GetSecretRefName())
	if err != nil {
		return nil, fmt.Errorf("unable to get secret %s/%s", credentialsBinding.GetSecretRefNamespace(), credentialsBinding.GetSecretRefName())
	}

	accessKeyID, secretAccessKey, err := aws.ExtractCredentials(secret)
	if err != nil {
		return nil, fmt.Errorf("failed to extract AWS credentials")
	}
	client, err := awsClientFactory.New(ctx, accessKeyID, secretAccessKey, values.Region)
	if err != nil {
		return nil, fmt.Errorf("unable to create AWS client")
	}

	return client, nil
}
