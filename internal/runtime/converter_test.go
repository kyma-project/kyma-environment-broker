package runtime

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"

	"github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConverting_Provisioning(t *testing.T) {
	// given
	instance := fixInstance()
	svc := NewConverter("eu")

	// when
	dto, _ := svc.NewDTO(instance)
	svc.ApplyProvisioningOperation(&dto, fixProvisioningOperation(domain.InProgress, time.Now()))

	// then
	assert.Equal(t, runtime.StateProvisioning, dto.Status.State)
}

func TestConverting_Provisioned(t *testing.T) {
	// given
	instance := fixInstance()
	svc := NewConverter("eu")

	// when
	dto, _ := svc.NewDTO(instance)
	svc.ApplyProvisioningOperation(&dto, fixProvisioningOperation(domain.Succeeded, time.Now()))

	// then
	assert.Equal(t, runtime.StateSucceeded, dto.Status.State)
}

func TestConverting_ProvisioningFailed(t *testing.T) {
	// given
	instance := fixInstance()
	svc := NewConverter("eu")

	// when
	dto, _ := svc.NewDTO(instance)
	svc.ApplyProvisioningOperation(&dto, fixProvisioningOperation(domain.Failed, time.Now()))

	// then
	assert.Equal(t, runtime.StateFailed, dto.Status.State)
}

func TestConverting_Updating(t *testing.T) {
	// given
	instance := fixInstance()
	svc := NewConverter("eu")

	// when
	dto, _ := svc.NewDTO(instance)
	svc.ApplyProvisioningOperation(&dto, fixProvisioningOperation(domain.Succeeded, time.Now()))
	svc.ApplyUpdateOperations(&dto, []internal.UpdatingOperation{{
		Operation: internal.Operation{
			CreatedAt:     time.Now().Add(time.Second),
			ID:            "prov-id",
			State:         domain.InProgress,
			UpdatedPlanID: broker.BuildRuntimeAWSPlanID,
		},
	}}, 1)

	// then
	assert.Equal(t, runtime.StateUpdating, dto.Status.State)
	assert.Equal(t, broker.BuildRuntimeAWSPlanName, dto.Status.Update.Data[0].UpdatedPlanName)
}

func TestConverting_UpdateFailed(t *testing.T) {
	// given
	instance := fixInstance()
	svc := NewConverter("eu")

	// when
	dto, _ := svc.NewDTO(instance)
	svc.ApplyProvisioningOperation(&dto, fixProvisioningOperation(domain.Succeeded, time.Now()))
	svc.ApplyUpdateOperations(&dto, []internal.UpdatingOperation{{
		Operation: internal.Operation{
			CreatedAt: time.Now().Add(time.Second),
			ID:        "prov-id",
			State:     domain.Failed,
		},
	}}, 1)

	// then
	assert.Equal(t, runtime.StateError, dto.Status.State)
}

func TestConverting_Suspending(t *testing.T) {
	t.Run("last operation should be deprovisioning", func(t *testing.T) {
		// given
		instance := fixInstance()
		svc := NewConverter("eu")

		// when
		dto, _ := svc.NewDTO(instance)
		svc.ApplyProvisioningOperation(&dto, fixProvisioningOperation(domain.Succeeded, time.Now()))
		svc.ApplySuspensionOperations(&dto, fixSuspensionOperation(domain.InProgress, time.Now().Add(time.Second)))

		// then
		assert.Equal(t, runtime.StateDeprovisioning, dto.Status.State)
	})

	t.Run("last operation should not be deprovisioning when it is pending", func(t *testing.T) {
		// given
		instance := fixInstance()
		svc := NewConverter("eu")

		// when
		dto, _ := svc.NewDTO(instance)
		svc.ApplyProvisioningOperation(&dto, fixProvisioningOperation(domain.InProgress, time.Now()))
		svc.ApplySuspensionOperations(&dto, fixSuspensionOperation(internal.OperationStatePending, time.Now().Add(time.Second)))

		// then
		assert.Equal(t, runtime.StateProvisioning, dto.Status.State)
	})
}

