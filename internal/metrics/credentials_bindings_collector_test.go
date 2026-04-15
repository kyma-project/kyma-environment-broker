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
		prometheus.Unregister(collector.claimedCredentialsBindings)
		prometheus.Unregister(collector.dirtyCredentialsBindings)
		prometheus.Unregister(collector.sharedCredentialsBindings)
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

	t.Run("suspended instance is not counted", func(t *testing.T) {
		suspended := fixSuspendedCredentialInstance("i-10", ga1, bindingAzure)
		require.NoError(t, instances.Insert(suspended))

		collector.updateInstancesMetrics()

		assert.Equal(t, float64(2), gaugeValue(collector, bindingAzure, ga1), "GA1/azure: suspended instance must not be counted")
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

func fixSuspendedCredentialInstance(id, globalAccountID, subscriptionSecretName string) internal.Instance {
	active := false
	return internal.Instance{
		InstanceID:             id,
		GlobalAccountID:        globalAccountID,
		SubscriptionSecretName: subscriptionSecretName,
		Parameters: internal.ProvisioningParameters{
			ErsContext: internal.ERSContext{
				Active: &active,
			},
		},
	}
}

func TestAvailableCredentialsBindingsCollector(t *testing.T) {
	t.Run("counts unclaimed bindings per hyperscaler type", func(t *testing.T) {
		awsUnclaimed1 := fixCredentialsBinding("aws-pool-1", gardenerNamespace, map[string]string{gardener.HyperscalerTypeLabelKey: "aws"})
		awsUnclaimed2 := fixCredentialsBinding("aws-pool-2", gardenerNamespace, map[string]string{gardener.HyperscalerTypeLabelKey: "aws"})
		azureUnclaimed := fixCredentialsBinding("azure-pool-1", gardenerNamespace, map[string]string{gardener.HyperscalerTypeLabelKey: "azure"})
		awsClaimed := fixCredentialsBinding("aws-claimed", gardenerNamespace, map[string]string{
			gardener.HyperscalerTypeLabelKey: "aws",
			gardener.TenantNameLabelKey:      "some-ga",
		})
		azureClaimed := fixCredentialsBinding("azure-claimed", gardenerNamespace, map[string]string{
			gardener.HyperscalerTypeLabelKey: "azure",
			gardener.TenantNameLabelKey:      "some-other-ga",
		})

		collector := newGardenerCollector(t, awsUnclaimed1, awsUnclaimed2, azureUnclaimed, awsClaimed, azureClaimed)
		collector.updateGardenerMetrics()

		assert.Equal(t, float64(2), poolGaugeValue(collector, "aws"), "2 unclaimed AWS bindings")
		assert.Equal(t, float64(1), poolGaugeValue(collector, "azure"), "1 unclaimed Azure binding")
	})

	t.Run("excludes shared bindings", func(t *testing.T) {
		awsAvailable := fixCredentialsBinding("aws-available", gardenerNamespace, map[string]string{
			gardener.HyperscalerTypeLabelKey: "aws",
		})
		awsShared := fixCredentialsBinding("aws-shared", gardenerNamespace, map[string]string{
			gardener.HyperscalerTypeLabelKey: "aws",
			gardener.SharedLabelKey:          "true",
		})
		azureAvailable := fixCredentialsBinding("azure-available", gardenerNamespace, map[string]string{
			gardener.HyperscalerTypeLabelKey: "azure",
		})
		azureShared := fixCredentialsBinding("azure-shared", gardenerNamespace, map[string]string{
			gardener.HyperscalerTypeLabelKey: "azure",
			gardener.SharedLabelKey:          "true",
		})

		collector := newGardenerCollector(t, awsAvailable, awsShared, azureAvailable, azureShared)
		collector.updateGardenerMetrics()

		assert.Equal(t, float64(1), poolGaugeValue(collector, "aws"), "shared AWS binding must not be counted as available")
		assert.Equal(t, float64(1), poolGaugeValue(collector, "azure"), "shared Azure binding must not be counted as available")
	})

	t.Run("excludes dirty bindings", func(t *testing.T) {
		awsAvailable := fixCredentialsBinding("aws-available", gardenerNamespace, map[string]string{
			gardener.HyperscalerTypeLabelKey: "aws",
		})
		awsDirty := fixCredentialsBinding("aws-dirty", gardenerNamespace, map[string]string{
			gardener.HyperscalerTypeLabelKey: "aws",
			gardener.DirtyLabelKey:           "true",
		})
		gcpAvailable := fixCredentialsBinding("gcp-available", gardenerNamespace, map[string]string{
			gardener.HyperscalerTypeLabelKey: "gcp",
		})
		gcpDirty := fixCredentialsBinding("gcp-dirty", gardenerNamespace, map[string]string{
			gardener.HyperscalerTypeLabelKey: "gcp",
			gardener.DirtyLabelKey:           "true",
		})

		collector := newGardenerCollector(t, awsAvailable, awsDirty, gcpAvailable, gcpDirty)
		collector.updateGardenerMetrics()

		assert.Equal(t, float64(1), poolGaugeValue(collector, "aws"), "dirty AWS binding must not be counted as available")
		assert.Equal(t, float64(1), poolGaugeValue(collector, "gcp"), "dirty GCP binding must not be counted as available")
	})
}

func TestClaimedCredentialsBindingsCollector(t *testing.T) {
	t.Run("counts claimed bindings per hyperscaler type and tenant", func(t *testing.T) {
		awsClaimed1 := fixCredentialsBinding("aws-claimed-1", gardenerNamespace, map[string]string{
			gardener.HyperscalerTypeLabelKey: "aws",
			gardener.TenantNameLabelKey:      "tenant-a",
		})
		awsClaimed2 := fixCredentialsBinding("aws-claimed-2", gardenerNamespace, map[string]string{
			gardener.HyperscalerTypeLabelKey: "aws",
			gardener.TenantNameLabelKey:      "tenant-a",
		})
		awsClaimedOtherTenant := fixCredentialsBinding("aws-claimed-3", gardenerNamespace, map[string]string{
			gardener.HyperscalerTypeLabelKey: "aws",
			gardener.TenantNameLabelKey:      "tenant-b",
		})
		azureClaimed := fixCredentialsBinding("azure-claimed-1", gardenerNamespace, map[string]string{
			gardener.HyperscalerTypeLabelKey: "azure",
			gardener.TenantNameLabelKey:      "tenant-a",
		})
		awsUnclaimed := fixCredentialsBinding("aws-unclaimed", gardenerNamespace, map[string]string{
			gardener.HyperscalerTypeLabelKey: "aws",
		})

		collector := newGardenerCollector(t, awsClaimed1, awsClaimed2, awsClaimedOtherTenant, azureClaimed, awsUnclaimed)
		collector.updateGardenerMetrics()

		assert.Equal(t, float64(2), claimedGaugeValue(collector, "aws", "tenant-a"), "2 AWS bindings claimed by tenant-a")
		assert.Equal(t, float64(1), claimedGaugeValue(collector, "aws", "tenant-b"), "1 AWS binding claimed by tenant-b")
		assert.Equal(t, float64(1), claimedGaugeValue(collector, "azure", "tenant-a"), "1 Azure binding claimed by tenant-a")
		assert.Equal(t, float64(0), claimedGaugeValue(collector, "aws", "tenant-c"), "unclaimed binding not counted")
	})
}

func TestDirtyCredentialsBindingsCollector(t *testing.T) {
	t.Run("counts dirty bindings per hyperscaler type", func(t *testing.T) {
		awsDirty1 := fixCredentialsBinding("aws-dirty-1", gardenerNamespace, map[string]string{
			gardener.HyperscalerTypeLabelKey: "aws",
			gardener.DirtyLabelKey:           "true",
		})
		awsDirty2 := fixCredentialsBinding("aws-dirty-2", gardenerNamespace, map[string]string{
			gardener.HyperscalerTypeLabelKey: "aws",
			gardener.DirtyLabelKey:           "true",
		})
		gcpDirty := fixCredentialsBinding("gcp-dirty-1", gardenerNamespace, map[string]string{
			gardener.HyperscalerTypeLabelKey: "gcp",
			gardener.DirtyLabelKey:           "true",
		})
		awsClean := fixCredentialsBinding("aws-clean", gardenerNamespace, map[string]string{
			gardener.HyperscalerTypeLabelKey: "aws",
		})

		collector := newGardenerCollector(t, awsDirty1, awsDirty2, gcpDirty, awsClean)
		collector.updateGardenerMetrics()

		assert.Equal(t, float64(2), dirtyGaugeValue(collector, "aws"), "2 dirty AWS bindings")
		assert.Equal(t, float64(1), dirtyGaugeValue(collector, "gcp"), "1 dirty GCP binding")
		assert.Equal(t, float64(0), dirtyGaugeValue(collector, "azure"), "no dirty Azure bindings")
	})
}

func TestSharedCredentialsBindingsCollector(t *testing.T) {
	t.Run("counts shared bindings per hyperscaler type", func(t *testing.T) {
		awsShared1 := fixCredentialsBinding("aws-shared-1", gardenerNamespace, map[string]string{
			gardener.HyperscalerTypeLabelKey: "aws",
			gardener.SharedLabelKey:          "true",
		})
		awsShared2 := fixCredentialsBinding("aws-shared-2", gardenerNamespace, map[string]string{
			gardener.HyperscalerTypeLabelKey: "aws",
			gardener.SharedLabelKey:          "true",
		})
		azureShared := fixCredentialsBinding("azure-shared-1", gardenerNamespace, map[string]string{
			gardener.HyperscalerTypeLabelKey: "azure",
			gardener.SharedLabelKey:          "true",
		})
		awsNotShared := fixCredentialsBinding("aws-not-shared", gardenerNamespace, map[string]string{
			gardener.HyperscalerTypeLabelKey: "aws",
		})

		collector := newGardenerCollector(t, awsShared1, awsShared2, azureShared, awsNotShared)
		collector.updateGardenerMetrics()

		assert.Equal(t, float64(2), sharedGaugeValue(collector, "aws"), "2 shared AWS bindings")
		assert.Equal(t, float64(1), sharedGaugeValue(collector, "azure"), "1 shared Azure binding")
		assert.Equal(t, float64(0), sharedGaugeValue(collector, "gcp"), "no shared GCP bindings")
	})
}

const gardenerNamespace = "test"

func newGardenerCollector(t *testing.T, objects ...runtime.Object) *CredentialsBindingsCollector {
	t.Helper()
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	gc := gardener.NewClient(gardener.NewDynamicFakeClient(objects...), gardenerNamespace)
	c := NewCredentialsBindingsCollector(nil, gc, 1*time.Minute, 5*time.Minute, log)
	t.Cleanup(func() {
		prometheus.Unregister(c.instancesPerCredentialsBinding)
		prometheus.Unregister(c.availableCredentialsBindings)
		prometheus.Unregister(c.claimedCredentialsBindings)
		prometheus.Unregister(c.dirtyCredentialsBindings)
		prometheus.Unregister(c.sharedCredentialsBindings)
	})
	return c
}

func poolGaugeValue(c *CredentialsBindingsCollector, hyperscalerType string) float64 {
	return testutil.ToFloat64(c.availableCredentialsBindings.With(prometheus.Labels{
		"hyperscaler_type": hyperscalerType,
	}))
}

func claimedGaugeValue(c *CredentialsBindingsCollector, hyperscalerType, tenantName string) float64 {
	return testutil.ToFloat64(c.claimedCredentialsBindings.With(prometheus.Labels{
		"hyperscaler_type": hyperscalerType,
		"tenant_name":      tenantName,
	}))
}

func dirtyGaugeValue(c *CredentialsBindingsCollector, hyperscalerType string) float64 {
	return testutil.ToFloat64(c.dirtyCredentialsBindings.With(prometheus.Labels{
		"hyperscaler_type": hyperscalerType,
	}))
}

func sharedGaugeValue(c *CredentialsBindingsCollector, hyperscalerType string) float64 {
	return testutil.ToFloat64(c.sharedCredentialsBindings.With(prometheus.Labels{
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
