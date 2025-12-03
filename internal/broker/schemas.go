package broker

import (
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal/config"
	"github.com/kyma-project/kyma-environment-broker/internal/provider/configuration"
	"github.com/pivotal-cf/brokerapi/v12/domain"
)

type SchemaService struct {
	planSpec          *configuration.PlanSpecifications
	providerSpec      *configuration.ProviderSpec
	defaultOIDCConfig *pkg.OIDCConfigDTO

	ingressFilteringPlans EnablePlans

	cfg             Config
	channelResolver config.ChannelResolver
}

func NewSchemaService(providerSpec *configuration.ProviderSpec, planSpec *configuration.PlanSpecifications, defaultOIDCConfig *pkg.OIDCConfigDTO, cfg Config, ingressFilteringPlans EnablePlans, channelResolver config.ChannelResolver) *SchemaService {
	return &SchemaService{
		planSpec:              planSpec,
		providerSpec:          providerSpec,
		defaultOIDCConfig:     defaultOIDCConfig,
		cfg:                   cfg,
		ingressFilteringPlans: ingressFilteringPlans,
		channelResolver:       channelResolver,
	}
}

func (s *SchemaService) Validate() error {
	for planName, regions := range s.planSpec.AllRegionsByPlan() {
		var provider pkg.CloudProvider
		switch planName {
		case AWSPlanName, BuildRuntimeAWSPlanName, PreviewPlanName:
			provider = pkg.AWS
		case GCPPlanName, BuildRuntimeGCPPlanName:
			provider = pkg.GCP
		case AzurePlanName, BuildRuntimeAzurePlanName, AzureLitePlanName:
			provider = pkg.Azure
		case SapConvergedCloudPlanName:
			provider = pkg.SapConvergedCloud
		case AlicloudPlanName:
			provider = pkg.Alicloud
		default:
			continue
		}
		for _, region := range regions {
			err := s.providerSpec.Validate(provider, region)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *SchemaService) Plans(plans PlansConfig, platformRegion string, cp pkg.CloudProvider) map[string]domain.ServicePlan {

	outputPlans := map[string]domain.ServicePlan{}

	if createSchema, updateSchema, available := s.AWSSchemas(platformRegion); available {
		outputPlans[AWSPlanID] = s.defaultServicePlan(AWSPlanID, AWSPlanName, plans, createSchema, updateSchema)
	}
	if createSchema, updateSchema, available := s.GCPSchemas(platformRegion); available {
		outputPlans[GCPPlanID] = s.defaultServicePlan(GCPPlanID, GCPPlanName, plans, createSchema, updateSchema)
	}
	if createSchema, updateSchema, available := s.AzureSchemas(platformRegion); available {
		outputPlans[AzurePlanID] = s.defaultServicePlan(AzurePlanID, AzurePlanName, plans, createSchema, updateSchema)
	}
	if createSchema, updateSchema, available := s.SapConvergedCloudSchemas(platformRegion); available {
		outputPlans[SapConvergedCloudPlanID] = s.defaultServicePlan(SapConvergedCloudPlanID, SapConvergedCloudPlanName, plans, createSchema, updateSchema)
	}
	if createSchema, updateSchema, available := s.AlicloudSchemas(platformRegion); available {
		outputPlans[AlicloudPlanID] = s.defaultServicePlan(AlicloudPlanID, AlicloudPlanName, plans, createSchema, updateSchema)
	}
	if createSchema, updateSchema, available := s.PreviewSchemas(platformRegion); available {
		outputPlans[PreviewPlanID] = s.defaultServicePlan(PreviewPlanID, PreviewPlanName, plans, createSchema, updateSchema)
	}
	if createSchema, updateSchema, available := s.BuildRuntimeAWSSchemas(platformRegion); available {
		outputPlans[BuildRuntimeAWSPlanID] = s.defaultServicePlan(BuildRuntimeAWSPlanID, BuildRuntimeAWSPlanName, plans, createSchema, updateSchema)
	}
	if createSchema, updateSchema, available := s.BuildRuntimeGcpSchemas(platformRegion); available {
		outputPlans[BuildRuntimeGCPPlanID] = s.defaultServicePlan(BuildRuntimeGCPPlanID, BuildRuntimeGCPPlanName, plans, createSchema, updateSchema)
	}
	if createSchema, updateSchema, available := s.BuildRuntimeAzureSchemas(platformRegion); available {
		outputPlans[BuildRuntimeAzurePlanID] = s.defaultServicePlan(BuildRuntimeAzurePlanID, BuildRuntimeAzurePlanName, plans, createSchema, updateSchema)
	}
	if azureLiteCreateSchema, azureLiteUpdateSchema, available := s.AzureLiteSchemas(platformRegion); available {
		outputPlans[AzureLitePlanID] = s.defaultServicePlan(AzureLitePlanID, AzureLitePlanName, plans, azureLiteCreateSchema, azureLiteUpdateSchema)
	}
	if freemiumCreateSchema, freemiumUpdateSchema, available := s.FreeSchemas(cp, platformRegion); available {
		outputPlans[FreemiumPlanID] = s.defaultServicePlan(FreemiumPlanID, FreemiumPlanName, plans, freemiumCreateSchema, freemiumUpdateSchema)
	}

	trialCreateSchema := s.TrialSchema(false)
	trialUpdateSchema := s.TrialSchema(true)
	outputPlans[TrialPlanID] = s.defaultServicePlan(TrialPlanID, TrialPlanName, plans, trialCreateSchema, trialUpdateSchema)

	ownClusterCreateSchema := s.OwnClusterSchema(false)
	ownClusterUpdateSchema := s.OwnClusterSchema(true)
	outputPlans[OwnClusterPlanID] = s.defaultServicePlan(OwnClusterPlanID, OwnClusterPlanName, plans, ownClusterCreateSchema, ownClusterUpdateSchema)

	return outputPlans
}

func (s *SchemaService) defaultServicePlan(id, name string, plans PlansConfig, createParams, updateParams *map[string]interface{}) domain.ServicePlan {
	updatable := s.planSpec.IsUpgradable(name) && s.cfg.EnablePlanUpgrades
	servicePlan := domain.ServicePlan{
		ID:          id,
		Name:        name,
		Description: defaultDescription(name, plans),
		Metadata:    defaultMetadata(name, plans),
		Schemas: &domain.ServiceSchemas{
			Instance: domain.ServiceInstanceSchema{
				Create: domain.Schema{
					Parameters: *createParams,
				},
				Update: domain.Schema{
					Parameters: *updateParams,
				},
			},
		},
		PlanUpdatable: &updatable,
	}

	return servicePlan
}

func (s *SchemaService) planSchemas(cp pkg.CloudProvider, planName, platformRegion string) (create, update *map[string]interface{}, available bool) {
	regions := s.planSpec.Regions(planName, platformRegion)
	if len(regions) == 0 {
		return nil, nil, false
	}
	machines := s.planSpec.RegularMachines(planName)
	if len(machines) == 0 {
		return nil, nil, false
	}
	regularAndAdditionalMachines := append(machines, s.planSpec.AdditionalMachines(planName)...)
	flags := s.createFlags(planName)

	createProperties := NewProvisioningProperties(
		s.providerSpec.MachineDisplayNames(cp, machines),
		s.providerSpec.MachineDisplayNames(cp, regularAndAdditionalMachines),
		s.providerSpec.RegionDisplayNames(cp, regions),
		machines,
		regularAndAdditionalMachines,
		regions,
		false,
		flags.rejectUnsupportedParameters,
		s.providerSpec,
		cp,
		s.cfg.DualStackDocsURL,
		planName,
		s.channelResolver,
	)
	updateProperties := NewProvisioningProperties(
		s.providerSpec.MachineDisplayNames(cp, machines),
		s.providerSpec.MachineDisplayNames(cp, regularAndAdditionalMachines),
		s.providerSpec.RegionDisplayNames(cp, regions),
		machines,
		regularAndAdditionalMachines,
		regions,
		true,
		flags.rejectUnsupportedParameters,
		s.providerSpec,
		cp,
		s.cfg.DualStackDocsURL,
		planName,
		s.channelResolver,
	)
	return createSchemaWithProperties(createProperties, s.defaultOIDCConfig, false, requiredSchemaProperties(), flags),
		createSchemaWithProperties(updateProperties, s.defaultOIDCConfig, true, requiredSchemaProperties(), flags), true
}

func (s *SchemaService) AzureSchemas(platformRegion string) (create, update *map[string]interface{}, available bool) {
	return s.planSchemas(pkg.Azure, AzurePlanName, platformRegion)
}

func (s *SchemaService) BuildRuntimeAzureSchemas(platformRegion string) (create, update *map[string]interface{}, available bool) {
	return s.planSchemas(pkg.Azure, BuildRuntimeAzurePlanName, platformRegion)
}

func (s *SchemaService) AWSSchemas(platformRegion string) (create, update *map[string]interface{}, available bool) {
	return s.planSchemas(pkg.AWS, AWSPlanName, platformRegion)
}

func (s *SchemaService) BuildRuntimeAWSSchemas(platformRegion string) (create, update *map[string]interface{}, available bool) {
	return s.planSchemas(pkg.AWS, BuildRuntimeAWSPlanName, platformRegion)
}

func (s *SchemaService) GCPSchemas(platformRegion string) (create, update *map[string]interface{}, available bool) {
	return s.planSchemas(pkg.GCP, GCPPlanName, platformRegion)
}

func (s *SchemaService) BuildRuntimeGcpSchemas(platformRegion string) (create, update *map[string]interface{}, available bool) {
	return s.planSchemas(pkg.GCP, BuildRuntimeGCPPlanName, platformRegion)
}

func (s *SchemaService) PreviewSchemas(platformRegion string) (create, update *map[string]interface{}, available bool) {
	return s.planSchemas(pkg.AWS, PreviewPlanName, platformRegion)
}

func (s *SchemaService) SapConvergedCloudSchemas(platformRegion string) (create, update *map[string]interface{}, available bool) {
	return s.planSchemas(pkg.SapConvergedCloud, SapConvergedCloudPlanName, platformRegion)
}

func (s *SchemaService) AlicloudSchemas(platformRegion string) (create, update *map[string]interface{}, available bool) {
	return s.planSchemas(pkg.Alicloud, AlicloudPlanName, platformRegion)
}

func (s *SchemaService) AzureLiteSchema(platformRegion string, regions []string, update bool) *map[string]interface{} {
	flags := s.createFlags(AzureLitePlanName)
	machines := s.planSpec.RegularMachines(AzureLitePlanName)
	displayNames := s.providerSpec.MachineDisplayNames(pkg.Azure, machines)

	properties := NewProvisioningProperties(
		displayNames,
		displayNames,
		s.providerSpec.RegionDisplayNames(pkg.Azure, regions),
		machines,
		machines,
		regions,
		update,
		flags.rejectUnsupportedParameters,
		s.providerSpec,
		pkg.Azure,
		s.cfg.DualStackDocsURL,
		AzureLitePlanName,
		s.channelResolver,
	)
	properties.AutoScalerMax.Minimum = 2
	properties.AutoScalerMax.Maximum = 40
	properties.AutoScalerMin.Minimum = 2
	properties.AutoScalerMin.Maximum = 40

	properties.AdditionalWorkerNodePools.Items.Properties.HAZones = nil
	properties.AdditionalWorkerNodePools.Items.ControlsOrder = removeString(properties.AdditionalWorkerNodePools.Items.ControlsOrder, "haZones")
	properties.AdditionalWorkerNodePools.Items.Required = removeString(properties.AdditionalWorkerNodePools.Items.Required, "haZones")
	properties.AdditionalWorkerNodePools.Items.Properties.AutoScalerMin.Minimum = 0
	properties.AdditionalWorkerNodePools.Items.Properties.AutoScalerMin.Maximum = 40
	properties.AdditionalWorkerNodePools.Items.Properties.AutoScalerMin.Default = 2
	properties.AdditionalWorkerNodePools.Items.Properties.AutoScalerMax.Minimum = 1
	properties.AdditionalWorkerNodePools.Items.Properties.AutoScalerMax.Maximum = 40
	properties.AdditionalWorkerNodePools.Items.Properties.AutoScalerMax.Default = 10

	if !update {
		properties.AutoScalerMax.Default = 10
		properties.AutoScalerMin.Default = 2
	}

	return createSchemaWithProperties(properties, s.defaultOIDCConfig, update, requiredSchemaProperties(), flags)
}

func (s *SchemaService) AzureLiteSchemas(platformRegion string) (create, update *map[string]interface{}, available bool) {
	regions := s.planSpec.Regions(AzureLitePlanName, platformRegion)
	if len(regions) == 0 {
		return nil, nil, false
	}
	return s.AzureLiteSchema(platformRegion, regions, false),
		s.AzureLiteSchema(platformRegion, regions, true), true
}

func (s *SchemaService) FreeSchema(provider pkg.CloudProvider, platformRegion string, update bool) *map[string]interface{} {
	var regions []string
	var regionsDisplayNames map[string]string
	switch provider {
	case pkg.Azure:
		regions = s.planSpec.Regions(AzurePlanName, platformRegion)
		regionsDisplayNames = s.providerSpec.RegionDisplayNames(pkg.Azure, regions)
	default: // AWS and other BTP regions
		regions = s.planSpec.Regions(AWSPlanName, platformRegion)
		regionsDisplayNames = s.providerSpec.RegionDisplayNames(pkg.AWS, regions)
	}
	flags := s.createFlags(FreemiumPlanName)

	properties := ProvisioningProperties{
		UpdateProperties: UpdateProperties{
			Name: NameProperty(update),
		},
		Region: &Type{
			Type:            "string",
			Enum:            ToInterfaceSlice(regions),
			MinLength:       1,
			EnumDisplayName: regionsDisplayNames,
		},
	}
	if !update {
		defaultChannel := "regular"
		if s.channelResolver != nil {
			defaultChannel, _ = s.channelResolver.GetChannelForPlan(FreemiumPlanName)
		}
		properties.Networking = NewNetworkingSchema(flags.rejectUnsupportedParameters, s.providerSpec, provider, s.cfg.DualStackDocsURL)
		properties.Modules = NewModulesSchema(flags.rejectUnsupportedParameters, defaultChannel)
	}

	return createSchemaWithProperties(properties, s.defaultOIDCConfig, update, requiredSchemaProperties(), flags)
}

func (s *SchemaService) FreeSchemas(provider pkg.CloudProvider, platformRegion string) (create, update *map[string]interface{}, available bool) {
	create = s.FreeSchema(provider, platformRegion, false)
	update = s.FreeSchema(provider, platformRegion, true)
	return create, update, true
}

func (s *SchemaService) TrialSchema(update bool) *map[string]interface{} {
	flags := s.createFlags(TrialPlanName)

	properties := ProvisioningProperties{
		UpdateProperties: UpdateProperties{
			Name: NameProperty(update),
		},
	}

	if !update {
		defaultChannel := "regular"
		if s.channelResolver != nil {
			defaultChannel, _ = s.channelResolver.GetChannelForPlan(TrialPlanName)
		}
		properties.Modules = NewModulesSchema(flags.rejectUnsupportedParameters, defaultChannel)
	}

	return createSchemaWithProperties(properties, s.defaultOIDCConfig, update, requiredTrialSchemaProperties(), flags)
}

func (s *SchemaService) OwnClusterSchema(update bool) *map[string]interface{} {
	properties := ProvisioningProperties{
		ShootName:   ShootNameProperty(),
		ShootDomain: ShootDomainProperty(),
		UpdateProperties: UpdateProperties{
			Name:       NameProperty(update),
			Kubeconfig: KubeconfigProperty(),
		},
	}

	if update {
		return createSchemaWith(properties.UpdateProperties, []string{}, s.cfg.RejectUnsupportedParameters)
	} else {
		defaultChannel := "regular"
		if s.channelResolver != nil {
			defaultChannel, _ = s.channelResolver.GetChannelForPlan(OwnClusterPlanName)
		}
		properties.Modules = NewModulesSchema(s.cfg.RejectUnsupportedParameters, defaultChannel)
		return createSchemaWith(properties, requiredOwnClusterSchemaProperties(), s.cfg.RejectUnsupportedParameters)
	}
}

func (s *SchemaService) createFlags(planName string) ControlFlagsObject {
	return NewControlFlagsObject(
		s.ingressFilteringPlans.Contains(planName),
		s.cfg.RejectUnsupportedParameters,
	)
}

func (s *SchemaService) RandomZones(cp pkg.CloudProvider, region string, zonesCount int) []string {
	return s.providerSpec.RandomZones(cp, region, zonesCount)
}

func (s *SchemaService) PlanRegions(planName, platformRegion string) []string {
	return s.planSpec.Regions(planName, platformRegion)
}
