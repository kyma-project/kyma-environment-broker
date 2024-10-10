package runtime_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kyma-project/kyma-environment-broker/internal/storage"

	"github.com/gorilla/mux"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/kyma-environment-broker/internal/runtime"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/rand"
)

func TestRuntimeHandler(t *testing.T) {
	k8sClient := fake.NewClientBuilder().Build()
	kimConfig := broker.KimConfig{
		Enabled: false,
	}

	t.Run("test pagination should work", func(t *testing.T) {
		// given
		provisionerClient := provisioner.NewFakeClient()

		db := storage.NewMemoryStorage()
		operations := db.Operations()
		instances := db.Instances()
		states := db.RuntimeStates()
		archived := db.InstancesArchived()
		bindings := db.Bindings()
		testID1 := "Test1"
		testID2 := "Test2"
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

		runtimeHandler := runtime.NewHandler(instances, operations, states, archived, bindings, 2, "", provisionerClient, k8sClient, kimConfig, logrus.New())

		req, err := http.NewRequest("GET", "/runtimes?page_size=1", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
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
		logrus.Print(out.Data)
		assert.Equal(t, 2, out.TotalCount)
		assert.Equal(t, 1, out.Count)
		assert.Equal(t, testID2, out.Data[0].InstanceID)

	})

	t.Run("test validation should work", func(t *testing.T) {
		// given
		provisionerClient := provisioner.NewFakeClient()

		db := storage.NewMemoryStorage()
		operations := db.Operations()
		instances := db.Instances()
		states := db.RuntimeStates()
		archived := db.InstancesArchived()

		runtimeHandler := runtime.NewHandler(instances, operations, states, archived, nil, 2, "region", provisionerClient, k8sClient, kimConfig, logrus.New())

		req, err := http.NewRequest("GET", "/runtimes?page_size=a", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
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
		provisionerClient := provisioner.NewFakeClient()

		db := storage.NewMemoryStorage()
		operations := db.Operations()
		instances := db.Instances()
		states := db.RuntimeStates()
		archived := db.InstancesArchived()
		testID1 := "Test1"
		testID2 := "Test2"
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

		runtimeHandler := runtime.NewHandler(instances, operations, states, archived, nil, 2, "", provisionerClient, k8sClient, kimConfig, logrus.New())

		req, err := http.NewRequest("GET", fmt.Sprintf("/runtimes?account=%s&subaccount=%s&instance_id=%s&runtime_id=%s&region=%s&shoot=%s", testID1, testID1, testID1, testID1, testID1, fmt.Sprintf("Shoot-%s", testID1)), nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
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
		provisionerClient := provisioner.NewFakeClient()
		db := storage.NewMemoryStorage()
		operations := db.Operations()
		instances := db.Instances()
		states := db.RuntimeStates()
		archived := db.InstancesArchived()
		testID1 := "Test1"
		testID2 := "Test2"
		testID3 := "Test3"
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

		runtimeHandler := runtime.NewHandler(instances, operations, states, archived, nil, 2, "", provisionerClient, k8sClient, kimConfig, logrus.New())

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
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
		provisionerClient := provisioner.NewFakeClient()
		db := storage.NewMemoryStorage()
		operations := db.Operations()
		instances := db.Instances()
		states := db.RuntimeStates()
		archived := db.InstancesArchived()
		testID1 := "Test1"
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

		runtimeHandler := runtime.NewHandler(instances, operations, states, archived, nil, 2, "", provisionerClient, k8sClient, kimConfig, logrus.New())

		req, err := http.NewRequest("GET", "/runtimes", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
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
		provisionerClient := provisioner.NewFakeClient()
		db := storage.NewMemoryStorage()
		operations := db.Operations()
		instances := db.Instances()
		states := db.RuntimeStates()
		archived := db.InstancesArchived()
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

		runtimeHandler := runtime.NewHandler(instances, operations, states, archived, nil, 2, "", provisionerClient, k8sClient, kimConfig, logrus.New())

		req, err := http.NewRequest("GET", "/runtimes", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
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
		provisionerClient := provisioner.NewFakeClient()
		db := storage.NewMemoryStorage()
		operations := db.Operations()
		instances := db.Instances()
		states := db.RuntimeStates()
		archived := db.InstancesArchived()
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

		runtimeHandler := runtime.NewHandler(instances, operations, states, archived, nil, 2, "", provisionerClient, k8sClient, kimConfig, logrus.New())

		req, err := http.NewRequest("GET", "/runtimes", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
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
		provisionerClient := provisioner.NewFakeClient()
		db := storage.NewMemoryStorage()
		operations := db.Operations()
		instances := db.Instances()
		states := db.RuntimeStates()
		archived := db.InstancesArchived()
		testID := "Test1"
		testTime := time.Now()
		testInstance := fixInstance(testID, testTime)

		err := instances.Insert(testInstance)
		require.NoError(t, err)

		provOp := fixture.FixProvisioningOperation(fixRandomID(), testID)
		err = operations.InsertOperation(provOp)
		require.NoError(t, err)
		updOp := fixture.FixUpdatingOperation(fixRandomID(), testID)
		updOp.State = domain.Succeeded
		updOp.CreatedAt = updOp.CreatedAt.Add(time.Minute)
		err = operations.InsertUpdatingOperation(updOp)
		require.NoError(t, err)

		runtimeHandler := runtime.NewHandler(instances, operations, states, archived, nil, 2, "", provisionerClient, k8sClient, kimConfig, logrus.New())

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
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
		assert.Equal(t, testID, out.Data[0].InstanceID)
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
		assert.Equal(t, testID, out.Data[0].InstanceID)
		assert.Nil(t, out.Data[0].Status.Provisioning)
		assert.Nil(t, out.Data[0].Status.Deprovisioning)
		assert.Equal(t, pkg.StateSucceeded, out.Data[0].Status.State)
	})

	t.Run("test kyma_config and cluster_config optional attributes", func(t *testing.T) {
		// given
		provisionerClient := provisioner.NewFakeClient()
		db := storage.NewMemoryStorage()
		operations := db.Operations()
		instances := db.Instances()
		states := db.RuntimeStates()
		archived := db.InstancesArchived()
		testID := "Test1"
		testTime := time.Now()
		testInstance := fixInstance(testID, testTime)

		err := instances.Insert(testInstance)
		require.NoError(t, err)

		provOp := fixture.FixProvisioningOperation(fixRandomID(), testID)
		err = operations.InsertOperation(provOp)
		require.NoError(t, err)
		updOp := fixture.FixUpdatingOperation(fixRandomID(), testID)
		updOp.State = domain.Failed
		updOp.CreatedAt = updOp.CreatedAt.Add(time.Minute)
		err = operations.InsertUpdatingOperation(updOp)
		require.NoError(t, err)
		upgClOp := fixture.FixUpgradeClusterOperation(fixRandomID(), testID)
		upgClOp.CreatedAt = updOp.CreatedAt.Add(2 * time.Minute)
		err = operations.InsertUpgradeClusterOperation(upgClOp)
		require.NoError(t, err)

		fixProvState := internal.RuntimeState{
			ID:          fixRandomID(),
			CreatedAt:   provOp.CreatedAt,
			RuntimeID:   testInstance.RuntimeID,
			OperationID: provOp.ID,
			KymaConfig: gqlschema.KymaConfigInput{
				Version: "1.22.0",
			},
			ClusterConfig: gqlschema.GardenerConfigInput{
				Name:              testID,
				KubernetesVersion: "1.18.18",
				Provider:          string(internal.AWS),
			},
		}
		err = states.Insert(fixProvState)
		require.NoError(t, err)
		fixUpgKymaState := internal.RuntimeState{
			ID:          fixRandomID(),
			CreatedAt:   updOp.CreatedAt,
			RuntimeID:   testInstance.RuntimeID,
			OperationID: updOp.Operation.ID,
			KymaConfig: gqlschema.KymaConfigInput{
				Version: "1.23.0",
				Profile: (*gqlschema.KymaProfile)(ptr.String("production")),
				Components: []*gqlschema.ComponentConfigurationInput{
					{
						Component: "istio",
						Namespace: "istio-system",
						Configuration: []*gqlschema.ConfigEntryInput{
							{
								Key:   "test_key",
								Value: "test_value",
							},
						},
					},
				},
			},
		}
		err = states.Insert(fixUpgKymaState)
		require.NoError(t, err)
		fixOpgClusterState := internal.RuntimeState{
			ID:          fixRandomID(),
			CreatedAt:   upgClOp.CreatedAt,
			RuntimeID:   testInstance.RuntimeID,
			OperationID: upgClOp.Operation.ID,
			ClusterConfig: gqlschema.GardenerConfigInput{
				Name:                testID,
				KubernetesVersion:   "1.19.19",
				Provider:            string(internal.AWS),
				MachineImage:        ptr.String("gardenlinux"),
				MachineImageVersion: ptr.String("1.0.0"),
			},
		}
		err = states.Insert(fixOpgClusterState)
		require.NoError(t, err)

		runtimeHandler := runtime.NewHandler(instances, operations, states, archived, nil, 2, "", provisionerClient, k8sClient, kimConfig, logrus.New())

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		runtimeHandler.AttachRoutes(router)

		// when
		req, err := http.NewRequest("GET", "/runtimes?kyma_config=true&cluster_config=true", nil)
		require.NoError(t, err)
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		var out pkg.RuntimesPage

		err = json.Unmarshal(rr.Body.Bytes(), &out)
		require.NoError(t, err)

		require.Equal(t, 1, out.TotalCount)
		require.Equal(t, 1, out.Count)
		assert.Equal(t, testID, out.Data[0].InstanceID)
		require.NotNil(t, out.Data[0].KymaConfig)
		assert.Equal(t, "1.23.0", out.Data[0].KymaConfig.Version)
		require.NotNil(t, out.Data[0].ClusterConfig)
		assert.Equal(t, "1.19.19", out.Data[0].ClusterConfig.KubernetesVersion)
	})

	t.Run("test gardener_config optional attribute", func(t *testing.T) {
		// given
		provisionerClient := provisioner.NewFakeClient()
		db := storage.NewMemoryStorage()
		operations := db.Operations()
		instances := db.Instances()
		states := db.RuntimeStates()
		archived := db.InstancesArchived()
		testID := "Test1"
		testTime := time.Now()
		testInstance := fixInstance(testID, testTime)
		testInstance.Provider = "aws"
		testInstance.RuntimeID = fmt.Sprintf("runtime-%s", testID)
		err := instances.Insert(testInstance)
		require.NoError(t, err)

		operation := fixture.FixProvisioningOperation(fixRandomID(), testID)
		err = operations.InsertOperation(operation)
		require.NoError(t, err)

		input, err := operation.InputCreator.CreateProvisionRuntimeInput()
		require.NoError(t, err)

		_, err = provisionerClient.ProvisionRuntimeWithIDs(operation.GlobalAccountID, operation.SubAccountID, operation.RuntimeID, operation.ID, input)
		require.NoError(t, err)

		runtimeHandler := runtime.NewHandler(instances, operations, states, archived, nil, 2, "", provisionerClient, k8sClient, kimConfig, logrus.New())

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		runtimeHandler.AttachRoutes(router)

		// when
		req, err := http.NewRequest("GET", "/runtimes?gardener_config=true", nil)
		require.NoError(t, err)
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		var out pkg.RuntimesPage

		err = json.Unmarshal(rr.Body.Bytes(), &out)
		require.NoError(t, err)

		require.Equal(t, 1, out.TotalCount)
		require.Equal(t, 1, out.Count)
		assert.Equal(t, testID, out.Data[0].InstanceID)
		require.NotNil(t, out.Data[0].Status.GardenerConfig)
		assert.Equal(t, "fake-region", *out.Data[0].Status.GardenerConfig.Region)
	})

	t.Run("test runtime_config optional attribute", func(t *testing.T) {
		// given
		provisionerClient := provisioner.NewFakeClient()
		db := storage.NewMemoryStorage()
		operations := db.Operations()
		instances := db.Instances()
		states := db.RuntimeStates()
		archived := db.InstancesArchived()
		testID := "Test1"
		testTime := time.Now()
		testInstance := fixInstance(testID, testTime)
		testInstance.Provider = "aws"
		testInstance.RuntimeID = fmt.Sprintf("runtime-%s", testID)
		err := instances.Insert(testInstance)
		require.NoError(t, err)

		operation := fixture.FixProvisioningOperation(fixRandomID(), testID)
		err = operations.InsertOperation(operation)
		require.NoError(t, err)

		input, err := operation.InputCreator.CreateProvisionRuntimeInput()
		require.NoError(t, err)

		_, err = provisionerClient.ProvisionRuntimeWithIDs(operation.GlobalAccountID, operation.SubAccountID, operation.RuntimeID, operation.ID, input)
		require.NoError(t, err)

		runtimeHandler := runtime.NewHandler(instances, operations, states, archived, nil, 2, "", provisionerClient, k8sClient, kimConfig, logrus.New())

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		runtimeHandler.AttachRoutes(router)

		// when
		req, err := http.NewRequest("GET", "/runtimes?runtime_config=true", nil)
		require.NoError(t, err)
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		var out pkg.RuntimesPage

		err = json.Unmarshal(rr.Body.Bytes(), &out)
		require.NoError(t, err)

		require.Equal(t, 1, out.TotalCount)
		require.Equal(t, 1, out.Count)
		assert.Equal(t, testID, out.Data[0].InstanceID)
		require.Nil(t, out.Data[0].RuntimeConfig)
	})

}

func TestRuntimeHandler_WithKimOnlyDrivenInstances(t *testing.T) {
	runtimeObj := fixRuntimeResource(t, "runtime-test1", "kcp-system")
	k8sClient := fake.NewClientBuilder().WithRuntimeObjects(runtimeObj.obj).Build()
	kimConfig := broker.KimConfig{
		Enabled:      true,
		Plans:        []string{"preview"},
		KimOnlyPlans: []string{"preview"},
	}
	runtimesNotKnownToProvisioner := map[string]interface{}{"runtime-test1": nil}
	provisionerClient := provisioner.NewFakeClientWithKimOnlyDrivenRuntimes(runtimesNotKnownToProvisioner)

	t.Run("test operation detail parameter and runtime state", func(t *testing.T) {
		// given
		db := storage.NewMemoryStorage()
		operations := db.Operations()
		instances := db.Instances()
		states := db.RuntimeStates()
		archived := db.InstancesArchived()
		testID := "Test1"
		testTime := time.Now()
		testInstance := fixInstanceForPreview(testID, testTime)

		err := instances.Insert(testInstance)
		require.NoError(t, err)

		provOp := fixture.FixProvisioningOperation(fixRandomID(), testID)
		err = operations.InsertOperation(provOp)
		require.NoError(t, err)
		updOp := fixture.FixUpdatingOperation(fixRandomID(), testID)
		updOp.State = domain.Succeeded
		updOp.CreatedAt = updOp.CreatedAt.Add(time.Minute)
		err = operations.InsertUpdatingOperation(updOp)
		require.NoError(t, err)

		runtimeHandler := runtime.NewHandler(instances, operations, states, archived, nil, 2, "", provisionerClient, k8sClient, kimConfig, logrus.New())

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
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
		assert.Equal(t, testID, out.Data[0].InstanceID)
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
		assert.Equal(t, testID, out.Data[0].InstanceID)
		assert.Nil(t, out.Data[0].Status.Provisioning)
		assert.Nil(t, out.Data[0].Status.Deprovisioning)
		assert.Equal(t, pkg.StateSucceeded, out.Data[0].Status.State)
	})

	t.Run("test kyma_config and cluster_config optional attributes", func(t *testing.T) {
		// given
		db := storage.NewMemoryStorage()
		operations := db.Operations()
		instances := db.Instances()
		states := db.RuntimeStates()
		archived := db.InstancesArchived()
		testID := "Test1"
		testTime := time.Now()
		testInstance := fixInstanceForPreview(testID, testTime)

		err := instances.Insert(testInstance)
		require.NoError(t, err)

		provOp := fixture.FixProvisioningOperation(fixRandomID(), testID)
		err = operations.InsertOperation(provOp)
		require.NoError(t, err)
		updOp := fixture.FixUpdatingOperation(fixRandomID(), testID)
		updOp.State = domain.Failed
		updOp.CreatedAt = updOp.CreatedAt.Add(time.Minute)
		err = operations.InsertUpdatingOperation(updOp)
		require.NoError(t, err)
		upgClOp := fixture.FixUpgradeClusterOperation(fixRandomID(), testID)
		upgClOp.CreatedAt = updOp.CreatedAt.Add(2 * time.Minute)
		err = operations.InsertUpgradeClusterOperation(upgClOp)
		require.NoError(t, err)

		fixProvState := internal.RuntimeState{
			ID:          fixRandomID(),
			CreatedAt:   provOp.CreatedAt,
			RuntimeID:   testInstance.RuntimeID,
			OperationID: provOp.ID,
			KymaConfig: gqlschema.KymaConfigInput{
				Version: "1.22.0",
			},
			ClusterConfig: gqlschema.GardenerConfigInput{
				Name:              testID,
				KubernetesVersion: "1.18.18",
				Provider:          string(internal.AWS),
			},
		}
		err = states.Insert(fixProvState)
		require.NoError(t, err)
		fixUpgKymaState := internal.RuntimeState{
			ID:          fixRandomID(),
			CreatedAt:   updOp.CreatedAt,
			RuntimeID:   testInstance.RuntimeID,
			OperationID: updOp.Operation.ID,
			KymaConfig: gqlschema.KymaConfigInput{
				Version: "1.23.0",
				Profile: (*gqlschema.KymaProfile)(ptr.String("production")),
				Components: []*gqlschema.ComponentConfigurationInput{
					{
						Component: "istio",
						Namespace: "istio-system",
						Configuration: []*gqlschema.ConfigEntryInput{
							{
								Key:   "test_key",
								Value: "test_value",
							},
						},
					},
				},
			},
		}
		err = states.Insert(fixUpgKymaState)
		require.NoError(t, err)
		fixOpgClusterState := internal.RuntimeState{
			ID:          fixRandomID(),
			CreatedAt:   upgClOp.CreatedAt,
			RuntimeID:   testInstance.RuntimeID,
			OperationID: upgClOp.Operation.ID,
			ClusterConfig: gqlschema.GardenerConfigInput{
				Name:                testID,
				KubernetesVersion:   "1.19.19",
				Provider:            string(internal.AWS),
				MachineImage:        ptr.String("gardenlinux"),
				MachineImageVersion: ptr.String("1.0.0"),
			},
		}
		err = states.Insert(fixOpgClusterState)
		require.NoError(t, err)

		runtimeHandler := runtime.NewHandler(instances, operations, states, archived, nil, 2, "", provisionerClient, k8sClient, kimConfig, logrus.New())

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		runtimeHandler.AttachRoutes(router)

		// when
		req, err := http.NewRequest("GET", "/runtimes?kyma_config=true&cluster_config=true", nil)
		require.NoError(t, err)
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		var out pkg.RuntimesPage

		err = json.Unmarshal(rr.Body.Bytes(), &out)
		require.NoError(t, err)

		require.Equal(t, 1, out.TotalCount)
		require.Equal(t, 1, out.Count)
		assert.Equal(t, testID, out.Data[0].InstanceID)
		require.NotNil(t, out.Data[0].KymaConfig)
		assert.Equal(t, "1.23.0", out.Data[0].KymaConfig.Version)
		require.NotNil(t, out.Data[0].ClusterConfig)
		assert.Equal(t, "1.19.19", out.Data[0].ClusterConfig.KubernetesVersion)
	})

	t.Run("test gardener_config optional attribute", func(t *testing.T) {
		// given
		db := storage.NewMemoryStorage()
		operations := db.Operations()
		instances := db.Instances()
		states := db.RuntimeStates()
		archived := db.InstancesArchived()
		testID := "Test1"
		testTime := time.Now()
		testInstance := fixInstanceForPreview(testID, testTime)
		testInstance.Provider = "aws"
		testInstance.RuntimeID = fmt.Sprintf("runtime-%s", testID)
		err := instances.Insert(testInstance)
		require.NoError(t, err)

		operation := fixture.FixProvisioningOperation(fixRandomID(), testID)
		err = operations.InsertOperation(operation)
		operation.KymaResourceNamespace = "kcp-system"
		require.NoError(t, err)

		input, err := operation.InputCreator.CreateProvisionRuntimeInput()
		require.NoError(t, err)

		_, err = provisionerClient.ProvisionRuntimeWithIDs(operation.GlobalAccountID, operation.SubAccountID, operation.RuntimeID, operation.ID, input)
		require.NoError(t, err)

		runtimeHandler := runtime.NewHandler(instances, operations, states, archived, nil, 2, "", provisionerClient, k8sClient, kimConfig, logrus.New())

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		runtimeHandler.AttachRoutes(router)

		// when
		req, err := http.NewRequest("GET", "/runtimes?gardener_config=true", nil)
		require.NoError(t, err)
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		var out pkg.RuntimesPage

		err = json.Unmarshal(rr.Body.Bytes(), &out)
		require.NoError(t, err)

		require.Equal(t, 1, out.TotalCount)
		require.Equal(t, 1, out.Count)
		assert.Equal(t, testID, out.Data[0].InstanceID)
		require.Nil(t, out.Data[0].Status.GardenerConfig)
		require.Nil(t, out.Data[0].RuntimeConfig)
	})

	t.Run("test gardener_config optional attribute with provisioner not knowing the runtime", func(t *testing.T) {
		// given
		db := storage.NewMemoryStorage()
		operations := db.Operations()
		instances := db.Instances()
		states := db.RuntimeStates()
		archived := db.InstancesArchived()
		testID := "test1"
		testTime := time.Now()
		testInstance := fixInstanceForPreview(testID, testTime)
		testInstance.Provider = "aws"
		testInstance.RuntimeID = fmt.Sprintf("runtime-%s", testID)
		err := instances.Insert(testInstance)
		require.NoError(t, err)

		operation := fixture.FixProvisioningOperation(fixRandomID(), testID)
		err = operations.InsertOperation(operation)
		operation.KymaResourceNamespace = "kcp-system"
		require.NoError(t, err)

		input, err := operation.InputCreator.CreateProvisionRuntimeInput()
		require.NoError(t, err)

		_, err = provisionerClient.ProvisionRuntimeWithIDs(operation.GlobalAccountID, operation.SubAccountID, operation.RuntimeID, operation.ID, input)
		require.NoError(t, err)

		kimDisabledForPreview := broker.KimConfig{
			Enabled:      true,
			Plans:        []string{"no-plan"},
			KimOnlyPlans: []string{"no-plan"},
		}

		runtimeHandler := runtime.NewHandler(instances, operations, states, archived, nil, 2, "", provisionerClient, k8sClient, kimDisabledForPreview, logrus.New())

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		runtimeHandler.AttachRoutes(router)

		// when
		req, err := http.NewRequest("GET", "/runtimes?gardener_config=true", nil)
		require.NoError(t, err)
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		var out pkg.RuntimesPage

		err = json.Unmarshal(rr.Body.Bytes(), &out)
		require.NoError(t, err)

		require.Equal(t, 1, out.TotalCount)
		require.Equal(t, 1, out.Count)
		assert.Equal(t, testID, out.Data[0].InstanceID)
		require.Nil(t, out.Data[0].Status.GardenerConfig)
		require.Nil(t, out.Data[0].RuntimeConfig)

	})

	t.Run("test runtime_config optional attribute", func(t *testing.T) {
		// given
		db := storage.NewMemoryStorage()
		operations := db.Operations()
		instances := db.Instances()
		states := db.RuntimeStates()
		archived := db.InstancesArchived()
		testID := "Test1"
		testTime := time.Now()
		testInstance := fixInstanceForPreview(testID, testTime)
		testInstance.Provider = "aws"
		testInstance.RuntimeID = fmt.Sprintf("runtime-%s", testID)
		err := instances.Insert(testInstance)
		require.NoError(t, err)

		operation := fixture.FixProvisioningOperation(fixRandomID(), testID)
		operation.KymaResourceNamespace = "kcp-system"

		err = operations.InsertOperation(operation)
		require.NoError(t, err)

		input, err := operation.InputCreator.CreateProvisionRuntimeInput()
		require.NoError(t, err)

		_, err = provisionerClient.ProvisionRuntimeWithIDs(operation.GlobalAccountID, operation.SubAccountID, operation.RuntimeID, operation.ID, input)
		require.NoError(t, err)

		runtimeHandler := runtime.NewHandler(instances, operations, states, archived, nil, 2, "", provisionerClient, k8sClient, kimConfig, logrus.New())

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		runtimeHandler.AttachRoutes(router)

		// when
		req, err := http.NewRequest("GET", "/runtimes?runtime_config=true", nil)
		require.NoError(t, err)
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		var out pkg.RuntimesPage

		err = json.Unmarshal(rr.Body.Bytes(), &out)
		require.NoError(t, err)

		require.Equal(t, 1, out.TotalCount)
		require.Equal(t, 1, out.Count)
		assert.Equal(t, testID, out.Data[0].InstanceID)
		require.NotNil(t, out.Data[0].RuntimeConfig)
		require.Nil(t, out.Data[0].Status.GardenerConfig)

		shootName, ok, err := unstructured.NestedString(*out.Data[0].RuntimeConfig, "spec", "shoot", "name")
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "kim-driven-shoot", shootName)

		workers, ok, err := unstructured.NestedSlice(*out.Data[0].RuntimeConfig, "spec", "shoot", "provider", "workers")
		assert.True(t, ok)
		assert.NoError(t, err)
		worker, ok, err := unstructured.NestedString(workers[0].(map[string]interface{}), "name")
		assert.True(t, ok)
		assert.NoError(t, err)
		assert.Equal(t, "worker-0", worker)

		_, ok, err = unstructured.NestedSlice(*out.Data[0].RuntimeConfig, "metadata", "managedFields")
		assert.False(t, ok)
	})

	t.Run("test bindings optional attribute", func(t *testing.T) {
		// given
		db := storage.NewMemoryStorage()
		operations := db.Operations()
		instances := db.Instances()
		states := db.RuntimeStates()
		archived := db.InstancesArchived()
		bindings := db.Bindings()
		testID := "Test1"
		testTime := time.Now()
		testInstance := fixInstanceForPreview(testID, testTime)
		testInstance.Provider = "aws"
		testInstance.RuntimeID = fmt.Sprintf("runtime-%s", testID)
		err := instances.Insert(testInstance)
		require.NoError(t, err)

		operation := fixture.FixProvisioningOperation(fixRandomID(), testID)
		operation.KymaResourceNamespace = "kcp-system"

		err = operations.InsertOperation(operation)
		require.NoError(t, err)

		binding := fixture.FixBinding("abcd")
		binding.InstanceID = testInstance.InstanceID
		err = bindings.Insert(&binding)
		require.NoError(t, err)

		input, err := operation.InputCreator.CreateProvisionRuntimeInput()
		require.NoError(t, err)

		_, err = provisionerClient.ProvisionRuntimeWithIDs(operation.GlobalAccountID, operation.SubAccountID, operation.RuntimeID, operation.ID, input)
		require.NoError(t, err)

		runtimeHandler := runtime.NewHandler(instances, operations, states, archived, bindings, 2, "", provisionerClient, k8sClient, kimConfig, logrus.New())

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
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

		assert.Equal(t, testID, out.Data[0].InstanceID)
		assert.NotNil(t, out.Data[0].Bindings)

		b := out.Data[0].Bindings[0]
		assert.Equal(t, binding.BindingType, b.Type)

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
