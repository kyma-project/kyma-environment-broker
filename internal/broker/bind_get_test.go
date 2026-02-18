package broker

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/fixture"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/driver/memory"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/pivotal-cf/brokerapi/v12/domain/apiresponses"
	"github.com/stretchr/testify/require"
)

func TestGetBinding(t *testing.T) {

	t.Run("should return 404 code for the expired binding", func(t *testing.T) {
		// given
		bindingsMemory := memory.NewBinding()
		operationsMemory := memory.NewOperation()

		operation := fixture.FixOperation("operation-001", "test-instance-id", internal.OperationTypeProvision)
		err := operationsMemory.InsertOperation(operation)
		require.NoError(t, err)

		expiredBinding := &internal.Binding{
			ID:         "test-binding-id",
			InstanceID: "test-instance-id",
			ExpiresAt:  time.Now().Add(-1 * time.Hour),
		}
		err = bindingsMemory.Insert(expiredBinding)
		require.NoError(t, err)

		endpoint := &GetBindingEndpoint{
			bindings:   bindingsMemory,
			operations: operationsMemory,
			log:        fixLogger(),
		}

		// when
		_, err = endpoint.GetBinding(context.Background(), "test-instance-id", "test-binding-id", domain.FetchBindingDetails{})

		// then
		require.NotNil(t, err)
		apiErr, ok := err.(*apiresponses.FailureResponse)
		require.True(t, ok)
		require.Equal(t, http.StatusNotFound, apiErr.ValidatedStatusCode(nil))

		errorResponse := apiErr.ErrorResponse().(apiresponses.ErrorResponse)
		require.Equal(t, "Binding expired", errorResponse.Description)
	})

	t.Run("should return 404 code for binding in progress", func(t *testing.T) {
		// given
		bindingsMemory := memory.NewBinding()
		operationsMemory := memory.NewOperation()

		operation := fixture.FixOperation("operation-001", "test-instance-id", internal.OperationTypeProvision)
		err := operationsMemory.InsertOperation(operation)
		require.NoError(t, err)

		expiredBinding := &internal.Binding{
			ID:         "test-binding-id",
			InstanceID: "test-instance-id",
			ExpiresAt:  time.Now().Add(5 * time.Minute),
		}
		err = bindingsMemory.Insert(expiredBinding)
		require.NoError(t, err)

		endpoint := &GetBindingEndpoint{
			bindings:   bindingsMemory,
			operations: operationsMemory,
			log:        fixLogger(),
		}

		// when
		_, err = endpoint.GetBinding(context.Background(), "test-instance-id", "test-binding-id", domain.FetchBindingDetails{})

		// then
		require.NotNil(t, err)
		apiErr, ok := err.(*apiresponses.FailureResponse)
		require.True(t, ok)
		require.Equal(t, http.StatusNotFound, apiErr.ValidatedStatusCode(nil))

		errorResponse := apiErr.ErrorResponse().(apiresponses.ErrorResponse)
		require.Equal(t, "Binding creation in progress", errorResponse.Description)
	})

	t.Run("should return 404 code for the binding of deprovisioned/suspended instance", func(t *testing.T) {
		// given
		bindingsMemory := memory.NewBinding()
		operationsMemory := memory.NewOperation()

		operation := fixture.FixOperation("operation-001", "test-instance-id", internal.OperationTypeDeprovision)
		err := operationsMemory.InsertOperation(operation)
		require.NoError(t, err)

		binding := &internal.Binding{
			ID:         "test-binding-id",
			InstanceID: "test-instance-id",
			ExpiresAt:  time.Now().Add(-1 * time.Hour),
		}
		err = bindingsMemory.Insert(binding)
		require.NoError(t, err)

		endpoint := &GetBindingEndpoint{
			bindings:   bindingsMemory,
			operations: operationsMemory,
			log:        fixLogger(),
		}

		// when
		_, err = endpoint.GetBinding(context.Background(), "test-instance-id", "test-binding-id", domain.FetchBindingDetails{})

		// then
		require.NotNil(t, err)
		apiErr, ok := err.(*apiresponses.FailureResponse)
		require.True(t, ok)
		require.Equal(t, http.StatusNotFound, apiErr.ValidatedStatusCode(nil))

		errorResponse := apiErr.ErrorResponse().(apiresponses.ErrorResponse)
		require.Equal(t, "Binding not found", errorResponse.Description)
	})
}
