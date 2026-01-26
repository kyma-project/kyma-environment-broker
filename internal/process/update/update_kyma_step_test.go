package update

import (
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestUpdateKymaStep_PlanNotChanged(t *testing.T) {
	// given
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	kcpClient := fake.NewClientBuilder().Build()
	db := storage.NewMemoryStorage()
	operations := db.Operations()

	operation := fixture.FixUpdatingOperation("op-id", "inst-id").Operation
	operation.KymaResourceNamespace = kcpSystemNamespace
	err = operations.InsertOperation(operation)
	require.NoError(t, err)

	step := NewUpdateKymaStep(db, kcpClient, fixture.FakeKymaConfigProvider{})

	// when
	_, backoff, err := step.Run(operation, fixLogger())

	// then
	assert.Zero(t, backoff)
	assert.NoError(t, err)
}

func TestUpdateKymaStep_HappyPath(t *testing.T) {
	// given
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	kcpClient := fake.NewClientBuilder().Build()
	err = fixture.FixKymaResourceWithGivenRuntimeID(kcpClient, "kyma-system", "runtime-inst-id")
	require.NoError(t, err)
	db := storage.NewMemoryStorage()
	operations := db.Operations()

	operation := fixture.FixUpdatingOperation("op-id", "inst-id").Operation
	operation.UpdatedPlanID = broker.AWSPlanID
	operation.KymaTemplate = fixture.KymaTemplate
	err = operations.InsertOperation(operation)
	require.NoError(t, err)

	step := NewUpdateKymaStep(db, kcpClient, nil)

	// when
	_, backoff, err := step.Run(operation, fixLogger())

	// then
	assert.Zero(t, backoff)
	assert.NoError(t, err)
}

func TestUpdateKymaStep_KymaTemplateFromProvider(t *testing.T) {
	// given
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	kcpClient := fake.NewClientBuilder().Build()
	err = fixture.FixKymaResourceWithGivenRuntimeID(kcpClient, "kyma-system", "runtime-inst-id")
	require.NoError(t, err)
	db := storage.NewMemoryStorage()
	operations := db.Operations()

	operation := fixture.FixUpdatingOperation("op-id", "inst-id").Operation
	operation.UpdatedPlanID = broker.AWSPlanID
	err = operations.InsertOperation(operation)
	require.NoError(t, err)

	step := NewUpdateKymaStep(db, kcpClient, fixture.FakeKymaConfigProvider{})

	// when
	_, backoff, err := step.Run(operation, fixLogger())

	// then
	assert.Zero(t, backoff)
	assert.NoError(t, err)
}

func TestUpdateKymaStep_NoKymaNamespace(t *testing.T) {
	// given
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	kcpClient := fake.NewClientBuilder().Build()
	err = fixture.FixKymaResourceWithGivenRuntimeID(kcpClient, "kyma-system", "runtime-inst-id")
	require.NoError(t, err)
	db := storage.NewMemoryStorage()
	operations := db.Operations()

	operation := fixture.FixUpdatingOperation("op-id", "inst-id").Operation
	operation.KymaResourceNamespace = ""
	operation.UpdatedPlanID = broker.AWSPlanID
	err = operations.InsertOperation(operation)
	require.NoError(t, err)

	step := NewUpdateKymaStep(db, kcpClient, fixture.FakeKymaConfigProvider{})

	// when
	_, backoff, err := step.Run(operation, fixLogger())

	// then
	assert.Zero(t, backoff)
	assert.NoError(t, err)
}

func TestUpdateKymaStep_NoKymaNameInOperation(t *testing.T) {
	// given
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	kcpClient := fake.NewClientBuilder().Build()
	err = fixture.FixKymaResourceWithGivenRuntimeID(kcpClient, "kyma-system", "runtime-inst-id")
	require.NoError(t, err)
	db := storage.NewMemoryStorage()

	operations := db.Operations()
	operation := fixture.FixUpdatingOperation("op-id", "inst-id").Operation
	operation.UpdatedPlanID = broker.AWSPlanID
	operation.KymaResourceName = ""
	operation.RuntimeID = ""
	err = operations.InsertOperation(operation)
	require.NoError(t, err)

	instances := db.Instances()
	err = instances.Insert(internal.Instance{InstanceID: "inst-id", RuntimeID: "runtime-inst-id"})
	require.NoError(t, err)

	step := NewUpdateKymaStep(db, kcpClient, fixture.FakeKymaConfigProvider{})

	// when
	_, backoff, err := step.Run(operation, fixLogger())

	// then
	assert.Zero(t, backoff)
	assert.NoError(t, err)
}

func TestUpdateKymaStep_NoKymaNameInInstance(t *testing.T) {
	// given
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	kcpClient := fake.NewClientBuilder().Build()
	err = fixture.FixKymaResourceWithGivenRuntimeID(kcpClient, "kyma-system", "runtime-inst-id")
	require.NoError(t, err)
	db := storage.NewMemoryStorage()

	operations := db.Operations()
	operation := fixture.FixUpdatingOperation("op-id", "inst-id").Operation
	operation.UpdatedPlanID = broker.AWSPlanID
	operation.KymaResourceName = ""
	operation.RuntimeID = ""
	err = operations.InsertOperation(operation)
	require.NoError(t, err)

	instances := db.Instances()
	err = instances.Insert(internal.Instance{InstanceID: "inst-id"})
	require.NoError(t, err)

	step := NewUpdateKymaStep(db, kcpClient, fixture.FakeKymaConfigProvider{})

	// when
	_, backoff, err := step.Run(operation, fixLogger())

	// then
	assert.Zero(t, backoff)
	assert.NoError(t, err)
}

func TestUpdateKymaStep_NoKymaResource(t *testing.T) {
	// given
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	kcpClient := fake.NewClientBuilder().Build()
	db := storage.NewMemoryStorage()
	operations := db.Operations()

	operation := fixture.FixUpdatingOperation("op-id", "inst-id").Operation
	operation.UpdatedPlanID = broker.AWSPlanID
	operation.KymaTemplate = fixture.KymaTemplate
	err = operations.InsertOperation(operation)
	require.NoError(t, err)

	step := NewUpdateKymaStep(db, kcpClient, nil)

	// when
	_, backoff, err := step.Run(operation, fixLogger())

	// then
	assert.Equal(t, 10*time.Second, backoff)
	assert.NoError(t, err)
}
