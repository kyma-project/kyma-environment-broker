package broker

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	mocks "github.com/kyma-project/kyma-environment-broker/internal/storage/automock"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/pivotal-cf/brokerapi/v8/domain/apiresponses"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestGetBinding(t *testing.T) {

	t.Run("should return 404 code for the expired binding", func(t *testing.T) {
		// given
		mockBindings := new(mocks.Bindings)

		expiredBinding := &internal.Binding{
			ExpiresAt: time.Now().Add(-1 * time.Hour),
		}

		mockBindings.On("Get", "test-instance-id", "test-binding-id").Return(expiredBinding, nil)

		endpoint := &GetBindingEndpoint{
			bindings: mockBindings,
			log:      &logrus.Logger{}, // Assuming you have a mock logger
		}

		// when
		_, err := endpoint.GetBinding(context.Background(), "test-instance-id", "test-binding-id", domain.FetchBindingDetails{})

		// then
		require.NotNil(t, err)
		apiErr, ok := err.(*apiresponses.FailureResponse)
		require.True(t, ok)
		require.Equal(t, http.StatusNotFound, apiErr.ValidatedStatusCode(nil))

		errorResponse := apiErr.ErrorResponse().(apiresponses.ErrorResponse)
		require.Equal(t, "Binding expired", errorResponse.Description)
		mockBindings.AssertExpectations(t)
	})
}
