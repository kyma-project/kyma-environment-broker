package kymacustomresource

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/syncqueues"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic/fake"
)

// Kyma CR K8s data
const (
	group                      = "operator.kyma-project.io"
	version                    = "v1beta2"
	resource                   = "kymas"
	kind                       = "Kyma"
	kymaCRName1                = "kyma-cr-1"
	kymaCRName2                = "kyma-cr-2"
	usedForProductionTestValue = "USED_FOR_PRODUCTION"
)

const (
	subaccountID = "subaccount-id-1"
	interval     = 100 * time.Millisecond
	timeout      = 2 * time.Second
)

var log = slog.New(slog.NewTextHandler(os.Stderr, nil))

func isBetaEnabledTrue(val string) bool {
	return val == "true"
}

func TestUpdater(t *testing.T) {
	// given
	gvr := schema.GroupVersionResource{Group: group, Version: version, Resource: resource}
	gvk := gvr.GroupVersion().WithKind(kind)
	listGVK := gvk
	listGVK.Kind += "List"

	var kymaKind, kymaKindList unstructured.Unstructured
	kymaKind.SetGroupVersionKind(gvk)
	kymaKindList.SetGroupVersionKind(listGVK)

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(gvr.GroupVersion(), &kymaKind, &kymaKindList)

	t.Run("should not update Kyma CRs when the queue is empty", func(t *testing.T) {
		// given
		kymaCRName := kymaCRName1
		mockKymaCR := &unstructured.Unstructured{}
		mockKymaCR.SetGroupVersionKind(gvk)
		mockKymaCR.SetName(kymaCRName)
		mockKymaCR.SetNamespace(namespace)
		require.NoError(t, unstructured.SetNestedField(mockKymaCR.Object, nil, "metadata", "creationTimestamp"))

		queue := syncqueues.NewPriorityQueueWithCallbacksForSize(log, nil, 4)
		fakeK8sClient := fake.NewSimpleDynamicClient(scheme, mockKymaCR)
		updater, err := NewUpdater(fakeK8sClient, queue, gvr, timeout, context.TODO(), log)
		require.NoError(t, err)

		// when
		go func(t *testing.T) {
			require.NoError(t, updater.Run())
		}(t)

		// then
		assert.True(t, queue.IsEmpty())

		actual, err := fakeK8sClient.Resource(gvr).Namespace(namespace).Get(context.TODO(), kymaCRName, metav1.GetOptions{})
		require.NoError(t, err)
		assert.NotContains(t, actual.GetLabels(), BetaEnabledLabelKey)
	})

	t.Run("should update a Kyma CR with the given subaccount id label when the queue has a matching element", func(t *testing.T) {
		// given
		kymaCRName := kymaCRName1
		mockKymaCR := &unstructured.Unstructured{}
		mockKymaCR.SetGroupVersionKind(gvk)
		mockKymaCR.SetName(kymaCRName)
		mockKymaCR.SetNamespace(namespace)
		mockKymaCR.SetLabels(map[string]string{subaccountIdLabelKey: subaccountID})
		require.NoError(t, unstructured.SetNestedField(mockKymaCR.Object, nil, "metadata", "creationTimestamp"))

		queue := syncqueues.NewPriorityQueueWithCallbacksForSize(log, nil, 4)
		queue.Insert(syncqueues.QueueElement{
			SubaccountID:      subaccountID,
			BetaEnabled:       "true",
			UsedForProduction: usedForProductionTestValue,
			ModifiedAt:        time.Now().Unix(),
		})
		assert.False(t, queue.IsEmpty())

		fakeK8sClient := fake.NewSimpleDynamicClient(scheme, mockKymaCR)
		updater, err := NewUpdater(fakeK8sClient, queue, gvr, timeout, context.TODO(), log)
		require.NoError(t, err)

		// when
		go func(t *testing.T) {
			require.NoError(t, updater.Run())
		}(t)

		// then
		err = wait.PollUntilContextTimeout(context.Background(), interval, timeout, true, func(ctx context.Context) (bool, error) {
			actual, err := fakeK8sClient.Resource(gvr).Namespace(namespace).Get(context.TODO(), kymaCRName, metav1.GetOptions{})
			require.NoError(t, err)
			if isBetaEnabledTrue(actual.GetLabels()[BetaEnabledLabelKey]) && actual.GetLabels()[UsedForProductionLabelKey] == usedForProductionTestValue {
				return true, nil
			}
			return false, nil
		})
		require.NoError(t, err)
		assert.True(t, queue.IsEmpty())
	})

	t.Run("should update all Kyma CRs with the given subaccount id label when the queue has a matching element", func(t *testing.T) {
		// given
		kymaCRName1, kymaCRName2 := kymaCRName1, kymaCRName2
		mockKymaCR1 := &unstructured.Unstructured{}
		mockKymaCR1.SetGroupVersionKind(gvk)
		mockKymaCR1.SetName(kymaCRName1)
		mockKymaCR1.SetNamespace(namespace)
		mockKymaCR1.SetLabels(map[string]string{subaccountIdLabelKey: subaccountID})
		require.NoError(t, unstructured.SetNestedField(mockKymaCR1.Object, nil, "metadata", "creationTimestamp"))

		mockKymaCR2 := mockKymaCR1.DeepCopy()
		mockKymaCR2.SetName(kymaCRName2)

		queue := syncqueues.NewPriorityQueueWithCallbacksForSize(log, nil, 4)

		queue.Insert(syncqueues.QueueElement{
			SubaccountID:      subaccountID,
			BetaEnabled:       "true",
			UsedForProduction: usedForProductionTestValue,
			ModifiedAt:        time.Now().Unix(),
		})
		assert.False(t, queue.IsEmpty())

		fakeK8sClient := fake.NewSimpleDynamicClient(scheme, mockKymaCR1, mockKymaCR2)
		updater, err := NewUpdater(fakeK8sClient, queue, gvr, timeout, context.TODO(), log)
		require.NoError(t, err)

		// when
		go func(t *testing.T) {
			require.NoError(t, updater.Run())
		}(t)

		// then
		err = wait.PollUntilContextTimeout(context.Background(), interval, timeout, true, func(ctx context.Context) (bool, error) {
			actual, err := fakeK8sClient.Resource(gvr).Namespace(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: fmt.Sprintf(subaccountIdLabelFormat, subaccountID)})
			assert.Len(t, actual.Items, 2)
			require.NoError(t, err)
			for _, un := range actual.Items {
				if !isBetaEnabledTrue(un.GetLabels()[BetaEnabledLabelKey]) && un.GetLabels()[UsedForProductionLabelKey] != usedForProductionTestValue {
					return false, nil
				}
			}
			return true, nil
		})
		require.NoError(t, err)
		assert.True(t, queue.IsEmpty())
	})

	t.Run("should update just one Kyma CR with the given subaccount id label when the queue has a matching element", func(t *testing.T) {
		// given
		kymaCRName1, kymaCRName2 := kymaCRName1, kymaCRName2
		otherSubaccountID := "subaccount-id-2"

		mockKymaCR1 := &unstructured.Unstructured{}
		mockKymaCR1.SetGroupVersionKind(gvk)
		mockKymaCR1.SetName(kymaCRName1)
		mockKymaCR1.SetNamespace(namespace)
		require.NoError(t, unstructured.SetNestedField(mockKymaCR1.Object, nil, "metadata", "creationTimestamp"))

		mockKymaCR2 := mockKymaCR1.DeepCopy()
		mockKymaCR2.SetName(kymaCRName2)

		mockKymaCR1.SetLabels(map[string]string{subaccountIdLabelKey: subaccountID})
		mockKymaCR2.SetLabels(map[string]string{subaccountIdLabelKey: otherSubaccountID})

		queue := syncqueues.NewPriorityQueueWithCallbacksForSize(log, nil, 4)
		queue.Insert(syncqueues.QueueElement{
			SubaccountID:      subaccountID,
			BetaEnabled:       "true",
			UsedForProduction: usedForProductionTestValue,
			ModifiedAt:        time.Now().Unix(),
		})
		assert.False(t, queue.IsEmpty())

		fakeK8sClient := fake.NewSimpleDynamicClient(scheme, mockKymaCR1, mockKymaCR2)
		updater, err := NewUpdater(fakeK8sClient, queue, gvr, timeout, context.TODO(), log)
		require.NoError(t, err)

		// when
		go func(t *testing.T) {
			require.NoError(t, updater.Run())
		}(t)

		// then
		err = wait.PollUntilContextTimeout(context.Background(), interval, timeout, true, func(ctx context.Context) (bool, error) {
			actual, err := fakeK8sClient.Resource(gvr).Namespace(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: fmt.Sprintf(subaccountIdLabelFormat, subaccountID)})
			require.NoError(t, err)
			assert.Len(t, actual.Items, 1)
			for _, un := range actual.Items {
				if !isBetaEnabledTrue(un.GetLabels()[BetaEnabledLabelKey]) && un.GetLabels()[UsedForProductionLabelKey] != usedForProductionTestValue {
					return false, nil
				}
			}
			return true, nil
		})
		require.NoError(t, err)

		actual, err := fakeK8sClient.Resource(gvr).Namespace(namespace).Get(context.TODO(), kymaCRName2, metav1.GetOptions{})
		require.NoError(t, err)
		assert.NotContains(t, actual.GetLabels(), BetaEnabledLabelKey)
		assert.NotContains(t, actual.GetLabels(), UsedForProductionLabelKey)
		assert.True(t, queue.IsEmpty())
	})
}
