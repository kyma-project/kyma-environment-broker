package update

import (
	"testing"

	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/process/input"
	inputAutomock "github.com/kyma-project/kyma-environment-broker/internal/process/input/automock"
	"github.com/kyma-project/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestUpgradeShootStep_Run(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()
	os := memoryStorage.Operations()
	rs := memoryStorage.RuntimeStates()
	cli := provisioner.NewFakeClient()
	step := NewUpgradeShootStep(os, rs, cli)
	operation := fixture.FixUpdatingOperation("op-id", "inst-id")
	operation.RuntimeID = "runtime-id"
	operation.ProvisionerOperationID = ""
	operation.ProvisioningParameters.ErsContext.UserID = "test-user-id"
	operation.ProvisioningParameters.Parameters.OIDC = &internal.OIDCConfigDTO{
		ClientID:       "client-id",
		GroupsClaim:    "groups",
		IssuerURL:      "https://issuer.url",
		SigningAlgs:    []string{"RSA256"},
		UsernameClaim:  "sub",
		UsernamePrefix: "-",
	}
	operation.InputCreator = fixInputCreator(t)
	err := os.InsertOperation(operation.Operation)
	require.NoError(t, err)
	runtimeState := fixture.FixRuntimeState("runtime-id", "runtime-id", "provisioning-op-1")
	runtimeState.ClusterConfig.OidcConfig = &gqlschema.OIDCConfigInput{
		ClientID:       "clientID",
		GroupsClaim:    "groupsClaim",
		IssuerURL:      "https://issuer.url",
		SigningAlgs:    []string{"PS512"},
		UsernameClaim:  "usernameClaim",
		UsernamePrefix: "usernamePrefix",
	}
	err = rs.Insert(runtimeState)
	require.NoError(t, err)

	// when
	newOperation, d, err := step.Run(operation.Operation, logrus.New())

	// then
	require.NoError(t, err)
	assert.Zero(t, d)
	assert.True(t, cli.IsShootUpgraded("runtime-id"))
	req, _ := cli.LastShootUpgrade("runtime-id")
	disabled := false
	assert.Equal(t, gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			OidcConfig: &gqlschema.OIDCConfigInput{
				ClientID:       "client-id",
				GroupsClaim:    "groups",
				IssuerURL:      "https://issuer.url",
				SigningAlgs:    []string{"RSA256"},
				UsernameClaim:  "sub",
				UsernamePrefix: "-",
			},
			ShootNetworkingFilterDisabled: &disabled,
		},
		Administrators: []string{"test-user-id"},
	}, req)
	assert.NotEmpty(t, newOperation.ProvisionerOperationID)
}

func fixInputCreator(t *testing.T) internal.ProvisionerInputCreator {
	const kymaVersion = "1.20"

	configProvider := &inputAutomock.ConfigurationProvider{}
	configProvider.On("ProvideForGivenPlan",
		mock.AnythingOfType("string"),
		mock.AnythingOfType("string")).
		Return(&internal.ConfigForPlan{}, nil)

	const k8sVersion = "1.18"
	ibf, err := input.NewInputBuilderFactory(configProvider, input.Config{
		KubernetesVersion:           k8sVersion,
		DefaultGardenerShootPurpose: "test",
	}, kymaVersion, fixTrialRegionMapping(), fixFreemiumProviders(), fixture.FixOIDCConfigDTO(), false)
	assert.NoError(t, err)

	pp := internal.ProvisioningParameters{
		PlanID:     broker.GCPPlanID,
		Parameters: internal.ProvisioningParametersDTO{},
	}
	creator, err := ibf.CreateUpgradeShootInput(pp)
	if err != nil {
		t.Errorf("cannot create input creator for %q plan", broker.GCPPlanID)
	}

	return creator
}

func fixTrialRegionMapping() map[string]string {
	return map[string]string{}
}

func fixFreemiumProviders() []string {
	return []string{"azure", "aws"}
}
