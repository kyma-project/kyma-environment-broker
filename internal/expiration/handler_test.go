package expiration_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/kyma-project/kyma-environment-broker/internal/expiration"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
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
}
