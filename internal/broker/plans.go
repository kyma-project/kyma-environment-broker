package broker

import (
	"strings"

	"github.com/labstack/gommon/log"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"golang.org/x/exp/maps"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
)

const (
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
	AlicloudPlanID            = "9f2c3b4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d"
	AlicloudPlanName          = "alicloud"
)

var PlanIDsMapping = map[PlanNameType]PlanIDType{
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
	AlicloudPlanName:          AlicloudPlanID,
}

type PlanIDType string
type PlanNameType string

var AvailablePlans = NewAvailablePlans(PlanIDsMapping)

type ControlFlagsObject struct {
	ingressFilteringEnabled     bool
	rejectUnsupportedParameters bool
}

type AvailablePlansType struct {
	idToName map[PlanIDType]PlanNameType
	nameToID map[PlanNameType]PlanIDType
}

func NewAvailablePlans(nameToIDMap map[PlanNameType]PlanIDType) *AvailablePlansType {
	r := reverseMap(nameToIDMap)
	if len(r) != len(nameToIDMap) {
		log.Error("plan IDs and names mapping is not bijective, cannot create AvailablePlans object")
		return nil
	}
	return &AvailablePlansType{
		idToName: r,
		nameToID: nameToIDMap,
	}
}

func reverseMap(initialMap map[PlanNameType]PlanIDType) map[PlanIDType]PlanNameType {
	reversedMap := make(map[PlanIDType]PlanNameType)
	for key, value := range initialMap {
		reversedMap[value] = key
	}
	return reversedMap
}

func (ap AvailablePlansType) GetPlanNameOrEmpty(planID PlanIDType) string {
	planName, _ := ap.idToName[planID]
	return string(planName)
}

func (ap AvailablePlansType) GetPlanNameByID(planID PlanIDType) (string, bool) {
	planName, exists := ap.idToName[planID]
	return string(planName), exists
}

func (ap AvailablePlansType) GetPlanIDByName(planName PlanNameType) (PlanIDType, bool) {
	planID, exists := ap.nameToID[planName]
	return planID, exists
}

func (ap AvailablePlansType) GetAllPlanIDs() []PlanIDType {
	return maps.Keys(ap.idToName)
}

func (ap AvailablePlansType) GetAllPlanNamesAsStrings() []string {
	names := make([]string, 0, len(ap.nameToID))
	for name := range ap.nameToID {
		names = append(names, string(name))
	}
	return names
}

func NewControlFlagsObject(ingressFilteringEnabled, rejectUnsupportedParameters bool) ControlFlagsObject {
	return ControlFlagsObject{
		ingressFilteringEnabled:     ingressFilteringEnabled,
		rejectUnsupportedParameters: rejectUnsupportedParameters,
	}
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

func requiredSchemaProperties() []string {
	return []string{"name", "region"}
}

func requiredTrialSchemaProperties() []string {
	return []string{"name"}
}

func requiredOwnClusterSchemaProperties() []string {
	return []string{"name", "kubeconfig", "shootName", "shootDomain"}
}

func createSchemaWithProperties(properties ProvisioningProperties,
	defaultOIDCConfig *pkg.OIDCConfigDTO,
	update bool,
	required []string,
	flags ControlFlagsObject) *map[string]interface{} {
	properties.OIDC = NewMultipleOIDCSchema(defaultOIDCConfig, update, flags.rejectUnsupportedParameters)
	properties.Administrators = AdministratorsProperty()
	if flags.ingressFilteringEnabled {
		properties.IngressFiltering = IngressFilteringProperty()
	}

	if update {
		return createSchemaWith(properties.UpdateProperties, []string{}, flags.rejectUnsupportedParameters)
	} else {
		return createSchemaWith(properties, required, flags.rejectUnsupportedParameters)
	}
}

func createSchemaWith(properties interface{}, required []string, rejectUnsupportedParameters bool) *map[string]interface{} {
	return unmarshalSchema(NewSchema(properties, required, rejectUnsupportedParameters))
}

func unmarshalSchema(schema *RootSchema) *map[string]interface{} {
	target := make(map[string]interface{})
	schema.ControlsOrder = DefaultControlsOrder()

	unmarshalled := unmarshalOrPanic(schema, &target).(*map[string]interface{})

	// update controls order
	props := (*unmarshalled)[PropertiesKey].(map[string]interface{})
	controlsOrder := (*unmarshalled)[ControlsOrderKey].([]interface{})
	(*unmarshalled)[ControlsOrderKey] = filter(&controlsOrder, props)

	return unmarshalled
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
