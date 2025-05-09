package broker

import (
	"io"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal/provider/configuration"
	"github.com/pivotal-cf/brokerapi/v12/domain"
)

type SchemaService struct {
	planSpec          *configuration.PlanSpecifications
	providerSpec      *configuration.ProviderSpec
	defaultOIDCConfig *pkg.OIDCConfigDTO

	ingressFilteringFeatureFlag bool
	ingressFilteringPlans       EnablePlans

	cfg Config
}

func NewSchemaService(providerConfig io.Reader, planConfig io.Reader, defaultOIDCConfig *pkg.OIDCConfigDTO, cfg Config, ingressFilteringEnabled bool, ingressFilteringPlans EnablePlans) (*SchemaService, error) {
	planSpec, err := configuration.NewPlanSpecifications(planConfig)
	if err != nil {
		return nil, err
	}
	providerSpec, err := configuration.NewProviderSpec(providerConfig)
	if err != nil {
		return nil, err
	}

	return &SchemaService{
		planSpec:                    planSpec,
		providerSpec:                providerSpec,
		defaultOIDCConfig:           defaultOIDCConfig,
		cfg:                         cfg,
		ingressFilteringFeatureFlag: ingressFilteringEnabled,
		ingressFilteringPlans:       ingressFilteringPlans,
	}, nil
}

func (s *SchemaService) Plans(plans PlansConfig, platformRegion string, cp pkg.CloudProvider) map[string]domain.ServicePlan {
	awsCreateSchema := s.AWSSchema(platformRegion, false)
	awsUpdateSchema := s.AWSSchema(platformRegion, true)
	gcpCreateSchema := s.GCPSchema(platformRegion, false)
	gcpUpdateSchema := s.GCPSchema(platformRegion, true)
	azureCreateSchema := s.AzureSchema(platformRegion, false)
	azureUpdateSchema := s.AzureSchema(platformRegion, true)
	azureLiteCreateSchema := s.AzureLiteSchema(platformRegion, false)
	azureLiteUpdateSchema := s.AzureLiteSchema(platformRegion, true)
	sapConvergedCloudCreateSchema := s.SapConvergedCloudSchema(platformRegion, false)
	sapConvergedCloudUpdateSchema := s.SapConvergedCloudSchema(platformRegion, true)
	freemiumCreateSchema := s.FreeSchema(cp, platformRegion, false)
	freemiumUpdateSchema := s.FreeSchema(cp, platformRegion, true)
	trialCreateSchema := s.TrialSchema(false)
	trialUpdateSchema := s.TrialSchema(true)
	ownClusterCreateSchema := s.OwnClusterSchema(false)
	ownClusterUpdateSchema := s.OwnClusterSchema(true)
	buildRuntimeAWSCreateSchema := s.BuildRuntimeAWSSchema(platformRegion, false)
	buildRuntimeAWSUpdateSchema := s.BuildRuntimeAWSSchema(platformRegion, true)
	buildRuntimeGCPCreateSchema := s.BuildRuntimeGcpSchema(platformRegion, false)
	buildRuntimeGCPUpdateSchema := s.BuildRuntimeGcpSchema(platformRegion, true)
	buildRuntimeAzureCreateSchema := s.BuildRuntimeAzureSchema(platformRegion, false)
	buildRuntimeAzureUpdateSchema := s.BuildRuntimeAzureSchema(platformRegion, true)
	previewCreateSchema := s.PreviewSchema(platformRegion, false)
	previewUpdateSchema := s.PreviewSchema(platformRegion, true)

	outputPlans := map[string]domain.ServicePlan{
		AWSPlanID:               defaultServicePlan(AWSPlanID, AWSPlanName, plans, awsCreateSchema, awsUpdateSchema),
		GCPPlanID:               defaultServicePlan(GCPPlanID, GCPPlanName, plans, gcpCreateSchema, gcpUpdateSchema),
		AzurePlanID:             defaultServicePlan(AzurePlanID, AzurePlanName, plans, azureCreateSchema, azureUpdateSchema),
		AzureLitePlanID:         defaultServicePlan(AzureLitePlanID, AzureLitePlanName, plans, azureLiteCreateSchema, azureLiteUpdateSchema),
		FreemiumPlanID:          defaultServicePlan(FreemiumPlanID, FreemiumPlanName, plans, freemiumCreateSchema, freemiumUpdateSchema),
		TrialPlanID:             defaultServicePlan(TrialPlanID, TrialPlanName, plans, trialCreateSchema, trialUpdateSchema),
		OwnClusterPlanID:        defaultServicePlan(OwnClusterPlanID, OwnClusterPlanName, plans, ownClusterCreateSchema, ownClusterUpdateSchema),
		PreviewPlanID:           defaultServicePlan(PreviewPlanID, PreviewPlanName, plans, previewCreateSchema, previewUpdateSchema),
		BuildRuntimeAWSPlanID:   defaultServicePlan(BuildRuntimeAWSPlanID, BuildRuntimeAWSPlanName, plans, buildRuntimeAWSCreateSchema, buildRuntimeAWSUpdateSchema),
		BuildRuntimeGCPPlanID:   defaultServicePlan(BuildRuntimeGCPPlanID, BuildRuntimeGCPPlanName, plans, buildRuntimeGCPCreateSchema, buildRuntimeGCPUpdateSchema),
		BuildRuntimeAzurePlanID: defaultServicePlan(BuildRuntimeAzurePlanID, BuildRuntimeAzurePlanName, plans, buildRuntimeAzureCreateSchema, buildRuntimeAzureUpdateSchema),
		SapConvergedCloudPlanID: defaultServicePlan(SapConvergedCloudPlanID, SapConvergedCloudPlanName, plans, sapConvergedCloudCreateSchema, sapConvergedCloudUpdateSchema),
	}

	return outputPlans
}

