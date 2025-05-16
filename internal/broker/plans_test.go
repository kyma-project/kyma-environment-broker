package broker

import (
	"bytes"
	"encoding/json"
	"os"
	"path"
	"testing"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

func TestSchemaService_Azure(t *testing.T) {
	schemaService := createSchemaService(t)

	got := schemaService.AzureSchema("cf-ch20", false)
	validateSchema(t, Marshal(got), "azure/azure-schema-additional-params-ingress-eu.json")

	got = schemaService.AzureSchema("cf-us21", false)
	validateSchema(t, Marshal(got), "azure/azure-schema-additional-params-ingress.json")
}

func TestSchemaService_Aws(t *testing.T) {
	schemaService := createSchemaService(t)

	got := schemaService.AWSSchema("cf-eu11", false)
	validateSchema(t, Marshal(got), "aws/aws-schema-additional-params-ingress-eu.json")

	got = schemaService.AWSSchema("cf-us11", false)
	validateSchema(t, Marshal(got), "aws/aws-schema-additional-params-ingress.json")
}

func TestSchemaService_Gcp(t *testing.T) {
	schemaService := createSchemaService(t)

	got := schemaService.GCPSchema("cf-us11", false)
	validateSchema(t, Marshal(got), "gcp/gcp-schema-additional-params-ingress.json")
}

func TestSchemaService_SapConvergedCloud(t *testing.T) {
	schemaService := createSchemaService(t)

	got := schemaService.SapConvergedCloudSchema("cf-eu20", false)
	validateSchema(t, Marshal(got), "sap-converged-cloud/sap-converged-cloud-schema-additional-params-ingress.json")
}

func TestSchemaService_FreeAWS(t *testing.T) {
	schemaService := createSchemaService(t)

	got := schemaService.FreeSchema(pkg.AWS, "cf-us21", false)
	validateSchema(t, Marshal(got), "aws/free-aws-schema-additional-params-ingress.json")

	got = schemaService.FreeSchema(pkg.AWS, "cf-eu11", false)
	validateSchema(t, Marshal(got), "aws/free-aws-schema-additional-params-ingress-eu.json")
}

func TestSchemaService_FreeAzure(t *testing.T) {
	schemaService := createSchemaService(t)

	got := schemaService.FreeSchema(pkg.Azure, "cf-us21", false)
	validateSchema(t, Marshal(got), "azure/free-azure-schema-additional-params-ingress.json")

	got = schemaService.FreeSchema(pkg.Azure, "cf-ch20", false)
	validateSchema(t, Marshal(got), "azure/free-azure-schema-additional-params-ingress-eu.json")
}

func TestSchemaService_AzureLite(t *testing.T) {
	schemaService := createSchemaService(t)

	got := schemaService.AzureLiteSchema("cf-us21", false)
	validateSchema(t, Marshal(got), "azure/azure-lite-schema-additional-params-ingress.json")

	got = schemaService.AzureLiteSchema("cf-ch20", false)
	validateSchema(t, Marshal(got), "azure/azure-lite-schema-additional-params-ingress-eu.json")
}

func TestSchemaService_Trial(t *testing.T) {
	schemaService := createSchemaService(t)

	got := schemaService.TrialSchema(false)
	validateSchema(t, Marshal(got), "azure/azure-trial-schema-additional-params-ingress.json")
}

func TestSchemaGenerator(t *testing.T) {
	azureLiteMachineNamesReduced := AzureLiteMachinesNames()
	azureLiteMachinesDisplayReduced := AzureLiteMachinesDisplay()

	azureLiteMachineNamesReduced = removeMachinesNamesFromList(azureLiteMachineNamesReduced, "Standard_D2s_v5")
	delete(azureLiteMachinesDisplayReduced, "Standard_D2s_v5")

	tests := []struct {
		name                   string
		generator              func(map[string]string, map[string]string, []string, bool, bool, bool, bool) *map[string]interface{}
		machineTypes           []string
		machineTypesDisplay    map[string]string
		regionDisplay          map[string]string
		path                   string
		file                   string
		updateFile             string
		fileOIDC               string
		updateFileOIDC         string
		createIngressFiltering string
		updateIngressFiltering string
	}{
		{
			name: "AWS schema is correct",
			generator: func(machinesDisplay, regionsDisplay map[string]string, machines []string, additionalParams, update, useAdditionalOIDCSchema, ingressFiltering bool) *map[string]interface{} {
				return AWSSchema(machinesDisplay, AwsMachinesDisplay(true), regionsDisplay, nil, machines, AwsMachinesNames(true), NewControlFlagsObject(additionalParams, useAdditionalOIDCSchema, additionalParams, ingressFiltering), update, false)
			},
			machineTypes:           AwsMachinesNames(false),
			machineTypesDisplay:    AwsMachinesDisplay(false),
			regionDisplay:          AWSRegionsDisplay(false),
			path:                   "aws",
			file:                   "aws-schema.json",
			updateFile:             "update-aws-schema.json",
			fileOIDC:               "aws-schema-additional-params.json",
			updateFileOIDC:         "update-aws-schema-additional-params.json",
			createIngressFiltering: "aws-schema-additional-params-ingress.json",
			updateIngressFiltering: "update-aws-schema-additional-params-ingress.json",
		},
		{
			name: "AWS schema with EU access restriction is correct",
			generator: func(machinesDisplay, regionsDisplay map[string]string, machines []string, additionalParams, update, useAdditionalOIDCSchema, ingressFiltering bool) *map[string]interface{} {
				return AWSSchema(machinesDisplay, AwsMachinesDisplay(true), regionsDisplay, nil, machines, AwsMachinesNames(true), NewControlFlagsObject(additionalParams, useAdditionalOIDCSchema, additionalParams, ingressFiltering), update, true)
			},
			machineTypes:           AwsMachinesNames(false),
			machineTypesDisplay:    AwsMachinesDisplay(false),
			regionDisplay:          AWSRegionsDisplay(true),
			path:                   "aws",
			file:                   "aws-schema-eu.json",
			updateFile:             "update-aws-schema.json",
			fileOIDC:               "aws-schema-additional-params-eu.json",
			updateFileOIDC:         "update-aws-schema-additional-params.json",
			createIngressFiltering: "aws-schema-additional-params-ingress-eu.json",
			updateIngressFiltering: "update-aws-schema-additional-params-ingress.json",
		},
		{
			name: "Azure schema is correct",
			generator: func(machinesDisplay, regionsDisplay map[string]string, machines []string, additionalParams, update, useAdditionalOIDCSchema, ingressFiltering bool) *map[string]interface{} {
				return AzureSchema(machinesDisplay, AzureMachinesDisplay(true), regionsDisplay, nil, machines, AzureMachinesNames(true), NewControlFlagsObject(additionalParams, useAdditionalOIDCSchema, additionalParams, ingressFiltering), update, false)
			},
			machineTypes:           AzureMachinesNames(false),
			machineTypesDisplay:    AzureMachinesDisplay(false),
			regionDisplay:          AzureRegionsDisplay(false),
			path:                   "azure",
			file:                   "azure-schema.json",
			updateFile:             "update-azure-schema.json",
			fileOIDC:               "azure-schema-additional-params.json",
			updateFileOIDC:         "update-azure-schema-additional-params.json",
			createIngressFiltering: "azure-schema-additional-params-ingress.json",
			updateIngressFiltering: "update-azure-schema-additional-params-ingress.json",
		},
		{
			name: "Azure schema with EU access restriction is correct",
			generator: func(machinesDisplay, regionsDisplay map[string]string, machines []string, additionalParams, update, useAdditionalOIDCSchema, ingressFiltering bool) *map[string]interface{} {
				return AzureSchema(machinesDisplay, AzureMachinesDisplay(true), regionsDisplay, nil, machines, AzureMachinesNames(true), NewControlFlagsObject(additionalParams, useAdditionalOIDCSchema, additionalParams, ingressFiltering), update, true)
			},
			machineTypes:           AzureMachinesNames(false),
			machineTypesDisplay:    AzureMachinesDisplay(false),
			regionDisplay:          AzureRegionsDisplay(true),
			path:                   "azure",
			file:                   "azure-schema-eu.json",
			updateFile:             "update-azure-schema.json",
			fileOIDC:               "azure-schema-additional-params-eu.json",
			updateFileOIDC:         "update-azure-schema-additional-params.json",
			createIngressFiltering: "azure-schema-additional-params-ingress-eu.json",
			updateIngressFiltering: "update-azure-schema-additional-params-ingress.json",
		},
		{
			name: "AzureLite schema is correct",
			generator: func(machinesDisplay, regionsDisplay map[string]string, machines []string, additionalParams, update, useAdditionalOIDCSchema, ingressFiltering bool) *map[string]interface{} {
				return AzureLiteSchema(machinesDisplay, regionsDisplay, nil, machines, NewControlFlagsObject(additionalParams, useAdditionalOIDCSchema, additionalParams, ingressFiltering), update, false)
			},
			machineTypes:           AzureLiteMachinesNames(),
			machineTypesDisplay:    AzureLiteMachinesDisplay(),
			regionDisplay:          AzureRegionsDisplay(false),
			path:                   "azure",
			file:                   "azure-lite-schema.json",
			updateFile:             "update-azure-lite-schema.json",
			fileOIDC:               "azure-lite-schema-additional-params.json",
			updateFileOIDC:         "update-azure-lite-schema-additional-params.json",
			createIngressFiltering: "azure-lite-schema-additional-params-ingress.json",
			updateIngressFiltering: "update-azure-lite-schema-additional-params-ingress.json",
		},
		{
			name: "AzureLite reduced schema is correct",
			generator: func(machinesDisplay, regionsDisplay map[string]string, machines []string, additionalParams, update, useAdditionalOIDCSchema, ingressFiltering bool) *map[string]interface{} {
				return AzureLiteSchema(machinesDisplay, regionsDisplay, nil, machines, NewControlFlagsObject(additionalParams, useAdditionalOIDCSchema, false, ingressFiltering), update, false)
			},
			machineTypes:           azureLiteMachineNamesReduced,
			machineTypesDisplay:    azureLiteMachinesDisplayReduced,
			regionDisplay:          AzureRegionsDisplay(false),
			path:                   "azure",
			file:                   "azure-lite-schema-reduced.json",
			updateFile:             "update-azure-lite-schema-reduced.json",
			fileOIDC:               "azure-lite-schema-additional-params-reduced.json",
			updateFileOIDC:         "update-azure-lite-schema-additional-params-reduced.json",
			createIngressFiltering: "azure-lite-schema-additional-params-reduced-ingress.json",
			updateIngressFiltering: "update-azure-lite-schema-additional-params-reduced-ingress.json",
		},
		{
			name: "AzureLite schema with EU access restriction is correct",
			generator: func(machinesDisplay, regionsDisplay map[string]string, machines []string, additionalParams, update, useAdditionalOIDCSchema, ingressFiltering bool) *map[string]interface{} {
				return AzureLiteSchema(machinesDisplay, regionsDisplay, nil, machines, NewControlFlagsObject(additionalParams, useAdditionalOIDCSchema, additionalParams, ingressFiltering), update, true)
			},
			machineTypes:           AzureLiteMachinesNames(),
			machineTypesDisplay:    AzureLiteMachinesDisplay(),
			regionDisplay:          AzureRegionsDisplay(true),
			path:                   "azure",
			file:                   "azure-lite-schema-eu.json",
			updateFile:             "update-azure-lite-schema.json",
			fileOIDC:               "azure-lite-schema-additional-params-eu.json",
			updateFileOIDC:         "update-azure-lite-schema-additional-params.json",
			createIngressFiltering: "azure-lite-schema-additional-params-ingress-eu.json",
			updateIngressFiltering: "update-azure-lite-schema-additional-params-ingress.json",
		},
		{
			name: "AzureLite reduced schema with EU access restriction is correct",
			generator: func(machinesDisplay, regionsDisplay map[string]string, machines []string, additionalParams, update, useAdditionalOIDCSchema, ingressFiltering bool) *map[string]interface{} {
				return AzureLiteSchema(machinesDisplay, regionsDisplay, nil, machines, NewControlFlagsObject(additionalParams, useAdditionalOIDCSchema, false, ingressFiltering), update, true)
			},
			machineTypes:           azureLiteMachineNamesReduced,
			machineTypesDisplay:    azureLiteMachinesDisplayReduced,
			regionDisplay:          AzureRegionsDisplay(true),
			path:                   "azure",
			file:                   "azure-lite-schema-eu-reduced.json",
			updateFile:             "update-azure-lite-schema-reduced.json",
			fileOIDC:               "azure-lite-schema-additional-params-eu-reduced.json",
			updateFileOIDC:         "update-azure-lite-schema-additional-params-reduced.json",
			createIngressFiltering: "azure-lite-schema-additional-params-eu-reduced-ingress.json",
			updateIngressFiltering: "update-azure-lite-schema-additional-params-reduced-ingress.json",
		},
		{
			name: "Freemium Azure schema is correct",
			generator: func(machinesDisplay, regionsDisplay map[string]string, machines []string, additionalParams, update, useAdditionalOIDCSchema, ingressFiltering bool) *map[string]interface{} {
				return FreemiumSchema(pkg.Azure, nil, regionsDisplay, NewControlFlagsObject(additionalParams, useAdditionalOIDCSchema, false, ingressFiltering), update, false)
			},
			machineTypes:           []string{},
			regionDisplay:          AzureRegionsDisplay(false),
			path:                   "azure",
			file:                   "free-azure-schema.json",
			updateFile:             "update-free-azure-schema.json",
			fileOIDC:               "free-azure-schema-additional-params.json",
			updateFileOIDC:         "update-free-azure-schema-additional-params.json",
			createIngressFiltering: "free-azure-schema-additional-params-ingress.json",
			updateIngressFiltering: "update-free-azure-schema-additional-params-ingress.json",
		},
		{
			name: "Freemium AWS schema is correct",
			generator: func(machinesDisplay, regionsDisplay map[string]string, machines []string, additionalParams, update, useAdditionalOIDCSchema, ingressFiltering bool) *map[string]interface{} {
				return FreemiumSchema(pkg.AWS, nil, regionsDisplay, NewControlFlagsObject(additionalParams, useAdditionalOIDCSchema, false, ingressFiltering), update, false)
			},
			machineTypes:           []string{},
			regionDisplay:          AWSRegionsDisplay(false),
			path:                   "aws",
			file:                   "free-aws-schema.json",
			updateFile:             "update-free-aws-schema.json",
			fileOIDC:               "free-aws-schema-additional-params.json",
			updateFileOIDC:         "update-free-aws-schema-additional-params.json",
			createIngressFiltering: "free-aws-schema-additional-params-ingress.json",
			updateIngressFiltering: "update-free-aws-schema-additional-params-ingress.json",
		},
		{
			name: "Freemium Azure schema with EU access restriction is correct",
			generator: func(machinesDisplay, regionsDisplay map[string]string, machines []string, additionalParams, update, useAdditionalOIDCSchema, ingressFiltering bool) *map[string]interface{} {
				return FreemiumSchema(pkg.Azure, nil, regionsDisplay, NewControlFlagsObject(additionalParams, useAdditionalOIDCSchema, false, ingressFiltering), update, true)
			},
			machineTypes:           []string{},
			regionDisplay:          AzureRegionsDisplay(true),
			path:                   "azure",
			file:                   "free-azure-schema-eu.json",
			updateFile:             "update-free-azure-schema.json",
			fileOIDC:               "free-azure-schema-additional-params-eu.json",
			updateFileOIDC:         "update-free-azure-schema-additional-params.json",
			createIngressFiltering: "free-azure-schema-additional-params-ingress-eu.json",
			updateIngressFiltering: "update-free-azure-schema-additional-params-ingress.json",
		},
		{
			name: "Freemium AWS schema with EU access restriction is correct",
			generator: func(machinesDisplay, regionsDisplay map[string]string, machines []string, additionalParams, update, useAdditionalOIDCSchema, ingressFiltering bool) *map[string]interface{} {
				return FreemiumSchema(pkg.AWS, nil, regionsDisplay, NewControlFlagsObject(additionalParams, useAdditionalOIDCSchema, false, ingressFiltering), update, true)
			},
			machineTypes:           []string{},
			regionDisplay:          AWSRegionsDisplay(true),
			path:                   "aws",
			file:                   "free-aws-schema-eu.json",
			updateFile:             "update-free-aws-schema.json",
			fileOIDC:               "free-aws-schema-additional-params-eu.json",
			updateFileOIDC:         "update-free-aws-schema-additional-params.json",
			createIngressFiltering: "free-aws-schema-additional-params-ingress-eu.json",
			updateIngressFiltering: "update-free-aws-schema-additional-params-ingress.json",
		},
		{
			name: "GCP schema is correct",
			generator: func(machinesDisplay, regionsDisplay map[string]string, machines []string, additionalParams, update, useAdditionalOIDCSchema, ingressFiltering bool) *map[string]interface{} {
				return GCPSchema(machinesDisplay, GcpMachinesDisplay(true), regionsDisplay, nil, machines, GcpMachinesNames(true), NewControlFlagsObject(additionalParams, useAdditionalOIDCSchema, additionalParams, ingressFiltering), update, false)
			},
			machineTypes:           GcpMachinesNames(false),
			machineTypesDisplay:    GcpMachinesDisplay(false),
			regionDisplay:          GcpRegionsDisplay(false),
			path:                   "gcp",
			file:                   "gcp-schema.json",
			updateFile:             "update-gcp-schema.json",
			fileOIDC:               "gcp-schema-additional-params.json",
			updateFileOIDC:         "update-gcp-schema-additional-params.json",
			createIngressFiltering: "gcp-schema-additional-params-ingress.json",
			updateIngressFiltering: "update-gcp-schema-additional-params-ingress.json",
		},
		{
			name: "GCP schema with assured workloads is correct",
			generator: func(machinesDisplay, regionsDisplay map[string]string, machines []string, additionalParams, update, useAdditionalOIDCSchema, ingressFiltering bool) *map[string]interface{} {
				return GCPSchema(machinesDisplay, GcpMachinesDisplay(true), regionsDisplay, nil, machines, GcpMachinesNames(true),
					NewControlFlagsObject(additionalParams, useAdditionalOIDCSchema, additionalParams, ingressFiltering),
					update, true)
			},
			machineTypes:           GcpMachinesNames(false),
			machineTypesDisplay:    GcpMachinesDisplay(false),
			regionDisplay:          GcpRegionsDisplay(true),
			path:                   "gcp",
			file:                   "gcp-schema-assured-workloads.json",
			updateFile:             "update-gcp-schema.json",
			fileOIDC:               "gcp-schema-additional-params-assured-workloads.json",
			updateFileOIDC:         "update-gcp-schema-additional-params.json",
			createIngressFiltering: "gcp-schema-additional-params-assured-workloads-ingress.json",
			updateIngressFiltering: "update-gcp-schema-additional-params-ingress.json",
		},
		{
			name: "SapConvergedCloud schema is correct",
			generator: func(machinesDisplay, regionsDisplay map[string]string, machines []string, additionalParams, update, useAdditionalOIDCSchema, ingressFiltering bool) *map[string]interface{} {
				convergedCloudRegionProvider := &OneForAllConvergedCloudRegionsProvider{}
				return SapConvergedCloudSchema(machinesDisplay, regionsDisplay, nil, machines, NewControlFlagsObject(additionalParams, useAdditionalOIDCSchema, additionalParams, ingressFiltering),
					update, convergedCloudRegionProvider.GetRegions(""))
			},
			machineTypes:           SapConvergedCloudMachinesNames(),
			machineTypesDisplay:    SapConvergedCloudMachinesDisplay(),
			path:                   "sap-converged-cloud",
			file:                   "sap-converged-cloud-schema.json",
			updateFile:             "update-sap-converged-cloud-schema.json",
			fileOIDC:               "sap-converged-cloud-schema-additional-params.json",
			updateFileOIDC:         "update-sap-converged-cloud-schema-additional-params.json",
			createIngressFiltering: "sap-converged-cloud-schema-additional-params-ingress.json",
			updateIngressFiltering: "update-sap-converged-cloud-schema-additional-params-ingress.json",
		},
		{
			name: "Trial schema is correct",
			generator: func(machinesDisplay, regionsDisplay map[string]string, machines []string, additionalParams, update, useAdditionalOIDCSchema, ingressFiltering bool) *map[string]interface{} {
				return TrialSchema(nil, NewControlFlagsObject(additionalParams, useAdditionalOIDCSchema, false, ingressFiltering), update)
			},
			machineTypes:           []string{},
			path:                   "azure",
			file:                   "azure-trial-schema.json",
			updateFile:             "update-azure-trial-schema.json",
			fileOIDC:               "azure-trial-schema-additional-params.json",
			updateFileOIDC:         "update-azure-trial-schema-additional-params.json",
			createIngressFiltering: "azure-trial-schema-additional-params-ingress.json",
			updateIngressFiltering: "update-azure-trial-schema-additional-params-ingress.json",
		},
		{
			name: "Own cluster schema is correct",
			generator: func(machinesDisplay, regionsDisplay map[string]string, machines []string, additionalParams, update, useAdditionalOIDCSchema, ingressFiltering bool) *map[string]interface{} {
				return OwnClusterSchema(update)
			},
			machineTypes:   []string{},
			path:           ".",
			file:           "own-cluster-schema.json",
			updateFile:     "update-own-cluster-schema.json",
			fileOIDC:       "own-cluster-schema-additional-params.json",
			updateFileOIDC: "update-own-cluster-schema-additional-params.json",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.generator(tt.machineTypesDisplay, tt.regionDisplay, tt.machineTypes, false, false, false, false)
			validateSchema(t, Marshal(got), tt.path+"/"+tt.file)

			got = tt.generator(tt.machineTypesDisplay, tt.regionDisplay, tt.machineTypes, false, true, false, false)
			validateSchema(t, Marshal(got), tt.path+"/"+tt.updateFile)

			got = tt.generator(tt.machineTypesDisplay, tt.regionDisplay, tt.machineTypes, true, false, false, false)
			validateSchema(t, Marshal(got), tt.path+"/"+tt.fileOIDC)

			got = tt.generator(tt.machineTypesDisplay, tt.regionDisplay, tt.machineTypes, true, true, false, false)
			validateSchema(t, Marshal(got), tt.path+"/"+tt.updateFileOIDC)

			//additionalParams, update, useAdditionalOIDCSchema, ingressFiltering bool
			if tt.createIngressFiltering != "" {
				got = tt.generator(tt.machineTypesDisplay, tt.regionDisplay, tt.machineTypes, true, false, false, true)
				validateSchema(t, Marshal(got), tt.path+"/"+tt.createIngressFiltering)
			}

			if tt.updateIngressFiltering != "" {
				got = tt.generator(tt.machineTypesDisplay, tt.regionDisplay, tt.machineTypes, true, true, false, true)
				validateSchema(t, Marshal(got), tt.path+"/"+tt.updateIngressFiltering)
			}
		})
	}
}

