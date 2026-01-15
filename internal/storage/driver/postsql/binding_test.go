package postsql_test

import (
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBinding(t *testing.T) {

	t.Run("should create, load and delete binding without errors", func(t *testing.T) {
		storageCleanup, brokerStorage, err := GetStorageForDatabaseTests()
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()

		// given
		testBindingId := "test"
		fixedBinding := fixture.FixBinding(testBindingId)

		err = brokerStorage.Bindings().Insert(&fixedBinding)
		assert.NoError(t, err)

		// when
		testInstanceID := "instance-" + testBindingId
		createdBinding, err := brokerStorage.Bindings().Get(testInstanceID, testBindingId)

		// then
		assert.NoError(t, err)
		assert.NotNil(t, createdBinding.ID)
		assert.Equal(t, fixedBinding.ID, createdBinding.ID)
		assert.NotNil(t, createdBinding.InstanceID)
		assert.Equal(t, fixedBinding.InstanceID, createdBinding.InstanceID)
		assert.NotNil(t, createdBinding.ExpirationSeconds)
		assert.Equal(t, fixedBinding.ExpirationSeconds, createdBinding.ExpirationSeconds)
		assert.NotNil(t, createdBinding.Kubeconfig)
		assert.Equal(t, fixedBinding.Kubeconfig, createdBinding.Kubeconfig)
		assert.Equal(t, fixedBinding.CreatedBy, createdBinding.CreatedBy)

		// when
		err = brokerStorage.Bindings().Delete(testInstanceID, testBindingId)

		// then
		nonExisting, err := brokerStorage.Bindings().Get("instance-"+testBindingId, testBindingId)
		assert.Error(t, err)
		assert.Nil(t, nonExisting)
	})

	t.Run("should return error when the same object inserted twice", func(t *testing.T) {
		storageCleanup, brokerStorage, err := GetStorageForDatabaseTests()
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()

		// given
		testBindingId := "test"
		fixedBinding := fixture.FixBinding(testBindingId)

		// when
		err = brokerStorage.Bindings().Insert(&fixedBinding)
		assert.NoError(t, err)

		err = brokerStorage.Bindings().Insert(&fixedBinding)

		// then
		assert.Error(t, err)
	})

	t.Run("should succeed when the same object is deleted twice", func(t *testing.T) {
		storageCleanup, brokerStorage, err := GetStorageForDatabaseTests()
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()

		// given
		testBindingId := "test"
		fixedBinding := fixture.FixBinding(testBindingId)

		err = brokerStorage.Bindings().Insert(&fixedBinding)
		assert.NoError(t, err)

		err = brokerStorage.Bindings().Delete(fixedBinding.InstanceID, fixedBinding.ID)
		assert.NoError(t, err)

		// then
		err = brokerStorage.Bindings().Delete(fixedBinding.InstanceID, fixedBinding.ID)
		assert.NoError(t, err)
	})

	t.Run("should list all created bindings", func(t *testing.T) {
		storageCleanup, brokerStorage, err := GetStorageForDatabaseTests()
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()

		// given
		sameInstanceID := uuid.New().String()
		fixedBinding := fixture.FixBindingWithInstanceID("1", sameInstanceID)
		err = brokerStorage.Bindings().Insert(&fixedBinding)
		assert.NoError(t, err)

		fixedBinding = fixture.FixBindingWithInstanceID("2", sameInstanceID)
		err = brokerStorage.Bindings().Insert(&fixedBinding)
		assert.NoError(t, err)

		fixedBinding = fixture.FixBindingWithInstanceID("3", sameInstanceID)
		err = brokerStorage.Bindings().Insert(&fixedBinding)
		assert.NoError(t, err)

		// when
		bindings, err := brokerStorage.Bindings().ListByInstanceID(sameInstanceID)

		// then
		assert.NoError(t, err)
		assert.Len(t, bindings, 3)
	})

	t.Run("should return bindings only for given instance", func(t *testing.T) {
		storageCleanup, brokerStorage, err := GetStorageForDatabaseTests()
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()

		// given
		sameInstanceID := uuid.New().String()
		differentInstanceID := uuid.New().String()
		fixedBinding := fixture.FixBindingWithInstanceID("1", sameInstanceID)
		err = brokerStorage.Bindings().Insert(&fixedBinding)
		assert.NoError(t, err)

		fixedBinding = fixture.FixBindingWithInstanceID("2", sameInstanceID)
		err = brokerStorage.Bindings().Insert(&fixedBinding)
		assert.NoError(t, err)

		fixedBinding = fixture.FixBindingWithInstanceID("3", differentInstanceID)
		err = brokerStorage.Bindings().Insert(&fixedBinding)
		assert.NoError(t, err)

		// when
		bindings, err := brokerStorage.Bindings().ListByInstanceID(sameInstanceID)

		// then
		assert.NoError(t, err)
		assert.Len(t, bindings, 2)

		for _, binding := range bindings {
			assert.Equal(t, sameInstanceID, binding.InstanceID)
		}
	})

	t.Run("should return empty list if no bindings exist for given instance", func(t *testing.T) {
		storageCleanup, brokerStorage, err := GetStorageForDatabaseTests()
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()

		// given
		sameInstanceID := uuid.New().String()
		fixedBinding := fixture.FixBindingWithInstanceID("1", sameInstanceID)
		err = brokerStorage.Bindings().Insert(&fixedBinding)
		assert.NoError(t, err)

		fixedBinding = fixture.FixBindingWithInstanceID("2", sameInstanceID)
		err = brokerStorage.Bindings().Insert(&fixedBinding)
		assert.NoError(t, err)

		fixedBinding = fixture.FixBindingWithInstanceID("3", sameInstanceID)
		err = brokerStorage.Bindings().Insert(&fixedBinding)
		assert.NoError(t, err)

		// when
		bindings, err := brokerStorage.Bindings().ListByInstanceID(uuid.New().String())

		// then
		assert.NoError(t, err)
		assert.Len(t, bindings, 0)

		for _, binding := range bindings {
			assert.Equal(t, sameInstanceID, binding.InstanceID)
		}
	})

	t.Run("should return empty list if no bindings exist for given instance", func(t *testing.T) {
		storageCleanup, brokerStorage, err := GetStorageForDatabaseTests()
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()

		// given
		sameInstanceID := uuid.New().String()

		// when
		bindings, err := brokerStorage.Bindings().ListByInstanceID(sameInstanceID)

		// then
		assert.NoError(t, err)
		assert.Len(t, bindings, 0)

		for _, binding := range bindings {
			assert.Equal(t, sameInstanceID, binding.InstanceID)
		}
	})
}

func TestBindingMetrics(t *testing.T) {
	storageCleanup, brokerStorage, err := GetStorageForDatabaseTests()
	require.NoError(t, err)
	require.NotNil(t, brokerStorage)
	defer func() {
		err := storageCleanup()
		assert.NoError(t, err)
	}()

	// given

	binding1 := fixture.FixBinding("binding1")
	binding1.ExpiresAt = time.Now().Add(1 * time.Hour)
	binding2 := fixture.FixBinding("binding2")
	binding2.ExpiresAt = time.Now().Add(-2 * time.Hour)

	require.NoError(t, brokerStorage.Bindings().Insert(&binding1))
	require.NoError(t, brokerStorage.Bindings().Insert(&binding2))

	// when
	got, err := brokerStorage.Bindings().GetStatistics()

	// then
	require.NoError(t, err)
	// assert if the expiration time is close to 120 minutes
	assert.Less(t, math.Abs(got.MinutesSinceEarliestExpiration-120.0), 0.05)
}

func TestBindingMetrics_NoBindings(t *testing.T) {
	storageCleanup, brokerStorage, err := GetStorageForDatabaseTests()
	require.NoError(t, err)
	require.NotNil(t, brokerStorage)
	defer func() {
		err := storageCleanup()
		assert.NoError(t, err)
	}()

	// when
	got, err := brokerStorage.Bindings().GetStatistics()

	// then
	require.NoError(t, err)

	// in case of no bindings, the metric should be 0
	assert.Equal(t, got.MinutesSinceEarliestExpiration, 0.0)
}

func TestBinding_ModeGCM(t *testing.T) {
	storageCleanup, brokerStorage, err := GetStorageForDatabaseTests()
	require.NoError(t, err)
	defer func() {
		err := storageCleanup()
		assert.NoError(t, err)
	}()

	// given
	testBindingId := "test"
	fixedBinding := fixture.FixBinding(testBindingId)

	// when
	err = brokerStorage.Bindings().Insert(&fixedBinding)
	assert.NoError(t, err)

	// when
	testInstanceID := "instance-" + testBindingId
	retrievedBinding, err := brokerStorage.Bindings().Get(testInstanceID, testBindingId)
	// then
	assert.NoError(t, err)
	assert.NotNil(t, retrievedBinding)
	assert.Equal(t, fixedBinding.Kubeconfig, retrievedBinding.Kubeconfig)
}