func TestConverting_Deprovisioning(t *testing.T) {
	t.Run("last operation should be deprovisioning", func(t *testing.T) {
		// given
		instance := fixInstance()
		svc := NewConverter("eu")

		// when
		dto, _ := svc.NewDTO(instance)
		svc.ApplyProvisioningOperation(&dto, fixProvisioningOperation(domain.Succeeded, time.Now()))
		svc.ApplyDeprovisioningOperation(&dto, fixDeprovisionOperation(domain.InProgress, time.Now().Add(time.Second)))

		// then
		assert.Equal(t, runtime.StateDeprovisioning, dto.Status.State)
	})

	t.Run("last operation should not be deprovisioning when it is pending", func(t *testing.T) {
		// given
		instance := fixInstance()
		svc := NewConverter("eu")

		// when
		dto, _ := svc.NewDTO(instance)
		svc.ApplyProvisioningOperation(&dto, fixProvisioningOperation(domain.InProgress, time.Now()))
		svc.ApplyDeprovisioningOperation(&dto, fixDeprovisionOperation(internal.OperationStatePending, time.Now().Add(time.Second)))

		// then
		assert.Equal(t, runtime.StateProvisioning, dto.Status.State)
	})
}

func TestConverting_DeprovisionFailed(t *testing.T) {
	// given
	instance := fixInstance()
	svc := NewConverter("eu")

	// when
	dto, _ := svc.NewDTO(instance)
	svc.ApplyProvisioningOperation(&dto, fixProvisioningOperation(domain.Succeeded, time.Now()))
	svc.ApplyDeprovisioningOperation(&dto, fixDeprovisionOperation(domain.Failed, time.Now().Add(time.Second)))

	// then
	assert.Equal(t, runtime.StateFailed, dto.Status.State)
}

func TestConverting_SuspendFailed(t *testing.T) {
	// given
	instance := fixInstance()
	svc := NewConverter("eu")

	// when
	dto, _ := svc.NewDTO(instance)
	svc.ApplyProvisioningOperation(&dto, fixProvisioningOperation(domain.Succeeded, time.Now()))
	svc.ApplySuspensionOperations(&dto, fixSuspensionOperation(domain.Failed, time.Now().Add(time.Second)))

	// then
	assert.Equal(t, runtime.StateFailed, dto.Status.State)
}

func TestConverting_SuspendedAndUpdated(t *testing.T) {
	// given
	instance := fixInstance()
	svc := NewConverter("eu")

	// when
	dto, _ := svc.NewDTO(instance)
	svc.ApplyProvisioningOperation(&dto, fixProvisioningOperation(domain.Succeeded, time.Now()))
	svc.ApplySuspensionOperations(&dto, fixSuspensionOperation(domain.Succeeded, time.Now().Add(time.Second)))
	svc.ApplyUpdateOperations(&dto, []internal.UpdatingOperation{{
		Operation: internal.Operation{
			CreatedAt: time.Now().Add(2 * time.Second),
			ID:        "prov-id",
			State:     domain.Succeeded,
		},
	}}, 1)

	// then
	assert.Equal(t, runtime.StateSuspended, dto.Status.State)
}

func TestConverting_SuspendedAndUpdateFAiled(t *testing.T) {
	// given
	instance := fixInstance()
	svc := NewConverter("eu")

	// when
	dto, _ := svc.NewDTO(instance)
	svc.ApplyProvisioningOperation(&dto, fixProvisioningOperation(domain.Succeeded, time.Now()))
	svc.ApplySuspensionOperations(&dto, fixSuspensionOperation(domain.Succeeded, time.Now().Add(time.Second)))
	svc.ApplyUpdateOperations(&dto, []internal.UpdatingOperation{{
		Operation: internal.Operation{
			CreatedAt: time.Now().Add(2 * time.Second),
			ID:        "prov-id",
			State:     domain.Failed,
		},
	}}, 1)

	// then
	assert.Equal(t, runtime.StateSuspended, dto.Status.State)
}

