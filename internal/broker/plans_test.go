package broker

import (
	"bytes"
	"encoding/json"
	"os"
	"path"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal/provider/configuration"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

const (
	platformRegionUS11 = "cf-us11"
	platformRegionUS21 = "cf-us21"
	platformRegionEU20 = "cf-eu20"
	platformRegionEU40 = "cf-eu40"

	freeAWSPlanName   = "free-aws"
	freeAzurePlanName = "free-azure"
)

func TestSchemaService_Alicloud(t *testing.T) {
	schemaService := createSchemaService(t)

	create, update, _ := schemaService.AlicloudSchemas(platformRegionEU40)
	validateSchema(t, Marshal(create), "alicloud/alicloud-schema-additional-params-ingress.json")
	validateSchema(t, Marshal(update), "alicloud/update-alicloud-schema-additional-params-ingress.json")
}

func TestSchemaService_Azure(t *testing.T) {
	schemaService := createSchemaService(t)

	create, _, _ := schemaService.AzureSchemas("cf-ch20")
	validateSchema(t, Marshal(create), "azure/azure-schema-additional-params-ingress-eu.json")

	create, update, _ := schemaService.AzureSchemas(platformRegionUS21)
	validateSchema(t, Marshal(create), "azure/azure-schema-additional-params-ingress.json")
	validateSchema(t, Marshal(update), "azure/update-azure-schema-additional-params-ingress.json")
}

func TestSchemaService_Aws(t *testing.T) {
	schemaService := createSchemaService(t)

	create, _, _ := schemaService.AWSSchemas("cf-eu11")
	validateSchema(t, Marshal(create), "aws/aws-schema-additional-params-ingress-eu.json")

	create, update, _ := schemaService.AWSSchemas(platformRegionUS11)
	validateSchema(t, Marshal(create), "aws/aws-schema-additional-params-ingress.json")
	validateSchema(t, Marshal(update), "aws/update-aws-schema-additional-params-ingress.json")
}

func TestSchemaService_Gcp(t *testing.T) {
	schemaService := createSchemaService(t)

	create, update, _ := schemaService.GCPSchemas(platformRegionUS11)
	validateSchema(t, Marshal(create), "gcp/gcp-schema-additional-params-ingress.json")
	validateSchema(t, Marshal(update), "gcp/update-gcp-schema-additional-params-ingress.json")
}

func TestSchemaService_SapConvergedCloud(t *testing.T) {
	schemaService := createSchemaService(t)

	create, update, _ := schemaService.SapConvergedCloudSchemas(platformRegionEU20)
	validateSchema(t, Marshal(create), "sap-converged-cloud/sap-converged-cloud-schema-additional-params-ingress.json")
	validateSchema(t, Marshal(update), "sap-converged-cloud/update-sap-converged-cloud-schema-additional-params-ingress.json")
}

func TestSchemaService_FreeAWS(t *testing.T) {
	schemaService := createSchemaService(t)

	got := schemaService.FreeSchema(pkg.AWS, platformRegionUS21, false)
	validateSchema(t, Marshal(got), "aws/free-aws-schema-additional-params-ingress.json")

	got = schemaService.FreeSchema(pkg.AWS, "cf-eu11", false)
	validateSchema(t, Marshal(got), "aws/free-aws-schema-additional-params-ingress-eu.json")
}

func TestSchemaService_FreeAzure(t *testing.T) {
	schemaService := createSchemaService(t)

	got := schemaService.FreeSchema(pkg.Azure, platformRegionUS21, false)
	validateSchema(t, Marshal(got), "azure/free-azure-schema-additional-params-ingress.json")

	got = schemaService.FreeSchema(pkg.Azure, "cf-ch20", false)
	validateSchema(t, Marshal(got), "azure/free-azure-schema-additional-params-ingress-eu.json")
}

func TestSchemaService_AzureLite(t *testing.T) {
	schemaService := createSchemaService(t)

	create, update, _ := schemaService.AzureLiteSchemas(platformRegionUS21)
	validateSchema(t, Marshal(create), "azure/azure-lite-schema-additional-params-ingress.json")
	validateSchema(t, Marshal(update), "azure/update-azure-lite-schema-additional-params-ingress.json")

	create, _, _ = schemaService.AzureLiteSchemas("cf-ch20")
	validateSchema(t, Marshal(create), "azure/azure-lite-schema-additional-params-ingress-eu.json")
}

func TestSchemaService_Trial(t *testing.T) {
	schemaService := createSchemaService(t)

	got := schemaService.TrialSchema(false)
	validateSchema(t, Marshal(got), "azure/azure-trial-schema-additional-params-ingress.json")
}

func TestSchemaService_GvisorPropertyPresentInAllPlans(t *testing.T) {
	schemaService := createSchemaServiceWithGvisor(t)
	expectedGvisor := gvisorProperty()

	for _, tc := range planSchemaCases(schemaService,
		AWSPlanName, AzurePlanName, AzureLitePlanName, GCPPlanName,
		SapConvergedCloudPlanName, AlicloudPlanName, PreviewPlanName,
		freeAWSPlanName, freeAzurePlanName, TrialPlanName,
	) {
		t.Run(tc.name, func(t *testing.T) {
			schema := tc.get()
			require.NotNil(t, schema)

			props, ok := (*schema)["properties"].(map[string]interface{})
			require.True(t, ok, "schema has no 'properties' key")

			gvisor, ok := props["gvisor"]
			require.True(t, ok, "expected 'gvisor' property to be present in schema")

			assert.Equal(t, expectedGvisor, gvisor)
		})
	}
}

func TestSchemaService_GvisorInControlsOrder(t *testing.T) {
	schemaService := createSchemaServiceWithGvisor(t)

	for _, tc := range planSchemaCases(schemaService,
		AWSPlanName, AzurePlanName, AzureLitePlanName, GCPPlanName,
		SapConvergedCloudPlanName, AlicloudPlanName, PreviewPlanName,
		freeAWSPlanName, freeAzurePlanName, TrialPlanName,
	) {
		t.Run(tc.name, func(t *testing.T) {
			schema := tc.get()
			require.NotNil(t, schema)
			controlsOrderContainsGvisor(t, schema)
		})
	}
}

func TestSchemaService_GvisorAbsentWhenFeatureFlagDisabled(t *testing.T) {
	schemaService := createSchemaService(t)

	for _, tc := range planSchemaCases(schemaService,
		AWSPlanName, AzurePlanName, AzureLitePlanName, GCPPlanName,
		SapConvergedCloudPlanName, AlicloudPlanName, PreviewPlanName,
		freeAWSPlanName, freeAzurePlanName, TrialPlanName,
	) {
		t.Run(tc.name, func(t *testing.T) {
			schema := tc.get()
			require.NotNil(t, schema)

			props, ok := (*schema)["properties"].(map[string]interface{})
			require.True(t, ok, "schema has no 'properties' key")
			assert.NotContains(t, props, "gvisor")

			order, ok := (*schema)[ControlsOrderKey].([]interface{})
			require.True(t, ok, "schema has no %q key", ControlsOrderKey)
			assert.NotContains(t, order, "gvisor")
		})
	}
}

func TestSchemaService_GvisorAbsentInAdditionalWorkerNodePoolsItemProperties(t *testing.T) {
	schemaService := createSchemaService(t)

	for _, tc := range planSchemaCases(schemaService,
		AWSPlanName, AzurePlanName, AzureLitePlanName, GCPPlanName,
		SapConvergedCloudPlanName, AlicloudPlanName, PreviewPlanName,
	) {
		t.Run(tc.name, func(t *testing.T) {
			schema := tc.get()
			require.NotNil(t, schema)

			itemProps := additionalWorkerNodePoolsItemProperties(t, schema)
			assert.NotContains(t, itemProps, "gvisor")
		})
	}
}

func TestSchemaService_GvisorAbsentInAdditionalWorkerNodePoolsItemControlsOrder(t *testing.T) {
	schemaService := createSchemaService(t)

	for _, tc := range planSchemaCases(schemaService,
		AWSPlanName, AzurePlanName, AzureLitePlanName, GCPPlanName,
		SapConvergedCloudPlanName, AlicloudPlanName, PreviewPlanName,
	) {
		t.Run(tc.name, func(t *testing.T) {
			schema := tc.get()
			require.NotNil(t, schema)

			order := additionalWorkerNodePoolsItemControlsOrder(t, schema)
			assert.NotContains(t, order, "gvisor")
		})
	}
}

func TestSchemaService_GvisorPresentInAdditionalWorkerNodePoolsItemControlsOrder(t *testing.T) {
	schemaService := createSchemaServiceWithGvisor(t)

	for _, tc := range planSchemaCases(schemaService,
		AWSPlanName, AzurePlanName, AzureLitePlanName, GCPPlanName,
		SapConvergedCloudPlanName, AlicloudPlanName, PreviewPlanName,
	) {
		t.Run(tc.name, func(t *testing.T) {
			schema := tc.get()
			require.NotNil(t, schema)

			order := additionalWorkerNodePoolsItemControlsOrder(t, schema)
			assert.Contains(t, order, "gvisor")
		})
	}
}

func TestSchemaService_GvisorPresentInAdditionalWorkerNodePoolsItemProperties(t *testing.T) {
	schemaService := createSchemaServiceWithGvisor(t)
	expectedGvisor := gvisorProperty()

	for _, tc := range planSchemaCases(schemaService,
		AWSPlanName, AzurePlanName, AzureLitePlanName, GCPPlanName,
		SapConvergedCloudPlanName, AlicloudPlanName, PreviewPlanName,
	) {
		t.Run(tc.name, func(t *testing.T) {
			schema := tc.get()
			require.NotNil(t, schema)

			itemProps := additionalWorkerNodePoolsItemProperties(t, schema)

			gvisor, ok := itemProps["gvisor"]
			require.True(t, ok, "expected 'gvisor' to be present in additionalWorkerNodePools item properties")
			assert.Equal(t, expectedGvisor, gvisor)
		})
	}
}

type planSchemaEntry struct {
	create func() *map[string]interface{}
	update func() *map[string]interface{}
}

func planSchemaCases(svc *SchemaService, planNames ...string) []struct {
	name string
	get  func() *map[string]interface{}
} {
	registry := map[string]planSchemaEntry{
		AWSPlanName: {
			create: func() *map[string]interface{} { s, _, _ := svc.AWSSchemas(platformRegionUS11); return s },
			update: func() *map[string]interface{} { _, s, _ := svc.AWSSchemas(platformRegionUS11); return s },
		},
		AzurePlanName: {
			create: func() *map[string]interface{} { s, _, _ := svc.AzureSchemas(platformRegionUS21); return s },
			update: func() *map[string]interface{} { _, s, _ := svc.AzureSchemas(platformRegionUS21); return s },
		},
		AzureLitePlanName: {
			create: func() *map[string]interface{} { s, _, _ := svc.AzureLiteSchemas(platformRegionUS21); return s },
			update: func() *map[string]interface{} { _, s, _ := svc.AzureLiteSchemas(platformRegionUS21); return s },
		},
		GCPPlanName: {
			create: func() *map[string]interface{} { s, _, _ := svc.GCPSchemas(platformRegionUS11); return s },
			update: func() *map[string]interface{} { _, s, _ := svc.GCPSchemas(platformRegionUS11); return s },
		},
		SapConvergedCloudPlanName: {
			create: func() *map[string]interface{} { s, _, _ := svc.SapConvergedCloudSchemas(platformRegionEU20); return s },
			update: func() *map[string]interface{} { _, s, _ := svc.SapConvergedCloudSchemas(platformRegionEU20); return s },
		},
		AlicloudPlanName: {
			create: func() *map[string]interface{} { s, _, _ := svc.AlicloudSchemas(platformRegionEU40); return s },
			update: func() *map[string]interface{} { _, s, _ := svc.AlicloudSchemas(platformRegionEU40); return s },
		},
		PreviewPlanName: {
			create: func() *map[string]interface{} { s, _, _ := svc.PreviewSchemas(platformRegionUS11); return s },
			update: func() *map[string]interface{} { _, s, _ := svc.PreviewSchemas(platformRegionUS11); return s },
		},
		freeAWSPlanName: {
			create: func() *map[string]interface{} { return svc.FreeSchema(pkg.AWS, platformRegionUS21, false) },
			update: func() *map[string]interface{} { return svc.FreeSchema(pkg.AWS, platformRegionUS21, true) },
		},
		freeAzurePlanName: {
			create: func() *map[string]interface{} { return svc.FreeSchema(pkg.Azure, platformRegionUS21, false) },
			update: func() *map[string]interface{} { return svc.FreeSchema(pkg.Azure, platformRegionUS21, true) },
		},
		TrialPlanName: {
			create: func() *map[string]interface{} { return svc.TrialSchema(false) },
			update: func() *map[string]interface{} { return svc.TrialSchema(true) },
		},
	}

	var cases []struct {
		name string
		get  func() *map[string]interface{}
	}
	for _, planName := range planNames {
		entry := registry[planName]
		cases = append(cases,
			struct {
				name string
				get  func() *map[string]interface{}
			}{planName + "-create", entry.create},
			struct {
				name string
				get  func() *map[string]interface{}
			}{planName + "-update", entry.update},
		)
	}
	return cases
}

func additionalWorkerNodePoolsItemProperties(t *testing.T, schema *map[string]interface{}) map[string]interface{} {
	t.Helper()

	items := additionalWorkerNodePoolsItems(t, schema)

	itemProps, ok := items["properties"].(map[string]interface{})
	require.True(t, ok, "'additionalWorkerNodePools.items' has no 'properties' key")

	return itemProps
}

func additionalWorkerNodePoolsItemControlsOrder(t *testing.T, schema *map[string]interface{}) []interface{} {
	t.Helper()

	items := additionalWorkerNodePoolsItems(t, schema)

	order, ok := items["_controlsOrder"].([]interface{})
	require.True(t, ok, "'additionalWorkerNodePools.items' has no '_controlsOrder' key")

	return order
}

func additionalWorkerNodePoolsItems(t *testing.T, schema *map[string]interface{}) map[string]interface{} {
	t.Helper()

	props, ok := (*schema)["properties"].(map[string]interface{})
	require.True(t, ok, "schema has no 'properties' key")

	awnp, ok := props["additionalWorkerNodePools"].(map[string]interface{})
	require.True(t, ok, "schema has no 'additionalWorkerNodePools' property")

	items, ok := awnp["items"].(map[string]interface{})
	require.True(t, ok, "'additionalWorkerNodePools' has no 'items' key")

	return items
}

func controlsOrderContainsGvisor(t *testing.T, schema *map[string]interface{}) {
	t.Helper()
	raw, ok := (*schema)[ControlsOrderKey].([]interface{})
	require.True(t, ok, "schema has no %q key", ControlsOrderKey)
	for _, v := range raw {
		if s, ok := v.(string); ok && s == "gvisor" {
			return
		}
	}
	t.Fatalf("%q does not contain %q", ControlsOrderKey, "gvisor")
}

func gvisorProperty() map[string]interface{} {
	return map[string]interface{}{
		"type":        "object",
		"description": "Configures the gVisor container runtime for a worker pool",
		"required":    []interface{}{"enabled"},
		"properties": map[string]interface{}{
			"enabled": map[string]interface{}{
				"type":    "boolean",
				"title":   "Enable gVisor container runtime",
				"default": false,
			},
		},
	}
}

func validateSchema(t *testing.T, actual []byte, file string) {
	var prettyExpected bytes.Buffer
	expected := readJsonFile(t, file)
	if len(expected) > 0 {
		err := json.Indent(&prettyExpected, []byte(expected), "", "  ")
		if err != nil {
			t.Error(err)
			t.Fail()
		}
	}

	var prettyActual bytes.Buffer
	if len(actual) > 0 {
		err := json.Indent(&prettyActual, actual, "", "  ")
		if err != nil {
			t.Error(err)
			t.Fail()
		}
	}
	if !assert.JSONEq(t, prettyExpected.String(), prettyActual.String()) {
		t.Errorf("%v Schema() = \n######### Actual ###########%v\n######### End Actual ########, expected \n##### Expected #####%v\n##### End Expected #####", file, prettyActual.String(), prettyExpected.String())
	}
}

func readJsonFile(t *testing.T, file string) string {
	t.Helper()

	filename := path.Join("testdata", file)
	jsonFile, err := os.ReadFile(filename)
	require.NoError(t, err)

	return string(jsonFile)
}

func TestRemoveString(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		remove   string
		expected []string
	}{
		{"Remove existing element", []string{"alpha", "beta", "gamma"}, "beta", []string{"alpha", "gamma"}},
		{"Remove non-existing element", []string{"alpha", "beta", "gamma"}, "delta", []string{"alpha", "beta", "gamma"}},
		{"Remove from empty slice", []string{}, "alpha", []string{}},
		{"Remove all occurrences", []string{"alpha", "alpha", "beta"}, "alpha", []string{"beta"}},
		{"Remove only element", []string{"alpha"}, "alpha", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeString(tt.input, tt.remove)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReverseMap_ReversesKeyValuePairs(t *testing.T) {
	input := map[PlanNameType]PlanIDType{"a": "1", "b": "2"}
	got := reverseMap(input)
	expected := map[PlanIDType]PlanNameType{"1": "a", "2": "b"}
	assert.Equal(t, expected, got)
}

func TestAvailablePlans_GetPlanIDByName_ReturnsIDWhenExistsAndFalseWhenNot(t *testing.T) {
	ap := NewAvailablePlans(PlanIDsMapping)
	id, ok := ap.GetPlanIDByName(AzurePlanName)
	assert.True(t, ok)
	assert.Equal(t, AzurePlanID, string(id))

	id, ok = ap.GetPlanIDByName("non-existent-name")
	assert.False(t, ok)
	assert.Empty(t, id)
}

func TestNewAvailablePlans_NonBijectiveMappingReturnsEmptyAvailablePlans(t *testing.T) {
	// create a map where two names map to the same ID (not bijective)
	nameToID := map[PlanNameType]PlanIDType{"planA": "id1", "planB": "id1"}
	ap := NewAvailablePlans(nameToID)
	assert.Nil(t, ap)
}

func createSchemaService(t *testing.T) *SchemaService {
	return createSchemaServiceWithConfig(t, Config{
		RejectUnsupportedParameters: true,
		EnablePlanUpgrades:          true,
		DualStackDocsURL:            "https://placeholder.com",
		ACLEnabledPlans:             []string{"gcp"},
	})
}

func createSchemaServiceWithGvisor(t *testing.T) *SchemaService {
	return createSchemaServiceWithConfig(t, Config{
		RejectUnsupportedParameters: true,
		EnablePlanUpgrades:          true,
		DualStackDocsURL:            "https://placeholder.com",
		GvisorEnabled:               true,
	})
}

func createSchemaServiceWithConfig(t *testing.T, cfg Config) *SchemaService {
	plans, err := configuration.NewPlanSpecificationsFromFile("testdata/plans.yaml")
	require.NoError(t, err)

	provider, err := configuration.NewProviderSpecFromFile("testdata/providers.yaml")
	require.NoError(t, err)

	channelResolver := &fixture.FakeChannelResolver{}

	return NewSchemaService(provider, plans, nil, cfg,
		StringList{TrialPlanName, AzurePlanName, AzureLitePlanName, AWSPlanName, GCPPlanName, SapConvergedCloudPlanName, FreemiumPlanName, AlicloudPlanName},
		channelResolver)
}
