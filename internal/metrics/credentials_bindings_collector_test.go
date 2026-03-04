package metrics

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	collector := NewCredentialsBindingsCollector(instances, 1*time.Minute, log)

	t.Run("initial counts are correct after first updateMetrics", func(t *testing.T) {
		collector.updateMetrics()

		assert.Equal(t, float64(2), gaugeValue(collector, bindingAzure, ga1), "GA1/azure: 2 active instances")
		assert.Equal(t, float64(1), gaugeValue(collector, bindingAzure2, ga1), "GA1/azure-2: 1 active instance")
		assert.Equal(t, float64(3), gaugeValue(collector, bindingAWS, ga2), "GA2/aws: 3 active instances")
	})

	t.Run("deleted instance is not counted", func(t *testing.T) {
		assert.Equal(t, float64(2), gaugeValue(collector, bindingAzure, ga1))
	})

	t.Run("counts are updated after new instance is added", func(t *testing.T) {
		require.NoError(t, instances.Insert(fixCredentialInstance("i-9", ga2, bindingAWS)))

		collector.updateMetrics()

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
