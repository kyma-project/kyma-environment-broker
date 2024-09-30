package broker

import (
	"strings"

	"github.com/kyma-incubator/compass/components/director/pkg/jsonschema"

	"github.com/pivotal-cf/brokerapi/v8/domain"

	"github.com/kyma-project/kyma-environment-broker/internal"
)

type PlanID string
type PlanName string

const (
	AllPlansSelector = "all_plans"

	GCPPlanID                 = "ca6e5357-707f-4565-bbbd-b3ab732597c6"
	GCPPlanName               = "gcp"
	AWSPlanID                 = "361c511f-f939-4621-b228-d0fb79a1fe15"
	AWSPlanName               = "aws"
	AzurePlanID               = "4deee563-e5ec-4731-b9b1-53b42d855f0c"
	AzurePlanName             = "azure"
	AzureLitePlanID           = "8cb22518-aa26-44c5-91a0-e669ec9bf443"
	AzureLitePlanName         = "azure_lite"
	TrialPlanID               = "7d55d31d-35ae-4438-bf13-6ffdfa107d9f"
	TrialPlanName             = "trial"
	SapConvergedCloudPlanID   = "03b812ac-c991-4528-b5bd-08b303523a63"
	SapConvergedCloudPlanName = "sap-converged-cloud"
	FreemiumPlanID            = "b1a5764e-2ea1-4f95-94c0-2b4538b37b55"
	FreemiumPlanName          = "free"
	OwnClusterPlanID          = "03e3cb66-a4c6-4c6a-b4b0-5d42224debea"
	OwnClusterPlanName        = "own_cluster"
	PreviewPlanID             = "5cb3d976-b85c-42ea-a636-79cadda109a9"
	PreviewPlanName           = "preview"
)

var PlanNamesMapping = map[string]string{
	GCPPlanID:               GCPPlanName,
	AWSPlanID:               AWSPlanName,
	AzurePlanID:             AzurePlanName,
	AzureLitePlanID:         AzureLitePlanName,
	TrialPlanID:             TrialPlanName,
	SapConvergedCloudPlanID: SapConvergedCloudPlanName,
	FreemiumPlanID:          FreemiumPlanName,
	OwnClusterPlanID:        OwnClusterPlanName,
	PreviewPlanID:           PreviewPlanName,
}

var PlanIDsMapping = map[string]string{
	AzurePlanName:             AzurePlanID,
	AWSPlanName:               AWSPlanID,
	AzureLitePlanName:         AzureLitePlanID,
	GCPPlanName:               GCPPlanID,
	TrialPlanName:             TrialPlanID,
	SapConvergedCloudPlanName: SapConvergedCloudPlanID,
	FreemiumPlanName:          FreemiumPlanID,
	OwnClusterPlanName:        OwnClusterPlanID,
	PreviewPlanName:           PreviewPlanID,
}

type TrialCloudRegion string

const (
	Europe TrialCloudRegion = "europe"
	Us     TrialCloudRegion = "us"
	Asia   TrialCloudRegion = "asia"
)

var validRegionsForTrial = map[TrialCloudRegion]struct{}{
	Europe: {},
	Us:     {},
	Asia:   {},
}

type JSONSchemaValidator interface {
	ValidateString(json string) (jsonschema.ValidationResult, error)
}

func AzureRegions(euRestrictedAccess bool) []string {
	if euRestrictedAccess {
		return []string{
			"switzerlandnorth",
		}
	}
	return []string{
		"eastus",
		"centralus",
		"westus2",
		"uksouth",
		"northeurope",
		"westeurope",
		"japaneast",
		"southeastasia",
		"australiaeast",
		"brazilsouth",
	}
}

