package metrics

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	ga1 = "ga-111"
	ga2 = "ga-222"

	bindingAzure  = "azure"
	bindingAzure2 = "azure-2"
	bindingAWS    = "aws"
)

func TestCredentialsBindingsCollector(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	db := storage.NewMemoryStorage()
	instances := db.Instances()

	require.NoError(t, instances.Insert(fixCredentialInstance("i-1", ga1, bindingAzure)))
	require.NoError(t, instances.Insert(fixCredentialInstance("i-2", ga1, bindingAzure)))
	require.NoError(t, instances.Insert(fixCredentialInstance("i-3", ga1, bindingAzure2)))
	require.NoError(t, instances.Insert(fixCredentialInstance("i-4", ga2, bindingAWS)))
	require.NoError(t, instances.Insert(fixCredentialInstance("i-5", ga2, bindingAWS)))
	require.NoError(t, instances.Insert(fixCredentialInstance("i-6", ga2, bindingAWS)))
	deleted := fixCredentialInstance("i-7", ga1, bindingAzure)
	deleted.DeletedAt = time.Now()
	require.NoError(t, instances.Insert(deleted))

	collector := NewCredentialsBindingsCollector(instances, nil, 1*time.Minute, 5*time.Minute, log)
	t.Cleanup(func() {
		prometheus.Unregister(collector.instancesPerCredentialsBinding)
		prometheus.Unregister(collector.availableCredentialsBindings)
	})

	t.Run("initial counts are correct after first poll", func(t *testing.T) {
		collector.updateInstancesMetrics()

		assert.Equal(t, float64(2), gaugeValue(collector, bindingAzure, ga1), "GA1/azure: 2 active instances")
		assert.Equal(t, float64(1), gaugeValue(collector, bindingAzure2, ga1), "GA1/azure-2: 1 active instance")
		assert.Equal(t, float64(3), gaugeValue(collector, bindingAWS, ga2), "GA2/aws: 3 active instances")
	})

	t.Run("deleted instance is not counted", func(t *testing.T) {
		assert.Equal(t, float64(2), gaugeValue(collector, bindingAzure, ga1))
	})

	t.Run("counts are updated after new instance is added", func(t *testing.T) {
		require.NoError(t, instances.Insert(fixCredentialInstance("i-9", ga2, bindingAWS)))

		collector.updateInstancesMetrics()

		assert.Equal(t, float64(4), gaugeValue(collector, bindingAWS, ga2), "GA2/aws: 4 instances after insert")
		assert.Equal(t, float64(2), gaugeValue(collector, bindingAzure, ga1))
		assert.Equal(t, float64(1), gaugeValue(collector, bindingAzure2, ga1))
	})
}

func gaugeValue(c *CredentialsBindingsCollector, binding, globalAccountID string) float64 {
	return testutil.ToFloat64(c.instancesPerCredentialsBinding.With(prometheus.Labels{
		"credentials_binding": binding,
		"global_account_id":   globalAccountID,
	}))
}

func fixCredentialInstance(id, globalAccountID, subscriptionSecretName string) internal.Instance {
	return internal.Instance{
		InstanceID:             id,
		GlobalAccountID:        globalAccountID,
		SubscriptionSecretName: subscriptionSecretName,
	}
}

