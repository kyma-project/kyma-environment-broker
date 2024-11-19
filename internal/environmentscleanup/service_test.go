package environmentscleanup

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	mocks "github.com/kyma-project/kyma-environment-broker/internal/environmentscleanup/automock"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	fixInstanceID1 = "instance-1"
	fixInstanceID2 = "instance-2"
	fixInstanceID3 = "instance-3"
	fixRuntimeID1  = "runtime-1"
	fixRuntimeID2  = "runtime-2"
	fixRuntimeID3  = "rntime-3"
	fixOperationID = "operation-id"

	fixAccountID       = "account-id"
	maxShootAge        = 24 * time.Hour
	shootLabelSelector = "owner.do-not-delete!=true"
)

func TestService_PerformCleanup(t *testing.T) {
	sch := k8sruntime.NewScheme()
	err := imv1.AddToScheme(sch)
	assert.NoError(t, err)
	k8sClient := fake.NewClientBuilder().WithScheme(sch).Build()

	t.Run("happy path", func(t *testing.T) {
		// given
		gcMock := &mocks.GardenerClient{}
		gcMock.On("List", mock.Anything, mock.AnythingOfType("v1.ListOptions")).Return(fixShootList(), nil)
		gcMock.On("Delete", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("v1.DeleteOptions")).Return(nil)
		gcMock.On("Update", mock.Anything, mock.Anything, mock.AnythingOfType("v1.UpdateOptions")).Return(nil, nil)
		bcMock := &mocks.BrokerClient{}
		bcMock.On("Deprovision", mock.AnythingOfType("internal.Instance")).Return(fixOperationID, nil)

		memoryStorage := storage.NewMemoryStorage()
		err := memoryStorage.Instances().Insert(internal.Instance{
			InstanceID: fixInstanceID1,
			RuntimeID:  fixRuntimeID1,
		})
		assert.NoError(t, err)
		err = memoryStorage.Instances().Insert(internal.Instance{
			InstanceID: fixInstanceID2,
			RuntimeID:  fixRuntimeID2,
		})
		assert.NoError(t, err)
		logger := logrus.New()

		svc := NewService(gcMock, bcMock, k8sClient, memoryStorage.Instances(), logger, maxShootAge, shootLabelSelector)

		// when
		err = svc.PerformCleanup()

		// then
		bcMock.AssertExpectations(t)
		gcMock.AssertExpectations(t)
		assert.NoError(t, err)
	})

	t.Run("should fail when unable to fetch shoots from gardener", func(t *testing.T) {
		// given
		gcMock := &mocks.GardenerClient{}
		gcMock.On("List", mock.Anything, mock.AnythingOfType("v1.ListOptions")).Return(&unstructured.
			UnstructuredList{}, fmt.Errorf("failed to reach gardener"))

		bcMock := &mocks.BrokerClient{}

		memoryStorage := storage.NewMemoryStorage()
		logger := logrus.New()

		svc := NewService(gcMock, bcMock, k8sClient, memoryStorage.Instances(), logger, maxShootAge, shootLabelSelector)

		// when
		err := svc.PerformCleanup()

		// then
		bcMock.AssertExpectations(t)
		gcMock.AssertExpectations(t)
		assert.Error(t, err)
	})

	t.Run("should return error when unable to find instance in db", func(t *testing.T) {
		// given
		gcMock := &mocks.GardenerClient{}
		gcMock.On("List", mock.Anything, mock.AnythingOfType("v1.ListOptions")).Return(fixShootList(), nil)
		gcMock.On("Delete", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("v1.DeleteOptions")).Return(nil)
		gcMock.On("Update", mock.Anything, mock.Anything, mock.AnythingOfType("v1.UpdateOptions")).Return(nil, nil)

		bcMock := &mocks.BrokerClient{}
		bcMock.On("Deprovision", mock.AnythingOfType("internal.Instance")).Return(fixOperationID, nil)

		memoryStorage := storage.NewMemoryStorage()
		err := memoryStorage.Instances().Insert(internal.Instance{
			InstanceID: "some-instance-id",
			RuntimeID:  "not-matching-id",
		})
		assert.NoError(t, err)
		err = memoryStorage.Instances().Insert(internal.Instance{
			InstanceID: fixInstanceID1,
			RuntimeID:  fixRuntimeID1,
		})
		assert.NoError(t, err)
		err = memoryStorage.Instances().Insert(internal.Instance{
			InstanceID: fixInstanceID2,
			RuntimeID:  fixRuntimeID2,
		})
		assert.NoError(t, err)
		logger := logrus.New()

		svc := NewService(gcMock, bcMock, k8sClient, memoryStorage.Instances(), logger, maxShootAge, shootLabelSelector)

		// when
		err = svc.PerformCleanup()

		// then
		bcMock.AssertExpectations(t)
		gcMock.AssertExpectations(t)

		assert.NoError(t, err)
	})

	t.Run("should return error on KEB deprovision call failure", func(t *testing.T) {
		// given
		gcMock := &mocks.GardenerClient{}
		gcMock.On("List", mock.Anything, mock.AnythingOfType("v1.ListOptions")).Return(fixShootList(), nil)
		bcMock := &mocks.BrokerClient{}
		bcMock.On("Deprovision", mock.AnythingOfType("internal.Instance")).Return("",
			fmt.Errorf("failed to deprovision instance"))

		memoryStorage := storage.NewMemoryStorage()
		err := memoryStorage.Instances().Insert(internal.Instance{
			InstanceID: fixInstanceID1,
			RuntimeID:  fixRuntimeID1,
		})
		assert.NoError(t, err)
		err = memoryStorage.Instances().Insert(internal.Instance{
			InstanceID: fixInstanceID2,
			RuntimeID:  fixRuntimeID2,
		})
		assert.NoError(t, err)
		err = memoryStorage.Instances().Insert(internal.Instance{
			InstanceID: fixInstanceID3,
			RuntimeID:  fixRuntimeID3,
		})
		assert.NoError(t, err)

		logger := logrus.New()

		svc := NewService(gcMock, bcMock, k8sClient, memoryStorage.Instances(), logger, maxShootAge, shootLabelSelector)

		// when
		err = svc.PerformCleanup()

		// then
		bcMock.AssertExpectations(t)
		gcMock.AssertExpectations(t)
		assert.Error(t, err)
	})

	t.Run("should pass when shoot has no runtime id annotation or account label", func(t *testing.T) {
		// given
		gcMock := &mocks.GardenerClient{}
		creationTime, parseErr := time.Parse(time.RFC3339, "2020-01-02T10:00:00-05:00")
		require.NoError(t, parseErr)
		unl := unstructured.UnstructuredList{
			Items: []unstructured.Unstructured{
				{
					Object: map[string]interface{}{
						"metadata": map[string]interface{}{
							"name":              "az-1234",
							"creationTimestamp": creationTime,
							"annotations": map[string]interface{}{
								shootAnnotationRuntimeId: fixRuntimeID1,
							},
							"clusterName": "cluster-one",
						},
						"spec": map[string]interface{}{
							"cloudProfileName": "az",
						},
					},
				},
				{
					Object: map[string]interface{}{
						"metadata": map[string]interface{}{
							"name":              "az-1234",
							"creationTimestamp": creationTime,
							"clusterName":       "cluster-one",
							"annotations":       map[string]interface{}{},
						},
						"spec": map[string]interface{}{
							"cloudProfileName": "az",
						},
					},
				},
				{
					Object: map[string]interface{}{
						"metadata": map[string]interface{}{
							"name":              "az-1234",
							"creationTimestamp": creationTime,
							"annotations": map[string]interface{}{
								shootAnnotationInfrastructureManagerRuntimeId: fixRuntimeID2,
								shootLabelAccountId:                           fixAccountID,
							},
							"clusterName": "cluster-one",
						},
						"spec": map[string]interface{}{
							"cloudProfileName": "az",
						},
					},
				},
			},
		}
		gcMock.On("List", mock.Anything, mock.AnythingOfType("v1.ListOptions")).Return(&unl, nil)
		gcMock.On("Delete", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("v1.DeleteOptions")).Return(nil)
		gcMock.On("Update", mock.Anything, mock.Anything, mock.AnythingOfType("v1.UpdateOptions")).Return(nil, nil)

		bcMock := &mocks.BrokerClient{}

		memoryStorage := storage.NewMemoryStorage()
		err := memoryStorage.Instances().Insert(internal.Instance{
			InstanceID: fixInstanceID1,
			RuntimeID:  fixRuntimeID1,
		})
		assert.NoError(t, err)

		var actualLog bytes.Buffer
		logger := logrus.New()
		logger.SetFormatter(&logrus.TextFormatter{
			DisableTimestamp: true,
		})
		logger.SetOutput(&actualLog)

		svc := NewService(gcMock, bcMock, k8sClient, memoryStorage.Instances(), logger, maxShootAge, shootLabelSelector)

		// when
		err = svc.PerformCleanup()

		// then
		bcMock.AssertExpectations(t)
		gcMock.AssertExpectations(t)
		assert.NoError(t, err)
	})

	t.Run("should delete runtime CR when unable to find instance in db", func(t *testing.T) {
		// given
		gcMock := &mocks.GardenerClient{}
		gcMock.On("List", mock.Anything, mock.AnythingOfType("v1.ListOptions")).Return(fixShootList(), nil)
		gcMock.On("Delete", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("v1.DeleteOptions")).Return(nil)
		gcMock.On("Update", mock.Anything, mock.Anything, mock.AnythingOfType("v1.UpdateOptions")).Return(nil, nil)
		bcMock := &mocks.BrokerClient{}
		bcMock.On("Deprovision", mock.AnythingOfType("internal.Instance")).Return(fixOperationID, nil)

		memoryStorage := storage.NewMemoryStorage()
		err := memoryStorage.Instances().Insert(internal.Instance{
			InstanceID: fixInstanceID1,
			RuntimeID:  fixRuntimeID1,
		})
		assert.NoError(t, err)
		err = memoryStorage.Instances().Insert(internal.Instance{
			InstanceID: fixInstanceID2,
			RuntimeID:  fixRuntimeID2,
		})
		assert.NoError(t, err)

		runtimeCR := &imv1.Runtime{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fixRuntimeID3,
				Namespace: kcpNamespace,
			},
			Spec: imv1.RuntimeSpec{
				Shoot: imv1.RuntimeShoot{
					Name: "az-4567",
				},
			},
		}
		err = k8sClient.Create(context.Background(), runtimeCR)
		assert.NoError(t, err)
		err = k8sClient.Get(context.Background(), client.ObjectKey{Name: fixRuntimeID3, Namespace: kcpNamespace}, &imv1.Runtime{})
		assert.NoError(t, err)

		logger := logrus.New()

		svc := NewService(gcMock, bcMock, k8sClient, memoryStorage.Instances(), logger, maxShootAge, shootLabelSelector)

		// when
		err = svc.PerformCleanup()

		// then
		bcMock.AssertExpectations(t)
		gcMock.AssertExpectations(t)
		assert.NoError(t, err)

		err = k8sClient.Get(context.Background(), client.ObjectKey{Name: fixRuntimeID3, Namespace: kcpNamespace}, &imv1.Runtime{})
		assert.EqualError(t, err, "runtimes.infrastructuremanager.kyma-project.io \"rntime-3\" not found")
	})

	t.Run("should not delete runtime CR with invalid shoot name", func(t *testing.T) {
		// given
		gcMock := &mocks.GardenerClient{}
		gcMock.On("List", mock.Anything, mock.AnythingOfType("v1.ListOptions")).Return(fixShootList(), nil)
		gcMock.On("Delete", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("v1.DeleteOptions")).Return(nil)
		gcMock.On("Update", mock.Anything, mock.Anything, mock.AnythingOfType("v1.UpdateOptions")).Return(nil, nil)
		bcMock := &mocks.BrokerClient{}
		bcMock.On("Deprovision", mock.AnythingOfType("internal.Instance")).Return(fixOperationID, nil)

		memoryStorage := storage.NewMemoryStorage()
		err := memoryStorage.Instances().Insert(internal.Instance{
			InstanceID: fixInstanceID1,
			RuntimeID:  fixRuntimeID1,
		})
		assert.NoError(t, err)
		err = memoryStorage.Instances().Insert(internal.Instance{
			InstanceID: fixInstanceID2,
			RuntimeID:  fixRuntimeID2,
		})
		assert.NoError(t, err)

		runtimeCR := &imv1.Runtime{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fixRuntimeID3,
				Namespace: kcpNamespace,
			},
			Spec: imv1.RuntimeSpec{
				Shoot: imv1.RuntimeShoot{
					Name: "invalid-name",
				},
			},
		}
		err = k8sClient.Create(context.Background(), runtimeCR)
		assert.NoError(t, err)
		err = k8sClient.Get(context.Background(), client.ObjectKey{Name: fixRuntimeID3, Namespace: kcpNamespace}, &imv1.Runtime{})
		assert.NoError(t, err)

		logger := logrus.New()

		svc := NewService(gcMock, bcMock, k8sClient, memoryStorage.Instances(), logger, maxShootAge, shootLabelSelector)

		// when
		err = svc.PerformCleanup()

		// then
		bcMock.AssertExpectations(t)
		gcMock.AssertExpectations(t)
		assert.NoError(t, err)

		err = k8sClient.Get(context.Background(), client.ObjectKey{Name: fixRuntimeID3, Namespace: kcpNamespace}, &imv1.Runtime{})
		assert.NoError(t, err)
	})
}