func TestSapConvergedSchema(t *testing.T) {

	t.Run("SapConvergedCloud schema uses regions from parameter to display region list", func(t *testing.T) {
		// given
		regions := []string{"region1", "region2"}

		// when
		schema := Plans(nil, "", nil, false, false, false, false, regions, false, false, false, []string{})
		convergedSchema, found := schema[SapConvergedCloudPlanID]
		schemaRegionsCreate := convergedSchema.Schemas.Instance.Create.Parameters["properties"].(map[string]interface{})["region"].(map[string]interface{})["enum"]

		// then
		assert.NotNil(t, schema)
		assert.True(t, found)
		assert.Equal(t, []interface{}([]interface{}{"region1", "region2"}), schemaRegionsCreate)
	})

	t.Run("SapConvergedCloud schema not generated if empty region list", func(t *testing.T) {
		// given
		regions := []string{}

		// when
		schema := Plans(nil, "", nil, false, false, false, false, regions, false, false, false, EnablePlans{})
		_, found := schema[SapConvergedCloudPlanID]

		// then
		assert.NotNil(t, schema)
		assert.False(t, found)

		// when
		schema = Plans(nil, "", nil, false, false, false, false, nil, false, false, false, EnablePlans{})
		_, found = schema[SapConvergedCloudPlanID]

		// then
		assert.NotNil(t, schema)
		assert.False(t, found)
	})
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
	if !assert.JSONEq(t, prettyActual.String(), prettyExpected.String()) {
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

func createSchemaService(t *testing.T) *SchemaService {
	plans, err := os.Open("testdata/plans.yaml")
	require.NoError(t, err)
	defer plans.Close()

	provider, err := os.Open("testdata/providers.yaml")
	require.NoError(t, err)
	defer provider.Close()

	schemaService, err := NewSchemaService(provider, plans, nil, Config{
		IncludeAdditionalParamsInSchema: true,
		EnableShootAndSeedSameRegion:    true,
		UseAdditionalOIDCSchema:         false,
		DisableMachineTypeUpdate:        true,
	}, true, EnablePlans{TrialPlanName, AzurePlanName, AzureLitePlanName, AWSPlanName, GCPPlanName, SapConvergedCloudPlanName, FreemiumPlanName})
	require.NoError(t, err)
	return schemaService
}