func TestAvailableCredentialsBindingsCollector(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	const namespace = "test"

	newCollector := func(t *testing.T, objects ...runtime.Object) *CredentialsBindingsCollector {
		t.Helper()
		gc := gardener.NewClient(gardener.NewDynamicFakeClient(objects...), namespace)
		c := NewCredentialsBindingsCollector(nil, gc, 1*time.Minute, 5*time.Minute, log)
		t.Cleanup(func() {
			prometheus.Unregister(c.instancesPerCredentialsBinding)
			prometheus.Unregister(c.availableCredentialsBindings)
		})
		return c
	}

	t.Run("counts unclaimed bindings per hyperscaler type", func(t *testing.T) {
		awsUnclaimed1 := fixCredentialsBinding("aws-pool-1", namespace, map[string]string{gardener.HyperscalerTypeLabelKey: "aws"})
		awsUnclaimed2 := fixCredentialsBinding("aws-pool-2", namespace, map[string]string{gardener.HyperscalerTypeLabelKey: "aws"})
		azureUnclaimed := fixCredentialsBinding("azure-pool-1", namespace, map[string]string{gardener.HyperscalerTypeLabelKey: "azure"})
		awsClaimed := fixCredentialsBinding("aws-claimed", namespace, map[string]string{
			gardener.HyperscalerTypeLabelKey: "aws",
			gardener.TenantNameLabelKey:      "some-ga",
		})
		azureClaimed := fixCredentialsBinding("azure-claimed", namespace, map[string]string{
			gardener.HyperscalerTypeLabelKey: "azure",
			gardener.TenantNameLabelKey:      "some-other-ga",
		})

		collector := newCollector(t, awsUnclaimed1, awsUnclaimed2, azureUnclaimed, awsClaimed, azureClaimed)
		collector.updateAvailableMetrics()

		assert.Equal(t, float64(2), poolGaugeValue(collector, "aws"), "2 unclaimed AWS bindings")
		assert.Equal(t, float64(1), poolGaugeValue(collector, "azure"), "1 unclaimed Azure binding")
	})

	t.Run("excludes shared bindings", func(t *testing.T) {
		awsAvailable := fixCredentialsBinding("aws-available", namespace, map[string]string{
			gardener.HyperscalerTypeLabelKey: "aws",
		})
		awsShared := fixCredentialsBinding("aws-shared", namespace, map[string]string{
			gardener.HyperscalerTypeLabelKey: "aws",
			gardener.SharedLabelKey:          "true",
		})
		azureAvailable := fixCredentialsBinding("azure-available", namespace, map[string]string{
			gardener.HyperscalerTypeLabelKey: "azure",
		})
		azureShared := fixCredentialsBinding("azure-shared", namespace, map[string]string{
			gardener.HyperscalerTypeLabelKey: "azure",
			gardener.SharedLabelKey:          "true",
		})

		collector := newCollector(t, awsAvailable, awsShared, azureAvailable, azureShared)
		collector.updateAvailableMetrics()

		assert.Equal(t, float64(1), poolGaugeValue(collector, "aws"), "shared AWS binding must not be counted as available")
		assert.Equal(t, float64(1), poolGaugeValue(collector, "azure"), "shared Azure binding must not be counted as available")
	})

	t.Run("excludes dirty bindings", func(t *testing.T) {
		awsAvailable := fixCredentialsBinding("aws-available", namespace, map[string]string{
			gardener.HyperscalerTypeLabelKey: "aws",
		})
		awsDirty := fixCredentialsBinding("aws-dirty", namespace, map[string]string{
			gardener.HyperscalerTypeLabelKey: "aws",
			gardener.DirtyLabelKey:           "true",
		})
		gcpAvailable := fixCredentialsBinding("gcp-available", namespace, map[string]string{
			gardener.HyperscalerTypeLabelKey: "gcp",
		})
		gcpDirty := fixCredentialsBinding("gcp-dirty", namespace, map[string]string{
			gardener.HyperscalerTypeLabelKey: "gcp",
			gardener.DirtyLabelKey:           "true",
		})

		collector := newCollector(t, awsAvailable, awsDirty, gcpAvailable, gcpDirty)
		collector.updateAvailableMetrics()

		assert.Equal(t, float64(1), poolGaugeValue(collector, "aws"), "dirty AWS binding must not be counted as available")
		assert.Equal(t, float64(1), poolGaugeValue(collector, "gcp"), "dirty GCP binding must not be counted as available")
	})
}

func poolGaugeValue(c *CredentialsBindingsCollector, hyperscalerType string) float64 {
	return testutil.ToFloat64(c.availableCredentialsBindings.With(prometheus.Labels{
		"hyperscaler_type": hyperscalerType,
	}))
}

func fixCredentialsBinding(name, namespace string, labels map[string]string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
		},
	}
	u.SetGroupVersionKind(gardener.CredentialsBindingGVK)
	u.SetLabels(labels)
	return u
}
