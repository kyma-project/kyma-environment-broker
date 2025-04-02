package broker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"strings"

	"github.com/kyma-project/kyma-environment-broker/internal/regionssupportingmachine"
	"github.com/kyma-project/kyma-environment-broker/internal/validator"
	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/kyma-project/kyma-environment-broker/internal/assuredworkloads"

	"github.com/kyma-project/kyma-environment-broker/internal/kubeconfig"
	"github.com/kyma-project/kyma-environment-broker/internal/whitelist"

	"github.com/kyma-project/kyma-environment-broker/internal/storage/dbmodel"

	"github.com/kyma-project/kyma-environment-broker/internal/networking"

	"github.com/hashicorp/go-multierror"

	"github.com/kyma-project/kyma-environment-broker/internal/euaccess"

	"k8s.io/client-go/tools/clientcmd"

	"github.com/google/uuid"
	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/dashboard"
	"github.com/kyma-project/kyma-environment-broker/internal/middleware"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/pivotal-cf/brokerapi/v12/domain/apiresponses"
)

//go:generate mockery --name=Queue --output=automock --outpkg=automock --case=underscore
//go:generate mockery --name=PlanValidator --output=automock --outpkg=automock --case=underscore

type (
	Queue interface {
		Add(operationId string)
	}

	PlanValidator interface {
		IsPlanSupport(planID string) bool
		GetDefaultOIDC() *pkg.OIDCConfigDTO
	}
)

type ProvisionEndpoint struct {
	config                  Config
	operationsStorage       storage.Operations
	instanceStorage         storage.Instances
	instanceArchivedStorage storage.InstancesArchived
	queue                   Queue
	builderFactory          PlanValidator
	enabledPlanIDs          map[string]struct{}
	plansConfig             PlansConfig
	planDefaults            PlanDefaults

	shootDomain       string
	shootProject      string
	shootDnsProviders gardener.DNSProvidersData

	dashboardConfig dashboard.Config
	kcBuilder       kubeconfig.KcBuilder

	freemiumWhiteList whitelist.Set

	convergedCloudRegionsProvider ConvergedCloudRegionProvider

	regionsSupportingMachine map[string][]string

	log *slog.Logger
}

const (
	CONVERGED_CLOUD_BLOCKED_MSG = "This offer is currently not available."
)