func AzureRegionsDisplay(euRestrictedAccess bool) map[string]string {
	if euRestrictedAccess {
		return map[string]string{
			"switzerlandnorth": "switzerlandnorth (Switzerland, Zurich)",
		}
	}
	return map[string]string{
		"eastus":        "eastus (US East, VA)",
		"centralus":     "centralus (US Central, IA)",
		"westus2":       "westus2 (US West, WA)",
		"uksouth":       "uksouth (UK South, London)",
		"northeurope":   "northeurope (Europe, Ireland)",
		"westeurope":    "westeurope (Europe, Netherlands)",
		"japaneast":     "japaneast (Japan, Tokyo)",
		"southeastasia": "southeastasia (Asia Pacific, Singapore)",
		"australiaeast": "australiaeast (Australia, Sydney)",
		"brazilsouth":   "brazilsouth (Brazil, São Paulo)",
	}
}

func GcpRegions(assuredWorkloads bool) []string {
	if assuredWorkloads {
		return []string{
			"me-central2",
		}
	}
	return []string{
		"europe-west3",
		"asia-south1",
		"us-central1",
		"me-central2",
		"asia-northeast2",
		"me-west1",
		"southamerica-east1",
		"australia-southeast1",
	}
}

func GcpRegionsDisplay(assuredWorkloads bool) map[string]string {
	if assuredWorkloads {
		return map[string]string{
			"me-central2": "me-central2 (KSA, Dammam)",
		}
	}
	return map[string]string{
		"europe-west3":         "europe-west3 (Europe, Frankfurt)",
		"asia-south1":          "asia-south1 (India, Mumbai)",
		"us-central1":          "us-central1 (US Central, IA)",
		"me-central2":          "me-central2 (KSA, Dammam)",
		"asia-northeast2":      "asia-northeast2 (Japan, Osaka)",
		"me-west1":             "me-west1 (Israel, Tel Aviv)",
		"southamerica-east1":   "southamerica-east1 (Brazil, São Paulo)",
		"australia-southeast1": "australia-southeast1 (Australia, Sydney)",
	}
}

func AWSRegions(euRestrictedAccess bool) []string {
	// be aware of zones defined in internal/provider/aws_provider.go
	if euRestrictedAccess {
		return []string{"eu-central-1"}
	}
	return []string{"eu-central-1", "eu-west-2", "ca-central-1", "sa-east-1", "us-east-1", "us-west-1",
		"ap-northeast-1", "ap-northeast-2", "ap-south-1", "ap-southeast-1", "ap-southeast-2"}
}

func AWSRegionsDisplay() map[string]string {
	return nil
}

func SapConvergedCloudRegionsDisplay() map[string]string {
	return nil
}

func AwsMachinesNames() []string {
	return []string{
		"m6i.large",
		"m6i.xlarge",
		"m6i.2xlarge",
		"m6i.4xlarge",
		"m6i.8xlarge",
		"m6i.12xlarge",
		"m5.large",
		"m5.xlarge",
		"m5.2xlarge",
		"m5.4xlarge",
		"m5.8xlarge",
		"m5.12xlarge",
	}
}

func AwsMachinesDisplay() map[string]string {
	return map[string]string{
		"m6i.large":    "m6i.large (2vCPU, 8GB RAM)",
		"m6i.xlarge":   "m6i.xlarge (4vCPU, 16GB RAM)",
		"m6i.2xlarge":  "m6i.2xlarge (8vCPU, 32GB RAM)",
		"m6i.4xlarge":  "m6i.4xlarge (16vCPU, 64GB RAM)",
		"m6i.8xlarge":  "m6i.8xlarge (32vCPU, 128GB RAM)",
		"m6i.12xlarge": "m6i.12xlarge (48vCPU, 192GB RAM)",
		"m5.large":     "m5.large (2vCPU, 8GB RAM)",
		"m5.xlarge":    "m5.xlarge (4vCPU, 16GB RAM)",
		"m5.2xlarge":   "m5.2xlarge (8vCPU, 32GB RAM)",
		"m5.4xlarge":   "m5.4xlarge (16vCPU, 64GB RAM)",
		"m5.8xlarge":   "m5.8xlarge (32vCPU, 128GB RAM)",
		"m5.12xlarge":  "m5.12xlarge (48vCPU, 192GB RAM)",
	}
}

