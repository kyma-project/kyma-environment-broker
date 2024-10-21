package broker

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/driver/memory"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/pivotal-cf/brokerapi/v8/domain/apiresponses"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestGetBinding(t *testing.T) {

	t.Run("should return 404 code for the expired binding", func(t *testing.T) {
		// given
		bindingsMemory := memory.NewBinding()

		expiredBinding := &internal.Binding{
			ID:         "test-binding-id",
			InstanceID: "test-instance-id",
			ExpiresAt:  time.Now().Add(-1 * time.Hour),
		}
		err := bindingsMemory.Insert(expiredBinding)
		require.NoError(t, err)

		endpoint := &GetBindingEndpoint{
			bindings: bindingsMemory,
			log:      &logrus.Logger{},
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
}
