package k8s

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func AddOrOverrideMetadata(k8sObject *unstructured.Unstructured, key, value string) error {
	if k8sObject == nil {
		return fmt.Errorf("object is nil")
	}

	labels := (*k8sObject).GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[key] = value
	(*k8sObject).SetLabels(labels)

	return nil
}