func AzureMachinesNames() []string {
	return []string{
		"Standard_D2s_v5",
		"Standard_D4s_v5",
		"Standard_D8s_v5",
		"Standard_D16s_v5",
		"Standard_D32s_v5",
		"Standard_D48s_v5",
		"Standard_D64s_v5",
		"Standard_D4_v3",
		"Standard_D8_v3",
		"Standard_D16_v3",
		"Standard_D32_v3",
		"Standard_D48_v3",
		"Standard_D64_v3",
	}
}

func AzureMachinesDisplay() map[string]string {
	return map[string]string{
		"Standard_D2s_v5":  "Standard_D2s_v5 (2vCPU, 8GB RAM)",
		"Standard_D4s_v5":  "Standard_D4s_v5 (4vCPU, 16GB RAM)",
		"Standard_D8s_v5":  "Standard_D8s_v5 (8vCPU, 32GB RAM)",
		"Standard_D16s_v5": "Standard_D16s_v5 (16vCPU, 64GB RAM)",
		"Standard_D32s_v5": "Standard_D32s_v5 (32vCPU, 128GB RAM)",
		"Standard_D48s_v5": "Standard_D48s_v5 (48vCPU, 192GB RAM)",
		"Standard_D64s_v5": "Standard_D64s_v5 (64vCPU, 256GB RAM)",
		"Standard_D4_v3":   "Standard_D4_v3 (4vCPU, 16GB RAM)",
		"Standard_D8_v3":   "Standard_D8_v3 (8vCPU, 32GB RAM)",
		"Standard_D16_v3":  "Standard_D16_v3 (16vCPU, 64GB RAM)",
		"Standard_D32_v3":  "Standard_D32_v3 (32vCPU, 128GB RAM)",
		"Standard_D48_v3":  "Standard_D48_v3 (48vCPU, 192GB RAM)",
		"Standard_D64_v3":  "Standard_D64_v3 (64vCPU, 256GB RAM)",
	}
}

func AzureLiteMachinesNames() []string {
	return []string{
		"Standard_D2s_v5",
		"Standard_D4s_v5",
		"Standard_D4_v3",
	}
}

func AzureLiteMachinesDisplay() map[string]string {
	return map[string]string{
		"Standard_D2s_v5": "Standard_D2s_v5 (2vCPU, 8GB RAM)",
		"Standard_D4s_v5": "Standard_D4s_v5 (4vCPU, 16GB RAM)",
		"Standard_D4_v3":  "Standard_D4_v3 (4vCPU, 16GB RAM)",
	}
}

func GcpMachinesNames() []string {
	return []string{
		"n2-standard-2",
		"n2-standard-4",
		"n2-standard-8",
		"n2-standard-16",
		"n2-standard-32",
		"n2-standard-48",
	}
}

func GcpMachinesDisplay() map[string]string {
	return map[string]string{
		"n2-standard-2":  "n2-standard-2 (2vCPU, 8GB RAM)",
		"n2-standard-4":  "n2-standard-4 (4vCPU, 16GB RAM)",
		"n2-standard-8":  "n2-standard-8 (8vCPU, 32GB RAM)",
		"n2-standard-16": "n2-standard-16 (16vCPU, 64GB RAM)",
		"n2-standard-32": "n2-standard-32 (32vCPU, 128GB RAM)",
		"n2-standard-48": "n2-standard-48 (48vCPU, 192B RAM)",
	}
}

func SapConvergedCloudMachinesNames() []string {
	return []string{
		"g_c2_m8",
		"g_c4_m16",
		"g_c6_m24",
		"g_c8_m32",
		"g_c12_m48",
		"g_c16_m64",
		"g_c32_m128",
		"g_c64_m256",
	}
}

func SapConvergedCloudMachinesDisplay() map[string]string {
	return map[string]string{
		"g_c2_m8":    "g_c2_m8 (2vCPU, 8GB RAM)",
		"g_c4_m16":   "g_c4_m16 (4vCPU, 16GB RAM)",
		"g_c6_m24":   "g_c6_m24 (6vCPU, 24GB RAM)",
		"g_c8_m32":   "g_c8_m32 (8vCPU, 32GB RAM)",
		"g_c12_m48":  "g_c12_m48 (12vCPU, 48GB RAM)",
		"g_c16_m64":  "g_c16_m64 (16vCPU, 64GB RAM)",
		"g_c32_m128": "g_c32_m128 (32vCPU, 128GB RAM)",
		"g_c64_m256": "g_c64_m256 (64vCPU, 256GB RAM)",
	}
}

