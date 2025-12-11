package postsql_test

import (
	"fmt"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
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

func TestBinding_ModeCFB(t *testing.T) {
	encrypter := storage.NewEncrypter("################################", false)
	storageCleanup, brokerStorage, err := GetStorageForDatabaseTestsWithEncrypter(encrypter)
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

	statsForBindings, err := brokerStorage.EncryptionModeStats().GetEncryptionModeStatsForBindings()
	require.NoError(t, err)

	// then
	assert.True(t, reflect.DeepEqual(map[string]int{storage.EncryptionModeCFB: 1}, statsForBindings))

	// when
	testInstanceID := "instance-" + testBindingId
	retrievedBinding, err := brokerStorage.Bindings().Get(testInstanceID, testBindingId)
	// then
	assert.NoError(t, err)
	assert.NotNil(t, retrievedBinding)
	assert.Equal(t, fixedBinding.Kubeconfig, retrievedBinding.Kubeconfig)
}

func TestBinding_ModeGCM(t *testing.T) {
	encrypter := storage.NewEncrypter("################################", false)
	encrypter.SetWriteGCMMode(true)
	storageCleanup, brokerStorage, err := GetStorageForDatabaseTestsWithEncrypter(encrypter)
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

	statsForBindings, err := brokerStorage.EncryptionModeStats().GetEncryptionModeStatsForBindings()
	require.NoError(t, err)

	// then
	assert.True(t, reflect.DeepEqual(map[string]int{storage.EncryptionModeGCM: 1}, statsForBindings))

	// when
	testInstanceID := "instance-" + testBindingId
	retrievedBinding, err := brokerStorage.Bindings().Get(testInstanceID, testBindingId)
	// then
	assert.NoError(t, err)
	assert.NotNil(t, retrievedBinding)
	assert.Equal(t, fixedBinding.Kubeconfig, retrievedBinding.Kubeconfig)
}

func TestBinding_BothModes(t *testing.T) {
	encrypter := storage.NewEncrypter("################################", false)
	storageCleanup, brokerStorage, err := GetStorageForDatabaseTestsWithEncrypter(encrypter)
	require.NoError(t, err)
	defer func() {
		err := storageCleanup()
		assert.NoError(t, err)
	}()

	// given

	instanceID := "test-instance-id"
	testBindingIdCFB := "binding-cfb"
	fixedBindingCFB := fixture.FixBindingWithInstanceID(testBindingIdCFB, instanceID)

	testBindingIdGCM := "binding-gcm"
	fixedBindingGCM := fixture.FixBindingWithInstanceID(testBindingIdGCM, instanceID)

	err = brokerStorage.Bindings().Insert(&fixedBindingCFB)
	assert.NoError(t, err)

	statsForUpdatedBindings, err := brokerStorage.EncryptionModeStats().GetEncryptionModeStatsForBindings()
	require.NoError(t, err)

	// then
	assert.True(t, reflect.DeepEqual(map[string]int{storage.EncryptionModeCFB: 1}, statsForUpdatedBindings))

	encrypter.SetWriteGCMMode(true)

	err = brokerStorage.Bindings().Insert(&fixedBindingGCM)
	assert.NoError(t, err)

	statsForUpdatedBindings, err = brokerStorage.EncryptionModeStats().GetEncryptionModeStatsForBindings()
	require.NoError(t, err)

	// then
	assert.True(t, reflect.DeepEqual(map[string]int{storage.EncryptionModeCFB: 1, storage.EncryptionModeGCM: 1}, statsForUpdatedBindings))

	// when
	retrievedBindingCFB, err := brokerStorage.Bindings().Get(instanceID, testBindingIdCFB)
	// then
	assert.NoError(t, err)
	assert.NotNil(t, retrievedBindingCFB)
	assert.Equal(t, fixedBindingCFB.Kubeconfig, retrievedBindingCFB.Kubeconfig)

	//when
	retrievedBindingGCM, err := brokerStorage.Bindings().Get(instanceID, testBindingIdGCM)
	// then
	assert.NoError(t, err)
	assert.NotNil(t, retrievedBindingGCM)
	assert.Equal(t, fixedBindingGCM.Kubeconfig, retrievedBindingGCM.Kubeconfig)

	// update bindings - the side efect is that they will be re-encrypted in the current mode
	err = brokerStorage.Bindings().Update(retrievedBindingCFB)
	assert.NoError(t, err)

	err = brokerStorage.Bindings().Update(retrievedBindingGCM)
	assert.NoError(t, err)

	retrievedUpdatedBindingCFB, err := brokerStorage.Bindings().Get(instanceID, testBindingIdCFB)
	// then
	assert.NoError(t, err)
	assert.NotNil(t, retrievedBindingCFB)
	assert.Equal(t, fixedBindingCFB.Kubeconfig, retrievedUpdatedBindingCFB.Kubeconfig)

	//when
	retrievedUpdatedBindingGCM, err := brokerStorage.Bindings().Get(instanceID, testBindingIdGCM)
	// then
	assert.NoError(t, err)
	assert.NotNil(t, retrievedBindingGCM)
	assert.Equal(t, fixedBindingGCM.Kubeconfig, retrievedUpdatedBindingGCM.Kubeconfig)

	statsForUpdatedBindings, err = brokerStorage.EncryptionModeStats().GetEncryptionModeStatsForBindings()
	require.NoError(t, err)

	// then
	assert.True(t, reflect.DeepEqual(map[string]int{storage.EncryptionModeGCM: 2}, statsForUpdatedBindings))
}

func TestListBindingsEncryptedUsingCFB_ReturnsBindingsSuccessfully(t *testing.T) {
	// given
	encrypter := storage.NewEncrypter("################################", false)
	storageCleanup, brokerStorage, err := GetStorageForDatabaseTestsWithEncrypter(encrypter)
	require.NoError(t, err)
	defer func() {
		err := storageCleanup()
		assert.NoError(t, err)
	}()

	binding1 := fixture.FixBinding("binding-1")
	binding1.Kubeconfig = "kubeconfig-1"
	binding2 := fixture.FixBinding("binding-2")
	binding2.Kubeconfig = "kubeconfig-2"

	// when
	err = brokerStorage.Bindings().Insert(&binding1)
	require.NoError(t, err)
	err = brokerStorage.Bindings().Insert(&binding2)
	require.NoError(t, err)

	// when
	bindings, err := brokerStorage.Bindings().ListBindingsEncryptedUsingCFB(10)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, len(bindings))

	// Verify bindings are present (order may vary)
	bindingMap := map[string]internal.Binding{}
	for _, b := range bindings {
		bindingMap[b.ID] = b
	}

	assert.Contains(t, bindingMap, "binding-1")
	assert.Contains(t, bindingMap, "binding-2")
	assert.Equal(t, "kubeconfig-1", bindingMap["binding-1"].Kubeconfig)
	assert.Equal(t, "kubeconfig-2", bindingMap["binding-2"].Kubeconfig)
}

