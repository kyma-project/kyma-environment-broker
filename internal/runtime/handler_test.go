package runtime_test

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/httputil"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kyma-project/kyma-environment-broker/internal/storage"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/kyma-environment-broker/internal/runtime"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/rand"
)

const (
	testID1 = "Test1"
	testID2 = "Test2"
	testID3 = "Test3"
	testID4 = "Test4"
)

func TestRuntimeHandler(t *testing.T) {
	k8sClient := fake.NewClientBuilder().Build()
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	t.Run("test pagination should work", func(t *testing.T) {
		// given

		db := storage.NewMemoryStorage()
		instances := db.Instances()
		testTime1 := time.Now()
		testTime2 := time.Now().Add(time.Minute)
		testInstance1 := internal.Instance{
			InstanceID: testID1,
			CreatedAt:  testTime1,
			Parameters: internal.ProvisioningParameters{},
		}
		testInstance2 := internal.Instance{
			InstanceID: testID2,
			CreatedAt:  testTime2,
			Parameters: internal.ProvisioningParameters{},
		}

		err := instances.Insert(testInstance1)
		require.NoError(t, err)
		err = instances.Insert(testInstance2)
		require.NoError(t, err)

		runtimeHandler := runtime.NewHandler(db, 2, "", k8sClient, log)

		req, err := http.NewRequest("GET", "/runtimes?page_size=1", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		router := httputil.NewRouter()
		runtimeHandler.AttachRoutes(router)

		// when
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		var out pkg.RuntimesPage

		err = json.Unmarshal(rr.Body.Bytes(), &out)
		require.NoError(t, err)

		assert.Equal(t, 2, out.TotalCount)
		assert.Equal(t, 1, out.Count)
		assert.Equal(t, testID1, out.Data[0].InstanceID)

		// given
		urlPath := fmt.Sprintf("/runtimes?page=2&page_size=1")
		req, err = http.NewRequest(http.MethodGet, urlPath, nil)
		require.NoError(t, err)
		rr = httptest.NewRecorder()

		// when
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		err = json.Unmarshal(rr.Body.Bytes(), &out)
		require.NoError(t, err)
		assert.Equal(t, 2, out.TotalCount)
		assert.Equal(t, 1, out.Count)
		assert.Equal(t, testID2, out.Data[0].InstanceID)

	})

	t.Run("test validation should work", func(t *testing.T) {
		// given

		db := storage.NewMemoryStorage()

		runtimeHandler := runtime.NewHandler(db, 2, "region", k8sClient, log)

		req, err := http.NewRequest("GET", "/runtimes?page_size=a", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		router := httputil.NewRouter()
		runtimeHandler.AttachRoutes(router)

		router.ServeHTTP(rr, req)

		require.Equal(t, http.StatusBadRequest, rr.Code)

		req, err = http.NewRequest("GET", "/runtimes?page_size=1,2,3", nil)
		require.NoError(t, err)

		rr = httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		require.Equal(t, http.StatusBadRequest, rr.Code)

		req, err = http.NewRequest("GET", "/runtimes?page_size=abc", nil)
		require.NoError(t, err)

		rr = httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		require.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("test filtering should work", func(t *testing.T) {
		// given
		db := storage.NewMemoryStorage()
		operations := db.Operations()
		instances := db.Instances()
		testTime1 := time.Now()
		testTime2 := time.Now().Add(time.Minute)
		testInstance1 := fixInstance(testID1, testTime1)
		testInstance2 := fixInstance(testID2, testTime2)
		testInstance1.InstanceDetails = fixture.FixInstanceDetails(testID1)
		testInstance2.InstanceDetails = fixture.FixInstanceDetails(testID2)
		testOp1 := fixture.FixProvisioningOperation("op1", testID1)
		testOp2 := fixture.FixProvisioningOperation("op2", testID2)

		err := instances.Insert(testInstance1)
		require.NoError(t, err)
		err = instances.Insert(testInstance2)
		require.NoError(t, err)
		err = operations.InsertOperation(testOp1)
		require.NoError(t, err)
		err = operations.InsertOperation(testOp2)
		require.NoError(t, err)

		runtimeHandler := runtime.NewHandler(db, 2, "", k8sClient, log)

		req, err := http.NewRequest("GET", fmt.Sprintf("/runtimes?account=%s&subaccount=%s&instance_id=%s&runtime_id=%s&region=%s&shoot=%s", testID1, testID1, testID1, testID1, testID1, fmt.Sprintf("Shoot-%s", testID1)), nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		router := httputil.NewRouter()
		runtimeHandler.AttachRoutes(router)

		// when
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		var out pkg.RuntimesPage

		err = json.Unmarshal(rr.Body.Bytes(), &out)
		require.NoError(t, err)

		assert.Equal(t, 1, out.TotalCount)
		assert.Equal(t, 1, out.Count)
		assert.Equal(t, testID1, out.Data[0].InstanceID)
	})

	t.Run("test state filtering should work", func(t *testing.T) {
		// given
		db := storage.NewMemoryStorage()
		operations := db.Operations()
		instances := db.Instances()
		testTime1 := time.Now()
		testTime2 := time.Now().Add(time.Minute)
		testInstance1 := fixInstance(testID1, testTime1)
		testInstance2 := fixInstance(testID2, testTime2)
		testInstance3 := fixInstance(testID3, time.Now().Add(2*time.Minute))

		err := instances.Insert(testInstance1)
		require.NoError(t, err)
		err = instances.Insert(testInstance2)
		require.NoError(t, err)
		err = instances.Insert(testInstance3)
		require.NoError(t, err)

		provOp1 := fixture.FixProvisioningOperation(fixRandomID(), testID1)
		err = operations.InsertOperation(provOp1)
		require.NoError(t, err)

		provOp2 := fixture.FixProvisioningOperation(fixRandomID(), testID2)
		err = operations.InsertOperation(provOp2)
		require.NoError(t, err)
		updOp2 := fixture.FixUpdatingOperation(fixRandomID(), testID2)
		updOp2.State = domain.Failed
		updOp2.CreatedAt = updOp2.CreatedAt.Add(time.Minute)
		err = operations.InsertUpdatingOperation(updOp2)
		require.NoError(t, err)

		provOp3 := fixture.FixProvisioningOperation(fixRandomID(), testID3)
		err = operations.InsertOperation(provOp3)
		require.NoError(t, err)
		updOp3 := fixture.FixUpdatingOperation(fixRandomID(), testID3)
		updOp3.State = domain.Failed
		updOp3.CreatedAt = updOp3.CreatedAt.Add(time.Minute)
		err = operations.InsertUpdatingOperation(updOp3)
		require.NoError(t, err)
		deprovOp3 := fixture.FixDeprovisioningOperation(fixRandomID(), testID3)
		deprovOp3.State = domain.Succeeded
		deprovOp3.CreatedAt = deprovOp3.CreatedAt.Add(2 * time.Minute)
		err = operations.InsertDeprovisioningOperation(deprovOp3)
		require.NoError(t, err)

		runtimeHandler := runtime.NewHandler(db, 2, "", k8sClient, log)

		rr := httptest.NewRecorder()
		router := httputil.NewRouter()
		runtimeHandler.AttachRoutes(router)

		// when
		req, err := http.NewRequest("GET", fmt.Sprintf("/runtimes?state=%s", pkg.StateSucceeded), nil)
		require.NoError(t, err)
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		var out pkg.RuntimesPage

		err = json.Unmarshal(rr.Body.Bytes(), &out)
		require.NoError(t, err)

		assert.Equal(t, 1, out.TotalCount)
		assert.Equal(t, 1, out.Count)
		assert.Equal(t, testID1, out.Data[0].InstanceID)

		// when
		rr = httptest.NewRecorder()
		req, err = http.NewRequest("GET", fmt.Sprintf("/runtimes?state=%s", pkg.StateError), nil)
		require.NoError(t, err)
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		err = json.Unmarshal(rr.Body.Bytes(), &out)
		require.NoError(t, err)

		assert.Equal(t, 1, out.TotalCount)
		assert.Equal(t, 1, out.Count)
		assert.Equal(t, testID2, out.Data[0].InstanceID)

		rr = httptest.NewRecorder()
		req, err = http.NewRequest("GET", fmt.Sprintf("/runtimes?state=%s", pkg.StateFailed), nil)
		require.NoError(t, err)
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		err = json.Unmarshal(rr.Body.Bytes(), &out)
		require.NoError(t, err)

		assert.Equal(t, 0, out.TotalCount)
		assert.Equal(t, 0, out.Count)
		assert.Len(t, out.Data, 0)
	})

	t.Run("should show suspension and unsuspension operations", func(t *testing.T) {
		// given
		db := storage.NewMemoryStorage()
		operations := db.Operations()
		instances := db.Instances()
		testTime1 := time.Now()
		testInstance1 := fixInstance(testID1, testTime1)

		unsuspensionOpId := "unsuspension-op-id"
		suspensionOpId := "suspension-op-id"

		err := instances.Insert(testInstance1)
		require.NoError(t, err)

		err = operations.InsertProvisioningOperation(internal.ProvisioningOperation{
			Operation: internal.Operation{
				ID:         "first-provisioning-id",
				Version:    0,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
				InstanceID: testID1,
				Type:       internal.OperationTypeProvision,
			},
		})
		require.NoError(t, err)
		err = operations.InsertProvisioningOperation(internal.ProvisioningOperation{
			Operation: internal.Operation{
				ID:         unsuspensionOpId,
				Version:    0,
				CreatedAt:  time.Now().Add(1 * time.Hour),
				UpdatedAt:  time.Now().Add(1 * time.Hour),
				InstanceID: testID1,
				Type:       internal.OperationTypeProvision,
			},
		})

		require.NoError(t, err)
		err = operations.InsertDeprovisioningOperation(internal.DeprovisioningOperation{
			Operation: internal.Operation{
				ID:         suspensionOpId,
				Version:    0,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
				InstanceID: testID1,
				Temporary:  true,
				Type:       internal.OperationTypeDeprovision,
			},
		})
		require.NoError(t, err)

		runtimeHandler := runtime.NewHandler(db, 2, "", k8sClient, log)

		req, err := http.NewRequest("GET", "/runtimes", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		router := httputil.NewRouter()
		runtimeHandler.AttachRoutes(router)

		// when
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		var out pkg.RuntimesPage

		err = json.Unmarshal(rr.Body.Bytes(), &out)
		require.NoError(t, err)

		assert.Equal(t, 1, out.TotalCount)
		assert.Equal(t, 1, out.Count)
		assert.Equal(t, testID1, out.Data[0].InstanceID)

		unsuspensionOps := out.Data[0].Status.Unsuspension.Data
		assert.Equal(t, 1, len(unsuspensionOps))
		assert.Equal(t, unsuspensionOpId, unsuspensionOps[0].OperationID)

		suspensionOps := out.Data[0].Status.Suspension.Data
		assert.Equal(t, 1, len(suspensionOps))
		assert.Equal(t, suspensionOpId, suspensionOps[0].OperationID)
	})

	t.Run("should distinguish between provisioning & unsuspension operations", func(t *testing.T) {
		// given
		db := storage.NewMemoryStorage()
		operations := db.Operations()
		instances := db.Instances()
		testInstance1 := fixture.FixInstance("instance-1")

		provisioningOpId := "provisioning-op-id"
		unsuspensionOpId := "unsuspension-op-id"

		err := instances.Insert(testInstance1)
		require.NoError(t, err)

		err = operations.InsertProvisioningOperation(internal.ProvisioningOperation{
			Operation: internal.Operation{
				ID:         provisioningOpId,
				Version:    0,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
				InstanceID: testInstance1.InstanceID,
				Type:       internal.OperationTypeProvision,
			},
		})
		require.NoError(t, err)

		err = operations.InsertProvisioningOperation(internal.ProvisioningOperation{
			Operation: internal.Operation{
				ID:         unsuspensionOpId,
				Version:    0,
				CreatedAt:  time.Now().Add(1 * time.Hour),
				UpdatedAt:  time.Now().Add(1 * time.Hour),
				InstanceID: testInstance1.InstanceID,
				Type:       internal.OperationTypeProvision,
			},
		})
		require.NoError(t, err)

		runtimeHandler := runtime.NewHandler(db, 2, "", k8sClient, log)

		req, err := http.NewRequest("GET", "/runtimes", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		router := httputil.NewRouter()
		runtimeHandler.AttachRoutes(router)

		// when
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		var out pkg.RuntimesPage

		err = json.Unmarshal(rr.Body.Bytes(), &out)
		require.NoError(t, err)

		assert.Equal(t, 1, out.TotalCount)
		assert.Equal(t, 1, out.Count)
		assert.Equal(t, testInstance1.InstanceID, out.Data[0].InstanceID)
		assert.Equal(t, provisioningOpId, out.Data[0].Status.Provisioning.OperationID)

		unsuspensionOps := out.Data[0].Status.Unsuspension.Data
		assert.Equal(t, 1, len(unsuspensionOps))
		assert.Equal(t, unsuspensionOpId, unsuspensionOps[0].OperationID)
	})

	t.Run("should distinguish between deprovisioning & suspension operations", func(t *testing.T) {
		// given
		db := storage.NewMemoryStorage()
		operations := db.Operations()
		instances := db.Instances()
		testInstance1 := fixture.FixInstance("instance-1")

		suspensionOpId := "suspension-op-id"
		deprovisioningOpId := "deprovisioning-op-id"

		err := instances.Insert(testInstance1)
		require.NoError(t, err)

		err = operations.InsertDeprovisioningOperation(internal.DeprovisioningOperation{
			Operation: internal.Operation{
				ID:         suspensionOpId,
				Version:    0,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
				InstanceID: testInstance1.InstanceID,
				Temporary:  true,
				Type:       internal.OperationTypeDeprovision,
			},
		})
		require.NoError(t, err)

		err = operations.InsertDeprovisioningOperation(internal.DeprovisioningOperation{
			Operation: internal.Operation{
				ID:         deprovisioningOpId,
				Version:    0,
				CreatedAt:  time.Now().Add(1 * time.Hour),
				UpdatedAt:  time.Now().Add(1 * time.Hour),
				InstanceID: testInstance1.InstanceID,
				Temporary:  false,
				Type:       internal.OperationTypeDeprovision,
			},
		})
		require.NoError(t, err)

		runtimeHandler := runtime.NewHandler(db, 2, "", k8sClient, log)

		req, err := http.NewRequest("GET", "/runtimes", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		router := httputil.NewRouter()
		runtimeHandler.AttachRoutes(router)

		// when
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		var out pkg.RuntimesPage

		err = json.Unmarshal(rr.Body.Bytes(), &out)
		require.NoError(t, err)

		assert.Equal(t, 1, out.TotalCount)
		assert.Equal(t, 1, out.Count)
		assert.Equal(t, testInstance1.InstanceID, out.Data[0].InstanceID)

		suspensionOps := out.Data[0].Status.Suspension.Data
		assert.Equal(t, 1, len(suspensionOps))
		assert.Equal(t, suspensionOpId, suspensionOps[0].OperationID)

		assert.Equal(t, deprovisioningOpId, out.Data[0].Status.Deprovisioning.OperationID)
	})

	t.Run("test operation detail parameter and runtime state", func(t *testing.T) {
		// given
		db := storage.NewMemoryStorage()
		operations := db.Operations()
		instances := db.Instances()
		testTime := time.Now()
		testInstance := fixInstance(testID1, testTime)

		err := instances.Insert(testInstance)
		require.NoError(t, err)

		provOp := fixture.FixProvisioningOperation(fixRandomID(), testID1)
		err = operations.InsertOperation(provOp)
		require.NoError(t, err)
		updOp := fixture.FixUpdatingOperation(fixRandomID(), testID1)
		updOp.State = domain.Succeeded
		updOp.CreatedAt = updOp.CreatedAt.Add(time.Minute)
		err = operations.InsertUpdatingOperation(updOp)
		require.NoError(t, err)

		runtimeHandler := runtime.NewHandler(db, 2, "", k8sClient, log)

		rr := httptest.NewRecorder()
		router := httputil.NewRouter()
		runtimeHandler.AttachRoutes(router)

		// when
		req, err := http.NewRequest("GET", fmt.Sprintf("/runtimes?op_detail=%s", pkg.AllOperation), nil)
		require.NoError(t, err)
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		var out pkg.RuntimesPage

		err = json.Unmarshal(rr.Body.Bytes(), &out)
		require.NoError(t, err)

		require.Equal(t, 1, out.TotalCount)
		require.Equal(t, 1, out.Count)
		assert.Equal(t, testID1, out.Data[0].InstanceID)
		assert.NotNil(t, out.Data[0].Status.Provisioning)
		assert.Nil(t, out.Data[0].Status.Deprovisioning)
		assert.Equal(t, pkg.StateSucceeded, out.Data[0].Status.State)

		// when
		rr = httptest.NewRecorder()
		req, err = http.NewRequest("GET", fmt.Sprintf("/runtimes?op_detail=%s", pkg.LastOperation), nil)
		require.NoError(t, err)
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		out = pkg.RuntimesPage{}
		err = json.Unmarshal(rr.Body.Bytes(), &out)
		require.NoError(t, err)

		require.Equal(t, 1, out.TotalCount)
		require.Equal(t, 1, out.Count)
		assert.Equal(t, testID1, out.Data[0].InstanceID)
		assert.Nil(t, out.Data[0].Status.Provisioning)
		assert.Nil(t, out.Data[0].Status.Deprovisioning)
		assert.Equal(t, pkg.StateSucceeded, out.Data[0].Status.State)
	})

}

func TestRuntimeHandler_WithKimOnlyDrivenInstances(t *testing.T) {
	runtimeObj := fixRuntimeResource(t, "runtime-test1", "kcp-system")
	k8sClient := fake.NewClientBuilder().Build()
	require.NoError(t, k8sClient.Create(t.Context(), runtimeObj.obj, &client.CreateOptions{}))

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	t.Run("test operation detail parameter and runtime state", func(t *testing.T) {
		// given
		db := storage.NewMemoryStorage()
		operations := db.Operations()
		instances := db.Instances()

		testTime := time.Now()
		testInstance := fixInstanceForPreview(testID1, testTime)

		err := instances.Insert(testInstance)
		require.NoError(t, err)

		provOp := fixture.FixProvisioningOperation(fixRandomID(), testID1)
		err = operations.InsertOperation(provOp)
		require.NoError(t, err)
		updOp := fixture.FixUpdatingOperation(fixRandomID(), testID1)
		updOp.State = domain.Succeeded
		updOp.CreatedAt = updOp.CreatedAt.Add(time.Minute)
		err = operations.InsertUpdatingOperation(updOp)
		require.NoError(t, err)

		runtimeHandler := runtime.NewHandler(db, 2, "", k8sClient, log)

		rr := httptest.NewRecorder()
		router := httputil.NewRouter()
		runtimeHandler.AttachRoutes(router)

		// when
		req, err := http.NewRequest("GET", fmt.Sprintf("/runtimes?op_detail=%s", pkg.AllOperation), nil)
		require.NoError(t, err)
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		var out pkg.RuntimesPage

		err = json.Unmarshal(rr.Body.Bytes(), &out)
		require.NoError(t, err)

		require.Equal(t, 1, out.TotalCount)
		require.Equal(t, 1, out.Count)
		assert.Equal(t, testID1, out.Data[0].InstanceID)
		assert.NotNil(t, out.Data[0].Status.Provisioning)
		assert.Nil(t, out.Data[0].Status.Deprovisioning)
		assert.Equal(t, pkg.StateSucceeded, out.Data[0].Status.State)

		// when
		rr = httptest.NewRecorder()
		req, err = http.NewRequest("GET", fmt.Sprintf("/runtimes?op_detail=%s", pkg.LastOperation), nil)
		require.NoError(t, err)
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		out = pkg.RuntimesPage{}
		err = json.Unmarshal(rr.Body.Bytes(), &out)
		require.NoError(t, err)

		require.Equal(t, 1, out.TotalCount)
		require.Equal(t, 1, out.Count)
		assert.Equal(t, testID1, out.Data[0].InstanceID)
		assert.Nil(t, out.Data[0].Status.Provisioning)
		assert.Nil(t, out.Data[0].Status.Deprovisioning)
		assert.Equal(t, pkg.StateSucceeded, out.Data[0].Status.State)
	})

	t.Run("test betaEnabled and usedForProduction", func(t *testing.T) {
		// given
		db := storage.NewMemoryStorage()
		operations := db.Operations()
		instances := db.Instances()
		subaccountStates := db.SubaccountStates()

		testTime := time.Now()
		testInstance1 := fixInstanceForPreview(testID1, testTime)
		testInstance1.SubAccountID = "subaccount-1"

		testTime = time.Now()
		testInstance2 := fixInstanceForPreview(testID2, testTime)
		testInstance2.SubAccountID = "subaccount-1"

		testTime = time.Now()
		testInstance3 := fixInstanceForPreview(testID3, testTime)
		testInstance3.SubAccountID = "subaccount-3"

		testTime = time.Now()
		testInstance4 := fixInstanceForPreview(testID4, testTime)
		testInstance4.SubAccountID = "subaccount-4"

		err := instances.Insert(testInstance1)
		require.NoError(t, err)

		err = instances.Insert(testInstance2)
		require.NoError(t, err)

		err = instances.Insert(testInstance3)
		require.NoError(t, err)

		err = instances.Insert(testInstance4)
		require.NoError(t, err)

		err = subaccountStates.UpsertState(internal.SubaccountState{ID: testInstance1.SubAccountID, UsedForProduction: "USED_FOR_PRODUCTION", BetaEnabled: "true", ModifiedAt: 1})
		require.NoError(t, err)

		err = subaccountStates.UpsertState(internal.SubaccountState{ID: testInstance3.SubAccountID, UsedForProduction: "", BetaEnabled: "false", ModifiedAt: 1})
		require.NoError(t, err)

		provOp := fixture.FixProvisioningOperation(fixRandomID(), testID1)
		err = operations.InsertOperation(provOp)
		require.NoError(t, err)
		provOp2 := fixture.FixProvisioningOperation(fixRandomID(), testID2)
		err = operations.InsertOperation(provOp2)
		require.NoError(t, err)
		provOp3 := fixture.FixProvisioningOperation(fixRandomID(), testID3)
		err = operations.InsertOperation(provOp3)
		require.NoError(t, err)
		provOp4 := fixture.FixProvisioningOperation(fixRandomID(), testID3)
		err = operations.InsertOperation(provOp4)
		require.NoError(t, err)
		updOp := fixture.FixUpdatingOperation(fixRandomID(), testID1)
		updOp.State = domain.Succeeded
		updOp.CreatedAt = updOp.CreatedAt.Add(time.Minute)
		err = operations.InsertUpdatingOperation(updOp)
		require.NoError(t, err)

		runtimeHandler := runtime.NewHandler(db, 4, "", k8sClient, log)

		rr := httptest.NewRecorder()
		router := httputil.NewRouter()
		runtimeHandler.AttachRoutes(router)

		// when
		req, err := http.NewRequest("GET", fmt.Sprintf("/runtimes?op_detail=%s", pkg.AllOperation), nil)
		require.NoError(t, err)
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		var out pkg.RuntimesPage

		err = json.Unmarshal(rr.Body.Bytes(), &out)
		require.NoError(t, err)

		require.Equal(t, 4, out.TotalCount)
		require.Equal(t, 4, out.Count)
		assert.Equal(t, testID1, out.Data[0].InstanceID)
		assert.NotNil(t, out.Data[0].Status.Provisioning)
		assert.Nil(t, out.Data[0].Status.Deprovisioning)
		assert.Equal(t, pkg.StateSucceeded, out.Data[0].Status.State)

		// when
		rr = httptest.NewRecorder()
		req, err = http.NewRequest("GET", fmt.Sprintf("/runtimes?op_detail=%s", pkg.LastOperation), nil)
		require.NoError(t, err)
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		out = pkg.RuntimesPage{}
		err = json.Unmarshal(rr.Body.Bytes(), &out)
		require.NoError(t, err)

		require.Equal(t, 4, out.TotalCount)
		require.Equal(t, 4, out.Count)
		assert.Equal(t, testID1, out.Data[0].InstanceID)
		assert.Nil(t, out.Data[0].Status.Provisioning)
		assert.Nil(t, out.Data[0].Status.Deprovisioning)
		assert.Equal(t, "true", out.Data[0].BetaEnabled)
		assert.Equal(t, "USED_FOR_PRODUCTION", out.Data[0].UsedForProduction)
		assert.Equal(t, "true", out.Data[1].BetaEnabled)
		assert.Equal(t, "USED_FOR_PRODUCTION", out.Data[1].UsedForProduction)
		assert.Equal(t, "false", out.Data[2].BetaEnabled)
		assert.Equal(t, 0, len(out.Data[2].UsedForProduction))
		assert.Equal(t, 0, len(out.Data[3].BetaEnabled))
		assert.Equal(t, 0, len(out.Data[3].UsedForProduction))
		assert.Equal(t, pkg.StateSucceeded, out.Data[0].Status.State)
	})

	t.Run("test bindings optional attribute", func(t *testing.T) {
		// given
		db := storage.NewMemoryStorage()
		operations := db.Operations()
		instances := db.Instances()
		bindings := db.Bindings()
		testTime := time.Now()
		testInstance := fixInstanceForPreview(testID1, testTime)
		testInstance.Provider = "aws"
		testInstance.RuntimeID = fmt.Sprintf("runtime-%s", testID1)
		err := instances.Insert(testInstance)
		require.NoError(t, err)

		operation := fixture.FixProvisioningOperation(fixRandomID(), testID1)
		operation.KymaResourceNamespace = "kcp-system"

		err = operations.InsertOperation(operation)
		require.NoError(t, err)

		binding := fixture.FixBinding("abcd")
		binding.InstanceID = testInstance.InstanceID
		err = bindings.Insert(&binding)
		require.NoError(t, err)

		runtimeHandler := runtime.NewHandler(db, 2, "", k8sClient, log)

		rr := httptest.NewRecorder()
		router := httputil.NewRouter()
		runtimeHandler.AttachRoutes(router)

		// when
		req, err := http.NewRequest("GET", "/runtimes?bindings=true", nil)
		require.NoError(t, err)
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		var out pkg.RuntimesPage

		err = json.Unmarshal(rr.Body.Bytes(), &out)
		require.NoError(t, err)

		assert.Equal(t, testID1, out.Data[0].InstanceID)
		assert.NotNil(t, out.Data[0].Bindings)
	})

	t.Run("test params sent by the platform are set", func(t *testing.T) {
		// given
		db := storage.NewMemoryStorage()
		operations := db.Operations()
		instances := db.Instances()
		testTime := time.Now()
		testInstance := fixInstanceForPreview(testID1, testTime)

		err := instances.Insert(testInstance)
		require.NoError(t, err)

		provOp := fixture.FixProvisioningOperation(fixRandomID(), testID1)
		err = operations.InsertOperation(provOp)
		require.NoError(t, err)

		runtimeHandler := runtime.NewHandler(db, 2, "", k8sClient, log)

		rr := httptest.NewRecorder()
		router := httputil.NewRouter()
		runtimeHandler.AttachRoutes(router)

		// when
		req, err := http.NewRequest("GET", "/runtimes", nil)
		require.NoError(t, err)
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		var out pkg.RuntimesPage

		err = json.Unmarshal(rr.Body.Bytes(), &out)
		require.NoError(t, err)
		assert.NotNil(t, out.Data[0].Status.Provisioning.Parameters.MachineType)
		assert.NotNil(t, out.Data[0].Parameters.MachineType)
	})

	t.Run("test licenseType and commercialModel", func(t *testing.T) {
		// given
		db := storage.NewMemoryStorage()
		operations := db.Operations()
		instances := db.Instances()
		testTime := time.Now()
		testInstance := fixInstanceForPreview(testID1, testTime)
		licenseType := "SAPDEV"
		testInstance.Parameters.ErsContext.LicenseType = &licenseType
		commercialModel := "SUBSCRIPTION"
		testInstance.Parameters.ErsContext.CommercialModel = &commercialModel

		err := instances.Insert(testInstance)
		require.NoError(t, err)

		provOp := fixture.FixProvisioningOperation(fixRandomID(), testID1)
		err = operations.InsertOperation(provOp)
		require.NoError(t, err)

		runtimeHandler := runtime.NewHandler(db, 2, "", k8sClient, log)

		rr := httptest.NewRecorder()
		router := httputil.NewRouter()
		runtimeHandler.AttachRoutes(router)

		// when
		req, err := http.NewRequest("GET", "/runtimes", nil)
		require.NoError(t, err)
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		var out pkg.RuntimesPage

		err = json.Unmarshal(rr.Body.Bytes(), &out)
		require.NoError(t, err)
		require.NotNil(t, out.Data[0].LicenseType)
		assert.Equal(t, licenseType, *out.Data[0].LicenseType)
		require.NotNil(t, out.Data[0].CommercialModel)
		assert.Equal(t, commercialModel, *out.Data[0].CommercialModel)
	})

	t.Run("test empty actions", func(t *testing.T) {
		// given
		db := storage.NewMemoryStorage()
		operations := db.Operations()
		instances := db.Instances()
		testTime := time.Now()
		testInstance := fixInstanceForPreview(testID1, testTime)

		err := instances.Insert(testInstance)
		require.NoError(t, err)

		provOp := fixture.FixProvisioningOperation(fixRandomID(), testID1)
		err = operations.InsertOperation(provOp)
		require.NoError(t, err)

		runtimeHandler := runtime.NewHandler(db, 2, "", k8sClient, log)

		rr := httptest.NewRecorder()
		router := httputil.NewRouter()
		runtimeHandler.AttachRoutes(router)

		// when
		req, err := http.NewRequest("GET", "/runtimes?actions=true", nil)
		require.NoError(t, err)
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		var out pkg.RuntimesPage

		err = json.Unmarshal(rr.Body.Bytes(), &out)
		require.NoError(t, err)
		assert.Nil(t, out.Data[0].Actions)
	})

	t.Run("test actions", func(t *testing.T) {
		// given
		db := storage.NewMemoryStorage()
		operations := db.Operations()
		instances := db.Instances()
		actions := db.Actions()
		testTime := time.Now()
		testInstance := fixInstanceForPreview(testID1, testTime)

		err := instances.Insert(testInstance)
		require.NoError(t, err)

		provOp := fixture.FixProvisioningOperation(fixRandomID(), testID1)
		err = operations.InsertOperation(provOp)
		require.NoError(t, err)

		err = actions.InsertAction(pkg.PlanUpdateActionType, testID1, "test-message-1", "old-value-1", "new-value-1")
		assert.NoError(t, err)
		err = actions.InsertAction(pkg.SubaccountMovementActionType, testID1, "test-message-2", "old-value-2", "new-value-2")
		assert.NoError(t, err)

		runtimeHandler := runtime.NewHandler(db, 2, "", k8sClient, log)

		rr := httptest.NewRecorder()
		router := httputil.NewRouter()
		runtimeHandler.AttachRoutes(router)

		// when
		req, err := http.NewRequest("GET", "/runtimes?actions=true", nil)
		require.NoError(t, err)
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		var out pkg.RuntimesPage

		err = json.Unmarshal(rr.Body.Bytes(), &out)
		require.NoError(t, err)
		assert.Len(t, out.Data[0].Actions, 2)
		assert.Equal(t, out.Data[0].Actions[0].Type, pkg.SubaccountMovementActionType)
		assert.Equal(t, out.Data[0].Actions[1].Type, pkg.PlanUpdateActionType)
	})
}

func fixInstance(id string, t time.Time) internal.Instance {
	return internal.Instance{
		InstanceID:      id,
		CreatedAt:       t,
		GlobalAccountID: id,
		SubAccountID:    id,
		RuntimeID:       id,
		ServiceID:       id,
		ServiceName:     id,
		ServicePlanID:   id,
		ServicePlanName: id,
		DashboardURL:    fmt.Sprintf("https://console.%s.kyma.local", id),
		ProviderRegion:  id,
		Parameters:      internal.ProvisioningParameters{},
	}
}

func fixInstanceForPreview(id string, t time.Time) internal.Instance {
	instance := fixInstance(id, t)
	instance.ServicePlanName = broker.PreviewPlanName
	instance.ServicePlanID = broker.PreviewPlanID
	instance.Parameters.Parameters = pkg.ProvisioningParametersDTO{
		Region:      ptr.String("fake-reśgion"),
		MachineType: ptr.String("fake-machine-type"),
	}
	return instance
}

func fixRandomID() string {
	return rand.String(16)
}

type RuntimeResourceType struct {
	obj *unstructured.Unstructured
}

func fixRuntimeResource(t *testing.T, name, namespace string) *RuntimeResourceType {
	runtimeResource := &unstructured.Unstructured{}
	runtimeResource.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "infrastructuremanager.kyma-project.io",
		Version: "v1",
		Kind:    "Runtime",
	})
	runtimeResource.SetName(name)
	runtimeResource.SetNamespace(namespace)

	worker := map[string]interface{}{}
	err := unstructured.SetNestedField(worker, "worker-0", "name")
	assert.NoError(t, err)
	err = unstructured.SetNestedField(worker, "m6i.large", "machine", "type")
	assert.NoError(t, err)

	managedField := map[string]interface{}{}
	err = unstructured.SetNestedSlice(runtimeResource.Object, []interface{}{managedField}, "metadata", "managedFields")
	assert.NoError(t, err)

	err = unstructured.SetNestedSlice(runtimeResource.Object, []interface{}{worker}, "spec", "shoot", "provider", "workers")
	assert.NoError(t, err)

	err = unstructured.SetNestedField(runtimeResource.Object, "kim-driven-shoot", "spec", "shoot", "name")
	assert.NoError(t, err)
	err = unstructured.SetNestedField(runtimeResource.Object, "test-client-id", "spec", "shoot", "kubernetes", "kubeAPIServer", "oidcConfig", "clientID")
	assert.NoError(t, err)
	err = unstructured.SetNestedField(runtimeResource.Object, "aws", "spec", "shoot", "provider", "type")
	assert.NoError(t, err)
	err = unstructured.SetNestedField(runtimeResource.Object, false, "spec", "security", "networking", "filter", "egress", "enabled")
	assert.NoError(t, err)
	err = unstructured.SetNestedField(runtimeResource.Object, "Ready", "status", "state")
	assert.NoError(t, err)

	return &RuntimeResourceType{obj: runtimeResource}
}