func (s *SchemaService) AzureSchema(platformRegion string, update bool) *map[string]interface{} {
	regions := s.planSpec.Regions("azure", platformRegion)
	flags := s.createFlags(AzurePlanName)

	properties := NewProvisioningProperties(AzureMachinesDisplay(false),
		AzureMachinesDisplay(true),
		s.providerSpec.RegionDisplayNames(pkg.Azure, regions),
		AzureMachinesNames(false),
		AzureMachinesNames(true),
		regions,
		update,
		flags.disabledMachineTypeUpdate,
	)
	return createSchemaWithProperties(properties, s.defaultOIDCConfig, update, requiredSchemaProperties(), flags)
}

func (s *SchemaService) AWSSchema(platformRegion string, update bool) *map[string]interface{} {
	regions := s.planSpec.Regions("aws", platformRegion)
	flags := s.createFlags(AWSPlanName)

	properties := NewProvisioningProperties(
		AwsMachinesDisplay(false),
		AwsMachinesDisplay(true),
		s.providerSpec.RegionDisplayNames(pkg.AWS, regions),
		AwsMachinesNames(false),
		AwsMachinesNames(true),
		regions,
		update,
		flags.disabledMachineTypeUpdate,
	)
	return createSchemaWithProperties(properties, s.defaultOIDCConfig, update, requiredSchemaProperties(), flags)
}

func (s *SchemaService) PreviewSchema(platformRegion string, update bool) *map[string]interface{} {
	regions := s.planSpec.Regions("preview", platformRegion)
	flags := s.createFlags(PreviewPlanName)

	properties := NewProvisioningProperties(
		AwsMachinesDisplay(false),
		AwsMachinesDisplay(true),
		s.providerSpec.RegionDisplayNames(pkg.AWS, regions),
		AwsMachinesNames(false),
		AwsMachinesNames(true),
		regions,
		update,
		flags.disabledMachineTypeUpdate,
	)
	return createSchemaWithProperties(properties, s.defaultOIDCConfig, update, requiredSchemaProperties(), flags)
}

