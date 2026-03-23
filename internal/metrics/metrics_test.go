package metrics

import (
	"testing"

	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/stretchr/testify/assert"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
)

func TestGetLabels(t *testing.T) {
	t.Run("returns all expected label keys", func(t *testing.T) {
		op := internal.Operation{
			ID:         "op-1",
			InstanceID: "inst-1",
			InstanceDetails: internal.InstanceDetails{
				RuntimeID: "runtime-1",
				ShootName: "shoot-1",
			},
			RuntimeOperation: internal.RuntimeOperation{
				GlobalAccountID: "ga-1"},
			ProvisioningParameters: internal.ProvisioningParameters{
				PlanID: broker.AWSPlanID,
			},
			Type:  internal.OperationTypeProvision,
			State: domain.Succeeded,
		}

		labels := GetLabels(op)

		assert.Equal(t, "op-1", labels["operation_id"])
		assert.Equal(t, "inst-1", labels["instance_id"])
		assert.Equal(t, "runtime-1", labels["runtime_id"])
		assert.Equal(t, "shoot-1", labels["shoot_name"])
		assert.Equal(t, "ga-1", labels["global_account_id"])
		assert.Equal(t, broker.AWSPlanID, labels["plan_id"])
		assert.Equal(t, string(internal.OperationTypeProvision), labels["type"])
		assert.Equal(t, string(domain.Succeeded), labels["state"])
		assert.Contains(t, labels, "error_category")
		assert.Contains(t, labels, "error_reason")
		assert.Contains(t, labels, "error")
	})

	t.Run("returns empty strings for zero-value fields", func(t *testing.T) {
		op := internal.Operation{}

		labels := GetLabels(op)

		assert.Equal(t, "", labels["operation_id"])
		assert.Equal(t, "", labels["instance_id"])
		assert.Equal(t, "", labels["runtime_id"])
		assert.Equal(t, "", labels["shoot_name"])
		assert.Equal(t, "", labels["global_account_id"])
		assert.Equal(t, "", labels["plan_id"])
		assert.Equal(t, "", labels["error_category"])
		assert.Equal(t, "", labels["error_reason"])
		assert.Equal(t, "", labels["error"])
	})

	t.Run("returns map with exactly 11 keys", func(t *testing.T) {
		op := internal.Operation{}

		labels := GetLabels(op)

		assert.Len(t, labels, 11)
	})
}
