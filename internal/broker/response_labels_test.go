package broker

import (
	"fmt"
	"testing"
	"time"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/kubeconfig"
	"github.com/kyma-project/kyma-environment-broker/internal/kubeconfig/automock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	runtimeID = "f7d634ae-4ce2-4916-be64-b6fb493155df"
	serverURL = "https://api.ac0d8d9.kyma-dev.shoot.canary.k8s-hana.ondemand.com"
)

func TestResponseLabels(t *testing.T) {
	t.Run("return all labels", func(t *testing.T) {
		// given
		instance := internal.Instance{
			InstanceID:   "inst1234",
			DashboardURL: "https://console.dashbord.test",
			RuntimeID:    runtimeID,
			Parameters: internal.ProvisioningParameters{
				Parameters: pkg.ProvisioningParametersDTO{
					Name: "test",
				},
			},
		}
		kcBuilder := &automock.KcBuilder{}
		kcBuilder.On("GetServerURL", instance.RuntimeID).Return(serverURL, nil)

		// when
		labels := ResponseLabels(instance, "https://example.com", kcBuilder)

		// then
		require.Len(t, labels, 3)
		require.Equal(t, "test", labels["Name"])
		require.Equal(t, "https://example.com/kubeconfig/inst1234", labels["KubeconfigURL"])
		require.Equal(t, "https://api.ac0d8d9.kyma-dev.shoot.canary.k8s-hana.ondemand.com", labels["APIServerURL"])
	})

	t.Run("disable kubeconfig URL label", func(t *testing.T) {
		// given
		instance := internal.Instance{
			Parameters: internal.ProvisioningParameters{
				Parameters: pkg.ProvisioningParametersDTO{
					Name: "test",
				},
			},
		}
		kcBuilder := &automock.KcBuilder{}

		// when
		labels := ResponseLabels(instance, "https://console.example.com", kcBuilder)

		// then
		require.Len(t, labels, 1)
		require.Equal(t, "test", labels["Name"])
	})

	t.Run("should return labels with expire info for not expired instance", func(t *testing.T) {
		// given
		instance := fixture.FixInstance("instanceID")
		kcBuilder := &automock.KcBuilder{}
		kcBuilder.On("GetServerURL", instance.RuntimeID).Return(serverURL, nil)
		defer kcBuilder.AssertExpectations(t)

		// when
		labels := ResponseLabelsWithExpirationInfo(instance, "https://example.com", "https://trial.docs.local", trialDocsKey, trialExpireDuration, trialExpiryDetailsKey, trialExpiredInfoFormat, kcBuilder)

		// then
		require.Len(t, labels, 4)
		assert.Contains(t, labels, trialExpiryDetailsKey)
		require.Equal(t, "cluster-test", labels["Name"])
		require.Equal(t, "https://example.com/kubeconfig/instanceID", labels["KubeconfigURL"])
		require.Equal(t, serverURL, labels["APIServerURL"])
	})

	t.Run("should return labels with expire info for instance soon to be expired", func(t *testing.T) {
		// given
		instance := fixture.FixInstance("instanceID")
		instance.CreatedAt = time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
		kcBuilder := &automock.KcBuilder{}
		kcBuilder.On("GetServerURL", instance.RuntimeID).Return(serverURL, nil)

		expectedMsg := fmt.Sprintf(notExpiredInfoFormat, "today")

		// when
		labels := ResponseLabelsWithExpirationInfo(instance, "https://example.com", "https://trial.docs.local", trialDocsKey, trialExpireDuration, trialExpiryDetailsKey, trialExpiredInfoFormat, kcBuilder)

		// then
		require.Len(t, labels, 4)
		assert.Contains(t, labels, trialExpiryDetailsKey)
		assert.Contains(t, labels, kubeconfigURLKey)
		assert.Contains(t, labels, apiServerURLKey)
		require.Equal(t, "cluster-test", labels["Name"])
		assert.Equal(t, expectedMsg, labels[trialExpiryDetailsKey])
		assert.Equal(t, serverURL, labels["APIServerURL"])
	})

	t.Run("should return labels with expire info for expired instance", func(t *testing.T) {
		// given
		instance := fixture.FixInstance("instanceID")
		instance.CreatedAt = time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
		expiryDate := time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC)
		instance.ExpiredAt = &expiryDate
		kcBuilder := &automock.KcBuilder{}
		kcBuilder.On("GetServerURL", instance.RuntimeID).Return(serverURL, nil)

		// when
		labels := ResponseLabelsWithExpirationInfo(instance, "https://example.com", "https://trial.docs.local", trialDocsKey, trialExpireDuration, trialExpiryDetailsKey, trialExpiredInfoFormat, kcBuilder)

		// then
		require.Len(t, labels, 3)
		assert.Contains(t, labels, trialExpiryDetailsKey)
		assert.Contains(t, labels, trialDocsKey)
		assert.NotContains(t, labels, kubeconfigURLKey)
		assert.NotContains(t, labels, apiServerURLKey)
		require.Equal(t, "cluster-test", labels["Name"])
	})

	t.Run("should return labels for own cluster", func(t *testing.T) {
		// given
		instance := fixture.FixInstance("instanceID")
		instance.ServicePlanID = OwnClusterPlanID
		kcBuilder := &automock.KcBuilder{}

		// when
		labels := ResponseLabelsWithExpirationInfo(instance, "https://example.com", "https://trial.docs.local", trialDocsKey, trialExpireDuration, trialExpiryDetailsKey, trialExpiredInfoFormat, kcBuilder)

		// then
		require.Len(t, labels, 2)
		assert.NotContains(t, labels, kubeconfigURLKey)
		assert.NotContains(t, labels, apiServerURLKey)
		require.Equal(t, "cluster-test", labels["Name"])
	})

	t.Run("should not return API server URL label", func(t *testing.T) {
		// given
		instance := fixture.FixInstance("instance-no-apiserverurl")

		kcBuilder := &automock.KcBuilder{}
		kcBuilder.On("GetServerURL", instance.RuntimeID).Return("", kubeconfig.NewNotFoundError("missing secret"))

		// when
		labels := ResponseLabels(instance, "https://example.com", kcBuilder)

		// then
		require.Len(t, labels, 2)
		assert.NotContains(t, labels, apiServerURLKey)
		require.Equal(t, "cluster-test", labels["Name"])
		require.Equal(t, fmt.Sprintf("https://example.com/kubeconfig/%s", instance.InstanceID), labels["KubeconfigURL"])
	})
}
