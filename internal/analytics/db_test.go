package analytics

import (
	"encoding/json"
	"testing"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseProvisioningParameters(t *testing.T) {
	params := internal.ProvisioningParameters{
		PlanID: "aws-plan",
		Parameters: pkg.ProvisioningParametersDTO{
			MachineType: strPtr("m6i.xlarge"),
		},
	}
	raw, err := json.Marshal(params)
	require.NoError(t, err)

	parsed, err := parseProvisioningParameters(string(raw))
	require.NoError(t, err)
	assert.Equal(t, "aws-plan", parsed.PlanID)
	assert.Equal(t, "m6i.xlarge", *parsed.Parameters.MachineType)
}

func TestParseProvisioningParameters_EmptyString(t *testing.T) {
	_, err := parseProvisioningParameters("")
	assert.Error(t, err)
}
