package broker

import (
	"strings"

	"github.com/pivotal-cf/brokerapi/v12/domain"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
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
	BuildRuntimeAWSPlanID     = "6aae0ff3-89f7-4f12-86de-51466145422e"
	BuildRuntimeAWSPlanName   = "build-runtime-aws"
	BuildRuntimeGCPPlanID     = "a310cd6b-6452-45a0-935d-d24ab53f9eba"
	BuildRuntimeGCPPlanName   = "build-runtime-gcp"
	BuildRuntimeAzurePlanID   = "499244b4-1bef-48c9-be68-495269899f8e"
	BuildRuntimeAzurePlanName = "build-runtime-azure"
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
	BuildRuntimeAWSPlanID:   BuildRuntimeAWSPlanName,
	BuildRuntimeGCPPlanID:   BuildRuntimeGCPPlanName,
	BuildRuntimeAzurePlanID: BuildRuntimeAzurePlanName,
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
	BuildRuntimeAWSPlanName:   BuildRuntimeAWSPlanID,
	BuildRuntimeGCPPlanName:   BuildRuntimeGCPPlanID,
	BuildRuntimeAzurePlanName: BuildRuntimeAzurePlanID,
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
		"canadacentral",
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
		"canadacentral": "canadacentral (Canada, Toronto)",
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
		"asia-northeast1",
		"asia-southeast1",
		"us-west1",
		"us-east4",
		"europe-west4",
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
		"asia-northeast1":      "asia-northeast1 (Japan, Tokyo)",
		"asia-southeast1":      "asia-southeast1 (Singapore, Jurong West)",
		"us-west1":             "us-west1 (North America, Oregon)",
		"us-east4":             "us-east4 (North America, Virginia)",
		"europe-west4":         "europe-west4 (Europe, Netherlands)",
	}
}

func AWSRegions(euRestrictedAccess bool) []string {
	// be aware of zones defined in internal/provider/aws_provider.go
	if euRestrictedAccess {
		return []string{"eu-central-1"}
	}
	return []string{
		"eu-central-1",
		"eu-west-2",
		"ca-central-1",
		"sa-east-1",
		"us-east-1",
		"us-west-2",
		"ap-northeast-1",
		"ap-northeast-2",
		"ap-south-1",
		"ap-southeast-1",
		"ap-southeast-2"}
}

func AWSRegionsDisplay(euRestrictedAccess bool) map[string]string {
	if euRestrictedAccess {
		return map[string]string{
			"eu-central-1": "eu-central-1 (Europe, Frankfurt)",
		}
	}
	return map[string]string{
		"eu-central-1":   "eu-central-1 (Europe, Frankfurt)",
		"eu-west-2":      "eu-west-2 (Europe, London)",
		"ca-central-1":   "ca-central-1 (Canada, Montreal)",
		"sa-east-1":      "sa-east-1 (Brazil, São Paulo)",
		"us-east-1":      "us-east-1 (US East, N. Virginia)",
		"us-west-2":      "us-west-2 (US West, Oregon)",
		"ap-northeast-1": "ap-northeast-1 (Asia Pacific, Tokyo)",
		"ap-northeast-2": "ap-northeast-2 (Asia Pacific, Seoul)",
		"ap-south-1":     "ap-south-1 (Asia Pacific, Mumbai)",
		"ap-southeast-1": "ap-southeast-1 (Asia Pacific, Singapore)",
		"ap-southeast-2": "ap-southeast-2 (Asia Pacific, Sydney)",
	}
}

func SapConvergedCloudRegionsDisplay() map[string]string {
	return nil
}

func AwsMachinesNames(additionalMachines bool) []string {
	machines := []string{
		"m6i.large",
		"m6i.xlarge",
		"m6i.2xlarge",
		"m6i.4xlarge",
		"m6i.8xlarge",
		"m6i.12xlarge",
		"m6i.16xlarge",
		"m5.large",
		"m5.xlarge",
		"m5.2xlarge",
		"m5.4xlarge",
		"m5.8xlarge",
		"m5.12xlarge",
		"m5.16xlarge",
	}

	if additionalMachines {
		machines = append(machines,
			"c7i.large",
			"c7i.xlarge",
			"c7i.2xlarge",
			"c7i.4xlarge",
			"c7i.8xlarge",
			"c7i.12xlarge",
			"c7i.16xlarge",
			"g6.xlarge",
			"g6.2xlarge",
			"g6.4xlarge",
			"g6.8xlarge",
			"g6.12xlarge",
			"g6.16xlarge",
			"g4dn.xlarge",
			"g4dn.2xlarge",
			"g4dn.4xlarge",
			"g4dn.8xlarge",
			"g4dn.12xlarge",
			"g4dn.16xlarge",
		)
	}

	return machines
}