func NewProvision(cfg Config,
	gardenerConfig gardener.Config,
	operationsStorage storage.Operations,
	instanceStorage storage.Instances,
	instanceArchivedStorage storage.InstancesArchived,
	queue Queue,
	builderFactory PlanValidator,
	plansConfig PlansConfig,
	planDefaults PlanDefaults,
	log *slog.Logger,
	dashboardConfig dashboard.Config,
	kcBuilder kubeconfig.KcBuilder,
	freemiumWhitelist whitelist.Set,
	convergedCloudRegionsProvider ConvergedCloudRegionProvider,
	regionsSupportingMachine map[string][]string,
) *ProvisionEndpoint {
	enabledPlanIDs := map[string]struct{}{}
	for _, planName := range cfg.EnablePlans {
		id := PlanIDsMapping[planName]
		enabledPlanIDs[id] = struct{}{}
	}

	return &ProvisionEndpoint{
		config:                        cfg,
		operationsStorage:             operationsStorage,
		instanceStorage:               instanceStorage,
		instanceArchivedStorage:       instanceArchivedStorage,
		queue:                         queue,
		builderFactory:                builderFactory,
		log:                           log.With("service", "ProvisionEndpoint"),
		enabledPlanIDs:                enabledPlanIDs,
		plansConfig:                   plansConfig,
		shootDomain:                   gardenerConfig.ShootDomain,
		shootProject:                  gardenerConfig.Project,
		shootDnsProviders:             gardenerConfig.DNSProviders,
		planDefaults:                  planDefaults,
		dashboardConfig:               dashboardConfig,
		freemiumWhiteList:             freemiumWhitelist,
		kcBuilder:                     kcBuilder,
		convergedCloudRegionsProvider: convergedCloudRegionsProvider,
		regionsSupportingMachine:      regionsSupportingMachine,
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
		err := fmt.Errorf("No region specified in request.")
		return domain.ProvisionedServiceSpec{}, apiresponses.NewFailureResponse(err, http.StatusInternalServerError, "provisioning")
	}
	platformProvider, found := middleware.ProviderFromContext(ctx)
	if !found {
		err := fmt.Errorf("No provider specified in request.")
		return domain.ProvisionedServiceSpec{}, apiresponses.NewFailureResponse(err, http.StatusInternalServerError, "provisioning")
	}

	// validation of incoming input
	ersContext, parameters, err := b.validateAndExtract(details, platformProvider, ctx, logger)
	if err != nil {
		errMsg := fmt.Sprintf("[instanceID: %s] %s", instanceID, err)
		return domain.ProvisionedServiceSpec{}, apiresponses.NewFailureResponse(err, http.StatusBadRequest, errMsg)
	}

	if b.config.DisableSapConvergedCloud && details.PlanID == SapConvergedCloudPlanID {
		err := fmt.Errorf(CONVERGED_CLOUD_BLOCKED_MSG)
		return domain.ProvisionedServiceSpec{}, apiresponses.NewFailureResponse(err, http.StatusBadRequest, CONVERGED_CLOUD_BLOCKED_MSG)
	}

	provisioningParameters := internal.ProvisioningParameters{
		PlanID:           details.PlanID,
		ServiceID:        details.ServiceID,
		ErsContext:       ersContext,
		Parameters:       parameters,
		PlatformRegion:   region,
		PlatformProvider: platformProvider,
	}

	logger.Info(fmt.Sprintf("Starting provisioning runtime: Name=%s, GlobalAccountID=%s, SubAccountID=%s, PlatformRegion=%s, ProvisioningParameterts.Region=%s, ShootAndSeedSameRegion=%t, ProvisioningParameterts.MachineType=%s",
		parameters.Name, ersContext.GlobalAccountID, ersContext.SubAccountID, region, valueOfPtr(parameters.Region),
		valueOfBoolPtr(parameters.ShootAndSeedSameRegion), valueOfPtr(parameters.MachineType)))
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
			Labels: ResponseLabels(operation, instance, b.config.URL, b.config.EnableKubeconfigURLLabel, b.kcBuilder),
		},
	}, nil
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

