package update

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"testing"

	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/stretchr/testify/require"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestUpdateRuntimeStep_NoRuntime(t *testing.T) {
	// given
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	kcpClient := fake.NewClientBuilder().Build()
	step := NewUpdateRuntimeStep(nil, kcpClient, 0)
	operation := fixture.FixUpdatingOperation("op-id", "inst-id").Operation
	operation.RuntimeResourceName = "runtime-name"
	operation.KymaResourceNamespace = "kyma-ns"

	// when
	_, backoff, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, backoff)
}

func TestUpdateRuntimeStep_RunUpdateMachineType(t *testing.T) {
	// given
	err := imv1.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)
	kcpClient := fake.NewClientBuilder().WithRuntimeObjects(fixRuntimeResource("runtime-name", false)).Build()
	step := NewUpdateRuntimeStep(nil, kcpClient, 0)
	operation := fixture.FixUpdatingOperation("op-id", "inst-id").Operation
	operation.RuntimeResourceName = "runtime-name"
	operation.KymaResourceNamespace = "kcp-system"
	operation.UpdatingParameters = internal.UpdatingParametersDTO{
		MachineType: ptr.String("new-machine-type"),
	}

	// when
	_, backoff, err := step.Run(operation, fixLogger())

	// then
	assert.NoError(t, err)
	assert.Zero(t, backoff)

	var gotRuntime imv1.Runtime
	err = kcpClient.Get(context.Background(), client.ObjectKey{Name: operation.RuntimeResourceName, Namespace: "kcp-system"}, &gotRuntime)
	require.NoError(t, err)
	assert.Equal(t, "new-machine-type", gotRuntime.Spec.Shoot.Provider.Workers[0].Machine.Type)

}

func fixRuntimeResource(name string, controlledByProvisioner bool) runtime.Object {
	maxSurge := intstr.FromInt32(1)
	maxUnavailable := intstr.FromInt32(0)
	return &imv1.Runtime{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: "kcp-system",
			Labels: map[string]string{
				imv1.LabelControlledByProvisioner: strconv.FormatBool(controlledByProvisioner),
			},
		},
		Spec: imv1.RuntimeSpec{
			Shoot: imv1.RuntimeShoot{
				Provider: imv1.Provider{
					Workers: []gardener.Worker{
						{
							Machine: gardener.Machine{
								Type: "original-type",
							},
							MaxSurge:       &maxSurge,
							MaxUnavailable: &maxUnavailable,
						},
					},
				},
			},
		},
	}
}

func fixLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})).With("testing", true)
}