func removeMachinesNamesFromList(machinesNames []string, machinesNamesToRemove ...string) []string {
	for i, machineName := range machinesNames {
		for _, machineNameToRemove := range machinesNamesToRemove {
			if machineName == machineNameToRemove {
				copy(machinesNames[i:], machinesNames[i+1:])
				machinesNames[len(machinesNames)-1] = ""
				machinesNames = machinesNames[:len(machinesNames)-1]
			}
		}
	}

	return machinesNames
}

func requiredSchemaProperties() []string {
	return []string{"name", "region"}
}

func requiredTrialSchemaProperties() []string {
	return []string{"name"}
}

func requiredOwnClusterSchemaProperties() []string {
	return []string{"name", "kubeconfig", "shootName", "shootDomain"}
}

func SapConvergedCloudSchema(machineTypesDisplay, regionsDisplay map[string]string, machineTypes []string, additionalParams, update bool, shootAndSeedFeatureFlag bool, sapConvergedCloudRegions []string) *map[string]interface{} {
	properties := NewProvisioningProperties(machineTypesDisplay, regionsDisplay, machineTypes, sapConvergedCloudRegions, update)
	return createSchemaWithProperties(properties, additionalParams, update, requiredSchemaProperties(), true, shootAndSeedFeatureFlag)
}

func PreviewSchema(machineTypesDisplay, regionsDisplay map[string]string, machineTypes []string, additionalParams, update bool, euAccessRestricted bool) *map[string]interface{} {
	properties := NewProvisioningProperties(machineTypesDisplay, regionsDisplay, machineTypes, AWSRegions(euAccessRestricted), update)
	properties.Networking = NewNetworkingSchema()
	return createSchemaWithProperties(properties, additionalParams, update, requiredSchemaProperties(), false, false)
}

func GCPSchema(machineTypesDisplay, regionsDisplay map[string]string, machineTypes []string, additionalParams, update bool, shootAndSeedFeatureFlag bool, assuredWorkloads bool) *map[string]interface{} {
	properties := NewProvisioningProperties(machineTypesDisplay, regionsDisplay, machineTypes, GcpRegions(assuredWorkloads), update)
	return createSchemaWithProperties(properties, additionalParams, update, requiredSchemaProperties(), true, shootAndSeedFeatureFlag)
}

func AWSSchema(machineTypesDisplay, regionsDisplay map[string]string, machineTypes []string, additionalParams, update bool, euAccessRestricted bool, shootAndSeedSameRegion bool) *map[string]interface{} {
	properties := NewProvisioningProperties(machineTypesDisplay, regionsDisplay, machineTypes, AWSRegions(euAccessRestricted), update)
	return createSchemaWithProperties(properties, additionalParams, update, requiredSchemaProperties(), true, shootAndSeedSameRegion)
}

func AzureSchema(machineTypesDisplay, regionsDisplay map[string]string, machineTypes []string, additionalParams, update bool, euAccessRestricted bool, shootAndSeedFeatureFlag bool) *map[string]interface{} {
	properties := NewProvisioningProperties(machineTypesDisplay, regionsDisplay, machineTypes, AzureRegions(euAccessRestricted), update)
	return createSchemaWithProperties(properties, additionalParams, update, requiredSchemaProperties(), true, shootAndSeedFeatureFlag)
}