func TestListBindingsEncryptedUsingCFB_ReturnsEmptyListWhenNoBindings(t *testing.T) {
	// given
	encrypter := storage.NewEncrypter("################################", false)
	storageCleanup, brokerStorage, err := GetStorageForDatabaseTestsWithEncrypter(encrypter)
	require.NoError(t, err)
	defer func() {
		err := storageCleanup()
		assert.NoError(t, err)
	}()

	// when
	bindings, err := brokerStorage.Bindings().ListBindingsEncryptedUsingCFB(10)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, len(bindings))
}

func TestListBindingsEncryptedUsingCFB_RespectsBatchSize(t *testing.T) {
	// given
	encrypter := storage.NewEncrypter("################################", false)
	storageCleanup, brokerStorage, err := GetStorageForDatabaseTestsWithEncrypter(encrypter)
	require.NoError(t, err)
	defer func() {
		err := storageCleanup()
		assert.NoError(t, err)
	}()

	// Insert 5 bindings
	for i := 1; i <= 5; i++ {
		binding := fixture.FixBinding(fmt.Sprintf("binding-%d", i))
		err = brokerStorage.Bindings().Insert(&binding)
		require.NoError(t, err)
	}

	// when
	bindings, err := brokerStorage.Bindings().ListBindingsEncryptedUsingCFB(2)

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, len(bindings))
}

