package deprovisioning

import (
	"testing"

	reconcilerApi "github.com/kyma-incubator/reconciler/pkg/keb"
	"github.com/kyma-project/kyma-environment-broker/internal/reconciler"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeregisterClusterStep_Run(t *testing.T) {
	// given
	cli := reconciler.NewFakeClient()
	memoryStorage := storage.NewMemoryStorage()
	step := NewDeregisterClusterStep(memoryStorage.Operations(), cli)
	op := fixDeprovisioningOperation()
	op.ClusterConfigurationVersion = 1
	err := memoryStorage.Operations().InsertDeprovisioningOperation(op)
	require.NoError(t, err)
	op.RuntimeID = "runtime-id"
	_, err = cli.ApplyClusterConfig(reconcilerApi.Cluster{
		RuntimeID: op.RuntimeID,
	})
	require.NoError(t, err)

	// when
	_, d, err := step.Run(op.Operation, logrus.New())

	// then
	require.NoError(t, err)
	assert.Zero(t, d)
	assert.True(t, cli.IsBeingDeleted(op.RuntimeID))
}

func TestDeregisterClusterStep_RunForNotExistingCluster(t *testing.T) {
	// given
	cli := reconciler.NewFakeClient()
	memoryStorage := storage.NewMemoryStorage()
	step := NewDeregisterClusterStep(memoryStorage.Operations(), cli)
	op := fixDeprovisioningOperation()
	op.ClusterConfigurationVersion = 1
	op.ClusterConfigurationDeleted = true
	err := memoryStorage.Operations().InsertDeprovisioningOperation(op)
	require.NoError(t, err)
	op.RuntimeID = "runtime-id"

	// when
	_, d, err := step.Run(op.Operation, logrus.New())

	// then
	require.NoError(t, err)
	assert.Zero(t, d)
	assert.False(t, cli.IsBeingDeleted(op.RuntimeID))
}