func AzureLiteSchema(machineTypesDisplay, regionsDisplay map[string]string, machineTypes []string, additionalParams, update bool, euAccessRestricted bool, shootAndSeedFeatureFlag bool) *map[string]interface{} {
	properties := NewProvisioningProperties(machineTypesDisplay, regionsDisplay, machineTypes, AzureRegions(euAccessRestricted), update)

	properties.AutoScalerMax.Minimum = 2
	properties.AutoScalerMin.Minimum = 2
	properties.AutoScalerMax.Maximum = 40

	if !update {
		properties.AutoScalerMax.Default = 10
		properties.AutoScalerMin.Default = 2
	}

	return createSchemaWithProperties(properties, additionalParams, update, requiredSchemaProperties(), true, shootAndSeedFeatureFlag)
}

func FreemiumSchema(provider internal.CloudProvider, regionsDisplay map[string]string, additionalParams, update bool, euAccessRestricted bool) *map[string]interface{} {
	if update && !additionalParams {
		return empty()
	}

	var regions []string
	switch provider {
	case internal.AWS:
		regions = AWSRegions(euAccessRestricted)
	case internal.Azure:
		regions = AzureRegions(euAccessRestricted)
	default:
		regions = AWSRegions(euAccessRestricted)
	}
	properties := ProvisioningProperties{
		Name: NameProperty(),
		Region: &Type{
			Type:            "string",
			Enum:            ToInterfaceSlice(regions),
			MinLength:       1,
			EnumDisplayName: regionsDisplay,
		},
	}
	if !update {
		properties.Networking = NewNetworkingSchema()
		properties.Modules = NewModulesSchema()
	}

	return createSchemaWithProperties(properties, additionalParams, update, requiredSchemaProperties(), false, false)
}

func TrialSchema(additionalParams, update bool) *map[string]interface{} {
	properties := ProvisioningProperties{
		Name: NameProperty(),
	}

	if !update {
		properties.Modules = NewModulesSchema()
	}

	if update && !additionalParams {
		return empty()
	}

	return createSchemaWithProperties(properties, additionalParams, update, requiredTrialSchemaProperties(), false, false)
}

func OwnClusterSchema(update bool) *map[string]interface{} {
	properties := ProvisioningProperties{
		Name:        NameProperty(),
		ShootName:   ShootNameProperty(),
		ShootDomain: ShootDomainProperty(),
		UpdateProperties: UpdateProperties{
			Kubeconfig: KubeconfigProperty(),
		},
	}

	if update {
		return createSchemaWith(properties.UpdateProperties, update, requiredOwnClusterSchemaProperties())
	} else {
		properties.Modules = NewModulesSchema()
		return createSchemaWith(properties, update, requiredOwnClusterSchemaProperties())
	}
}

func empty() *map[string]interface{} {
	empty := make(map[string]interface{}, 0)
	return &empty
}

func createSchemaWithProperties(properties ProvisioningProperties, additionalParams, update bool, required []string, shootAndSeedSameRegion bool, shootAndSeedFeatureFlag bool) *map[string]interface{} {
	if additionalParams {
		properties.IncludeAdditional()
	}

	if shootAndSeedFeatureFlag && additionalParams && shootAndSeedSameRegion {
		properties.ShootAndSeedSameRegion = ShootAndSeedSameRegionProperty()
	}

	if update {
		return createSchemaWith(properties.UpdateProperties, update, required)
	} else {
		return createSchemaWith(properties, update, required)
	}
}

func createSchemaWith(properties interface{}, update bool, requiered []string) *map[string]interface{} {
	schema := NewSchema(properties, update, requiered)

	return unmarshalSchema(schema)
}

func unmarshalSchema(schema *RootSchema) *map[string]interface{} {
	target := make(map[string]interface{})
	schema.ControlsOrder = DefaultControlsOrder()

	unmarshaled := unmarshalOrPanic(schema, &target).(*map[string]interface{})

	// update controls order
	props := (*unmarshaled)[PropertiesKey].(map[string]interface{})
	controlsOrder := (*unmarshaled)[ControlsOrderKey].([]interface{})
	(*unmarshaled)[ControlsOrderKey] = filter(&controlsOrder, props)

	return unmarshaled
}

