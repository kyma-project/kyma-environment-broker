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

func TestSchemaService_Alicloud(t *testing.T) {
	schemaService := createSchemaService(t)

	create, update, _ := schemaService.AlicloudSchemas("cf-eu40")
	validateSchema(t, Marshal(create), "alicloud/alicloud-schema-additional-params-ingress.json")
	validateSchema(t, Marshal(update), "alicloud/update-alicloud-schema-additional-params-ingress.json")
}

func TestSchemaService_Azure(t *testing.T) {
	schemaService := createSchemaService(t)

	create, _, _ := schemaService.AzureSchemas("cf-ch20")
	validateSchema(t, Marshal(create), "azure/azure-schema-additional-params-ingress-eu.json")

	create, update, _ := schemaService.AzureSchemas("cf-us21")
	validateSchema(t, Marshal(create), "azure/azure-schema-additional-params-ingress.json")
	validateSchema(t, Marshal(update), "azure/update-azure-schema-additional-params-ingress.json")
}

func TestSchemaService_Aws(t *testing.T) {
	schemaService := createSchemaService(t)

	create, update, _ := schemaService.AWSSchemas("cf-eu11")
	validateSchema(t, Marshal(create), "aws/aws-schema-additional-params-ingress-eu.json")

	create, update, _ = schemaService.AWSSchemas("cf-us11")
	validateSchema(t, Marshal(create), "aws/aws-schema-additional-params-ingress.json")
	validateSchema(t, Marshal(update), "aws/update-aws-schema-additional-params-ingress.json")
}

func TestSchemaService_Gcp(t *testing.T) {
	schemaService := createSchemaService(t)

	create, update, _ := schemaService.GCPSchemas("cf-us11")
	validateSchema(t, Marshal(create), "gcp/gcp-schema-additional-params-ingress.json")
	validateSchema(t, Marshal(update), "gcp/update-gcp-schema-additional-params-ingress.json")
}

func TestSchemaService_SapConvergedCloud(t *testing.T) {
	schemaService := createSchemaService(t)

	create, update, _ := schemaService.SapConvergedCloudSchemas("cf-eu20")
	validateSchema(t, Marshal(create), "sap-converged-cloud/sap-converged-cloud-schema-additional-params-ingress.json")
	validateSchema(t, Marshal(update), "sap-converged-cloud/update-sap-converged-cloud-schema-additional-params-ingress.json")
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

	create, update, _ := schemaService.AzureLiteSchemas("cf-us21")
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
	plans, err := configuration.NewPlanSpecificationsFromFile("testdata/plans.yaml")
	require.NoError(t, err)

	provider, err := configuration.NewProviderSpecFromFile("testdata/providers.yaml")
	require.NoError(t, err)

	channelResolver := &fixture.FakeChannelResolver{}

	schemaService := NewSchemaService(provider, plans, nil, Config{
		RejectUnsupportedParameters: true,
		EnablePlanUpgrades:          true,
		DualStackDocsURL:            "https://placeholder.com",
	}, EnablePlans{TrialPlanName, AzurePlanName, AzureLitePlanName, AWSPlanName, GCPPlanName, SapConvergedCloudPlanName, FreemiumPlanName, AlicloudPlanName}, channelResolver)
	return schemaService
}
