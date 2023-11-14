package expiration_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/expiration"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v8/domain"
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

	t.Run("should expire and suspend the instance", func(t *testing.T) {
		// given
		instanceID := "inst-trial-01"
		trialInstance := fixture.FixInstance(instanceID)
		trialInstance.ServicePlanID = broker.TrialPlanID
		trialInstance.ServicePlanName = broker.TrialPlanName
		err := storage.Instances().Insert(trialInstance)
		require.NoError(t, err)

		reqPath := fmt.Sprintf(requestPathFormat, instanceID)
		req := httptest.NewRequest("PUT", reqPath, nil)
		w := httptest.NewRecorder()

		// when
		router.ServeHTTP(w, req)
		resp := w.Result()

		// then
		assert.Equal(t, http.StatusAccepted, resp.StatusCode)

		// when
		actualInstance, err := storage.Instances().GetByID(instanceID)
		require.NoError(t, err)

		// then
		assert.False(t, *actualInstance.Parameters.ErsContext.Active)
		assert.NotNil(t, actualInstance.ExpiredAt)
	})

	t.Run("should repeat suspension on previously expired instance", func(t *testing.T) {
		// given
		instanceID := "inst-trial-02"
		trialInstance := fixture.FixInstance(instanceID)
		trialInstance.ServicePlanID = broker.TrialPlanID
		trialInstance.ServicePlanName = broker.TrialPlanName
		expectedExpirationTime := time.Now()
		trialInstance.ExpiredAt = &expectedExpirationTime
		expectedActiveValue := false
		trialInstance.Parameters.ErsContext.Active = &expectedActiveValue
		err := storage.Instances().Insert(trialInstance)
		require.NoError(t, err)

		reqPath := fmt.Sprintf(requestPathFormat, instanceID)
		req := httptest.NewRequest("PUT", reqPath, nil)
		w := httptest.NewRecorder()

		// when
		router.ServeHTTP(w, req)
		resp := w.Result()

		// then
		assert.Equal(t, http.StatusAccepted, resp.StatusCode)

		// when
		actualInstance, err := storage.Instances().GetByID(instanceID)
		require.NoError(t, err)

		// then
		assert.False(t, *actualInstance.Parameters.ErsContext.Active)
		assert.Equal(t, expectedExpirationTime, *actualInstance.ExpiredAt)
	})

	t.Run("should expire and suspend the instance on previously failed deprovisioning", func(t *testing.T) {
		// given
		instanceID := "inst-trial-03"
		trialInstance := fixture.FixInstance(instanceID)
		trialInstance.ServicePlanID = broker.TrialPlanID
		trialInstance.ServicePlanName = broker.TrialPlanName
		err := storage.Instances().Insert(trialInstance)
		require.NoError(t, err)

		deprovisioningOpID := "inst-trial-03-failed-deprovisioning"
		deprovisioningOp := fixture.FixDeprovisioningOperation(deprovisioningOpID, instanceID)
		deprovisioningOp.State = domain.Failed
		err = storage.Operations().InsertDeprovisioningOperation(deprovisioningOp)
		require.NoError(t, err)

		reqPath := fmt.Sprintf(requestPathFormat, instanceID)
		req := httptest.NewRequest("PUT", reqPath, nil)
		w := httptest.NewRecorder()

		// when
		router.ServeHTTP(w, req)
		resp := w.Result()

		// then
		assert.Equal(t, http.StatusAccepted, resp.StatusCode)

		// when
		actualInstance, err := storage.Instances().GetByID(instanceID)
		require.NoError(t, err)

		// then
		assert.False(t, *actualInstance.Parameters.ErsContext.Active)
		assert.NotNil(t, actualInstance.ExpiredAt)
	})

	t.Run("should retry expiration on in progress suspension", func(t *testing.T) {
		// given
		instanceID := "inst-trial-04"
		trialInstance := fixture.FixInstance(instanceID)
		trialInstance.ServicePlanID = broker.TrialPlanID
		trialInstance.ServicePlanName = broker.TrialPlanName
		err := storage.Instances().Insert(trialInstance)
		require.NoError(t, err)

		deprovisioningOpID := "inst-trial-04-suspension-in-progress"
		deprovisioningOp := fixture.FixDeprovisioningOperation(deprovisioningOpID, instanceID)
		deprovisioningOp.Temporary = true
		deprovisioningOp.State = domain.InProgress
		err = storage.Operations().InsertDeprovisioningOperation(deprovisioningOp)
		require.NoError(t, err)

		reqPath := fmt.Sprintf(requestPathFormat, instanceID)
		req := httptest.NewRequest("PUT", reqPath, nil)
		w := httptest.NewRecorder()

		// when
		router.ServeHTTP(w, req)
		resp := w.Result()

		// then
		assert.Equal(t, http.StatusAccepted, resp.StatusCode)

		// when
		actualInstance, err := storage.Instances().GetByID(instanceID)
		require.NoError(t, err)

		// then
		assert.False(t, *actualInstance.Parameters.ErsContext.Active)
		assert.NotNil(t, actualInstance.ExpiredAt)

		// when
		actualLastOp, err := storage.Operations().GetLastOperation(instanceID)
		require.NoError(t, err)

		// then
		assert.True(t, actualLastOp.ID == deprovisioningOpID)
		assert.Equal(t, domain.InProgress, actualLastOp.State)
	})
}