// Plans is designed to hold plan defaulting logic
// keep internal/hyperscaler/azure/config.go in sync with any changes to available zones
func Plans(plans PlansConfig, provider internal.CloudProvider, includeAdditionalParamsInSchema bool, euAccessRestricted bool, useSmallerMachineTypes bool, shootAndSeedFeatureFlag bool, sapConvergedCloudRegions []string, assuredWorkloads bool) map[string]domain.ServicePlan {
	awsMachineNames := AwsMachinesNames()
	awsMachinesDisplay := AwsMachinesDisplay()
	awsRegionsDisplay := AWSRegionsDisplay()
	azureMachinesNames := AzureMachinesNames()
	azureMachinesDisplay := AzureMachinesDisplay()
	azureRegionsDisplay := AzureRegionsDisplay(euAccessRestricted)
	azureLiteMachinesNames := AzureLiteMachinesNames()
	azureLiteMachinesDisplay := AzureLiteMachinesDisplay()
	gcpMachinesNames := GcpMachinesNames()
	gcpMachinesDisplay := GcpMachinesDisplay()
	gcpRegionsDisplay := GcpRegionsDisplay(assuredWorkloads)

	if !useSmallerMachineTypes {
		azureLiteMachinesNames = removeMachinesNamesFromList(azureLiteMachinesNames, "Standard_D2s_v5")
		delete(azureLiteMachinesDisplay, "Standard_D2s_v5")
	}

	awsSchema := AWSSchema(awsMachinesDisplay, awsRegionsDisplay, awsMachineNames, includeAdditionalParamsInSchema, false, euAccessRestricted, shootAndSeedFeatureFlag)
	// awsHASchema := AWSHASchema(awsMachinesDisplay, awsMachines, includeAdditionalParamsInSchema, false)
	azureSchema := AzureSchema(azureMachinesDisplay, azureRegionsDisplay, azureMachinesNames, includeAdditionalParamsInSchema, false, euAccessRestricted, shootAndSeedFeatureFlag)
	azureLiteSchema := AzureLiteSchema(azureLiteMachinesDisplay, azureRegionsDisplay, azureLiteMachinesNames, includeAdditionalParamsInSchema, false, euAccessRestricted, shootAndSeedFeatureFlag)
	freemiumSchema := FreemiumSchema(provider, azureRegionsDisplay, includeAdditionalParamsInSchema, false, euAccessRestricted)
	gcpSchema := GCPSchema(gcpMachinesDisplay, gcpRegionsDisplay, gcpMachinesNames, includeAdditionalParamsInSchema, false, shootAndSeedFeatureFlag, assuredWorkloads)
	ownClusterSchema := OwnClusterSchema(false)
	previewCatalogSchema := PreviewSchema(awsMachinesDisplay, awsRegionsDisplay, awsMachineNames, includeAdditionalParamsInSchema, false, euAccessRestricted)

	trialSchema := TrialSchema(includeAdditionalParamsInSchema, false)

	outputPlans := map[string]domain.ServicePlan{
		AWSPlanID:        defaultServicePlan(AWSPlanID, AWSPlanName, plans, awsSchema, AWSSchema(awsMachinesDisplay, awsRegionsDisplay, awsMachineNames, includeAdditionalParamsInSchema, true, euAccessRestricted, shootAndSeedFeatureFlag)),
		GCPPlanID:        defaultServicePlan(GCPPlanID, GCPPlanName, plans, gcpSchema, GCPSchema(gcpMachinesDisplay, gcpRegionsDisplay, gcpMachinesNames, includeAdditionalParamsInSchema, true, shootAndSeedFeatureFlag, assuredWorkloads)),
		AzurePlanID:      defaultServicePlan(AzurePlanID, AzurePlanName, plans, azureSchema, AzureSchema(azureMachinesDisplay, azureRegionsDisplay, azureMachinesNames, includeAdditionalParamsInSchema, true, euAccessRestricted, shootAndSeedFeatureFlag)),
		AzureLitePlanID:  defaultServicePlan(AzureLitePlanID, AzureLitePlanName, plans, azureLiteSchema, AzureLiteSchema(azureLiteMachinesDisplay, azureRegionsDisplay, azureLiteMachinesNames, includeAdditionalParamsInSchema, true, euAccessRestricted, shootAndSeedFeatureFlag)),
		FreemiumPlanID:   defaultServicePlan(FreemiumPlanID, FreemiumPlanName, plans, freemiumSchema, FreemiumSchema(provider, azureRegionsDisplay, includeAdditionalParamsInSchema, true, euAccessRestricted)),
		TrialPlanID:      defaultServicePlan(TrialPlanID, TrialPlanName, plans, trialSchema, TrialSchema(includeAdditionalParamsInSchema, true)),
		OwnClusterPlanID: defaultServicePlan(OwnClusterPlanID, OwnClusterPlanName, plans, ownClusterSchema, OwnClusterSchema(true)),
		PreviewPlanID:    defaultServicePlan(PreviewPlanID, PreviewPlanName, plans, previewCatalogSchema, AWSSchema(awsMachinesDisplay, awsRegionsDisplay, awsMachineNames, includeAdditionalParamsInSchema, true, euAccessRestricted, false)),
	}

	if len(sapConvergedCloudRegions) != 0 {
		sapConvergedCloudMachinesNames := SapConvergedCloudMachinesNames()
		sapConvergedCloudMachinesDisplay := SapConvergedCloudMachinesDisplay()
		sapConvergedCloudRegionsDisplay := SapConvergedCloudRegionsDisplay()
		sapConvergedCloudSchema := SapConvergedCloudSchema(sapConvergedCloudMachinesDisplay, sapConvergedCloudRegionsDisplay, sapConvergedCloudMachinesNames, includeAdditionalParamsInSchema, false, shootAndSeedFeatureFlag, sapConvergedCloudRegions)
		outputPlans[SapConvergedCloudPlanID] = defaultServicePlan(SapConvergedCloudPlanID, SapConvergedCloudPlanName, plans, sapConvergedCloudSchema, SapConvergedCloudSchema(sapConvergedCloudMachinesDisplay, sapConvergedCloudRegionsDisplay, sapConvergedCloudMachinesNames, includeAdditionalParamsInSchema, true, shootAndSeedFeatureFlag, sapConvergedCloudRegions))
	}

	return outputPlans
}

