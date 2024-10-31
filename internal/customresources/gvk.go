package customresources

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	KymaCr            = "kyma"
	GardenerClusterCr = "gardenercluster"
	RuntimeCr         = "runtime"
)

var gvkMap = map[string]schema.GroupVersionKind{
	KymaCr: {
		Group:   "operator.kyma-project.io",
		Version: "v1beta2",
		Kind:    "Kyma",
	},
	GardenerClusterCr: {
		Group:   "infrastructuremanager.kyma-project.io",
		Version: "v1",
		Kind:    "GardenerCluster",
	},
	RuntimeCr: {
		Group:   "infrastructuremanager.kyma-project.io",
		Version: "v1",
		Kind:    "Runtime",
	},
}

func GvkByName(name string) (schema.GroupVersionKind, error) {
	gvk, ok := gvkMap[strings.ToLower(name)]
	if !ok {
		return schema.GroupVersionKind{}, fmt.Errorf("unknown name: %s", name)
	}
	return gvk, nil
}