func (s *SchemaService) BuildRuntimeAWSSchema(platformRegion string, update bool) *map[string]interface{} {
	regions := s.planSpec.Regions("build-runtime-aws", platformRegion)
	flags := s.createFlags(BuildRuntimeAWSPlanName)

	properties := NewProvisioningProperties(
		AwsMachinesDisplay(false),
		AwsMachinesDisplay(true),
		s.providerSpec.RegionDisplayNames(pkg.AWS, regions),
		AwsMachinesNames(false),
		AwsMachinesNames(true),
		regions,
		update,
		flags.disabledMachineTypeUpdate,
	)
	return createSchemaWithProperties(properties, s.defaultOIDCConfig, update, requiredSchemaProperties(), flags)
}

func (s *SchemaService) GCPSchema(platformRegion string, update bool) *map[string]interface{} {
	regions := s.planSpec.Regions("gcp", platformRegion)
	flags := s.createFlags(GCPPlanName)

	properties := NewProvisioningProperties(
		GcpMachinesDisplay(false),
		GcpMachinesDisplay(true),
		s.providerSpec.RegionDisplayNames(pkg.GCP, regions),
		GcpMachinesNames(false),
		GcpMachinesNames(true),
		regions,
		update,
		flags.disabledMachineTypeUpdate,
	)
	return createSchemaWithProperties(properties, s.defaultOIDCConfig, update, requiredSchemaProperties(), flags)
}

func (s *SchemaService) BuildRuntimeGcpSchema(platformRegion string, update bool) *map[string]interface{} {
	regions := s.planSpec.Regions(BuildRuntimeGCPPlanName, platformRegion)
	flags := s.createFlags(BuildRuntimeGCPPlanName)

	properties := NewProvisioningProperties(
		GcpMachinesDisplay(false),
		GcpMachinesDisplay(true),
		s.providerSpec.RegionDisplayNames(pkg.GCP, regions),
		GcpMachinesNames(false),
		GcpMachinesNames(true),
		regions,
		update,
		flags.disabledMachineTypeUpdate,
	)
	return createSchemaWithProperties(properties, s.defaultOIDCConfig, update, requiredSchemaProperties(), flags)
}

func (s *SchemaService) BuildRuntimeAzureSchema(platformRegion string, update bool) *map[string]interface{} {
	regions := s.planSpec.Regions(BuildRuntimeAzurePlanName, platformRegion)
	flags := s.createFlags(BuildRuntimeAzurePlanName)

	properties := NewProvisioningProperties(
		AzureMachinesDisplay(false),
		AzureMachinesDisplay(true),
		s.providerSpec.RegionDisplayNames(pkg.Azure, regions),
		AzureMachinesNames(false),
		AzureMachinesNames(true),
		regions,
		update,
		flags.disabledMachineTypeUpdate,
	)
	return createSchemaWithProperties(properties, s.defaultOIDCConfig, update, requiredSchemaProperties(), flags)
}

func (s *SchemaService) SapConvergedCloudSchema(platformRegion string, update bool) *map[string]interface{} {
	regions := s.planSpec.Regions("sap-converged-cloud", platformRegion)
	flags := s.createFlags(SapConvergedCloudPlanName)

	properties := NewProvisioningProperties(
		SapConvergedCloudMachinesDisplay(),
		SapConvergedCloudMachinesDisplay(),
		s.providerSpec.RegionDisplayNames(pkg.SapConvergedCloud, regions),
		SapConvergedCloudMachinesNames(),
		SapConvergedCloudMachinesNames(),
		regions,
		update,
		flags.disabledMachineTypeUpdate,
	)
	return createSchemaWithProperties(properties, s.defaultOIDCConfig, update, requiredSchemaProperties(), flags)
}

