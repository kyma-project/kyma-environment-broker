package steps

import (
	"strings"

	"github.com/kyma-project/kyma-environment-broker/common/runtime"

	"github.com/kyma-project/kyma-environment-broker/internal/customresources"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ApplyLabelsAndAnnotationsForLM Set common labels and annotations for kyma lifecycle manager
func ApplyLabelsAndAnnotationsForLM(object client.Object, operation internal.Operation) {

	cloudProvider := runtime.CloudProviderFromString(operation.ProviderValues.ProviderType)
	l := object.GetLabels()
	l = SetCommonLabels(l, operation)

	l[customresources.RegionLabel] = operation.Region
	l[customresources.ManagedByLabel] = "lifecycle-manager"
	l[customresources.CloudProviderLabel] = string(cloudProvider)

	if isKymaResourceInternal(operation) {
		l[customresources.InternalLabel] = "true"
	}

	object.SetLabels(l)

	a := object.GetAnnotations()
	if a == nil {
		a = make(map[string]string)
	}
	a["skr-domain"] = operation.ShootDomain
	object.SetAnnotations(a)
}

func KymaName(operation internal.Operation) string {
	if operation.KymaResourceName != "" {
		return operation.KymaResourceName
	}
	return CreateKymaNameFromOperation(operation)
}

func KymaRuntimeResourceName(operation internal.Operation) string {
	return KymaRuntimeResourceNameFromID(operation.RuntimeID)
}

func KymaNameFromInstance(instance *internal.Instance) string {
	return KymaRuntimeResourceNameFromID(instance.RuntimeID)
}

func KymaRuntimeResourceNameFromID(ID string) string {
	return strings.ToLower(ID)
}

func CreateKymaNameFromOperation(operation internal.Operation) string {
	return strings.ToLower(operation.RuntimeID)
}

func isKymaResourceInternal(operation internal.Operation) bool {
	return !*operation.ProvisioningParameters.ErsContext.ExternalLicenseType()
}
