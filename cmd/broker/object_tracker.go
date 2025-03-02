package main

import (
	"sync"
	"time"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	k8stesting "k8s.io/client-go/testing"
)

func NewTestingObjectTracker(sch *runtime.Scheme) *testingObjectTracker {
	return &testingObjectTracker{
		target:           k8stesting.NewObjectTracker(sch, scheme.Codecs.UniversalDecoder()),
		runtimesToDelete: map[string]struct{}{},
	}
}

type testingObjectTracker struct {
	target k8stesting.ObjectTracker

	mu               sync.RWMutex
	runtimesToDelete map[string]struct{}
}

type Deleter interface {
	ProcessRuntimeDeletion(name string)
}

var _ k8stesting.ObjectTracker = &testingObjectTracker{}
var _ Deleter = &testingObjectTracker{}

func (ot *testingObjectTracker) Add(obj runtime.Object) error {
	return ot.target.Add(obj)
}

func (ot *testingObjectTracker) Get(gvr schema.GroupVersionResource, ns, name string) (runtime.Object, error) {
	return ot.target.Get(gvr, ns, name)
}

func (ot *testingObjectTracker) Create(gvr schema.GroupVersionResource, obj runtime.Object, ns string) error {
	return ot.target.Create(gvr, obj, ns)
}

func (ot *testingObjectTracker) Update(gvr schema.GroupVersionResource, obj runtime.Object, ns string) error {
	return ot.target.Update(gvr, obj, ns)
}

func (ot *testingObjectTracker) List(gvr schema.GroupVersionResource, gvk schema.GroupVersionKind, ns string) (runtime.Object, error) {
	return ot.target.List(gvr, gvk, ns)
}

func (ot *testingObjectTracker) Delete(gvr schema.GroupVersionResource, ns, name string) error {
	if gvr.Resource == "runtimes" {
		ot.mu.Lock()
		defer ot.mu.Unlock()
		ot.runtimesToDelete[name] = struct{}{}
		return nil
	}
	return ot.target.Delete(gvr, ns, name)
}

func (ot *testingObjectTracker) Watch(gvr schema.GroupVersionResource, ns string) (watch.Interface, error) {
	return ot.target.Watch(gvr, ns)
}

func (ot *testingObjectTracker) ProcessRuntimeDeletion(name string) {
	for {
		time.Sleep(time.Millisecond)
		ot.mu.RLock()
		_, ok := ot.runtimesToDelete[name]
		ot.mu.RUnlock()
		if ok {
			break
		}
	}
	ot.deleteRuntimeIfExist(name)
}

func (ot *testingObjectTracker) deleteRuntimeIfExist(name string) {
	ot.mu.RLock()
	defer ot.mu.RUnlock()
	if _, ok := ot.runtimesToDelete[name]; ok {
		_ = ot.target.Delete(imv1.GroupVersion.WithResource("runtimes"), "kyma-system", name)
	}
}