func (b *ProvisionEndpoint) validateAndExtract(details domain.ProvisionDetails, provider pkg.CloudProvider, ctx context.Context, l *slog.Logger) (internal.ERSContext, pkg.ProvisioningParametersDTO, error) {
	var ersContext internal.ERSContext
	var parameters pkg.ProvisioningParametersDTO

	if details.ServiceID != KymaServiceID {
		return ersContext, parameters, fmt.Errorf("service_id not recognized")
	}
	if _, exists := b.enabledPlanIDs[details.PlanID]; !exists {
		return ersContext, parameters, fmt.Errorf("plan ID %q is not recognized", details.PlanID)
	}

	ersContext, err := b.extractERSContext(details)
	logger := l.With("globalAccountID", ersContext.GlobalAccountID)
	if err != nil {
		return ersContext, parameters, fmt.Errorf("while extracting ers context: %w", err)
	}

	parameters, err = b.extractInputParameters(details)
	if err != nil {
		return ersContext, parameters, fmt.Errorf("while extracting input parameters: %w", err)
	}
	defaults, err := b.planDefaults(details.PlanID, provider, parameters.Provider)
	if err != nil {
		return ersContext, parameters, fmt.Errorf("while obtaining plan defaults: %w", err)
	}

	if !regionssupportingmachine.IsSupported(b.regionsSupportingMachine, valueOfPtr(parameters.Region), valueOfPtr(parameters.MachineType)) {
		return ersContext, parameters, fmt.Errorf(
			"In the region %s, the machine type %s is not available, it is supported in the %v",
			valueOfPtr(parameters.Region),
			valueOfPtr(parameters.MachineType),
			strings.Join(regionssupportingmachine.SupportedRegions(b.regionsSupportingMachine, valueOfPtr(parameters.MachineType)), ", "),
		)
	}

	if err := b.validateNetworking(parameters); err != nil {
		return ersContext, parameters, err
	}

	var autoscalerMin, autoscalerMax int
	if defaults.GardenerConfig != nil {
		p := defaults.GardenerConfig
		autoscalerMin, autoscalerMax = p.AutoScalerMin, p.AutoScalerMax
	}
	if err := parameters.AutoScalerParameters.Validate(autoscalerMin, autoscalerMax); err != nil {
		return ersContext, parameters, apiresponses.NewFailureResponse(err, http.StatusUnprocessableEntity, err.Error())
	}
	if parameters.OIDC.IsProvided() {
		if err := parameters.OIDC.Validate(nil); err != nil {
			return ersContext, parameters, apiresponses.NewFailureResponse(err, http.StatusUnprocessableEntity, err.Error())
		}
	}

	if parameters.AdditionalWorkerNodePools != nil {
		if !supportsAdditionalWorkerNodePools(details.PlanID) {
			message := fmt.Sprintf("additional worker node pools are not supported for plan ID: %s", details.PlanID)
			return ersContext, parameters, apiresponses.NewFailureResponse(fmt.Errorf(message), http.StatusUnprocessableEntity, message)
		}
		if !AreNamesUnique(parameters.AdditionalWorkerNodePools) {
			message := "names of additional worker node pools must be unique"
			return ersContext, parameters, apiresponses.NewFailureResponse(fmt.Errorf(message), http.StatusUnprocessableEntity, message)
		}
		for _, additionalWorkerNodePool := range parameters.AdditionalWorkerNodePools {
			if err := additionalWorkerNodePool.Validate(); err != nil {
				return ersContext, parameters, apiresponses.NewFailureResponse(err, http.StatusUnprocessableEntity, err.Error())
			}
		}
		if isExternalCustomer(ersContext) {
			if err := checkGPUMachinesUsage(parameters.AdditionalWorkerNodePools); err != nil {
				return ersContext, parameters, apiresponses.NewFailureResponse(err, http.StatusUnprocessableEntity, err.Error())
			}
		}
		if err := checkUnsupportedMachines(b.regionsSupportingMachine, valueOfPtr(parameters.Region), parameters.AdditionalWorkerNodePools); err != nil {
			return ersContext, parameters, apiresponses.NewFailureResponse(err, http.StatusUnprocessableEntity, err.Error())
		}
	}

	planValidator, err := b.validator(&details, provider, ctx)
	if err != nil {
		return ersContext, parameters, fmt.Errorf("while creating plan validator: %w", err)
	}

	var rawParameters any
	if err = json.Unmarshal(details.RawParameters, &rawParameters); err != nil {
		return ersContext, parameters, fmt.Errorf("while unmarshaling raw parameters: %w", err)
	}

	if err = planValidator.Validate(rawParameters); err != nil {
		return ersContext, parameters, fmt.Errorf("while validating input parameters: %s", validator.FormatError(err))
	}

	// EU Access
	if isEuRestrictedAccess(ctx) {
		logger.Info("EU Access restricted instance creation")
	}

	parameters.LicenceType = b.determineLicenceType(details.PlanID)

	found := b.builderFactory.IsPlanSupport(details.PlanID)
	if !found {
		return ersContext, parameters, fmt.Errorf("the plan ID not known, planID: %s", details.PlanID)
	}

	if IsOwnClusterPlan(details.PlanID) {
		decodedKubeconfig, err := base64.StdEncoding.DecodeString(parameters.Kubeconfig)
		if err != nil {
			return ersContext, parameters, fmt.Errorf("while decoding kubeconfig: %w", err)
		}
		parameters.Kubeconfig = string(decodedKubeconfig)
		err = validateKubeconfig(parameters.Kubeconfig)
		if err != nil {
			return ersContext, parameters, fmt.Errorf("while validating kubeconfig: %w", err)
		}
	}

	if IsTrialPlan(details.PlanID) && parameters.Region != nil && *parameters.Region != "" {
		_, valid := validRegionsForTrial[TrialCloudRegion(*parameters.Region)]
		if !valid {
			return ersContext, parameters, fmt.Errorf("invalid region specified in request for trial")
		}
	}

	if IsTrialPlan(details.PlanID) && b.config.OnlySingleTrialPerGA {
		count, err := b.instanceStorage.GetNumberOfInstancesForGlobalAccountID(ersContext.GlobalAccountID)
		if err != nil {
			return ersContext, parameters, fmt.Errorf("while checking if a trial Kyma instance exists for given global account: %w", err)
		}

		if count > 0 {
			logger.Info("Provisioning Trial SKR rejected, such instance was already created for this Global Account")
			return ersContext, parameters, fmt.Errorf("trial Kyma was created for the global account, but there is only one allowed")
		}
	}

	if IsFreemiumPlan(details.PlanID) && b.config.OnlyOneFreePerGA && whitelist.IsNotWhitelisted(ersContext.GlobalAccountID, b.freemiumWhiteList) {
		count, err := b.instanceArchivedStorage.TotalNumberOfInstancesArchivedForGlobalAccountID(ersContext.GlobalAccountID, FreemiumPlanID)
		if err != nil {
			return ersContext, parameters, fmt.Errorf("while checking if a free Kyma instance existed for given global account: %w", err)
		}
		if count > 0 {
			logger.Info("Provisioning Free SKR rejected, such instance was already created for this Global Account")
			return ersContext, parameters, fmt.Errorf("provisioning request rejected, you have already used the available free service plan quota in this global account")
		}

		instanceFilter := dbmodel.InstanceFilter{
			GlobalAccountIDs: []string{ersContext.GlobalAccountID},
			PlanIDs:          []string{FreemiumPlanID},
			States:           []dbmodel.InstanceState{dbmodel.InstanceSucceeded},
		}
		_, _, count, err = b.instanceStorage.List(instanceFilter)
		if err != nil {
			return ersContext, parameters, fmt.Errorf("while checking if a free Kyma instance existed for given global account: %w", err)
		}
		if count > 0 {
			logger.Info("Provisioning Free SKR rejected, such instance was already created for this Global Account")
			return ersContext, parameters, fmt.Errorf("provisioning request rejected, you have already used the available free service plan quota in this global account")
		}
	}

	return ersContext, parameters, nil
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

func isExternalCustomer(ersContext internal.ERSContext) bool {
	return *ersContext.DisableEnterprisePolicyFilter()
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

	return fmt.Errorf(errorMsg.String())
}

func checkUnsupportedMachines(regionsSupportingMachine map[string][]string, region string, additionalWorkerNodePools []pkg.AdditionalWorkerNodePool) error {
	unsupportedMachines := make(map[string][]string)
	var orderedMachineTypes []string

	for _, pool := range additionalWorkerNodePools {
		if !regionssupportingmachine.IsSupported(regionsSupportingMachine, region, pool.MachineType) {
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
		availableRegions := strings.Join(regionssupportingmachine.SupportedRegions(regionsSupportingMachine, machineType), ", ")
		errorMsg.WriteString(fmt.Sprintf("%s (used in: %s), it is supported in the %s", machineType, strings.Join(unsupportedMachines[machineType], ", "), availableRegions))
	}

	return fmt.Errorf(errorMsg.String())
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
	if !b.config.UseAdditionalOIDCSchema && parameters.OIDC != nil {
		if parameters.OIDC.OIDCConfigDTO != nil && parameters.OIDC.OIDCConfigDTO.RequiredClaims != nil {
			parameters.OIDC.OIDCConfigDTO.RequiredClaims = nil
		}
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
			Labels: ResponseLabels(*operation, *instance, b.config.URL, b.config.EnableKubeconfigURLLabel, b.kcBuilder),
		},
	}, nil
}

func (b *ProvisionEndpoint) determineLicenceType(planId string) *string {
	if planId == AzureLitePlanID || IsTrialPlan(planId) {
		return ptr.String(internal.LicenceTypeLite)
	}

	return nil
}

func (b *ProvisionEndpoint) validator(details *domain.ProvisionDetails, provider pkg.CloudProvider, ctx context.Context) (*jsonschema.Schema, error) {
	platformRegion, _ := middleware.RegionFromContext(ctx)
	plans := Plans(b.plansConfig, provider, nil, b.config.IncludeAdditionalParamsInSchema, euaccess.IsEURestrictedAccess(platformRegion), b.config.UseSmallerMachineTypes, b.config.EnableShootAndSeedSameRegion, b.convergedCloudRegionsProvider.GetRegions(platformRegion), assuredworkloads.IsKSA(platformRegion), b.config.UseAdditionalOIDCSchema)
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

func validateOverlapping(n1 net.IPNet, n2 net.IPNet) error {

	if n1.Contains(n2.IP) || n2.Contains(n1.IP) {
		return fmt.Errorf("%s overlaps %s", n1.String(), n2.String())
	}

	return nil
}