func TestConverting_ProvisioningOperationConverter(t *testing.T) {
	// given
	instance := fixInstance()
	svc := NewConverter("eu")

	// when
	dto, _ := svc.NewDTO(instance)

	//expected stages in order
	expected := []string{"start", "create_runtime", "check_kyma", "post_actions"}

	t.Run("finished orders should be not set", func(t *testing.T) {
		svc.ApplyProvisioningOperation(&dto, fixProvisioningOperation(domain.Succeeded, time.Now()))

		// then
		assert.Equal(t, []string(nil), dto.Status.Provisioning.FinishedStages)
	})

	t.Run("finished orders should be set in order", func(t *testing.T) {
		svc.ApplyProvisioningOperation(&dto, fixProvisioningOperationWithStagesAndVersion(domain.Succeeded, time.Now()))

		// then
		assert.True(t, reflect.DeepEqual(expected, dto.Status.Provisioning.FinishedStages))
	})
}

func TestConverting_ProvisioningParams(t *testing.T) {
	// given
	instance := fixInstance()
	svc := NewConverter("eu")

	// when
	dto, _ := svc.NewDTO(instance)

	// then
	assert.Equal(t, instance.Parameters.Parameters, dto.Parameters)
}

func fixSuspensionOperation(state domain.LastOperationState, createdAt time.Time) []internal.DeprovisioningOperation {
	return []internal.DeprovisioningOperation{{
		Operation: internal.Operation{
			CreatedAt: createdAt,
			ID:        "s-id",
			State:     state,
			Temporary: true,
		},
	}}
}

func fixDeprovisionOperation(state domain.LastOperationState, createdAt time.Time) *internal.DeprovisioningOperation {
	return &internal.DeprovisioningOperation{
		Operation: internal.Operation{
			CreatedAt: createdAt,
			ID:        "s-id",
			State:     state,
		},
	}
}

func fixInstance() internal.Instance {
	return internal.Instance{
		InstanceID:                  "instance-id",
		RuntimeID:                   "runtime-id",
		GlobalAccountID:             "global-account-id",
		SubscriptionGlobalAccountID: "subgid",
		SubAccountID:                "sub-account-id",
		Parameters: internal.ProvisioningParameters{
			Parameters: runtime.ProvisioningParametersDTO{
				Name: "instance-name",
			},
		},
	}
}

func fixProvisioningOperation(state domain.LastOperationState, createdAt time.Time) *internal.ProvisioningOperation {
	return &internal.ProvisioningOperation{
		Operation: internal.Operation{
			CreatedAt: createdAt,
			ID:        "prov-id",
			State:     state,
		},
	}
}

func fixProvisioningOperationWithStagesAndVersion(state domain.LastOperationState, createdAt time.Time) *internal.ProvisioningOperation {
	return &internal.ProvisioningOperation{
		Operation: internal.Operation{
			CreatedAt:      createdAt,
			ID:             "prov-id",
			State:          state,
			FinishedStages: []string{"start", "create_runtime", "check_kyma", "post_actions"},
		},
	}
}

func TestApplyOperation_RawParametersUsedWhenSet(t *testing.T) {
	// given
	instance := fixInstance()
	svc := NewConverter("eu")
	dto, _ := svc.NewDTO(instance)

	rawParams := json.RawMessage(`{"autoScalerMax":21}`)
	op := &internal.ProvisioningOperation{
		Operation: internal.Operation{
			CreatedAt:     time.Now(),
			ID:            "op-id",
			State:         domain.Succeeded,
			RawParameters: rawParams,
			ProvisioningParameters: internal.ProvisioningParameters{
				Parameters: runtime.ProvisioningParametersDTO{
					Name: "should-not-appear",
				},
			},
		},
	}

	// when
	svc.ApplyProvisioningOperation(&dto, op)

	// then
	require.NotNil(t, dto.Status.Provisioning)
	assert.Equal(t, json.RawMessage(`{"autoScalerMax":21}`), dto.Status.Provisioning.RawParameters)
	assert.Equal(t, "", dto.Status.Provisioning.Parameters.Name, "typed Parameters should reflect raw payload, not merged state")
}

