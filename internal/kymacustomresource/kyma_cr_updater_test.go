package kymacustomresource

import (
	"context"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal/syncqueues"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
)

// Kyma CR K8s data
const (
	group    = "operator.kyma-project.io"
	version  = "v1beta2"
	resource = "kymas"
	kind     = "Kyma"
)

const (
	subaccountID = "subaccount-id-1"
)

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
		kymaCRName := "kyma-cr-1"
		mockKymaCR := unstructured.Unstructured{}
		mockKymaCR.SetGroupVersionKind(gvk)
		mockKymaCR.SetName(kymaCRName)
		mockKymaCR.SetNamespace(namespace)
		require.NoError(t, unstructured.SetNestedField(mockKymaCR.Object, nil, "metadata", "creationTimestamp"))
		queue := &fakePriorityQueue{}
		fakeK8sClient := fake.NewSimpleDynamicClient(scheme, &mockKymaCR)
		updater, err := NewUpdater(fakeK8sClient, queue, gvr)
		require.NoError(t, err)

		// when
		require.NoError(t, updater.Run())

		// then
		assert.True(t, queue.IsEmpty())

		actual, err := fakeK8sClient.Resource(gvr).Get(context.TODO(), kymaCRName, metav1.GetOptions{})
		require.NoError(t, err)
		assert.NotContains(t, actual.GetLabels(), betaEnabledLabelKey)
	})

}

type fakePriorityQueue struct {
	elements []syncqueues.QueueElement
}

func (f *fakePriorityQueue) Insert(e syncqueues.QueueElement) {
	f.elements = append(f.elements, e)
}

func (f *fakePriorityQueue) Extract() syncqueues.QueueElement {
	extractedElement := f.elements[0]
	f.elements = f.elements[1:]
	return extractedElement
}

func (f *fakePriorityQueue) IsEmpty() bool {
	return len(f.elements) == 0
}
