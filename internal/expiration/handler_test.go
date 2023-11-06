package expiration_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/kyma-project/kyma-environment-broker/internal/expiration"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const requestPathFormat = "/expire/service_instance/%s"

func TestExpiration(t *testing.T) {
	router := mux.NewRouter()
	deprovisioningQueue := process.NewFakeQueue()
	storage := storage.NewMemoryStorage()
	logger := logrus.New()
	handler := expiration.NewHandler(storage.Instances(), storage.Operations(), deprovisioningQueue, logger)
	handler.AttachRoutes(router)

	t.Run("should receive 404 Not Found response", func(t *testing.T) {
		// given
		instanceID := "inst-404-not-found"
		reqPath := fmt.Sprintf(requestPathFormat, instanceID)
		req := httptest.NewRequest("PUT", reqPath, nil)
		w := httptest.NewRecorder()

		// when
		router.ServeHTTP(w, req)
		resp := w.Result()

		// then
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("should receive 400 Bad request response when instance is not trial", func(t *testing.T) {
		// given
		instanceID := "inst-azure-01"
		azureInstance := fixture.FixInstance(instanceID)
		err := storage.Instances().Insert(azureInstance)
		require.NoError(t, err)

		reqPath := fmt.Sprintf(requestPathFormat, instanceID)
		req := httptest.NewRequest("PUT", reqPath, nil)
		w := httptest.NewRecorder()

		// when
		router.ServeHTTP(w, req)
		resp := w.Result()

		// then
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		// when
		actualInstance, err := storage.Instances().GetByID(instanceID)
		require.NoError(t, err)

		// then
		assert.True(t, *actualInstance.Parameters.ErsContext.Active)
		assert.Nil(t, actualInstance.ExpiredAt)
	})
}