func defaultServicePlan(id, name string, plans PlansConfig, createParams, updateParams *map[string]interface{}) domain.ServicePlan {
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
	}

	return servicePlan
}

func defaultDescription(planName string, plans PlansConfig) string {
	plan, ok := plans[planName]
	if !ok || len(plan.Description) == 0 {
		return strings.ToTitle(planName)
	}

	return plan.Description
}

func defaultMetadata(planName string, plans PlansConfig) *domain.ServicePlanMetadata {
	plan, ok := plans[planName]
	if !ok || len(plan.Metadata.DisplayName) == 0 {
		return &domain.ServicePlanMetadata{
			DisplayName: strings.ToTitle(planName),
		}
	}
	return &domain.ServicePlanMetadata{
		DisplayName: plan.Metadata.DisplayName,
	}
}

func IsTrialPlan(planID string) bool {
	switch planID {
	case TrialPlanID:
		return true
	default:
		return false
	}
}

func IsSapConvergedCloudPlan(planID string) bool {
	switch planID {
	case SapConvergedCloudPlanID:
		return true
	default:
		return false
	}
}

func IsPreviewPlan(planID string) bool {
	switch planID {
	case PreviewPlanID:
		return true
	default:
		return false
	}
}

func IsAzurePlan(planID string) bool {
	switch planID {
	case AzurePlanID, AzureLitePlanID:
		return true
	default:
		return false
	}
}

func IsFreemiumPlan(planID string) bool {
	switch planID {
	case FreemiumPlanID:
		return true
	default:
		return false
	}
}

func IsOwnClusterPlan(planID string) bool {
	return planID == OwnClusterPlanID
}

func filter(items *[]interface{}, included map[string]interface{}) interface{} {
	output := make([]interface{}, 0)
	for i := 0; i < len(*items); i++ {
		value := (*items)[i]

		if _, ok := included[value.(string)]; ok {
			output = append(output, value)
		}
	}

	return output
}