func AwsMachinesDisplay(additionalMachines bool) map[string]string {
	machines := map[string]string{
		"m6i.large":    "m6i.large (2vCPU, 8GB RAM)",
		"m6i.xlarge":   "m6i.xlarge (4vCPU, 16GB RAM)",
		"m6i.2xlarge":  "m6i.2xlarge (8vCPU, 32GB RAM)",
		"m6i.4xlarge":  "m6i.4xlarge (16vCPU, 64GB RAM)",
		"m6i.8xlarge":  "m6i.8xlarge (32vCPU, 128GB RAM)",
		"m6i.12xlarge": "m6i.12xlarge (48vCPU, 192GB RAM)",
		"m6i.16xlarge": "m6i.16xlarge (64vCPU, 256GB RAM)",
		"m5.large":     "m5.large (2vCPU, 8GB RAM)",
		"m5.xlarge":    "m5.xlarge (4vCPU, 16GB RAM)",
		"m5.2xlarge":   "m5.2xlarge (8vCPU, 32GB RAM)",
		"m5.4xlarge":   "m5.4xlarge (16vCPU, 64GB RAM)",
		"m5.8xlarge":   "m5.8xlarge (32vCPU, 128GB RAM)",
		"m5.12xlarge":  "m5.12xlarge (48vCPU, 192GB RAM)",
		"m5.16xlarge":  "m5.16xlarge (64vCPU, 256GB RAM)",
	}

	if additionalMachines {
		machines["c7i.large"] = "c7i.large (2vCPU, 4GB RAM)"
		machines["c7i.xlarge"] = "c7i.xlarge (4vCPU, 8GB RAM)"
		machines["c7i.2xlarge"] = "c7i.2xlarge (8vCPU, 16GB RAM)"
		machines["c7i.4xlarge"] = "c7i.4xlarge (16vCPU, 32GB RAM)"
		machines["c7i.8xlarge"] = "c7i.8xlarge (32vCPU, 64GB RAM)"
		machines["c7i.12xlarge"] = "c7i.12xlarge (48vCPU, 96GB RAM)"
		machines["c7i.16xlarge"] = "c7i.16xlarge (64vCPU, 128GB RAM)"
		machines["g6.xlarge"] = "g6.xlarge (1GPU, 4vCPU, 16GB RAM)*"
		machines["g6.2xlarge"] = "g6.2xlarge (1GPU, 8vCPU, 32GB RAM)*"
		machines["g6.4xlarge"] = "g6.4xlarge (1GPU, 16vCPU, 64GB RAM)*"
		machines["g6.8xlarge"] = "g6.8xlarge (1GPU, 32vCPU, 128GB RAM)*"
		machines["g6.12xlarge"] = "g6.12xlarge (4GPU, 48vCPU, 192GB RAM)*"
		machines["g6.16xlarge"] = "g6.16xlarge (1GPU, 64vCPU, 256GB RAM)*"
		machines["g4dn.xlarge"] = "g4dn.xlarge (1GPU, 4vCPU, 16GB RAM)*"
		machines["g4dn.2xlarge"] = "g4dn.2xlarge (1GPU, 8vCPU, 32GB RAM)*"
		machines["g4dn.4xlarge"] = "g4dn.4xlarge (1GPU, 16vCPU, 64GB RAM)*"
		machines["g4dn.8xlarge"] = "g4dn.8xlarge (1GPU, 32vCPU, 128GB RAM)*"
		machines["g4dn.12xlarge"] = "g4dn.12xlarge (4GPU, 48vCPU, 192GB RAM)*"
		machines["g4dn.16xlarge"] = "g4dn.16xlarge (1GPU, 64vCPU, 256GB RAM)*"
	}

	return machines
}