func TestListBindingsEncryptedUsingCFB_HandlesEncryptedData(t *testing.T) {
	// given
	encrypter := storage.NewEncrypter("################################", false)
	storageCleanup, brokerStorage, err := GetStorageForDatabaseTestsWithEncrypter(encrypter)
	require.NoError(t, err)
	defer func() {
		err := storageCleanup()
		assert.NoError(t, err)
	}()

	binding := fixture.FixBinding("binding-1")
	binding.Kubeconfig = "encrypted-kubeconfig-data"

	// when
	err = brokerStorage.Bindings().Insert(&binding)
	require.NoError(t, err)

	// when
	bindings, err := brokerStorage.Bindings().ListBindingsEncryptedUsingCFB(10)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, len(bindings))
	assert.Equal(t, binding.Kubeconfig, bindings[0].Kubeconfig)
}

func TestListBindingsEncryptedUsingCFB_ReturnsCFBEncryptedBindingsOnly(t *testing.T) {
	// given
	encrypter := storage.NewEncrypter("################################", false)
	storageCleanup, brokerStorage, err := GetStorageForDatabaseTestsWithEncrypter(encrypter)
	require.NoError(t, err)
	defer func() {
		err := storageCleanup()
		assert.NoError(t, err)
	}()

	cfbBinding := fixture.FixBinding("binding-cfb")
	err = brokerStorage.Bindings().Insert(&cfbBinding)
	require.NoError(t, err)

	// Switch to GCM mode for next binding
	encrypter.SetWriteGCMMode(true)

	gcmBinding := fixture.FixBinding("binding-gcm")
	err = brokerStorage.Bindings().Insert(&gcmBinding)
	require.NoError(t, err)

	// when
	bindings, err := brokerStorage.Bindings().ListBindingsEncryptedUsingCFB(10)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, len(bindings))
	assert.Equal(t, "binding-cfb", bindings[0].ID)
}

func TestListBindingsEncryptedUsingCFB_PreservesBindingFields(t *testing.T) {
	// given
	encrypter := storage.NewEncrypter("################################", false)
	storageCleanup, brokerStorage, err := GetStorageForDatabaseTestsWithEncrypter(encrypter)
	require.NoError(t, err)
	defer func() {
		err := storageCleanup()
		assert.NoError(t, err)
	}()

	binding := fixture.FixBinding("binding-1")
	binding.CreatedBy = "test-user"
	binding.ExpirationSeconds = 3600

	// when
	err = brokerStorage.Bindings().Insert(&binding)
	require.NoError(t, err)

	// when
	bindings, err := brokerStorage.Bindings().ListBindingsEncryptedUsingCFB(10)

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, len(bindings))
	assert.Equal(t, "binding-1", bindings[0].ID)
	assert.Equal(t, "test-user", bindings[0].CreatedBy)
	assert.Equal(t, int64(3600), bindings[0].ExpirationSeconds)
}

func TestListBindingsEncryptedUsingCFB_HandlesMultipleBindingsPerInstance(t *testing.T) {
	// given
	encrypter := storage.NewEncrypter("################################", false)
	storageCleanup, brokerStorage, err := GetStorageForDatabaseTestsWithEncrypter(encrypter)
	require.NoError(t, err)
	defer func() {
		err := storageCleanup()
		assert.NoError(t, err)
	}()

	instanceID := "instance-1"

	// Create multiple bindings for the same instance
	for i := 1; i <= 3; i++ {
		binding := fixture.FixBinding(fmt.Sprintf("binding-%d", i))
		binding.InstanceID = instanceID
		binding.Kubeconfig = fmt.Sprintf("kubeconfig-data-%d", i)
		err = brokerStorage.Bindings().Insert(&binding)
		require.NoError(t, err)
	}

	// when
	bindings, err := brokerStorage.Bindings().ListBindingsEncryptedUsingCFB(10)

	// then
	require.NoError(t, err)
	assert.Equal(t, 3, len(bindings))

	// Verify all bindings are from the same instance
	for _, b := range bindings {
		assert.Equal(t, instanceID, b.InstanceID)
	}
}

