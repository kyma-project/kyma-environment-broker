package broker

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"path"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

func TestSchemaGenerator(t *testing.T) {
	modulesEnabled := true
	tests := []struct {
		name                string
		generator           func(map[string]string, []string, bool, bool) *map[string]interface{}
		machineTypes        []string
		machineTypesDisplay map[string]string
		path                string
		file                string
		updateFile          string
		fileOIDC            string
		updateFileOIDC      string
	}{
		{
			name: "AWS schema is correct",
			generator: func(machinesDisplay map[string]string, machines []string, additionalParams, update bool) *map[string]interface{} {
				return AWSSchema(machinesDisplay, machines, additionalParams, update, false, false, modulesEnabled)
			},
			machineTypes:   []string{"m5.xlarge", "m5.2xlarge", "m5.4xlarge", "m5.8xlarge", "m5.12xlarge", "m6i.xlarge", "m6i.2xlarge", "m6i.4xlarge", "m6i.8xlarge", "m6i.12xlarge"},
			path:           "aws",
			file:           "aws-schema.json",
			updateFile:     "update-aws-schema.json",
			fileOIDC:       "aws-schema-additional-params.json",
			updateFileOIDC: "update-aws-schema-additional-params.json",
		},
		{
			name: "AWS schema with region required is correct",
			generator: func(machinesDisplay map[string]string, machines []string, additionalParams, update bool) *map[string]interface{} {
				return AWSSchema(machinesDisplay, machines, additionalParams, update, false, true, modulesEnabled)
			},
			machineTypes:   []string{"m5.xlarge", "m5.2xlarge", "m5.4xlarge", "m5.8xlarge", "m5.12xlarge", "m6i.xlarge", "m6i.2xlarge", "m6i.4xlarge", "m6i.8xlarge", "m6i.12xlarge"},
			path:           "aws",
			file:           "aws-schema-region-required.json",
			updateFile:     "update-aws-schema.json",
			fileOIDC:       "aws-schema-additional-params-region-required.json",
			updateFileOIDC: "update-aws-schema-additional-params.json",
		},
		{
			name: "AWS schema with EU access restriction is correct",
			generator: func(machinesDisplay map[string]string, machines []string, additionalParams, update bool) *map[string]interface{} {
				return AWSSchema(machinesDisplay, machines, additionalParams, update, true, false, modulesEnabled)
			},
			machineTypes:   []string{"m5.xlarge", "m5.2xlarge", "m5.4xlarge", "m5.8xlarge", "m5.12xlarge", "m6i.xlarge", "m6i.2xlarge", "m6i.4xlarge", "m6i.8xlarge", "m6i.12xlarge"},
			path:           "aws",
			file:           "aws-schema-eu.json",
			updateFile:     "update-aws-schema.json",
			fileOIDC:       "aws-schema-additional-params-eu.json",
			updateFileOIDC: "update-aws-schema-additional-params.json",
		},
		{
			name: "AWS schema with region required and EU access restriction is correct",
			generator: func(machinesDisplay map[string]string, machines []string, additionalParams, update bool) *map[string]interface{} {
				return AWSSchema(machinesDisplay, machines, additionalParams, update, true, true, modulesEnabled)
			},
			machineTypes:   []string{"m5.xlarge", "m5.2xlarge", "m5.4xlarge", "m5.8xlarge", "m5.12xlarge", "m6i.xlarge", "m6i.2xlarge", "m6i.4xlarge", "m6i.8xlarge", "m6i.12xlarge"},
			path:           "aws",
			file:           "aws-schema-eu-region-required.json",
			updateFile:     "update-aws-schema.json",
			fileOIDC:       "aws-schema-additional-params-eu-region-required.json",
			updateFileOIDC: "update-aws-schema-additional-params.json",
		},
		{
			name: "Azure schema is correct",
			generator: func(machinesDisplay map[string]string, machines []string, additionalParams, update bool) *map[string]interface{} {
				return AzureSchema(machinesDisplay, machines, additionalParams, update, false, false, modulesEnabled)
			},
			machineTypes:   []string{"Standard_D4_v3", "Standard_D8_v3", "Standard_D16_v3", "Standard_D32_v3", "Standard_D48_v3", "Standard_D64_v3"},
			path:           "azure",
			file:           "azure-schema.json",
			updateFile:     "update-azure-schema.json",
			fileOIDC:       "azure-schema-additional-params.json",
			updateFileOIDC: "update-azure-schema-additional-params.json",
		},
		{
			name: "Azure schema with region required is correct",
			generator: func(machinesDisplay map[string]string, machines []string, additionalParams, update bool) *map[string]interface{} {
				return AzureSchema(machinesDisplay, machines, additionalParams, update, false, true, modulesEnabled)
			},
			machineTypes:   []string{"Standard_D4_v3", "Standard_D8_v3", "Standard_D16_v3", "Standard_D32_v3", "Standard_D48_v3", "Standard_D64_v3"},
			path:           "azure",
			file:           "azure-schema-region-required.json",
			updateFile:     "update-azure-schema.json",
			fileOIDC:       "azure-schema-additional-params-region-required.json",
			updateFileOIDC: "update-azure-schema-additional-params.json",
		},
		{
			name: "Azure schema with EU access restriction is correct",
			generator: func(machinesDisplay map[string]string, machines []string, additionalParams, update bool) *map[string]interface{} {
				return AzureSchema(machinesDisplay, machines, additionalParams, update, true, false, modulesEnabled)
			},
			machineTypes:   []string{"Standard_D4_v3", "Standard_D8_v3", "Standard_D16_v3", "Standard_D32_v3", "Standard_D48_v3", "Standard_D64_v3"},
			path:           "azure",
			file:           "azure-schema-eu.json",
			updateFile:     "update-azure-schema.json",
			fileOIDC:       "azure-schema-additional-params-eu.json",
			updateFileOIDC: "update-azure-schema-additional-params.json",
		},
		{
			name: "Azure schema with region required and EU access restriction is correct",
			generator: func(machinesDisplay map[string]string, machines []string, additionalParams, update bool) *map[string]interface{} {
				return AzureSchema(machinesDisplay, machines, additionalParams, update, true, true, modulesEnabled)
			},
			machineTypes:   []string{"Standard_D4_v3", "Standard_D8_v3", "Standard_D16_v3", "Standard_D32_v3", "Standard_D48_v3", "Standard_D64_v3"},
			path:           "azure",
			file:           "azure-schema-eu-region-required.json",
			updateFile:     "update-azure-schema.json",
			fileOIDC:       "azure-schema-additional-params-eu-region-required.json",
			updateFileOIDC: "update-azure-schema-additional-params.json",
		},
		{
			name: "AzureLite schema is correct",
			generator: func(machinesDisplay map[string]string, machines []string, additionalParams, update bool) *map[string]interface{} {
				return AzureLiteSchema(machinesDisplay, machines, additionalParams, update, false, false, modulesEnabled)
			},
			machineTypes:        []string{"Standard_D4_v3"},
			machineTypesDisplay: map[string]string{"Standard_D4_v3": "Standard_D4_v3 (4vCPU, 16GB RAM)"},
			path:                "azure",
			file:                "azure-lite-schema.json",
			updateFile:          "update-azure-lite-schema.json",
			fileOIDC:            "azure-lite-schema-additional-params.json",
			updateFileOIDC:      "update-azure-lite-schema-additional-params.json",
		},
		{
			name: "AzureLite schema with region required is correct",
			generator: func(machinesDisplay map[string]string, machines []string, additionalParams, update bool) *map[string]interface{} {
				return AzureLiteSchema(machinesDisplay, machines, additionalParams, update, false, true, modulesEnabled)
			},
			machineTypes:        []string{"Standard_D4_v3"},
			machineTypesDisplay: map[string]string{"Standard_D4_v3": "Standard_D4_v3 (4vCPU, 16GB RAM)"},
			path:                "azure",
			file:                "azure-lite-schema-region-required.json",
			updateFile:          "update-azure-lite-schema.json",
			fileOIDC:            "azure-lite-schema-additional-params-region-required.json",
			updateFileOIDC:      "update-azure-lite-schema-additional-params.json",
		},
		{
			name: "AzureLite schema with EU access restriction is correct",
			generator: func(machinesDisplay map[string]string, machines []string, additionalParams, update bool) *map[string]interface{} {
				return AzureLiteSchema(machinesDisplay, machines, additionalParams, update, true, false, modulesEnabled)
			},
			machineTypes:        []string{"Standard_D4_v3"},
			machineTypesDisplay: map[string]string{"Standard_D4_v3": "Standard_D4_v3 (4vCPU, 16GB RAM)"},
			path:                "azure",
			file:                "azure-lite-schema-eu.json",
			updateFile:          "update-azure-lite-schema.json",
			fileOIDC:            "azure-lite-schema-additional-params-eu.json",
			updateFileOIDC:      "update-azure-lite-schema-additional-params.json",
		},
		{
			name: "AzureLite schema with region required and EU access restriction is correct",
			generator: func(machinesDisplay map[string]string, machines []string, additionalParams, update bool) *map[string]interface{} {
				return AzureLiteSchema(machinesDisplay, machines, additionalParams, update, true, true, modulesEnabled)
			},
			path:                "azure",
			machineTypes:        []string{"Standard_D4_v3"},
			machineTypesDisplay: map[string]string{"Standard_D4_v3": "Standard_D4_v3 (4vCPU, 16GB RAM)"},
			file:                "azure-lite-schema-eu-region-required.json",
			updateFile:          "update-azure-lite-schema.json",
			fileOIDC:            "azure-lite-schema-additional-params-eu-region-required.json",
			updateFileOIDC:      "update-azure-lite-schema-additional-params.json",
		},
		{
			name: "Freemium schema is correct",
			generator: func(machinesDisplay map[string]string, machines []string, additionalParams, update bool) *map[string]interface{} {
				return FreemiumSchema(internal.Azure, additionalParams, update, false, false, modulesEnabled)
			},
			machineTypes:   []string{},
			path:           "azure",
			file:           "free-azure-schema.json",
			updateFile:     "update-free-azure-schema.json",
			fileOIDC:       "free-azure-schema-additional-params.json",
			updateFileOIDC: "update-free-azure-schema-additional-params.json",
		},
		{
			name: "Freemium schema with region required is correct",
			generator: func(machinesDisplay map[string]string, machines []string, additionalParams, update bool) *map[string]interface{} {
				return FreemiumSchema(internal.Azure, additionalParams, update, false, true, modulesEnabled)
			},
			machineTypes:   []string{},
			path:           "azure",
			file:           "free-azure-schema-region-required.json",
			updateFile:     "update-free-azure-schema.json",
			fileOIDC:       "free-azure-schema-additional-params-region-required.json",
			updateFileOIDC: "update-free-azure-schema-additional-params.json",
		},
		{
			name: " Freemium schema is correct",
			generator: func(machinesDisplay map[string]string, machines []string, additionalParams, update bool) *map[string]interface{} {
				return FreemiumSchema(internal.AWS, additionalParams, update, false, false, modulesEnabled)
			},
			machineTypes:   []string{},
			path:           "aws",
			file:           "free-aws-schema.json",
			updateFile:     "update-free-aws-schema.json",
			fileOIDC:       "free-aws-schema-additional-params.json",
			updateFileOIDC: "update-free-aws-schema-additional-params.json",
		},
		{
			name: "Freemium schema with region required is correct",
			generator: func(machinesDisplay map[string]string, machines []string, additionalParams, update bool) *map[string]interface{} {
				return FreemiumSchema(internal.AWS, additionalParams, update, false, true, modulesEnabled)
			},
			machineTypes:   []string{},
			path:           "aws",
			file:           "free-aws-schema-region-required.json",
			updateFile:     "update-free-aws-schema.json",
			fileOIDC:       "free-aws-schema-additional-params-region-required.json",
			updateFileOIDC: "update-free-aws-schema-additional-params.json",
		},
		{
			name: "Freemium schema with EU access restriction is correct",
			generator: func(machinesDisplay map[string]string, machines []string, additionalParams, update bool) *map[string]interface{} {
				return FreemiumSchema(internal.Azure, additionalParams, update, true, false, modulesEnabled)
			},
			machineTypes:   []string{},
			path:           "azure",
			file:           "free-azure-schema-eu.json",
			updateFile:     "update-free-azure-schema.json",
			fileOIDC:       "free-azure-schema-additional-params-eu.json",
			updateFileOIDC: "update-free-azure-schema-additional-params.json",
		},
		{
			name: "Freemium schema with region required and EU access restriction is correct",
			generator: func(machinesDisplay map[string]string, machines []string, additionalParams, update bool) *map[string]interface{} {
				return FreemiumSchema(internal.Azure, additionalParams, update, true, true, modulesEnabled)
			},
			machineTypes:   []string{},
			path:           "azure",
			file:           "free-azure-schema-eu-region-required.json",
			updateFile:     "update-free-azure-schema.json",
			fileOIDC:       "free-azure-schema-additional-params-eu-region-required.json",
			updateFileOIDC: "update-free-azure-schema-additional-params.json",
		},
		{
			name: "Freemium schema with EU access restriction is correct",
			generator: func(machinesDisplay map[string]string, machines []string, additionalParams, update bool) *map[string]interface{} {
				return FreemiumSchema(internal.AWS, additionalParams, update, true, false, modulesEnabled)
			},
			machineTypes:   []string{},
			path:           "aws",
			file:           "free-aws-schema-eu.json",
			updateFile:     "update-free-aws-schema.json",
			fileOIDC:       "free-aws-schema-additional-params-eu.json",
			updateFileOIDC: "update-free-aws-schema-additional-params.json",
		},
		{
			name: "Freemium schema with region required and EU access restriction is correct",
			generator: func(machinesDisplay map[string]string, machines []string, additionalParams, update bool) *map[string]interface{} {
				return FreemiumSchema(internal.AWS, additionalParams, update, true, true, modulesEnabled)
			},
			machineTypes:   []string{},
			path:           "aws",
			file:           "free-aws-schema-eu-region-required.json",
			updateFile:     "update-free-aws-schema.json",
			fileOIDC:       "free-aws-schema-additional-params-eu-region-required.json",
			updateFileOIDC: "update-free-aws-schema-additional-params.json",
		},
		{
			name: "GCP schema is correct",
			generator: func(machinesDisplay map[string]string, machines []string, additionalParams, update bool) *map[string]interface{} {
				return GCPSchema(machinesDisplay, machines, additionalParams, update, false, modulesEnabled)
			},
			machineTypes:   []string{"n2-standard-4", "n2-standard-8", "n2-standard-16", "n2-standard-32", "n2-standard-48"},
			path:           "gcp",
			file:           "gcp-schema.json",
			updateFile:     "update-gcp-schema.json",
			fileOIDC:       "gcp-schema-additional-params.json",
			updateFileOIDC: "update-gcp-schema-additional-params.json",
		},
		{
			name: "GCP schema with region required is correct",
			generator: func(machinesDisplay map[string]string, machines []string, additionalParams, update bool) *map[string]interface{} {
				return GCPSchema(machinesDisplay, machines, additionalParams, update, true, modulesEnabled)
			},
			machineTypes:   []string{"n2-standard-4", "n2-standard-8", "n2-standard-16", "n2-standard-32", "n2-standard-48"},
			path:           "gcp",
			file:           "gcp-schema-region-required.json",
			updateFile:     "update-gcp-schema.json",
			fileOIDC:       "gcp-schema-additional-params-region-required.json",
			updateFileOIDC: "update-gcp-schema-additional-params.json",
		},
		{
			name: "SapConvergedCloud schema is correct",
			generator: func(machinesDisplay map[string]string, machines []string, additionalParams, update bool) *map[string]interface{} {
				return SapConvergedCloudSchema(machinesDisplay, machines, additionalParams, update, false, modulesEnabled)
			},
			machineTypes:   []string{"g_c4_m16", "g_c6_m24", "g_c8_m32", "g_c12_m48", "g_c16_m64", "g_c32_m128", "g_c64_m256"},
			path:           "sap-converged-cloud",
			file:           "sap-converged-cloud-schema.json",
			updateFile:     "update-sap-converged-cloud-schema.json",
			fileOIDC:       "sap-converged-cloud-schema-additional-params.json",
			updateFileOIDC: "update-sap-converged-cloud-schema-additional-params.json",
		},
		{
			name: "SapConvergedCloud schema with region required is correct",
			generator: func(machinesDisplay map[string]string, machines []string, additionalParams, update bool) *map[string]interface{} {
				return SapConvergedCloudSchema(machinesDisplay, machines, additionalParams, update, true, modulesEnabled)
			},
			machineTypes:   []string{"g_c4_m16", "g_c6_m24", "g_c8_m32", "g_c12_m48", "g_c16_m64", "g_c32_m128", "g_c64_m256"},
			path:           "sap-converged-cloud",
			file:           "sap-converged-cloud-schema-region-required.json",
			updateFile:     "update-sap-converged-cloud-schema.json",
			fileOIDC:       "sap-converged-cloud-schema-additional-params-region-required.json",
			updateFileOIDC: "update-sap-converged-cloud-schema-additional-params.json",
		},
		{
			name: "Trial schema is correct",
			generator: func(machinesDisplay map[string]string, machines []string, additionalParams, update bool) *map[string]interface{} {
				return TrialSchema(additionalParams, update, modulesEnabled)
			},
			machineTypes:   []string{},
			path:           "azure",
			file:           "azure-trial-schema.json",
			updateFile:     "update-azure-trial-schema.json",
			fileOIDC:       "azure-trial-schema-additional-params.json",
			updateFileOIDC: "update-azure-trial-schema-additional-params.json",
		},
		{
			name: "Own cluster schema is correct",
			generator: func(machinesDisplay map[string]string, machines []string, additionalParams, update bool) *map[string]interface{} {
				return OwnClusterSchema(update, modulesEnabled)
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
			got := tt.generator(tt.machineTypesDisplay, tt.machineTypes, false, false)
			validateSchema(t, Marshal(got), tt.path+"/"+tt.file)

			got = tt.generator(tt.machineTypesDisplay, tt.machineTypes, false, true)
			validateSchema(t, Marshal(got), tt.path+"/"+tt.updateFile)

			got = tt.generator(tt.machineTypesDisplay, tt.machineTypes, true, false)
			validateSchema(t, Marshal(got), tt.path+"/"+tt.fileOIDC)

			got = tt.generator(tt.machineTypesDisplay, tt.machineTypes, true, true)
			validateSchema(t, Marshal(got), tt.path+"/"+tt.updateFileOIDC)
		})
	}
}

func validateSchema(t *testing.T, got []byte, file string) {
	var prettyWant bytes.Buffer
	want := readJsonFile(t, file)
	if len(want) > 0 {
		err := json.Indent(&prettyWant, []byte(want), "", "  ")
		if err != nil {
			t.Error(err)
			t.Fail()
		}
	}

	var prettyGot bytes.Buffer
	if len(got) > 0 {
		err := json.Indent(&prettyGot, got, "", "  ")
		if err != nil {
			t.Error(err)
			t.Fail()
		}
	}
	if !assert.JSONEq(t, prettyGot.String(), prettyWant.String()) {
		t.Errorf("%v Schema() = \n######### GOT ###########%v\n######### ENDGOT ########, want \n##### WANT #####%v\n##### ENDWANT #####", file, prettyGot.String(), prettyWant.String())
	}
}

func readJsonFile(t *testing.T, file string) string {
	t.Helper()

	filename := path.Join("testdata", file)
	yamlFile, err := ioutil.ReadFile(filename)
	require.NoError(t, err)

	return string(yamlFile)
}