func AzureMachinesNames(additionalMachines bool) []string {
	machines := []string{
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

	if additionalMachines {
		machines = append(machines,
			"Standard_F2s_v2",
			"Standard_F4s_v2",
			"Standard_F8s_v2",
			"Standard_F16s_v2",
			"Standard_F32s_v2",
			"Standard_F48s_v2",
			"Standard_F64s_v2",
			"Standard_NC4as_T4_v3",
			"Standard_NC8as_T4_v3",
			"Standard_NC16as_T4_v3",
			"Standard_NC64as_T4_v3",
		)
	}

	return machines
}

func AzureMachinesDisplay(additionalMachines bool) map[string]string {
	machines := map[string]string{
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

	if additionalMachines {
		machines["Standard_F2s_v2"] = "Standard_F2s_v2 (2vCPU, 4GB RAM)"
		machines["Standard_F4s_v2"] = "Standard_F4s_v2 (4vCPU, 8GB RAM)"
		machines["Standard_F8s_v2"] = "Standard_F8s_v2 (8vCPU, 16GB RAM)"
		machines["Standard_F16s_v2"] = "Standard_F16s_v2 (16vCPU, 32GB RAM)"
		machines["Standard_F32s_v2"] = "Standard_F32s_v2 (32vCPU, 64GB RAM)"
		machines["Standard_F48s_v2"] = "Standard_F48s_v2 (48vCPU, 96GB RAM)"
		machines["Standard_F64s_v2"] = "Standard_F64s_v2 (64vCPU, 128GB RAM)"
		machines["Standard_NC4as_T4_v3"] = "Standard_NC4as_T4_v3 (1GPU, 4vCPU, 28GB RAM)*"
		machines["Standard_NC8as_T4_v3"] = "Standard_NC8as_T4_v3 (1GPU, 8vCPU, 56GB RAM)*"
		machines["Standard_NC16as_T4_v3"] = "Standard_NC16as_T4_v3 (1GPU, 16vCPU, 110GB RAM)*"
		machines["Standard_NC64as_T4_v3"] = "Standard_NC64as_T4_v3 (4GPU, 64vCPU, 440GB RAM)*"
	}

	return machines
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

func GcpMachinesNames(additionalMachines bool) []string {
	machines := []string{
		"n2-standard-2",
		"n2-standard-4",
		"n2-standard-8",
		"n2-standard-16",
		"n2-standard-32",
		"n2-standard-48",
		"n2-standard-64",
	}

	if additionalMachines {
		machines = append(machines,
			"c2d-highcpu-2",
			"c2d-highcpu-4",
			"c2d-highcpu-8",
			"c2d-highcpu-16",
			"c2d-highcpu-32",
			"c2d-highcpu-56",
			"g2-standard-4",
			"g2-standard-8",
			"g2-standard-12",
			"g2-standard-16",
			"g2-standard-24",
			"g2-standard-32",
			"g2-standard-48",
		)
	}

	return machines
}

func GcpMachinesDisplay(additionalMachines bool) map[string]string {
	machines := map[string]string{
		"n2-standard-2":  "n2-standard-2 (2vCPU, 8GB RAM)",
		"n2-standard-4":  "n2-standard-4 (4vCPU, 16GB RAM)",
		"n2-standard-8":  "n2-standard-8 (8vCPU, 32GB RAM)",
		"n2-standard-16": "n2-standard-16 (16vCPU, 64GB RAM)",
		"n2-standard-32": "n2-standard-32 (32vCPU, 128GB RAM)",
		"n2-standard-48": "n2-standard-48 (48vCPU, 192GB RAM)",
		"n2-standard-64": "n2-standard-64 (64vCPU, 256GB RAM)",
	}

	if additionalMachines {
		machines["c2d-highcpu-2"] = "c2d-highcpu-2 (2vCPU, 4GB RAM)"
		machines["c2d-highcpu-4"] = "c2d-highcpu-4 (4vCPU, 8GB RAM)"
		machines["c2d-highcpu-8"] = "c2d-highcpu-8 (8vCPU, 16GB RAM)"
		machines["c2d-highcpu-16"] = "c2d-highcpu-16 (16vCPU, 32GB RAM)"
		machines["c2d-highcpu-32"] = "c2d-highcpu-32 (32vCPU, 64GB RAM)"
		machines["c2d-highcpu-56"] = "c2d-highcpu-56 (56vCPU, 112GB RAM)"
		machines["g2-standard-4"] = "g2-standard-4 (1GPU, 4vCPU, 16GB RAM)*"
		machines["g2-standard-8"] = "g2-standard-8 (1GPU, 8vCPU, 32GB RAM)*"
		machines["g2-standard-12"] = "g2-standard-12 (1GPU, 12vCPU, 48GB RAM)*"
		machines["g2-standard-16"] = "g2-standard-16 (1GPU, 16vCPU, 64GB RAM)*"
		machines["g2-standard-24"] = "g2-standard-24 (2GPU, 24vCPU, 96GB RAM)*"
		machines["g2-standard-32"] = "g2-standard-32 (1GPU, 32vCPU, 128GB RAM)*"
		machines["g2-standard-48"] = "g2-standard-48 (4GPU, 48vCPU, 192GB RAM)*"
	}

	return machines
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

func SapConvergedCloudSchema(machineTypesDisplay, regionsDisplay map[string]string, defaultOIDCConfig *pkg.OIDCConfigDTO, machineTypes []string, additionalParams, update bool, shootAndSeedFeatureFlag bool, sapConvergedCloudRegions []string, useAdditionalOIDCSchema bool) *map[string]interface{} {
	properties := NewProvisioningProperties(machineTypesDisplay, machineTypesDisplay, regionsDisplay, machineTypes, machineTypes, sapConvergedCloudRegions, update)
	return createSchemaWithProperties(properties, defaultOIDCConfig, additionalParams, update, requiredSchemaProperties(), true, shootAndSeedFeatureFlag, useAdditionalOIDCSchema)
}

func PreviewSchema(machineTypesDisplay, additionalMachineTypesDisplay, regionsDisplay map[string]string, defaultOIDCConfig *pkg.OIDCConfigDTO, machineTypes, additionalMachineTypes []string, additionalParams, update bool, euAccessRestricted bool, useAdditionalOIDCSchema bool) *map[string]interface{} {
	properties := NewProvisioningProperties(machineTypesDisplay, additionalMachineTypesDisplay, regionsDisplay, machineTypes, additionalMachineTypes, AWSRegions(euAccessRestricted), update)
	properties.Networking = NewNetworkingSchema()
	return createSchemaWithProperties(properties, defaultOIDCConfig, additionalParams, update, requiredSchemaProperties(), false, false, useAdditionalOIDCSchema)
}

func GCPSchema(machineTypesDisplay, additionalMachineTypesDisplay, regionsDisplay map[string]string, defaultOIDCConfig *pkg.OIDCConfigDTO, machineTypes, additionalMachineTypes []string, additionalParams, update bool, shootAndSeedFeatureFlag bool, assuredWorkloads bool, useAdditionalOIDCSchema bool) *map[string]interface{} {
	properties := NewProvisioningProperties(machineTypesDisplay, additionalMachineTypesDisplay, regionsDisplay, machineTypes, additionalMachineTypes, GcpRegions(assuredWorkloads), update)
	return createSchemaWithProperties(properties, defaultOIDCConfig, additionalParams, update, requiredSchemaProperties(), true, shootAndSeedFeatureFlag, useAdditionalOIDCSchema)
}

func AWSSchema(machineTypesDisplay, additionalMachineTypesDisplay, regionsDisplay map[string]string, defaultOIDCConfig *pkg.OIDCConfigDTO, machineTypes, additionalMachineTypes []string, additionalParams, update bool, euAccessRestricted bool, shootAndSeedSameRegion bool, useAdditionalOIDCSchema bool) *map[string]interface{} {
	properties := NewProvisioningProperties(machineTypesDisplay, additionalMachineTypesDisplay, regionsDisplay, machineTypes, additionalMachineTypes, AWSRegions(euAccessRestricted), update)
	return createSchemaWithProperties(properties, defaultOIDCConfig, additionalParams, update, requiredSchemaProperties(), true, shootAndSeedSameRegion, useAdditionalOIDCSchema)
}

func AzureSchema(machineTypesDisplay, additionalMachineTypesDisplay, regionsDisplay map[string]string, defaultOIDCConfig *pkg.OIDCConfigDTO, machineTypes, additionalMachineTypes []string, additionalParams, update bool, euAccessRestricted bool, shootAndSeedFeatureFlag bool, useAdditionalOIDCSchema bool) *map[string]interface{} {
	properties := NewProvisioningProperties(machineTypesDisplay, additionalMachineTypesDisplay, regionsDisplay, machineTypes, additionalMachineTypes, AzureRegions(euAccessRestricted), update)
	return createSchemaWithProperties(properties, defaultOIDCConfig, additionalParams, update, requiredSchemaProperties(), true, shootAndSeedFeatureFlag, useAdditionalOIDCSchema)
}

func AzureLiteSchema(machineTypesDisplay, regionsDisplay map[string]string, defaultOIDCConfig *pkg.OIDCConfigDTO, machineTypes []string, additionalParams, update bool, euAccessRestricted bool, shootAndSeedFeatureFlag bool, useAdditionalOIDCSchema bool) *map[string]interface{} {
	properties := NewProvisioningProperties(machineTypesDisplay, machineTypesDisplay, regionsDisplay, machineTypes, machineTypes, AzureRegions(euAccessRestricted), update)

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

	return createSchemaWithProperties(properties, defaultOIDCConfig, additionalParams, update, requiredSchemaProperties(), true, shootAndSeedFeatureFlag, useAdditionalOIDCSchema)
}

func FreemiumSchema(provider pkg.CloudProvider, defaultOIDCConfig *pkg.OIDCConfigDTO, regionsDisplay map[string]string, additionalParams, update bool, euAccessRestricted bool, useAdditionalOIDCSchema bool) *map[string]interface{} {
	if update && !additionalParams {
		return empty()
	}

	var regions []string
	switch provider {
	case pkg.AWS:
		regions = AWSRegions(euAccessRestricted)
	case pkg.Azure:
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

	return createSchemaWithProperties(properties, defaultOIDCConfig, additionalParams, update, requiredSchemaProperties(), false, false, useAdditionalOIDCSchema)
}

func TrialSchema(defaultOIDCConfig *pkg.OIDCConfigDTO, additionalParams, update, useAdditionalOIDCSchema bool) *map[string]interface{} {
	properties := ProvisioningProperties{
		Name: NameProperty(),
	}

	if !update {
		properties.Modules = NewModulesSchema()
	}

	if update && !additionalParams {
		return empty()
	}

	return createSchemaWithProperties(properties, defaultOIDCConfig, additionalParams, update, requiredTrialSchemaProperties(), false, false, useAdditionalOIDCSchema)
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

func createSchemaWithProperties(properties ProvisioningProperties, defaultOIDCConfig *pkg.OIDCConfigDTO, additionalParams, update bool, required []string, shootAndSeedSameRegion bool, shootAndSeedFeatureFlag bool, useAdditionalOIDCSchema bool) *map[string]interface{} {
	if additionalParams {
		properties.IncludeAdditional(useAdditionalOIDCSchema, defaultOIDCConfig, update)
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
func Plans(plans PlansConfig, provider pkg.CloudProvider, defaultOIDCConfig *pkg.OIDCConfigDTO, includeAdditionalParamsInSchema bool, euAccessRestricted bool, useSmallerMachineTypes bool, shootAndSeedFeatureFlag bool, sapConvergedCloudRegions []string, assuredWorkloads bool, useAdditionalOIDCSchema bool) map[string]domain.ServicePlan {
	awsMachineNames := AwsMachinesNames(false)
	awsMachinesDisplay := AwsMachinesDisplay(false)
	awsAdditionalMachineNames := AwsMachinesNames(true)
	awsAdditionalMachinesDisplay := AwsMachinesDisplay(true)
	awsRegionsDisplay := AWSRegionsDisplay(euAccessRestricted)
	azureMachinesNames := AzureMachinesNames(false)
	azureMachinesDisplay := AzureMachinesDisplay(false)
	azureAdditionalMachinesNames := AzureMachinesNames(true)
	azureAdditionalMachinesDisplay := AzureMachinesDisplay(true)
	azureRegionsDisplay := AzureRegionsDisplay(euAccessRestricted)
	azureLiteMachinesNames := AzureLiteMachinesNames()
	azureLiteMachinesDisplay := AzureLiteMachinesDisplay()
	gcpMachinesNames := GcpMachinesNames(false)
	gcpMachinesDisplay := GcpMachinesDisplay(false)
	gcpAdditionalMachinesNames := GcpMachinesNames(true)
	gcpAdditionalMachinesDisplay := GcpMachinesDisplay(true)
	gcpRegionsDisplay := GcpRegionsDisplay(assuredWorkloads)

	if !useSmallerMachineTypes {
		azureLiteMachinesNames = removeMachinesNamesFromList(azureLiteMachinesNames, "Standard_D2s_v5")
		delete(azureLiteMachinesDisplay, "Standard_D2s_v5")
	}

	awsCreateSchema := AWSSchema(awsMachinesDisplay, awsAdditionalMachinesDisplay, awsRegionsDisplay, defaultOIDCConfig, awsMachineNames, awsAdditionalMachineNames, includeAdditionalParamsInSchema, false, euAccessRestricted, shootAndSeedFeatureFlag, useAdditionalOIDCSchema)
	awsUpdateSchema := AWSSchema(awsMachinesDisplay, awsAdditionalMachinesDisplay, awsRegionsDisplay, defaultOIDCConfig, awsMachineNames, awsAdditionalMachineNames, includeAdditionalParamsInSchema, true, euAccessRestricted, shootAndSeedFeatureFlag, useAdditionalOIDCSchema)
	azureCreateSchema := AzureSchema(azureMachinesDisplay, azureAdditionalMachinesDisplay, azureRegionsDisplay, defaultOIDCConfig, azureMachinesNames, azureAdditionalMachinesNames, includeAdditionalParamsInSchema, false, euAccessRestricted, shootAndSeedFeatureFlag, useAdditionalOIDCSchema)
	azureUpdateSchema := AzureSchema(azureMachinesDisplay, azureAdditionalMachinesDisplay, azureRegionsDisplay, defaultOIDCConfig, azureMachinesNames, azureAdditionalMachinesNames, includeAdditionalParamsInSchema, true, euAccessRestricted, shootAndSeedFeatureFlag, useAdditionalOIDCSchema)
	azureLiteCreateSchema := AzureLiteSchema(azureLiteMachinesDisplay, azureRegionsDisplay, defaultOIDCConfig, azureLiteMachinesNames, includeAdditionalParamsInSchema, false, euAccessRestricted, shootAndSeedFeatureFlag, useAdditionalOIDCSchema)
	azureLiteUpdateSchema := AzureLiteSchema(azureLiteMachinesDisplay, azureRegionsDisplay, defaultOIDCConfig, azureLiteMachinesNames, includeAdditionalParamsInSchema, true, euAccessRestricted, shootAndSeedFeatureFlag, useAdditionalOIDCSchema)
	freemiumCreateSchema := FreemiumSchema(provider, defaultOIDCConfig, azureRegionsDisplay, includeAdditionalParamsInSchema, false, euAccessRestricted, useAdditionalOIDCSchema)
	freemiumUpdateSchema := FreemiumSchema(provider, defaultOIDCConfig, azureRegionsDisplay, includeAdditionalParamsInSchema, true, euAccessRestricted, useAdditionalOIDCSchema)
	gcpCreateSchema := GCPSchema(gcpMachinesDisplay, gcpAdditionalMachinesDisplay, gcpRegionsDisplay, defaultOIDCConfig, gcpMachinesNames, gcpAdditionalMachinesNames, includeAdditionalParamsInSchema, false, shootAndSeedFeatureFlag, assuredWorkloads, useAdditionalOIDCSchema)
	gcpUpdateSchema := GCPSchema(gcpMachinesDisplay, gcpAdditionalMachinesDisplay, gcpRegionsDisplay, defaultOIDCConfig, gcpMachinesNames, gcpAdditionalMachinesNames, includeAdditionalParamsInSchema, true, shootAndSeedFeatureFlag, assuredWorkloads, useAdditionalOIDCSchema)
	ownClusterCreateSchema := OwnClusterSchema(false)
	ownClusterUpdateSchema := OwnClusterSchema(true)
	previewCreateSchema := PreviewSchema(awsMachinesDisplay, awsAdditionalMachinesDisplay, awsRegionsDisplay, defaultOIDCConfig, awsMachineNames, awsAdditionalMachineNames, includeAdditionalParamsInSchema, false, euAccessRestricted, useAdditionalOIDCSchema)
	previewUpdateSchema := PreviewSchema(awsMachinesDisplay, awsAdditionalMachinesDisplay, awsRegionsDisplay, defaultOIDCConfig, awsMachineNames, awsAdditionalMachineNames, includeAdditionalParamsInSchema, true, euAccessRestricted, useAdditionalOIDCSchema)
	trialCreateSchema := TrialSchema(defaultOIDCConfig, includeAdditionalParamsInSchema, false, useAdditionalOIDCSchema)
	trialUpdateSchema := TrialSchema(defaultOIDCConfig, includeAdditionalParamsInSchema, true, useAdditionalOIDCSchema)

	outputPlans := map[string]domain.ServicePlan{
		AWSPlanID:               defaultServicePlan(AWSPlanID, AWSPlanName, plans, awsCreateSchema, awsUpdateSchema),
		GCPPlanID:               defaultServicePlan(GCPPlanID, GCPPlanName, plans, gcpCreateSchema, gcpUpdateSchema),
		AzurePlanID:             defaultServicePlan(AzurePlanID, AzurePlanName, plans, azureCreateSchema, azureUpdateSchema),
		AzureLitePlanID:         defaultServicePlan(AzureLitePlanID, AzureLitePlanName, plans, azureLiteCreateSchema, azureLiteUpdateSchema),
		FreemiumPlanID:          defaultServicePlan(FreemiumPlanID, FreemiumPlanName, plans, freemiumCreateSchema, freemiumUpdateSchema),
		TrialPlanID:             defaultServicePlan(TrialPlanID, TrialPlanName, plans, trialCreateSchema, trialUpdateSchema),
		OwnClusterPlanID:        defaultServicePlan(OwnClusterPlanID, OwnClusterPlanName, plans, ownClusterCreateSchema, ownClusterUpdateSchema),
		PreviewPlanID:           defaultServicePlan(PreviewPlanID, PreviewPlanName, plans, previewCreateSchema, previewUpdateSchema),
		BuildRuntimeAWSPlanID:   defaultServicePlan(BuildRuntimeAWSPlanID, BuildRuntimeAWSPlanName, plans, awsCreateSchema, awsUpdateSchema),
		BuildRuntimeGCPPlanID:   defaultServicePlan(BuildRuntimeGCPPlanID, BuildRuntimeGCPPlanName, plans, gcpCreateSchema, gcpUpdateSchema),
		BuildRuntimeAzurePlanID: defaultServicePlan(BuildRuntimeAzurePlanID, BuildRuntimeAzurePlanName, plans, azureCreateSchema, azureUpdateSchema),
	}

	if len(sapConvergedCloudRegions) != 0 {
		sapConvergedCloudMachinesNames := SapConvergedCloudMachinesNames()
		sapConvergedCloudMachinesDisplay := SapConvergedCloudMachinesDisplay()
		sapConvergedCloudRegionsDisplay := SapConvergedCloudRegionsDisplay()
		sapConvergedCloudCreateSchema := SapConvergedCloudSchema(sapConvergedCloudMachinesDisplay, sapConvergedCloudRegionsDisplay, defaultOIDCConfig, sapConvergedCloudMachinesNames, includeAdditionalParamsInSchema, false, shootAndSeedFeatureFlag, sapConvergedCloudRegions, useAdditionalOIDCSchema)
		sapConvergedCloudUpdateSchema := SapConvergedCloudSchema(sapConvergedCloudMachinesDisplay, sapConvergedCloudRegionsDisplay, defaultOIDCConfig, sapConvergedCloudMachinesNames, includeAdditionalParamsInSchema, true, shootAndSeedFeatureFlag, sapConvergedCloudRegions, useAdditionalOIDCSchema)
		outputPlans[SapConvergedCloudPlanID] = defaultServicePlan(SapConvergedCloudPlanID, SapConvergedCloudPlanName, plans, sapConvergedCloudCreateSchema, sapConvergedCloudUpdateSchema)
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

func removeString(slice []string, str string) []string {
	result := []string{}
	for _, v := range slice {
		if v != str {
			result = append(result, v)
		}
	}
	return result
}