func (s *SchemaService) AzureLiteSchema(platformRegion string, update bool) *map[string]interface{} {
	regions := s.planSpec.Regions("azure", platformRegion)
	flags := s.createFlags(AzureLitePlanName)

	properties := NewProvisioningProperties(
		AzureLiteMachinesDisplay(),
		AzureLiteMachinesDisplay(),
		s.providerSpec.RegionDisplayNames(pkg.Azure, regions),
		AzureLiteMachinesNames(),
		AzureLiteMachinesNames(),
		regions,
		update,
		flags.disabledMachineTypeUpdate,
	)
	properties.AutoScalerMax.Minimum = 2
	properties.AutoScalerMin.Minimum = 2
	properties.AutoScalerMax.Maximum = 40

	properties.AdditionalWorkerNodePools.Items.Properties.HAZones = nil
	properties.AdditionalWorkerNodePools.Items.ControlsOrder = removeString(properties.AdditionalWorkerNodePools.Items.ControlsOrder, "haZones")
	properties.AdditionalWorkerNodePools.Items.Required = removeString(properties.AdditionalWorkerNodePools.Items.Required, "haZones")
	properties.AdditionalWorkerNodePools.Items.Properties.AutoScalerMin.Default = 2
	properties.AdditionalWorkerNodePools.Items.Properties.AutoScalerMax.Default = 10
	properties.AdditionalWorkerNodePools.Items.Properties.AutoScalerMax.Maximum = 40

	if !update {
		properties.AutoScalerMax.Default = 10
		properties.AutoScalerMin.Default = 2
	}

	return createSchemaWithProperties(properties, s.defaultOIDCConfig, update, requiredSchemaProperties(), flags)
}

func (s *SchemaService) FreeSchema(provider pkg.CloudProvider, platformRegion string, update bool) *map[string]interface{} {
	var regions []string
	var regionsDisplayNames map[string]string
	switch provider {
	case pkg.Azure:
		regions = s.planSpec.Regions("azure", platformRegion)
		regionsDisplayNames = s.providerSpec.RegionDisplayNames(pkg.Azure, regions)
	default: // AWS and other BTP regions
		regions = s.planSpec.Regions("aws", platformRegion)
		regionsDisplayNames = s.providerSpec.RegionDisplayNames(pkg.AWS, regions)
	}
	flags := s.createFlags(FreemiumPlanName)
	flags.shootAndSeedFeatureEnabled = false

	properties := ProvisioningProperties{
		Name: NameProperty(),
		Region: &Type{
			Type:            "string",
			Enum:            ToInterfaceSlice(regions),
			MinLength:       1,
			EnumDisplayName: regionsDisplayNames,
		},
	}
	if !update {
		properties.Networking = NewNetworkingSchema()
		properties.Modules = NewModulesSchema()
	}

	return createSchemaWithProperties(properties, s.defaultOIDCConfig, update, requiredSchemaProperties(), flags)
}

func (s *SchemaService) TrialSchema(update bool) *map[string]interface{} {
	flags := s.createFlags(TrialPlanName)
	flags.shootAndSeedFeatureEnabled = false

	properties := ProvisioningProperties{
		Name: NameProperty(),
	}

	if !update {
		properties.Modules = NewModulesSchema()
	}

	if update && !flags.includeAdditionalParameters {
		return empty()
	}

	return createSchemaWithProperties(properties, s.defaultOIDCConfig, update, requiredTrialSchemaProperties(), flags)
}

func (s *SchemaService) OwnClusterSchema(update bool) *map[string]interface{} {
	properties := ProvisioningProperties{
		Name:        NameProperty(),
		ShootName:   ShootNameProperty(),
		ShootDomain: ShootDomainProperty(),
		UpdateProperties: UpdateProperties{
			Kubeconfig: KubeconfigProperty(),
		},
	}

	if update {
		return createSchemaWith(properties.UpdateProperties, []string{})
	} else {
		properties.Modules = NewModulesSchema()
		return createSchemaWith(properties, requiredOwnClusterSchemaProperties())
	}
}

func (s *SchemaService) createFlags(planName string) ControlFlagsObject {
	return ControlFlagsObject{
		includeAdditionalParameters: s.cfg.IncludeAdditionalParamsInSchema,
		useAdditionalOIDCSchema:     s.cfg.UseAdditionalOIDCSchema,
		shootAndSeedFeatureEnabled:  s.cfg.EnableShootAndSeedSameRegion,
		ingressFilteringEnabled:     s.ingressFilteringFeatureFlag && s.ingressFilteringPlans.Contains(planName),
		disabledMachineTypeUpdate:   s.cfg.DisableMachineTypeUpdate,
	}
}