func TestReEncryptBinding_PreservesBindingMetadata(t *testing.T) {
	// given
	encrypter := storage.NewEncrypter("################################", false)
	storageCleanup, brokerStorage, err := GetStorageForDatabaseTestsWithEncrypter(encrypter)
	require.NoError(t, err)
	defer func() {
		err := storageCleanup()
		assert.NoError(t, err)
	}()

	binding := fixture.FixBinding("binding-1")
	binding.CreatedBy = "test-user"
	binding.ExpirationSeconds = 3600

	// when
	err = brokerStorage.Bindings().Insert(&binding)
	require.NoError(t, err)

	err = brokerStorage.Bindings().ReEncryptBinding(&binding)

	// then
	require.NoError(t, err)

	retrievedBinding, err := brokerStorage.Bindings().Get(binding.InstanceID, binding.ID)
	require.NoError(t, err)
	assert.Equal(t, "binding-1", retrievedBinding.ID)
	assert.Equal(t, "test-user", retrievedBinding.CreatedBy)
	assert.Equal(t, int64(3600), retrievedBinding.ExpirationSeconds)
}

func TestReEncryptBinding_ReEncryptsFromCFBToGCM(t *testing.T) {
	// given
	encrypter := storage.NewEncrypter("################################", false)
	storageCleanup, brokerStorage, err := GetStorageForDatabaseTestsWithEncrypter(encrypter)
	require.NoError(t, err)
	defer func() {
		err := storageCleanup()
		assert.NoError(t, err)
	}()

	binding := fixture.FixBinding("binding-1")
	binding.Kubeconfig = "kubeconfig-data"

	// when
	err = brokerStorage.Bindings().Insert(&binding)
	require.NoError(t, err)

	// Verify it was inserted with CFB mode
	statsBeforeReencrypt, err := brokerStorage.EncryptionModeStats().GetEncryptionModeStatsForBindings()
	require.NoError(t, err)
	assert.Equal(t, 1, statsBeforeReencrypt[storage.EncryptionModeCFB])

	// Switch to GCM mode
	encrypter.SetWriteGCMMode(true)

	// Re-encrypt the binding
	err = brokerStorage.Bindings().ReEncryptBinding(&binding)

	// then
	require.NoError(t, err)

	// Verify it was re-encrypted with GCM mode
	statsAfterReencrypt, err := brokerStorage.EncryptionModeStats().GetEncryptionModeStatsForBindings()
	require.NoError(t, err)
	assert.Equal(t, 0, statsAfterReencrypt[storage.EncryptionModeCFB])
	assert.Equal(t, 1, statsAfterReencrypt[storage.EncryptionModeGCM])

	// Verify data is still accessible
	retrievedBinding, err := brokerStorage.Bindings().Get(binding.InstanceID, binding.ID)
	require.NoError(t, err)
	assert.Equal(t, "kubeconfig-data", retrievedBinding.Kubeconfig)
}

func TestReEncryptBinding_PreservesEncryptedDataAfterReencryption(t *testing.T) {
	// given
	encrypter := storage.NewEncrypter("################################", false)
	storageCleanup, brokerStorage, err := GetStorageForDatabaseTestsWithEncrypter(encrypter)
	require.NoError(t, err)
	defer func() {
		err := storageCleanup()
		assert.NoError(t, err)
	}()

	binding := fixture.FixBinding("binding-1")
	binding.Kubeconfig = "test-kubeconfig-data"

	// when
	err = brokerStorage.Bindings().Insert(&binding)
	require.NoError(t, err)

	// Re-encrypt
	err = brokerStorage.Bindings().ReEncryptBinding(&binding)

	// then
	require.NoError(t, err)

	retrievedBinding, err := brokerStorage.Bindings().Get(binding.InstanceID, binding.ID)
	require.NoError(t, err)
	assert.Equal(t, "test-kubeconfig-data", retrievedBinding.Kubeconfig)
}

