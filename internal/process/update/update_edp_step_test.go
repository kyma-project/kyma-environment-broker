package update

import (
	"strings"
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/edp"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateEdp(t *testing.T) {
	// given
	db := storage.NewMemoryStorage()
	ops := db.Operations()
	cli := edp.NewFakeClient()
	operation := fixture.FixUpdatingOperation("opid", "iid").Operation
	operation.UpdatedPlanID = broker.BuildRuntimeAWSPlanID
	sid := strings.ToLower(operation.ProvisioningParameters.ErsContext.SubAccountID)
	err := cli.CreateMetadataTenant(sid, "test-env", edp.MetadataTenantPayload{
		Key:   edp.MaasConsumerServicePlan,
		Value: "standard",
	}, fixLogger())
	require.NoError(t, err, "expected no error during metadata tenant creation")
	step := NewEDPUpdateStep(ops, edp.Config{
		Required:    false,
		Disabled:    false,
		Environment: "test-env",
	}, cli)

	// when
	_, backoff, err := step.Run(operation, fixLogger())
	require.NoError(t, err, "expected no error during EDP update step")
	assert.Equal(t, time.Duration(0), backoff, "expected no backoff time")

	// then
	got, found := cli.GetMetadataItem(sid, "test-env", edp.MaasConsumerServicePlan)
	require.True(t, found)
	assert.Equal(t, "build-runtime", got.Value, "expected service plan to be updated to 'build-runtime'")
}

func TestUpdateEdp_skipIfNoPlanChange(t *testing.T) {
	// given
	db := storage.NewMemoryStorage()
	ops := db.Operations()
	operation := fixture.FixUpdatingOperation("opid", "iid").Operation
	step := NewEDPUpdateStep(ops, edp.Config{
		Required:    false,
		Disabled:    false,
		Environment: "test-env",
	}, nil) // edp client must not be used

	// when
	_, backoff, err := step.Run(operation, fixLogger())
	require.NoError(t, err, "expected no error during EDP update step")

	// then
	assert.Equal(t, time.Duration(0), backoff, "expected no backoff time")

}
