package deprovisioning

import (
	"context"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/logger"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
)

func TestDeleteRuntimeResourceStep_RuntimeResourceDoesNotExists(t *testing.T) {
	// given
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	kcpClient := fake.NewClientBuilder().Build()
	op := fixture.FixDeprovisioningOperationAsOperation(fixOperationID, fixInstanceID)
	op.RuntimeResourceName = "runtime-name"
	op.KymaResourceNamespace = "kyma-ns"
	memoryStorage := storage.NewMemoryStorage()
	log := logger.NewLogDummy()

	// when
	step := NewDeleteRuntimeResourceStep(memoryStorage.Operations(), kcpClient)
	_, backoff, err := step.Run(op, log)

	// then

	assert.NoError(t, err)
	assert.Zero(t, backoff)
	assertRuntimeDoesNotExists(t, kcpClient, "kyma-ns", "runtime-name")
}

func TestDeleteRuntimeResourceStep_RuntimeResourceExists(t *testing.T) {
	// given
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	op := fixture.FixDeprovisioningOperationAsOperation(fixOperationID, fixInstanceID)
	op.RuntimeResourceName = "runtime-name"
	op.KymaResourceNamespace = "kyma-ns"
	memoryStorage := storage.NewMemoryStorage()
	kcpClient := fake.NewClientBuilder().WithRuntimeObjects(fixRuntimeResource("kyma-ns", "runtime-name")).Build()
	log := logger.NewLogDummy()

	// when
	step := NewDeleteRuntimeResourceStep(memoryStorage.Operations(), kcpClient)
	_, backoff, err := step.Run(op, log)

	// then
	assert.NoError(t, err)
	assert.Zero(t, backoff)
	assertRuntimeDoesNotExists(t, kcpClient, "kyma-ns", "runtime-name")
}

func assertRuntimeDoesNotExists(t *testing.T, kcpClient client.WithWatch, namespace string, name string) {
	err := kcpClient.Get(context.Background(), client.ObjectKey{Name: name, Namespace: namespace}, &imv1.Runtime{})
	assert.Error(t, err)
	assert.True(t, errors.IsNotFound(err))
}

func fixRuntimeResource(namespace string, name string) runtime.Object {
	return &imv1.Runtime{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}