func TestReEncryptBinding_ReturnsErrorForNonExistentBinding(t *testing.T) {
	// given
	encrypter := storage.NewEncrypter("################################", false)
	storageCleanup, brokerStorage, err := GetStorageForDatabaseTestsWithEncrypter(encrypter)
	require.NoError(t, err)
	defer func() {
		err := storageCleanup()
		assert.NoError(t, err)
	}()

	nonExistentBinding := fixture.FixBinding("non-existent-binding")

	// when
	err = brokerStorage.Bindings().ReEncryptBinding(&nonExistentBinding)

	// then
	require.Error(t, err)
}

func TestReEncryptBinding_HandlesMultipleReencryptions(t *testing.T) {
	// given
	encrypter := storage.NewEncrypter("################################", false)
	storageCleanup, brokerStorage, err := GetStorageForDatabaseTestsWithEncrypter(encrypter)
	require.NoError(t, err)
	defer func() {
		err := storageCleanup()
		assert.NoError(t, err)
	}()

	binding := fixture.FixBinding("binding-1")
	binding.Kubeconfig = "kubeconfig-data"

	// when
	err = brokerStorage.Bindings().Insert(&binding)
	require.NoError(t, err)

	// Re-encrypt first time with CFB
	err = brokerStorage.Bindings().ReEncryptBinding(&binding)
	require.NoError(t, err)

	// Verify data is accessible after first re-encryption
	retrieved1, err := brokerStorage.Bindings().Get(binding.InstanceID, binding.ID)
	require.NoError(t, err)
	assert.Equal(t, "kubeconfig-data", retrieved1.Kubeconfig)

	// Switch to GCM and re-encrypt again
	encrypter.SetWriteGCMMode(true)
	err = brokerStorage.Bindings().ReEncryptBinding(&binding)
	require.NoError(t, err)

	// Verify data is still accessible after switching encryption mode
	retrieved2, err := brokerStorage.Bindings().Get(binding.InstanceID, binding.ID)
	require.NoError(t, err)
	assert.Equal(t, "kubeconfig-data", retrieved2.Kubeconfig)
}

func TestReEncryptBinding_ReEncryptsFromCFBToGCMWithStats(t *testing.T) {
	// given
	encrypter := storage.NewEncrypter("################################", false)
	storageCleanup, brokerStorage, err := GetStorageForDatabaseTestsWithEncrypter(encrypter)
	require.NoError(t, err)
	defer func() {
		err := storageCleanup()
		assert.NoError(t, err)
	}()

	binding := fixture.FixBinding("binding-1")
	binding.Kubeconfig = "kubeconfig-data"

	// when
	err = brokerStorage.Bindings().Insert(&binding)
	require.NoError(t, err)

	// Switch to GCM mode
	encrypter.SetWriteGCMMode(true)

	// Re-encrypt the binding
	err = brokerStorage.Bindings().ReEncryptBinding(&binding)

	// then
	require.NoError(t, err)

	// Verify encryption stats changed
	statsAfterReencrypt, err := brokerStorage.EncryptionModeStats().GetEncryptionModeStatsForBindings()
	require.NoError(t, err)
	assert.Equal(t, 1, statsAfterReencrypt[storage.EncryptionModeGCM])
	assert.Equal(t, 0, statsAfterReencrypt[storage.EncryptionModeCFB])

	// Verify CFB bindings list is now empty
	bindings, err := brokerStorage.Bindings().ListBindingsEncryptedUsingCFB(10)
	require.NoError(t, err)
	assert.Equal(t, 0, len(bindings))
}