func fixShootList() *unstructured.UnstructuredList {
	return &unstructured.UnstructuredList{
		Items: fixShootListItems(),
	}
}

func fixShootListItems() []unstructured.Unstructured {
	creationTime, _ := time.Parse(time.RFC3339, "2020-01-02T10:00:00-05:00")
	unl := unstructured.UnstructuredList{
		Items: []unstructured.Unstructured{
			{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"name":              "simple-shoot",
						"creationTimestamp": creationTime,
						"labels": map[string]interface{}{
							"should-be-deleted": "true",
						},
						"annotations": map[string]interface{}{},
					},
					"spec": map[string]interface{}{
						"cloudProfileName": "az",
					},
				},
			},
			{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"name":              "az-1234",
						"creationTimestamp": creationTime,
						"labels": map[string]interface{}{
							"should-be-deleted": "true",
							shootLabelAccountId: fixAccountID,
						},
						"annotations": map[string]interface{}{
							shootAnnotationRuntimeId: fixRuntimeID1,
						},
						"clusterName": "cluster-one",
					},
					"spec": map[string]interface{}{
						"cloudProfileName": "az",
					},
				},
			},
			{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"name":              "gcp-1234",
						"creationTimestamp": creationTime,
						"labels": map[string]interface{}{
							"should-be-deleted": "true",
							shootLabelAccountId: fixAccountID,
						},
						"annotations": map[string]interface{}{
							shootAnnotationRuntimeId: fixRuntimeID2,
						},
						"clusterName": "cluster-two",
					},
					"spec": map[string]interface{}{
						"cloudProfileName": "gcp",
					},
				},
			},
			{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"name":              "az-4567",
						"creationTimestamp": creationTime,
						"labels": map[string]interface{}{
							shootLabelAccountId: fixAccountID,
						},
						"annotations": map[string]interface{}{
							shootAnnotationRuntimeId: fixRuntimeID3,
						},
						"clusterName": "cluster-one",
					},
					"spec": map[string]interface{}{
						"cloudProfileName": "az",
					},
				},
			},
		},
	}
	return unl.Items
}
