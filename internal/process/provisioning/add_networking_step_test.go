package provisioning

import (
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddNetworkingStep_HappyPath(t *testing.T) {
	// given
	st := storage.NewMemoryStorage()
	step := NewAddNetworkingStep(st.Operations())

	op := fixture.FixProvisioningOperation("op-id", "instance-id")
	require.NoError(t, st.Operations().InsertOperation(op))

	// when
	result, retry, err := step.Run(op, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, retry)
	_ = result
}