func TestApplyOperation_FallbackToMergedStateWhenRawParametersEmpty(t *testing.T) {
	// given
	instance := fixInstance()
	svc := NewConverter("eu")
	dto, _ := svc.NewDTO(instance)

	op := &internal.ProvisioningOperation{
		Operation: internal.Operation{
			CreatedAt: time.Now(),
			ID:        "op-id",
			State:     domain.Succeeded,
			ProvisioningParameters: internal.ProvisioningParameters{
				Parameters: runtime.ProvisioningParametersDTO{
					Name: "from-merged-state",
				},
			},
		},
	}

	// when
	svc.ApplyProvisioningOperation(&dto, op)

	// then
	require.NotNil(t, dto.Status.Provisioning)
	assert.Nil(t, dto.Status.Provisioning.RawParameters)
	assert.Equal(t, "from-merged-state", dto.Status.Provisioning.Parameters.Name)
}

func TestApplyOperation_SensitiveFieldsStrippedFromRawParameters(t *testing.T) {
	// given
	instance := fixInstance()
	svc := NewConverter("eu")
	dto, _ := svc.NewDTO(instance)

	rawParams := json.RawMessage(`{"name":"test","kubeconfig":"secret","targetSecret":"also-secret"}`)
	op := &internal.ProvisioningOperation{
		Operation: internal.Operation{
			CreatedAt:     time.Now(),
			ID:            "op-id",
			State:         domain.Succeeded,
			RawParameters: rawParams,
		},
	}

	// when
	svc.ApplyProvisioningOperation(&dto, op)

	// then
	require.NotNil(t, dto.Status.Provisioning)
	var got map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(dto.Status.Provisioning.RawParameters, &got))
	assert.Equal(t, json.RawMessage(`"test"`), got["name"])
	assert.NotContains(t, got, "kubeconfig")
	assert.NotContains(t, got, "targetSecret")
}

func TestApplyOperation_UpdateRawParametersShowsOnlySubmittedFields(t *testing.T) {
	// given
	instance := fixInstance()
	svc := NewConverter("eu")
	dto, _ := svc.NewDTO(instance)
	svc.ApplyProvisioningOperation(&dto, fixProvisioningOperation(domain.Succeeded, time.Now()))

	rawUpdateParams := json.RawMessage(`{"autoScalerMax":21}`)
	updateOp := internal.UpdatingOperation{
		Operation: internal.Operation{
			CreatedAt:     time.Now().Add(time.Second),
			ID:            "upd-id",
			State:         domain.Succeeded,
			RawParameters: rawUpdateParams,
			ProvisioningParameters: internal.ProvisioningParameters{
				Parameters: runtime.ProvisioningParametersDTO{
					// merged state contains more fields than submitted
					Name: "merged-name",
				},
			},
		},
	}

	// when
	svc.ApplyUpdateOperations(&dto, []internal.UpdatingOperation{updateOp}, 1)

	// then
	require.Len(t, dto.Status.Update.Data, 1)
	got := dto.Status.Update.Data[0]
	assert.Equal(t, json.RawMessage(`{"autoScalerMax":21}`), got.RawParameters)
	assert.Equal(t, "", got.Parameters.Name, "Parameters should reflect raw payload, not merged state")
}

func TestSanitizeRawParams(t *testing.T) {
	t.Run("removes kubeconfig and targetSecret", func(t *testing.T) {
		raw := json.RawMessage(`{"name":"x","kubeconfig":"k","targetSecret":"s","region":"eu"}`)
		got := sanitizeRawParams(raw)
		var m map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(got, &m))
		assert.NotContains(t, m, "kubeconfig")
		assert.NotContains(t, m, "targetSecret")
		assert.Contains(t, m, "name")
		assert.Contains(t, m, "region")
	})

	t.Run("returns original on invalid JSON", func(t *testing.T) {
		raw := json.RawMessage(`not-json`)
		got := sanitizeRawParams(raw)
		assert.Equal(t, raw, got)
	})

	t.Run("handles empty input", func(t *testing.T) {
		raw := json.RawMessage(`{}`)
		got := sanitizeRawParams(raw)
		assert.Equal(t, json.RawMessage(`{}`), got)
	})
}
